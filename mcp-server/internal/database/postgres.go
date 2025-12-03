package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// DB represents the database connection pool
type DB struct {
	pool *pgxpool.Pool
}

// Document represents a document with embeddings
type Document struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float32              `json:"embedding,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	CreatedBy *string                `json:"created_by,omitempty"` // Use pointer to handle NULL
}

// SearchResult represents a document with similarity score
type SearchResult struct {
	Document Document
	Score    float64
}

// NewDB creates a new database connection pool
func NewDB(ctx context.Context, cfg Config) (*DB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode, cfg.MaxConns, cfg.MinConns,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Register pgvector type
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// pgvector types are automatically registered in newer versions
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	db.pool.Close()
}

// SetTenantContext sets the tenant ID for row-level security
func (db *DB) SetTenantContext(ctx context.Context, tx pgx.Tx, tenantID string) error {
	// Note: SET commands don't support parameter binding ($1), so we use fmt.Sprintf
	// The tenantID is validated to be a UUID by the JWT validator, so this is safe
	query := fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID)
	_, err := tx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}
	return nil
}

// BeginTx starts a new transaction with tenant context
func (db *DB) BeginTx(ctx context.Context, tenantID string) (pgx.Tx, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := db.SetTenantContext(ctx, tx, tenantID); err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	return tx, nil
}

// InsertDocument inserts a new document
func (db *DB) InsertDocument(ctx context.Context, tenantID string, doc *Document) error {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO documents (tenant_id, title, content, metadata, embedding, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	var embedding interface{}
	if doc.Embedding != nil {
		embedding = pgvector.NewVector(doc.Embedding)
	}

	err = tx.QueryRow(ctx, query,
		tenantID,
		doc.Title,
		doc.Content,
		doc.Metadata,
		embedding,
		doc.CreatedBy,
	).Scan(&doc.ID, &doc.CreatedAt, &doc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return tx.Commit(ctx)
}

// GetDocument retrieves a document by ID
func (db *DB) GetDocument(ctx context.Context, tenantID, docID string) (*Document, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, tenant_id, title, content, metadata, embedding, created_at, updated_at, created_by
		FROM documents
		WHERE id = $1
	`

	doc := &Document{}
	var embedding *pgvector.Vector // Use pointer to handle NULL

	err = tx.QueryRow(ctx, query, docID).Scan(
		&doc.ID,
		&doc.TenantID,
		&doc.Title,
		&doc.Content,
		&doc.Metadata,
		&embedding,
		&doc.CreatedAt,
		&doc.UpdatedAt,
		&doc.CreatedBy,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Handle NULL embeddings
	if embedding != nil && embedding.Slice() != nil {
		doc.Embedding = embedding.Slice()
	}

	return doc, nil
}

// SearchDocuments performs a text search on documents
func (db *DB) SearchDocuments(ctx context.Context, tenantID, query string, limit int) ([]*Document, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	searchQuery := `
		SELECT id, tenant_id, title, content, metadata, created_at, updated_at, created_by
		FROM documents
		WHERE
			title ILIKE $1 OR
			content ILIKE $1 OR
			metadata::text ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := tx.Query(ctx, searchQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.Title,
			&doc.Content,
			&doc.Metadata,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// VectorSearch performs similarity search using pgvector
func (db *DB) VectorSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]SearchResult, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT
			id, tenant_id, title, content, metadata, embedding, created_at, updated_at, created_by,
			1 - (embedding <=> $1) AS similarity_score
		FROM documents
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`

	vec := pgvector.NewVector(embedding)
	rows, err := tx.Query(ctx, query, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		doc := &Document{}
		var score float64
		var dbEmbedding pgvector.Vector

		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.Title,
			&doc.Content,
			&doc.Metadata,
			&dbEmbedding,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.CreatedBy,
			&score,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		doc.Embedding = dbEmbedding.Slice()
		results = append(results, SearchResult{
			Document: *doc,
			Score:    score,
		})
	}

	return results, nil
}

// ListDocuments lists all documents for a tenant
func (db *DB) ListDocuments(ctx context.Context, tenantID string, limit, offset int) ([]*Document, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, tenant_id, title, content, metadata, created_at, updated_at, created_by
		FROM documents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := tx.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var documents []*Document
	for rows.Next() {
		doc := &Document{}
		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.Title,
			&doc.Content,
			&doc.Metadata,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&doc.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// UpdateDocument updates an existing document
func (db *DB) UpdateDocument(ctx context.Context, tenantID string, doc *Document) error {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE documents
		SET title = $1, content = $2, metadata = $3, embedding = $4
		WHERE id = $5
		RETURNING updated_at
	`

	var embedding interface{}
	if doc.Embedding != nil {
		embedding = pgvector.NewVector(doc.Embedding)
	}

	err = tx.QueryRow(ctx, query,
		doc.Title,
		doc.Content,
		doc.Metadata,
		embedding,
		doc.ID,
	).Scan(&doc.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fmt.Errorf("document not found")
	}
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	return tx.Commit(ctx)
}

// DeleteDocument deletes a document by ID
func (db *DB) DeleteDocument(ctx context.Context, tenantID, docID string) error {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `DELETE FROM documents WHERE id = $1`

	result, err := tx.Exec(ctx, query, docID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("document not found")
	}

	return tx.Commit(ctx)
}

// GetTenantSettings retrieves tenant settings
func (db *DB) GetTenantSettings(ctx context.Context, tenantID string) (map[string]interface{}, error) {
	query := `SELECT settings FROM tenants WHERE id = $1 AND is_active = true`

	var settings map[string]interface{}
	err := db.pool.QueryRow(ctx, query, tenantID).Scan(&settings)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("tenant not found or inactive")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant settings: %w", err)
	}

	return settings, nil
}

package database

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"
)

// HybridSearchParams holds parameters for hybrid search
type HybridSearchParams struct {
	Query         string
	Embedding     []float32
	Limit         int
	BM25Weight    float64 // Weight for lexical search (0.0 to 1.0)
	VectorWeight  float64 // Weight for semantic search (0.0 to 1.0)
	MinBM25Score  float64 // Minimum BM25 score threshold
	MinVectorSim  float64 // Minimum vector similarity threshold
}

// HybridSearchResult represents a result from hybrid search
type HybridSearchResult struct {
	Document      Document
	BM25Score     float64
	VectorScore   float64
	CombinedScore float64
}

// HybridSearch performs a hybrid search combining BM25 (full-text) and vector similarity
// This implements a Reciprocal Rank Fusion (RRF) approach for combining results
func (db *DB) HybridSearch(ctx context.Context, tenantID string, params HybridSearchParams) ([]HybridSearchResult, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Normalize weights if they don't sum to 1.0
	totalWeight := params.BM25Weight + params.VectorWeight
	if totalWeight == 0 {
		params.BM25Weight = 0.5
		params.VectorWeight = 0.5
		totalWeight = 1.0
	}
	bm25Weight := params.BM25Weight / totalWeight
	vectorWeight := params.VectorWeight / totalWeight

	if params.Limit <= 0 {
		params.Limit = 10
	}

	// Hybrid search query using PostgreSQL's full-text search (BM25-like) and pgvector
	// We use ts_rank_cd which implements a ranking similar to BM25
	query := `
		WITH bm25_results AS (
			SELECT
				id,
				tenant_id,
				title,
				content,
				metadata,
				embedding,
				created_at,
				updated_at,
				created_by,
				ts_rank_cd(
					to_tsvector('english', title || ' ' || content),
					plainto_tsquery('english', $1)
				) AS bm25_score,
				ROW_NUMBER() OVER (ORDER BY ts_rank_cd(
					to_tsvector('english', title || ' ' || content),
					plainto_tsquery('english', $1)
				) DESC) AS bm25_rank
			FROM documents
			WHERE to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)
		),
		vector_results AS (
			SELECT
				id,
				tenant_id,
				title,
				content,
				metadata,
				embedding,
				created_at,
				updated_at,
				created_by,
				1 - (embedding <=> $2) AS vector_score,
				ROW_NUMBER() OVER (ORDER BY embedding <=> $2) AS vector_rank
			FROM documents
			WHERE embedding IS NOT NULL
		),
		combined AS (
			SELECT
				COALESCE(b.id, v.id) AS id,
				COALESCE(b.tenant_id, v.tenant_id) AS tenant_id,
				COALESCE(b.title, v.title) AS title,
				COALESCE(b.content, v.content) AS content,
				COALESCE(b.metadata, v.metadata) AS metadata,
				COALESCE(b.embedding, v.embedding) AS embedding,
				COALESCE(b.created_at, v.created_at) AS created_at,
				COALESCE(b.updated_at, v.updated_at) AS updated_at,
				COALESCE(b.created_by, v.created_by) AS created_by,
				COALESCE(b.bm25_score, 0) AS bm25_score,
				COALESCE(v.vector_score, 0) AS vector_score,
				-- Reciprocal Rank Fusion score
				(
					COALESCE(1.0 / (60 + b.bm25_rank), 0) * $3 +
					COALESCE(1.0 / (60 + v.vector_rank), 0) * $4
				) AS combined_score
			FROM bm25_results b
			FULL OUTER JOIN vector_results v ON b.id = v.id
			WHERE
				COALESCE(b.bm25_score, 0) >= $5
				OR COALESCE(v.vector_score, 0) >= $6
		)
		SELECT
			id, tenant_id, title, content, metadata, embedding,
			created_at, updated_at, created_by,
			bm25_score, vector_score, combined_score
		FROM combined
		ORDER BY combined_score DESC
		LIMIT $7
	`

	var embedding interface{}
	if params.Embedding != nil {
		embedding = pgvector.NewVector(params.Embedding)
	}

	rows, err := tx.Query(ctx, query,
		params.Query,
		embedding,
		bm25Weight,
		vectorWeight,
		params.MinBM25Score,
		params.MinVectorSim,
		params.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to perform hybrid search: %w", err)
	}
	defer rows.Close()

	var results []HybridSearchResult
	for rows.Next() {
		var doc Document
		var bm25Score, vectorScore, combinedScore float64
		var dbEmbedding *pgvector.Vector // Use pointer to handle NULL

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
			&bm25Score,
			&vectorScore,
			&combinedScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hybrid search result: %w", err)
		}

		if dbEmbedding != nil && dbEmbedding.Slice() != nil {
			doc.Embedding = dbEmbedding.Slice()
		}

		results = append(results, HybridSearchResult{
			Document:      doc,
			BM25Score:     bm25Score,
			VectorScore:   vectorScore,
			CombinedScore: combinedScore,
		})
	}

	return results, nil
}

// SimpleHybridSearch performs a simpler version of hybrid search
// Uses weighted average of BM25 and vector similarity scores
func (db *DB) SimpleHybridSearch(ctx context.Context, tenantID string, params HybridSearchParams) ([]HybridSearchResult, error) {
	tx, err := db.BeginTx(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Normalize weights
	totalWeight := params.BM25Weight + params.VectorWeight
	if totalWeight == 0 {
		params.BM25Weight = 0.5
		params.VectorWeight = 0.5
		totalWeight = 1.0
	}
	bm25Weight := params.BM25Weight / totalWeight
	vectorWeight := params.VectorWeight / totalWeight

	if params.Limit <= 0 {
		params.Limit = 10
	}

	// Simpler hybrid query using weighted scores
	query := `
		SELECT
			id, tenant_id, title, content, metadata, embedding,
			created_at, updated_at, created_by,
			ts_rank_cd(
				to_tsvector('english', title || ' ' || content),
				plainto_tsquery('english', $1)
			) AS bm25_score,
			CASE
				WHEN embedding IS NOT NULL THEN 1 - (embedding <=> $2)
				ELSE 0
			END AS vector_score,
			(
				ts_rank_cd(
					to_tsvector('english', title || ' ' || content),
					plainto_tsquery('english', $1)
				) * $3 +
				CASE
					WHEN embedding IS NOT NULL THEN (1 - (embedding <=> $2)) * $4
					ELSE 0
				END
			) AS combined_score
		FROM documents
		WHERE
			to_tsvector('english', title || ' ' || content) @@ plainto_tsquery('english', $1)
			OR (embedding IS NOT NULL AND (1 - (embedding <=> $2)) >= $6)
		ORDER BY combined_score DESC
		LIMIT $5
	`

	var embedding interface{}
	if params.Embedding != nil {
		embedding = pgvector.NewVector(params.Embedding)
	}

	rows, err := tx.Query(ctx, query,
		params.Query,
		embedding,
		bm25Weight,
		vectorWeight,
		params.Limit,
		params.MinVectorSim,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to perform simple hybrid search: %w", err)
	}
	defer rows.Close()

	var results []HybridSearchResult
	for rows.Next() {
		var doc Document
		var bm25Score, vectorScore, combinedScore float64
		var dbEmbedding *pgvector.Vector // Use pointer to handle NULL

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
			&bm25Score,
			&vectorScore,
			&combinedScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if dbEmbedding != nil && dbEmbedding.Slice() != nil {
			doc.Embedding = dbEmbedding.Slice()
		}

		results = append(results, HybridSearchResult{
			Document:      doc,
			BM25Score:     bm25Score,
			VectorScore:   vectorScore,
			CombinedScore: combinedScore,
		})
	}

	return results, nil
}

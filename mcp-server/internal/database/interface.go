package database

import "context"

// Store defines the interface for database operations
// This interface enables testing with mocks
type Store interface {
	// GetDocument retrieves a document by ID for a specific tenant
	GetDocument(ctx context.Context, tenantID, docID string) (*Document, error)

	// SearchDocuments performs full-text search on documents
	SearchDocuments(ctx context.Context, tenantID, query string, limit int) ([]*Document, error)

	// ListDocuments lists documents for a tenant with pagination
	ListDocuments(ctx context.Context, tenantID string, limit, offset int) ([]*Document, error)

	// HybridSearch performs hybrid BM25 + vector search with RRF
	HybridSearch(ctx context.Context, tenantID string, params HybridSearchParams) ([]HybridSearchResult, error)

	// SimpleHybridSearch performs simple weighted hybrid search
	SimpleHybridSearch(ctx context.Context, tenantID string, params HybridSearchParams) ([]HybridSearchResult, error)
}

// Ensure DB implements Store interface
var _ Store = (*DB)(nil)

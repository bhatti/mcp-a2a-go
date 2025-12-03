// +build integration

package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration tests for PostgreSQL database operations
// Run with: go test -tags=integration -v ./internal/database/

const testTenantID = "11111111-1111-1111-1111-111111111111" // acme-corp from init-db.sql

func getTestDBConfig() Config {
	return Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnvOrDefault("DB_USER", "mcp_user"),
		Password: getEnvOrDefault("DB_PASSWORD", "mcp_secure_pass"),
		DBName:   getEnvOrDefault("DB_NAME", "mcp_db"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		MaxConns: 10,
		MinConns: 2,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupTestDB(t *testing.T) *DB {
	cfg := getTestDBConfig()
	db, err := NewDB(context.Background(), cfg)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

func TestGetDocument_WithNullEmbedding(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert a test document WITHOUT embedding
	testDoc := &Document{
		TenantID:  testTenantID,
		Title:     "Test Document Without Embedding",
		Content:   "This document has no embedding vector and should not cause scan errors",
		Metadata:  map[string]interface{}{"test": true, "category": "integration-test"},
		Embedding: nil, // Explicitly no embedding
	}

	err := db.InsertDocument(ctx, testTenantID, testDoc)
	require.NoError(t, err, "Failed to insert test document")
	require.NotEmpty(t, testDoc.ID, "Document ID should be generated")

	// Now retrieve the document - this should NOT fail with NULL scan error
	retrieved, err := db.GetDocument(ctx, testTenantID, testDoc.ID)
	require.NoError(t, err, "Failed to retrieve document with NULL embedding")
	assert.NotNil(t, retrieved, "Retrieved document should not be nil")
	assert.Equal(t, testDoc.ID, retrieved.ID)
	assert.Equal(t, testDoc.Title, retrieved.Title)
	assert.Equal(t, testDoc.Content, retrieved.Content)
	assert.Nil(t, retrieved.Embedding, "Embedding should be nil for document without embedding")

	// Cleanup
	err = db.DeleteDocument(ctx, testTenantID, testDoc.ID)
	require.NoError(t, err, "Failed to delete test document")
}

func TestGetDocument_WithEmbedding(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create a test embedding vector (1536 dimensions for OpenAI ada-002)
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	// Insert a test document WITH embedding
	testDoc := &Document{
		TenantID:  testTenantID,
		Title:     "Test Document With Embedding",
		Content:   "This document has an embedding vector",
		Metadata:  map[string]interface{}{"test": true, "category": "integration-test"},
		Embedding: embedding,
	}

	err := db.InsertDocument(ctx, testTenantID, testDoc)
	require.NoError(t, err, "Failed to insert test document")
	require.NotEmpty(t, testDoc.ID, "Document ID should be generated")

	// Retrieve the document
	retrieved, err := db.GetDocument(ctx, testTenantID, testDoc.ID)
	require.NoError(t, err, "Failed to retrieve document with embedding")
	assert.NotNil(t, retrieved, "Retrieved document should not be nil")
	assert.Equal(t, testDoc.ID, retrieved.ID)
	assert.Equal(t, testDoc.Title, retrieved.Title)
	assert.Equal(t, testDoc.Content, retrieved.Content)
	assert.NotNil(t, retrieved.Embedding, "Embedding should not be nil")
	assert.Equal(t, len(embedding), len(retrieved.Embedding), "Embedding dimension should match")

	// Cleanup
	err = db.DeleteDocument(ctx, testTenantID, testDoc.ID)
	require.NoError(t, err, "Failed to delete test document")
}

func TestListDocuments_WithMixedEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create test documents with and without embeddings
	docs := []*Document{
		{
			TenantID:  testTenantID,
			Title:     "Doc 1 - No Embedding",
			Content:   "Content 1",
			Metadata:  map[string]interface{}{"test": true},
			Embedding: nil,
		},
		{
			TenantID:  testTenantID,
			Title:     "Doc 2 - Has Embedding",
			Content:   "Content 2",
			Metadata:  map[string]interface{}{"test": true},
			Embedding: make([]float32, 1536),
		},
		{
			TenantID:  testTenantID,
			Title:     "Doc 3 - No Embedding",
			Content:   "Content 3",
			Metadata:  map[string]interface{}{"test": true},
			Embedding: nil,
		},
	}

	// Insert all documents
	for _, doc := range docs {
		err := db.InsertDocument(ctx, testTenantID, doc)
		require.NoError(t, err, "Failed to insert document: "+doc.Title)
	}

	// List documents should handle mixed embeddings
	listed, err := db.ListDocuments(ctx, testTenantID, 10, 0)
	require.NoError(t, err, "Failed to list documents")
	assert.GreaterOrEqual(t, len(listed), 3, "Should have at least 3 documents")

	// Cleanup
	for _, doc := range docs {
		err = db.DeleteDocument(ctx, testTenantID, doc.ID)
		require.NoError(t, err, "Failed to delete document: "+doc.Title)
	}
}

func TestSearchDocuments_WithNullEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert test document without embedding
	testDoc := &Document{
		TenantID:  testTenantID,
		Title:     "Security Policy Test",
		Content:   "Test content about security and authentication",
		Metadata:  map[string]interface{}{"test": true, "category": "security"},
		Embedding: nil,
	}

	err := db.InsertDocument(ctx, testTenantID, testDoc)
	require.NoError(t, err, "Failed to insert test document")

	// Search should work even with NULL embeddings
	results, err := db.SearchDocuments(ctx, testTenantID, "security", 10)
	require.NoError(t, err, "Failed to search documents")
	assert.GreaterOrEqual(t, len(results), 1, "Should find at least one document")

	// Verify our test document is in results
	found := false
	for _, doc := range results {
		if doc.ID == testDoc.ID {
			found = true
			assert.Equal(t, testDoc.Title, doc.Title)
			break
		}
	}
	assert.True(t, found, "Should find our test document in search results")

	// Cleanup
	err = db.DeleteDocument(ctx, testTenantID, testDoc.ID)
	require.NoError(t, err, "Failed to delete test document")
}

func TestVectorSearch_SkipsNullEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create query embedding
	queryEmbedding := make([]float32, 1536)
	for i := range queryEmbedding {
		queryEmbedding[i] = float32(i) * 0.001
	}

	// Vector search should only return documents with embeddings
	results, err := db.VectorSearch(ctx, testTenantID, queryEmbedding, 5)
	require.NoError(t, err, "Vector search should not fail")

	// All returned documents should have embeddings
	for _, result := range results {
		assert.NotNil(t, result.Document.Embedding, "Vector search should only return docs with embeddings")
	}
}

func TestHybridSearch_HandlesNullEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create query embedding
	queryEmbedding := make([]float32, 1536)
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.1
	}

	params := HybridSearchParams{
		Query:        "security policy",
		Embedding:    queryEmbedding,
		Limit:        10,
		BM25Weight:   0.5,
		VectorWeight: 0.5,
		MinBM25Score: 0.0,
		MinVectorSim: 0.0,
	}

	// Hybrid search should handle documents without embeddings gracefully
	results, err := db.HybridSearch(ctx, testTenantID, params)
	require.NoError(t, err, "Hybrid search should not fail with NULL embeddings")
	assert.NotNil(t, results, "Results should not be nil")

	// Documents without embeddings should still appear in results (with 0 vector score)
	for _, result := range results {
		t.Logf("Document: %s, BM25: %.2f, Vector: %.2f, Combined: %.2f",
			result.Document.Title, result.BM25Score, result.VectorScore, result.CombinedScore)
	}
}

func TestSimpleHybridSearch_HandlesNullEmbeddings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create query embedding
	queryEmbedding := make([]float32, 1536)
	for i := range queryEmbedding {
		queryEmbedding[i] = 0.1
	}

	params := HybridSearchParams{
		Query:        "security",
		Embedding:    queryEmbedding,
		Limit:        10,
		BM25Weight:   0.5,
		VectorWeight: 0.5,
	}

	// Simple hybrid search should also handle NULL embeddings
	results, err := db.SimpleHybridSearch(ctx, testTenantID, params)
	require.NoError(t, err, "Simple hybrid search should not fail with NULL embeddings")
	assert.NotNil(t, results, "Results should not be nil")

	for _, result := range results {
		t.Logf("Document: %s, BM25: %.2f, Vector: %.2f, Combined: %.2f, HasEmbedding: %v",
			result.Document.Title, result.BM25Score, result.VectorScore,
			result.CombinedScore, result.Document.Embedding != nil)
	}
}

func TestGetDocument_FromInitialSampleData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// List documents to get actual IDs from sample data
	docs, err := db.ListDocuments(ctx, testTenantID, 10, 0)
	require.NoError(t, err, "Failed to list documents")
	require.NotEmpty(t, docs, "Should have sample documents from init-db.sql")

	// Try to retrieve each document by ID
	for _, doc := range docs {
		t.Run("Retrieve_"+doc.Title, func(t *testing.T) {
			retrieved, err := db.GetDocument(ctx, testTenantID, doc.ID)
			require.NoError(t, err, "Failed to retrieve document: "+doc.ID)
			assert.NotNil(t, retrieved, "Retrieved document should not be nil")
			assert.Equal(t, doc.ID, retrieved.ID)
			assert.Equal(t, doc.Title, retrieved.Title)
			assert.Equal(t, doc.Content, retrieved.Content)

			t.Logf("✓ Successfully retrieved: %s (ID: %s, HasEmbedding: %v)",
				retrieved.Title, retrieved.ID, retrieved.Embedding != nil)
		})
	}
}

func TestUpdateDocument_PreservesEmbedding(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Create embedding
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1
	}

	// Insert document with embedding
	doc := &Document{
		TenantID:  testTenantID,
		Title:     "Original Title",
		Content:   "Original Content",
		Metadata:  map[string]interface{}{"version": 1},
		Embedding: embedding,
	}

	err := db.InsertDocument(ctx, testTenantID, doc)
	require.NoError(t, err)

	// Update document
	doc.Title = "Updated Title"
	doc.Content = "Updated Content"
	doc.Metadata = map[string]interface{}{"version": 2}

	err = db.UpdateDocument(ctx, testTenantID, doc)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := db.GetDocument(ctx, testTenantID, doc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, "Updated Content", retrieved.Content)
	assert.NotNil(t, retrieved.Embedding, "Embedding should be preserved")

	// Cleanup
	err = db.DeleteDocument(ctx, testTenantID, doc.ID)
	require.NoError(t, err)
}

func TestTenantIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Insert document for tenant 1
	doc1 := &Document{
		TenantID: testTenantID,
		Title:    "Tenant 1 Document",
		Content:  "This belongs to tenant 1",
		Metadata: map[string]interface{}{"tenant": 1},
	}

	err := db.InsertDocument(ctx, testTenantID, doc1)
	require.NoError(t, err)

	// Try to retrieve with different tenant ID (should fail due to RLS)
	otherTenantID := "22222222-2222-2222-2222-222222222222"
	_, err = db.GetDocument(ctx, otherTenantID, doc1.ID)
	assert.Error(t, err, "Should not be able to access document from different tenant")

	// Cleanup
	err = db.DeleteDocument(ctx, testTenantID, doc1.ID)
	require.NoError(t, err)
}

func TestConcurrentRetrievals(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Get sample documents
	docs, err := db.ListDocuments(ctx, testTenantID, 5, 0)
	require.NoError(t, err)
	require.NotEmpty(t, docs)

	// Perform concurrent retrievals
	numWorkers := 10
	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < 5; j++ {
				for _, doc := range docs {
					retrieved, err := db.GetDocument(ctx, testTenantID, doc.ID)
					if err != nil {
						t.Errorf("Worker %d failed to retrieve document: %v", workerID, err)
						return
					}
					if retrieved == nil {
						t.Errorf("Worker %d got nil document", workerID)
						return
					}
				}
			}
		}(i)
	}

	// Wait for all workers
	timeout := time.After(30 * time.Second)
	for i := 0; i < numWorkers; i++ {
		select {
		case <-done:
			// Worker completed
		case <-timeout:
			t.Fatal("Concurrent retrieval test timed out")
		}
	}

	t.Log("✓ All concurrent retrievals completed successfully")
}

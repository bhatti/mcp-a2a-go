package tools

import (
	"context"
	"testing"
	"time"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHybridSearchToolDefinition(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewHybridSearchTool(mockDB)

	def := tool.Definition()

	assert.Equal(t, "hybrid_search", def.Name)
	assert.NotEmpty(t, def.Description)
	assert.NotNil(t, def.InputSchema)
	assert.Equal(t, "object", def.InputSchema["type"])

	// Verify required fields
	required, ok := def.InputSchema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "query")
}

func TestHybridSearchToolExecute(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		setupAuth func(ctx context.Context) context.Context
		args      map[string]interface{}
		setupMock func(m *MockStore)
		wantErr   bool
		validate  func(t *testing.T, result protocol.ToolCallResult)
	}{
		{
			name: "successful search with results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query":         "machine learning",
				"limit":         5,
				"bm25_weight":   0.6,
				"vector_weight": 0.4,
			},
			setupMock: func(m *MockStore) {
				results := []database.HybridSearchResult{
					{
						Document: database.Document{
							ID:        "doc-1",
							TenantID:  "tenant-123",
							Title:     "ML Guide",
							Content:   "A comprehensive guide to machine learning algorithms",
							Metadata:  map[string]interface{}{"category": "tutorial"},
							CreatedAt: now,
						},
						BM25Score:     2.5,
						VectorScore:   0.85,
						CombinedScore: 1.84, // (2.5 * 0.6) + (0.85 * 0.4)
					},
				}
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.MatchedBy(func(params database.HybridSearchParams) bool {
					return params.Query == "machine learning" &&
						params.Limit == 5 &&
						params.BM25Weight == 0.6 &&
						params.VectorWeight == 0.4
				})).Return(results, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				assert.Equal(t, "text", result.Content[0].Type)
				// Now returns JSON array
				assert.Contains(t, result.Content[0].Text, "doc_id")
				assert.Contains(t, result.Content[0].Text, "ML Guide")
				assert.Contains(t, result.Content[0].Text, "bm25_score")
				assert.Contains(t, result.Content[0].Text, "vector_score")
				assert.Contains(t, result.Content[0].Text, "2.5")
				assert.Contains(t, result.Content[0].Text, "0.85")
			},
		},
		{
			name: "search with embedding vector",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query":         "AI",
				"embedding":     []interface{}{0.1, 0.2, 0.3}, // Simplified embedding
				"limit":         10,
				"bm25_weight":   0.5,
				"vector_weight": 0.5,
			},
			setupMock: func(m *MockStore) {
				results := []database.HybridSearchResult{}
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.MatchedBy(func(params database.HybridSearchParams) bool {
					return params.Query == "AI" &&
						len(params.Embedding) == 3 &&
						params.Embedding[0] == 0.1
				})).Return(results, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				// Empty results return JSON null or empty array
				text := result.Content[0].Text
				assert.True(t, text == "null" || text == "[]", "Expected 'null' or '[]', got: %s", text)
			},
		},
		{
			name: "search with no results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "nonexistent topic",
			},
			setupMock: func(m *MockStore) {
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.Anything).
					Return([]database.HybridSearchResult{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				// Empty results return JSON null or empty array
				text := result.Content[0].Text
				assert.True(t, text == "null" || text == "[]", "Expected 'null' or '[]', got: %s", text)
			},
		},
		{
			name: "missing authentication",
			setupAuth: func(ctx context.Context) context.Context {
				return ctx // No auth context
			},
			args: map[string]interface{}{
				"query": "test",
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "missing required query parameter",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit": 10,
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "empty query",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "",
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "default limit and weights",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test",
			},
			setupMock: func(m *MockStore) {
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.MatchedBy(func(params database.HybridSearchParams) bool {
					return params.Limit == 10 &&
						params.BM25Weight == 0.5 &&
						params.VectorWeight == 0.5
				})).Return([]database.HybridSearchResult{}, nil)
			},
			wantErr: false,
		},
		{
			name: "limit capped at 50",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test",
				"limit": 100,
			},
			setupMock: func(m *MockStore) {
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.MatchedBy(func(params database.HybridSearchParams) bool {
					return params.Limit == 50
				})).Return([]database.HybridSearchResult{}, nil)
			},
			wantErr: false,
		},
		{
			name: "custom weights",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query":         "test",
				"bm25_weight":   0.7,
				"vector_weight": 0.3,
			},
			setupMock: func(m *MockStore) {
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.MatchedBy(func(params database.HybridSearchParams) bool {
					return params.BM25Weight == 0.7 && params.VectorWeight == 0.3
				})).Return([]database.HybridSearchResult{}, nil)
			},
			wantErr: false,
		},
		{
			name: "database error",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test",
			},
			setupMock: func(m *MockStore) {
				m.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.Anything).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockStore)
			tt.setupMock(mockDB)

			tool := NewHybridSearchTool(mockDB)
			ctx := tt.setupAuth(context.Background())

			result, err := tool.Execute(ctx, tt.args)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestHybridSearchToolInvalidArguments(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewHybridSearchTool(mockDB)

	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	// Test with invalid argument type (channel can't be marshaled)
	args := map[string]interface{}{
		"query": make(chan int),
	}

	_, err := tool.Execute(ctx, args)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkHybridSearchToolExecute(b *testing.B) {
	mockDB := new(MockStore)
	now := time.Now()

	results := []database.HybridSearchResult{
		{
			Document: database.Document{
				ID:        "doc-1",
				Title:     "Benchmark Doc",
				Content:   "Content for benchmarking",
				Metadata:  map[string]interface{}{},
				CreatedAt: now,
			},
			BM25Score:     2.0,
			VectorScore:   0.8,
			CombinedScore: 1.4,
		},
	}

	mockDB.On("SimpleHybridSearch", mock.Anything, "tenant-123", mock.Anything).
		Return(results, nil)

	tool := NewHybridSearchTool(mockDB)
	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	args := map[string]interface{}{
		"query": "benchmark query",
		"limit": 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

package tools

import (
	"context"
	"testing"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock implementation of the database.Store interface
type MockStore struct {
	mock.Mock
}

func (m *MockStore) SearchDocuments(ctx context.Context, tenantID, query string, limit int) ([]*database.Document, error) {
	args := m.Called(ctx, tenantID, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*database.Document), args.Error(1)
}

func (m *MockStore) GetDocument(ctx context.Context, tenantID, docID string) (*database.Document, error) {
	args := m.Called(ctx, tenantID, docID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.Document), args.Error(1)
}

func (m *MockStore) ListDocuments(ctx context.Context, tenantID string, limit, offset int) ([]*database.Document, error) {
	args := m.Called(ctx, tenantID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*database.Document), args.Error(1)
}

func (m *MockStore) HybridSearch(ctx context.Context, tenantID string, params database.HybridSearchParams) ([]database.HybridSearchResult, error) {
	args := m.Called(ctx, tenantID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.HybridSearchResult), args.Error(1)
}

func (m *MockStore) SimpleHybridSearch(ctx context.Context, tenantID string, params database.HybridSearchParams) ([]database.HybridSearchResult, error) {
	args := m.Called(ctx, tenantID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.HybridSearchResult), args.Error(1)
}

func TestSearchToolDefinition(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewSearchTool(mockDB)

	def := tool.Definition()

	assert.Equal(t, "search_documents", def.Name)
	assert.NotEmpty(t, def.Description)
	assert.NotNil(t, def.InputSchema)
	assert.Equal(t, "object", def.InputSchema["type"])

	// Verify required fields
	required, ok := def.InputSchema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "query")
}

func TestSearchToolExecute(t *testing.T) {
	tests := []struct {
		name       string
		setupAuth  func(ctx context.Context) context.Context
		args       map[string]interface{}
		setupMock  func(m *MockStore)
		wantErr    bool
		validate   func(t *testing.T, result protocol.ToolCallResult)
	}{
		{
			name: "successful search with results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test query",
				"limit": 10,
			},
			setupMock: func(m *MockStore) {
				docs := []*database.Document{
					{
						ID:       "doc-1",
						Title:    "Test Document 1",
						Content:  "Content 1",
						Metadata: map[string]interface{}{"category": "test"},
					},
					{
						ID:      "doc-2",
						Title:   "Test Document 2",
						Content: "Content 2",
					},
				}
				m.On("SearchDocuments", mock.Anything, "tenant-123", "test query", 10).
					Return(docs, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				assert.Equal(t, "text", result.Content[0].Type)
				assert.Contains(t, result.Content[0].Text, "Found 2 document(s)")
				assert.Contains(t, result.Content[0].Text, "Test Document 1")
			},
		},
		{
			name: "search with no results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "nonexistent",
			},
			setupMock: func(m *MockStore) {
				m.On("SearchDocuments", mock.Anything, "tenant-123", "nonexistent", 10).
					Return([]*database.Document{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Contains(t, result.Content[0].Text, "No documents found")
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
			name: "limit defaults to 10",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test",
			},
			setupMock: func(m *MockStore) {
				m.On("SearchDocuments", mock.Anything, "tenant-123", "test", 10).
					Return([]*database.Document{}, nil)
			},
			wantErr: false,
		},
		{
			name: "limit capped at 100",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"query": "test",
				"limit": 500,
			},
			setupMock: func(m *MockStore) {
				m.On("SearchDocuments", mock.Anything, "tenant-123", "test", 100).
					Return([]*database.Document{}, nil)
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
				m.On("SearchDocuments", mock.Anything, "tenant-123", "test", 10).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockStore)
			tt.setupMock(mockDB)

			tool := NewSearchTool(mockDB)
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

func TestSearchToolInvalidArguments(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewSearchTool(mockDB)

	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	// Test with invalid argument type (channel can't be marshaled)
	args := map[string]interface{}{
		"query": make(chan int),
	}

	_, err := tool.Execute(ctx, args)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkSearchToolExecute(b *testing.B) {
	mockDB := new(MockStore)
	mockDB.On("SearchDocuments", mock.Anything, "tenant-123", "benchmark query", 10).
		Return([]*database.Document{
			{ID: "doc-1", Title: "Doc 1", Content: "Content 1"},
		}, nil)

	tool := NewSearchTool(mockDB)
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

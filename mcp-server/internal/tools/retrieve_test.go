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

func TestRetrieveToolDefinition(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewRetrieveTool(mockDB)

	def := tool.Definition()

	assert.Equal(t, "retrieve_document", def.Name)
	assert.NotEmpty(t, def.Description)
	assert.NotNil(t, def.InputSchema)
	assert.Equal(t, "object", def.InputSchema["type"])

	// Verify required fields
	required, ok := def.InputSchema["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "document_id")
}

func TestRetrieveToolExecute(t *testing.T) {
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
			name: "successful retrieval",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"document_id": "doc-1",
			},
			setupMock: func(m *MockStore) {
				createdBy := "user-123"
				doc := &database.Document{
					ID:        "doc-1",
					TenantID:  "tenant-123",
					Title:     "Test Document",
					Content:   "This is test content",
					Metadata:  map[string]interface{}{"category": "test"},
					CreatedAt: now,
					UpdatedAt: now,
					CreatedBy: &createdBy,
				}
				m.On("GetDocument", mock.Anything, "tenant-123", "doc-1").
					Return(doc, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				assert.Equal(t, "text", result.Content[0].Type)
				assert.Contains(t, result.Content[0].Text, "Document Retrieved")
				assert.Contains(t, result.Content[0].Text, "doc-1")
				assert.Contains(t, result.Content[0].Text, "Test Document")
				assert.Contains(t, result.Content[0].Text, "This is test content")
				assert.Contains(t, result.Content[0].Text, "user-123")
			},
		},
		{
			name: "missing authentication",
			setupAuth: func(ctx context.Context) context.Context {
				return ctx // No auth context
			},
			args: map[string]interface{}{
				"document_id": "doc-1",
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "missing required document_id parameter",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"other_param": "value",
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "empty document_id",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"document_id": "",
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "document not found",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"document_id": "nonexistent",
			},
			setupMock: func(m *MockStore) {
				m.On("GetDocument", mock.Anything, "tenant-123", "nonexistent").
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "document without created_by field",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"document_id": "doc-2",
			},
			setupMock: func(m *MockStore) {
				doc := &database.Document{
					ID:        "doc-2",
					TenantID:  "tenant-123",
					Title:     "System Document",
					Content:   "Auto-generated content",
					Metadata:  map[string]interface{}{},
					CreatedAt: now,
					UpdatedAt: now,
					CreatedBy: nil, // No creator
				}
				m.On("GetDocument", mock.Anything, "tenant-123", "doc-2").
					Return(doc, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				assert.Contains(t, result.Content[0].Text, "System Document")
				// Should not contain "Created By" line when empty
				assert.NotContains(t, result.Content[0].Text, "Created By:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockStore)
			tt.setupMock(mockDB)

			tool := NewRetrieveTool(mockDB)
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

func TestRetrieveToolInvalidArguments(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewRetrieveTool(mockDB)

	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	// Test with invalid argument type (channel can't be marshaled)
	args := map[string]interface{}{
		"document_id": make(chan int),
	}

	_, err := tool.Execute(ctx, args)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkRetrieveToolExecute(b *testing.B) {
	mockDB := new(MockStore)
	now := time.Now()

	createdBy := "bench-user"
	doc := &database.Document{
		ID:        "doc-1",
		TenantID:  "tenant-123",
		Title:     "Benchmark Document",
		Content:   "Benchmark content",
		Metadata:  map[string]interface{}{"test": true},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: &createdBy,
	}

	mockDB.On("GetDocument", mock.Anything, "tenant-123", "doc-1").
		Return(doc, nil)

	tool := NewRetrieveTool(mockDB)
	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	args := map[string]interface{}{
		"document_id": "doc-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

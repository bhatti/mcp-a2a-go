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

func TestListToolDefinition(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewListTool(mockDB)

	def := tool.Definition()

	assert.Equal(t, "list_documents", def.Name)
	assert.NotEmpty(t, def.Description)
	assert.NotNil(t, def.InputSchema)
	assert.Equal(t, "object", def.InputSchema["type"])
}

func TestListToolExecute(t *testing.T) {
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
			name: "successful list with results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit":  10,
				"offset": 0,
			},
			setupMock: func(m *MockStore) {
				docs := []*database.Document{
					{
						ID:        "doc-1",
						TenantID:  "tenant-123",
						Title:     "Document 1",
						Content:   "Content for document 1",
						Metadata:  map[string]interface{}{"category": "test"},
						CreatedAt: now,
					},
					{
						ID:        "doc-2",
						TenantID:  "tenant-123",
						Title:     "Document 2",
						Content:   "Content for document 2",
						Metadata:  map[string]interface{}{},
						CreatedAt: now,
					},
				}
				m.On("ListDocuments", mock.Anything, "tenant-123", 10, 0).
					Return(docs, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, result protocol.ToolCallResult) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				assert.Equal(t, "text", result.Content[0].Type)
				assert.Contains(t, result.Content[0].Text, "Found 2 document(s)")
				assert.Contains(t, result.Content[0].Text, "Document 1")
				assert.Contains(t, result.Content[0].Text, "Document 2")
				assert.Contains(t, result.Content[0].Text, "Category: test")
			},
		},
		{
			name: "list with no results",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit":  10,
				"offset": 0,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 10, 0).
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
				"limit": 10,
			},
			setupMock: func(m *MockStore) {
				// No mock setup needed
			},
			wantErr: true,
		},
		{
			name: "default limit when not specified",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 20, 0).
					Return([]*database.Document{}, nil)
			},
			wantErr: false,
		},
		{
			name: "custom offset",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit":  5,
				"offset": 10,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 5, 10).
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
				"limit":  500,
				"offset": 0,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 100, 0).
					Return([]*database.Document{}, nil)
			},
			wantErr: false,
		},
		{
			name: "negative offset becomes zero",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit":  10,
				"offset": -5,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 10, 0).
					Return([]*database.Document{}, nil)
			},
			wantErr: false,
		},
		{
			name: "zero limit becomes default",
			setupAuth: func(ctx context.Context) context.Context {
				return context.WithValue(ctx, auth.ContextKeyTenantID, "tenant-123")
			},
			args: map[string]interface{}{
				"limit":  0,
				"offset": 0,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 20, 0).
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
				"limit": 10,
			},
			setupMock: func(m *MockStore) {
				m.On("ListDocuments", mock.Anything, "tenant-123", 10, 0).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockStore)
			tt.setupMock(mockDB)

			tool := NewListTool(mockDB)
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

func TestListToolInvalidArguments(t *testing.T) {
	mockDB := new(MockStore)
	tool := NewListTool(mockDB)

	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	// Test with invalid argument type (channel can't be marshaled)
	args := map[string]interface{}{
		"limit": make(chan int),
	}

	_, err := tool.Execute(ctx, args)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkListToolExecute(b *testing.B) {
	mockDB := new(MockStore)
	now := time.Now()

	docs := []*database.Document{
		{ID: "doc-1", Title: "Doc 1", Content: "Content 1", Metadata: map[string]interface{}{}, CreatedAt: now},
		{ID: "doc-2", Title: "Doc 2", Content: "Content 2", Metadata: map[string]interface{}{}, CreatedAt: now},
	}

	mockDB.On("ListDocuments", mock.Anything, "tenant-123", 20, 0).
		Return(docs, nil)

	tool := NewListTool(mockDB)
	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	args := map[string]interface{}{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tool.Execute(ctx, args)
	}
}

package tools

import (
	"context"
	"testing"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.tools)
	assert.Equal(t, 0, len(registry.tools))
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	// Register a tool
	searchTool := NewSearchTool(mockDB)
	registry.Register(searchTool)

	// Verify tool was registered
	tool, ok := registry.Get("search_documents")
	assert.True(t, ok)
	assert.NotNil(t, tool)
	assert.Equal(t, "search_documents", tool.Definition().Name)
}

func TestRegistryRegisterMultipleTools(t *testing.T) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	// Register multiple tools
	registry.Register(NewSearchTool(mockDB))
	registry.Register(NewRetrieveTool(mockDB))
	registry.Register(NewListTool(mockDB))
	registry.Register(NewHybridSearchTool(mockDB))

	// Verify all tools were registered
	searchTool, ok := registry.Get("search_documents")
	assert.True(t, ok)
	assert.NotNil(t, searchTool)

	retrieveTool, ok := registry.Get("retrieve_document")
	assert.True(t, ok)
	assert.NotNil(t, retrieveTool)

	listTool, ok := registry.Get("list_documents")
	assert.True(t, ok)
	assert.NotNil(t, listTool)

	hybridTool, ok := registry.Get("hybrid_search")
	assert.True(t, ok)
	assert.NotNil(t, hybridTool)
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	searchTool := NewSearchTool(mockDB)
	registry.Register(searchTool)

	tests := []struct {
		name     string
		toolName string
		wantOk   bool
	}{
		{
			name:     "existing tool",
			toolName: "search_documents",
			wantOk:   true,
		},
		{
			name:     "non-existent tool",
			toolName: "unknown_tool",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, ok := registry.Get(tt.toolName)
			assert.Equal(t, tt.wantOk, ok)
			if tt.wantOk {
				assert.NotNil(t, tool)
			} else {
				assert.Nil(t, tool)
			}
		})
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	// Empty registry
	tools := registry.List()
	assert.NotNil(t, tools)
	assert.Equal(t, 0, len(tools))

	// Register tools
	registry.Register(NewSearchTool(mockDB))
	registry.Register(NewRetrieveTool(mockDB))
	registry.Register(NewListTool(mockDB))

	// List tools
	tools = registry.List()
	assert.Equal(t, 3, len(tools))

	// Verify all tool names are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["search_documents"])
	assert.True(t, toolNames["retrieve_document"])
	assert.True(t, toolNames["list_documents"])
}

func TestRegistryExecute(t *testing.T) {
	mockDB := new(MockStore)
	registry := NewRegistry()

	// Register search tool
	searchTool := NewSearchTool(mockDB)
	registry.Register(searchTool)

	ctx := context.WithValue(context.Background(), auth.ContextKeyTenantID, "tenant-123")

	t.Run("successful execute", func(t *testing.T) {
		// Setup mock
		mockDB.On("SearchDocuments", ctx, "tenant-123", "test", 10).
			Return([]*database.Document{}, nil).Once()

		// Execute tool
		result, err := registry.Execute(ctx, "search_documents", map[string]interface{}{
			"query": "test",
			"limit": 10,
		})

		require.NoError(t, err)
		assert.False(t, result.IsError)
		mockDB.AssertExpectations(t)
	})

	t.Run("tool not found", func(t *testing.T) {
		// Execute non-existent tool
		result, err := registry.Execute(ctx, "unknown_tool", map[string]interface{}{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool not found")
		assert.True(t, result.IsError)
	})
}

// Benchmark tests
func BenchmarkRegistryGet(b *testing.B) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	// Register multiple tools
	registry.Register(NewSearchTool(mockDB))
	registry.Register(NewRetrieveTool(mockDB))
	registry.Register(NewListTool(mockDB))
	registry.Register(NewHybridSearchTool(mockDB))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Get("search_documents")
	}
}

func BenchmarkRegistryList(b *testing.B) {
	registry := NewRegistry()
	mockDB := new(MockStore)

	// Register multiple tools
	registry.Register(NewSearchTool(mockDB))
	registry.Register(NewRetrieveTool(mockDB))
	registry.Register(NewListTool(mockDB))
	registry.Register(NewHybridSearchTool(mockDB))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.List()
	}
}

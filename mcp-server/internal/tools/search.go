package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// SearchTool implements document text search
type SearchTool struct {
	db database.Store
}

// NewSearchTool creates a new search tool
func NewSearchTool(db database.Store) *SearchTool {
	return &SearchTool{db: db}
}

// Definition returns the tool definition for MCP
func (t *SearchTool) Definition() protocol.Tool {
	return protocol.Tool{
		Name:        "search_documents",
		Description: "Search documents by text query. Searches across title, content, and metadata fields.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query text",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results to return (default: 10, max: 100)",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	}
}

// SearchParams represents the parameters for search
type SearchParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

// Execute performs the search operation
func (t *SearchTool) Execute(ctx context.Context, args map[string]interface{}) (protocol.ToolCallResult, error) {
	// Extract tenant ID from context
	tenantID, err := auth.ExtractTenantID(ctx)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("authentication required: %w", err)
	}

	// Parse parameters
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("invalid arguments: %w", err)
	}

	var params SearchParams
	if err := json.Unmarshal(argsJSON, &params); err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate parameters
	if params.Query == "" {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("query is required")
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Perform search
	documents, err := t.db.SearchDocuments(ctx, tenantID, params.Query, params.Limit)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("search failed: %w", err)
	}

	// Format results
	var resultText string
	if len(documents) == 0 {
		resultText = fmt.Sprintf("No documents found matching query: %s", params.Query)
	} else {
		resultText = fmt.Sprintf("Found %d document(s) matching query: %s\n\n", len(documents), params.Query)
		for i, doc := range documents {
			resultText += fmt.Sprintf("Document %d:\n", i+1)
			resultText += fmt.Sprintf("  ID: %s\n", doc.ID)
			resultText += fmt.Sprintf("  Title: %s\n", doc.Title)
			resultText += fmt.Sprintf("  Content Preview: %.200s...\n", doc.Content)
			if doc.Metadata != nil {
				metadataJSON, _ := json.Marshal(doc.Metadata)
				resultText += fmt.Sprintf("  Metadata: %s\n", string(metadataJSON))
			}
			resultText += fmt.Sprintf("  Created: %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05"))
			resultText += "\n"
		}
	}

	return protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			{
				Type: "text",
				Text: resultText,
			},
		},
		IsError: false,
	}, nil
}

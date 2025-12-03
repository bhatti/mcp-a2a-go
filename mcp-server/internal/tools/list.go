package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// ListTool implements document listing
type ListTool struct {
	db database.Store
}

// NewListTool creates a new list tool
func NewListTool(db database.Store) *ListTool {
	return &ListTool{db: db}
}

// Definition returns the tool definition for MCP
func (t *ListTool) Definition() protocol.Tool {
	return protocol.Tool{
		Name:        "list_documents",
		Description: "List all documents for the current tenant with pagination support.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of documents to return (default: 20, max: 100)",
					"default":     20,
				},
				"offset": map[string]interface{}{
					"type":        "number",
					"description": "Number of documents to skip (default: 0)",
					"default":     0,
				},
			},
		},
	}
}

// ListParams represents the parameters for list
type ListParams struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Execute lists documents
func (t *ListTool) Execute(ctx context.Context, args map[string]interface{}) (protocol.ToolCallResult, error) {
	// Extract tenant ID from context
	tenantID, err := auth.ExtractTenantID(ctx)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("authentication required: %w", err)
	}

	// Parse parameters
	var params ListParams
	if len(args) > 0 {
		argsJSON, err := json.Marshal(args)
		if err != nil {
			return protocol.ToolCallResult{IsError: true}, fmt.Errorf("invalid arguments: %w", err)
		}

		if err := json.Unmarshal(argsJSON, &params); err != nil {
			return protocol.ToolCallResult{IsError: true}, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	// Set defaults
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	// List documents
	documents, err := t.db.ListDocuments(ctx, tenantID, params.Limit, params.Offset)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("failed to list documents: %w", err)
	}

	// Format results
	var resultText string
	if len(documents) == 0 {
		resultText = "No documents found."
	} else {
		resultText = fmt.Sprintf("Found %d document(s) (offset: %d, limit: %d):\n\n", len(documents), params.Offset, params.Limit)
		for i, doc := range documents {
			resultText += fmt.Sprintf("%d. %s\n", i+1+params.Offset, doc.Title)
			resultText += fmt.Sprintf("   ID: %s\n", doc.ID)
			resultText += fmt.Sprintf("   Preview: %.100s...\n", doc.Content)
			if doc.Metadata != nil {
				if category, ok := doc.Metadata["category"].(string); ok {
					resultText += fmt.Sprintf("   Category: %s\n", category)
				}
			}
			resultText += fmt.Sprintf("   Created: %s\n", doc.CreatedAt.Format("2006-01-02"))
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

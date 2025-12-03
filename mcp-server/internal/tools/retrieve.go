package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// RetrieveTool implements document retrieval by ID
type RetrieveTool struct {
	db database.Store
}

// NewRetrieveTool creates a new retrieve tool
func NewRetrieveTool(db database.Store) *RetrieveTool {
	return &RetrieveTool{db: db}
}

// Definition returns the tool definition for MCP
func (t *RetrieveTool) Definition() protocol.Tool {
	return protocol.Tool{
		Name:        "retrieve_document",
		Description: "Retrieve a specific document by its ID. Returns the full document content and metadata.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"document_id": map[string]interface{}{
					"type":        "string",
					"description": "The unique identifier of the document to retrieve",
				},
			},
			"required": []string{"document_id"},
		},
	}
}

// RetrieveParams represents the parameters for retrieve
type RetrieveParams struct {
	DocumentID string `json:"document_id"`
}

// Execute retrieves a document by ID
func (t *RetrieveTool) Execute(ctx context.Context, args map[string]interface{}) (protocol.ToolCallResult, error) {
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

	var params RetrieveParams
	if err := json.Unmarshal(argsJSON, &params); err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("invalid arguments: %w", err)
	}

	if params.DocumentID == "" {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("document_id is required")
	}

	// Retrieve document
	doc, err := t.db.GetDocument(ctx, tenantID, params.DocumentID)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("failed to retrieve document: %w", err)
	}

	// Format result
	metadataJSON, _ := json.Marshal(doc.Metadata)
	resultText := fmt.Sprintf("Document Retrieved:\n\n")
	resultText += fmt.Sprintf("ID: %s\n", doc.ID)
	resultText += fmt.Sprintf("Title: %s\n", doc.Title)
	resultText += fmt.Sprintf("Content:\n%s\n\n", doc.Content)
	resultText += fmt.Sprintf("Metadata: %s\n", string(metadataJSON))
	resultText += fmt.Sprintf("Created: %s\n", doc.CreatedAt.Format("2006-01-02 15:04:05"))
	resultText += fmt.Sprintf("Updated: %s\n", doc.UpdatedAt.Format("2006-01-02 15:04:05"))
	if doc.CreatedBy != nil && *doc.CreatedBy != "" {
		resultText += fmt.Sprintf("Created By: %s\n", *doc.CreatedBy)
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

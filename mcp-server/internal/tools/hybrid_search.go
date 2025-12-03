package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
)

// HybridSearchTool implements hybrid BM25 + vector search
type HybridSearchTool struct {
	db database.Store
}

// NewHybridSearchTool creates a new hybrid search tool
func NewHybridSearchTool(db database.Store) *HybridSearchTool {
	return &HybridSearchTool{db: db}
}

// Definition returns the tool definition for MCP
func (t *HybridSearchTool) Definition() protocol.Tool {
	return protocol.Tool{
		Name:        "hybrid_search",
		Description: "Perform hybrid search combining BM25 lexical search with vector semantic similarity. Returns the most relevant documents using both keyword matching and semantic understanding.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query text",
				},
				"embedding": map[string]interface{}{
					"type":        "array",
					"description": "Query embedding vector (1536 dimensions for OpenAI ada-002)",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results to return (default: 10, max: 50)",
					"default":     10,
				},
				"bm25_weight": map[string]interface{}{
					"type":        "number",
					"description": "Weight for BM25 lexical search (0.0 to 1.0, default: 0.5)",
					"default":     0.5,
				},
				"vector_weight": map[string]interface{}{
					"type":        "number",
					"description": "Weight for vector semantic search (0.0 to 1.0, default: 0.5)",
					"default":     0.5,
				},
			},
			"required": []string{"query"},
		},
	}
}

// HybridSearchParams represents the parameters for hybrid search
type HybridSearchParams struct {
	Query        string    `json:"query"`
	Embedding    []float32 `json:"embedding,omitempty"`
	Limit        int       `json:"limit"`
	BM25Weight   float64   `json:"bm25_weight"`
	VectorWeight float64   `json:"vector_weight"`
}

// Execute performs the hybrid search operation
func (t *HybridSearchTool) Execute(ctx context.Context, args map[string]interface{}) (protocol.ToolCallResult, error) {
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

	var params HybridSearchParams
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
	if params.Limit > 50 {
		params.Limit = 50
	}
	if params.BM25Weight == 0 && params.VectorWeight == 0 {
		params.BM25Weight = 0.5
		params.VectorWeight = 0.5
	}

	// Perform hybrid search
	dbParams := database.HybridSearchParams{
		Query:        params.Query,
		Embedding:    params.Embedding,
		Limit:        params.Limit,
		BM25Weight:   params.BM25Weight,
		VectorWeight: params.VectorWeight,
		MinBM25Score: 0.0,
		MinVectorSim: 0.0,
	}

	results, err := t.db.SimpleHybridSearch(ctx, tenantID, dbParams)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Format results as JSON for UI consumption
	type DocumentResult struct {
		DocID       string                 `json:"doc_id"`
		TenantID    string                 `json:"tenant_id"`
		Title       string                 `json:"title"`
		Content     string                 `json:"content"`
		Score       float64                `json:"score"`
		BM25Score   float64                `json:"bm25_score"`
		VectorScore float64                `json:"vector_score"`
		BM25Rank    int                    `json:"bm25_rank"`
		VectorRank  int                    `json:"vector_rank"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		CreatedAt   string                 `json:"created_at"`
	}

	var jsonResults []DocumentResult
	for i, result := range results {
		doc := result.Document
		jsonResults = append(jsonResults, DocumentResult{
			DocID:       doc.ID,
			TenantID:    doc.TenantID,
			Title:       doc.Title,
			Content:     doc.Content,
			Score:       result.CombinedScore,
			BM25Score:   result.BM25Score,
			VectorScore: result.VectorScore,
			BM25Rank:    i + 1,
			VectorRank:  i + 1,
			Metadata:    doc.Metadata,
			CreatedAt:   doc.CreatedAt.Format(time.RFC3339),
		})
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(jsonResults)
	if err != nil {
		return protocol.ToolCallResult{IsError: true}, fmt.Errorf("failed to marshal results: %w", err)
	}

	return protocol.ToolCallResult{
		Content: []protocol.ContentBlock{
			{
				Type: "text",
				Text: string(jsonData),
			},
		},
		IsError: false,
	}, nil
}

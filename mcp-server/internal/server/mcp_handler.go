package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	// "github.com/bhatti/mcp-a2a-go/mcp-server/internal/observability"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/tools"
	// "go.opentelemetry.io/otel/attribute"
	// "go.opentelemetry.io/otel/trace"
)

const (
	MCPProtocolVersion = "2024-11-05"
	ServerName         = "mcp-rag-server"
	ServerVersion      = "1.0.0"
)

// MCPHandler handles MCP JSON-RPC requests
type MCPHandler struct {
	toolRegistry *tools.Registry
	telemetry    interface{} // Disabled temporarily - was *observability.Telemetry
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(toolRegistry *tools.Registry, telemetry interface{}) *MCPHandler {
	return &MCPHandler{
		toolRegistry: toolRegistry,
		telemetry:    telemetry,
	}
}

// ServeHTTP implements http.Handler
func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.sendErrorResponse(w, nil, protocol.ParseError, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var req protocol.Request
	if err := json.Unmarshal(body, &req); err != nil {
		h.sendErrorResponse(w, nil, protocol.ParseError, "Invalid JSON")
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.sendErrorResponse(w, req.ID, protocol.InvalidRequest, err.Error())
		return
	}

	// Start tracing span - DISABLED TEMPORARILY
	// spanCtx, span := h.telemetry.StartSpan(ctx, "mcp.request",
	// 	attribute.String("method", req.Method),
	// 	attribute.String("request_id", fmt.Sprintf("%v", req.ID)),
	// )
	// defer span.End()

	// Handle the request
	response := h.handleRequest(ctx, &req)

	// Record metrics - DISABLED TEMPORARILY
	duration := time.Since(startTime)
	_ = duration // Avoid unused variable warning
	// success := response.Error == nil
	// h.telemetry.RecordRequest(ctx, req.Method, duration, success)

	// Send response
	h.sendResponse(w, response)
}

// handleRequest processes a JSON-RPC request and returns a response
func (h *MCPHandler) handleRequest(ctx context.Context, req *protocol.Request) *protocol.Response {
	switch req.Method {
	case protocol.MethodInitialize:
		return h.handleInitialize(ctx, req)
	case protocol.MethodToolsList:
		return h.handleToolsList(ctx, req)
	case protocol.MethodToolsCall:
		return h.handleToolsCall(ctx, req)
	default:
		return protocol.NewErrorResponse(req.ID, protocol.MethodNotFound,
			fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

// handleInitialize handles the initialize request
func (h *MCPHandler) handleInitialize(ctx context.Context, req *protocol.Request) *protocol.Response {
	var initReq protocol.InitializeRequest
	if err := req.ParseParams(&initReq); err != nil {
		return protocol.NewErrorResponse(req.ID, protocol.InvalidParams,
			"Invalid initialize params: "+err.Error(), nil)
	}

	result := protocol.InitializeResult{
		ProtocolVersion: MCPProtocolVersion,
		Capabilities: protocol.ServerCapabilities{
			Tools: &protocol.ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: protocol.ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
	}

	return protocol.NewResponse(req.ID, result)
}

// handleToolsList handles the tools/list request
func (h *MCPHandler) handleToolsList(ctx context.Context, req *protocol.Request) *protocol.Response {
	tools := h.toolRegistry.List()

	result := protocol.ToolsListResult{
		Tools: tools,
	}

	return protocol.NewResponse(req.ID, result)
}

// handleToolsCall handles the tools/call request
func (h *MCPHandler) handleToolsCall(ctx context.Context, req *protocol.Request) *protocol.Response {
	var toolReq protocol.ToolCallRequest
	if err := req.ParseParams(&toolReq); err != nil {
		return protocol.NewErrorResponse(req.ID, protocol.InvalidParams,
			"Invalid tool call params: "+err.Error(), nil)
	}

	// Start tool call span - DISABLED TEMPORARILY
	// spanCtx, span := h.telemetry.StartSpan(ctx, "mcp.tool.call",
	// 	attribute.String("tool", toolReq.Name),
	// )
	// defer span.End()

	startTime := time.Now()
	_ = startTime // Avoid unused variable warning

	// Execute tool
	result, err := h.toolRegistry.Execute(ctx, toolReq.Name, toolReq.Arguments)
	if err != nil {
		// Record metrics - DISABLED TEMPORARILY
		// h.telemetry.RecordToolCall(ctx, toolReq.Name, time.Since(startTime), false)
		// span.SetStatus(trace.StatusError, err.Error())

		return protocol.NewErrorResponse(req.ID, protocol.InternalError,
			fmt.Sprintf("Tool execution failed: %s", err.Error()), nil)
	}

	// Record metrics - DISABLED TEMPORARILY
	// h.telemetry.RecordToolCall(ctx, toolReq.Name, time.Since(startTime), !result.IsError)

	// if result.IsError {
	// 	span.SetStatus(trace.StatusError, "tool returned error")
	// }

	return protocol.NewResponse(req.ID, result)
}

// sendResponse sends a JSON-RPC response
func (h *MCPHandler) sendResponse(w http.ResponseWriter, response *protocol.Response) {
	w.Header().Set("Content-Type", "application/json")

	// Set HTTP status based on error type
	// JSON-RPC 2.0 protocol errors return HTTP 200 (the HTTP request succeeded)
	// MCP application errors use semantic HTTP status codes
	if response.Error != nil {
		switch response.Error.Code {
		// MCP application-level errors - use semantic HTTP codes
		case protocol.AuthenticationRequired, protocol.AuthorizationFailed:
			w.WriteHeader(http.StatusUnauthorized)
		case protocol.RateLimitExceeded:
			w.WriteHeader(http.StatusTooManyRequests)
		case protocol.ResourceNotFound:
			w.WriteHeader(http.StatusNotFound)
		case protocol.ValidationError:
			w.WriteHeader(http.StatusBadRequest)
		// Standard JSON-RPC protocol errors - return HTTP 200
		case protocol.ParseError, protocol.InvalidRequest, protocol.MethodNotFound,
			protocol.InvalidParams, protocol.InternalError, protocol.ServerError:
			w.WriteHeader(http.StatusOK)
		default:
			// Unknown errors default to 500
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// sendErrorResponse sends a JSON-RPC error response
func (h *MCPHandler) sendErrorResponse(w http.ResponseWriter, id interface{}, code int, message string) {
	response := protocol.NewErrorResponse(id, code, message, nil)
	h.sendResponse(w, response)
}

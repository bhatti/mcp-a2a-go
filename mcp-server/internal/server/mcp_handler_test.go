package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/auth"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/database"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStore implements database.Store for testing
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

func TestNewMCPHandler(t *testing.T) {
	mockDB := new(MockStore)
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchTool(mockDB))

	handler := NewMCPHandler(registry, nil)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.toolRegistry)
}

func TestMCPHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Test GET request (should be rejected)
	req := httptest.NewRequest("GET", "/mcp", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestMCPHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code) // JSON-RPC errors return 200

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.ParseError, response.Error.Code)
}

func TestMCPHandler_ServeHTTP_InvalidRequest(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Send request with missing required fields
	reqBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		// Missing "method" field
	})

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.InvalidRequest, response.Error.Code)
}

func TestMCPHandler_Initialize(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Create initialize request
	initReq, err := protocol.NewRequest("1", protocol.MethodInitialize, protocol.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		ClientInfo: protocol.ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})
	require.NoError(t, err)

	reqBody, err := json.Marshal(initReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.Response
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	// Verify result contains server info
	resultJSON, _ := json.Marshal(response.Result)
	var initResult protocol.InitializeResult
	json.Unmarshal(resultJSON, &initResult)

	assert.Equal(t, MCPProtocolVersion, initResult.ProtocolVersion)
	assert.Equal(t, ServerName, initResult.ServerInfo.Name)
	assert.Equal(t, ServerVersion, initResult.ServerInfo.Version)
	assert.NotNil(t, initResult.Capabilities.Tools)
}

func TestMCPHandler_ToolsList(t *testing.T) {
	mockDB := new(MockStore)
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchTool(mockDB))
	registry.Register(tools.NewRetrieveTool(mockDB))

	handler := NewMCPHandler(registry, nil)

	// Create tools/list request
	listReq, err := protocol.NewRequest("2", protocol.MethodToolsList, nil)
	require.NoError(t, err)

	reqBody, err := json.Marshal(listReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.Response
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	// Verify result contains tools
	resultJSON, _ := json.Marshal(response.Result)
	var listResult protocol.ToolsListResult
	json.Unmarshal(resultJSON, &listResult)

	assert.Len(t, listResult.Tools, 2)
	toolNames := []string{listResult.Tools[0].Name, listResult.Tools[1].Name}
	assert.Contains(t, toolNames, "search_documents")
	assert.Contains(t, toolNames, "retrieve_document")
}

func TestMCPHandler_ToolsCall_Success(t *testing.T) {
	mockDB := new(MockStore)

	// Setup mock to return documents
	mockDB.On("SearchDocuments", mock.Anything, "tenant-123", "test query", 10).
		Return([]*database.Document{
			{ID: "doc-1", Title: "Test Doc", Content: "Test content"},
		}, nil)

	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchTool(mockDB))

	handler := NewMCPHandler(registry, nil)

	// Create tools/call request
	callReq, err := protocol.NewRequest("3", protocol.MethodToolsCall, protocol.ToolCallRequest{
		Name: "search_documents",
		Arguments: map[string]interface{}{
			"query": "test query",
			"limit": 10,
		},
	})
	require.NoError(t, err)

	reqBody, err := json.Marshal(callReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	ctx := context.WithValue(req.Context(), auth.ContextKeyTenantID, "tenant-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.Response
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Nil(t, response.Error)
	assert.NotNil(t, response.Result)

	mockDB.AssertExpectations(t)
}

func TestMCPHandler_ToolsCall_ToolNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Call non-existent tool
	callReq, err := protocol.NewRequest("4", protocol.MethodToolsCall, protocol.ToolCallRequest{
		Name:      "nonexistent_tool",
		Arguments: map[string]interface{}{},
	})
	require.NoError(t, err)

	reqBody, err := json.Marshal(callReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var response protocol.Response
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.InternalError, response.Error.Code)
	assert.Contains(t, response.Error.Message, "tool not found")
}

func TestMCPHandler_ToolsCall_InvalidParams(t *testing.T) {
	mockDB := new(MockStore)
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchTool(mockDB))

	handler := NewMCPHandler(registry, nil)

	// Create tools/call with invalid params structure
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": "5",
		"method": "tools/call",
		"params": "invalid-params-type"
	}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.InvalidParams, response.Error.Code)
}

func TestMCPHandler_MethodNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Call unknown method
	unknownReq, err := protocol.NewRequest("6", "unknown/method", nil)
	require.NoError(t, err)

	reqBody, err := json.Marshal(unknownReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	var response protocol.Response
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.MethodNotFound, response.Error.Code)
	assert.Contains(t, response.Error.Message, "unknown/method")
}

func TestMCPHandler_ResponseHeaders(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	initReq, err := protocol.NewRequest("1", protocol.MethodInitialize, protocol.InitializeRequest{
		ProtocolVersion: "2024-11-05",
	})
	require.NoError(t, err)

	reqBody, _ := json.Marshal(initReq)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

// Benchmark tests
func BenchmarkMCPHandler_Initialize(b *testing.B) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	initReq, err := protocol.NewRequest("1", protocol.MethodInitialize, protocol.InitializeRequest{
		ProtocolVersion: "2024-11-05",
	})
	if err != nil {
		b.Fatal(err)
	}

	reqBody, _ := json.Marshal(initReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

func TestMCPHandler_Initialize_InvalidParams(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	// Create initialize request with invalid params (wrong type)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(`{
		"jsonrpc": "2.0",
		"id": "1",
		"method": "initialize",
		"params": "invalid-should-be-object"
	}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.Response
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, protocol.InvalidParams, response.Error.Code)
}

func TestMCPHandler_SendResponse_AuthError(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	rr := httptest.NewRecorder()
	response := protocol.NewErrorResponse("1", protocol.AuthenticationRequired, "Auth required", nil)

	handler.sendResponse(rr, response)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestMCPHandler_SendResponse_RateLimitError(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	rr := httptest.NewRecorder()
	response := protocol.NewErrorResponse("1", protocol.RateLimitExceeded, "Rate limit exceeded", nil)

	handler.sendResponse(rr, response)

	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestMCPHandler_SendResponse_NotFoundError(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	rr := httptest.NewRecorder()
	response := protocol.NewErrorResponse("1", protocol.ResourceNotFound, "Not found", nil)

	handler.sendResponse(rr, response)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMCPHandler_SendResponse_ValidationError(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	rr := httptest.NewRecorder()
	response := protocol.NewErrorResponse("1", protocol.ValidationError, "Validation failed", nil)

	handler.sendResponse(rr, response)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestMCPHandler_SendResponse_UnknownError(t *testing.T) {
	registry := tools.NewRegistry()
	handler := NewMCPHandler(registry, nil)

	rr := httptest.NewRecorder()
	// Use an error code that doesn't match any known cases
	response := protocol.NewErrorResponse("1", -99999, "Unknown error", nil)

	handler.sendResponse(rr, response)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func BenchmarkMCPHandler_ToolsList(b *testing.B) {
	mockDB := new(MockStore)
	registry := tools.NewRegistry()
	registry.Register(tools.NewSearchTool(mockDB))
	registry.Register(tools.NewRetrieveTool(mockDB))
	registry.Register(tools.NewListTool(mockDB))
	registry.Register(tools.NewHybridSearchTool(mockDB))

	handler := NewMCPHandler(registry, nil)

	listReq, err := protocol.NewRequest("2", protocol.MethodToolsList, nil)
	if err != nil {
		b.Fatal(err)
	}
	reqBody, _ := json.Marshal(listReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

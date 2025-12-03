package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/agentcard"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/cost"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer() *Server {
	return &Server{
		taskStore:     tasks.NewMemoryStore(),
		agentStore:    agentcard.NewStore(),
		costTracker:   cost.NewTracker(),
		budgetManager: cost.NewBudgetManager(),
	}
}

func TestServer_GetAgentCard(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Register an agent
	card := protocol.NewAgentCard("test-agent", "Test Agent", "1.0.0", "A test agent")
	card.AddCapability(protocol.Capability{
		Name:        "search",
		Description: "Search capability",
	})
	server.agentStore.Register(ctx, card)

	// Make request
	req := httptest.NewRequest("GET", "/agent", nil)
	rr := httptest.NewRecorder()

	server.handleGetAgentCard(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.AgentCard
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "test-agent", response.ID)
	assert.Len(t, response.Capabilities, 1)
}

func TestServer_CreateTask(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Register agent
	card := protocol.NewAgentCard("test-agent", "Test Agent", "1.0.0", "Test")
	card.AddCapability(protocol.Capability{Name: "search"})
	server.agentStore.Register(ctx, card)

	// Set budget
	server.budgetManager.SetBudget(ctx, "user-1", 10.0)

	// Create task request
	reqBody := map[string]interface{}{
		"user_id":    "user-1",
		"agent_id":   "test-agent",
		"capability": "search",
		"input": map[string]interface{}{
			"query": "test query",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleCreateTask(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)

	var response protocol.Task
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "test-agent", response.AgentID)
	assert.Equal(t, "search", response.Capability)
	assert.Equal(t, protocol.TaskStatePending, response.State)
}

func TestServer_CreateTask_InvalidJSON(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleCreateTask(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestServer_CreateTask_BudgetExceeded(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Register agent
	card := protocol.NewAgentCard("test-agent", "Test", "1.0.0", "Test")
	card.AddCapability(protocol.Capability{Name: "search"})
	server.agentStore.Register(ctx, card)

	// Set low budget
	server.budgetManager.SetBudget(ctx, "user-1", 0.001)

	reqBody := map[string]interface{}{
		"user_id":    "user-1",
		"agent_id":   "test-agent",
		"capability": "search",
		"input":      map[string]interface{}{"query": "test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleCreateTask(rr, req)

	assert.Equal(t, http.StatusPaymentRequired, rr.Code)
}

func TestServer_GetTask(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Create a task
	task := protocol.NewTask("agent-1", "search", map[string]interface{}{"query": "test"})
	server.taskStore.Create(ctx, task)

	req := httptest.NewRequest("GET", "/tasks/"+task.ID, nil)
	rr := httptest.NewRecorder()

	server.handleGetTask(rr, req, task.ID)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response protocol.Task
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, task.ID, response.ID)
}

func TestServer_GetTask_NotFound(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("GET", "/tasks/non-existent", nil)
	rr := httptest.NewRecorder()

	server.handleGetTask(rr, req, "non-existent")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServer_ListTasks(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Create multiple tasks
	task1 := protocol.NewTask("agent-1", "search", nil)
	task2 := protocol.NewTask("agent-1", "analyze", nil)
	server.taskStore.Create(ctx, task1)
	server.taskStore.Create(ctx, task2)

	req := httptest.NewRequest("GET", "/tasks", nil)
	rr := httptest.NewRecorder()

	server.handleListTasks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response []protocol.Task
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
}

func TestServer_ListTasks_WithAgentFilter(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Create tasks for different agents
	task1 := protocol.NewTask("agent-1", "search", nil)
	task2 := protocol.NewTask("agent-2", "analyze", nil)
	server.taskStore.Create(ctx, task1)
	server.taskStore.Create(ctx, task2)

	req := httptest.NewRequest("GET", "/tasks?agent_id=agent-1", nil)
	rr := httptest.NewRecorder()

	server.handleListTasks(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response []protocol.Task
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "agent-1", response[0].AgentID)
}

func TestServer_CancelTask(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Create a task
	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	req := httptest.NewRequest("DELETE", "/tasks/"+task.ID, nil)
	rr := httptest.NewRecorder()

	server.handleCancelTask(rr, req, task.ID)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify task was cancelled
	retrieved, _ := server.taskStore.Get(ctx, task.ID)
	assert.Equal(t, protocol.TaskStateCancelled, retrieved.State)
}

func TestServer_CancelTask_NotFound(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("DELETE", "/tasks/non-existent", nil)
	rr := httptest.NewRecorder()

	server.handleCancelTask(rr, req, "non-existent")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServer_CancelTask_AlreadyCompleted(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Create a completed task
	task := protocol.NewTask("agent-1", "search", nil)
	task.SetResult(map[string]interface{}{"data": "result"})
	server.taskStore.Create(ctx, task)

	req := httptest.NewRequest("DELETE", "/tasks/"+task.ID, nil)
	rr := httptest.NewRecorder()

	server.handleCancelTask(rr, req, task.ID)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestServer_HealthCheck(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

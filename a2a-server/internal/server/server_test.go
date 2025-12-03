package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/agentcard"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/cost"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/tasks"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	taskStore := tasks.NewMemoryStore()
	agentStore := agentcard.NewStore()
	costTracker := cost.NewTracker()
	budgetManager := cost.NewBudgetManager()
	agentCard := protocol.NewAgentCard("test", "Test", "1.0.0", "Test")

	server := NewServer(taskStore, agentStore, costTracker, budgetManager, agentCard)

	assert.NotNil(t, server)
	assert.NotNil(t, server.taskStore)
	assert.NotNil(t, server.agentStore)
	assert.NotNil(t, server.costTracker)
	assert.NotNil(t, server.budgetManager)
	assert.Equal(t, agentCard, server.agentCard)
}

func TestServer_CreateTask_AgentNotFound(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Don't register agent
	server.budgetManager.SetBudget(ctx, "user-1", 10.0)

	reqBody := map[string]interface{}{
		"user_id":    "user-1",
		"agent_id":   "non-existent",
		"capability": "search",
		"input":      map[string]interface{}{"query": "test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleCreateTask(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServer_CreateTask_BudgetNotConfigured(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Register agent but don't set budget
	card := protocol.NewAgentCard("agent-1", "Test", "1.0.0", "Test")
	card.AddCapability(protocol.Capability{Name: "search"})
	server.agentStore.Register(ctx, card)

	reqBody := map[string]interface{}{
		"user_id":    "user-without-budget",
		"agent_id":   "agent-1",
		"capability": "search",
		"input":      map[string]interface{}{"query": "test"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/tasks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleCreateTask(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestServer_GetAgentCard_NoAgentRegistered(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("GET", "/agent", nil)
	rr := httptest.NewRecorder()

	server.handleGetAgentCard(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServer_Routes_TaskEvents_SSE(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/tasks/"+task.ID+"/events", nil)
	rr := httptest.NewRecorder()

	// Test SSE endpoint routing
	go mux.ServeHTTP(rr, req)

	// Wait for handler to start
	time.Sleep(20 * time.Millisecond)

	// Publish event
	server.taskStore.PublishEvent(ctx, protocol.TaskEvent{
		TaskID:    task.ID,
		State:     protocol.TaskStateRunning,
		Message:   "Test",
		Timestamp: time.Now(),
	})

	time.Sleep(30 * time.Millisecond)

	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
}

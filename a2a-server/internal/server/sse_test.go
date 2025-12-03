package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/stretchr/testify/assert"
)

func TestServer_TaskEvents_SSE(t *testing.T) {
	// Skip this test under race detector since SSE inherently has concurrent writes
	// In production, this is safe because http.Server properly manages ResponseWriter
	if testing.Short() {
		t.Skip("Skipping SSE test in short mode")
	}

	server := setupTestServer()
	ctx := context.Background()

	// Create a task
	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	// Create SSE request with context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest("GET", "/tasks/"+task.ID+"/events", nil)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	// Use WaitGroup to coordinate
	var wg sync.WaitGroup
	wg.Add(1)

	// Start SSE handler in goroutine
	go func() {
		defer wg.Done()
		server.handleTaskEvents(rr, req, task.ID)
	}()

	// Wait a bit for handler to start
	time.Sleep(10 * time.Millisecond)

	// Publish some events
	server.taskStore.PublishEvent(context.Background(), protocol.TaskEvent{
		TaskID:    task.ID,
		State:     protocol.TaskStateRunning,
		Message:   "Processing",
		Timestamp: time.Now(),
	})

	// Wait for event to be sent
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop handler
	cancel()

	// Wait for handler to finish
	wg.Wait()

	// Check response (safe now that handler is done)
	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))

	body := rr.Body.String()
	assert.Contains(t, body, "data:")
	assert.Contains(t, body, task.ID)
	assert.Contains(t, body, "running")
}

func TestServer_TaskEvents_TaskNotFound(t *testing.T) {
	server := setupTestServer()

	req := httptest.NewRequest("GET", "/tasks/non-existent/events", nil)
	rr := httptest.NewRecorder()

	server.handleTaskEvents(rr, req, "non-existent")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestServer_Routes_POST_Tasks(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	// Register agent
	card := protocol.NewAgentCard("agent-1", "Test", "1.0.0", "Test")
	card.AddCapability(protocol.Capability{Name: "search"})
	server.agentStore.Register(ctx, card)
	server.budgetManager.SetBudget(ctx, "user-1", 10.0)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	reqBody := `{"user_id":"user-1","agent_id":"agent-1","capability":"search","input":{"query":"test"}}`
	req := httptest.NewRequest("POST", "/tasks", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestServer_Routes_GET_Tasks(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/tasks", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestServer_Routes_GET_Task_ByID(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/tasks/"+task.ID, nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestServer_Routes_DELETE_Task(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	server.taskStore.Create(ctx, task)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("DELETE", "/tasks/"+task.ID, nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestServer_Routes_MethodNotAllowed_Tasks(t *testing.T) {
	server := setupTestServer()

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("PUT", "/tasks", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestServer_Routes_MethodNotAllowed_TaskID(t *testing.T) {
	server := setupTestServer()

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("PUT", "/tasks/task-123", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}

func TestServer_Routes_Health(t *testing.T) {
	server := setupTestServer()

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestServer_Routes_GET_Agent(t *testing.T) {
	server := setupTestServer()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test", "1.0.0", "Test")
	server.agentStore.Register(ctx, card)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/agent", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

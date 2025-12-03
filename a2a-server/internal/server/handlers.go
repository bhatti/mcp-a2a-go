package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
)

// CreateTaskRequest represents a request to create a task
type CreateTaskRequest struct {
	UserID     string                 `json:"user_id"`
	AgentID    string                 `json:"agent_id"`
	Capability string                 `json:"capability"`
	Input      map[string]interface{} `json:"input"`
}

// handleGetAgentCard handles GET /agent requests
func (s *Server) handleGetAgentCard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the first registered agent (in production, this would be based on agent ID)
	cards := s.agentStore.List(ctx)
	if len(cards) == 0 {
		http.Error(w, "No agent registered", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards[0])
}

// handleCreateTask handles POST /tasks requests
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate agent exists
	_, err := s.agentStore.Get(ctx, req.AgentID)
	if err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Estimate cost (simplified - use fixed estimate for demo)
	estimatedCost := 0.01 // $0.01 per task

	// Check budget
	allowed, err := s.budgetManager.CheckAndUpdate(ctx, req.UserID, estimatedCost)
	if err != nil {
		http.Error(w, "Budget not configured", http.StatusBadRequest)
		return
	}
	if !allowed {
		http.Error(w, "Budget exceeded", http.StatusPaymentRequired)
		return
	}

	// Create task
	task := protocol.NewTask(req.AgentID, req.Capability, req.Input)
	if err := s.taskStore.Create(ctx, task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// handleGetTask handles GET /tasks/{id} requests
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, taskID string) {
	ctx := r.Context()

	task, err := s.taskStore.Get(ctx, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleListTasks handles GET /tasks requests
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	agentID := r.URL.Query().Get("agent_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	tasks, err := s.taskStore.List(ctx, agentID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// handleCancelTask handles DELETE /tasks/{id} requests
func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request, taskID string) {
	ctx := r.Context()

	task, err := s.taskStore.Get(ctx, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Check if task is already in terminal state
	if task.State.IsTerminal() {
		http.Error(w, "Task already in terminal state", http.StatusConflict)
		return
	}

	// Cancel the task
	task.Cancel("Cancelled by user")
	if err := s.taskStore.Update(ctx, task); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish cancellation event
	s.taskStore.PublishEvent(ctx, protocol.TaskEvent{
		TaskID:  taskID,
		State:   protocol.TaskStateCancelled,
		Message: "Task cancelled",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleHealth handles GET /health requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

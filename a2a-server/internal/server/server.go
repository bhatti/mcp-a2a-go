package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/agentcard"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/cost"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/tasks"
)

// Server is the A2A HTTP server
type Server struct {
	taskStore     tasks.Store
	agentStore    *agentcard.Store
	costTracker   *cost.Tracker
	budgetManager *cost.BudgetManager
	agentCard     *protocol.AgentCard
}

// NewServer creates a new A2A server
func NewServer(
	taskStore tasks.Store,
	agentStore *agentcard.Store,
	costTracker *cost.Tracker,
	budgetManager *cost.BudgetManager,
	agentCard *protocol.AgentCard,
) *Server {
	return &Server{
		taskStore:     taskStore,
		agentStore:    agentStore,
		costTracker:   costTracker,
		budgetManager: budgetManager,
		agentCard:     agentCard,
	}
}

// RegisterRoutes registers all HTTP routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/agent", s.handleGetAgentCard)
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.handleCreateTask(w, r)
		case http.MethodGet:
			s.handleListTasks(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// Extract task ID from path
		path := strings.TrimPrefix(r.URL.Path, "/tasks/")
		parts := strings.Split(path, "/")
		taskID := parts[0]

		if len(parts) > 1 && parts[1] == "events" {
			// SSE endpoint
			s.handleTaskEvents(w, r, taskID)
			return
		}

		switch r.Method {
		case http.MethodGet:
			s.handleGetTask(w, r, taskID)
		case http.MethodDelete:
			s.handleCancelTask(w, r, taskID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	// Register agent card if provided
	if s.agentCard != nil {
		if err := s.agentStore.Register(context.Background(), s.agentCard); err != nil {
			log.Printf("Warning: Failed to register agent card: %v", err)
		}
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting A2A server on %s", addr)
	return server.ListenAndServe()
}

// handleTaskEvents handles SSE streaming for task events
func (s *Server) handleTaskEvents(w http.ResponseWriter, r *http.Request, taskID string) {
	ctx := r.Context()

	// Verify task exists
	_, err := s.taskStore.Get(ctx, taskID)
	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Subscribe to task events
	eventCh := s.taskStore.Subscribe(ctx, taskID)
	defer s.taskStore.Unsubscribe(ctx, taskID, eventCh)

	// Send events to client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return
			}

			// Format SSE message
			fmt.Fprintf(w, "data: {\"task_id\":\"%s\",\"state\":\"%s\",\"message\":\"%s\"}\n\n",
				event.TaskID, event.State, event.Message)
			flusher.Flush()

		case <-ctx.Done():
			return
		}
	}
}

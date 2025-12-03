package tasks

import (
	"context"
	"fmt"
	"sync"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
)

// Store defines the interface for task storage
type Store interface {
	Create(ctx context.Context, task *protocol.Task) error
	Get(ctx context.Context, id string) (*protocol.Task, error)
	Update(ctx context.Context, task *protocol.Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, agentID string, limit, offset int) ([]*protocol.Task, error)
	Subscribe(ctx context.Context, taskID string) <-chan protocol.TaskEvent
	Unsubscribe(ctx context.Context, taskID string, ch <-chan protocol.TaskEvent)
	PublishEvent(ctx context.Context, event protocol.TaskEvent)
}

// MemoryStore implements in-memory task storage
type MemoryStore struct {
	mu          sync.RWMutex
	tasks       map[string]*protocol.Task
	subscribers map[string][]chan protocol.TaskEvent
}

// NewMemoryStore creates a new in-memory task store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks:       make(map[string]*protocol.Task),
		subscribers: make(map[string][]chan protocol.TaskEvent),
	}
}

// Create creates a new task
func (s *MemoryStore) Create(ctx context.Context, task *protocol.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	s.tasks[task.ID] = task
	return nil
}

// Get retrieves a task by ID
func (s *MemoryStore) Get(ctx context.Context, id string) (*protocol.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task %s not found", id)
	}

	return task, nil
}

// Update updates an existing task
func (s *MemoryStore) Update(ctx context.Context, task *protocol.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("task %s not found", task.ID)
	}

	s.tasks[task.ID] = task
	return nil
}

// Delete deletes a task
func (s *MemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("task %s not found", id)
	}

	delete(s.tasks, id)
	return nil
}

// List lists tasks with optional filtering by agent ID
func (s *MemoryStore) List(ctx context.Context, agentID string, limit, offset int) ([]*protocol.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*protocol.Task
	for _, task := range s.tasks {
		if agentID == "" || task.AgentID == agentID {
			tasks = append(tasks, task)
		}
	}

	// Apply offset and limit
	start := offset
	if start > len(tasks) {
		return []*protocol.Task{}, nil
	}

	end := start + limit
	if end > len(tasks) {
		end = len(tasks)
	}

	return tasks[start:end], nil
}

// Subscribe subscribes to task events
func (s *MemoryStore) Subscribe(ctx context.Context, taskID string) <-chan protocol.TaskEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := make(chan protocol.TaskEvent, 10)
	s.subscribers[taskID] = append(s.subscribers[taskID], ch)
	return ch
}

// Unsubscribe unsubscribes from task events
func (s *MemoryStore) Unsubscribe(ctx context.Context, taskID string, ch <-chan protocol.TaskEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscribers := s.subscribers[taskID]
	for i, sub := range subscribers {
		if sub == ch {
			// Remove from slice
			s.subscribers[taskID] = append(subscribers[:i], subscribers[i+1:]...)
			close(sub)
			break
		}
	}

	// Clean up empty subscriber list
	if len(s.subscribers[taskID]) == 0 {
		delete(s.subscribers, taskID)
	}
}

// PublishEvent publishes an event to all subscribers
func (s *MemoryStore) PublishEvent(ctx context.Context, event protocol.TaskEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscribers := s.subscribers[event.TaskID]
	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
			// Skip if channel is full
		}
	}
}

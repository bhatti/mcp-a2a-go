package protocol

import (
	"time"

	"github.com/google/uuid"
)

// TaskState represents the state of a task
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// String returns the string representation of the task state
func (ts TaskState) String() string {
	return string(ts)
}

// IsTerminal returns true if the task state is terminal (completed, failed, or cancelled)
func (ts TaskState) IsTerminal() bool {
	return ts == TaskStateCompleted || ts == TaskStateFailed || ts == TaskStateCancelled
}

// Task represents a unit of work in the A2A protocol
type Task struct {
	ID          string                 `json:"id"`
	AgentID     string                 `json:"agent_id"`
	Capability  string                 `json:"capability"`
	Input       map[string]interface{} `json:"input,omitempty"`
	State       TaskState              `json:"state"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
}

// NewTask creates a new task with pending state
func NewTask(agentID, capability string, input map[string]interface{}) *Task {
	now := time.Now()
	return &Task{
		ID:         uuid.New().String(),
		AgentID:    agentID,
		Capability: capability,
		Input:      input,
		State:      TaskStatePending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// UpdateState updates the task state and timestamp
func (t *Task) UpdateState(state TaskState) {
	t.State = state
	t.UpdatedAt = time.Now()
}

// SetResult sets the task result and marks it as completed
func (t *Task) SetResult(result map[string]interface{}) {
	t.Result = result
	t.State = TaskStateCompleted
	t.CompletedAt = time.Now()
	t.UpdatedAt = t.CompletedAt
}

// SetError sets the task error and marks it as failed
func (t *Task) SetError(err string) {
	t.Error = err
	t.State = TaskStateFailed
	t.CompletedAt = time.Now()
	t.UpdatedAt = t.CompletedAt
}

// Cancel cancels the task
func (t *Task) Cancel(reason string) {
	t.Error = reason
	t.State = TaskStateCancelled
	t.CompletedAt = time.Now()
	t.UpdatedAt = t.CompletedAt
}

// Capability represents a capability that an agent can perform
type Capability struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"input_schema,omitempty"`
	OutputSchema map[string]interface{} `json:"output_schema,omitempty"`
}

// AgentCard represents an agent's capabilities and metadata
type AgentCard struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Capabilities []Capability `json:"capabilities"`
}

// NewAgentCard creates a new agent card
func NewAgentCard(id, name, version, description string) *AgentCard {
	return &AgentCard{
		ID:           id,
		Name:         name,
		Version:      version,
		Description:  description,
		Capabilities: make([]Capability, 0),
	}
}

// AddCapability adds a capability to the agent card
func (ac *AgentCard) AddCapability(cap Capability) {
	ac.Capabilities = append(ac.Capabilities, cap)
}

// TaskEvent represents a real-time event for task updates (SSE)
type TaskEvent struct {
	TaskID    string                 `json:"task_id"`
	State     TaskState              `json:"state"`
	Message   string                 `json:"message,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

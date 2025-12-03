package protocol

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskState_String(t *testing.T) {
	tests := []struct {
		name  string
		state TaskState
		want  string
	}{
		{"pending", TaskStatePending, "pending"},
		{"running", TaskStateRunning, "running"},
		{"completed", TaskStateCompleted, "completed"},
		{"failed", TaskStateFailed, "failed"},
		{"cancelled", TaskStateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestTaskState_IsTerminal(t *testing.T) {
	tests := []struct {
		name  string
		state TaskState
		want  bool
	}{
		{"pending is not terminal", TaskStatePending, false},
		{"running is not terminal", TaskStateRunning, false},
		{"completed is terminal", TaskStateCompleted, true},
		{"failed is terminal", TaskStateFailed, true},
		{"cancelled is terminal", TaskStateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.IsTerminal())
		})
	}
}

func TestNewTask(t *testing.T) {
	task := NewTask("agent-1", "test_capability", map[string]interface{}{
		"query": "test",
	})

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "agent-1", task.AgentID)
	assert.Equal(t, "test_capability", task.Capability)
	assert.Equal(t, TaskStatePending, task.State)
	assert.NotNil(t, task.Input)
	assert.NotZero(t, task.CreatedAt)
	assert.NotZero(t, task.UpdatedAt)
}

func TestTask_UpdateState(t *testing.T) {
	task := NewTask("agent-1", "test", nil)
	initialUpdated := task.UpdatedAt

	// Wait a bit to ensure timestamp changes
	time.Sleep(time.Millisecond)

	task.UpdateState(TaskStateRunning)

	assert.Equal(t, TaskStateRunning, task.State)
	assert.True(t, task.UpdatedAt.After(initialUpdated))
}

func TestTask_SetResult(t *testing.T) {
	task := NewTask("agent-1", "test", nil)

	result := map[string]interface{}{
		"status": "success",
		"data":   "test data",
	}

	task.SetResult(result)

	assert.Equal(t, TaskStateCompleted, task.State)
	assert.NotNil(t, task.Result)
	assert.Equal(t, "success", task.Result["status"])
	assert.NotZero(t, task.CompletedAt)
}

func TestTask_SetError(t *testing.T) {
	task := NewTask("agent-1", "test", nil)

	task.SetError("something went wrong")

	assert.Equal(t, TaskStateFailed, task.State)
	assert.Equal(t, "something went wrong", task.Error)
	assert.NotZero(t, task.CompletedAt)
}

func TestTask_Cancel(t *testing.T) {
	task := NewTask("agent-1", "test", nil)

	task.Cancel("user requested cancellation")

	assert.Equal(t, TaskStateCancelled, task.State)
	assert.Equal(t, "user requested cancellation", task.Error)
	assert.NotZero(t, task.CompletedAt)
}

func TestTask_JSON(t *testing.T) {
	task := NewTask("agent-1", "search", map[string]interface{}{
		"query": "test query",
		"limit": 10,
	})

	// Marshal to JSON
	data, err := json.Marshal(task)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded Task
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.AgentID, decoded.AgentID)
	assert.Equal(t, task.Capability, decoded.Capability)
	assert.Equal(t, task.State, decoded.State)
	assert.Equal(t, "test query", decoded.Input["query"])
	assert.Equal(t, float64(10), decoded.Input["limit"]) // JSON numbers are float64
}

func TestAgentCard_New(t *testing.T) {
	card := NewAgentCard(
		"research-agent",
		"Research Assistant",
		"1.0.0",
		"Performs research tasks with cost tracking",
	)

	assert.Equal(t, "research-agent", card.ID)
	assert.Equal(t, "Research Assistant", card.Name)
	assert.Equal(t, "1.0.0", card.Version)
	assert.Equal(t, "Performs research tasks with cost tracking", card.Description)
	assert.NotNil(t, card.Capabilities)
	assert.Empty(t, card.Capabilities)
}

func TestAgentCard_AddCapability(t *testing.T) {
	card := NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test")

	cap := Capability{
		Name:        "search_papers",
		Description: "Search academic papers",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	card.AddCapability(cap)

	assert.Len(t, card.Capabilities, 1)
	assert.Equal(t, "search_papers", card.Capabilities[0].Name)
}

func TestAgentCard_JSON(t *testing.T) {
	card := NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test agent")
	card.AddCapability(Capability{
		Name:        "test_capability",
		Description: "A test capability",
	})

	// Marshal to JSON
	data, err := json.Marshal(card)
	require.NoError(t, err)

	// Unmarshal from JSON
	var decoded AgentCard
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, card.ID, decoded.ID)
	assert.Equal(t, card.Name, decoded.Name)
	assert.Equal(t, card.Version, decoded.Version)
	assert.Len(t, decoded.Capabilities, 1)
	assert.Equal(t, "test_capability", decoded.Capabilities[0].Name)
}

func TestTaskEvent(t *testing.T) {
	event := TaskEvent{
		TaskID:    "task-123",
		State:     TaskStateRunning,
		Message:   "Processing request",
		Timestamp: time.Now(),
	}

	// Test JSON serialization
	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded TaskEvent
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.TaskID, decoded.TaskID)
	assert.Equal(t, event.State, decoded.State)
	assert.Equal(t, event.Message, decoded.Message)
}

package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	assert.NotNil(t, store)
	assert.NotNil(t, store.tasks)
	assert.NotNil(t, store.subscribers)
}

func TestMemoryStore_Create(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", map[string]interface{}{
		"query": "test",
	})

	err := store.Create(ctx, task)
	require.NoError(t, err)

	// Verify task was stored
	retrieved, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.AgentID, retrieved.AgentID)
	assert.Equal(t, task.Capability, retrieved.Capability)
}

func TestMemoryStore_Create_Duplicate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)

	err := store.Create(ctx, task)
	require.NoError(t, err)

	// Try to create again - should fail
	err = store.Create(ctx, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	store.Create(ctx, task)

	// Get existing task
	retrieved, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrieved.ID)

	// Get non-existent task
	_, err = store.Get(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	store.Create(ctx, task)

	// Update task state
	task.UpdateState(protocol.TaskStateRunning)
	err := store.Update(ctx, task)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, protocol.TaskStateRunning, retrieved.State)
}

func TestMemoryStore_Update_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)

	err := store.Update(ctx, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create multiple tasks
	task1 := protocol.NewTask("agent-1", "search", nil)
	task2 := protocol.NewTask("agent-1", "analyze", nil)
	task3 := protocol.NewTask("agent-2", "summarize", nil)

	store.Create(ctx, task1)
	store.Create(ctx, task2)
	store.Create(ctx, task3)

	// List all tasks
	tasks, err := store.List(ctx, "", 10, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// List tasks for specific agent
	tasks, err = store.List(ctx, "agent-1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// List with limit
	tasks, err = store.List(ctx, "", 2, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// List with offset
	tasks, err = store.List(ctx, "", 10, 2)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	store.Create(ctx, task)

	// Delete task
	err := store.Delete(ctx, task.ID)
	require.NoError(t, err)

	// Verify deleted
	_, err = store.Get(ctx, task.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Delete(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryStore_Subscribe(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	store.Create(ctx, task)

	// Subscribe to task events
	eventCh := store.Subscribe(ctx, task.ID)
	assert.NotNil(t, eventCh)

	// Update task in goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		task.UpdateState(protocol.TaskStateRunning)
		store.PublishEvent(ctx, protocol.TaskEvent{
			TaskID:    task.ID,
			State:     protocol.TaskStateRunning,
			Message:   "Started processing",
			Timestamp: time.Now(),
		})
	}()

	// Wait for event
	select {
	case event := <-eventCh:
		assert.Equal(t, task.ID, event.TaskID)
		assert.Equal(t, protocol.TaskStateRunning, event.State)
		assert.Equal(t, "Started processing", event.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestMemoryStore_Unsubscribe(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	task := protocol.NewTask("agent-1", "search", nil)
	store.Create(ctx, task)

	// Subscribe
	eventCh := store.Subscribe(ctx, task.ID)
	assert.NotNil(t, eventCh)

	// Unsubscribe
	store.Unsubscribe(ctx, task.ID, eventCh)

	// Verify channel is closed
	select {
	case _, ok := <-eventCh:
		assert.False(t, ok, "Channel should be closed")
	case <-time.After(10 * time.Millisecond):
		t.Fatal("Channel should be closed")
	}
}

func TestMemoryStore_PublishEvent_NoSubscribers(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Publishing to non-existent task should not panic
	store.PublishEvent(ctx, protocol.TaskEvent{
		TaskID:    "non-existent",
		State:     protocol.TaskStateRunning,
		Timestamp: time.Now(),
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Create tasks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			task := protocol.NewTask("agent-1", "search", map[string]interface{}{
				"index": idx,
			})
			err := store.Create(ctx, task)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all tasks were created
	tasks, err := store.List(ctx, "agent-1", 20, 0)
	require.NoError(t, err)
	assert.Len(t, tasks, 10)
}

package server

import (
	"context"
	"log"
	"time"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/tasks"
)

// TaskProcessor processes tasks in the background (demo implementation)
type TaskProcessor struct {
	taskStore tasks.Store
	interval  time.Duration
	stopCh    chan struct{}
}

// NewTaskProcessor creates a new task processor
func NewTaskProcessor(taskStore tasks.Store, interval time.Duration) *TaskProcessor {
	return &TaskProcessor{
		taskStore: taskStore,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the task processor
func (p *TaskProcessor) Start(ctx context.Context) {
	go p.run(ctx)
}

// Stop stops the task processor
func (p *TaskProcessor) Stop() {
	close(p.stopCh)
}

// run is the main processing loop
func (p *TaskProcessor) run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Println("Task processor started")

	for {
		select {
		case <-ticker.C:
			p.processPendingTasks(ctx)
		case <-p.stopCh:
			log.Println("Task processor stopped")
			return
		case <-ctx.Done():
			log.Println("Task processor stopped (context cancelled)")
			return
		}
	}
}

// processPendingTasks finds and processes pending tasks
func (p *TaskProcessor) processPendingTasks(ctx context.Context) {
	// Get all tasks (in production, query only pending tasks)
	allTasks, err := p.taskStore.List(ctx, "", 100, 0)
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
		return
	}

	for _, task := range allTasks {
		// Only process pending tasks
		if task.State == protocol.TaskStatePending {
			go p.processTask(ctx, task)
		}
	}
}

// processTask simulates task execution
func (p *TaskProcessor) processTask(ctx context.Context, task *protocol.Task) {
	// Transition to running
	task.UpdateState(protocol.TaskStateRunning)
	if err := p.taskStore.Update(ctx, task); err != nil {
		log.Printf("Error updating task %s to running: %v", task.ID, err)
		return
	}

	// Publish running event
	p.taskStore.PublishEvent(ctx, protocol.TaskEvent{
		TaskID:  task.ID,
		State:   protocol.TaskStateRunning,
		Message: "Task started",
	})

	log.Printf("Task %s started (simulating execution)", task.ID[:8])

	// Simulate task execution (2-5 seconds)
	executionTime := 2*time.Second + time.Duration(task.ID[0]%3)*time.Second
	time.Sleep(executionTime)

	// Simulate 90% success, 10% failure
	success := task.ID[0]%10 != 0

	if success {
		// Complete successfully
		result := map[string]interface{}{
			"status":     "success",
			"capability": task.Capability,
			"message":    "Task completed successfully",
			"timestamp":  time.Now().Format(time.RFC3339),
			"cost":       0.01, // $0.01 cost
		}

		task.SetResult(result)
		if err := p.taskStore.Update(ctx, task); err != nil {
			log.Printf("Error updating task %s to completed: %v", task.ID, err)
			return
		}

		p.taskStore.PublishEvent(ctx, protocol.TaskEvent{
			TaskID:  task.ID,
			State:   protocol.TaskStateCompleted,
			Message: "Task completed successfully",
		})

		log.Printf("Task %s completed successfully", task.ID[:8])
	} else {
		// Fail with error
		task.SetError("Simulated task failure")
		if err := p.taskStore.Update(ctx, task); err != nil {
			log.Printf("Error updating task %s to failed: %v", task.ID, err)
			return
		}

		p.taskStore.PublishEvent(ctx, protocol.TaskEvent{
			TaskID:  task.ID,
			State:   protocol.TaskStateFailed,
			Message: "Task failed",
		})

		log.Printf("Task %s failed", task.ID[:8])
	}
}

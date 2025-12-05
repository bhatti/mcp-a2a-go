package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all the metrics instruments for the A2A server
type Metrics struct {
	// Request metrics
	RequestCount     metric.Int64Counter
	RequestDuration  metric.Float64Histogram
	ActiveRequests   metric.Int64UpDownCounter

	// Task lifecycle metrics
	TaskCount          metric.Int64Counter
	TaskDuration       metric.Float64Histogram
	TaskQueueDepth     metric.Int64UpDownCounter
	ActiveTasks        metric.Int64UpDownCounter

	// Cost tracking metrics
	CostTotal          metric.Float64Counter
	TokensTotal        metric.Int64Counter
	BudgetRemaining    metric.Float64Gauge
	BudgetUtilization  metric.Float64Histogram

	// SSE metrics
	SSEConnections     metric.Int64UpDownCounter
	SSEEventsSent      metric.Int64Counter

	// Capability execution metrics
	CapabilityExecutionCount    metric.Int64Counter
	CapabilityExecutionDuration metric.Float64Histogram

	// Error metrics
	ErrorCount metric.Int64Counter
}

// NewMetrics creates and registers all metrics instruments
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}
	var err error

	// Request metrics
	m.RequestCount, err = meter.Int64Counter(
		"a2a.request.count",
		metric.WithDescription("Total number of A2A requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request count metric: %w", err)
	}

	m.RequestDuration, err = meter.Float64Histogram(
		"a2a.request.duration",
		metric.WithDescription("Duration of A2A requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration metric: %w", err)
	}

	m.ActiveRequests, err = meter.Int64UpDownCounter(
		"a2a.request.active",
		metric.WithDescription("Number of active A2A requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active requests metric: %w", err)
	}

	// Task lifecycle metrics
	m.TaskCount, err = meter.Int64Counter(
		"a2a.task.count",
		metric.WithDescription("Total number of tasks"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task count metric: %w", err)
	}

	m.TaskDuration, err = meter.Float64Histogram(
		"a2a.task.duration",
		metric.WithDescription("Duration of task execution in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task duration metric: %w", err)
	}

	m.TaskQueueDepth, err = meter.Int64UpDownCounter(
		"a2a.task.queue.depth",
		metric.WithDescription("Number of tasks in queue by priority"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task queue depth metric: %w", err)
	}

	m.ActiveTasks, err = meter.Int64UpDownCounter(
		"a2a.task.active",
		metric.WithDescription("Number of actively executing tasks"),
		metric.WithUnit("{task}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active tasks metric: %w", err)
	}

	// Cost tracking metrics
	m.CostTotal, err = meter.Float64Counter(
		"a2a.cost.total",
		metric.WithDescription("Total cost in USD"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cost total metric: %w", err)
	}

	m.TokensTotal, err = meter.Int64Counter(
		"a2a.tokens.total",
		metric.WithDescription("Total number of tokens used"),
		metric.WithUnit("{token}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokens total metric: %w", err)
	}

	m.BudgetRemaining, err = meter.Float64Gauge(
		"a2a.budget.remaining",
		metric.WithDescription("Remaining budget in USD"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget remaining metric: %w", err)
	}

	m.BudgetUtilization, err = meter.Float64Histogram(
		"a2a.budget.utilization",
		metric.WithDescription("Budget utilization percentage"),
		metric.WithUnit("%"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create budget utilization metric: %w", err)
	}

	// SSE metrics
	m.SSEConnections, err = meter.Int64UpDownCounter(
		"a2a.sse.connections",
		metric.WithDescription("Number of active SSE connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sse connections metric: %w", err)
	}

	m.SSEEventsSent, err = meter.Int64Counter(
		"a2a.sse.events.sent",
		metric.WithDescription("Total number of SSE events sent"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sse events sent metric: %w", err)
	}

	// Capability execution metrics
	m.CapabilityExecutionCount, err = meter.Int64Counter(
		"a2a.capability.execution.count",
		metric.WithDescription("Total number of capability executions"),
		metric.WithUnit("{execution}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create capability execution count metric: %w", err)
	}

	m.CapabilityExecutionDuration, err = meter.Float64Histogram(
		"a2a.capability.execution.duration",
		metric.WithDescription("Duration of capability executions in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create capability execution duration metric: %w", err)
	}

	// Error metrics
	m.ErrorCount, err = meter.Int64Counter(
		"a2a.error.count",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error count metric: %w", err)
	}

	return m, nil
}

// RecordRequest records metrics for an A2A request
func (m *Metrics) RecordRequest(ctx context.Context, path string, method string, status string, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("http.path", path),
		attribute.String("http.method", method),
		attribute.String("status", status),
	)

	m.RequestCount.Add(ctx, 1, attrs)
	m.RequestDuration.Record(ctx, durationMs, attrs)
}

// RecordTask records metrics for a task lifecycle event
func (m *Metrics) RecordTask(ctx context.Context, taskType string, status string, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("task.type", taskType),
		attribute.String("status", status),
	)

	m.TaskCount.Add(ctx, 1, attrs)
	if durationMs > 0 {
		m.TaskDuration.Record(ctx, durationMs, attrs)
	}
}

// RecordCapabilityExecution records metrics for a capability execution
func (m *Metrics) RecordCapabilityExecution(ctx context.Context, capabilityName string, status string, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("capability.name", capabilityName),
		attribute.String("status", status),
	)

	m.CapabilityExecutionCount.Add(ctx, 1, attrs)
	m.CapabilityExecutionDuration.Record(ctx, durationMs, attrs)
}

// RecordCost records cost metrics
func (m *Metrics) RecordCost(ctx context.Context, model string, costUSD float64, tokens int64) {
	costAttrs := metric.WithAttributes(
		attribute.String("model", model),
	)

	m.CostTotal.Add(ctx, costUSD, costAttrs)
	m.TokensTotal.Add(ctx, tokens, costAttrs)
}

// RecordBudgetRemaining records the remaining budget for a user
func (m *Metrics) RecordBudgetRemaining(ctx context.Context, tier string, remaining float64) {
	attrs := metric.WithAttributes(
		attribute.String("budget.tier", tier),
	)

	m.BudgetRemaining.Record(ctx, remaining, attrs)
}

// RecordSSEConnection records SSE connection metrics
func (m *Metrics) RecordSSEConnection(ctx context.Context, delta int64) {
	m.SSEConnections.Add(ctx, delta)
}

// RecordSSEEvent records SSE event sent metrics
func (m *Metrics) RecordSSEEvent(ctx context.Context, eventType string) {
	attrs := metric.WithAttributes(
		attribute.String("event.type", eventType),
	)

	m.SSEEventsSent.Add(ctx, 1, attrs)
}

// RecordError records an error occurrence
func (m *Metrics) RecordError(ctx context.Context, errorType string, operation string) {
	attrs := metric.WithAttributes(
		attribute.String("error.type", errorType),
		attribute.String("operation", operation),
	)

	m.ErrorCount.Add(ctx, 1, attrs)
}

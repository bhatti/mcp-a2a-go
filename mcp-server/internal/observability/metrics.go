package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all the metrics instruments for the MCP server
type Metrics struct {
	// Request metrics
	RequestCount     metric.Int64Counter
	RequestDuration  metric.Float64Histogram
	ActiveRequests   metric.Int64UpDownCounter

	// Tool execution metrics
	ToolExecutionCount    metric.Int64Counter
	ToolExecutionDuration metric.Float64Histogram

	// Database metrics
	DBQueryDuration       metric.Float64Histogram
	DBQueryCount          metric.Int64Counter
	DBConnectionPoolActive metric.Int64UpDownCounter
	DBConnectionPoolIdle   metric.Int64UpDownCounter

	// Search metrics
	SearchResultCount metric.Int64Histogram
	HybridSearchScore metric.Float64Histogram

	// Document metrics
	DocumentsRetrieved metric.Int64Counter

	// Error metrics
	ErrorCount metric.Int64Counter
}

// NewMetrics creates and registers all metrics instruments
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}
	var err error

	// Request metrics
	m.RequestCount, err = meter.Int64Counter(
		"mcp.request.count",
		metric.WithDescription("Total number of MCP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request count metric: %w", err)
	}

	m.RequestDuration, err = meter.Float64Histogram(
		"mcp.request.duration",
		metric.WithDescription("Duration of MCP requests in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration metric: %w", err)
	}

	m.ActiveRequests, err = meter.Int64UpDownCounter(
		"mcp.request.active",
		metric.WithDescription("Number of active MCP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active requests metric: %w", err)
	}

	// Tool execution metrics
	m.ToolExecutionCount, err = meter.Int64Counter(
		"mcp.tool.execution.count",
		metric.WithDescription("Total number of tool executions"),
		metric.WithUnit("{execution}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool execution count metric: %w", err)
	}

	m.ToolExecutionDuration, err = meter.Float64Histogram(
		"mcp.tool.execution.duration",
		metric.WithDescription("Duration of tool executions in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool execution duration metric: %w", err)
	}

	// Database metrics
	m.DBQueryDuration, err = meter.Float64Histogram(
		"mcp.db.query.duration",
		metric.WithDescription("Duration of database queries in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db query duration metric: %w", err)
	}

	m.DBQueryCount, err = meter.Int64Counter(
		"mcp.db.query.count",
		metric.WithDescription("Total number of database queries"),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db query count metric: %w", err)
	}

	m.DBConnectionPoolActive, err = meter.Int64UpDownCounter(
		"mcp.db.connection_pool.active",
		metric.WithDescription("Number of active database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db connection pool active metric: %w", err)
	}

	m.DBConnectionPoolIdle, err = meter.Int64UpDownCounter(
		"mcp.db.connection_pool.idle",
		metric.WithDescription("Number of idle database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db connection pool idle metric: %w", err)
	}

	// Search metrics
	m.SearchResultCount, err = meter.Int64Histogram(
		"mcp.search.results",
		metric.WithDescription("Number of search results returned"),
		metric.WithUnit("{result}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create search result count metric: %w", err)
	}

	m.HybridSearchScore, err = meter.Float64Histogram(
		"mcp.hybrid_search.score",
		metric.WithDescription("Hybrid search relevance scores"),
		metric.WithUnit("{score}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create hybrid search score metric: %w", err)
	}

	// Document metrics
	m.DocumentsRetrieved, err = meter.Int64Counter(
		"mcp.documents.retrieved",
		metric.WithDescription("Total number of documents retrieved"),
		metric.WithUnit("{document}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create documents retrieved metric: %w", err)
	}

	// Error metrics
	m.ErrorCount, err = meter.Int64Counter(
		"mcp.error.count",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error count metric: %w", err)
	}

	return m, nil
}

// RecordRequest records metrics for an MCP request
func (m *Metrics) RecordRequest(ctx context.Context, method string, status string, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("status", status),
	)

	m.RequestCount.Add(ctx, 1, attrs)
	m.RequestDuration.Record(ctx, durationMs, attrs)
}

// RecordToolExecution records metrics for a tool execution
func (m *Metrics) RecordToolExecution(ctx context.Context, toolName string, status string, durationMs float64) {
	attrs := metric.WithAttributes(
		attribute.String("tool.name", toolName),
		attribute.String("status", status),
	)

	m.ToolExecutionCount.Add(ctx, 1, attrs)
	m.ToolExecutionDuration.Record(ctx, durationMs, attrs)
}

// RecordDBQuery records metrics for a database query
func (m *Metrics) RecordDBQuery(ctx context.Context, queryType string, durationMs float64, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	attrs := metric.WithAttributes(
		attribute.String("query.type", queryType),
		attribute.String("status", status),
	)

	m.DBQueryCount.Add(ctx, 1, attrs)
	m.DBQueryDuration.Record(ctx, durationMs, attrs)
}

// RecordSearchResults records the number of search results
func (m *Metrics) RecordSearchResults(ctx context.Context, searchType string, count int64) {
	attrs := metric.WithAttributes(
		attribute.String("search.type", searchType),
	)

	m.SearchResultCount.Record(ctx, count, attrs)
}

// RecordError records an error occurrence
func (m *Metrics) RecordError(ctx context.Context, errorType string, operation string) {
	attrs := metric.WithAttributes(
		attribute.String("error.type", errorType),
		attribute.String("operation", operation),
	)

	m.ErrorCount.Add(ctx, 1, attrs)
}

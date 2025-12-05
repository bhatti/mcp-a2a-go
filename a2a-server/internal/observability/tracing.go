package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanFromContext returns the current span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// SetSpanAttributes sets multiple attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span and sets error status
func RecordError(ctx context.Context, err error, description string) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, description)
}

// RecordErrorWithAttributes records an error with additional attributes
func RecordErrorWithAttributes(ctx context.Context, err error, description string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, description)
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	span.SetStatus(code, description)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// Common attribute helpers for consistent naming

// TenantID creates a tenant.id attribute
func TenantID(id string) attribute.KeyValue {
	return attribute.String("tenant.id", id)
}

// UserID creates a user.id attribute (for traces only, not metrics)
func UserID(id string) attribute.KeyValue {
	return attribute.String("user.id", id)
}

// ToolName creates a tool.name attribute
func ToolName(name string) attribute.KeyValue {
	return attribute.String("tool.name", name)
}

// QueryType creates a query.type attribute
func QueryType(qtype string) attribute.KeyValue {
	return attribute.String("query.type", qtype)
}

// SearchType creates a search.type attribute
func SearchType(stype string) attribute.KeyValue {
	return attribute.String("search.type", stype)
}

// ResultCount creates a result.count attribute
func ResultCount(count int) attribute.KeyValue {
	return attribute.Int("result.count", count)
}

// ErrorType creates an error.type attribute
func ErrorType(etype string) attribute.KeyValue {
	return attribute.String("error.type", etype)
}

// HTTPMethod creates an http.method attribute
func HTTPMethod(method string) attribute.KeyValue {
	return attribute.String("http.method", method)
}

// HTTPStatusCode creates an http.status_code attribute
func HTTPStatusCode(code int) attribute.KeyValue {
	return attribute.Int("http.status_code", code)
}

// RPCMethod creates an rpc.method attribute
func RPCMethod(method string) attribute.KeyValue {
	return attribute.String("rpc.method", method)
}

// DBSystem creates a db.system attribute
func DBSystem(system string) attribute.KeyValue {
	return attribute.String("db.system", system)
}

// DBOperation creates a db.operation attribute
func DBOperation(operation string) attribute.KeyValue {
	return attribute.String("db.operation", operation)
}

// TraceID returns the trace ID from the current span
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID returns the span ID from the current span
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// WithTraceLog returns a formatted log prefix with trace information
func WithTraceLog(ctx context.Context, message string) string {
	traceID := TraceID(ctx)
	spanID := SpanID(ctx)
	if traceID != "" && spanID != "" {
		return fmt.Sprintf("[trace_id=%s span_id=%s] %s", traceID, spanID, message)
	}
	return message
}

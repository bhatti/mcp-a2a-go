package middleware

import (
	"net/http"

	"github.com/bhatti/mcp-a2a-go/mcp-server/internal/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware provides HTTP request tracing with OpenTelemetry
type TracingMiddleware struct {
	telemetry *observability.Telemetry
}

// NewTracingMiddleware creates a new tracing middleware
func NewTracingMiddleware(telemetry *observability.Telemetry) *TracingMiddleware {
	return &TracingMiddleware{
		telemetry: telemetry,
	}
}

// Handler wraps an http.Handler with tracing
func (tm *TracingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tm.telemetry == nil || tm.telemetry.Tracer == nil {
			// Tracing not enabled, pass through
			next.ServeHTTP(w, r)
			return
		}

		// Extract trace context from incoming request headers (W3C Trace Context)
		ctx := r.Context()
		propagator := otel.GetTextMapPropagator()
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

		// Start a new span for this HTTP request
		ctx, span := tm.telemetry.Tracer.Start(ctx, "http.request",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
				attribute.String("http.user_agent", r.UserAgent()),
			),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		// Create a response writer wrapper to capture status code
		wrappedWriter := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default status
		}

		// Call the next handler with the updated context
		next.ServeHTTP(wrappedWriter, r.WithContext(ctx))

		// Record span attributes based on response
		span.SetAttributes(
			attribute.Int("http.status_code", wrappedWriter.statusCode),
			attribute.Int("http.response_size", wrappedWriter.written),
		)

		// Set span status based on HTTP status code
		if wrappedWriter.statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(wrappedWriter.statusCode))
		} else {
			span.SetStatus(codes.Ok, "Request completed successfully")
		}
	})
}

// statusRecorder wraps http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    int
}

// WriteHeader captures the status code
func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (sr *statusRecorder) Write(b []byte) (int, error) {
	n, err := sr.ResponseWriter.Write(b)
	sr.written += n
	return n, err
}

package observability

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds the configuration for telemetry setup
type Config struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	OTLPEndpoint    string
	SamplingRate    float64 // 0.0 to 1.0, default 1.0 (100%)
	EnableTracing   bool
	EnableMetrics   bool
}

// Telemetry holds the OpenTelemetry providers and helpers
type Telemetry struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	Tracer         trace.Tracer
	Metrics        *Metrics
	config         Config
}

// NewTelemetry initializes OpenTelemetry with tracing and metrics
func NewTelemetry(ctx context.Context, cfg Config) (*Telemetry, error) {
	// Set defaults
	if cfg.ServiceName == "" {
		cfg.ServiceName = "mcp-server"
	}
	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "1.0.0"
	}
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	if cfg.SamplingRate == 0 {
		cfg.SamplingRate = 1.0 // 100% by default
	}
	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = "http://jaeger:4318" // HTTP endpoint for OTLP
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	t := &Telemetry{
		config: cfg,
	}

	// Initialize tracing
	if cfg.EnableTracing {
		if err := t.initTracing(ctx, res); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
		log.Printf("OpenTelemetry tracing initialized (endpoint: %s, sampling: %.0f%%)",
			cfg.OTLPEndpoint, cfg.SamplingRate*100)
	}

	// Initialize metrics
	if cfg.EnableMetrics {
		if err := t.initMetrics(res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
		log.Println("OpenTelemetry metrics initialized (Prometheus exporter)")
	}

	return t, nil
}

// initTracing sets up the trace provider with OTLP exporter
func (t *Telemetry) initTracing(ctx context.Context, res *resource.Resource) error {
	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(t.config.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // Use insecure for local development
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create sampler based on configuration
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(t.config.SamplingRate),
	)

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Set global propagator for W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.TracerProvider = tp
	t.Tracer = tp.Tracer(t.config.ServiceName)

	return nil
}

// initMetrics sets up the meter provider with Prometheus exporter
func (t *Telemetry) initMetrics(res *resource.Resource) error {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	t.MeterProvider = mp

	// Initialize metrics
	meter := mp.Meter(t.config.ServiceName)
	metrics, err := NewMetrics(meter)
	if err != nil {
		return fmt.Errorf("failed to create metrics: %w", err)
	}
	t.Metrics = metrics

	return nil
}

// Shutdown gracefully shuts down the telemetry providers
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var err error

	if t.TracerProvider != nil {
		if shutdownErr := t.TracerProvider.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr)
		}
	}

	if t.MeterProvider != nil {
		if shutdownErr := t.MeterProvider.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; failed to shutdown meter provider: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr)
			}
		}
	}

	return err
}

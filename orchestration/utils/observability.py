"""
OpenTelemetry observability utilities for Python workflows.

This module provides OpenTelemetry tracing setup and helpers that complement
Langfuse for LLM-specific observability. Together they provide:
- OpenTelemetry: Service-to-service tracing, infrastructure metrics
- Langfuse: LLM call tracking, prompt debugging, token costs
"""

import logging
import os
from typing import Optional

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor

logger = logging.getLogger(__name__)


def setup_otel_tracing(
    service_name: str,
    service_version: str = "1.0.0",
    environment: str = "development",
    otlp_endpoint: Optional[str] = None,
    enable_auto_instrumentation: bool = True,
) -> trace.Tracer:
    """
    Setup OpenTelemetry tracing with OTLP exporter.

    Args:
        service_name: Name of the service (e.g., "rag-workflow", "research-workflow")
        service_version: Version of the service
        environment: Environment name (development, staging, production)
        otlp_endpoint: OTLP endpoint URL (default: from env or http://jaeger:4318/v1/traces)
        enable_auto_instrumentation: Whether to auto-instrument HTTP clients

    Returns:
        Tracer instance for manual instrumentation

    Example:
        >>> tracer = setup_otel_tracing("rag-workflow")
        >>> with tracer.start_as_current_span("process_documents") as span:
        ...     span.set_attribute("doc_count", 5)
        ...     # Process documents
    """
    # Get OTLP endpoint from env or use default
    if otlp_endpoint is None:
        otlp_endpoint = os.getenv(
            "OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318/v1/traces"
        )

    # Create resource with service information
    resource = Resource.create(
        {
            "service.name": service_name,
            "service.version": service_version,
            "deployment.environment": environment,
        }
    )

    # Create OTLP HTTP exporter
    otlp_exporter = OTLPSpanExporter(
        endpoint=otlp_endpoint,
        # Timeout in seconds
        timeout=30,
    )

    # Create tracer provider
    provider = TracerProvider(resource=resource)

    # Add batch span processor for efficient export
    processor = BatchSpanProcessor(
        otlp_exporter,
        max_queue_size=2048,
        schedule_delay_millis=5000,
        max_export_batch_size=512,
    )
    provider.add_span_processor(processor)

    # Set as global tracer provider
    trace.set_tracer_provider(provider)

    # Auto-instrument HTTP clients if enabled
    if enable_auto_instrumentation:
        try:
            RequestsInstrumentor().instrument()
            logger.info("Auto-instrumented requests library")
        except Exception as e:
            logger.warning(f"Failed to instrument requests: {e}")

        try:
            HTTPXClientInstrumentor().instrument()
            logger.info("Auto-instrumented httpx library")
        except Exception as e:
            logger.warning(f"Failed to instrument httpx: {e}")

    logger.info(
        f"OpenTelemetry tracing initialized: service={service_name}, "
        f"endpoint={otlp_endpoint}, environment={environment}"
    )

    # Return tracer for manual instrumentation
    return trace.get_tracer(service_name, service_version)


def get_tracer(service_name: str, service_version: str = "1.0.0") -> trace.Tracer:
    """
    Get a tracer instance for manual instrumentation.

    This assumes setup_otel_tracing() has already been called.

    Args:
        service_name: Name of the service
        service_version: Version of the service

    Returns:
        Tracer instance
    """
    return trace.get_tracer(service_name, service_version)


def add_span_attributes(span: trace.Span, **attributes):
    """
    Add multiple attributes to a span.

    Args:
        span: Span to add attributes to
        **attributes: Key-value pairs to add as attributes

    Example:
        >>> with tracer.start_as_current_span("search") as span:
        ...     add_span_attributes(
        ...         span,
        ...         query="machine learning",
        ...         top_k=5,
        ...         user_id="demo-user"
        ...     )
    """
    for key, value in attributes.items():
        span.set_attribute(key, value)


def add_span_event(span: trace.Span, name: str, **attributes):
    """
    Add an event to a span.

    Args:
        span: Span to add event to
        name: Event name
        **attributes: Event attributes

    Example:
        >>> with tracer.start_as_current_span("llm_call") as span:
        ...     add_span_event(span, "token_limit_warning", tokens=1500, limit=2000)
    """
    span.add_event(name, attributes=attributes)


def record_exception(span: trace.Span, exception: Exception, **attributes):
    """
    Record an exception on a span.

    Args:
        span: Span to record exception on
        exception: Exception to record
        **attributes: Additional attributes

    Example:
        >>> try:
        ...     # Some operation
        ...     pass
        ... except Exception as e:
        ...     record_exception(span, e, operation="search", query="test")
        ...     raise
    """
    span.record_exception(exception, attributes=attributes)
    span.set_status(trace.Status(trace.StatusCode.ERROR, str(exception)))


class OTelContextManager:
    """
    Context manager for creating spans with automatic error handling.

    Example:
        >>> tracer = get_tracer("my-service")
        >>> with OTelContextManager(tracer, "operation_name", attr1="value1") as span:
        ...     # Do work
        ...     span.set_attribute("result_count", 10)
    """

    def __init__(self, tracer: trace.Tracer, span_name: str, **attributes):
        self.tracer = tracer
        self.span_name = span_name
        self.attributes = attributes
        self.span = None

    def __enter__(self) -> trace.Span:
        self.span = self.tracer.start_span(self.span_name)
        if self.attributes:
            add_span_attributes(self.span, **self.attributes)
        return self.span

    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type is not None and self.span:
            record_exception(self.span, exc_val)
        if self.span:
            self.span.end()
        return False


# Convenience functions for common span attributes

def set_user_attributes(span: trace.Span, user_id: str, tenant_id: Optional[str] = None):
    """Set user-related attributes on a span (for traces only, not metrics)."""
    span.set_attribute("user.id", user_id)
    if tenant_id:
        span.set_attribute("tenant.id", tenant_id)


def set_llm_attributes(
    span: trace.Span,
    model: str,
    temperature: Optional[float] = None,
    max_tokens: Optional[int] = None,
):
    """Set LLM-related attributes on a span."""
    span.set_attribute("llm.model", model)
    if temperature is not None:
        span.set_attribute("llm.temperature", temperature)
    if max_tokens is not None:
        span.set_attribute("llm.max_tokens", max_tokens)


def set_search_attributes(
    span: trace.Span,
    query: str,
    search_type: str,
    top_k: Optional[int] = None,
):
    """Set search-related attributes on a span."""
    span.set_attribute("search.query", query)
    span.set_attribute("search.type", search_type)
    if top_k is not None:
        span.set_attribute("search.top_k", top_k)


def set_task_attributes(
    span: trace.Span,
    task_id: str,
    task_type: str,
    priority: Optional[str] = None,
):
    """Set task-related attributes on a span."""
    span.set_attribute("task.id", task_id)
    span.set_attribute("task.type", task_type)
    if priority:
        span.set_attribute("task.priority", priority)

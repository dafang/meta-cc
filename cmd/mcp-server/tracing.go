package main

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/yaleh/meta-cc/internal/mcp/observability"
)

// InitTracing initializes OpenTelemetry distributed tracing
func InitTracing() (func(), error) {
	return observability.InitTracing()
}

// GetTracer returns the global tracer instance
func GetTracer() trace.Tracer {
	return observability.GetTracer()
}

// GetTraceID extracts the trace ID from a context
func GetTraceID(ctx context.Context) string {
	return observability.GetTraceID(ctx)
}

// GetSpanID extracts the span ID from a context
func GetSpanID(ctx context.Context) string {
	return observability.GetSpanID(ctx)
}

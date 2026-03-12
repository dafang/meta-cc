package observability

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/yaleh/meta-cc/internal/config"
)

// loggerContextKey is the context key for storing the logger
type loggerContextKey struct{}

var tracer trace.Tracer

// InitLogger initializes the global slog logger with centralized configuration
func InitLogger(cfg *config.Config) {
	// Use log level from centralized config
	// Config handles LOG_LEVEL fallback for backward compatibility
	logLevel := cfg.Log.Level

	// Create JSON handler for structured logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true, // Include file:line for debugging
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// NewRequestLogger creates a request-scoped logger with request_id and tool_name
func NewRequestLogger(toolName string) (*slog.Logger, string) {
	requestID := uuid.New().String()
	logger := slog.Default().With(
		"request_id", requestID,
		"tool_name", toolName,
	)
	return logger, requestID
}

// WithLogger attaches a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext retrieves the logger from the context
// If no logger is found, returns the default logger
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

var errorClassificationRules = []struct {
	errorType string
	patterns  []string
}{
	{errorType: "parse_error", patterns: []string{"parse", "unmarshal", "invalid JSON", "decode"}},
	{errorType: "validation_error", patterns: []string{"validation", "invalid", "missing", "required"}},
	{errorType: "io_error", patterns: []string{"no such file", "permission denied", "cannot open", "read", "write"}},
	{errorType: "execution_error", patterns: []string{"execution", "command", "process"}},
	{errorType: "network_error", patterns: []string{"network", "connection", "timeout", "http"}},
}

// ClassifyError classifies an error into a category for logging
func ClassifyError(err error) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()
	for _, rule := range errorClassificationRules {
		if containsAny(errMsg, rule.patterns) {
			return rule.errorType
		}
	}

	return "general_error"
}

func containsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

// InitTracing initializes OpenTelemetry distributed tracing
func InitTracing() (func(), error) {
	// Create stdout exporter for testing/development
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stderr), // Write traces to stderr to avoid mixing with JSON-RPC
	)
	if err != nil {
		return nil, err
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("meta-cc-mcp"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Configure sampler (AlwaysOn for development, configurable for production)
	sampler := sdktrace.AlwaysSample()
	if samplingRatio := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); samplingRatio != "" {
		// TraceIDRatioBased sampler can be configured via environment variable
		// Example: OTEL_TRACES_SAMPLER_ARG=0.1 for 10% sampling
		slog.Debug("trace sampling configured",
			"sampling_ratio", samplingRatio,
		)
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Register as global trace provider
	otel.SetTracerProvider(tp)

	// Get tracer for this service
	tracer = tp.Tracer("meta-cc-mcp")

	slog.Info("distributed tracing initialized",
		"exporter", "stdout",
		"service_name", "meta-cc-mcp",
	)

	// Return cleanup function
	cleanup := func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("failed to shutdown trace provider",
				"error", err.Error(),
			)
		}
	}

	return cleanup, nil
}

// GetTracer returns the global tracer instance
func GetTracer() trace.Tracer {
	return tracer
}

// GetTraceID extracts the trace ID from a context
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from a context
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

package main

import (
	"context"
	"log/slog"

	"github.com/yaleh/meta-cc/internal/config"
	"github.com/yaleh/meta-cc/internal/mcp/observability"
)

// InitLogger initializes the global slog logger with centralized configuration
func InitLogger(cfg *config.Config) {
	observability.InitLogger(cfg)
}

// NewRequestLogger creates a request-scoped logger with request_id and tool_name
func NewRequestLogger(toolName string) (*slog.Logger, string) {
	return observability.NewRequestLogger(toolName)
}

// WithLogger attaches a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return observability.WithLogger(ctx, logger)
}

// LoggerFromContext retrieves the logger from the context
func LoggerFromContext(ctx context.Context) *slog.Logger {
	return observability.LoggerFromContext(ctx)
}

// classifyError classifies an error into a category for logging
func classifyError(err error) string {
	return observability.ClassifyError(err)
}

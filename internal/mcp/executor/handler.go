package executor

import (
	"context"

	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
)

// SpecialToolHandler is a function that handles a single special MCP tool invocation.
// It returns the string output and any error; metrics recording is handled by the caller.
type SpecialToolHandler func(ctx context.Context, e *ToolExecutor, params map[string]interface{}) (string, error)

// QueryHandlerFunc handles a convenience query tool and returns a QueryResult.
// Handlers self-register via init() in handlers.go using registerQueryHandler.
type QueryHandlerFunc func(e *ToolExecutor, scope string, args map[string]interface{}) (mcquery.QueryResult, error)

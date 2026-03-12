package executor

import "context"

// SpecialToolHandler is a function that handles a single special MCP tool invocation.
// It returns the string output and any error; metrics recording is handled by the caller.
type SpecialToolHandler func(ctx context.Context, e *ToolExecutor, params map[string]interface{}) (string, error)

package query

import (
	"github.com/yaleh/meta-cc/internal/parser"
	"github.com/yaleh/meta-cc/internal/query/resources"
)

// Sentinel errors for consistent error handling by callers.
var (
	ErrSessionLoad   = resources.ErrSessionLoad
	ErrFilterInvalid = resources.ErrFilterInvalid
)

// RunToolsQuery loads tool calls using the provided SessionLoader, applies filters, sorting,
// and pagination according to the provided options, and returns the resulting slice.
func RunToolsQuery(loader SessionLoader, opts ToolsQueryOptions) ([]parser.ToolCall, error) {
	return resources.RunToolsQuery(loader, opts)
}

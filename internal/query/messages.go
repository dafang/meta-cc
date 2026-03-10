package query

import (
	"github.com/yaleh/meta-cc/internal/query/resources"
)

// ErrInvalidPattern indicates the provided regex pattern could not be compiled.
var ErrInvalidPattern = resources.ErrInvalidPattern

// RunUserMessagesQuery extracts user messages from the provided SessionLoader, applies
// pattern filtering, context expansion, sorting, and pagination according to options.
func RunUserMessagesQuery(loader SessionLoader, opts UserMessagesQueryOptions) ([]UserMessage, error) {
	return resources.RunUserMessagesQuery(loader, opts)
}

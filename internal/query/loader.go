package query

import "github.com/yaleh/meta-cc/internal/parser"

// SessionLoader abstracts session data loading so query functions
// don't depend on a concrete pipeline implementation.
type SessionLoader interface {
	Entries() []parser.SessionEntry
	ExtractToolCalls() []parser.ToolCall
	BuildTurnIndex() map[string]int
}

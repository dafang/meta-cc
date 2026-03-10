package types

// SessionLoader abstracts session data loading so query functions
// don't depend on a concrete pipeline implementation.
type SessionLoader interface {
	Entries() []SessionEntry
	ExtractToolCalls() []ToolCall
	BuildTurnIndex() map[string]int
}

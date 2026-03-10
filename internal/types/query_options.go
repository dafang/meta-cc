package types

// ToolsQueryOptions configures RunToolsQuery behaviour.
type ToolsQueryOptions struct {
	Limit      int    // Max records to return (0 = no limit)
	Offset     int    // Number of records to skip before returning results
	SortBy     string // Sort field (timestamp, tool, status, uuid)
	Reverse    bool   // Reverse sort order
	Status     string // Filter by status (success|error)
	Tool       string // Filter by tool name
	Where      string // Key=value filter expression
	Expression string // Advanced expression filter (SQL-like)
}

// UserMessagesQueryOptions configures RunUserMessagesQuery behaviour.
type UserMessagesQueryOptions struct {
	Pattern string // Regex pattern to match message content
	Context int    // Number of turns of context before/after
	Limit   int    // Max messages to return (0 = no limit)
	Offset  int    // Number of results to skip before returning
	SortBy  string // Sort field (turn_sequence, timestamp, uuid)
	Reverse bool   // Reverse sort order
}

// ContextEntry captures contextual turns surrounding a user message.
type ContextEntry struct {
	Turn      int      `json:"turn"`
	Role      string   `json:"role"`
	Summary   string   `json:"summary"`
	ToolCalls []string `json:"tool_calls,omitempty"`
}

// UserMessage represents a user message enriched with metadata and optional context.
type UserMessage struct {
	TurnSequence  int            `json:"turn_sequence"`
	UUID          string         `json:"uuid"`
	Timestamp     string         `json:"timestamp"`
	Content       string         `json:"content"`
	ContextBefore []ContextEntry `json:"context_before,omitempty"`
	ContextAfter  []ContextEntry `json:"context_after,omitempty"`
}

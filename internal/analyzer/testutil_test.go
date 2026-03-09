package analyzer

import "github.com/yaleh/meta-cc/internal/parser"

// makeToolCalls creates ToolCall fixtures directly
func makeToolCalls(toolName string, status string, errMsg string) []parser.ToolCall {
	return []parser.ToolCall{
		{
			UUID:      "test-uuid-1",
			ToolName:  toolName,
			Status:    status,
			Error:     errMsg,
			Timestamp: "2025-10-02T10:00:00.000Z",
		},
	}
}

// makeEntry creates a single SessionEntry fixture for testing
func makeEntry(uuid string, timestamp string) parser.SessionEntry {
	return parser.SessionEntry{
		Type:      "assistant",
		UUID:      uuid,
		Timestamp: timestamp,
	}
}

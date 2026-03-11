package main

import filterspkg "github.com/yaleh/meta-cc/internal/mcp/filters"

const (
	// DefaultMaxMessageLength is the default for max_message_length parameter.
	// Set to 0 to disable truncation (use hybrid output mode for large results).
	DefaultMaxMessageLength = filterspkg.DefaultMaxMessageLength

	DefaultPreviewLength = filterspkg.DefaultPreviewLength
)

// TruncateMessageContent truncates the 'content' field in user messages
// to prevent context overflow from large session summaries.
//
// Use hybrid output mode instead of truncation for complete data preservation.
//
// Parameters:
//   - messages: Array of message objects (maps)
//   - maxLen: Maximum length for content field (0 or negative = no truncation)
//
// Returns:
//   - New array with truncated messages (originals unchanged)
//
// Truncated messages include:
//   - content_truncated: true
//   - original_length: original content length
func TruncateMessageContent(messages []interface{}, maxLen int) []interface{} {
	return filterspkg.TruncateMessageContent(messages, maxLen)
}

// ApplyContentSummary returns only message metadata (no full content).
// Useful for pattern matching without needing full message text.
//
// Use hybrid output mode instead for complete data preservation.
//
// Parameters:
//   - messages: Array of message objects (maps)
//   - previewLength: Max runes for content_preview (0 or negative = DefaultPreviewLength)
//
// Returns:
//   - New array with summary objects containing:
//   - turn_sequence: message turn number
//   - timestamp: message timestamp
//   - content_preview: first previewLength characters of content
//
// All other fields are removed to reduce output size.
func ApplyContentSummary(messages []interface{}, previewLength int) []interface{} {
	return filterspkg.ApplyContentSummary(messages, previewLength)
}

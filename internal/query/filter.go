package query

import (
	"regexp"
	"strings"

	"github.com/yaleh/meta-cc/internal/types"
)

// ApplyFilter applies filter conditions to resources
// Returns filtered slice of the same type as input
func ApplyFilter(resources interface{}, filter FilterSpec) interface{} {
	// If filter is empty, return all resources
	if filter.IsEmpty() {
		return resources
	}

	// Handle different resource types
	switch r := resources.(type) {
	case []types.SessionEntry:
		return filterEntries(r, filter)
	case []MessageView:
		return filterMessages(r, filter)
	case []types.ToolCall:
		return filterTools(r, filter)
	default:
		// Unknown type, return as is
		return resources
	}
}

// filterEntries filters SessionEntry slice
func filterEntries(entries []types.SessionEntry, filter FilterSpec) []types.SessionEntry {
	var result []types.SessionEntry
	for _, entry := range entries {
		if matchesFilter(entry, filter) {
			result = append(result, entry)
		}
	}
	return result
}

// filterMessages filters MessageView slice
func filterMessages(messages []MessageView, filter FilterSpec) []MessageView {
	var result []MessageView
	for _, msg := range messages {
		if matchesFilter(msg, filter) {
			result = append(result, msg)
		}
	}
	return result
}

// filterTools filters ToolCall slice
func filterTools(tools []types.ToolCall, filter FilterSpec) []types.ToolCall {
	var result []types.ToolCall
	for _, tool := range tools {
		if matchesFilter(tool, filter) {
			result = append(result, tool)
		}
	}
	return result
}

// matchesFilter checks if a resource matches filter conditions
// Supports SessionEntry, MessageView, and ToolCall
func matchesFilter(resource interface{}, filter FilterSpec) bool {
	// Entry-level filters
	if filter.Type != "" {
		if entry, ok := resource.(types.SessionEntry); ok {
			if entry.Type != filter.Type {
				return false
			}
		}
	}

	if filter.UUID != "" {
		uuid := extractUUID(resource)
		if uuid != filter.UUID {
			return false
		}
	}

	if filter.SessionID != "" {
		sessionID := extractSessionID(resource)
		if sessionID != filter.SessionID {
			return false
		}
	}

	if filter.ParentUUID != "" {
		parentUUID := extractParentUUID(resource)
		if parentUUID != filter.ParentUUID {
			return false
		}
	}

	if filter.GitBranch != "" {
		gitBranch := extractGitBranch(resource)
		if gitBranch != filter.GitBranch {
			return false
		}
	}

	// Time range filter
	if filter.TimeRange != nil {
		timestamp := extractTimestamp(resource)
		if !matchesTimeRange(timestamp, filter.TimeRange) {
			return false
		}
	}

	// Message-level filters
	if filter.Role != "" {
		if msg, ok := resource.(MessageView); ok {
			if msg.Role != filter.Role {
				return false
			}
		}
		// Also check entries with messages
		if entry, ok := resource.(types.SessionEntry); ok {
			if entry.Message != nil && entry.Message.Role != filter.Role {
				return false
			}
		}
	}

	if filter.ContentMatch != "" {
		content := extractContent(resource)
		if !matchesPattern(content, filter.ContentMatch) {
			return false
		}
	}

	// Tool-level filters
	if filter.ToolName != "" {
		if tool, ok := resource.(types.ToolCall); ok {
			if !matchesPattern(tool.ToolName, filter.ToolName) {
				return false
			}
		}
	}

	if filter.ToolStatus != "" {
		if tool, ok := resource.(types.ToolCall); ok {
			if tool.Status != filter.ToolStatus {
				return false
			}
		}
	}

	if filter.HasError != nil {
		if tool, ok := resource.(types.ToolCall); ok {
			hasError := tool.Error != ""
			if hasError != *filter.HasError {
				return false
			}
		}
	}

	return true
}

// Field extraction functions

func extractUUID(resource interface{}) string {
	switch r := resource.(type) {
	case types.SessionEntry:
		return r.UUID
	case MessageView:
		return r.UUID
	case types.ToolCall:
		return r.UUID
	}
	return ""
}

func extractSessionID(resource interface{}) string {
	switch r := resource.(type) {
	case types.SessionEntry:
		return r.SessionID
	case MessageView:
		return r.SessionID
	}
	return ""
}

func extractParentUUID(resource interface{}) string {
	switch r := resource.(type) {
	case types.SessionEntry:
		return r.ParentUUID
	case MessageView:
		return r.ParentUUID
	}
	return ""
}

func extractGitBranch(resource interface{}) string {
	switch r := resource.(type) {
	case types.SessionEntry:
		return r.GitBranch
	case MessageView:
		return r.GitBranch
	}
	return ""
}

func extractTimestamp(resource interface{}) string {
	switch r := resource.(type) {
	case types.SessionEntry:
		return r.Timestamp
	case MessageView:
		return r.Timestamp
	case types.ToolCall:
		return r.Timestamp
	}
	return ""
}

func extractContent(resource interface{}) string {
	switch r := resource.(type) {
	case MessageView:
		return r.Content
	case types.SessionEntry:
		if r.Message != nil {
			var content strings.Builder
			for _, block := range r.Message.Content {
				if block.Type == "text" {
					content.WriteString(block.Text)
				}
			}
			return content.String()
		}
	}
	return ""
}

// matchesTimeRange checks if timestamp is within time range
func matchesTimeRange(timestamp string, timeRange *TimeRange) bool {
	if timeRange.Start != "" && timestamp < timeRange.Start {
		return false
	}
	if timeRange.End != "" && timestamp > timeRange.End {
		return false
	}
	return true
}

// matchesPattern checks if value matches pattern (supports regex or simple string match)
func matchesPattern(value, pattern string) bool {
	// Try regex match first
	if re, err := regexp.Compile(pattern); err == nil {
		return re.MatchString(value)
	}
	// Fallback to simple string match
	return value == pattern
}

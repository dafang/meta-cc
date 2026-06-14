package pipeline

import (
	"regexp"
	"strings"

	"github.com/yaleh/meta-cc/internal/types"
)

// ApplyFilter applies filter conditions to resources.
func ApplyFilter(resources interface{}, filter FilterSpec) interface{} {
	if filter.IsEmpty() {
		return resources
	}
	switch r := resources.(type) {
	case []types.SessionEntry:
		return filterEntries(r, filter)
	case []MessageView:
		return filterMessages(r, filter)
	case []types.ToolCall:
		return filterTools(r, filter)
	default:
		return resources
	}
}

func filterEntries(entries []types.SessionEntry, filter FilterSpec) []types.SessionEntry {
	var result []types.SessionEntry
	for _, entry := range entries {
		if matchesFilter(entry, filter) {
			result = append(result, entry)
		}
	}
	return result
}

func filterMessages(messages []MessageView, filter FilterSpec) []MessageView {
	var result []MessageView
	for _, msg := range messages {
		if matchesFilter(msg, filter) {
			result = append(result, msg)
		}
	}
	return result
}

func filterTools(tools []types.ToolCall, filter FilterSpec) []types.ToolCall {
	var result []types.ToolCall
	for _, tool := range tools {
		if matchesFilter(tool, filter) {
			result = append(result, tool)
		}
	}
	return result
}

func matchesFilter(resource interface{}, filter FilterSpec) bool {
	if filter.Type != "" {
		if entry, ok := resource.(types.SessionEntry); ok {
			if entry.Type != filter.Type {
				return false
			}
		}
	}
	if filter.UUID != "" {
		if extractUUID(resource) != filter.UUID {
			return false
		}
	}
	if filter.SessionID != "" {
		if extractSessionID(resource) != filter.SessionID {
			return false
		}
	}
	if filter.ParentUUID != "" {
		if extractParentUUID(resource) != filter.ParentUUID {
			return false
		}
	}
	if filter.GitBranch != "" {
		if extractGitBranch(resource) != filter.GitBranch {
			return false
		}
	}
	if filter.TimeRange != nil {
		if !matchesTimeRange(extractTimestamp(resource), filter.TimeRange) {
			return false
		}
	}
	if filter.Role != "" {
		if msg, ok := resource.(MessageView); ok {
			if msg.Role != filter.Role {
				return false
			}
		}
		if entry, ok := resource.(types.SessionEntry); ok {
			if entry.Message != nil && entry.Message.Role != filter.Role {
				return false
			}
		}
	}
	if filter.ContentMatch != "" {
		if !matchesPattern(extractContent(resource), filter.ContentMatch) {
			return false
		}
	}
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
			if (tool.Error != "") != *filter.HasError {
				return false
			}
		}
	}
	return true
}

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

func matchesTimeRange(timestamp string, timeRange *TimeRange) bool {
	if timeRange.Start != "" && timestamp < timeRange.Start {
		return false
	}
	if timeRange.End != "" && timestamp > timeRange.End {
		return false
	}
	return true
}

func matchesPattern(value, pattern string) bool {
	if re, err := regexp.Compile(pattern); err == nil {
		return re.MatchString(value)
	}
	return value == pattern
}

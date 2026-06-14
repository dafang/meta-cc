package pipeline

import "github.com/yaleh/meta-cc/internal/types"

// ApplyAggregate applies aggregation operations to resources.
func ApplyAggregate(resources interface{}, aggregate AggregateSpec) interface{} {
	if aggregate.IsEmpty() {
		return resources
	}

	var items []interface{}
	switch r := resources.(type) {
	case []types.SessionEntry:
		for _, item := range r {
			items = append(items, item)
		}
	case []MessageView:
		for _, item := range r {
			items = append(items, item)
		}
	case []types.ToolCall:
		for _, item := range r {
			items = append(items, item)
		}
	default:
		return resources
	}

	switch aggregate.Function {
	case "count":
		return aggregateCount(items, aggregate.Field)
	case "group":
		return aggregateGroup(items, aggregate.Field)
	default:
		return resources
	}
}

func aggregateCount(items []interface{}, field string) []map[string]interface{} {
	if field == "" {
		return []map[string]interface{}{{"count": len(items)}}
	}
	counts := make(map[string]int)
	for _, item := range items {
		counts[extractFieldValue(item, field)]++
	}
	var result []map[string]interface{}
	for value, count := range counts {
		result = append(result, map[string]interface{}{field: value, "count": count})
	}
	return result
}

func aggregateGroup(items []interface{}, field string) []map[string]interface{} {
	groups := make(map[string][]interface{})
	for _, item := range items {
		value := extractFieldValue(item, field)
		groups[value] = append(groups[value], item)
	}
	var result []map[string]interface{}
	for value, groupItems := range groups {
		result = append(result, map[string]interface{}{
			field:   value,
			"count": len(groupItems),
			"items": groupItems,
		})
	}
	return result
}

func extractFieldValue(resource interface{}, field string) string {
	switch field {
	case "tool_name":
		if tool, ok := resource.(types.ToolCall); ok {
			return tool.ToolName
		}
	case "status":
		if tool, ok := resource.(types.ToolCall); ok {
			return tool.Status
		}
	case "role":
		if msg, ok := resource.(MessageView); ok {
			return msg.Role
		}
	case "type":
		if entry, ok := resource.(types.SessionEntry); ok {
			return entry.Type
		}
	case "session_id":
		if entry, ok := resource.(types.SessionEntry); ok {
			return entry.SessionID
		}
		if msg, ok := resource.(MessageView); ok {
			return msg.SessionID
		}
	case "git_branch":
		if entry, ok := resource.(types.SessionEntry); ok {
			return entry.GitBranch
		}
		if msg, ok := resource.(MessageView); ok {
			return msg.GitBranch
		}
	}
	return ""
}

package resources

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yaleh/meta-cc/internal/filter"
	"github.com/yaleh/meta-cc/internal/types"
)

// RunToolsQuery loads tool calls using the provided SessionLoader, applies filters, sorting,
// and pagination according to the provided options, and returns the resulting slice.
func RunToolsQuery(loader types.SessionLoader, opts types.ToolsQueryOptions) ([]types.ToolCall, error) {
	calls := loader.ExtractToolCalls()

	filtered, err := applyToolFilters(calls, opts)
	if err != nil {
		return nil, err
	}

	sortToolCalls(filtered, opts.SortBy, opts.Reverse)

	return applyToolPagination(filtered, opts.Limit, opts.Offset), nil
}

func applyToolFilters(toolCalls []types.ToolCall, opts types.ToolsQueryOptions) ([]types.ToolCall, error) {
	filtered := toolCalls
	var err error

	if opts.Expression != "" {
		filtered, err = applyExpressionFilter(filtered, opts.Expression)
		if err != nil {
			return nil, err
		}
	}

	if opts.Where != "" {
		if isAdvancedWhere(opts.Where) {
			normalized := normalizeAdvancedWhere(opts.Where)
			filtered, err = applyExpressionFilter(filtered, normalized)
		} else {
			filtered, err = applySimpleWhere(filtered, opts.Where)
		}
		if err != nil {
			return nil, err
		}
	}

	return applyFlagFilters(filtered, opts.Status, opts.Tool), nil
}

func applyExpressionFilter(toolCalls []types.ToolCall, expression string) ([]types.ToolCall, error) {
	if expression == "" {
		return toolCalls, nil
	}

	expr, err := filter.ParseExpression(expression)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFilterInvalid, err)
	}

	var filtered []types.ToolCall
	for _, tc := range toolCalls {
		record := map[string]interface{}{
			"tool":   tc.ToolName,
			"status": tc.Status,
			"uuid":   tc.UUID,
			"error":  tc.Error,
		}

		match, evalErr := expr.Evaluate(record)
		if evalErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrFilterInvalid, evalErr)
		}

		if match {
			filtered = append(filtered, tc)
		}
	}

	return filtered, nil
}

func applySimpleWhere(toolCalls []types.ToolCall, where string) ([]types.ToolCall, error) {
	result, err := filter.ApplyWhere(toolCalls, where, "tool_calls")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFilterInvalid, err)
	}
	return result.([]types.ToolCall), nil
}

func applyFlagFilters(toolCalls []types.ToolCall, status, tool string) []types.ToolCall {
	var result []types.ToolCall

	for _, tc := range toolCalls {
		if !matchesStatus(tc, status) {
			continue
		}
		if tool != "" && tc.ToolName != tool {
			continue
		}
		result = append(result, tc)
	}

	return result
}

func matchesStatus(tc types.ToolCall, status string) bool {
	if status == "" {
		return true
	}

	switch status {
	case "error":
		return tc.Status == "error" || tc.Error != ""
	case "success":
		return tc.Status != "error" && tc.Error == ""
	default:
		return true
	}
}

func sortToolCalls(toolCalls []types.ToolCall, sortBy string, reverse bool) {
	if sortBy == "" {
		// Default sort by timestamp to maintain deterministic order
		sort.SliceStable(toolCalls, func(i, j int) bool {
			if reverse {
				return toolCalls[i].Timestamp > toolCalls[j].Timestamp
			}
			return toolCalls[i].Timestamp < toolCalls[j].Timestamp
		})
		return
	}

	sort.SliceStable(toolCalls, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "timestamp":
			less = toolCalls[i].Timestamp < toolCalls[j].Timestamp
		case "tool":
			less = toolCalls[i].ToolName < toolCalls[j].ToolName
		case "status":
			less = toolCalls[i].Status < toolCalls[j].Status
		case "uuid":
			less = toolCalls[i].UUID < toolCalls[j].UUID
		default:
			less = toolCalls[i].Timestamp < toolCalls[j].Timestamp
		}

		if reverse {
			return !less
		}
		return less
	})
}

func applyToolPagination(toolCalls []types.ToolCall, limit, offset int) []types.ToolCall {
	config := filter.PaginationConfig{Limit: limit, Offset: offset}
	return filter.ApplyPagination(toolCalls, config)
}

func isAdvancedWhere(where string) bool {
	lower := strings.ToLower(where)
	if strings.Contains(lower, " like ") || strings.Contains(lower, " between ") || strings.Contains(lower, " in ") {
		return true
	}
	if strings.Contains(lower, " and ") || strings.Contains(lower, " or ") {
		return true
	}
	if strings.ContainsAny(where, "%'_") {
		return true
	}
	if strings.Contains(where, ">") || strings.Contains(where, "<") {
		return true
	}
	return false
}

func normalizeAdvancedWhere(where string) string {
	replacer := strings.NewReplacer("=", " = ", ">", " > ", "<", " < ")
	normalized := replacer.Replace(where)
	return strings.Join(strings.Fields(normalized), " ")
}

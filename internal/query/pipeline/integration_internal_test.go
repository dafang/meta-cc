package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaleh/meta-cc/internal/types"
)

func TestQueryE2E_FilterTransformAggregate(t *testing.T) {
	entries := createComplexTestEntries()

	t.Run("filter_failed_reads", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Filter: FilterSpec{
				ToolName:   "Read",
				ToolStatus: "error",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		tools, ok := result.([]types.ToolCall)
		require.True(t, ok)
		assert.Len(t, tools, 1, "Should have exactly 1 failed Read")
		assert.Equal(t, "Read", tools[0].ToolName)
		assert.Equal(t, "error", tools[0].Status)
	})

	t.Run("count_tools_by_name", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Aggregate: AggregateSpec{
				Function: "count",
				Field:    "tool_name",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		results, ok := result.([]map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, results)

		toolCounts := make(map[string]int)
		for _, r := range results {
			toolName := r["tool_name"].(string)
			count := r["count"].(int)
			toolCounts[toolName] = count
		}

		assert.Equal(t, 2, toolCounts["Read"], "Should have 2 Read calls")
		assert.Equal(t, 1, toolCounts["Edit"], "Should have 1 Edit call")
	})

	t.Run("filter_by_git_branch", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
			Filter: FilterSpec{
				GitBranch: "feature/new-feature",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		filtered, ok := result.([]types.SessionEntry)
		require.True(t, ok)
		assert.NotEmpty(t, filtered)

		for _, entry := range filtered {
			assert.Equal(t, "feature/new-feature", entry.GitBranch)
		}
	})

	t.Run("filter_by_session", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
			Filter: FilterSpec{
				SessionID: "session-2",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		filtered, ok := result.([]types.SessionEntry)
		require.True(t, ok)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "session-2", filtered[0].SessionID)
	})

	t.Run("user_messages_with_text", func(t *testing.T) {
		params := QueryParams{
			Resource: "messages",
			Filter: FilterSpec{
				Role: "user",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		messages, ok := result.([]MessageView)
		require.True(t, ok)
		assert.NotEmpty(t, messages)

		userCount := 0
		for _, msg := range messages {
			if msg.Role == "user" {
				userCount++
				assert.NotEmpty(t, msg.UUID)
				assert.NotEmpty(t, msg.SessionID)
				assert.NotEmpty(t, msg.Timestamp)
			}
		}
		assert.Greater(t, userCount, 0, "Should have at least one user message")
	})

	t.Run("successful_tools_only", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Filter: FilterSpec{
				ToolStatus: "success",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		tools, ok := result.([]types.ToolCall)
		require.True(t, ok)
		assert.Len(t, tools, 2, "Should have 2 successful tool calls")

		for _, tool := range tools {
			assert.Equal(t, "success", tool.Status)
		}
	})

	t.Run("count_entries_by_type", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
			Aggregate: AggregateSpec{
				Function: "count",
				Field:    "type",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		results, ok := result.([]map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, results)

		typeCounts := make(map[string]int)
		for _, r := range results {
			entryType := r["type"].(string)
			count := r["count"].(int)
			typeCounts[entryType] = count
		}

		assert.Greater(t, typeCounts["user"], 0)
		assert.Greater(t, typeCounts["assistant"], 0)
	})
}

func TestQueryE2E_EmptyResults(t *testing.T) {
	entries := createComplexTestEntries()

	t.Run("filter_nonexistent_tool", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Filter: FilterSpec{
				ToolName: "NonExistentTool",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		tools, ok := result.([]types.ToolCall)
		require.True(t, ok)
		assert.Empty(t, tools)
	})

	t.Run("filter_nonexistent_session", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
			Filter: FilterSpec{
				SessionID: "nonexistent-session",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		filtered, ok := result.([]types.SessionEntry)
		require.True(t, ok)
		assert.Empty(t, filtered)
	})

	t.Run("empty_input_entries", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
		}

		result, err := Query([]types.SessionEntry{}, params)
		require.NoError(t, err)

		tools, ok := result.([]types.ToolCall)
		require.True(t, ok)
		assert.Empty(t, tools)
	})
}

func TestQueryE2E_ComplexFilters(t *testing.T) {
	entries := createComplexTestEntries()

	t.Run("filter_read_errors_only", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Filter: FilterSpec{
				ToolName:   "Read",
				ToolStatus: "error",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		tools, ok := result.([]types.ToolCall)
		require.True(t, ok)
		assert.Len(t, tools, 1)
		assert.Equal(t, "Read", tools[0].ToolName)
		assert.Equal(t, "error", tools[0].Status)
	})

	t.Run("filter_main_branch_user_messages", func(t *testing.T) {
		params := QueryParams{
			Resource: "messages",
			Filter: FilterSpec{
				Role:      "user",
				GitBranch: "main",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		messages, ok := result.([]MessageView)
		require.True(t, ok)
		assert.NotEmpty(t, messages)

		for _, msg := range messages {
			assert.Equal(t, "user", msg.Role)
		}
	})

	t.Run("filter_by_session_and_type", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
			Filter: FilterSpec{
				SessionID: "session-1",
				Type:      "user",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		filtered, ok := result.([]types.SessionEntry)
		require.True(t, ok)
		assert.NotEmpty(t, filtered)

		for _, entry := range filtered {
			assert.Equal(t, "session-1", entry.SessionID)
			assert.Equal(t, "user", entry.Type)
		}
	})
}

func TestQueryE2E_ErrorHandling(t *testing.T) {
	entries := createComplexTestEntries()

	t.Run("invalid_resource_type", func(t *testing.T) {
		params := QueryParams{
			Resource: "invalid_resource",
		}

		_, err := Query(entries, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("invalid_aggregate_function", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Aggregate: AggregateSpec{
				Function: "invalid_func",
			},
		}

		_, err := Query(entries, params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("nil_entries", func(t *testing.T) {
		params := QueryParams{
			Resource: "entries",
		}

		result, err := Query(nil, params)
		require.NoError(t, err)

		filtered, ok := result.([]types.SessionEntry)
		require.True(t, ok)
		assert.Empty(t, filtered)
	})
}

func TestQueryE2E_Aggregation(t *testing.T) {
	entries := createComplexTestEntries()

	t.Run("count_all_tools", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Aggregate: AggregateSpec{
				Function: "count",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		results, ok := result.([]map[string]interface{})
		require.True(t, ok)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0], "count")
		count := results[0]["count"].(int)
		assert.Equal(t, 3, count, "Should have 3 total tool calls")
	})

	t.Run("count_by_status", func(t *testing.T) {
		params := QueryParams{
			Resource: "tools",
			Aggregate: AggregateSpec{
				Function: "count",
				Field:    "status",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		results, ok := result.([]map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, results)

		statusCounts := make(map[string]int)
		for _, r := range results {
			status := r["status"].(string)
			count := r["count"].(int)
			statusCounts[status] = count
		}

		assert.Equal(t, 2, statusCounts["success"])
		assert.Equal(t, 1, statusCounts["error"])
	})

	t.Run("group_messages_by_role", func(t *testing.T) {
		params := QueryParams{
			Resource: "messages",
			Aggregate: AggregateSpec{
				Function: "count",
				Field:    "role",
			},
		}

		result, err := Query(entries, params)
		require.NoError(t, err)

		results, ok := result.([]map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, results)

		roleCounts := make(map[string]int)
		for _, r := range results {
			role := r["role"].(string)
			count := r["count"].(int)
			roleCounts[role] = count
		}

		assert.Greater(t, roleCounts["user"], 0)
		assert.Greater(t, roleCounts["assistant"], 0)
	})
}

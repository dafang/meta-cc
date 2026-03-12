package executor

import (
	"fmt"
	"strings"

	"github.com/yaleh/meta-cc/internal/config"
	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
)

// handlers.go implements the 10 convenience tools (Layer 1)
// These tools wrap ExecuteQuery() with pre-configured jq expressions

// HandleQueryUserMessages implements query_user_messages convenience tool
func (e *ToolExecutor) HandleQueryUserMessages(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	pattern := GetStringParam(args, "pattern", "")
	contentType := GetStringParam(args, "content_type", "string")
	limit := GetIntParam(args, "limit", 0)
	minContentLength := GetIntParam(args, "min_content_length", 0)
	maxContentLength := GetIntParam(args, "max_content_length", 0)
	workingDir := GetStringParam(args, "working_dir", "")
	sinceStr := GetStringParam(args, "since", "")
	untilStr := GetStringParam(args, "until", "")
	excludeSystem := GetBoolParam(args, "exclude_system_messages", false)

	// Parse time range before any session lookup (fail fast on bad input)
	tr, err := mcquery.ParseTimeRange(sinceStr, untilStr)
	if err != nil {
		return mcquery.QueryResult{}, err
	}

	// Content length filtering only applies to string content type
	if contentType != "string" && (minContentLength > 0 || maxContentLength > 0) {
		return mcquery.QueryResult{}, fmt.Errorf("content length filtering (min_content_length/max_content_length) only applies to string content type, not %q", contentType)
	}

	// Build jq filter based on content type
	var jqFilter string
	if contentType == "string" {
		jqFilter = `select(.type == "user" and (.message.content | type == "string"))`
	} else {
		jqFilter = `select(.type == "user" and (.message.content | type == "array"))`
	}

	// Add pattern filter if provided
	if pattern != "" {
		escapedPattern := EscapeJQ(pattern)
		jqFilter = fmt.Sprintf(`%s | select(.message.content | test("%s"))`, jqFilter, escapedPattern)
	}

	// Add content length filters if provided (string content only)
	if minContentLength > 0 {
		jqFilter = fmt.Sprintf(`%s | select(.message.content | length >= %d)`, jqFilter, minContentLength)
	}
	if maxContentLength > 0 {
		jqFilter = fmt.Sprintf(`%s | select(.message.content | length <= %d)`, jqFilter, maxContentLength)
	}

	// Exclude Claude Code system-injected messages (only applies to string content type)
	if excludeSystem && (contentType == "string" || contentType == "") {
		jqFilter += ` | select(.message.content | (startswith("<local-command-caveat>") or startswith("<command-name>") or startswith("<local-command-stdout>") or startswith("<task-notification>")) | not)`
	}

	return e.ExecuteQueryWithTimeRange(scope, jqFilter, limit, workingDir, tr)
}

// HandleQueryTools implements query_tools convenience tool
func (e *ToolExecutor) HandleQueryTools(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	toolName := GetStringParam(args, "tool", "")
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	// Base filter for all tool_use blocks
	jqFilter := `select(.type == "assistant") | select(.message.content[] | .type == "tool_use")`

	// Add tool name filter if provided
	if toolName != "" {
		escapedTool := EscapeJQ(toolName)
		jqFilter = fmt.Sprintf(`%s | select(.message.content[] | select(.type == "tool_use" and .name == "%s"))`, jqFilter, escapedTool)
	}

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryToolErrors implements query_tool_errors convenience tool
func (e *ToolExecutor) HandleQueryToolErrors(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	jqFilter := `select(.type == "user" and (.message.content | type == "array")) | ` +
		`select(.message.content[] | select(.type == "tool_result" and .is_error == true))`

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryTokenUsage implements query_token_usage convenience tool
func (e *ToolExecutor) HandleQueryTokenUsage(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	jqFilter := `select(.type == "assistant" and has("message")) | select(.message | has("usage"))`

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryConversationFlow implements query_conversation_flow convenience tool
func (e *ToolExecutor) HandleQueryConversationFlow(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")
	sinceStr := GetStringParam(args, "since", "")
	untilStr := GetStringParam(args, "until", "")

	tr, err := mcquery.ParseTimeRange(sinceStr, untilStr)
	if err != nil {
		return mcquery.QueryResult{}, err
	}

	jqFilter := `select(.type == "user" or .type == "assistant")`

	return e.ExecuteQueryWithTimeRange(scope, jqFilter, limit, workingDir, tr)
}

// HandleQuerySystemErrors implements query_system_errors convenience tool
func (e *ToolExecutor) HandleQuerySystemErrors(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	jqFilter := `select(.type == "system" and .subtype == "api_error")`

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryFileSnapshots implements query_file_snapshots convenience tool
func (e *ToolExecutor) HandleQueryFileSnapshots(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	jqFilter := `select(.type == "file-history-snapshot" and has("messageId"))`

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryTimestamps implements query_timestamps convenience tool
func (e *ToolExecutor) HandleQueryTimestamps(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")
	sinceStr := GetStringParam(args, "since", "")
	untilStr := GetStringParam(args, "until", "")

	tr, err := mcquery.ParseTimeRange(sinceStr, untilStr)
	if err != nil {
		return mcquery.QueryResult{}, err
	}

	jqFilter := `select(.timestamp != null)`

	return e.ExecuteQueryWithTimeRange(scope, jqFilter, limit, workingDir, tr)
}

// HandleQuerySummaries implements query_summaries convenience tool
func (e *ToolExecutor) HandleQuerySummaries(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	keyword := GetStringParam(args, "keyword", "")
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	jqFilter := `select(.type == "summary")`

	if keyword != "" {
		escapedKeyword := EscapeJQ(keyword)
		jqFilter = fmt.Sprintf(`%s | select(.summary | test("%s"; "i"))`, jqFilter, escapedKeyword)
	}

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// HandleQueryToolBlocks implements query_tool_blocks convenience tool
func (e *ToolExecutor) HandleQueryToolBlocks(cfg *config.Config, scope string, args map[string]interface{}) (mcquery.QueryResult, error) {
	blockType := GetStringParam(args, "block_type", "tool_use")
	limit := GetIntParam(args, "limit", 0)
	workingDir := GetStringParam(args, "working_dir", "")

	if blockType != "tool_use" && blockType != "tool_result" {
		return mcquery.QueryResult{}, fmt.Errorf("invalid block_type: %s (must be 'tool_use' or 'tool_result')", blockType)
	}

	var jqFilter string
	if blockType == "tool_use" {
		jqFilter = `select(.type == "assistant") | .message.content[] | select(.type == "tool_use")`
	} else {
		jqFilter = `select(.type == "user" and (.message.content | type == "array")) | .message.content[] | select(.type == "tool_result")`
	}

	return e.ExecuteQuery(scope, jqFilter, limit, workingDir)
}

// EscapeJQ escapes special characters in strings for jq expressions.
func EscapeJQ(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

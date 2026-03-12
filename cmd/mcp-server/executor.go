package main

import (
	"time"

	"github.com/yaleh/meta-cc/internal/config"
	execpkg "github.com/yaleh/meta-cc/internal/mcp/executor"
	filterspkg "github.com/yaleh/meta-cc/internal/mcp/filters"
	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
)

// Phase 25: convenience tools (query_user_messages, query_tools, etc.) execute jq internally.
// The executor does NOT apply jq_filter a second time for Phase 25 tools.
// Business logic for all Phase 25 tools lives in internal/mcp/executor.

// ToolExecutor wraps execpkg.ToolExecutor so we can define additional
// methods in this package (e.g. lowercase wrappers for test compatibility).
type ToolExecutor struct {
	*execpkg.ToolExecutor
}

// toolPipelineConfig is a local struct kept for backward compatibility with tests.
// Business logic lives in execpkg.ToolPipelineConfig / execpkg.NewToolPipelineConfig.
type toolPipelineConfig struct {
	jqFilter         string
	statsOnly        bool
	statsFirst       bool
	outputFormat     string
	maxMessageLength int
	contentSummary   bool
	previewLength    int
	groupBySession   bool
	statsLevel       string
	contextTurns     int
}

func newToolPipelineConfig(args map[string]interface{}) toolPipelineConfig {
	return toolPipelineConfig{
		jqFilter:         getStringParam(args, "jq_filter", ".[]"),
		statsOnly:        getBoolParam(args, "stats_only", false),
		statsFirst:       getBoolParam(args, "stats_first", false),
		outputFormat:     getStringParam(args, "output_format", "jsonl"),
		maxMessageLength: getIntParam(args, "max_message_length", 0),
		contentSummary:   getBoolParam(args, "content_summary", false),
		previewLength:    getIntParam(args, "preview_length", DefaultPreviewLength),
		groupBySession:   getBoolParam(args, "group_by_session", false),
		statsLevel:       getStringParam(args, "stats_level", "turn"),
		contextTurns:     getIntParam(args, "context_turns", 0),
	}
}

func (c toolPipelineConfig) requiresMessageFilters() bool {
	return c.maxMessageLength > 0 || c.contentSummary
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{ToolExecutor: execpkg.NewToolExecutor()}
}

// ExecuteTool delegates to the internal executor.
func (e *ToolExecutor) ExecuteTool(cfg *config.Config, toolName string, args map[string]interface{}) (string, error) {
	return e.ToolExecutor.ExecuteTool(cfg, toolName, args)
}

func determineScope(toolName string, args map[string]interface{}) string {
	return execpkg.DetermineScope(toolName, args)
}

func recordToolSuccess(toolName, scope string, start time.Time) {
	execpkg.RecordToolSuccess(toolName, scope, start)
}

func recordToolFailure(toolName, scope string, start time.Time, errorType string) {
	execpkg.RecordToolFailure(toolName, scope, start, errorType)
}

// Helper functions - delegate to executor package
func getStringParam(args map[string]interface{}, key, defaultVal string) string {
	return execpkg.GetStringParam(args, key, defaultVal)
}

func getBoolParam(args map[string]interface{}, key string, defaultVal bool) bool {
	return execpkg.GetBoolParam(args, key, defaultVal)
}

func getIntParam(args map[string]interface{}, key string, defaultVal int) int {
	return execpkg.GetIntParam(args, key, defaultVal)
}

func getFloatParam(args map[string]interface{}, key string, defaultVal float64) float64 {
	return execpkg.GetFloatParam(args, key, defaultVal)
}

// parseJSONL is a lowercase wrapper for test backward compatibility.
func (e *ToolExecutor) parseJSONL(jsonlData string) ([]interface{}, error) {
	return e.ToolExecutor.ParseJSONL(jsonlData)
}

// applyMessageFiltersToData is a lowercase wrapper for test backward compatibility.
func (e *ToolExecutor) applyMessageFiltersToData(messages []interface{}, maxMessageLength int, contentSummary bool, previewLength int) []interface{} {
	return e.ToolExecutor.ApplyMessageFiltersToData(messages, maxMessageLength, contentSummary, previewLength)
}

// Ensure imports are used
var _ mcquery.QueryResult
var _ = filterspkg.DefaultPreviewLength
var _ = (*config.Config)(nil)

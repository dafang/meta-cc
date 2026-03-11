package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/yaleh/meta-cc/internal/analysis"
	"github.com/yaleh/meta-cc/internal/config"
	mcerrors "github.com/yaleh/meta-cc/internal/errors"
	filterspkg "github.com/yaleh/meta-cc/internal/mcp/filters"
	"github.com/yaleh/meta-cc/internal/mcp/metrics"
	pipelinepkg "github.com/yaleh/meta-cc/internal/mcp/pipeline"
	schemapkg "github.com/yaleh/meta-cc/internal/mcp/schema"
	querypkg "github.com/yaleh/meta-cc/internal/query"
)

type ToolExecutor struct {
	analysisSvc analysis.AnalysisService
}

type toolPipelineConfig struct {
	jqFilter         string
	statsOnly        bool
	statsFirst       bool
	outputFormat     string
	maxMessageLength int
	contentSummary   bool
	previewLength    int
	groupBySession   bool
	statsLevel       string // "turn" (default) or "session"
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

func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		analysisSvc: analysis.New(),
	}
}

func determineScope(toolName string, args map[string]interface{}) string {
	defaultScope := "project"
	if toolName == "get_session_stats" {
		defaultScope = "session"
	}
	return getStringParam(args, "scope", defaultScope)
}

func recordToolSuccess(toolName, scope string, start time.Time) {
	elapsed := time.Since(start)
	metrics.RecordToolCall(toolName, scope, "success")
	metrics.RecordToolExecutionDuration(toolName, scope, elapsed)
}

func recordToolFailure(toolName, scope string, start time.Time, errorType string) {
	elapsed := time.Since(start)
	metrics.RecordToolCall(toolName, scope, "error")
	metrics.RecordToolExecutionDuration(toolName, scope, elapsed)
	metrics.RecordError(toolName, errorType, metrics.GetErrorSeverity(errorType))
}

func (e *ToolExecutor) executeSpecialTool(cfg *config.Config, toolName, scope string, args map[string]interface{}, start time.Time) (string, bool, error) {
	switch toolName {
	case "cleanup_temp_files":
		output, err := executeCleanupTool(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_session_directory":
		result, err := handleGetSessionDirectory(context.Background(), args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		// Convert result to JSON string
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		recordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "inspect_session_files":
		result, err := handleInspectSessionFiles(context.Background(), args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		// Convert result to JSON string
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		recordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "execute_stage2_query":
		result, err := handleExecuteStage2Query(context.Background(), args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		// Convert result to JSON string
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		recordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "analyze_bugs":
		output, err := e.analysisSvc.AnalyzeBugs(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "analyze_errors":
		output, err := e.analysisSvc.AnalyzeErrors(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "quality_scan":
		output, err := e.analysisSvc.QualityScan(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_work_patterns":
		output, err := e.analysisSvc.GetWorkPatterns(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_session_metadata":
		result, err := handleGetSessionMetadata(context.Background(), args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		// Convert result to JSON string
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		recordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "get_timeline":
		output, err := e.analysisSvc.GetTimeline(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_tech_debt":
		output, err := e.analysisSvc.GetTechDebt(args)
		if err != nil {
			errorType := classifyError(err)
			recordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		recordToolSuccess(toolName, scope, start)
		return output, true, nil

	default:
		return "", false, nil
	}
}

// ExecuteTool executes a meta-cc command and applies jq filtering
func (e *ToolExecutor) ExecuteTool(cfg *config.Config, toolName string, args map[string]interface{}) (string, error) {
	scope := determineScope(toolName, args)
	start := time.Now()

	if output, handled, err := e.executeSpecialTool(cfg, toolName, scope, args, start); handled {
		return output, err
	}

	// Validate scope value before any further processing
	if scope != "project" && scope != "session" {
		recordToolFailure(toolName, scope, start, "validation_error")
		return "", fmt.Errorf("invalid scope %q: must be \"project\" or \"session\"", scope)
	}

	// Validate tool exists via schema lookup before dispatch
	schema, schemaErr := getToolSchemaByName(toolName)
	if schemaErr != nil {
		recordToolFailure(toolName, scope, start, "validation_error")
		return "", fmt.Errorf("unknown tool %s in executor: %w", toolName, mcerrors.ErrUnknownTool)
	}

	// Validate that all provided argument keys are declared in the tool schema
	if validationErr := schemapkg.ValidateArgKeys(args, schema); validationErr != nil {
		recordToolFailure(toolName, scope, start, "validation_error")
		return "", validationErr
	}

	config := newToolPipelineConfig(args)
	var queryResult QueryResult
	var err error

	switch toolName {
	// Phase 27 Stage 27.1: query and query_raw tools removed
	// Use the 10 shortcut query tools instead

	// Layer 1: Convenience Tools (10 high-frequency queries)
	// Phase 27 Stage 27.5: These tools now return QueryResult directly
	case "query_user_messages":
		queryResult, err = e.handleQueryUserMessages(cfg, scope, args)
	case "query_tools":
		queryResult, err = e.handleQueryTools(cfg, scope, args)
	case "query_tool_errors":
		queryResult, err = e.handleQueryToolErrors(cfg, scope, args)
	case "query_token_usage":
		queryResult, err = e.handleQueryTokenUsage(cfg, scope, args)
	case "query_conversation_flow":
		queryResult, err = e.handleQueryConversationFlow(cfg, scope, args)
	case "query_system_errors":
		queryResult, err = e.handleQuerySystemErrors(cfg, scope, args)
	case "query_file_snapshots":
		queryResult, err = e.handleQueryFileSnapshots(cfg, scope, args)
	case "query_timestamps":
		queryResult, err = e.handleQueryTimestamps(cfg, scope, args)
	case "query_summaries":
		queryResult, err = e.handleQuerySummaries(cfg, scope, args)
	case "query_tool_blocks":
		queryResult, err = e.handleQueryToolBlocks(cfg, scope, args)
	}

	if err != nil {
		errorType := classifyError(err)
		slog.Error("tool execution failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", errorType,
		)
		recordToolFailure(toolName, scope, start, errorType)
		return "", err
	}

	// Phase 27 Stage 27.5: Convenience tools now return QueryResult directly
	// No need to parse JSONL or apply jq filters (filters are already applied internally)
	// Note: Phase 25 originally introduced these convenience tools with jq execution
	// Note: Phase 51 moved applyMessageFiltersToData into buildResponse so stats always
	// see raw data with original camelCase sessionId (fixes session_count=0 bug).

	output, err := e.buildResponse(cfg, queryResult, args, toolName, config)
	if err != nil {
		return "", err
	}

	slog.Debug("tool execution pipeline completed successfully",
		"tool_name", toolName,
		"output_length", len(output),
	)

	recordToolSuccess(toolName, scope, start)
	return output, nil
}

func (e *ToolExecutor) buildResponse(cfg *config.Config, result QueryResult, args map[string]interface{}, toolName string, pipeline toolPipelineConfig) (string, error) {
	rawData := result.Entries

	var output string
	var err error

	if pipeline.statsLevel != "" && pipeline.statsLevel != "turn" && pipeline.statsLevel != "session" {
		return "", fmt.Errorf("invalid stats_level: must be 'turn' or 'session'")
	}

	if pipeline.groupBySession && pipeline.statsOnly {
		return "", fmt.Errorf("group_by_session and stats_only are mutually exclusive")
	}

	if pipeline.statsOnly {
		// stats_only: compute stats from raw data (camelCase sessionId preserved)
		output, err = pipelinepkg.BuildStatsOnlyResponse(rawData, toolName, pipeline.statsLevel)
		if err != nil {
			return "", err
		}
		return pipelinepkg.InjectWarnings(output, result.Warnings)
	}

	// Apply message filters for detail rendering AFTER stats path
	// so stats always see raw data with original camelCase sessionId.
	parsedData := rawData
	if toolName == "query_user_messages" && pipeline.requiresMessageFilters() {
		parsedData = e.applyMessageFiltersToData(rawData, pipeline.maxMessageLength, pipeline.contentSummary, pipeline.previewLength)
	}

	// Expand context turns (before groupBySession so group_by_session sees context turns too)
	if pipeline.contextTurns > 0 && toolName == "query_user_messages" &&
		getStringParam(args, "content_type", "string") != "array" {
		baseDir, err := getQueryBaseDir(
			getStringParam(args, "scope", "project"),
			getStringParam(args, "working_dir", ""),
		)
		if err != nil {
			return "", err
		}
		parsedData, err = e.expandContextTurns(parsedData, pipeline.contextTurns, baseDir)
		if err != nil {
			return "", err
		}
	}

	// Group by session after message filters (so content_summary is applied to turns first)
	if pipeline.groupBySession && toolName == "query_user_messages" {
		parsedData = querypkg.GroupBySession(parsedData)
	}

	adaptFn := func(data []interface{}, params map[string]interface{}, toolName string) (interface{}, error) {
		return adaptResponse(cfg, data, params, toolName)
	}
	serializeFn := func(response interface{}) (string, error) {
		return serializeResponse(response)
	}

	if pipeline.statsFirst {
		output, err = pipelinepkg.BuildStatsFirstResponse(rawData, parsedData, args, toolName, pipeline.statsLevel, adaptFn, serializeFn)
	} else {
		output, err = pipelinepkg.BuildStandardResponse(parsedData, args, toolName, adaptFn, serializeFn)
	}

	if err != nil {
		return "", err
	}

	// Inject warnings into the JSON response
	return pipelinepkg.InjectWarnings(output, result.Warnings)
}

// parseJSONL parses JSONL string into array of interfaces
func (e *ToolExecutor) parseJSONL(jsonlData string) ([]interface{}, error) {
	jsonlData = strings.TrimSpace(jsonlData)

	// Handle special cases: empty input or "[]" (exit code 2 scenario)
	if jsonlData == "" || jsonlData == "[]" {
		slog.Debug("parseJSONL: empty input or no results",
			"input", jsonlData,
		)
		return []interface{}{}, nil
	}

	lines := strings.Split(jsonlData, "\n")
	var data []interface{}

	for i, line := range lines {
		if line == "" {
			continue
		}

		var obj interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			slog.Error("failed to parse JSONL line",
				"line_number", i+1,
				"line_content", line,
				"error", err.Error(),
				"error_type", "parse_error",
			)
			return nil, fmt.Errorf("invalid JSON on line %d: %w", i+1, mcerrors.ErrParseError)
		}
		data = append(data, obj)
	}

	slog.Debug("parseJSONL completed",
		"record_count", len(data),
	)

	return data, nil
}

// applyMessageFiltersToData applies content truncation or summary mode to user messages (data array)
func (e *ToolExecutor) applyMessageFiltersToData(messages []interface{}, maxMessageLength int, contentSummary bool, previewLength int) []interface{} {
	return filterspkg.ApplyMessageFiltersToData(messages, maxMessageLength, contentSummary, previewLength)
}

// expandContextTurns takes rawData (matched entries) and expands each matched turn
// by including up to N turns before and after it (within the same session).
// Matched turns are marked with "context":false; surrounding context turns with "context":true.
// Overlapping windows are merged (no duplicates). Order is chronological within each session.
func (e *ToolExecutor) expandContextTurns(rawData []interface{}, N int, baseDir string) ([]interface{}, error) {
	return filterspkg.ExpandContextTurns(rawData, N, baseDir)
}

// Helper functions
func getStringParam(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultVal
}

func getBoolParam(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultVal
}

func getIntParam(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultVal
}

func getFloatParam(args map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := args[key].(float64); ok {
		return v
	}
	return defaultVal
}

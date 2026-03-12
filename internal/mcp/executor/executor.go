package executor

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
	obspkg "github.com/yaleh/meta-cc/internal/mcp/observability"
	pipelinepkg "github.com/yaleh/meta-cc/internal/mcp/pipeline"
	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
	schemapkg "github.com/yaleh/meta-cc/internal/mcp/schema"
	toolspkg "github.com/yaleh/meta-cc/internal/mcp/tools"
	internalquery "github.com/yaleh/meta-cc/internal/query"
)

// ToolExecutor executes MCP tools for session history analysis.
type ToolExecutor struct {
	AnalysisSvc analysis.AnalysisService
}

// ToolPipelineConfig holds configuration for a tool execution pipeline.
type ToolPipelineConfig struct {
	JQFilter         string
	StatsOnly        bool
	StatsFirst       bool
	OutputFormat     string
	MaxMessageLength int
	ContentSummary   bool
	PreviewLength    int
	GroupBySession   bool
	StatsLevel       string // "turn" (default) or "session"
	ContextTurns     int
}

// NewToolPipelineConfig creates a ToolPipelineConfig from args map.
func NewToolPipelineConfig(args map[string]interface{}) ToolPipelineConfig {
	return ToolPipelineConfig{
		JQFilter:         GetStringParam(args, "jq_filter", ".[]"),
		StatsOnly:        GetBoolParam(args, "stats_only", false),
		StatsFirst:       GetBoolParam(args, "stats_first", false),
		OutputFormat:     GetStringParam(args, "output_format", "jsonl"),
		MaxMessageLength: GetIntParam(args, "max_message_length", 0),
		ContentSummary:   GetBoolParam(args, "content_summary", false),
		PreviewLength:    GetIntParam(args, "preview_length", filterspkg.DefaultPreviewLength),
		GroupBySession:   GetBoolParam(args, "group_by_session", false),
		StatsLevel:       GetStringParam(args, "stats_level", "turn"),
		ContextTurns:     GetIntParam(args, "context_turns", 0),
	}
}

func (c ToolPipelineConfig) requiresMessageFilters() bool {
	return c.MaxMessageLength > 0 || c.ContentSummary
}

// NewToolExecutor creates a new ToolExecutor.
func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{
		AnalysisSvc: analysis.New(),
	}
}

// DetermineScope returns the scope for a tool call.
func DetermineScope(toolName string, args map[string]interface{}) string {
	defaultScope := "project"
	if toolName == "get_session_stats" {
		defaultScope = "session"
	}
	return GetStringParam(args, "scope", defaultScope)
}

// RecordToolSuccess records a successful tool execution.
func RecordToolSuccess(toolName, scope string, start time.Time) {
	elapsed := time.Since(start)
	metrics.RecordToolCall(toolName, scope, "success")
	metrics.RecordToolExecutionDuration(toolName, scope, elapsed)
}

// RecordToolFailure records a failed tool execution.
func RecordToolFailure(toolName, scope string, start time.Time, errorType string) {
	elapsed := time.Since(start)
	metrics.RecordToolCall(toolName, scope, "error")
	metrics.RecordToolExecutionDuration(toolName, scope, elapsed)
	metrics.RecordError(toolName, errorType, metrics.GetErrorSeverity(errorType))
}

// ExecuteSpecialTool handles special tools that don't go through the standard pipeline.
func (e *ToolExecutor) ExecuteSpecialTool(cfg *config.Config, toolName, scope string, args map[string]interface{}, start time.Time) (string, bool, error) {
	switch toolName {
	case "cleanup_temp_files":
		output, err := responsepkg.ExecuteCleanupTool(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_session_directory":
		result, err := mcquery.HandleGetSessionDirectory(context.Background(), args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		RecordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "inspect_session_files":
		result, err := mcquery.HandleInspectSessionFiles(context.Background(), args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		RecordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "execute_stage2_query":
		result, err := mcquery.HandleExecuteStage2Query(context.Background(), args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		RecordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "analyze_bugs":
		output, err := e.AnalysisSvc.AnalyzeBugs(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "analyze_errors":
		output, err := e.AnalysisSvc.AnalyzeErrors(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "quality_scan":
		output, err := e.AnalysisSvc.QualityScan(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_work_patterns":
		output, err := e.AnalysisSvc.GetWorkPatterns(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_session_metadata":
		result, err := mcquery.HandleGetSessionMetadata(context.Background(), args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		jsonData, err := json.Marshal(result)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, fmt.Errorf("failed to marshal result: %w", err)
		}
		RecordToolSuccess(toolName, scope, start)
		return string(jsonData), true, nil

	case "get_timeline":
		output, err := e.AnalysisSvc.GetTimeline(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	case "get_tech_debt":
		output, err := e.AnalysisSvc.GetTechDebt(args)
		if err != nil {
			errorType := obspkg.ClassifyError(err)
			RecordToolFailure(toolName, scope, start, errorType)
			return "", true, err
		}
		RecordToolSuccess(toolName, scope, start)
		return output, true, nil

	default:
		return "", false, nil
	}
}

// ExecuteTool executes a meta-cc command and applies jq filtering.
func (e *ToolExecutor) ExecuteTool(cfg *config.Config, toolName string, args map[string]interface{}) (string, error) {
	scope := DetermineScope(toolName, args)
	start := time.Now()

	if output, handled, err := e.ExecuteSpecialTool(cfg, toolName, scope, args, start); handled {
		return output, err
	}

	// Validate scope value before any further processing
	if scope != "project" && scope != "session" {
		RecordToolFailure(toolName, scope, start, "validation_error")
		return "", fmt.Errorf("invalid scope %q: must be \"project\" or \"session\"", scope)
	}

	// Validate tool exists via schema lookup before dispatch
	schemaIndex := toolspkg.BuildToolSchemaIndex()
	schema, schemaErr := toolspkg.GetToolSchemaByName(schemaIndex, toolName)
	if schemaErr != nil {
		RecordToolFailure(toolName, scope, start, "validation_error")
		return "", fmt.Errorf("unknown tool %s in executor: %w", toolName, mcerrors.ErrUnknownTool)
	}

	// Validate that all provided argument keys are declared in the tool schema
	if validationErr := schemapkg.ValidateArgKeys(args, schema); validationErr != nil {
		RecordToolFailure(toolName, scope, start, "validation_error")
		return "", validationErr
	}

	pipeline := NewToolPipelineConfig(args)
	var queryResult mcquery.QueryResult
	var err error

	switch toolName {
	// Phase 27 Stage 27.1: query and query_raw tools removed
	// Use the 10 shortcut query tools instead

	// Layer 1: Convenience Tools (10 high-frequency queries)
	case "query_user_messages":
		queryResult, err = e.HandleQueryUserMessages(cfg, scope, args)
	case "query_tools":
		queryResult, err = e.HandleQueryTools(cfg, scope, args)
	case "query_tool_errors":
		queryResult, err = e.HandleQueryToolErrors(cfg, scope, args)
	case "query_token_usage":
		queryResult, err = e.HandleQueryTokenUsage(cfg, scope, args)
	case "query_conversation_flow":
		queryResult, err = e.HandleQueryConversationFlow(cfg, scope, args)
	case "query_system_errors":
		queryResult, err = e.HandleQuerySystemErrors(cfg, scope, args)
	case "query_file_snapshots":
		queryResult, err = e.HandleQueryFileSnapshots(cfg, scope, args)
	case "query_timestamps":
		queryResult, err = e.HandleQueryTimestamps(cfg, scope, args)
	case "query_summaries":
		queryResult, err = e.HandleQuerySummaries(cfg, scope, args)
	case "query_tool_blocks":
		queryResult, err = e.HandleQueryToolBlocks(cfg, scope, args)
	}

	if err != nil {
		errorType := obspkg.ClassifyError(err)
		slog.Error("tool execution failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", errorType,
		)
		RecordToolFailure(toolName, scope, start, errorType)
		return "", err
	}

	output, err := e.BuildResponse(cfg, queryResult, args, toolName, pipeline)
	if err != nil {
		return "", err
	}

	slog.Debug("tool execution pipeline completed successfully",
		"tool_name", toolName,
		"output_length", len(output),
	)

	RecordToolSuccess(toolName, scope, start)
	return output, nil
}

// BuildResponse constructs the final response for a query result.
func (e *ToolExecutor) BuildResponse(cfg *config.Config, result mcquery.QueryResult, args map[string]interface{}, toolName string, pipeline ToolPipelineConfig) (string, error) {
	rawData := result.Entries

	var output string
	var err error

	if pipeline.StatsLevel != "" && pipeline.StatsLevel != "turn" && pipeline.StatsLevel != "session" {
		return "", fmt.Errorf("invalid stats_level: must be 'turn' or 'session'")
	}

	if pipeline.GroupBySession && pipeline.StatsOnly {
		return "", fmt.Errorf("group_by_session and stats_only are mutually exclusive")
	}

	if pipeline.StatsOnly {
		output, err = pipelinepkg.BuildStatsOnlyResponse(rawData, toolName, pipeline.StatsLevel)
		if err != nil {
			return "", err
		}
		return pipelinepkg.InjectWarnings(output, result.Warnings)
	}

	// Apply message filters for detail rendering AFTER stats path
	parsedData := rawData
	if toolName == "query_user_messages" && pipeline.requiresMessageFilters() {
		parsedData = e.ApplyMessageFiltersToData(rawData, pipeline.MaxMessageLength, pipeline.ContentSummary, pipeline.PreviewLength)
	}

	// Expand context turns
	if pipeline.ContextTurns > 0 && toolName == "query_user_messages" &&
		GetStringParam(args, "content_type", "string") != "array" {
		baseDir, err := mcquery.GetQueryBaseDir(
			GetStringParam(args, "scope", "project"),
			GetStringParam(args, "working_dir", ""),
		)
		if err != nil {
			return "", err
		}
		parsedData, err = e.ExpandContextTurns(parsedData, pipeline.ContextTurns, baseDir)
		if err != nil {
			return "", err
		}
	}

	// Group by session after message filters
	if pipeline.GroupBySession && toolName == "query_user_messages" {
		parsedData = internalquery.GroupBySession(parsedData)
	}

	if pipeline.StatsFirst {
		output, err = pipelinepkg.BuildStatsFirstResponse(cfg, rawData, parsedData, args, toolName, pipeline.StatsLevel)
	} else {
		output, err = pipelinepkg.BuildStandardResponse(cfg, parsedData, args, toolName)
	}

	if err != nil {
		return "", err
	}

	return pipelinepkg.InjectWarnings(output, result.Warnings)
}

// ParseJSONL parses JSONL string into array of interfaces.
func (e *ToolExecutor) ParseJSONL(jsonlData string) ([]interface{}, error) {
	jsonlData = strings.TrimSpace(jsonlData)

	if jsonlData == "" || jsonlData == "[]" {
		slog.Debug("ParseJSONL: empty input or no results",
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

	slog.Debug("ParseJSONL completed",
		"record_count", len(data),
	)

	return data, nil
}

// ApplyMessageFiltersToData applies content truncation or summary mode to user messages.
func (e *ToolExecutor) ApplyMessageFiltersToData(messages []interface{}, maxMessageLength int, contentSummary bool, previewLength int) []interface{} {
	return filterspkg.ApplyMessageFiltersToData(messages, maxMessageLength, contentSummary, previewLength)
}

// ExpandContextTurns expands matched turns by including N surrounding turns.
func (e *ToolExecutor) ExpandContextTurns(rawData []interface{}, N int, baseDir string) ([]interface{}, error) {
	return filterspkg.ExpandContextTurns(rawData, N, baseDir)
}

// ExecuteQuery is an internal helper for convenience tools.
func (e *ToolExecutor) ExecuteQuery(scope string, jqFilter string, limit int, workingDir string) (mcquery.QueryResult, error) {
	return e.ExecuteQueryWithTimeRange(scope, jqFilter, limit, workingDir, mcquery.ParsedTimeRange{})
}

// ExecuteQueryWithTimeRange is like ExecuteQuery but applies time-range filtering.
func (e *ToolExecutor) ExecuteQueryWithTimeRange(scope string, jqFilter string, limit int, workingDir string, tr mcquery.ParsedTimeRange) (mcquery.QueryResult, error) {
	baseDir, err := mcquery.GetQueryBaseDir(scope, workingDir)
	if err != nil {
		return mcquery.QueryResult{}, fmt.Errorf("failed to get base directory: %w", err)
	}

	executor := mcquery.NewQueryExecutor(baseDir)

	code, err := executor.CompileExpression(jqFilter)
	if err != nil {
		return mcquery.QueryResult{}, fmt.Errorf("invalid jq expression: %w", err)
	}

	files, err := mcquery.GetJSONLFiles(baseDir)
	if err != nil {
		return mcquery.QueryResult{}, fmt.Errorf("failed to list JSONL files: %w", err)
	}

	if len(files) == 0 {
		return mcquery.QueryResult{}, fmt.Errorf("no JSONL files found in %s", baseDir)
	}

	ctx := context.Background()
	result := executor.StreamFilesWithTimeRange(ctx, files, code, limit, tr)
	return result, nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/yaleh/meta-cc/internal/analysis"
	"github.com/yaleh/meta-cc/internal/config"
	mcerrors "github.com/yaleh/meta-cc/internal/errors"
	querypkg "github.com/yaleh/meta-cc/internal/query"
)

// timestampStatsTools is the set of tool names that should use GenerateTimestampStats
// instead of GenerateStats when producing stats_only or stats_first output.
// These tools return records that lack a tool/ToolName field but have timestamp data,
// so time-bucketed stats are more meaningful than the meaningless "unknown" key.
var timestampStatsTools = map[string]bool{
	"query_user_messages":     true,
	"query_conversation_flow": true,
	"query_timestamps":        true,
	"query_summaries":         true,
}

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
	RecordToolCall(toolName, scope, "success")
	RecordToolExecutionDuration(toolName, scope, elapsed)
}

func recordToolFailure(toolName, scope string, start time.Time, errorType string) {
	elapsed := time.Since(start)
	RecordToolCall(toolName, scope, "error")
	RecordToolExecutionDuration(toolName, scope, elapsed)
	RecordError(toolName, errorType, GetErrorSeverity(errorType))
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
	if validationErr := validateArgKeys(args, schema); validationErr != nil {
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
		output, err = e.buildStatsOnlyResponse(rawData, toolName, pipeline.statsLevel)
		if err != nil {
			return "", err
		}
		return injectWarnings(output, result.Warnings)
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

	if pipeline.statsFirst {
		output, err = e.buildStatsFirstResponse(cfg, rawData, parsedData, args, toolName, pipeline.statsLevel)
	} else {
		output, err = e.buildStandardResponse(cfg, parsedData, args, toolName)
	}

	if err != nil {
		return "", err
	}

	// Inject warnings into the JSON response
	return injectWarnings(output, result.Warnings)
}

// injectWarnings adds a "warnings" field to a JSON response string.
// If the output is valid JSON object, it adds the field. Otherwise returns as-is.
func injectWarnings(output string, warnings []string) (string, error) {
	if warnings == nil {
		warnings = []string{}
	}

	// Try to parse as JSON object
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		// Not a JSON object (e.g., stats_only plain text) — skip injection
		return output, nil
	}

	parsed["warnings"] = warnings

	result, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("failed to re-serialize response with warnings: %w", err)
	}
	return string(result), nil
}

func (e *ToolExecutor) buildStatsOnlyResponse(parsedData []interface{}, toolName string, statsLevel string) (string, error) {
	jsonlData, err := e.dataToJSONL(parsedData)
	if err != nil {
		slog.Error("dataToJSONL conversion failed (stats_only)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	var output string
	if statsLevel == "session" && toolName == "query_user_messages" {
		output, err = querypkg.GenerateSessionStats(jsonlData)
	} else if timestampStatsTools[toolName] {
		output, err = querypkg.GenerateTimestampStats(jsonlData)
	} else {
		output, err = querypkg.GenerateStats(jsonlData)
	}
	if err != nil {
		slog.Error("stats generation failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "execution_error",
		)
		return "", err
	}

	return output, nil
}

func (e *ToolExecutor) buildStatsFirstResponse(cfg *config.Config, rawData []interface{}, parsedData []interface{}, args map[string]interface{}, toolName string, statsLevel string) (string, error) {
	// Use rawData for stats (sessionId field preserved, not renamed by content_summary)
	jsonlData, err := e.dataToJSONL(rawData)
	if err != nil {
		slog.Error("dataToJSONL conversion failed (stats_first)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	var stats string
	if statsLevel == "session" && toolName == "query_user_messages" {
		stats, _ = querypkg.GenerateSessionStats(jsonlData)
	} else if timestampStatsTools[toolName] {
		stats, _ = querypkg.GenerateTimestampStats(jsonlData)
	} else {
		stats, _ = querypkg.GenerateStats(jsonlData)
	}

	// Use parsedData for detail rendering (may have content_summary applied)
	response, err := adaptResponse(cfg, parsedData, args, toolName)
	if err != nil {
		slog.Error("response adaptation failed (stats_first)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "execution_error",
		)
		return "", err
	}

	serialized, err := serializeResponse(response)
	if err != nil {
		slog.Error("response serialization failed (stats_first)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	return stats + "\n---\n" + serialized, nil
}

func (e *ToolExecutor) buildStandardResponse(cfg *config.Config, parsedData []interface{}, args map[string]interface{}, toolName string) (string, error) {
	response, err := adaptResponse(cfg, parsedData, args, toolName)
	if err != nil {
		slog.Error("response adaptation failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "execution_error",
		)
		return "", fmt.Errorf("response adaptation error for tool %s: %w", toolName, err)
	}

	output, err := serializeResponse(response)
	if err != nil {
		slog.Error("response serialization failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	return output, nil
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

// dataToJSONL converts array of interfaces to JSONL string
func (e *ToolExecutor) dataToJSONL(data []interface{}) (string, error) {
	var output strings.Builder
	for i, record := range data {
		jsonBytes, err := json.Marshal(record)
		if err != nil {
			slog.Error("failed to marshal record to JSON",
				"record_index", i,
				"error", err.Error(),
				"error_type", "parse_error",
			)
			return "", err
		}
		output.Write(jsonBytes)
		output.WriteString("\n")
	}
	return output.String(), nil
}

// applyMessageFiltersToData applies content truncation or summary mode to user messages (data array)
func (e *ToolExecutor) applyMessageFiltersToData(messages []interface{}, maxMessageLength int, contentSummary bool, previewLength int) []interface{} {
	if contentSummary {
		return ApplyContentSummary(messages, previewLength)
	}
	return TruncateMessageContent(messages, maxMessageLength)
}

// expandContextTurns takes rawData (matched entries) and expands each matched turn
// by including up to N turns before and after it (within the same session).
// Matched turns are marked with "context":false; surrounding context turns with "context":true.
// Overlapping windows are merged (no duplicates). Order is chronological within each session.
func (e *ToolExecutor) expandContextTurns(rawData []interface{}, N int, baseDir string) ([]interface{}, error) {
	if N <= 0 || len(rawData) == 0 {
		return rawData, nil
	}

	// 1. Build set of matched UUIDs and collect distinct sessionIds (preserving order)
	matchedUUIDs := make(map[string]bool)
	var sessionOrder []string
	sessionSeen := make(map[string]bool)

	for _, entry := range rawData {
		obj, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		uuid, _ := obj["uuid"].(string)
		if uuid != "" {
			matchedUUIDs[uuid] = true
		}
		// Raw data uses camelCase sessionId
		sessionID, _ := obj["sessionId"].(string)
		if sessionID == "" {
			// Fallback to snake_case
			sessionID, _ = obj["session_id"].(string)
		}
		if sessionID != "" && !sessionSeen[sessionID] {
			sessionOrder = append(sessionOrder, sessionID)
			sessionSeen[sessionID] = true
		}
	}

	// 2. For each distinct sessionId, load all turns for that session
	sessionTurns := make(map[string][]interface{})
	for _, sessionID := range sessionOrder {
		turns, err := loadTurnsForSession(baseDir, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to load turns for session %s: %w", sessionID, err)
		}
		sessionTurns[sessionID] = turns
	}

	// 3. For each session, find windows around matched turns and mark context field
	// Use seen-uuid map to deduplicate across overlapping windows
	seenUUIDs := make(map[string]bool)
	var result []interface{}

	for _, sessionID := range sessionOrder {
		turns := sessionTurns[sessionID]
		if len(turns) == 0 {
			continue
		}

		// Build UUID→index map for this session
		uuidToIndex := make(map[string]int, len(turns))
		for i, turn := range turns {
			obj, ok := turn.(map[string]interface{})
			if !ok {
				continue
			}
			uuid, _ := obj["uuid"].(string)
			if uuid != "" {
				uuidToIndex[uuid] = i
			}
		}

		// Collect all window indices for matched turns in this session
		// Process in index order for chronological output
		windowSet := make(map[int]bool)
		for _, entry := range rawData {
			obj, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			entrySessionID, _ := obj["sessionId"].(string)
			if entrySessionID == "" {
				entrySessionID, _ = obj["session_id"].(string)
			}
			if entrySessionID != sessionID {
				continue
			}
			uuid, _ := obj["uuid"].(string)
			idx, exists := uuidToIndex[uuid]
			if !exists {
				continue
			}
			lo := idx - N
			if lo < 0 {
				lo = 0
			}
			hi := idx + N
			if hi >= len(turns) {
				hi = len(turns) - 1
			}
			for i := lo; i <= hi; i++ {
				windowSet[i] = true
			}
		}

		// Emit turns in index order, skipping duplicates
		for i := 0; i < len(turns); i++ {
			if !windowSet[i] {
				continue
			}
			turnObj, ok := turns[i].(map[string]interface{})
			if !ok {
				continue
			}
			uuid, _ := turnObj["uuid"].(string)
			if uuid != "" && seenUUIDs[uuid] {
				continue
			}
			if uuid != "" {
				seenUUIDs[uuid] = true
			}

			// Copy the object and add the "context" field
			newObj := make(map[string]interface{}, len(turnObj)+1)
			for k, v := range turnObj {
				newObj[k] = v
			}
			if matchedUUIDs[uuid] {
				newObj["context"] = false
			} else {
				newObj["context"] = true
			}
			result = append(result, newObj)
		}
	}

	return result, nil
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

// validateArgKeys checks that all keys in args are declared in the tool schema.
// Returns an error listing unknown keys and the valid options.
func validateArgKeys(args map[string]interface{}, schema ToolSchema) error {
	if len(args) == 0 {
		return nil
	}

	var unknown []string
	for key := range args {
		if _, ok := schema.Properties[key]; !ok {
			unknown = append(unknown, key)
		}
	}

	if len(unknown) == 0 {
		return nil
	}

	// Sort for deterministic error messages
	sort.Strings(unknown)

	var valid []string
	for key := range schema.Properties {
		valid = append(valid, key)
	}
	sort.Strings(valid)

	return fmt.Errorf("unknown parameter(s): %s; valid parameters are: %s",
		strings.Join(unknown, ", "),
		strings.Join(valid, ", "))
}

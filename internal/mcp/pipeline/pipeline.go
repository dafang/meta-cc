// Package pipeline provides response-building helpers for the MCP server's tool executor.
// These helpers were extracted from cmd/mcp-server/executor.go to separate response
// construction logic from the orchestration layer.
package pipeline

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yaleh/meta-cc/internal/config"
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
	querypkg "github.com/yaleh/meta-cc/internal/query"
)

// TimestampStatsTools is the set of tool names that should use GenerateTimestampStats
// instead of GenerateStats when producing stats_only or stats_first output.
// These tools return records that lack a tool/ToolName field but have timestamp data,
// so time-bucketed stats are more meaningful than the meaningless "unknown" key.
var TimestampStatsTools = map[string]bool{
	"query_user_messages":     true,
	"query_conversation_flow": true,
	"query_timestamps":        true,
	"query_summaries":         true,
}

// InjectWarnings adds a "warnings" field to a JSON response string.
// If the output is valid JSON object, it adds the field. Otherwise returns as-is.
func InjectWarnings(output string, warnings []string) (string, error) {
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

// DataToJSONL converts array of interfaces to JSONL string.
func DataToJSONL(data []interface{}) (string, error) {
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

// BuildStatsOnlyResponse generates a stats-only response for the given data.
// statsLevel may be "turn" (default) or "session".
func BuildStatsOnlyResponse(parsedData []interface{}, toolName string, statsLevel string) (string, error) {
	jsonlData, err := DataToJSONL(parsedData)
	if err != nil {
		slog.Error("DataToJSONL conversion failed (stats_only)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	var output string
	if statsLevel == "session" && toolName == "query_user_messages" {
		output, err = querypkg.GenerateSessionStats(jsonlData)
	} else if TimestampStatsTools[toolName] {
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

// BuildStatsFirstResponse generates a stats-first response: stats header followed by
// serialized detail data.
func BuildStatsFirstResponse(
	cfg *config.Config,
	rawData []interface{},
	parsedData []interface{},
	args map[string]interface{},
	toolName string,
	statsLevel string,
) (string, error) {
	// Use rawData for stats (sessionId field preserved, not renamed by content_summary)
	jsonlData, err := DataToJSONL(rawData)
	if err != nil {
		slog.Error("DataToJSONL conversion failed (stats_first)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "parse_error",
		)
		return "", err
	}

	var stats string
	if statsLevel == "session" && toolName == "query_user_messages" {
		stats, _ = querypkg.GenerateSessionStats(jsonlData)
	} else if TimestampStatsTools[toolName] {
		stats, _ = querypkg.GenerateTimestampStats(jsonlData)
	} else {
		stats, _ = querypkg.GenerateStats(jsonlData)
	}

	// Use parsedData for detail rendering (may have content_summary applied)
	response, err := responsepkg.AdaptResponse(cfg, parsedData, args, toolName)
	if err != nil {
		slog.Error("response adaptation failed (stats_first)",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "execution_error",
		)
		return "", err
	}

	serialized, err := responsepkg.SerializeResponse(response)
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

// BuildStandardResponse generates a standard (non-stats) response for the given data.
func BuildStandardResponse(
	cfg *config.Config,
	parsedData []interface{},
	args map[string]interface{},
	toolName string,
) (string, error) {
	response, err := responsepkg.AdaptResponse(cfg, parsedData, args, toolName)
	if err != nil {
		slog.Error("response adaptation failed",
			"tool_name", toolName,
			"error", err.Error(),
			"error_type", "execution_error",
		)
		return "", fmt.Errorf("response adaptation error for tool %s: %w", toolName, err)
	}

	output, err := responsepkg.SerializeResponse(response)
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

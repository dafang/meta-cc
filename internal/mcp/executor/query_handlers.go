package executor

import (
	"context"
	"encoding/json"
	"fmt"

	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
)

func init() {
	registerHandler("cleanup_temp_files", handleCleanupTempFiles)
	registerHandler("get_session_directory", handleGetSessionDirectory)
	registerHandler("inspect_session_files", handleInspectSessionFiles)
	registerHandler("execute_stage2_query", handleExecuteStage2Query)
	registerHandler("get_session_metadata", handleGetSessionMetadata)
}

func handleCleanupTempFiles(_ context.Context, _ *ToolExecutor, params map[string]interface{}) (string, error) {
	return responsepkg.ExecuteCleanupTool(params)
}

func handleGetSessionDirectory(ctx context.Context, _ *ToolExecutor, params map[string]interface{}) (string, error) {
	result, err := mcquery.HandleGetSessionDirectory(ctx, params)
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(jsonData), nil
}

func handleInspectSessionFiles(ctx context.Context, _ *ToolExecutor, params map[string]interface{}) (string, error) {
	result, err := mcquery.HandleInspectSessionFiles(ctx, params)
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(jsonData), nil
}

func handleExecuteStage2Query(ctx context.Context, _ *ToolExecutor, params map[string]interface{}) (string, error) {
	result, err := mcquery.HandleExecuteStage2Query(ctx, params)
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(jsonData), nil
}

func handleGetSessionMetadata(ctx context.Context, _ *ToolExecutor, params map[string]interface{}) (string, error) {
	result, err := mcquery.HandleGetSessionMetadata(ctx, params)
	if err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(jsonData), nil
}

package main

import (
	"context"
	"fmt"

	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
)

// parsedTimeRange and parseTimeRange are defined in query_executor.go

// handleQuery and handleQueryRaw deleted in Phase 27 Stage 27.1
// These tools were removed to simplify the query interface
// Users should use the 10 shortcut query tools instead

// executeQuery is an internal helper for convenience tools
// It executes a jq query and returns results as QueryResult
// This allows proper JSONL formatting by response adapters
// workingDir specifies the project directory for session lookup;
// empty string ("") means use os.Getwd() as fallback (backward compatible).
func (e *ToolExecutor) executeQuery(scope string, jqFilter string, limit int, workingDir string) (QueryResult, error) {
	return e.executeQueryWithTimeRange(scope, jqFilter, limit, workingDir, parsedTimeRange{})
}

// executeQueryWithTimeRange is like executeQuery but applies time-range filtering before jq execution.
// tr.Since and tr.Until are optional (nil = no bound).
func (e *ToolExecutor) executeQueryWithTimeRange(scope string, jqFilter string, limit int, workingDir string, tr parsedTimeRange) (QueryResult, error) {
	baseDir, err := getQueryBaseDir(scope, workingDir)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get base directory: %w", err)
	}

	executor := NewQueryExecutor(baseDir)

	code, err := executor.compileExpression(jqFilter)
	if err != nil {
		return QueryResult{}, fmt.Errorf("invalid jq expression: %w", err)
	}

	files, err := getJSONLFiles(baseDir)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to list JSONL files: %w", err)
	}

	if len(files) == 0 {
		return QueryResult{}, fmt.Errorf("no JSONL files found in %s", baseDir)
	}

	ctx := context.Background()
	result := executor.streamFilesWithTimeRange(ctx, files, code, limit, tr)
	return result, nil
}

// getQueryBaseDir delegates to query package
func getQueryBaseDir(scope, workingDir string) (string, error) {
	return querypkg.GetQueryBaseDir(scope, workingDir)
}

// getJSONLFiles delegates to query package
func getJSONLFiles(dir string) ([]string, error) {
	return querypkg.GetJSONLFiles(dir)
}

// loadTurnsForSession delegates to query package
func loadTurnsForSession(baseDir, sessionID string) ([]interface{}, error) {
	return querypkg.LoadTurnsForSession(baseDir, sessionID)
}

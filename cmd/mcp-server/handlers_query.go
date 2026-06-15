package main

import (
	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
)

// handleQuery and handleQueryRaw deleted in Phase 27 Stage 27.1
// These tools were removed to simplify the query interface
// Users should use the 10 shortcut query tools instead

// executeQuery is an internal helper for convenience tools.
// Delegates to internal/mcp/executor.ToolExecutor.ExecuteQuery.
func (e *ToolExecutor) executeQuery(scope string, jqFilter string, limit int, workingDir string) (querypkg.QueryResult, error) {
	return e.ToolExecutor.ExecuteQuery(scope, jqFilter, limit, workingDir)
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

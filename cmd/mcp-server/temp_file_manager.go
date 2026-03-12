package main

import (
	responsepkg "github.com/yaleh/meta-cc/internal/mcp/response"
)

// getSessionCacheDir returns the session-scoped cache directory
func getSessionCacheDir() (string, error) {
	return responsepkg.GetSessionCacheDir()
}

// CleanupSessionCache removes the session cache directory
func CleanupSessionCache() error {
	return responsepkg.CleanupSessionCache()
}

// TempFileManager manages temporary JSONL files with concurrency safety
type TempFileManager = responsepkg.TempFileManager

// createTempFilePath generates a unique temporary file path
func createTempFilePath(sessionHash, queryType string) string {
	return responsepkg.CreateTempFilePath(sessionHash, queryType)
}

// writeJSONLFile writes data to a JSONL file
func writeJSONLFile(path string, data []interface{}) error {
	return responsepkg.WriteJSONLFile(path, data)
}

// cleanupOldFiles removes temporary files older than maxAgeDays
func cleanupOldFiles(maxAgeDays int) ([]string, int64, error) {
	return responsepkg.CleanupOldFiles(maxAgeDays)
}

// executeCleanupTool handles the cleanup_temp_files MCP tool
func executeCleanupTool(args map[string]interface{}) (string, error) {
	return responsepkg.ExecuteCleanupTool(args)
}

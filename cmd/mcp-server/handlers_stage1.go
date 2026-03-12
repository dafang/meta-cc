package main

import (
	"context"

	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
	queryfiles "github.com/yaleh/meta-cc/internal/query/files"
)

// handlers_stage1.go implements Stage 1 tools of the two-stage query architecture
// Stage 1: Metadata and directory inspection tools for query planning

// directoryMetadata holds metadata about a session directory
type directoryMetadata = querypkg.DirectoryMetadata

// handleGetSessionDirectory implements get_session_directory tool
func handleGetSessionDirectory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return querypkg.HandleGetSessionDirectory(ctx, args)
}

// getDirectoryForScope returns the directory path for the given scope
func getDirectoryForScope(scope string) (string, error) {
	return querypkg.GetDirectoryForScope(scope)
}

// collectDirectoryMetadata scans a directory and collects metadata about .jsonl files
func collectDirectoryMetadata(directory string) (*directoryMetadata, error) {
	return querypkg.CollectDirectoryMetadata(directory)
}

// handleInspectSessionFiles implements inspect_session_files tool
func handleInspectSessionFiles(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return querypkg.HandleInspectSessionFiles(ctx, args)
}

// handleGetSessionMetadata implements get_session_metadata tool
func handleGetSessionMetadata(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return querypkg.HandleGetSessionMetadata(ctx, args)
}

// countLines counts the number of lines in a file
func countLines(filename string) (int, error) {
	return querypkg.CountLines(filename)
}

// Ensure queryfiles is used (imported for InspectFiles which is called via HandleInspectSessionFiles)
var _ = queryfiles.InspectFiles

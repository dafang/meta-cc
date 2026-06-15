package main

import (
	"context"

	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
)

// handlers_stage1.go implements Stage 1 tools of the two-stage query architecture
// Stage 1: Metadata and directory inspection tools for query planning

// handleGetSessionDirectory implements get_session_directory tool
func handleGetSessionDirectory(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return querypkg.HandleGetSessionDirectory(ctx, args)
}

// countLines counts the number of lines in a file
func countLines(filename string) (int, error) {
	return querypkg.CountLines(filename)
}

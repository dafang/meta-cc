package main

import (
	"context"

	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
)

// handlers_stage2.go implements Stage 2 tools of the two-stage query architecture
// Stage 2: Actual query execution on selected files with filtering, sorting, transformation, and limits

// handleExecuteStage2Query implements execute_stage2_query tool
func handleExecuteStage2Query(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return querypkg.HandleExecuteStage2Query(ctx, args)
}

package main

import "context"

// RunQuery executes a jq query against the given JSONL files.
func (e *QueryExecutor) RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error) {
	return e.QueryExecutor.RunQuery(ctx, files, filter, transform, limit)
}

// RunQueryWithTimeRange is like RunQuery but applies time range filtering before jq execution.
func (e *QueryExecutor) RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr parsedTimeRange) (QueryResult, error) {
	return e.QueryExecutor.RunQueryWithTimeRange(ctx, files, filter, transform, limit, tr)
}

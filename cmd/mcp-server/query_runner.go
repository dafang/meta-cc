package main

import "context"

// JQRunner executes jq queries against JSONL session files.
// It is implemented by QueryExecutor and can be mocked in tests.
type JQRunner interface {
	RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error)
	RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr parsedTimeRange) (QueryResult, error)
}

// Ensure QueryExecutor implements JQRunner at compile time.
var _ JQRunner = (*QueryExecutor)(nil)

// RunQuery executes a jq query against the given JSONL files.
// filter is a jq filter expression; transform is an optional pipe-appended transform.
// limit 0 means no limit.
func (e *QueryExecutor) RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error) {
	expr := e.buildExpression(filter, transform)
	code, err := e.compileExpression(expr)
	if err != nil {
		return QueryResult{}, err
	}
	return e.streamFiles(ctx, files, code, limit), nil
}

// RunQueryWithTimeRange is like RunQuery but applies time range filtering before jq execution.
// tr.Since and tr.Until are optional (nil = no bound).
func (e *QueryExecutor) RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr parsedTimeRange) (QueryResult, error) {
	expr := e.buildExpression(filter, transform)
	code, err := e.compileExpression(expr)
	if err != nil {
		return QueryResult{}, err
	}
	return e.streamFilesWithTimeRange(ctx, files, code, limit, tr), nil
}

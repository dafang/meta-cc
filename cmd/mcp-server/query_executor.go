package main

import (
	"context"

	"github.com/itchyny/gojq"

	querypkg "github.com/yaleh/meta-cc/internal/mcp/query"
)

// Type aliases for query package types
type ExpressionCache = querypkg.ExpressionCache
type QueryRequest = querypkg.QueryRequest
type QueryResponse = querypkg.QueryResponse
type QueryResult = querypkg.QueryResult

// parsedTimeRange is a local alias for querypkg.ParsedTimeRange
// (lowercase to maintain backward compatibility with existing test code)
type parsedTimeRange = querypkg.ParsedTimeRange

// QueryExecutor wraps querypkg.QueryExecutor providing lowercase method access
// for backward compatibility with existing tests.
type QueryExecutor struct {
	*querypkg.QueryExecutor
	// cache is a lowercase alias for the embedded Cache field, for test compat
	cache *querypkg.ExpressionCache
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(baseDir string) *QueryExecutor {
	inner := querypkg.NewQueryExecutor(baseDir)
	return &QueryExecutor{
		QueryExecutor: inner,
		cache:         inner.Cache,
	}
}

// parseTimeRange parses since/until strings (RFC3339) into a parsedTimeRange.
func parseTimeRange(sinceStr, untilStr string) (parsedTimeRange, error) {
	return querypkg.ParseTimeRange(sinceStr, untilStr)
}

// Lowercase method wrappers for test backward compatibility

func (e *QueryExecutor) buildExpression(filter, transform string) string {
	return e.QueryExecutor.BuildExpression(filter, transform)
}

func (e *QueryExecutor) compileExpression(expr string) (*gojq.Code, error) {
	return e.QueryExecutor.CompileExpression(expr)
}

func (e *QueryExecutor) streamFiles(ctx context.Context, files []string, code *gojq.Code, limit int) QueryResult {
	return e.QueryExecutor.StreamFiles(ctx, files, code, limit)
}

func (e *QueryExecutor) streamFilesWithTimeRange(ctx context.Context, files []string, code *gojq.Code, limit int, tr parsedTimeRange) QueryResult {
	return e.QueryExecutor.StreamFilesWithTimeRange(ctx, files, code, limit, tr)
}

func (e *QueryExecutor) processFile(ctx context.Context, filepath string, code *gojq.Code) ([]interface{}, error) {
	return e.QueryExecutor.ProcessFile(ctx, filepath, code)
}

func (e *QueryExecutor) processFileWithTimeRange(ctx context.Context, filepath string, code *gojq.Code, tr parsedTimeRange) ([]interface{}, error) {
	return e.QueryExecutor.ProcessFileWithTimeRange(ctx, filepath, code, tr)
}

// JQRunner interface for this package (uses parsedTimeRange)
type JQRunner interface {
	RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error)
	RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr parsedTimeRange) (QueryResult, error)
}

// Ensure QueryExecutor implements JQRunner at compile time.
var _ JQRunner = (*QueryExecutor)(nil)

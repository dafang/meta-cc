package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/itchyny/gojq"

	"github.com/yaleh/meta-cc/internal/types"
)

// QueryExecutor executes jq queries on JSONL session data with expression caching
type QueryExecutor struct {
	baseDir string
	cache   *ExpressionCache
}

// ExpressionCache provides LRU caching for compiled jq expressions
type ExpressionCache struct {
	mu      sync.RWMutex
	entries map[string]interface{} // stores *gojq.Code
	keys    []string               // LRU tracking
	maxSize int
}

// QueryRequest represents a query request
type QueryRequest struct {
	JQFilter    string
	JQTransform string
	Scope       string
	Limit       int
	SortBy      string
}

// QueryResponse represents query results
type QueryResponse struct {
	Entries []interface{}
}

// QueryResult represents query results with optional warnings about skipped files
type QueryResult struct {
	Entries  []interface{}
	Warnings []string
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(baseDir string) *QueryExecutor {
	return &QueryExecutor{
		baseDir: baseDir,
		cache: &ExpressionCache{
			entries: make(map[string]interface{}),
			keys:    []string{},
			maxSize: 100,
		},
	}
}

// buildExpression combines filter and transform into a single jq expression
func (e *QueryExecutor) buildExpression(filter, transform string) string {
	// Default to identity filter
	if filter == "" {
		filter = "."
	}

	// If transform is provided, pipe it
	if transform != "" {
		return fmt.Sprintf("%s | %s", filter, transform)
	}

	return filter
}

// compileExpression compiles a jq expression with caching
func (e *QueryExecutor) compileExpression(expr string) (*gojq.Code, error) {
	// Normalize empty expression to identity
	if expr == "" {
		expr = "."
	}

	// Check cache first
	if cached := e.cache.Get(expr); cached != nil {
		if code, ok := cached.(*gojq.Code); ok {
			return code, nil
		}
	}

	// Parse and compile expression
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression '%s': %w", expr, err)
	}

	// Compile to bytecode
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("failed to compile jq expression '%s': %w", expr, err)
	}

	// Cache the compiled code
	e.cache.Put(expr, code)

	return code, nil
}

// streamFiles processes multiple JSONL files with streaming
func (e *QueryExecutor) streamFiles(ctx context.Context, files []string, code *gojq.Code, limit int) QueryResult {
	var result QueryResult

	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result
		default:
		}

		fileResults, err := e.processFile(ctx, file, code)
		if err != nil {
			// Log warning and continue processing other files
			slog.Warn("skipping file due to read error", "file", file, "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped %s: %s", file, err.Error()))
			continue
		}

		result.Entries = append(result.Entries, fileResults...)

		// Check limit
		if limit > 0 && len(result.Entries) >= limit {
			result.Entries = result.Entries[:limit]
			return result
		}
	}

	return result
}

// streamFilesWithTimeRange is like streamFiles but applies TimeRange filtering before jq execution.
func (e *QueryExecutor) streamFilesWithTimeRange(ctx context.Context, files []string, code *gojq.Code, limit int, tr TimeRange) QueryResult {
	var result QueryResult

	for _, file := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result
		default:
		}

		fileResults, err := e.processFileWithTimeRange(ctx, file, code, tr)
		if err != nil {
			// Log warning and continue processing other files
			slog.Warn("skipping file due to read error", "file", file, "error", err)
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped %s: %s", file, err.Error()))
			continue
		}

		result.Entries = append(result.Entries, fileResults...)

		// Check limit
		if limit > 0 && len(result.Entries) >= limit {
			result.Entries = result.Entries[:limit]
			return result
		}
	}

	return result
}

// processFileWithTimeRange is like processFile but filters each entry by its timestamp field
// before running the jq expression. Entries with unparseable or missing timestamps are included.
func (e *QueryExecutor) processFileWithTimeRange(ctx context.Context, filepath string, code *gojq.Code, tr TimeRange) ([]interface{}, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	var results []interface{}
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	buf := make([]byte, types.MaxScannerLineBytes)
	scanner.Buffer(buf, types.MaxScannerLineBytes)

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return results, nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON line to map for timestamp inspection
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip invalid JSON lines (don't fail entire file)
			continue
		}

		// Apply time range filter if bounds are set
		if tr.Since != nil || tr.Until != nil {
			if ts, ok := entry["timestamp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					if tr.Since != nil && t.Before(*tr.Since) {
						continue
					}
					if tr.Until != nil && !t.Before(*tr.Until) {
						continue
					}
				}
				// unparseable timestamp: include the entry (non-fatal)
			}
			// missing timestamp field: include the entry
		}

		// Execute jq query on this entry
		iter := code.Run(entry)
		for {
			value, ok := iter.Next()
			if !ok {
				break
			}

			// Check for errors
			if err, ok := value.(error); ok {
				// Skip entries that cause jq errors
				_ = err
				continue
			}

			results = append(results, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return results, fmt.Errorf("error reading file %s: %w", filepath, err)
	}

	return results, nil
}

// processFile processes a single JSONL file
func (e *QueryExecutor) processFile(ctx context.Context, filepath string, code *gojq.Code) ([]interface{}, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filepath, err)
	}
	defer file.Close()

	var results []interface{}
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	buf := make([]byte, types.MaxScannerLineBytes)
	scanner.Buffer(buf, types.MaxScannerLineBytes)

	lineNum := 0
	for scanner.Scan() {
		lineNum++

		// Check context cancellation
		select {
		case <-ctx.Done():
			return results, nil
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse JSON line
		var entry interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip invalid JSON lines (don't fail entire file)
			continue
		}

		// Execute jq query on this entry
		iter := code.Run(entry)
		for {
			value, ok := iter.Next()
			if !ok {
				break
			}

			// Check for errors
			if err, ok := value.(error); ok {
				// Skip entries that cause jq errors
				_ = err
				continue
			}

			results = append(results, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return results, fmt.Errorf("error reading file %s: %w", filepath, err)
	}

	return results, nil
}

// Get retrieves a cached expression
func (c *ExpressionCache) Get(expr string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.entries[expr]
}

// Put stores a compiled expression in cache with LRU eviction
func (c *ExpressionCache) Put(expr string, code interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if entry already exists
	if _, exists := c.entries[expr]; exists {
		// Update existing entry (move to end for LRU)
		c.removeKey(expr)
		c.keys = append(c.keys, expr)
		c.entries[expr] = code
		return
	}

	// LRU eviction if cache is full
	if len(c.entries) >= c.maxSize {
		oldest := c.keys[0]
		delete(c.entries, oldest)
		c.keys = c.keys[1:]
	}

	// Add new entry
	c.entries[expr] = code
	c.keys = append(c.keys, expr)
}

// removeKey removes a key from the keys slice (helper for LRU)
func (c *ExpressionCache) removeKey(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			return
		}
	}
}

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestCompileExpression tests jq expression compilation
func TestCompileExpression(t *testing.T) {
	executor := NewQueryExecutor("")

	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "simple filter",
			expr:    ".",
			wantErr: false,
		},
		{
			name:    "array iterator",
			expr:    ".[]",
			wantErr: false,
		},
		{
			name:    "select filter",
			expr:    "select(.type == \"user\")",
			wantErr: false,
		},
		{
			name:    "pipe expression",
			expr:    ". | select(.type == \"assistant\")",
			wantErr: false,
		},
		{
			name:    "object construction",
			expr:    "{timestamp, type}",
			wantErr: false,
		},
		{
			name:    "invalid expression",
			expr:    "select(",
			wantErr: true,
		},
		{
			name:    "empty expression defaults to identity",
			expr:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := executor.compileExpression(tt.expr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for expression %q, got nil", tt.expr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for expression %q: %v", tt.expr, err)
				}
				if code == nil {
					t.Errorf("expected non-nil code for expression %q", tt.expr)
				}
			}
		})
	}
}

// TestExpressionCache tests LRU cache behavior
func TestExpressionCache(t *testing.T) {
	cache := &ExpressionCache{
		entries: make(map[string]interface{}),
		keys:    []string{},
		maxSize: 3,
	}

	// Test cache miss
	if code := cache.Get("expr1"); code != nil {
		t.Error("expected cache miss, got hit")
	}

	// Test cache put and get
	cache.Put("expr1", "code1")
	if code := cache.Get("expr1"); code != "code1" {
		t.Errorf("expected 'code1', got %v", code)
	}

	// Fill cache to max
	cache.Put("expr2", "code2")
	cache.Put("expr3", "code3")

	if len(cache.entries) != 3 {
		t.Errorf("expected cache size 3, got %d", len(cache.entries))
	}

	// Test LRU eviction
	cache.Put("expr4", "code4")
	if len(cache.entries) != 3 {
		t.Errorf("expected cache size 3 after eviction, got %d", len(cache.entries))
	}

	// First entry should be evicted
	if code := cache.Get("expr1"); code != nil {
		t.Error("expected expr1 to be evicted")
	}

	// Newer entries should still exist
	if code := cache.Get("expr2"); code != "code2" {
		t.Errorf("expected 'code2', got %v", code)
	}
	if code := cache.Get("expr3"); code != "code3" {
		t.Errorf("expected 'code3', got %v", code)
	}
	if code := cache.Get("expr4"); code != "code4" {
		t.Errorf("expected 'code4', got %v", code)
	}

	// Test updating existing entry (should move to end)
	cache.Put("expr2", "code2-updated")
	if code := cache.Get("expr2"); code != "code2-updated" {
		t.Errorf("expected 'code2-updated', got %v", code)
	}

	// Add one more - expr3 should be evicted (oldest after expr2 was moved)
	cache.Put("expr5", "code5")
	if code := cache.Get("expr3"); code != nil {
		t.Error("expected expr3 to be evicted after LRU reordering")
	}
	if code := cache.Get("expr2"); code != "code2-updated" {
		t.Error("expected expr2 to still exist after update")
	}
}

// TestCacheHitRate tests cache effectiveness
func TestCacheHitRate(t *testing.T) {
	executor := NewQueryExecutor("")

	expressions := []string{
		"select(.type == \"user\")",
		"select(.type == \"assistant\")",
		"select(.type == \"user\")",      // repeat
		"select(.type == \"assistant\")", // repeat
		"select(.type == \"user\")",      // repeat
	}

	hits := 0
	misses := 0

	for _, expr := range expressions {
		if executor.cache.Get(expr) != nil {
			hits++
		} else {
			misses++
		}
		executor.compileExpression(expr) // compile and cache
	}

	// First 2 are misses, next 3 are hits
	if hits != 3 {
		t.Errorf("expected 3 cache hits, got %d", hits)
	}
	if misses != 2 {
		t.Errorf("expected 2 cache misses, got %d", misses)
	}

	// Hit rate should be 60% (3/5)
	hitRate := float64(hits) / float64(len(expressions))
	if hitRate < 0.6 {
		t.Errorf("expected hit rate >= 60%%, got %.2f%%", hitRate*100)
	}
}

// TestBuildExpression tests expression building logic
func TestBuildExpression(t *testing.T) {
	executor := NewQueryExecutor("")

	tests := []struct {
		name      string
		filter    string
		transform string
		want      string
	}{
		{
			name:      "empty filter defaults to identity",
			filter:    "",
			transform: "",
			want:      ".",
		},
		{
			name:      "filter only",
			filter:    "select(.type == \"user\")",
			transform: "",
			want:      "select(.type == \"user\")",
		},
		{
			name:      "filter and transform",
			filter:    "select(.type == \"user\")",
			transform: "{timestamp, type}",
			want:      "select(.type == \"user\") | {timestamp, type}",
		},
		{
			name:      "identity filter with transform",
			filter:    ".",
			transform: "{timestamp}",
			want:      ". | {timestamp}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := executor.buildExpression(tt.filter, tt.transform)
			if got != tt.want {
				t.Errorf("buildExpression(%q, %q) = %q, want %q",
					tt.filter, tt.transform, got, tt.want)
			}
		})
	}
}

// TestStreamFiles tests JSONL streaming and filtering
func TestStreamFiles(t *testing.T) {
	// Create temp directory with test JSONL files
	tmpDir := t.TempDir()

	// Create test data
	entries := []map[string]interface{}{
		{"type": "user", "timestamp": "2025-01-01T10:00:00Z", "content": "hello"},
		{"type": "assistant", "timestamp": "2025-01-01T10:01:00Z", "content": "hi"},
		{"type": "user", "timestamp": "2025-01-01T10:02:00Z", "content": "bye"},
	}

	// Write to JSONL file
	file := filepath.Join(tmpDir, "session.jsonl")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	// Test: filter user messages only
	code, err := executor.compileExpression("select(.type == \"user\")")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	qr := executor.streamFiles(ctx, []string{file}, code, 0)

	// Should have 2 user messages
	if len(qr.Entries) != 2 {
		t.Errorf("expected 2 results, got %d", len(qr.Entries))
	}

	for _, result := range qr.Entries {
		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", result)
		}
		if resultMap["type"] != "user" {
			t.Errorf("expected type='user', got %v", resultMap["type"])
		}
	}
}

// TestStreamFilesWithLimit tests limit parameter
func TestStreamFilesWithLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 10 entries
	var entries []map[string]interface{}
	for i := 0; i < 10; i++ {
		entries = append(entries, map[string]interface{}{
			"type": "user",
			"id":   i,
		})
	}

	file := filepath.Join(tmpDir, "session.jsonl")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for _, entry := range entries {
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Test with limit=5
	qr := executor.streamFiles(ctx, []string{file}, code, 5)
	if len(qr.Entries) != 5 {
		t.Errorf("expected 5 results with limit=5, got %d", len(qr.Entries))
	}

	// Test with limit=0 (no limit)
	qr = executor.streamFiles(ctx, []string{file}, code, 0)
	if len(qr.Entries) != 10 {
		t.Errorf("expected 10 results with limit=0, got %d", len(qr.Entries))
	}
}

// TestProcessFile tests individual file processing
func TestProcessFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with valid and invalid JSON
	file := filepath.Join(tmpDir, "mixed.jsonl")
	content := `{"type": "user", "id": 1}
{"type": "assistant", "id": 2}
invalid json line
{"type": "user", "id": 3}
`
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression("select(.type == \"user\")")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	results, err := executor.processFile(ctx, file, code)

	// Should process valid lines and skip invalid ones
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have 2 user entries (id=1 and id=3)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

// TestQueryExecutionPerformance tests query performance
func TestQueryExecutionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	tmpDir := t.TempDir()

	// Create 1000 test records
	file := filepath.Join(tmpDir, "large.jsonl")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer f.Close()

	for i := 0; i < 1000; i++ {
		entry := map[string]interface{}{
			"type":      "user",
			"id":        i,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression("select(.id < 100)")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Measure execution time
	start := time.Now()
	qr := executor.streamFiles(ctx, []string{file}, code, 0)
	elapsed := time.Since(start)

	// Should complete in < 100ms for 1000 records
	if elapsed > 100*time.Millisecond {
		t.Errorf("query execution took %v, expected < 100ms", elapsed)
	}

	// Verify results
	if len(qr.Entries) != 100 {
		t.Errorf("expected 100 results, got %d", len(qr.Entries))
	}

	t.Logf("Processed 1000 records in %v", elapsed)
}

// TestProcessFileLargeLines tests that processFile handles lines up to 4 MB
func TestProcessFileLargeLines(t *testing.T) {
	tests := []struct {
		name string
		size int // approximate line size in bytes
	}{
		{"1.5 MB line", 1_500_000},
		{"2 MB line (regression)", 2_000_000},
		{"4 MB line", 4_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			file := filepath.Join(tmpDir, "large-line.jsonl")

			// Build a JSON object with a large "data" field to reach target size.
			// {"data":"AAA..."}  — overhead is ~12 bytes for the JSON wrapper.
			padding := make([]byte, tt.size-12)
			for i := range padding {
				padding[i] = 'A'
			}
			entry := map[string]interface{}{
				"data": string(padding),
			}
			lineBytes, err := json.Marshal(entry)
			if err != nil {
				t.Fatalf("failed to marshal entry: %v", err)
			}
			lineBytes = append(lineBytes, '\n')

			if err := os.WriteFile(file, lineBytes, 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			executor := NewQueryExecutor(tmpDir)
			ctx := context.Background()

			code, err := executor.compileExpression(".")
			if err != nil {
				t.Fatalf("failed to compile expression: %v", err)
			}

			results, err := executor.processFile(ctx, file, code)
			if err != nil {
				t.Fatalf("processFile returned error for %s: %v", tt.name, err)
			}
			if len(results) != 1 {
				t.Errorf("expected 1 result, got %d", len(results))
			}
		})
	}
}

// TestStreamFilesWithWarnings tests that streamFiles returns partial results and warnings
// when some files cannot be read (e.g., unreadable permissions or scanner errors)
func TestStreamFilesWithWarnings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid JSONL file
	validFile := filepath.Join(tmpDir, "valid.jsonl")
	validEntries := []map[string]interface{}{
		{"type": "user", "id": 1},
		{"type": "assistant", "id": 2},
	}
	f, err := os.Create(validFile)
	if err != nil {
		t.Fatalf("failed to create valid file: %v", err)
	}
	for _, entry := range validEntries {
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	// Create a file that cannot be read (no read permissions)
	unreadableFile := filepath.Join(tmpDir, "unreadable.jsonl")
	if err := os.WriteFile(unreadableFile, []byte(`{"type":"user"}`+"\n"), 0000); err != nil {
		t.Fatalf("failed to create unreadable file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Process both files: valid first, then unreadable
	result := executor.streamFiles(ctx, []string{validFile, unreadableFile}, code, 0)

	// Should have entries from the valid file
	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries from valid file, got %d", len(result.Entries))
	}

	// Should have a warning about the unreadable file
	if len(result.Warnings) == 0 {
		t.Error("expected warnings about unreadable file, got none")
	}

	// Verify warning contains the file path
	found := false
	for _, w := range result.Warnings {
		if stringContains(w, "unreadable.jsonl") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning mentioning unreadable file, got: %v", result.Warnings)
	}
}

// TestStreamFilesNoWarnings tests that streamFiles returns empty warnings on success
func TestStreamFilesNoWarnings(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "ok.jsonl")
	if err := os.WriteFile(file, []byte(`{"type":"user","id":1}`+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	result := executor.streamFiles(ctx, []string{file}, code, 0)

	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.Entries))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}

// TestStreamFilesNonexistentFile tests warning for non-existent files
func TestStreamFilesNonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.jsonl")
	if err := os.WriteFile(validFile, []byte(`{"id":1}`+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	nonexistent := filepath.Join(tmpDir, "does-not-exist.jsonl")

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	result := executor.streamFiles(ctx, []string{validFile, nonexistent}, code, 0)

	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry from valid file, got %d", len(result.Entries))
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

// TestProcessFile_LargeImageLine_NoError verifies that processFile handles a
// JSONL file containing a line with a ~5 MB base64 image without returning an
// error. After the migration to ReadLineFiltered the large base64 payload is
// transparently truncated so the entry is still valid JSON and parsed normally.
func TestProcessFile_LargeImageLine_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "image-line.jsonl")

	// Build a ~5 MB base64 payload embedded in a JSON line.
	largePayload := make([]byte, 5_000_000)
	for i := range largePayload {
		largePayload[i] = 'A'
	}
	// Construct a line that looks like a tool-result with an inline image.
	// Use a JSON object that is valid before the truncation happens.
	line := `{"type":"tool_result","id":"tr1","content":"data:image/png;base64,` +
		string(largePayload) + `"}`
	if err := os.WriteFile(file, []byte(line+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	results, err := executor.processFile(ctx, file, code)
	if err != nil {
		t.Fatalf("processFile returned error for large image line: %v", err)
	}
	// The entry is included (possibly with truncated image data).
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// TestProcessFileWithTimeRange_LargeImageLine_NoError verifies that
// processFileWithTimeRange handles a ~5 MB image line without error.
func TestProcessFileWithTimeRange_LargeImageLine_NoError(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "image-line-tr.jsonl")

	largePayload := make([]byte, 5_000_000)
	for i := range largePayload {
		largePayload[i] = 'B'
	}
	line := `{"type":"tool_result","id":"tr2","timestamp":"2025-01-01T00:00:00Z","content":"data:image/png;base64,` +
		string(largePayload) + `"}`
	if err := os.WriteFile(file, []byte(line+"\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// No time range constraints — all entries should be included.
	tr := parsedTimeRange{}
	results, err := executor.processFileWithTimeRange(ctx, file, code, tr)
	if err != nil {
		t.Fatalf("processFileWithTimeRange returned error for large image line: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

// stringContains is a helper to check if a string contains a substring
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestContextCancellation tests context cancellation during streaming
func TestContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create large file
	file := filepath.Join(tmpDir, "large.jsonl")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	for i := 0; i < 10000; i++ {
		entry := map[string]interface{}{"id": i}
		data, _ := json.Marshal(entry)
		f.Write(data)
		f.WriteString("\n")
	}
	f.Close()

	executor := NewQueryExecutor(tmpDir)
	ctx, cancel := context.WithCancel(context.Background())

	code, err := executor.compileExpression(".")
	if err != nil {
		t.Fatalf("failed to compile expression: %v", err)
	}

	// Cancel context immediately
	cancel()

	qr := executor.streamFiles(ctx, []string{file}, code, 0)

	// Should return early due to cancellation
	if len(qr.Entries) > 100 {
		t.Errorf("expected few results due to cancellation, got %d", len(qr.Entries))
	}
}

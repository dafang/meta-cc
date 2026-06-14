package engine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testUser1 = `{"type":"user","timestamp":"2025-01-15T10:00:00Z","message":{"content":"fix bug"}}`
	testUser2 = `{"type":"user","timestamp":"2025-01-15T11:00:00Z","message":{"content":"add feature"}}`
	testUser3 = `{"type":"user","timestamp":"2025-01-15T12:00:00Z","message":{"content":"refactor code"}}`
	testAsst1 = `{"type":"assistant","timestamp":"2025-01-15T10:30:00Z","message":{"content":"fixing..."}}`
	testAsst2 = `{"type":"assistant","timestamp":"2025-01-15T11:30:00Z","message":{"content":"adding..."}}`
)

func TestExecuteStage2Query_BasicFilter(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test1.jsonl")

	testData := testUser1 + "\n" + testAsst1 + "\n" + testUser2 + "\n" + testAsst2 + "\n" + testUser3 + "\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result.Results))
	}
	for i, res := range result.Results {
		resMap := res.(map[string]interface{})
		if resMap["type"] != "user" {
			t.Errorf("Result %d has type %v, expected user", i, resMap["type"])
		}
	}
	if result.Metadata.FilesProcessed != 1 {
		t.Errorf("Expected 1 file processed, got %d", result.Metadata.FilesProcessed)
	}
	if result.Metadata.ResultsReturned != 3 {
		t.Errorf("Expected 3 results returned, got %d", result.Metadata.ResultsReturned)
	}
	if result.Metadata.TotalRecordsScanned != 5 {
		t.Errorf("Expected 5 records scanned, got %d", result.Metadata.TotalRecordsScanned)
	}
}

func TestExecuteStage2Query_FilterAndSort(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test2.jsonl")

	testData := testUser2 + "\n" + testUser1 + "\n" + testUser3 + "\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
		Sort:   "sort_by(.timestamp)",
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result.Results))
	}

	timestamps := []string{
		"2025-01-15T10:00:00Z",
		"2025-01-15T11:00:00Z",
		"2025-01-15T12:00:00Z",
	}
	for i, res := range result.Results {
		resMap := res.(map[string]interface{})
		if resMap["timestamp"] != timestamps[i] {
			t.Errorf("Result %d timestamp %v, expected %s", i, resMap["timestamp"], timestamps[i])
		}
	}
}

func TestExecuteStage2Query_FilterSortTransform(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test3.jsonl")

	testData := testUser1 + "\n" + testUser2 + "\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:     []string{testFile},
		Filter:    `select(.type == "user")`,
		Sort:      "sort_by(.timestamp)",
		Transform: "{type, timestamp}",
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result.Results))
	}
	for i, res := range result.Results {
		resMap := res.(map[string]interface{})
		if len(resMap) != 2 {
			t.Errorf("Result %d has %d fields, expected 2", i, len(resMap))
		}
		if _, ok := resMap["type"]; !ok {
			t.Errorf("Result %d missing type field", i)
		}
		if _, ok := resMap["timestamp"]; !ok {
			t.Errorf("Result %d missing timestamp field", i)
		}
		if _, ok := resMap["message"]; ok {
			t.Errorf("Result %d has message field (should be excluded)", i)
		}
	}
}

func TestExecuteStage2Query_FilterAndLimit(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test4.jsonl")

	testData := testUser1 + "\n" + testUser2 + "\n" + testUser3 + "\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
		Limit:  2,
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("Expected 2 results (limited), got %d", len(result.Results))
	}
	if !result.Metadata.Truncated {
		t.Error("Expected truncated=true when limit is reached")
	}
	if result.Metadata.ResultsReturned != 2 {
		t.Errorf("Expected results_returned=2, got %d", result.Metadata.ResultsReturned)
	}
}

func TestExecuteStage2Query_EmptyResultSet(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test5.jsonl")

	testData := testAsst1 + "\n" + testAsst2 + "\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result.Results))
	}
	if result.Metadata.Truncated {
		t.Error("Expected truncated=false for empty result set")
	}
}

func TestExecuteStage2Query_InvalidJQExpression(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test6.jsonl")

	if err := os.WriteFile(testFile, []byte(testUser1+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(invalid syntax here)`,
	}

	_, err := ExecuteStage2Query(query)
	if err == nil {
		t.Error("Expected error for invalid jq expression, got nil")
	}
}

func TestExecuteStage2Query_NonExistentFile(t *testing.T) {
	query := &Stage2Query{
		Files:  []string{"/nonexistent/file.jsonl"},
		Filter: `select(.type == "user")`,
	}

	_, err := ExecuteStage2Query(query)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestExecuteStage2Query_MultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	testFile1 := filepath.Join(tempDir, "test_a.jsonl")
	testFile2 := filepath.Join(tempDir, "test_b.jsonl")

	if err := os.WriteFile(testFile1, []byte(testUser1+"\n"+testUser2+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}
	if err := os.WriteFile(testFile2, []byte(testUser3+"\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile1, testFile2},
		Filter: `select(.type == "user")`,
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 3 {
		t.Errorf("Expected 3 results from 2 files, got %d", len(result.Results))
	}
	if result.Metadata.FilesProcessed != 2 {
		t.Errorf("Expected 2 files processed, got %d", result.Metadata.FilesProcessed)
	}
}

func TestBuildJQExpression(t *testing.T) {
	tests := []struct {
		name      string
		filter    string
		sort      string
		transform string
		expected  string
	}{
		{
			name:     "filter only",
			filter:   `select(.type == "user")`,
			expected: `.[] | select(.type == "user")`,
		},
		{
			name:     "filter and sort",
			filter:   `select(.type == "user")`,
			sort:     "sort_by(.timestamp)",
			expected: `[.[] | select(.type == "user")] | sort_by(.timestamp) | .[]`,
		},
		{
			name:      "filter, sort, and transform",
			filter:    `select(.type == "user")`,
			sort:      "sort_by(.timestamp)",
			transform: "{type, timestamp}",
			expected:  `[.[] | select(.type == "user")] | sort_by(.timestamp) | .[] | {type, timestamp}`,
		},
		{
			name:      "filter and transform",
			filter:    `select(.type == "user")`,
			transform: "{type}",
			expected:  `.[] | select(.type == "user") | {type}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJQExpression(tt.filter, tt.sort, tt.transform)
			if result != tt.expected {
				t.Errorf("buildJQExpression() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestReadJSONLFile_LargeImageLine_ReturnsData(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large_image.jsonl")

	largeBase64 := strings.Repeat("A", 5*1024*1024)
	imageLine := `{"type":"user","sessionId":"s1","message":{"content":[{"type":"tool_result","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"` + largeBase64 + `"}}]}]}}`
	normalLine := `{"type":"assistant","sessionId":"s1","content":"hello"}`

	content := imageLine + "\n" + normalLine + "\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	records, err := readJSONLFile(testFile)
	if err != nil {
		t.Fatalf("readJSONLFile returned unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	imageJSON, err := json.Marshal(records[0])
	if err != nil {
		t.Fatalf("failed to re-marshal image record: %v", err)
	}
	if bytes.Contains(imageJSON, []byte(largeBase64)) {
		t.Error("large base64 data should have been stripped")
	}
	if !bytes.Contains(imageJSON, []byte("binary-omitted")) {
		t.Error("expected <binary-omitted> placeholder")
	}

	normalRec := records[1].(map[string]interface{})
	if normalRec["content"] != "hello" {
		t.Errorf("normal record content unexpected: %v", normalRec["content"])
	}
}

func BenchmarkExecuteStage2Query_3MB(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "large.jsonl")

	var testData strings.Builder
	for i := 0; i < 3000; i++ {
		record := map[string]interface{}{
			"type":      "user",
			"timestamp": "2025-01-15T10:00:00Z",
			"message": map[string]interface{}{
				"content": "This is a test message with some content to make it realistic sized",
			},
			"metadata": map[string]interface{}{
				"index": i,
				"tags":  []string{"test", "benchmark", "performance"},
			},
		}
		jsonBytes, _ := json.Marshal(record)
		testData.Write(jsonBytes)
		testData.WriteString("\n")
	}

	if err := os.WriteFile(testFile, []byte(testData.String()), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
		Limit:  100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ExecuteStage2Query(query)
		if err != nil {
			b.Fatalf("ExecuteStage2Query failed: %v", err)
		}
		if result.Metadata.ExecutionTimeMs >= 100 {
			b.Errorf("Execution took %dms, expected < 100ms", result.Metadata.ExecutionTimeMs)
		}
	}
}

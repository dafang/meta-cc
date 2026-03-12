package jq

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadJSONLFile_LargeImageLine_ReturnsData verifies that readJSONLFile in the jq
// package handles a JSONL file containing a 5MB image line gracefully: instead of
// returning an error (old Scanner behaviour), it returns 2 records and the base64
// data is replaced with "<binary-omitted>".
func TestReadJSONLFile_LargeImageLine_ReturnsData(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large_image.jsonl")

	largeBase64 := strings.Repeat("A", 5*1024*1024) // 5 MB base64 data
	// Use actual Claude Code image structure that triggers stripImageData
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

	// Re-marshal the image record and verify the large base64 was replaced
	imageJSON, err := json.Marshal(records[0])
	if err != nil {
		t.Fatalf("failed to re-marshal image record: %v", err)
	}
	if bytes.Contains(imageJSON, []byte(largeBase64)) {
		t.Error("large base64 data should have been stripped from image record")
	}
	if !bytes.Contains(imageJSON, []byte("binary-omitted")) {
		t.Error("expected <binary-omitted> placeholder in image record")
	}

	// The normal record should be intact
	normalRec, ok := records[1].(map[string]interface{})
	if !ok {
		t.Fatalf("record[1] is not a map, got %T", records[1])
	}
	if normalRec["content"] != "hello" {
		t.Errorf("normal record content unexpected: %v", normalRec["content"])
	}
}

// TestExecuteStage2Query_BasicFilter verifies the basic jq package Stage2 query.
func TestExecuteStage2Query_BasicFilter(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.jsonl")

	lines := []string{
		`{"type":"user","timestamp":"2025-01-15T10:00:00Z"}`,
		`{"type":"assistant","timestamp":"2025-01-15T10:30:00Z"}`,
		`{"type":"user","timestamp":"2025-01-15T11:00:00Z"}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	query := &Stage2Query{
		Files:  []string{testFile},
		Filter: `select(.type == "user")`,
	}

	result, err := ExecuteStage2Query(query)
	if err != nil {
		t.Fatalf("ExecuteStage2Query failed: %v", err)
	}

	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}

	if result.Metadata.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.Metadata.FilesProcessed)
	}

	if result.Metadata.TotalRecordsScanned != 3 {
		t.Errorf("expected 3 records scanned, got %d", result.Metadata.TotalRecordsScanned)
	}
}

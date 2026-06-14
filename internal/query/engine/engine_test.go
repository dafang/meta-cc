package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yaleh/meta-cc/internal/query/engine"
)

func TestApplyJQFilter_BasicFilter(t *testing.T) {
	jsonl := `{"type":"user","msg":"hello"}
{"type":"assistant","msg":"world"}`

	// ApplyJQFilter receives the full array as input; callers must include .[]
	got, err := engine.ApplyJQFilter(jsonl, `.[] | select(.type == "user")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty result")
	}
}

func TestApplyJQFilter_EmptyExpression(t *testing.T) {
	jsonl := `{"x":1}
{"x":2}`
	got, err := engine.ApplyJQFilter(jsonl, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty result from default expression")
	}
}

func TestApplyJQFilter_InvalidExpression(t *testing.T) {
	_, err := engine.ApplyJQFilter(`{"x":1}`, "!!invalid!!")
	if err == nil {
		t.Fatal("expected error for invalid jq expression")
	}
}

func TestExecuteStage2Query_BasicQuery(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.jsonl")
	content := `{"type":"user","n":1}
{"type":"assistant","n":2}
{"type":"user","n":3}
`
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	q := &engine.Stage2Query{
		Files:  []string{f},
		Filter: `select(.type == "user")`,
	}
	result, err := engine.ExecuteStage2Query(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Results))
	}
	if result.Metadata.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", result.Metadata.FilesProcessed)
	}
}

func TestExecuteStage2Query_WithLimit(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.jsonl")
	var content string
	for i := 0; i < 10; i++ {
		content += `{"n":1}` + "\n"
	}
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	q := &engine.Stage2Query{
		Files:  []string{f},
		Filter: ".",
		Limit:  3,
	}
	result, err := engine.ExecuteStage2Query(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
	if !result.Metadata.Truncated {
		t.Error("expected Truncated=true")
	}
}

func TestExecuteStage2Query_NoFiles(t *testing.T) {
	q := &engine.Stage2Query{Filter: "."}
	_, err := engine.ExecuteStage2Query(q)
	if err == nil {
		t.Fatal("expected error for empty files")
	}
}

func TestExecuteStage2Query_NoFilter(t *testing.T) {
	q := &engine.Stage2Query{Files: []string{"x.jsonl"}}
	_, err := engine.ExecuteStage2Query(q)
	if err == nil {
		t.Fatal("expected error for empty filter")
	}
}

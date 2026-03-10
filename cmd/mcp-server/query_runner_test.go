package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// mockJQRunner implements JQRunner for testing.
type mockJQRunner struct {
	results QueryResult
	err     error
}

func (m *mockJQRunner) RunQuery(_ context.Context, _ []string, _, _ string, _ int) (QueryResult, error) {
	return m.results, m.err
}

func (m *mockJQRunner) RunQueryWithTimeRange(_ context.Context, _ []string, _, _ string, _ int, _ TimeRange) (QueryResult, error) {
	return m.results, m.err
}

// Verify interface compliance at compile time.
var _ JQRunner = (*QueryExecutor)(nil)
var _ JQRunner = (*mockJQRunner)(nil)

func TestJQRunner_InterfaceCompliance(t *testing.T) {
	// The compile-time checks above are the primary assertion.
	t.Log("QueryExecutor implements JQRunner interface")
}

func TestMockJQRunner(t *testing.T) {
	mock := &mockJQRunner{
		results: QueryResult{Entries: []any{"entry1"}},
	}

	result, err := mock.RunQuery(context.Background(), nil, ".", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(result.Entries))
	}
}

func TestMockJQRunner_WithTimeRange(t *testing.T) {
	mock := &mockJQRunner{
		results: QueryResult{Entries: []any{"entry1", "entry2"}},
	}

	result, err := mock.RunQueryWithTimeRange(context.Background(), nil, ".", "", 0, TimeRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Entries))
	}
}

func TestMockJQRunner_Error(t *testing.T) {
	expectedErr := context.DeadlineExceeded
	mock := &mockJQRunner{err: expectedErr}

	_, err := mock.RunQuery(context.Background(), nil, ".", "", 0)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// TestQueryExecutor_RunQuery verifies RunQuery delegates correctly.
func TestQueryExecutor_RunQuery(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "session.jsonl")
	content := `{"type":"user","id":1}` + "\n" +
		`{"type":"assistant","id":2}` + "\n" +
		`{"type":"user","id":3}` + "\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	result, err := executor.RunQuery(ctx, []string{file}, `select(.type == "user")`, "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result.Entries))
	}
}

// TestQueryExecutor_RunQuery_InvalidFilter verifies error propagation for bad jq.
func TestQueryExecutor_RunQuery_InvalidFilter(t *testing.T) {
	executor := NewQueryExecutor(t.TempDir())
	_, err := executor.RunQuery(context.Background(), nil, "select(", "", 0)
	if err == nil {
		t.Error("expected error for invalid jq filter, got nil")
	}
}

// TestQueryExecutor_RunQueryWithTimeRange verifies time-range filtering.
func TestQueryExecutor_RunQueryWithTimeRange(t *testing.T) {
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "session.jsonl")
	content := `{"type":"user","timestamp":"2025-01-01T10:00:00Z"}` + "\n" +
		`{"type":"user","timestamp":"2025-06-01T10:00:00Z"}` + "\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	executor := NewQueryExecutor(tmpDir)
	ctx := context.Background()

	tr, err := parseTimeRange("2025-03-01T00:00:00Z", "")
	if err != nil {
		t.Fatalf("failed to parse time range: %v", err)
	}

	result, err := executor.RunQueryWithTimeRange(ctx, []string{file}, ".", "", 0, tr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the June entry is after March 2025
	if len(result.Entries) != 1 {
		t.Errorf("expected 1 entry after time filter, got %d", len(result.Entries))
	}
}

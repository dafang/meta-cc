package pipeline_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/config"
	"github.com/yaleh/meta-cc/internal/mcp/pipeline"
)

// testConfig returns a minimal config suitable for pipeline tests.
func testConfig() *config.Config {
	return &config.Config{
		Output: config.OutputConfig{
			Mode:            "auto",
			InlineThreshold: 32768,
		},
	}
}

// helpers

func makeRecord(fields map[string]interface{}) interface{} {
	return fields
}

// ─── InjectWarnings ───────────────────────────────────────────────────────────

func TestInjectWarnings_NoWarnings(t *testing.T) {
	input := `{"mode":"inline","data":[]}`
	out, err := pipeline.InjectWarnings(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	w, ok := parsed["warnings"]
	if !ok {
		t.Fatal("expected 'warnings' field")
	}
	// nil warnings should become empty slice
	arr, ok := w.([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", w)
	}
	if len(arr) != 0 {
		t.Fatalf("expected empty warnings, got %v", arr)
	}
}

func TestInjectWarnings_WithWarnings(t *testing.T) {
	input := `{"mode":"inline","data":[]}`
	warns := []string{"warn1", "warn2"}
	out, err := pipeline.InjectWarnings(input, warns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	arr, ok := parsed["warnings"].([]interface{})
	if !ok {
		t.Fatalf("expected array, got %T", parsed["warnings"])
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(arr))
	}
}

func TestInjectWarnings_NonJSONPassthrough(t *testing.T) {
	input := "plain text stats output"
	out, err := pipeline.InjectWarnings(input, []string{"w"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != input {
		t.Fatalf("expected passthrough, got %q", out)
	}
}

// ─── DataToJSONL ──────────────────────────────────────────────────────────────

func TestDataToJSONL_Empty(t *testing.T) {
	out, err := pipeline.DataToJSONL(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty string, got %q", out)
	}
}

func TestDataToJSONL_SingleRecord(t *testing.T) {
	data := []interface{}{map[string]interface{}{"tool": "Bash", "status": "success"}}
	out, err := pipeline.DataToJSONL(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if obj["tool"] != "Bash" {
		t.Fatalf("unexpected tool: %v", obj["tool"])
	}
}

func TestDataToJSONL_MultipleRecords(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"n": 1},
		map[string]interface{}{"n": 2},
		map[string]interface{}{"n": 3},
	}
	out, err := pipeline.DataToJSONL(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

// ─── BuildStatsOnlyResponse ───────────────────────────────────────────────────

func TestBuildStatsOnlyResponse_Empty(t *testing.T) {
	// Should not error on empty data
	out, err := pipeline.BuildStatsOnlyResponse(nil, "query_tools", "turn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result is a stats string (may be empty or contain headers)
	_ = out
}

func TestBuildStatsOnlyResponse_TimestampTool(t *testing.T) {
	// query_user_messages uses timestamp stats
	data := []interface{}{
		map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z", "role": "user", "content": "hello"},
	}
	out, err := pipeline.BuildStatsOnlyResponse(data, "query_user_messages", "turn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}

func TestBuildStatsOnlyResponse_SessionLevel(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"sessionId": "abc123", "role": "user", "content": "hello"},
	}
	out, err := pipeline.BuildStatsOnlyResponse(data, "query_user_messages", "session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}

func TestBuildStatsOnlyResponse_StandardTool(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"tool_name": "Bash", "status": "success"},
		map[string]interface{}{"tool_name": "Read", "status": "error"},
	}
	out, err := pipeline.BuildStatsOnlyResponse(data, "query_tools", "turn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}

// ─── TimestampStatsTools ──────────────────────────────────────────────────────

func TestTimestampStatsTools_Contents(t *testing.T) {
	expected := []string{
		"query_user_messages",
		"query_conversation_flow",
		"query_timestamps",
		"query_summaries",
	}
	for _, name := range expected {
		if !pipeline.TimestampStatsTools[name] {
			t.Errorf("expected %q to be in TimestampStatsTools", name)
		}
	}
	if pipeline.TimestampStatsTools["query_tools"] {
		t.Error("query_tools should not be in TimestampStatsTools")
	}
}

// ─── BuildStatsFirstResponse ──────────────────────────────────────────────────

func TestBuildStatsFirstResponse_Basic(t *testing.T) {
	rawData := []interface{}{
		map[string]interface{}{"tool_name": "Bash", "status": "success"},
	}
	parsedData := rawData

	out, err := pipeline.BuildStatsFirstResponse(
		testConfig(),
		rawData, parsedData,
		map[string]interface{}{},
		"query_tools", "turn",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "---") {
		t.Errorf("expected separator '---' in output, got: %s", out)
	}
}

func TestBuildStatsFirstResponse_TimestampTool(t *testing.T) {
	rawData := []interface{}{
		map[string]interface{}{"timestamp": "2024-01-01T10:00:00Z", "role": "user"},
	}
	out, err := pipeline.BuildStatsFirstResponse(
		testConfig(),
		rawData, rawData,
		map[string]interface{}{},
		"query_user_messages", "turn",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}

func TestBuildStatsFirstResponse_SessionLevel(t *testing.T) {
	rawData := []interface{}{
		map[string]interface{}{"sessionId": "abc", "role": "user", "content": "hi"},
	}
	out, err := pipeline.BuildStatsFirstResponse(
		testConfig(),
		rawData, rawData,
		map[string]interface{}{},
		"query_user_messages", "session",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = out
}

// ─── BuildStandardResponse ────────────────────────────────────────────────────

func TestBuildStandardResponse_Basic(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"tool_name": "Bash"},
	}
	out, err := pipeline.BuildStandardResponse(
		testConfig(),
		data,
		map[string]interface{}{},
		"query_tools",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "inline") {
		t.Errorf("expected 'inline' in output, got: %s", out)
	}
}

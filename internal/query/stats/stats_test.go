package stats_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/query/stats"
)

// ─── GenerateStats ────────────────────────────────────────────────────────────

func TestGenerateStats_BasicToolNames(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}
{"tool":"Bash","status":"success"}`

	result, err := stats.GenerateStats(jsonlData)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	counts := parseStatsCounts(t, result)
	if counts["Bash"] != 2 {
		t.Errorf("expected Bash count=2, got %d", counts["Bash"])
	}
	if counts["Read"] != 1 {
		t.Errorf("expected Read count=1, got %d", counts["Read"])
	}
}

func TestGenerateStats_AlternativeFieldName(t *testing.T) {
	jsonlData := `{"ToolName":"Edit","status":"success"}
{"ToolName":"Edit","status":"error"}`

	result, err := stats.GenerateStats(jsonlData)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}
	counts := parseStatsCounts(t, result)
	if counts["Edit"] != 2 {
		t.Errorf("expected Edit count=2, got %d", counts["Edit"])
	}
}

func TestGenerateStats_Empty(t *testing.T) {
	result, err := stats.GenerateStats("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty output for empty input, got %q", result)
	}
}

func TestGenerateStats_UnknownKey(t *testing.T) {
	jsonlData := `{"no_tool_field":true}`

	result, err := stats.GenerateStats(jsonlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	counts := parseStatsCounts(t, result)
	if counts["unknown"] != 1 {
		t.Errorf("expected unknown count=1, got %d", counts["unknown"])
	}
}

// ─── GenerateTimestampStats ───────────────────────────────────────────────────

func TestGenerateTimestampStats_Basic(t *testing.T) {
	jsonlData := `{"timestamp":"2024-01-01T10:00:00Z","sessionId":"s1","role":"user"}
{"timestamp":"2024-01-01T10:30:00Z","sessionId":"s1","role":"assistant"}
{"timestamp":"2024-01-01T11:00:00Z","sessionId":"s2","role":"user"}`

	result, err := stats.GenerateTimestampStats(jsonlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitNonEmpty(result)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (summary + buckets), got %d", len(lines))
	}

	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("invalid summary JSON: %v", err)
	}
	if int(summary["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", summary["total"])
	}
	if int(summary["session_count"].(float64)) != 2 {
		t.Errorf("expected session_count=2, got %v", summary["session_count"])
	}
}

func TestGenerateTimestampStats_Empty(t *testing.T) {
	result, err := stats.GenerateTimestampStats("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

func TestGenerateTimestampStats_SkipsMissingTimestamp(t *testing.T) {
	jsonlData := `{"sessionId":"s1","role":"user"}
{"timestamp":"2024-01-01T10:00:00Z","sessionId":"s2","role":"user"}`

	result, err := stats.GenerateTimestampStats(jsonlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := splitNonEmpty(result)
	// Only 1 record with valid timestamp; summary + 1 bucket
	if len(lines) < 1 {
		t.Fatal("expected at least summary line")
	}
	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("invalid summary JSON: %v", err)
	}
	if int(summary["total"].(float64)) != 1 {
		t.Errorf("expected total=1 (skipping record without timestamp), got %v", summary["total"])
	}
}

// ─── GenerateSessionStats ─────────────────────────────────────────────────────

func TestGenerateSessionStats_Basic(t *testing.T) {
	jsonlData := `{"sessionId":"abc","timestamp":"2024-01-01T10:00:00Z","role":"user"}
{"sessionId":"abc","timestamp":"2024-01-01T10:05:00Z","role":"assistant"}
{"sessionId":"def","timestamp":"2024-01-01T11:00:00Z","role":"user"}`

	result, err := stats.GenerateSessionStats(jsonlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := splitNonEmpty(result)
	// summary + 2 session lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %s", len(lines), result)
	}

	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("invalid summary JSON: %v", err)
	}
	if int(summary["total_sessions"].(float64)) != 2 {
		t.Errorf("expected total_sessions=2, got %v", summary["total_sessions"])
	}
}

func TestGenerateSessionStats_Empty(t *testing.T) {
	result, err := stats.GenerateSessionStats("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

// ─── GroupBySession ───────────────────────────────────────────────────────────

func TestGroupBySession_Basic(t *testing.T) {
	entries := []interface{}{
		map[string]interface{}{"sessionId": "s1", "timestamp": "2024-01-01T10:00:00Z", "content": "hello"},
		map[string]interface{}{"sessionId": "s2", "timestamp": "2024-01-01T11:00:00Z", "content": "world"},
		map[string]interface{}{"sessionId": "s1", "timestamp": "2024-01-01T10:05:00Z", "content": "again"},
	}

	result := stats.GroupBySession(entries)

	if len(result) != 2 {
		t.Fatalf("expected 2 session groups, got %d", len(result))
	}

	g0 := result[0].(map[string]interface{})
	if g0["session_id"] != "s1" {
		t.Errorf("expected first group to be s1, got %v", g0["session_id"])
	}
	if int(g0["match_count"].(int)) != 2 {
		t.Errorf("expected match_count=2 for s1, got %v", g0["match_count"])
	}
}

func TestGroupBySession_SnakeCaseSessionID(t *testing.T) {
	entries := []interface{}{
		map[string]interface{}{"session_id": "ss1", "content": "hi"},
	}
	result := stats.GroupBySession(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	g := result[0].(map[string]interface{})
	if g["session_id"] != "ss1" {
		t.Errorf("expected session_id=ss1, got %v", g["session_id"])
	}
}

func TestGroupBySession_Empty(t *testing.T) {
	result := stats.GroupBySession(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %v", result)
	}
}

func TestGroupBySession_ContextTurnsNotCounted(t *testing.T) {
	entries := []interface{}{
		map[string]interface{}{"sessionId": "s1", "content": "match", "context": false},
		map[string]interface{}{"sessionId": "s1", "content": "ctx", "context": true},
	}
	result := stats.GroupBySession(entries)
	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	g := result[0].(map[string]interface{})
	if int(g["match_count"].(int)) != 1 {
		t.Errorf("expected match_count=1 (context entry excluded), got %v", g["match_count"])
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func parseStatsCounts(t *testing.T, output string) map[string]int {
	t.Helper()
	counts := make(map[string]int)
	for _, line := range splitNonEmpty(output) {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid stats JSON line %q: %v", line, err)
		}
		key, _ := obj["key"].(string)
		count := int(obj["count"].(float64))
		counts[key] = count
	}
	return counts
}

package query

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyJQFilter_Simple(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}
{"tool":"Edit","status":"success"}`

	jqExpr := `.[] | select(.status == "error")`

	result, err := ApplyJQFilter(jsonlData, jqExpr)
	if err != nil {
		t.Fatalf("ApplyJQFilter failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 result, got %d", len(lines))
	}

	if !strings.Contains(result, "Read") {
		t.Error("expected Read in result")
	}
}

func TestApplyJQFilter_Projection(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success","duration":100}
{"tool":"Read","status":"error","duration":50}`

	jqExpr := `.[] | {tool: .tool, status: .status}`

	result, err := ApplyJQFilter(jsonlData, jqExpr)
	if err != nil {
		t.Fatalf("ApplyJQFilter failed: %v", err)
	}

	// Verify projection (no duration field)
	if strings.Contains(result, "duration") {
		t.Error("expected duration to be excluded")
	}
}

func TestApplyJQFilter_DefaultExpression(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}`

	// Empty jq expression should default to ".[]"
	result, err := ApplyJQFilter(jsonlData, "")
	if err != nil {
		t.Fatalf("ApplyJQFilter failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 results, got %d", len(lines))
	}
}

func TestApplyJQFilter_InvalidExpression(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}`

	// Invalid jq expression
	_, err := ApplyJQFilter(jsonlData, ".[ invalid syntax")
	if err == nil {
		t.Error("expected error for invalid jq expression")
	}
}

func TestApplyJQFilter_EmptyData(t *testing.T) {
	result, err := ApplyJQFilter("", ".[]")
	if err != nil {
		t.Fatalf("ApplyJQFilter failed: %v", err)
	}

	if strings.TrimSpace(result) != "" {
		t.Error("expected empty result for empty data")
	}
}

func TestGenerateStats(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"error"}
{"tool":"Bash","status":"error"}
{"tool":"Read","status":"error"}`

	stats, err := GenerateStats(jsonlData)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	// Verify stats format
	if !strings.Contains(stats, "Bash") {
		t.Error("expected Bash in stats")
	}
	if !strings.Contains(stats, "count") {
		t.Error("expected count field")
	}

	// Verify count is correct (Bash should appear twice)
	lines := strings.Split(strings.TrimSpace(stats), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 stat entries, got %d", len(lines))
	}
}

func TestGenerateStats_AlternativeFieldNames(t *testing.T) {
	// Test with "ToolName" field instead of "tool"
	jsonlData := `{"ToolName":"Bash","Status":"error"}
{"ToolName":"Read","Status":"success"}`

	stats, err := GenerateStats(jsonlData)
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	if !strings.Contains(stats, "Bash") {
		t.Error("expected Bash in stats")
	}
	if !strings.Contains(stats, "Read") {
		t.Error("expected Read in stats")
	}
}

func TestGenerateStats_EmptyData(t *testing.T) {
	stats, err := GenerateStats("")
	if err != nil {
		t.Fatalf("GenerateStats failed: %v", err)
	}

	if strings.TrimSpace(stats) != "" {
		t.Error("expected empty stats for empty data")
	}
}

func TestGenerateTimestampStats(t *testing.T) {
	// 5 records: 2 in hour 06, 2 in hour 07, 1 in hour 08
	// 2 distinct sessions
	jsonlData := `{"timestamp":"2026-03-09T06:10:00Z","sessionId":"sess-A","type":"user"}
{"timestamp":"2026-03-09T06:50:00Z","sessionId":"sess-A","type":"user"}
{"timestamp":"2026-03-09T07:05:00Z","sessionId":"sess-B","type":"user"}
{"timestamp":"2026-03-09T07:55:00Z","sessionId":"sess-B","type":"user"}
{"timestamp":"2026-03-09T08:30:00Z","sessionId":"sess-A","type":"user"}`

	result, err := GenerateTimestampStats(jsonlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) < 4 { // 1 summary + 3 hour lines
		t.Fatalf("expected at least 4 lines, got %d: %s", len(lines), result)
	}

	// First line is summary
	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("failed to parse summary line: %v", err)
	}
	if int(summary["total"].(float64)) != 5 {
		t.Errorf("total = %v, want 5", summary["total"])
	}
	if int(summary["session_count"].(float64)) != 2 {
		t.Errorf("session_count = %v, want 2", summary["session_count"])
	}
	if summary["time_range"] == nil {
		t.Error("time_range missing")
	}

	// Remaining lines are hourly buckets
	hourCounts := map[string]int{}
	for _, line := range lines[1:] {
		var bucket map[string]interface{}
		if err := json.Unmarshal([]byte(line), &bucket); err != nil {
			t.Fatalf("failed to parse bucket line: %v", err)
		}
		hour := bucket["hour"].(string)
		count := int(bucket["count"].(float64))
		hourCounts[hour] = count
	}
	if hourCounts["2026-03-09T06"] != 2 {
		t.Errorf("hour 06: got %d, want 2", hourCounts["2026-03-09T06"])
	}
	if hourCounts["2026-03-09T07"] != 2 {
		t.Errorf("hour 07: got %d, want 2", hourCounts["2026-03-09T07"])
	}
	if hourCounts["2026-03-09T08"] != 1 {
		t.Errorf("hour 08: got %d, want 1", hourCounts["2026-03-09T08"])
	}
}

func TestGenerateTimestampStats_Empty(t *testing.T) {
	result, err := GenerateTimestampStats("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(result) != "" {
		t.Error("expected empty result for empty input")
	}
}

func TestParseJQExpressionQuotedError(t *testing.T) {
	_, err := parseJQExpression(`'.[]'`)
	if err == nil {
		t.Fatal("expected quoted expression to return error")
	}
	if !strings.Contains(err.Error(), "appears to be quoted") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestParseJSONLRecordsInvalidJSON(t *testing.T) {
	_, err := parseJSONLRecords("not-json\n")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if !strings.Contains(err.Error(), "invalid JSON at line 1") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEncodeJQResultsMarshalError(t *testing.T) {
	result, err := encodeJQResults([]interface{}{make(chan int)})
	if err == nil {
		t.Fatal("expected marshal error for channel value")
	}
	if result != "" {
		t.Fatalf("expected empty result string, got %q", result)
	}
}

// TestApplyJQFilter_QuotedExpressionError verifies improved error message for quoted expressions
func TestApplyJQFilter_QuotedExpressionError(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}`

	// Test common mistake: wrapping jq expression in quotes
	testCases := []struct {
		name     string
		badExpr  string
		expected string
	}{
		{
			name:     "single quoted expression",
			badExpr:  `'.[] | {tool: .tool}'`,
			expected: "appears to be quoted",
		},
		{
			name:     "single quoted complex expression",
			badExpr:  `'.[] | {turn: .turn, content: .content[0:100]}'`,
			expected: "appears to be quoted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyJQFilter(jsonlData, tc.badExpr)
			if err == nil {
				t.Errorf("expected error for quoted expression: %s", tc.badExpr)
				return
			}

			// Verify error message contains helpful guidance
			if !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("error message should contain '%s' for expression '%s', got: %v",
					tc.expected, tc.badExpr, err)
			}

			// Verify error message suggests correct syntax
			if !strings.Contains(err.Error(), ".[] | {field: .field}") {
				t.Errorf("error message should suggest correct syntax for expression '%s', got: %v",
					tc.badExpr, err)
			}

			t.Logf("Error for '%s': %v", tc.badExpr, err)
		})
	}
}

// TestGroupBySession tests the GroupBySession function
func TestGroupBySession(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		entries := []interface{}{
			map[string]interface{}{"sessionId": "sess-A", "timestamp": "2026-03-09T06:00:00Z", "content": "msg1"},
			map[string]interface{}{"sessionId": "sess-A", "timestamp": "2026-03-09T06:01:00Z", "content": "msg2"},
			map[string]interface{}{"sessionId": "sess-B", "timestamp": "2026-03-09T07:00:00Z", "content": "msg3"},
			map[string]interface{}{"sessionId": "sess-B", "timestamp": "2026-03-09T07:01:00Z", "content": "msg4"},
		}

		result := GroupBySession(entries)

		if len(result) != 2 {
			t.Fatalf("expected 2 session objects, got %d", len(result))
		}

		for _, item := range result {
			obj, ok := item.(map[string]interface{})
			if !ok {
				t.Fatal("expected map[string]interface{}")
			}
			if _, has := obj["session_id"]; !has {
				t.Error("expected session_id field")
			}
			if _, has := obj["match_count"]; !has {
				t.Error("expected match_count field")
			}
			if _, has := obj["first_match"]; !has {
				t.Error("expected first_match field")
			}
			if _, has := obj["last_match"]; !has {
				t.Error("expected last_match field")
			}
			if _, has := obj["turns"]; !has {
				t.Error("expected turns field")
			}
		}

		sessA := result[0].(map[string]interface{})
		if sessA["session_id"] != "sess-A" {
			t.Errorf("expected sess-A first, got %v", sessA["session_id"])
		}
		if int(sessA["match_count"].(int)) != 2 {
			t.Errorf("sess-A match_count = %v, want 2", sessA["match_count"])
		}
		sessB := result[1].(map[string]interface{})
		if sessB["session_id"] != "sess-B" {
			t.Errorf("expected sess-B second, got %v", sessB["session_id"])
		}
		if int(sessB["match_count"].(int)) != 2 {
			t.Errorf("sess-B match_count = %v, want 2", sessB["match_count"])
		}
	})

	t.Run("OrderPreserved", func(t *testing.T) {
		entries := []interface{}{
			map[string]interface{}{"sessionId": "sess-A", "timestamp": "2026-03-09T06:00:00Z"},
			map[string]interface{}{"sessionId": "sess-B", "timestamp": "2026-03-09T07:00:00Z"},
			map[string]interface{}{"sessionId": "sess-A", "timestamp": "2026-03-09T06:01:00Z"},
			map[string]interface{}{"sessionId": "sess-B", "timestamp": "2026-03-09T07:01:00Z"},
		}

		result := GroupBySession(entries)

		if len(result) != 2 {
			t.Fatalf("expected 2 session objects, got %d", len(result))
		}

		first := result[0].(map[string]interface{})
		if first["session_id"] != "sess-A" {
			t.Errorf("expected sess-A first (first-seen order), got %v", first["session_id"])
		}

		second := result[1].(map[string]interface{})
		if second["session_id"] != "sess-B" {
			t.Errorf("expected sess-B second, got %v", second["session_id"])
		}
	})

	t.Run("SnakeCaseSessionId", func(t *testing.T) {
		entries := []interface{}{
			map[string]interface{}{"session_id": "sess-X", "timestamp": "2026-03-09T06:00:00Z", "content_preview": "hello"},
			map[string]interface{}{"session_id": "sess-X", "timestamp": "2026-03-09T06:01:00Z", "content_preview": "world"},
			map[string]interface{}{"session_id": "sess-Y", "timestamp": "2026-03-09T07:00:00Z", "content_preview": "foo"},
		}

		result := GroupBySession(entries)

		if len(result) != 2 {
			t.Fatalf("expected 2 session objects, got %d", len(result))
		}

		sessX := result[0].(map[string]interface{})
		if sessX["session_id"] != "sess-X" {
			t.Errorf("expected sess-X, got %v", sessX["session_id"])
		}
		if int(sessX["match_count"].(int)) != 2 {
			t.Errorf("sess-X match_count = %v, want 2", sessX["match_count"])
		}
	})

	t.Run("CamelCaseSessionId", func(t *testing.T) {
		entries := []interface{}{
			map[string]interface{}{"sessionId": "sess-camel-1", "timestamp": "2026-03-09T06:00:00Z"},
			map[string]interface{}{"sessionId": "sess-camel-2", "timestamp": "2026-03-09T07:00:00Z"},
			map[string]interface{}{"sessionId": "sess-camel-1", "timestamp": "2026-03-09T06:02:00Z"},
		}

		result := GroupBySession(entries)

		if len(result) != 2 {
			t.Fatalf("expected 2 session objects, got %d", len(result))
		}

		// Output should use snake_case session_id
		obj := result[0].(map[string]interface{})
		if _, has := obj["session_id"]; !has {
			t.Error("output should use snake_case session_id field")
		}
		if _, has := obj["sessionId"]; has {
			t.Error("output should NOT have camelCase sessionId field")
		}
		if obj["session_id"] != "sess-camel-1" {
			t.Errorf("expected sess-camel-1 first, got %v", obj["session_id"])
		}
	})

	t.Run("SingleSession", func(t *testing.T) {
		entries := []interface{}{
			map[string]interface{}{"sessionId": "only-session", "timestamp": "2026-03-09T06:00:00Z"},
			map[string]interface{}{"sessionId": "only-session", "timestamp": "2026-03-09T06:01:00Z"},
			map[string]interface{}{"sessionId": "only-session", "timestamp": "2026-03-09T06:02:00Z"},
		}

		result := GroupBySession(entries)

		if len(result) != 1 {
			t.Fatalf("expected 1 session object, got %d", len(result))
		}

		obj := result[0].(map[string]interface{})
		if int(obj["match_count"].(int)) != len(entries) {
			t.Errorf("match_count = %v, want %d", obj["match_count"], len(entries))
		}
	})
}

// TestApplyJQFilter_GenuineSyntaxStillReportsOriginalError verifies that genuine syntax errors still get appropriate error messages
func TestApplyJQFilter_GenuineSyntaxStillReportsOriginalError(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}`

	// Test genuine syntax errors (not quote-related)
	testCases := []struct {
		name     string
		badExpr  string
		expected string
	}{
		{
			name:     "invalid bracket syntax",
			badExpr:  `. [ invalid syntax`,
			expected: "invalid jq expression",
		},
		{
			name:     "missing closing brace",
			badExpr:  `.[] | select(.tool == "Bash"`,
			expected: "invalid jq expression",
		},
		{
			name:     "invalid function",
			badExpr:  `.[] | invalid_function()`,
			expected: "invalid jq expression",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyJQFilter(jsonlData, tc.badExpr)
			if err == nil {
				t.Errorf("expected error for invalid expression: %s", tc.badExpr)
				return
			}

			// Verify error message doesn't incorrectly suggest quote issues
			if strings.Contains(err.Error(), "appears to be quoted") {
				t.Errorf("genuine syntax error should not suggest quote issues for expression '%s', got: %v",
					tc.badExpr, err)
			}

			// Should still indicate invalid jq expression
			if !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("error message should contain '%s' for expression '%s', got: %v",
					tc.expected, tc.badExpr, err)
			}

			t.Logf("Error for '%s': %v", tc.badExpr, err)
		})
	}
}

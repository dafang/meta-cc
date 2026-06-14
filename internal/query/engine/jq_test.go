package engine

import (
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

	if strings.Contains(result, "duration") {
		t.Error("expected duration to be excluded")
	}
}

func TestApplyJQFilter_DefaultExpression(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}`

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

func TestApplyJQFilter_QuotedExpressionError(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}
{"tool":"Read","status":"error"}`

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
			if !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("error message should contain '%s', got: %v", tc.expected, err)
			}
			if !strings.Contains(err.Error(), ".[] | {field: .field}") {
				t.Errorf("error message should suggest correct syntax, got: %v", err)
			}
		})
	}
}

func TestApplyJQFilter_GenuineSyntaxStillReportsOriginalError(t *testing.T) {
	jsonlData := `{"tool":"Bash","status":"success"}`

	testCases := []struct {
		name     string
		badExpr  string
		expected string
	}{
		{"invalid bracket syntax", `. [ invalid syntax`, "invalid jq expression"},
		{"missing closing brace", `.[] | select(.tool == "Bash"`, "invalid jq expression"},
		{"invalid function", `.[] | invalid_function()`, "invalid jq expression"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ApplyJQFilter(jsonlData, tc.badExpr)
			if err == nil {
				t.Errorf("expected error for: %s", tc.badExpr)
				return
			}
			if strings.Contains(err.Error(), "appears to be quoted") {
				t.Errorf("genuine syntax error should not suggest quote issues for: %s", tc.badExpr)
			}
			if !strings.Contains(err.Error(), tc.expected) {
				t.Errorf("expected '%s' in error for: %s, got: %v", tc.expected, tc.badExpr, err)
			}
		})
	}
}

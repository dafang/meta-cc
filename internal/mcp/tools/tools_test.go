package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/mcp/tools"
)

func TestStandardToolParameters(t *testing.T) {
	params := tools.StandardToolParameters()

	requiredParams := []string{
		"scope", "provider", "jq_filter", "stats_only",
		"stats_first", "inline_threshold_bytes", "output_format",
	}

	for _, param := range requiredParams {
		if _, ok := params[param]; !ok {
			t.Errorf("missing standard parameter: %s", param)
		}
	}
}

func TestMergeParameters(t *testing.T) {
	specific := map[string]tools.Property{
		"limit": {
			Type:        "number",
			Description: "Max results",
		},
		"scope": {
			Type:        "string",
			Description: "Custom scope description",
		},
	}

	merged := tools.MergeParameters(specific)

	if _, ok := merged["limit"]; !ok {
		t.Error("specific parameter 'limit' missing")
	}
	if _, ok := merged["jq_filter"]; !ok {
		t.Error("standard parameter 'jq_filter' missing")
	}
	if merged["scope"].Description != "Custom scope description" {
		t.Errorf("parameter override failed, got: %s", merged["scope"].Description)
	}
}

func TestGetToolDefinitions(t *testing.T) {
	defs := tools.GetToolDefinitions()
	if len(defs) == 0 {
		t.Error("expected non-empty tool definitions")
	}

	// Verify all tools serialize to JSON
	for _, tool := range defs {
		_, err := json.Marshal(tool)
		if err != nil {
			t.Errorf("tool %s failed to serialize: %v", tool.Name, err)
		}
	}
}

func TestBuildToolSchemaIndex(t *testing.T) {
	index := tools.BuildToolSchemaIndex()
	if len(index) == 0 {
		t.Error("expected non-empty schema index")
	}

	// Check a known tool
	if _, ok := index["query_user_messages"]; !ok {
		t.Error("expected query_user_messages in index")
	}
}

func TestGetToolSchemaByName(t *testing.T) {
	index := tools.BuildToolSchemaIndex()

	schema, err := tools.GetToolSchemaByName(index, "query_user_messages")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema.Type != "object" {
		t.Errorf("expected object type, got %s", schema.Type)
	}

	_, err = tools.GetToolSchemaByName(index, "nonexistent_tool")
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' in error, got: %v", err)
	}
}

func TestValidateToolArgs_ValidTool_ValidArgs(t *testing.T) {
	err := tools.ValidateToolArgs("query_user_messages", map[string]interface{}{
		"pattern": "error",
		"limit":   float64(10),
	})
	if err != nil {
		t.Fatalf("unexpected error for valid tool+args: %v", err)
	}
}

func TestValidateToolArgs_ValidTool_EmptyArgs(t *testing.T) {
	err := tools.ValidateToolArgs("query_tool_errors", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error for empty args: %v", err)
	}
}

func TestValidateToolArgs_ValidTool_InvalidArgKey(t *testing.T) {
	err := tools.ValidateToolArgs("query_user_messages", map[string]interface{}{
		"unknown_key": "value",
	})
	if err == nil {
		t.Fatal("expected error for invalid arg key")
	}
	if !strings.Contains(err.Error(), "unknown_key") {
		t.Errorf("expected 'unknown_key' in error, got: %v", err)
	}
}

func TestValidateToolArgs_UnknownTool(t *testing.T) {
	err := tools.ValidateToolArgs("no_such_tool", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "no_such_tool") {
		t.Errorf("expected tool name in error, got: %v", err)
	}
}

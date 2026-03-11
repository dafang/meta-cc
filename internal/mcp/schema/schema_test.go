package schema

import (
	"strings"
	"testing"
)

func TestValidateArgKeys_Empty(t *testing.T) {
	s := ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"scope": {Type: "string"},
		},
	}
	if err := ValidateArgKeys(nil, s); err != nil {
		t.Errorf("expected nil for nil args, got %v", err)
	}
	if err := ValidateArgKeys(map[string]interface{}{}, s); err != nil {
		t.Errorf("expected nil for empty args, got %v", err)
	}
}

func TestValidateArgKeys_AllValid(t *testing.T) {
	s := ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"scope": {Type: "string"},
			"limit": {Type: "number"},
		},
	}
	args := map[string]interface{}{
		"scope": "project",
		"limit": 10,
	}
	if err := ValidateArgKeys(args, s); err != nil {
		t.Errorf("expected nil for valid args, got %v", err)
	}
}

func TestValidateArgKeys_UnknownKey(t *testing.T) {
	s := ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"scope": {Type: "string"},
		},
	}
	args := map[string]interface{}{
		"scope":   "project",
		"unknown": "value",
	}
	err := ValidateArgKeys(args, s)
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error to mention 'unknown', got: %v", err)
	}
	if !strings.Contains(err.Error(), "scope") {
		t.Errorf("expected error to list valid keys including 'scope', got: %v", err)
	}
}

func TestValidateArgKeys_MultipleUnknown(t *testing.T) {
	s := ToolSchema{
		Type: "object",
		Properties: map[string]Property{
			"scope": {Type: "string"},
		},
	}
	args := map[string]interface{}{
		"alpha": "a",
		"beta":  "b",
	}
	err := ValidateArgKeys(args, s)
	if err == nil {
		t.Fatal("expected error for multiple unknown keys, got nil")
	}
	// Both unknown keys should appear (sorted)
	if !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta") {
		t.Errorf("expected both unknown keys in error message, got: %v", err)
	}
}

func TestValidateArgKeys_EmptySchema(t *testing.T) {
	s := ToolSchema{
		Type:       "object",
		Properties: map[string]Property{},
	}
	args := map[string]interface{}{
		"foo": "bar",
	}
	err := ValidateArgKeys(args, s)
	if err == nil {
		t.Fatal("expected error for args against empty schema, got nil")
	}
}

func TestBuildSchemaIndex_Empty(t *testing.T) {
	index := BuildSchemaIndex(nil)
	if len(index) != 0 {
		t.Errorf("expected empty index for nil defs, got %d entries", len(index))
	}
}

func TestBuildSchemaIndex_MultipleTools(t *testing.T) {
	defs := []Tool{
		{Name: "tool_a", InputSchema: ToolSchema{Type: "object", Properties: map[string]Property{"x": {Type: "string"}}}},
		{Name: "tool_b", InputSchema: ToolSchema{Type: "object", Properties: map[string]Property{"y": {Type: "number"}}}},
	}
	index := BuildSchemaIndex(defs)
	if len(index) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(index))
	}
	if _, ok := index["tool_a"]; !ok {
		t.Error("expected tool_a in index")
	}
	if _, ok := index["tool_b"]; !ok {
		t.Error("expected tool_b in index")
	}
}

func TestGetByName_Found(t *testing.T) {
	defs := []Tool{
		{Name: "my_tool", InputSchema: ToolSchema{Type: "object", Properties: map[string]Property{"p": {Type: "string"}}}},
	}
	index := BuildSchemaIndex(defs)
	s, err := GetByName(index, "my_tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Type != "object" {
		t.Errorf("expected type 'object', got '%s'", s.Type)
	}
	if _, ok := s.Properties["p"]; !ok {
		t.Error("expected property 'p' in schema")
	}
}

func TestGetByName_NotFound(t *testing.T) {
	index := BuildSchemaIndex(nil)
	_, err := GetByName(index, "no_such_tool")
	if err == nil {
		t.Fatal("expected error for missing tool, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' in error message, got: %v", err)
	}
}

func TestToolSchema_Types(t *testing.T) {
	// Ensure the types can be constructed and used
	p := Property{
		Type:        "array",
		Description: "list of items",
		Items: &Property{
			Type: "string",
		},
	}
	ts := ToolSchema{
		Type:       "object",
		Properties: map[string]Property{"files": p},
		Required:   []string{"files"},
	}
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool.",
		InputSchema: ts,
	}
	if tool.Name != "test_tool" {
		t.Errorf("unexpected tool name: %s", tool.Name)
	}
	if tool.InputSchema.Properties["files"].Items == nil {
		t.Error("expected Items to be non-nil for array property")
	}
}

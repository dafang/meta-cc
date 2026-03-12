package executor

import (
	"context"
	"testing"
)

func TestSpecialToolRegistry_AnalysisHandlers(t *testing.T) {
	analysisTool := []string{
		"analyze_bugs",
		"analyze_errors",
		"quality_scan",
		"get_work_patterns",
		"get_timeline",
		"get_tech_debt",
	}
	for _, tool := range analysisTool {
		if _, ok := specialToolRegistry[tool]; !ok {
			t.Errorf("analysis tool %q not registered in specialToolRegistry", tool)
		}
	}
}

func TestSpecialToolRegistry_QueryHandlers(t *testing.T) {
	queryTools := []string{
		"cleanup_temp_files",
		"get_session_directory",
		"inspect_session_files",
		"execute_stage2_query",
		"get_session_metadata",
	}
	for _, tool := range queryTools {
		if _, ok := specialToolRegistry[tool]; !ok {
			t.Errorf("query tool %q not registered in specialToolRegistry", tool)
		}
	}
}

func TestSpecialToolRegistry_UnknownTool(t *testing.T) {
	_, ok := specialToolRegistry["nonexistent_tool"]
	if ok {
		t.Error("expected nonexistent_tool to not be registered")
	}
}

func TestRegisterHandler_AddsToRegistry(t *testing.T) {
	const testTool = "test_tool_registry_unit"
	called := false
	registerHandler(testTool, func(_ context.Context, _ *ToolExecutor, _ map[string]interface{}) (string, error) {
		called = true
		return "ok", nil
	})
	defer delete(specialToolRegistry, testTool)

	h, ok := specialToolRegistry[testTool]
	if !ok {
		t.Fatal("handler not registered")
	}
	out, err := h(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "ok" {
		t.Errorf("expected 'ok', got %q", out)
	}
	if !called {
		t.Error("handler was not called")
	}
}

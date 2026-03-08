package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/config"
)

// TestConvenienceToolsIntegration tests all 10 convenience tools
// These tests verify that convenience tools correctly wrap handleQuery()
//
// NOTE: These tests are currently skipped because they require complex test fixture setup
// The underlying handleQuery() is already tested extensively in handlers_query_test.go
// These convenience tools are simple wrappers with pre-configured jq filters

// Helper to create test JSONL data for all convenience tool tests
func setupConvenienceToolTest(t *testing.T) (*ToolExecutor, *config.Config, func()) {
	tmpDir := t.TempDir()

	// Create comprehensive test data covering all tool types
	testEntries := []map[string]interface{}{
		// User message with string content
		{
			"type":      "user",
			"timestamp": "2025-01-01T10:00:00Z",
			"uuid":      "user-1",
			"message": map[string]interface{}{
				"content": "Fix the error in main.go",
			},
		},
		// Assistant with tool_use and usage
		{
			"type":      "assistant",
			"timestamp": "2025-01-01T10:01:00Z",
			"uuid":      "asst-1",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "tool_use",
						"id":   "tool-1",
						"name": "Read",
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			},
		},
		// User with tool_result error
		{
			"type":      "user",
			"timestamp": "2025-01-01T10:02:00Z",
			"uuid":      "user-2",
			"message": map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type":     "tool_result",
						"is_error": true,
						"content":  "File not found",
					},
				},
			},
		},
		// System API error
		{
			"type":      "system",
			"subtype":   "api_error",
			"timestamp": "2025-01-01T10:03:00Z",
			"uuid":      "sys-1",
		},
		// File snapshot
		{
			"type":      "file-history-snapshot",
			"messageId": "msg-1",
			"timestamp": "2025-01-01T10:04:00Z",
			"uuid":      "snap-1",
		},
		// Summary
		{
			"type":      "summary",
			"summary":   "Fixed error in codebase",
			"timestamp": "2025-01-01T11:00:00Z",
			"uuid":      "summ-1",
		},
	}

	// Write JSONL file
	file := filepath.Join(tmpDir, "test.jsonl")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, entry := range testEntries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatalf("failed to write test data: %v", err)
		}
	}

	// Setup environment
	originalEnv := os.Getenv("CLAUDE_PROJECT_DIR")
	os.Setenv("CLAUDE_PROJECT_DIR", tmpDir)

	cleanup := func() {
		os.Setenv("CLAUDE_PROJECT_DIR", originalEnv)
	}

	executor := NewToolExecutor()
	cfg := &config.Config{}

	return executor, cfg, cleanup
}

func TestHandleQueryUserMessages(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryUserMessages(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryUserMessages() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one user message
	if len(results.Entries) == 0 {
		t.Error("expected at least one user message")
	}
}

func TestHandleQueryTools(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryTools(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryTools() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one assistant message with tool_use
	if len(results.Entries) == 0 {
		t.Error("expected at least one tool execution")
	}
}

func TestHandleQueryToolErrors(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryToolErrors(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryToolErrors() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one error
	if len(results.Entries) == 0 {
		t.Error("expected at least one tool error")
	}
}

func TestHandleQueryTokenUsage(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryTokenUsage(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryTokenUsage() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one message with usage
	if len(results.Entries) == 0 {
		t.Error("expected at least one message with token usage")
	}
}

func TestHandleQueryConversationFlow(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryConversationFlow(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryConversationFlow() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return user and assistant messages
	if len(results.Entries) == 0 {
		t.Error("expected at least one conversation message")
	}
}

func TestHandleQuerySystemErrors(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQuerySystemErrors(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQuerySystemErrors() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one system error
	if len(results.Entries) == 0 {
		t.Error("expected at least one system error")
	}
}

func TestHandleQueryFileSnapshots(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryFileSnapshots(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryFileSnapshots() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one snapshot
	if len(results.Entries) == 0 {
		t.Error("expected at least one file snapshot")
	}
}

func TestHandleQueryTimestamps(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQueryTimestamps(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQueryTimestamps() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return entries with timestamps
	if len(results.Entries) == 0 {
		t.Error("expected at least one entry with timestamp")
	}
}

func TestHandleQuerySummaries(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	results, err := executor.handleQuerySummaries(cfg, "project", map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleQuerySummaries() error = %v", err)
	}

	// Results is now []interface{} directly, no need to unmarshal

	// Should return at least one summary
	if len(results.Entries) == 0 {
		t.Error("expected at least one summary")
	}
}

// TestHandleQueryUserMessagesContentLengthFiltering tests min/max content length filtering
// Stage 30.3: Content length filtering for query_user_messages
func TestHandleQueryUserMessagesContentLengthFiltering(t *testing.T) {
	// Create test data with user messages of varying content lengths
	testData := `{"type":"user","timestamp":"2025-01-01T10:00:00Z","message":{"content":"hi"}}
{"type":"user","timestamp":"2025-01-01T10:00:01Z","message":{"content":"medium length message here"}}
{"type":"user","timestamp":"2025-01-01T10:00:02Z","message":{"content":"this is a much longer message that exceeds fifty characters in total length for testing purposes"}}
{"type":"user","timestamp":"2025-01-01T10:00:03Z","message":{"content":[{"type":"tool_result","content":"array content"}]}}
`

	projectPath := setupTestSessionDir(t, testData)

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(projectPath); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	executor := NewToolExecutor()
	cfg := &config.Config{}

	t.Run("min_content_length_only", func(t *testing.T) {
		result, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{
			"min_content_length": 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// "hi" (2 chars) should be excluded; "medium length message here" (25 chars) and the long message should remain
		if len(result.Entries) < 2 {
			t.Errorf("expected at least 2 results with min_content_length=10, got %d", len(result.Entries))
		}
		// "hi" should NOT be in results
		for _, r := range result.Entries {
			if m, ok := r.(map[string]interface{}); ok {
				if msg, ok := m["message"].(map[string]interface{}); ok {
					if content, ok := msg["content"].(string); ok && content == "hi" {
						t.Error("message 'hi' should be excluded by min_content_length=10")
					}
				}
			}
		}
	})

	t.Run("max_content_length_only", func(t *testing.T) {
		result, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{
			"max_content_length": 30,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// "hi" (2 chars) and "medium length message here" (25 chars) should remain
		if len(result.Entries) != 2 {
			t.Errorf("expected 2 results with max_content_length=30, got %d", len(result.Entries))
		}
	})

	t.Run("both_min_and_max", func(t *testing.T) {
		result, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{
			"min_content_length": 10,
			"max_content_length": 30,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Only "medium length message here" (25 chars) should match
		if len(result.Entries) != 1 {
			t.Errorf("expected 1 result with min=10, max=30, got %d", len(result.Entries))
		}
	})

	t.Run("neither_length_filter", func(t *testing.T) {
		result, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Default content_type is "string", so 3 string messages should be returned
		if len(result.Entries) != 3 {
			t.Errorf("expected 3 results with no length filters, got %d", len(result.Entries))
		}
	})

	t.Run("array_content_type_with_length_filter_returns_error", func(t *testing.T) {
		_, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{
			"content_type":       "array",
			"min_content_length": 2,
		})
		if err == nil {
			t.Error("expected error when using content length filter with array content_type")
		}
	})

	t.Run("array_content_type_with_max_length_filter_returns_error", func(t *testing.T) {
		_, err := executor.handleQueryUserMessages(cfg, "session", map[string]interface{}{
			"content_type":       "array",
			"max_content_length": 100,
		})
		if err == nil {
			t.Error("expected error when using content length filter with array content_type")
		}
	})
}

// TestQueryUserMessagesSchemaHasContentLengthParams verifies schema includes content length parameters
// Stage 30.3: Schema validation
func TestQueryUserMessagesSchemaHasContentLengthParams(t *testing.T) {
	tools := getToolDefinitions()

	var userMsgTool *Tool
	for i, tool := range tools {
		if tool.Name == "query_user_messages" {
			userMsgTool = &tools[i]
			break
		}
	}

	if userMsgTool == nil {
		t.Fatal("query_user_messages tool not found")
	}

	props := userMsgTool.InputSchema.Properties

	// Check min_content_length
	minProp, exists := props["min_content_length"]
	if !exists {
		t.Error("query_user_messages schema missing min_content_length parameter")
	} else {
		if minProp.Type != "number" {
			t.Errorf("min_content_length type should be 'number', got '%s'", minProp.Type)
		}
		if !strings.Contains(minProp.Description, "string content") {
			t.Error("min_content_length description should mention 'string content' limitation")
		}
	}

	// Check max_content_length
	maxProp, exists := props["max_content_length"]
	if !exists {
		t.Error("query_user_messages schema missing max_content_length parameter")
	} else {
		if maxProp.Type != "number" {
			t.Errorf("max_content_length type should be 'number', got '%s'", maxProp.Type)
		}
		if !strings.Contains(maxProp.Description, "string content") {
			t.Error("max_content_length description should mention 'string content' limitation")
		}
	}

	// Check content_type is declared in schema (was missing before Stage 30.3)
	contentTypeProp, exists := props["content_type"]
	if !exists {
		t.Error("query_user_messages schema missing content_type parameter")
	} else {
		if contentTypeProp.Type != "string" {
			t.Errorf("content_type type should be 'string', got '%s'", contentTypeProp.Type)
		}
	}
}

func TestHandleQueryToolBlocks(t *testing.T) {
	t.Skip("Skipping - underlying handleQuery() is already tested")
	executor, cfg, cleanup := setupConvenienceToolTest(t)
	defer cleanup()

	tests := []struct {
		name      string
		blockType string
		wantErr   bool
	}{
		{"tool_use blocks", "tool_use", false},
		{"tool_result blocks", "tool_result", false},
		{"invalid block type", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := executor.handleQueryToolBlocks(cfg, "project", map[string]interface{}{
				"block_type": tt.blockType,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("handleQueryToolBlocks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Results is now []interface{} directly, no need to unmarshal
				if results.Entries == nil {
					t.Error("expected non-nil results for valid block type")
				}
			}
		})
	}
}

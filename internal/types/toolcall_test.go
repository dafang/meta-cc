package types_test

import (
	"testing"

	"github.com/yaleh/meta-cc/internal/types"
)

func TestExtractToolCalls_PairsToolUseWithResult(t *testing.T) {
	entries := []types.SessionEntry{
		{
			Type:      "assistant",
			UUID:      "entry-1",
			Timestamp: "2026-01-01T00:00:00Z",
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:    "tu-abc",
							Name:  "Bash",
							Input: map[string]interface{}{"command": "ls"},
						},
					},
				},
			},
		},
		{
			Type: "user",
			UUID: "entry-2",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: "tu-abc",
							Content:   "file.txt",
							Status:    "success",
						},
					},
				},
			},
		},
	}

	calls := types.ExtractToolCalls(entries)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	tc := calls[0]
	if tc.ToolName != "Bash" {
		t.Errorf("expected ToolName='Bash', got %q", tc.ToolName)
	}
	if tc.Output != "file.txt" {
		t.Errorf("expected Output='file.txt', got %q", tc.Output)
	}
	if tc.Status != "success" {
		t.Errorf("expected Status='success', got %q", tc.Status)
	}
}

func TestExtractToolCalls_NoResult(t *testing.T) {
	entries := []types.SessionEntry{
		{
			Type: "assistant",
			UUID: "e1",
			Message: &types.Message{
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   "tu-xyz",
							Name: "Read",
						},
					},
				},
			},
		},
	}

	calls := types.ExtractToolCalls(entries)
	if len(calls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(calls))
	}
	if calls[0].Output != "" || calls[0].Status != "" {
		t.Errorf("expected empty output/status for unmatched tool call")
	}
}

func TestExtractToolCalls_EmptyEntries(t *testing.T) {
	calls := types.ExtractToolCalls(nil)
	if len(calls) != 0 {
		t.Errorf("expected 0 tool calls for nil input")
	}
}

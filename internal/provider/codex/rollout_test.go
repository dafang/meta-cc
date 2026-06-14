package codex

import (
	"path/filepath"
	"testing"
)

func TestDetectSchemaVersion(t *testing.T) {
	if got := detectSchemaVersion([]byte(`{"type":"session_meta"}`)); got != schemaLegacy {
		t.Fatalf("legacy detect failed")
	}
	if got := detectSchemaVersion([]byte(`{"type":"turn.started"}`)); got != schemaNew {
		t.Fatalf("new detect failed")
	}
}

func TestLoadTurnsFromRolloutLegacyAndNew(t *testing.T) {
	legacy, usage, err := loadTurnsFromRollout(filepath.Join("..", "..", "..", "tests", "fixtures", "codex", "rollout-legacy-sample.jsonl"), 100)
	if err != nil {
		t.Fatalf("legacy load: %v", err)
	}
	if len(legacy) != 1 || legacy[0].UserText == "" || len(legacy[0].ToolCalls) != 1 {
		t.Fatalf("unexpected legacy turns: %#v", legacy)
	}
	if usage.InputTokens != 0 {
		t.Fatalf("unexpected legacy usage: %#v", usage)
	}

	newTurns, _, err := loadTurnsFromRollout(filepath.Join("..", "..", "..", "tests", "fixtures", "codex", "rollout-new-sample.jsonl"), 100)
	if err != nil {
		t.Fatalf("new load: %v", err)
	}
	if len(newTurns) != 1 || newTurns[0].AssistantText == "" || len(newTurns[0].ToolCalls) != 1 {
		t.Fatalf("unexpected new turns: %#v", newTurns)
	}
}

func TestLoadTurnsFromRolloutLegacyCustomToolsAndTokenCount(t *testing.T) {
	turns, usage, err := loadTurnsFromRollout(filepath.Join("..", "..", "..", "tests", "fixtures", "codex", "rollout-legacy-rich-sample.jsonl"), 100)
	if err != nil {
		t.Fatalf("rich legacy load: %v", err)
	}
	if len(turns) != 1 {
		t.Fatalf("expected one turn, got %#v", turns)
	}
	turn := turns[0]
	if len(turn.ToolCalls) != 2 {
		t.Fatalf("expected function and custom tool calls, got %#v", turn.ToolCalls)
	}
	if turn.ToolCalls[1].Name != "apply_patch" || !turn.ToolCalls[1].IsError || turn.ToolCalls[1].Output != "patch failed" {
		t.Fatalf("custom tool output not normalized: %#v", turn.ToolCalls[1])
	}
	if turn.TokenUsage.InputTokens != 10 || turn.TokenUsage.OutputTokens != 3 || turn.TokenUsage.CacheTokens != 2 {
		t.Fatalf("turn usage mismatch: %#v", turn.TokenUsage)
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 30 || usage.CacheTokens != 20 {
		t.Fatalf("total usage mismatch: %#v", usage)
	}
}

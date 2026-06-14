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
	legacy, err := loadTurnsFromRollout(filepath.Join("..", "..", "..", "tests", "fixtures", "codex", "rollout-legacy-sample.jsonl"), 100)
	if err != nil {
		t.Fatalf("legacy load: %v", err)
	}
	if len(legacy) != 1 || legacy[0].UserText == "" || len(legacy[0].ToolCalls) != 1 {
		t.Fatalf("unexpected legacy turns: %#v", legacy)
	}

	newTurns, err := loadTurnsFromRollout(filepath.Join("..", "..", "..", "tests", "fixtures", "codex", "rollout-new-sample.jsonl"), 100)
	if err != nil {
		t.Fatalf("new load: %v", err)
	}
	if len(newTurns) != 1 || newTurns[0].AssistantText == "" || len(newTurns[0].ToolCalls) != 1 {
		t.Fatalf("unexpected new turns: %#v", newTurns)
	}
}

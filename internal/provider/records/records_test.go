package records

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/yaleh/meta-cc/internal/conversation"
)

func TestNormalizeCodexTurnRecords(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	session := conversation.Session{
		ID:       "codex-session",
		Provider: conversation.ProviderCodex,
		CWD:      "/tmp/project",
		Model:    "gpt-5",
		TokenUsage: conversation.TokenUsage{
			InputTokens: 999,
		},
	}
	turns := []conversation.Turn{{
		ID:            "turn-1",
		UserText:      "hello",
		AssistantText: "ack",
		TokenUsage: conversation.TokenUsage{
			InputTokens:  10,
			OutputTokens: 3,
			CacheTokens:  2,
		},
		ToolCalls: []conversation.ToolCall{{
			ID:        "call-1",
			Name:      "apply_patch",
			Input:     json.RawMessage(`{"input":"patch"}`),
			Output:    "failed",
			IsError:   true,
			Timestamp: now,
		}},
		Timestamp: now,
	}}

	got := Normalize(session, turns)
	if len(got) != 3 {
		t.Fatalf("expected user, assistant, and tool result records, got %#v", got)
	}
	assistant, ok := got[1]["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("assistant message missing: %#v", got[1])
	}
	usage, ok := assistant["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("assistant usage missing: %#v", assistant)
	}
	if usage["input_tokens"] != 10 || usage["output_tokens"] != 3 || usage["cache_tokens"] != 2 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	resultContent := got[2]["message"].(map[string]interface{})["content"].([]interface{})
	result := resultContent[0].(map[string]interface{})
	if result["status"] != "error" || result["error"] != "failed" {
		t.Fatalf("tool error result not normalized: %#v", result)
	}
}

func TestNormalizeCodexSessionTokenUsageDoesNotCreateUsageRecord(t *testing.T) {
	session := conversation.Session{
		ID:       "codex-session",
		Provider: conversation.ProviderCodex,
		CWD:      "/tmp/project",
		TokenUsage: conversation.TokenUsage{
			InputTokens: 999,
		},
	}
	turns := []conversation.Turn{{
		ID:            "turn-1",
		AssistantText: "ack",
		Timestamp:     time.Unix(1700000000, 0).UTC(),
	}}

	got := Normalize(session, turns)
	message := got[0]["message"].(map[string]interface{})
	if _, ok := message["usage"]; ok {
		t.Fatalf("codex sqlite tokens_used should not become per-turn usage: %#v", message)
	}
}

func TestNormalizeToolInputNullBecomesEmptyMap(t *testing.T) {
	session := conversation.Session{
		ID:       "codex-session",
		Provider: conversation.ProviderCodex,
		CWD:      "/tmp/project",
	}
	turns := []conversation.Turn{{
		ID: "turn-1",
		ToolCalls: []conversation.ToolCall{{
			ID:    "call-null",
			Name:  "empty_input",
			Input: json.RawMessage(`null`),
		}},
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}}

	got := Normalize(session, turns)
	message := got[0]["message"].(map[string]interface{})
	content := message["content"].([]interface{})
	toolUse := content[0].(map[string]interface{})
	input, ok := toolUse["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected non-nil input map, got %#v", toolUse["input"])
	}
	if len(input) != 0 {
		t.Fatalf("expected empty input map, got %#v", input)
	}
}

func TestFilterSessionsForScopeDoesNotMutateInput(t *testing.T) {
	sessions := []conversation.Session{
		{ID: "keep", Provider: conversation.ProviderCodex, CWD: "/project", CreatedAt: time.Unix(2, 0)},
		{ID: "drop", Provider: conversation.ProviderCodex, CWD: "/other", CreatedAt: time.Unix(1, 0)},
	}
	originalSecond := sessions[1]

	filtered := FilterSessionsForScope(sessions, "project", "/project", conversation.ProviderCodex)

	if len(filtered) != 1 || filtered[0].ID != "keep" {
		t.Fatalf("unexpected filtered sessions: %#v", filtered)
	}
	if sessions[1].ID != originalSecond.ID || sessions[1].CWD != originalSecond.CWD || !sessions[1].CreatedAt.Equal(originalSecond.CreatedAt) {
		t.Fatalf("FilterSessionsForScope mutated input slice: got %#v, want %#v", sessions[1], originalSecond)
	}
}

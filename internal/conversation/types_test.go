package conversation

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRoundTripSession(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	want := Session{
		ID:        "sess-1",
		Provider:  ProviderCodex,
		Title:     "title",
		CWD:       "/tmp/project",
		Model:     "gpt-5",
		CreatedAt: now,
		TokenUsage: TokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
			CacheTokens:  3,
		},
		Turns: []Turn{{
			ID:        "turn-1",
			UserText:  "hello",
			Timestamp: now,
		}},
		Extensions: json.RawMessage(`{"rollout_path":"x"}`),
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}

	var got Session
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	if got.ID != want.ID || got.Provider != want.Provider || string(got.Extensions) != string(want.Extensions) {
		t.Fatalf("session mismatch: %#v", got)
	}
}

func TestRoundTripTurnAndToolCall(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	want := Turn{
		ID:            "turn-1",
		UserText:      "user",
		AssistantText: "assistant",
		Timestamp:     now,
		ToolCalls: []ToolCall{{
			ID:        "call-1",
			Name:      "exec_command",
			Input:     json.RawMessage(`{"cmd":"pwd"}`),
			Output:    "/tmp",
			IsError:   false,
			Timestamp: now,
		}},
		Extensions: json.RawMessage(`{"x":1}`),
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal turn: %v", err)
	}

	var got Turn
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal turn: %v", err)
	}

	if got.ID != want.ID || got.ToolCalls[0].Name != "exec_command" || string(got.Extensions) != `{"x":1}` {
		t.Fatalf("turn mismatch: %#v", got)
	}
}

func TestProviderConstantsAndOmitempty(t *testing.T) {
	if ProviderClaude != "claude" || ProviderCodex != "codex" {
		t.Fatalf("unexpected provider constants")
	}

	data, err := json.Marshal(Session{
		ID:        "sess-1",
		Provider:  ProviderClaude,
		CWD:       "/tmp",
		CreatedAt: time.Unix(1700000000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["turns"]; ok {
		t.Fatalf("turns should be omitted")
	}
	if _, ok := raw["extensions"]; ok {
		t.Fatalf("extensions should be omitted")
	}
}

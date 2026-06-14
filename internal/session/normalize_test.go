package session

import (
	"encoding/json"
	"testing"
)

func TestNormalizerConvertsCodexMessagesAndTools(t *testing.T) {
	normalizer := NewNormalizer()
	lines := []string{
		`{"timestamp":"2026-06-14T06:00:00Z","type":"session_meta","payload":{"id":"codex-session","cwd":"/tmp/project","model":"gpt-5"}}`,
		`{"timestamp":"2026-06-14T06:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"fix codex support"}]}}`,
		`{"timestamp":"2026-06-14T06:00:02Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","call_id":"call_1","arguments":"{\"cmd\":\"go test ./...\",\"workdir\":\"/tmp/project\"}"}}`,
		`{"timestamp":"2026-06-14T06:00:03Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_1","output":"ok"}}`,
		`{"timestamp":"2026-06-14T06:00:04Z","type":"response_item","payload":{"type":"custom_tool_call","name":"apply_patch","call_id":"call_2","input":"*** Begin Patch\n*** End Patch"}}`,
		`{"timestamp":"2026-06-14T06:00:05Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_2","status":"failed","output":"patch failed"}}`,
	}

	var entries []map[string]interface{}
	for _, line := range lines {
		normalized, err := normalizer.NormalizeLine([]byte(line))
		if err != nil {
			t.Fatalf("NormalizeLine failed: %v", err)
		}
		entries = append(entries, normalized...)
	}

	if len(entries) != 5 {
		t.Fatalf("expected 5 normalized entries, got %d", len(entries))
	}
	seenUUIDs := map[string]bool{}
	for _, entry := range entries {
		uuid, _ := entry["uuid"].(string)
		if uuid == "" {
			t.Fatalf("expected normalized entry uuid: %#v", entry)
		}
		if seenUUIDs[uuid] {
			t.Fatalf("normalized entry uuid is not unique: %s", uuid)
		}
		seenUUIDs[uuid] = true
	}

	user := entries[0]
	if user["type"] != "user" {
		t.Fatalf("expected user entry, got %v", user["type"])
	}
	if user["sessionId"] != "codex-session" || user["cwd"] != "/tmp/project" {
		t.Fatalf("expected session context to be preserved, got sessionId=%v cwd=%v", user["sessionId"], user["cwd"])
	}
	userMsg := user["message"].(map[string]interface{})
	if userMsg["content"] != "fix codex support" {
		t.Fatalf("expected user content string, got %#v", userMsg["content"])
	}

	toolUse := contentBlock(t, entries[1])
	if toolUse["type"] != "tool_use" || toolUse["name"] != "exec_command" || toolUse["id"] != "call_1" {
		t.Fatalf("unexpected function_call mapping: %#v", toolUse)
	}
	input := toolUse["input"].(map[string]interface{})
	if input["cmd"] != "go test ./..." {
		t.Fatalf("expected parsed arguments input, got %#v", input)
	}

	toolResult := contentBlock(t, entries[2])
	if toolResult["type"] != "tool_result" || toolResult["tool_use_id"] != "call_1" {
		t.Fatalf("unexpected function_call_output mapping: %#v", toolResult)
	}
	if toolResult["is_error"] != false {
		t.Fatalf("expected successful output, got %#v", toolResult)
	}

	customTool := contentBlock(t, entries[3])
	if customTool["type"] != "tool_use" || customTool["name"] != "apply_patch" {
		t.Fatalf("unexpected custom_tool_call mapping: %#v", customTool)
	}
	customInput := customTool["input"].(map[string]interface{})
	if customInput["input"] == "" {
		t.Fatalf("expected raw custom tool input fallback, got %#v", customInput)
	}

	failedResult := contentBlock(t, entries[4])
	if failedResult["is_error"] != true || failedResult["status"] != "error" {
		t.Fatalf("expected failed custom tool output to become error result, got %#v", failedResult)
	}
}

func TestNormalizerLeavesClaudeRecordUnchanged(t *testing.T) {
	normalizer := NewNormalizer()
	raw := []byte(`{"type":"user","sessionId":"claude-session","message":{"role":"user","content":"hello"}}`)

	entries, err := normalizer.NormalizeLine(raw)
	if err != nil {
		t.Fatalf("NormalizeLine failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}

	var original map[string]interface{}
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatalf("unmarshal original: %v", err)
	}
	if entries[0]["sessionId"] != original["sessionId"] {
		t.Fatalf("expected original Claude record to pass through, got %#v", entries[0])
	}
}

func contentBlock(t *testing.T, entry map[string]interface{}) map[string]interface{} {
	t.Helper()
	message := entry["message"].(map[string]interface{})
	content := message["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected one content block, got %d", len(content))
	}
	block := content[0].(map[string]interface{})
	return block
}

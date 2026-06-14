package parser

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/yaleh/meta-cc/internal/testutil"
)

func TestParseSession_ValidFile(t *testing.T) {
	// 使用测试 fixture（包含 4 行：1 个 file-history-snapshot + 3 个消息）
	filePath := testutil.FixtureDir() + "/sample-session.jsonl"

	parser := NewSessionParser(filePath)
	entries, err := parser.ParseEntries()

	if err != nil {
		t.Fatalf("Failed to parse session: %v", err)
	}

	// 应该只返回消息类型（过滤掉 file-history-snapshot）
	expectedEntries := 3
	if len(entries) != expectedEntries {
		t.Errorf("Expected %d message entries, got %d", expectedEntries, len(entries))
	}

	// 验证第一个条目（user）
	entry0 := entries[0]
	if entry0.Type != "user" {
		t.Errorf("Expected type 'user', got '%s'", entry0.Type)
	}
	if entry0.UUID != "cfef2966-a593-4169-9956-ee24c804b717" {
		t.Errorf("Unexpected UUID: %s", entry0.UUID)
	}
	if entry0.Message == nil {
		t.Fatal("Expected Message to be non-nil")
	}
	if entry0.Message.Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", entry0.Message.Role)
	}

	// 验证第二个条目（assistant with tool）
	entry1 := entries[1]
	if entry1.Type != "assistant" {
		t.Errorf("Expected type 'assistant', got '%s'", entry1.Type)
	}
	if entry1.Message == nil {
		t.Fatal("Expected Message to be non-nil")
	}
	if len(entry1.Message.Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(entry1.Message.Content))
	}

	// 验证工具调用
	hasToolUse := false
	for _, block := range entry1.Message.Content {
		if block.Type == "tool_use" && block.ToolUse != nil {
			hasToolUse = true
			if block.ToolUse.Name != "Grep" {
				t.Errorf("Expected tool name 'Grep', got '%s'", block.ToolUse.Name)
			}
		}
	}
	if !hasToolUse {
		t.Error("Expected tool_use in entry 1")
	}

	// 验证第三个条目（tool result）
	entry2 := entries[2]
	if entry2.Type != "user" {
		t.Errorf("Expected type 'user', got '%s'", entry2.Type)
	}
	if entry2.Message == nil {
		t.Fatal("Expected Message to be non-nil")
	}
	if len(entry2.Message.Content) < 1 {
		t.Fatal("Expected at least 1 content block")
	}
	if entry2.Message.Content[0].Type != "tool_result" {
		t.Errorf("Expected type 'tool_result', got '%s'", entry2.Message.Content[0].Type)
	}
}

func TestParseEntriesFromContent_CodexJSONL(t *testing.T) {
	content := strings.Join([]string{
		`{"timestamp":"2026-06-14T06:00:00Z","type":"session_meta","payload":{"id":"codex-session","cwd":"/tmp/project"}}`,
		`{"timestamp":"2026-06-14T06:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"codex parity"}]}}`,
		`{"timestamp":"2026-06-14T06:00:02Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","call_id":"call_1","arguments":"{\"cmd\":\"go test ./...\",\"workdir\":\"/tmp/project\"}"}}`,
		`{"timestamp":"2026-06-14T06:00:03Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_1","output":"ok"}}`,
	}, "\n")

	entries, err := ParseEntriesFromContent(content)
	if err != nil {
		t.Fatalf("ParseEntriesFromContent failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 message entries, got %d", len(entries))
	}
	if entries[0].Type != "user" || entries[0].Message.Content[0].Text != "codex parity" {
		t.Fatalf("unexpected user entry: %#v", entries[0])
	}

	toolCalls := ExtractToolCalls(entries)
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].ToolName != "exec_command" || toolCalls[0].Input["cmd"] != "go test ./..." {
		t.Fatalf("unexpected tool call: %#v", toolCalls[0])
	}
}

func TestParseSession_EmptyFile(t *testing.T) {
	tempFile := testutil.TempSessionFile(t, "")

	parser := NewSessionParser(tempFile)
	entries, err := parser.ParseEntries()

	if err != nil {
		t.Fatalf("Expected no error for empty file, got: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty file, got %d", len(entries))
	}
}

func TestParseSession_InvalidJSON(t *testing.T) {
	content := `{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":[]},"uuid":"abc"}
invalid json line
{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":[]},"uuid":"def"}`

	tempFile := testutil.TempSessionFile(t, content)

	parser := NewSessionParser(tempFile)
	_, err := parser.ParseEntries()

	if err == nil {
		t.Error("Expected error for invalid JSON line")
	}
}

func TestParseSession_SkipEmptyLines(t *testing.T) {
	content := `{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":[]},"uuid":"abc"}

{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":[]},"uuid":"def"}

`

	tempFile := testutil.TempSessionFile(t, content)

	parser := NewSessionParser(tempFile)
	entries, err := parser.ParseEntries()

	if err != nil {
		t.Fatalf("Failed to parse session with empty lines: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries (empty lines skipped), got %d", len(entries))
	}
}

func TestParseSession_FileNotFound(t *testing.T) {
	parser := NewSessionParser("/nonexistent/file.jsonl")
	_, err := parser.ParseEntries()

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseSession_FilterNonMessageTypes(t *testing.T) {
	// 测试过滤非消息类型（如 file-history-snapshot）
	content := `{"type":"file-history-snapshot","messageId":"abc","snapshot":{}}
{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":[]},"uuid":"user1"}
{"type":"some-other-type","data":"ignored"}
{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":[]},"uuid":"asst1"}`

	tempFile := testutil.TempSessionFile(t, content)

	parser := NewSessionParser(tempFile)
	entries, err := parser.ParseEntries()

	// 应该只返回 user 和 assistant 类型
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 message entries (non-message types filtered), got %d", len(entries))
	}

	// 验证都是消息类型
	for _, entry := range entries {
		if !entry.IsMessage() {
			t.Errorf("Expected only message types, got '%s'", entry.Type)
		}
	}
}

func TestParseEntriesFromContent_ValidContent(t *testing.T) {
	content := `{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":"Hello"},"uuid":"user1"}
{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":"Hi"},"uuid":"asst1"}`

	entries, err := ParseEntriesFromContent(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	if entries[0].Type != "user" {
		t.Errorf("Expected first entry type 'user', got '%s'", entries[0].Type)
	}

	if entries[1].Type != "assistant" {
		t.Errorf("Expected second entry type 'assistant', got '%s'", entries[1].Type)
	}
}

func TestParseEntriesFromContent_EmptyContent(t *testing.T) {
	entries, err := ParseEntriesFromContent("")

	if err != nil {
		t.Fatalf("Expected no error for empty content, got: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries for empty content, got %d", len(entries))
	}
}

func TestParseEntriesFromContent_SkipEmptyLines(t *testing.T) {
	content := `{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":"Test"},"uuid":"user1"}

{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":"Response"},"uuid":"asst1"}

`

	entries, err := ParseEntriesFromContent(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries (empty lines skipped), got %d", len(entries))
	}
}

func TestParseEntriesFromContent_InvalidJSON(t *testing.T) {
	content := `{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":"Test"},"uuid":"user1"}
invalid json line
{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":"Response"},"uuid":"asst1"}`

	_, err := ParseEntriesFromContent(content)

	if err == nil {
		t.Error("Expected error for invalid JSON line")
	}
}

func TestParseEntriesFromContent_FilterNonMessageTypes(t *testing.T) {
	content := `{"type":"file-history-snapshot","messageId":"abc","snapshot":{}}
{"type":"user","timestamp":"2025-10-02T06:07:13.673Z","message":{"role":"user","content":"Test"},"uuid":"user1"}
{"type":"some-other-type","data":"ignored"}
{"type":"assistant","timestamp":"2025-10-02T06:08:57.769Z","message":{"role":"assistant","content":"Response"},"uuid":"asst1"}`

	entries, err := ParseEntriesFromContent(content)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 message entries (non-message types filtered), got %d", len(entries))
	}

	for _, entry := range entries {
		if !entry.IsMessage() {
			t.Errorf("Expected only message types, got '%s'", entry.Type)
		}
	}
}

func TestParseEntries_LargeImageLine_NotSkipped(t *testing.T) {
	// Build a ~4MB+ base64 image line (exceeds LargeLineWarnBytes=4MB) followed by a normal entry
	rawData := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 768*1024) // 3MB binary → ~4MB base64
	b64data := base64.StdEncoding.EncodeToString(rawData)

	imageLine := fmt.Sprintf(
		`{"type":"user","uuid":"image-entry-uuid","message":{"role":"user","content":[{"type":"tool_result","content":[{"type":"image","source":{"type":"base64","media_type":"image/png","data":"%s"}}]}]}}`,
		b64data,
	)
	normalLine := `{"type":"user","uuid":"normal-entry-uuid","message":{"role":"user","content":[{"type":"text","text":"hello after image"}]}}`

	content := imageLine + "\n" + normalLine + "\n"
	tmpFile := testutil.TempSessionFile(t, content)

	p := NewSessionParser(tmpFile)
	entries, err := p.ParseEntries()
	if err != nil {
		t.Fatalf("ParseEntries should not fail on large image line, got: %v", err)
	}

	// Both lines are user messages; the image line has its data stripped but is still valid
	found := false
	for _, e := range entries {
		if e.UUID == "normal-entry-uuid" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected normal entry after image line to be present")
	}
	if len(entries) < 2 {
		t.Errorf("Expected at least 2 entries (image + normal), got %d", len(entries))
	}
}

package types_test

import (
	"encoding/json"
	"testing"

	"github.com/yaleh/meta-cc/internal/types"
)

func TestSessionEntry_IsMessage(t *testing.T) {
	tests := []struct {
		entryType string
		want      bool
	}{
		{"user", true},
		{"assistant", true},
		{"file-history-snapshot", false},
		{"", false},
	}
	for _, tt := range tests {
		e := &types.SessionEntry{Type: tt.entryType}
		if got := e.IsMessage(); got != tt.want {
			t.Errorf("IsMessage() for type %q = %v, want %v", tt.entryType, got, tt.want)
		}
	}
}

func TestMessage_JSONRoundtrip_StringContent(t *testing.T) {
	raw := `{"role":"user","content":"hello world"}`
	var m types.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(m.Content) != 1 || m.Content[0].Type != "text" || m.Content[0].Text != "hello world" {
		t.Errorf("unexpected content: %+v", m.Content)
	}
}

func TestMessage_JSONRoundtrip_ArrayContent(t *testing.T) {
	raw := `{"role":"assistant","content":[{"type":"text","text":"hi"}]}`
	var m types.Message
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(m.Content) != 1 || m.Content[0].Text != "hi" {
		t.Errorf("unexpected content: %+v", m.Content)
	}
	data, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var m2 types.Message
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatalf("re-Unmarshal error: %v", err)
	}
	if len(m2.Content) != 1 || m2.Content[0].Text != "hi" {
		t.Errorf("round-trip content mismatch: %+v", m2.Content)
	}
}

func TestToolResult_StringContent(t *testing.T) {
	raw := `{"tool_use_id":"abc","content":"output text","is_error":false}`
	var tr types.ToolResult
	if err := json.Unmarshal([]byte(raw), &tr); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if tr.Content != "output text" {
		t.Errorf("expected 'output text', got %q", tr.Content)
	}
}

func TestToolResult_ArrayContent(t *testing.T) {
	raw := `{"tool_use_id":"abc","content":[{"type":"text","text":"line1"},{"type":"text","text":"line2"}],"is_error":false}`
	var tr types.ToolResult
	if err := json.Unmarshal([]byte(raw), &tr); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if tr.Content != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got %q", tr.Content)
	}
}

func TestToolResult_ErrorContent(t *testing.T) {
	raw := `{"tool_use_id":"abc","content":"error msg","is_error":true}`
	var tr types.ToolResult
	if err := json.Unmarshal([]byte(raw), &tr); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if tr.Error != "error msg" {
		t.Errorf("expected Error='error msg', got %q", tr.Error)
	}
}

func TestContentBlock_ToolUse(t *testing.T) {
	raw := `{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}`
	var cb types.ContentBlock
	if err := json.Unmarshal([]byte(raw), &cb); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if cb.ToolUse == nil || cb.ToolUse.Name != "Bash" {
		t.Errorf("expected ToolUse.Name='Bash', got %+v", cb.ToolUse)
	}
	// Round-trip
	data, err := json.Marshal(cb)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var cb2 types.ContentBlock
	if err := json.Unmarshal(data, &cb2); err != nil {
		t.Fatalf("re-Unmarshal error: %v", err)
	}
	if cb2.ToolUse == nil || cb2.ToolUse.Name != "Bash" {
		t.Errorf("round-trip mismatch: %+v", cb2)
	}
}

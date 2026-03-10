package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SessionEntry represents a single entry in a Claude Code session file.
// It can be a user message, assistant message, or other type (e.g., file-history-snapshot).
type SessionEntry struct {
	Type       string   `json:"type"`       // "user", "assistant", "file-history-snapshot", etc.
	Timestamp  string   `json:"timestamp"`  // ISO 8601: "2025-10-02T06:07:13.673Z"
	UUID       string   `json:"uuid"`       // unique entry identifier
	ParentUUID string   `json:"parentUuid"` // parent entry UUID (for chain building)
	SessionID  string   `json:"sessionId"`  // session identifier
	CWD        string   `json:"cwd"`        // working directory
	Version    string   `json:"version"`    // Claude Code version
	GitBranch  string   `json:"gitBranch"`  // git branch
	Message    *Message `json:"message"`    // message content (only for user/assistant types)
}

// IsMessage reports whether the entry is a message type (user or assistant).
func (e *SessionEntry) IsMessage() bool {
	return e.Type == "user" || e.Type == "assistant"
}

// Message represents message content.
type Message struct {
	ID         string                 `json:"id"`          // message ID (set for assistant messages)
	Role       string                 `json:"role"`        // "user" or "assistant"
	Model      string                 `json:"model"`       // model name (set for assistant messages)
	Content    []ContentBlock         `json:"-"`           // content blocks (custom JSON handling)
	StopReason string                 `json:"stop_reason"` // stop reason
	Usage      map[string]interface{} `json:"usage"`       // token usage stats
}

// UnmarshalJSON handles custom deserialization: content can be a string or []ContentBlock.
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		ContentRaw json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.ContentRaw) == 0 {
		return nil
	}

	// Try as string first
	var contentStr string
	if err := json.Unmarshal(aux.ContentRaw, &contentStr); err == nil {
		m.Content = []ContentBlock{
			{Type: "text", Text: contentStr},
		}
		return nil
	}

	// Otherwise parse as array
	return json.Unmarshal(aux.ContentRaw, &m.Content)
}

// MarshalJSON ensures Content is serialized correctly.
func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(&struct {
		Content []ContentBlock `json:"content"`
		*Alias
	}{
		Content: m.Content,
		Alias:   (*Alias)(m),
	})
}

// ContentBlock represents one content block in a message.
// It can be text, a tool call, or a tool result.
type ContentBlock struct {
	Type       string      `json:"type"`
	Text       string      `json:"text,omitempty"`
	ToolUse    *ToolUse    `json:"-"` // custom serialization
	ToolResult *ToolResult `json:"-"` // custom serialization
}

// ToolUse represents a tool invocation request.
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool invocation.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"-"` // custom handling (can be string or array)
	IsError   bool   `json:"is_error"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
}

// UnmarshalJSON handles custom deserialization: content can be a string or array.
func (tr *ToolResult) UnmarshalJSON(data []byte) error {
	type Alias ToolResult
	aux := &struct {
		ContentRaw json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(tr),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.ContentRaw) == 0 {
		return nil
	}

	// Try as string
	var contentStr string
	if err := json.Unmarshal(aux.ContentRaw, &contentStr); err == nil {
		tr.Content = contentStr
		if tr.IsError && tr.Error == "" {
			tr.Error = contentStr
		}
		return nil
	}

	// Otherwise parse as array of text blocks
	var contentBlocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(aux.ContentRaw, &contentBlocks); err != nil {
		return fmt.Errorf("failed to unmarshal tool_result content: %w", err)
	}

	var texts []string
	for _, block := range contentBlocks {
		if block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	tr.Content = strings.Join(texts, "\n")

	if tr.IsError && tr.Error == "" {
		tr.Error = tr.Content
	}

	return nil
}

// UnmarshalJSON handles custom deserialization for ContentBlock.
func (cb *ContentBlock) UnmarshalJSON(data []byte) error {
	type Alias ContentBlock
	aux := &struct {
		*Alias
		RawToolUse    json.RawMessage `json:"tool_use,omitempty"`
		RawToolResult json.RawMessage `json:"tool_result,omitempty"`
	}{
		Alias: (*Alias)(cb),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return fmt.Errorf("failed to unmarshal ContentBlock: %w", err)
	}

	switch cb.Type {
	case "text":
		// already handled by default deserialization

	case "tool_use":
		type ToolUseBlock struct {
			Type  string                 `json:"type"`
			ID    string                 `json:"id"`
			Name  string                 `json:"name"`
			Input map[string]interface{} `json:"input"`
		}
		var tub ToolUseBlock
		if err := json.Unmarshal(data, &tub); err != nil {
			return fmt.Errorf("failed to unmarshal tool_use: %w", err)
		}
		cb.ToolUse = &ToolUse{
			ID:    tub.ID,
			Name:  tub.Name,
			Input: tub.Input,
		}

	case "tool_result":
		var toolResult ToolResult
		if err := json.Unmarshal(data, &toolResult); err != nil {
			return fmt.Errorf("failed to unmarshal tool_result: %w", err)
		}
		cb.ToolResult = &toolResult

	default:
		// unknown type — retain type field, no error
	}

	return nil
}

// MarshalJSON handles custom serialization for ContentBlock.
func (cb ContentBlock) MarshalJSON() ([]byte, error) {
	switch cb.Type {
	case "text":
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{Type: cb.Type, Text: cb.Text})

	case "tool_use":
		if cb.ToolUse == nil {
			return nil, fmt.Errorf("tool_use type but ToolUse is nil")
		}
		return json.Marshal(struct {
			Type  string                 `json:"type"`
			ID    string                 `json:"id"`
			Name  string                 `json:"name"`
			Input map[string]interface{} `json:"input"`
		}{
			Type:  cb.Type,
			ID:    cb.ToolUse.ID,
			Name:  cb.ToolUse.Name,
			Input: cb.ToolUse.Input,
		})

	case "tool_result":
		if cb.ToolResult == nil {
			return nil, fmt.Errorf("tool_result type but ToolResult is nil")
		}
		return json.Marshal(struct {
			Type      string `json:"type"`
			ToolUseID string `json:"tool_use_id"`
			Content   string `json:"content"`
			IsError   bool   `json:"is_error"`
			Status    string `json:"status,omitempty"`
			Error     string `json:"error,omitempty"`
		}{
			Type:      cb.Type,
			ToolUseID: cb.ToolResult.ToolUseID,
			Content:   cb.ToolResult.Content,
			IsError:   cb.ToolResult.IsError,
			Status:    cb.ToolResult.Status,
			Error:     cb.ToolResult.Error,
		})

	default:
		return json.Marshal(struct {
			Type string `json:"type"`
		}{Type: cb.Type})
	}
}

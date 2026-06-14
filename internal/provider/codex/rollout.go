package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/yaleh/meta-cc/internal/conversation"
)

type schemaVersion int

const (
	schemaLegacy schemaVersion = iota
	schemaNew
)

func loadTurnsFromSession(session conversation.Session, maxLines int) ([]conversation.Turn, error) {
	var ext struct {
		RolloutPath string `json:"rollout_path"`
	}
	if err := json.Unmarshal(session.Extensions, &ext); err != nil {
		return nil, err
	}
	if ext.RolloutPath == "" {
		return nil, fmt.Errorf("missing rollout_path for session %s", session.ID)
	}
	return loadTurnsFromRollout(ext.RolloutPath, maxLines)
}

func loadTurnsFromRollout(path string, maxLines int) ([]conversation.Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	var (
		lineCount int
		version   schemaVersion
		detected  bool
		builder   = newTurnBuilder()
	)

	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		lineCount++
		if lineCount > maxLines {
			slog.Warn("codex rollout truncated", "path", path, "max_lines", maxLines)
			break
		}
		if !detected {
			version = detectSchemaVersion(line)
			detected = true
		}
		if version == schemaNew {
			builder.applyNew(line)
		} else {
			builder.applyLegacy(line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	builder.flush()
	return builder.turns, nil
}

func detectSchemaVersion(firstLine []byte) schemaVersion {
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(firstLine, &payload); err == nil && strings.Contains(payload.Type, ".") {
		return schemaNew
	}
	return schemaLegacy
}

type turnBuilder struct {
	current     *conversation.Turn
	toolCallMap map[string]int
	turns       []conversation.Turn
	unknown     []json.RawMessage
}

func newTurnBuilder() *turnBuilder {
	return &turnBuilder{toolCallMap: make(map[string]int)}
}

func (b *turnBuilder) flush() {
	if b.current != nil && len(b.unknown) > 0 {
		ext, _ := json.Marshal(map[string][]json.RawMessage{
			"codex_events": b.unknown,
		})
		b.current.Extensions = ext
	}
	if b.current != nil && (b.current.UserText != "" || b.current.AssistantText != "" || len(b.current.ToolCalls) > 0 || len(b.current.Extensions) > 0) {
		b.turns = append(b.turns, *b.current)
	}
	b.current = nil
	b.toolCallMap = make(map[string]int)
	b.unknown = nil
}

func (b *turnBuilder) ensureTurn(id, timestamp string) {
	if b.current == nil {
		ts, _ := time.Parse(time.RFC3339, timestamp)
		b.current = &conversation.Turn{ID: id, Timestamp: ts.UTC()}
	}
}

func (b *turnBuilder) applyLegacy(line []byte) {
	var event struct {
		Timestamp string          `json:"timestamp"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &event); err != nil {
		return
	}

	switch event.Type {
	case "session_meta":
		return
	case "turn_context":
		var payload struct {
			TurnID string `json:"turn_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if b.current != nil && payload.TurnID != "" && b.current.ID != payload.TurnID {
			b.flush()
		}
		b.ensureTurn(payload.TurnID, event.Timestamp)
	case "event_msg":
		var payload struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			TurnID  string `json:"turn_id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if payload.Type == "task_started" {
			b.flush()
			b.ensureTurn(payload.TurnID, event.Timestamp)
			return
		}
		b.ensureTurn(payload.TurnID, event.Timestamp)
		switch payload.Type {
		case "user_message":
			b.current.UserText = payload.Message
		case "agent_message":
			if b.current.AssistantText != "" {
				b.current.AssistantText += "\n"
			}
			b.current.AssistantText += payload.Message
		case "token_count":
			return
		default:
			b.appendUnknown(line)
		}
	case "response_item":
		var envelope struct {
			Type      string          `json:"type"`
			Name      string          `json:"name"`
			Arguments string          `json:"arguments"`
			CallID    string          `json:"call_id"`
			Output    string          `json:"output"`
			Role      string          `json:"role"`
			Content   json.RawMessage `json:"content"`
		}
		_ = json.Unmarshal(event.Payload, &envelope)
		b.ensureTurn("", event.Timestamp)
		switch envelope.Type {
		case "function_call":
			ts, _ := time.Parse(time.RFC3339, event.Timestamp)
			call := conversation.ToolCall{
				ID:        envelope.CallID,
				Name:      envelope.Name,
				Input:     json.RawMessage(envelope.Arguments),
				Timestamp: ts.UTC(),
			}
			b.toolCallMap[envelope.CallID] = len(b.current.ToolCalls)
			b.current.ToolCalls = append(b.current.ToolCalls, call)
		case "function_call_output":
			if idx, ok := b.toolCallMap[envelope.CallID]; ok {
				b.current.ToolCalls[idx].Output = envelope.Output
			}
		case "message":
			text := extractResponseItemText(envelope.Content)
			switch envelope.Role {
			case "user":
				if b.current.UserText == "" {
					b.current.UserText = text
				}
			case "assistant":
				if b.current.AssistantText != "" && text != "" {
					b.current.AssistantText += "\n"
				}
				b.current.AssistantText += text
			case "developer", "system":
				return
			default:
				b.appendUnknown(line)
			}
		case "reasoning":
			return
		default:
			b.appendUnknown(line)
		}
	default:
		b.appendUnknown(line)
	}
}

func (b *turnBuilder) applyNew(line []byte) {
	var event struct {
		Timestamp string          `json:"timestamp"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &event); err != nil {
		return
	}

	switch event.Type {
	case "thread.started":
		return
	case "turn.started":
		var payload struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		b.flush()
		b.ensureTurn(payload.ID, event.Timestamp)
	case "item.message":
		var payload struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		b.ensureTurn("", event.Timestamp)
		if payload.Role == "user" {
			b.current.UserText = payload.Content
		} else if payload.Role == "assistant" {
			b.current.AssistantText = payload.Content
		}
	case "item.tool_call":
		var payload struct {
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		b.ensureTurn("", event.Timestamp)
		ts, _ := time.Parse(time.RFC3339, event.Timestamp)
		b.toolCallMap[payload.ID] = len(b.current.ToolCalls)
		b.current.ToolCalls = append(b.current.ToolCalls, conversation.ToolCall{
			ID:        payload.ID,
			Name:      payload.Name,
			Input:     payload.Input,
			Timestamp: ts.UTC(),
		})
	case "item.tool_result":
		var payload struct {
			ID      string `json:"id"`
			Output  string `json:"output"`
			IsError bool   `json:"is_error"`
		}
		_ = json.Unmarshal(event.Payload, &payload)
		if idx, ok := b.toolCallMap[payload.ID]; ok {
			b.current.ToolCalls[idx].Output = payload.Output
			b.current.ToolCalls[idx].IsError = payload.IsError
		}
	case "turn.completed":
		b.flush()
	case "turn.failed", "error":
		b.appendUnknown(line)
	default:
		b.appendUnknown(line)
	}
}

func (b *turnBuilder) appendUnknown(line []byte) {
	b.ensureTurn("", time.Now().UTC().Format(time.RFC3339))
	b.unknown = append(b.unknown, append(json.RawMessage(nil), line...))
}

func extractResponseItemText(content json.RawMessage) string {
	var items []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(content, &items); err != nil {
		return ""
	}
	var parts []string
	for _, item := range items {
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	return strings.Join(parts, "\n")
}

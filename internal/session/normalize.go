package session

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Normalizer converts supported host transcript records into the Claude-like
// schema used by the rest of meta-cc.
type Normalizer struct {
	SessionID  string
	CWD        string
	Model      string
	parentUUID string
	seq        int
}

// NewNormalizer creates a stateful normalizer for one JSONL file.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// NormalizeLine parses one JSONL object and returns zero or more normalized
// records. Claude Code records are returned unchanged. Codex response_item
// records are converted to Claude-like user/assistant/tool records.
func (n *Normalizer) NormalizeLine(raw []byte) ([]map[string]interface{}, error) {
	var record map[string]interface{}
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return n.NormalizeRecord(record), nil
}

// NormalizeRecord converts a decoded record into the common schema.
func (n *Normalizer) NormalizeRecord(record map[string]interface{}) []map[string]interface{} {
	if n == nil {
		n = NewNormalizer()
	}

	recordType, _ := record["type"].(string)
	if !isCodexRecord(recordType) {
		return []map[string]interface{}{record}
	}

	n.captureCodexContext(record)
	if recordType != "response_item" {
		return nil
	}

	payload, ok := asMap(record["payload"])
	if !ok {
		return nil
	}

	timestamp, _ := record["timestamp"].(string)
	payloadType, _ := payload["type"].(string)
	switch payloadType {
	case "message":
		return n.normalizeCodexMessage(timestamp, payload)
	case "function_call", "custom_tool_call":
		return []map[string]interface{}{n.normalizeCodexToolUse(timestamp, payload, payloadType)}
	case "function_call_output", "custom_tool_call_output":
		return []map[string]interface{}{n.normalizeCodexToolResult(timestamp, payload)}
	default:
		return nil
	}
}

func (n *Normalizer) captureCodexContext(record map[string]interface{}) {
	payload, _ := asMap(record["payload"])
	recordType, _ := record["type"].(string)

	if recordType == "session_meta" {
		if id, ok := firstString(payload, "id", "session_id", "sessionId"); ok {
			n.SessionID = id
		}
		if cwd, ok := firstString(payload, "cwd", "working_dir", "workingDir"); ok {
			n.CWD = cwd
		}
		if model, ok := firstString(payload, "model"); ok {
			n.Model = model
		}
		return
	}

	if recordType == "turn_context" {
		if cwd, ok := firstString(payload, "cwd", "working_dir", "workingDir"); ok {
			n.CWD = cwd
		}
		if model, ok := firstString(payload, "model"); ok {
			n.Model = model
		}
	}
}

func (n *Normalizer) normalizeCodexMessage(timestamp string, payload map[string]interface{}) []map[string]interface{} {
	role, _ := payload["role"].(string)
	text := codexText(payload["content"])

	switch role {
	case "user":
		return []map[string]interface{}{n.baseEntry("user", timestamp, map[string]interface{}{
			"role":    "user",
			"content": text,
		}, payload)}
	case "assistant":
		content := make([]interface{}, 0, 1)
		if text != "" {
			content = append(content, map[string]interface{}{
				"type": "text",
				"text": text,
			})
		}
		return []map[string]interface{}{n.baseEntry("assistant", timestamp, map[string]interface{}{
			"role":    "assistant",
			"model":   n.Model,
			"content": content,
		}, payload)}
	case "developer", "system":
		return []map[string]interface{}{n.baseEntry("system", timestamp, map[string]interface{}{
			"role":    role,
			"content": text,
		}, payload)}
	default:
		return nil
	}
}

func (n *Normalizer) normalizeCodexToolUse(timestamp string, payload map[string]interface{}, payloadType string) map[string]interface{} {
	callID, _ := firstString(payload, "call_id", "callId", "id")
	name, _ := firstString(payload, "name")

	input := map[string]interface{}{}
	if payloadType == "function_call" {
		input = parseInput(payload["arguments"], "arguments")
	} else {
		input = parseInput(payload["input"], "input")
	}

	return n.baseEntry("assistant", timestamp, map[string]interface{}{
		"role":  "assistant",
		"model": n.Model,
		"content": []interface{}{
			map[string]interface{}{
				"type":  "tool_use",
				"id":    callID,
				"name":  name,
				"input": input,
			},
		},
	}, payload)
}

func (n *Normalizer) normalizeCodexToolResult(timestamp string, payload map[string]interface{}) map[string]interface{} {
	callID, _ := firstString(payload, "call_id", "callId", "id")
	output := stringify(payload["output"])
	status, _ := firstString(payload, "status")
	isError, _ := payload["is_error"].(bool)
	if status != "" && status != "completed" && status != "success" {
		isError = true
	}
	if errText, ok := firstString(payload, "error"); ok {
		isError = true
		if output == "" {
			output = errText
		}
	}

	resultStatus := "success"
	if isError {
		resultStatus = "error"
	} else if status != "" {
		resultStatus = status
	}

	block := map[string]interface{}{
		"type":        "tool_result",
		"tool_use_id": callID,
		"content":     output,
		"is_error":    isError,
		"status":      resultStatus,
	}
	if isError {
		block["error"] = output
	}

	return n.baseEntry("user", timestamp, map[string]interface{}{
		"role":    "user",
		"content": []interface{}{block},
	}, payload)
}

func (n *Normalizer) baseEntry(entryType, timestamp string, message map[string]interface{}, payload map[string]interface{}) map[string]interface{} {
	uuid := n.nextUUID(payload)
	entry := map[string]interface{}{
		"type":       entryType,
		"timestamp":  timestamp,
		"uuid":       uuid,
		"parentUuid": n.parentUUID,
		"sessionId":  n.SessionID,
		"cwd":        n.CWD,
		"message":    message,
	}
	n.parentUUID = uuid
	return entry
}

func (n *Normalizer) nextUUID(payload map[string]interface{}) string {
	n.seq++
	id, _ := firstString(payload, "id", "call_id", "callId")
	payloadType, _ := firstString(payload, "type")
	seed := fmt.Sprintf("%s:%s:%s:%s:%d", n.SessionID, n.CWD, payloadType, id, n.seq)
	if id == "" && payloadType == "" {
		seed = fmt.Sprintf("%s:%s:%d", n.SessionID, n.CWD, n.seq)
	}
	sum := sha1.Sum([]byte(seed))
	return "codex-" + hex.EncodeToString(sum[:8])
}

func isCodexRecord(recordType string) bool {
	switch recordType {
	case "session_meta", "event_msg", "response_item", "turn_context", "compacted":
		return true
	default:
		return false
	}
}

func codexText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			m, ok := asMap(item)
			if !ok {
				continue
			}
			if text, ok := firstString(m, "text", "input_text", "output_text"); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return stringify(content)
	}
}

func parseInput(raw interface{}, fallbackKey string) map[string]interface{} {
	switch v := raw.(type) {
	case nil:
		return map[string]interface{}{}
	case map[string]interface{}:
		return v
	case string:
		if strings.TrimSpace(v) == "" {
			return map[string]interface{}{}
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return parsed
		}
		return map[string]interface{}{fallbackKey: v}
	default:
		return map[string]interface{}{fallbackKey: v}
	}
}

func asMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}

func firstString(m map[string]interface{}, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := m[key].(string); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

func stringify(v interface{}) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case []byte:
		return string(value)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprint(value)
		}
		return string(data)
	}
}

// ProjectPathInRecord reports whether a normalized or raw record references
// the project path. It is intentionally conservative and only used by tests
// and future callers that need structured project matching.
func ProjectPathInRecord(record map[string]interface{}, projectPath string) bool {
	if projectPath == "" {
		return false
	}
	clean := filepath.Clean(projectPath)
	if cwd, ok := record["cwd"].(string); ok && filepath.Clean(cwd) == clean {
		return true
	}
	payload, _ := asMap(record["payload"])
	if cwd, ok := firstString(payload, "cwd", "working_dir", "workingDir"); ok && filepath.Clean(cwd) == clean {
		return true
	}
	return false
}

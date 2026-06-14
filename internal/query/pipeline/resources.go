package pipeline

import (
	"fmt"

	"github.com/yaleh/meta-cc/internal/types"
)

// MessageView represents a flattened message view.
type MessageView struct {
	UUID          string               `json:"uuid"`
	SessionID     string               `json:"session_id"`
	ParentUUID    string               `json:"parent_uuid"`
	Timestamp     string               `json:"timestamp"`
	Role          string               `json:"role"`
	Content       string               `json:"content,omitempty"`
	ContentBlocks []types.ContentBlock `json:"content_blocks"`
	GitBranch     string               `json:"git_branch,omitempty"`
}

// SelectResource selects the appropriate resource view based on resource type.
func SelectResource(entries []types.SessionEntry, resource string) (interface{}, error) {
	switch resource {
	case "entries":
		return entries, nil
	case "messages":
		return extractMessages(entries), nil
	case "tools":
		return types.ExtractToolCalls(entries), nil
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resource)
	}
}

func extractMessages(entries []types.SessionEntry) []MessageView {
	var messages []MessageView
	for _, entry := range entries {
		if entry.Message == nil {
			continue
		}
		if entry.Type != "user" && entry.Type != "assistant" {
			continue
		}
		msg := MessageView{
			UUID:          entry.UUID,
			SessionID:     entry.SessionID,
			ParentUUID:    entry.ParentUUID,
			Timestamp:     entry.Timestamp,
			Role:          entry.Message.Role,
			ContentBlocks: entry.Message.Content,
			GitBranch:     entry.GitBranch,
		}
		msg.Content = extractTextContent(entry.Message.Content)
		messages = append(messages, msg)
	}
	return messages
}

func extractTextContent(blocks []types.ContentBlock) string {
	var text string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			if text != "" {
				text += "\n"
			}
			text += block.Text
		}
	}
	return text
}

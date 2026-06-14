package claude

import (
	"encoding/json"
	"time"

	"github.com/yaleh/meta-cc/internal/conversation"
	"github.com/yaleh/meta-cc/internal/types"
)

type turnPair struct {
	user      *types.SessionEntry
	assistant *types.SessionEntry
}

func buildTurns(entries []types.SessionEntry) []turnPair {
	var pairs []turnPair
	assistantByParent := make(map[string]*types.SessionEntry)
	for i := range entries {
		if entries[i].Type == "assistant" && entries[i].ParentUUID != "" {
			assistantByParent[entries[i].ParentUUID] = &entries[i]
		}
	}

	for i := range entries {
		if entries[i].Type != "user" {
			continue
		}
		pairs = append(pairs, turnPair{
			user:      &entries[i],
			assistant: assistantByParent[entries[i].UUID],
		})
	}
	return pairs
}

func joinToolCalls(pair turnPair) []conversation.ToolCall {
	if pair.assistant == nil || pair.assistant.Message == nil {
		return nil
	}

	results := make(map[string]*types.ToolResult)
	if pair.user != nil && pair.user.Message != nil {
		for _, block := range pair.user.Message.Content {
			if block.Type == "tool_result" && block.ToolResult != nil {
				results[block.ToolResult.ToolUseID] = block.ToolResult
			}
		}
	}

	var calls []conversation.ToolCall
	for _, block := range pair.assistant.Message.Content {
		if block.Type != "tool_use" || block.ToolUse == nil {
			continue
		}
		input, _ := json.Marshal(block.ToolUse.Input)
		ts, _ := time.Parse(time.RFC3339, pair.assistant.Timestamp)
		call := conversation.ToolCall{
			ID:        block.ToolUse.ID,
			Name:      block.ToolUse.Name,
			Input:     input,
			Timestamp: ts.UTC(),
		}
		if result := results[block.ToolUse.ID]; result != nil {
			call.Output = result.Content
			call.IsError = result.IsError || result.Status == "error" || result.Error != ""
		}
		calls = append(calls, call)
	}
	return calls
}

func entryText(entry *types.SessionEntry) string {
	if entry == nil || entry.Message == nil {
		return ""
	}
	var text string
	for _, block := range entry.Message.Content {
		if block.Type == "text" && block.Text != "" {
			if text != "" {
				text += "\n"
			}
			text += block.Text
		}
	}
	return text
}

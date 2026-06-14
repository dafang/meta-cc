package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaleh/meta-cc/internal/types"
)

func TestSelectResource(t *testing.T) {
	entries := createTestEntries()

	tests := []struct {
		name         string
		resource     string
		wantMinCount int
		wantErr      bool
	}{
		{
			name:         "select_entries",
			resource:     "entries",
			wantMinCount: 3,
			wantErr:      false,
		},
		{
			name:         "select_messages",
			resource:     "messages",
			wantMinCount: 2,
			wantErr:      false,
		},
		{
			name:         "select_tools",
			resource:     "tools",
			wantMinCount: 1,
			wantErr:      false,
		},
		{
			name:     "invalid_resource",
			resource: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SelectResource(entries, tt.resource)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			var count int
			switch tt.resource {
			case "entries":
				got, ok := result.([]types.SessionEntry)
				require.True(t, ok, "Result should be []types.SessionEntry")
				count = len(got)
			case "messages":
				messages, ok := result.([]MessageView)
				require.True(t, ok, "Result should be []MessageView")
				count = len(messages)
			case "tools":
				tools, ok := result.([]types.ToolCall)
				require.True(t, ok, "Result should be []types.ToolCall")
				count = len(tools)
			}

			assert.GreaterOrEqual(t, count, tt.wantMinCount,
				"Expected at least %d results, got %d", tt.wantMinCount, count)
		})
	}
}

func TestExtractMessages(t *testing.T) {
	entries := createTestEntries()

	messages := extractMessages(entries)

	require.NotEmpty(t, messages)

	for _, msg := range messages {
		assert.NotEmpty(t, msg.UUID)
		assert.NotEmpty(t, msg.SessionID)
		assert.NotEmpty(t, msg.Timestamp)
		assert.Contains(t, []string{"user", "assistant"}, msg.Role)
		assert.NotNil(t, msg.ContentBlocks)
	}

	var hasUser, hasAssistant bool
	for _, msg := range messages {
		if msg.Role == "user" {
			hasUser = true
		}
		if msg.Role == "assistant" {
			hasAssistant = true
		}
	}
	assert.True(t, hasUser, "Should have at least one user message")
	assert.True(t, hasAssistant, "Should have at least one assistant message")
}

func TestExtractToolExecutions(t *testing.T) {
	entries := createTestEntries()

	tools := types.ExtractToolCalls(entries)

	require.NotEmpty(t, tools)

	for _, tool := range tools {
		assert.NotEmpty(t, tool.UUID)
		assert.NotEmpty(t, tool.ToolName)
		assert.NotEmpty(t, tool.Timestamp)
		assert.NotNil(t, tool.Input)
	}

	var hasRead bool
	for _, tool := range tools {
		if tool.ToolName == "Read" {
			hasRead = true
			assert.Equal(t, "success", tool.Status)
			assert.Equal(t, "file content", tool.Output)
		}
	}
	assert.True(t, hasRead, "Should have Read tool execution")
}

func TestMessageView(t *testing.T) {
	entries := createTestEntries()
	messages := extractMessages(entries)

	require.NotEmpty(t, messages)

	msg := messages[0]

	assert.NotEmpty(t, msg.UUID, "UUID should not be empty")
	assert.NotEmpty(t, msg.SessionID, "SessionID should not be empty")
	assert.NotEmpty(t, msg.Timestamp, "Timestamp should not be empty")
	assert.NotEmpty(t, msg.Role, "Role should not be empty")
	assert.NotEmpty(t, msg.ContentBlocks, "ContentBlocks should not be empty")

	if msg.Role == "user" {
		foundText := false
		for _, block := range msg.ContentBlocks {
			if block.Type == "text" && block.Text != "" {
				foundText = true
				break
			}
		}
		assert.True(t, foundText || len(msg.ContentBlocks) > 0,
			"User message should have text content or at least one content block")
	}
}

func TestExtractMessagesEmptyInput(t *testing.T) {
	messages := extractMessages([]types.SessionEntry{})
	assert.Empty(t, messages, "Should return empty slice for empty input")
}

func TestExtractToolExecutionsEmptyInput(t *testing.T) {
	tools := types.ExtractToolCalls([]types.SessionEntry{})
	assert.Empty(t, tools, "Should return empty slice for empty input")
}

func TestExtractMessagesNoMessages(t *testing.T) {
	entries := []types.SessionEntry{
		{
			Type:      "summary",
			UUID:      "summary-1",
			Timestamp: "2025-10-23T00:00:00Z",
		},
	}

	messages := extractMessages(entries)
	assert.Empty(t, messages, "Should return empty slice when no message entries")
}

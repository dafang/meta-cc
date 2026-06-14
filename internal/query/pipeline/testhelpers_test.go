package pipeline

import (
	"fmt"

	"github.com/yaleh/meta-cc/internal/types"
)

func createTestEntries() []types.SessionEntry {
	return []types.SessionEntry{
		{
			Type:       "user",
			UUID:       "user-1",
			Timestamp:  "2025-10-23T00:00:00Z",
			SessionID:  "session-1",
			ParentUUID: "parent-1",
			GitBranch:  "main",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: "Read the file",
					},
				},
			},
		},
		{
			Type:       "assistant",
			UUID:       "assistant-1",
			Timestamp:  "2025-10-23T00:01:00Z",
			SessionID:  "session-1",
			ParentUUID: "user-1",
			GitBranch:  "main",
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   "tool-1",
							Name: "Read",
							Input: map[string]interface{}{
								"file_path": "/test/file.go",
							},
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "user-2",
			Timestamp:  "2025-10-23T00:02:00Z",
			SessionID:  "session-1",
			ParentUUID: "assistant-1",
			GitBranch:  "main",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: "tool-1",
							Content:   "file content",
							IsError:   false,
							Status:    "success",
						},
					},
				},
			},
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func createComplexTestEntries() []types.SessionEntry {
	return []types.SessionEntry{
		{
			Type:       "user",
			UUID:       "user-1",
			Timestamp:  "2025-10-23T10:00:00Z",
			SessionID:  "session-1",
			ParentUUID: "",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: "Read the configuration file",
					},
				},
			},
		},
		{
			Type:       "assistant",
			UUID:       "assistant-1",
			Timestamp:  "2025-10-23T10:00:10Z",
			SessionID:  "session-1",
			ParentUUID: "user-1",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   "tool-read-1",
							Name: "Read",
							Input: map[string]interface{}{
								"file_path": "/home/user/project/config.yaml",
							},
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "tool-result-1",
			Timestamp:  "2025-10-23T10:00:15Z",
			SessionID:  "session-1",
			ParentUUID: "assistant-1",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: "tool-read-1",
							Content:   "server: localhost\nport: 8080",
							IsError:   false,
							Status:    "success",
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "user-2",
			Timestamp:  "2025-10-23T10:01:00Z",
			SessionID:  "session-1",
			ParentUUID: "tool-result-1",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: "Try to read a non-existent file",
					},
				},
			},
		},
		{
			Type:       "assistant",
			UUID:       "assistant-2",
			Timestamp:  "2025-10-23T10:01:10Z",
			SessionID:  "session-1",
			ParentUUID: "user-2",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   "tool-read-2",
							Name: "Read",
							Input: map[string]interface{}{
								"file_path": "/home/user/project/nonexistent.txt",
							},
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "tool-result-2",
			Timestamp:  "2025-10-23T10:01:15Z",
			SessionID:  "session-1",
			ParentUUID: "assistant-2",
			GitBranch:  "main",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: "tool-read-2",
							Content:   "",
							IsError:   true,
							Status:    "error",
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "user-3",
			Timestamp:  "2025-10-23T10:02:00Z",
			SessionID:  "session-1",
			ParentUUID: "tool-result-2",
			GitBranch:  "feature/new-feature",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: "Write to a file",
					},
				},
			},
		},
		{
			Type:       "assistant",
			UUID:       "assistant-3",
			Timestamp:  "2025-10-23T10:02:10Z",
			SessionID:  "session-1",
			ParentUUID: "user-3",
			GitBranch:  "feature/new-feature",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   "tool-edit-1",
							Name: "Edit",
							Input: map[string]interface{}{
								"file_path":  "/home/user/project/README.md",
								"old_string": "# Old Title",
								"new_string": "# New Title",
							},
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "tool-result-3",
			Timestamp:  "2025-10-23T10:02:15Z",
			SessionID:  "session-1",
			ParentUUID: "assistant-3",
			GitBranch:  "feature/new-feature",
			CWD:        "/home/user/project",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: "tool-edit-1",
							Content:   "File edited successfully",
							IsError:   false,
							Status:    "success",
						},
					},
				},
			},
		},
		{
			Type:       "user",
			UUID:       "user-4",
			Timestamp:  "2025-10-23T11:00:00Z",
			SessionID:  "session-2",
			ParentUUID: "",
			GitBranch:  "main",
			CWD:        "/home/user/other",
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: "Different session query",
					},
				},
			},
		},
	}
}

func generateTestEntries(n int) []types.SessionEntry {
	entries := make([]types.SessionEntry, 0, n)

	for i := 0; i < n; i++ {
		sessionID := fmt.Sprintf("session-%d", i%10)
		gitBranch := "main"
		if i%5 == 0 {
			gitBranch = "feature/branch"
		}

		entries = append(entries, types.SessionEntry{
			Type:       "user",
			UUID:       fmt.Sprintf("user-%d", i),
			Timestamp:  fmt.Sprintf("2025-10-23T%02d:%02d:00Z", i/60, i%60),
			SessionID:  sessionID,
			ParentUUID: fmt.Sprintf("parent-%d", i-1),
			GitBranch:  gitBranch,
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "text",
						Text: fmt.Sprintf("User message %d", i),
					},
				},
			},
		})

		toolName := "Read"
		if i%3 == 0 {
			toolName = "Edit"
		} else if i%5 == 0 {
			toolName = "Write"
		}

		entries = append(entries, types.SessionEntry{
			Type:       "assistant",
			UUID:       fmt.Sprintf("assistant-%d", i),
			Timestamp:  fmt.Sprintf("2025-10-23T%02d:%02d:05Z", i/60, i%60),
			SessionID:  sessionID,
			ParentUUID: fmt.Sprintf("user-%d", i),
			GitBranch:  gitBranch,
			Message: &types.Message{
				Role: "assistant",
				Content: []types.ContentBlock{
					{
						Type: "tool_use",
						ToolUse: &types.ToolUse{
							ID:   fmt.Sprintf("tool-%d", i),
							Name: toolName,
							Input: map[string]interface{}{
								"file_path": fmt.Sprintf("/test/file%d.txt", i),
							},
						},
					},
				},
			},
		})

		status := "success"
		if i%10 == 0 {
			status = "error"
		}

		entries = append(entries, types.SessionEntry{
			Type:       "user",
			UUID:       fmt.Sprintf("tool-result-%d", i),
			Timestamp:  fmt.Sprintf("2025-10-23T%02d:%02d:10Z", i/60, i%60),
			SessionID:  sessionID,
			ParentUUID: fmt.Sprintf("assistant-%d", i),
			GitBranch:  gitBranch,
			Message: &types.Message{
				Role: "user",
				Content: []types.ContentBlock{
					{
						Type: "tool_result",
						ToolResult: &types.ToolResult{
							ToolUseID: fmt.Sprintf("tool-%d", i),
							Content:   fmt.Sprintf("Result for tool %d", i),
							IsError:   status == "error",
							Status:    status,
						},
					},
				},
			},
		})
	}

	return entries
}

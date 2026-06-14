package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/yaleh/meta-cc/internal/conversation"
	"github.com/yaleh/meta-cc/internal/locator"
	mcquery "github.com/yaleh/meta-cc/internal/mcp/query"
	providerpkg "github.com/yaleh/meta-cc/internal/provider"
	claudeprovider "github.com/yaleh/meta-cc/internal/provider/claude"
	codexprovider "github.com/yaleh/meta-cc/internal/provider/codex"
)

func (e *ToolExecutor) ExecuteQueryForProvider(providerName, scope, jqFilter string, limit int, workingDir string) (mcquery.QueryResult, error) {
	return e.ExecuteQueryWithTimeRangeForProvider(providerName, scope, jqFilter, limit, workingDir, mcquery.ParsedTimeRange{})
}

func (e *ToolExecutor) ExecuteQueryWithTimeRangeForProvider(providerName, scope, jqFilter string, limit int, workingDir string, tr mcquery.ParsedTimeRange) (mcquery.QueryResult, error) {
	if providerName == "" || providerName == "claude" {
		return e.ExecuteQueryWithTimeRange(scope, jqFilter, limit, workingDir, tr)
	}

	projectPath := workingDir
	if projectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return mcquery.QueryResult{}, err
		}
		projectPath = cwd
	}
	projectPath, _ = filepath.Abs(projectPath)

	registry := providerpkg.NewRegistry(
		claudeprovider.NewProvider(locator.NewSessionLocator(), projectPath),
		codexprovider.NewProvider(locator.NewCodexLocator()),
	)

	filters, err := parseProviderFilter(providerName)
	if err != nil {
		return mcquery.QueryResult{}, err
	}
	records, warnings, err := buildProviderRecords(context.Background(), registry, filters, scope, projectPath)
	if err != nil {
		return mcquery.QueryResult{}, err
	}
	results, err := runProviderJQ(records, jqFilter, limit, tr)
	if err != nil {
		return mcquery.QueryResult{}, err
	}
	return mcquery.QueryResult{Entries: results, Warnings: warnings}, nil
}

func parseProviderFilter(providerName string) ([]conversation.ProviderID, error) {
	switch providerName {
	case "claude":
		return []conversation.ProviderID{conversation.ProviderClaude}, nil
	case "codex":
		return []conversation.ProviderID{conversation.ProviderCodex}, nil
	case "all":
		return []conversation.ProviderID{conversation.ProviderClaude, conversation.ProviderCodex}, nil
	default:
		return nil, fmt.Errorf("invalid provider %q: must be \"claude\", \"codex\", or \"all\"", providerName)
	}
}

func buildProviderRecords(ctx context.Context, registry *providerpkg.Registry, filters []conversation.ProviderID, scope, projectPath string) ([]map[string]interface{}, []string, error) {
	var (
		records  []map[string]interface{}
		warnings []string
	)

	for _, p := range registry.Providers(filters) {
		if !p.IsAvailable(ctx) {
			warnings = append(warnings, fmt.Sprintf("provider %s unavailable", p.ID()))
			continue
		}
		sessions, err := p.ListSessions(ctx)
		if err != nil {
			return nil, warnings, err
		}
		sessions = filterSessionsForScope(sessions, scope, projectPath, p.ID())
		for _, session := range sessions {
			turns, err := p.LoadTurns(ctx, session.ID)
			if err != nil {
				return nil, warnings, err
			}
			records = append(records, normalizedRecords(session, turns)...)
		}
	}
	return records, warnings, nil
}

func filterSessionsForScope(sessions []conversation.Session, scope, projectPath string, providerID conversation.ProviderID) []conversation.Session {
	filtered := sessions[:0]
	for _, session := range sessions {
		if providerID == conversation.ProviderCodex && scope == "project" && session.CWD != "" && projectPath != "" && session.CWD != projectPath {
			continue
		}
		filtered = append(filtered, session)
	}
	if scope != "session" || len(filtered) <= 1 {
		return filtered
	}
	slices.SortFunc(filtered, func(a, b conversation.Session) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return 0
	})
	return filtered[:1]
}

func normalizedRecords(session conversation.Session, turns []conversation.Turn) []map[string]interface{} {
	var records []map[string]interface{}
	for _, turn := range turns {
		ts := turn.Timestamp.Format(time.RFC3339)
		if turn.UserText != "" {
			records = append(records, map[string]interface{}{
				"type":       "user",
				"provider":   session.Provider,
				"session_id": session.ID,
				"cwd":        session.CWD,
				"timestamp":  ts,
				"message": map[string]interface{}{
					"role":    "user",
					"content": turn.UserText,
				},
			})
		}
		if turn.AssistantText != "" || len(turn.ToolCalls) > 0 {
			content := make([]interface{}, 0, len(turn.ToolCalls)+1)
			if turn.AssistantText != "" {
				content = append(content, map[string]interface{}{"type": "text", "text": turn.AssistantText})
			}
			for _, call := range turn.ToolCalls {
				var input map[string]interface{}
				_ = json.Unmarshal(call.Input, &input)
				content = append(content, map[string]interface{}{
					"type":  "tool_use",
					"id":    call.ID,
					"name":  call.Name,
					"input": input,
				})
			}
			records = append(records, map[string]interface{}{
				"type":       "assistant",
				"provider":   session.Provider,
				"session_id": session.ID,
				"cwd":        session.CWD,
				"timestamp":  ts,
				"message": map[string]interface{}{
					"role":    "assistant",
					"model":   session.Model,
					"usage":   map[string]interface{}{"input_tokens": session.TokenUsage.InputTokens, "output_tokens": session.TokenUsage.OutputTokens, "cache_tokens": session.TokenUsage.CacheTokens},
					"content": content,
				},
			})
		}
		var toolResults []interface{}
		for _, call := range turn.ToolCalls {
			if call.Output == "" && !call.IsError {
				continue
			}
			entry := map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": call.ID,
				"content":     call.Output,
			}
			if call.IsError {
				entry["is_error"] = true
				entry["status"] = "error"
				entry["error"] = call.Output
			}
			toolResults = append(toolResults, entry)
		}
		if len(toolResults) > 0 {
			records = append(records, map[string]interface{}{
				"type":       "user",
				"provider":   session.Provider,
				"session_id": session.ID,
				"cwd":        session.CWD,
				"timestamp":  ts,
				"message": map[string]interface{}{
					"role":    "user",
					"content": toolResults,
				},
			})
		}
	}
	return records
}

func runProviderJQ(records []map[string]interface{}, jqFilter string, limit int, tr mcquery.ParsedTimeRange) ([]interface{}, error) {
	executor := mcquery.NewQueryExecutor("")
	code, err := executor.CompileExpression(jqFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression: %w", err)
	}

	var out []interface{}
	for _, record := range records {
		if !inTimeRange(record["timestamp"], tr) {
			continue
		}
		iter := code.Run(record)
		for {
			value, ok := iter.Next()
			if !ok {
				break
			}
			if _, isErr := value.(error); isErr {
				continue
			}
			out = append(out, value)
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
	}
	return out, nil
}

func inTimeRange(raw interface{}, tr mcquery.ParsedTimeRange) bool {
	if tr.Since == nil && tr.Until == nil {
		return true
	}
	ts, ok := raw.(string)
	if !ok {
		return true
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return true
	}
	if tr.Since != nil && t.Before(*tr.Since) {
		return false
	}
	if tr.Until != nil && !t.Before(*tr.Until) {
		return false
	}
	return true
}

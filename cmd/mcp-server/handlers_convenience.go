package main

import (
	"github.com/yaleh/meta-cc/internal/config"
	execpkg "github.com/yaleh/meta-cc/internal/mcp/executor"
)

// handlers_convenience.go provides lowercase wrappers for test backward compatibility.
// Business logic lives in internal/mcp/executor/handlers.go.

func (e *ToolExecutor) handleQueryUserMessages(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryUserMessages(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryTools(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryTools(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryToolErrors(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryToolErrors(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryTokenUsage(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryTokenUsage(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryConversationFlow(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryConversationFlow(cfg, scope, args)
}

func (e *ToolExecutor) handleQuerySystemErrors(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQuerySystemErrors(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryFileSnapshots(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryFileSnapshots(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryTimestamps(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryTimestamps(cfg, scope, args)
}

func (e *ToolExecutor) handleQuerySummaries(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQuerySummaries(cfg, scope, args)
}

func (e *ToolExecutor) handleQueryToolBlocks(cfg *config.Config, scope string, args map[string]interface{}) (QueryResult, error) {
	return e.ToolExecutor.HandleQueryToolBlocks(cfg, scope, args)
}

// escapeJQ delegates to internal executor package
func escapeJQ(s string) string {
	return execpkg.EscapeJQ(s)
}

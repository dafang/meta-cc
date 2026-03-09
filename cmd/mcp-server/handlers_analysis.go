package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yaleh/meta-cc/internal/analyzer"
	"github.com/yaleh/meta-cc/internal/config"
	"github.com/yaleh/meta-cc/internal/locator"
	"github.com/yaleh/meta-cc/internal/parser"
)

// loadEntriesAndToolCalls is a shared helper that locates session files,
// parses them, and extracts tool calls. It supports "project" (default)
// and "session" scopes, and an optional working_dir override.
func loadEntriesAndToolCalls(cfg *config.Config, args map[string]interface{}) ([]parser.SessionEntry, []parser.ToolCall, error) {
	// 1. Get scope from args["scope"], default "project"
	scope := "project"
	if s, ok := args["scope"].(string); ok && s != "" {
		scope = s
	}

	// 2. Get working dir
	workingDir := ""
	if w, ok := args["working_dir"].(string); ok {
		workingDir = w
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = "."
		}
	}

	// 3. Locate JSONL files using locator
	loc := locator.NewSessionLocator()
	var files []string
	if scope == "session" {
		sessionFile, err := loc.FromProjectPath(workingDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to locate session: %w", err)
		}
		files = []string{sessionFile}
	} else {
		var err error
		files, err = loc.AllSessionsFromProject(workingDir)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to locate project sessions: %w", err)
		}
	}

	// 4. Parse all files, accumulate entries
	var allEntries []parser.SessionEntry
	for _, f := range files {
		p := parser.NewSessionParser(f)
		entries, err := p.ParseEntries()
		if err != nil {
			continue // skip malformed files
		}
		allEntries = append(allEntries, entries...)
	}

	// 5. Extract tool calls
	toolCalls := parser.ExtractToolCalls(allEntries)
	return allEntries, toolCalls, nil
}

// executeAnalyzeErrorsTool implements the analyze_errors MCP tool.
// It aggregates tool errors by tool name and error type.
func executeAnalyzeErrorsTool(cfg *config.Config, args map[string]interface{}) (string, error) {
	entries, toolCalls, err := loadEntriesAndToolCalls(cfg, args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}

	limit := getIntParam(args, "limit", 0)

	result, err := analyzer.AnalyzeErrors(entries, toolCalls, limit)
	if err != nil {
		return "", fmt.Errorf("failed to analyze errors: %w", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(data), nil
}

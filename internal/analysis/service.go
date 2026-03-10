// Package analysis provides a service facade that encapsulates the full
// pipeline of: locate session files → parse → run analyzer functions.
// cmd/mcp-server uses this package instead of importing internal/parser
// and internal/analyzer directly.
package analysis

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yaleh/meta-cc/internal/analyzer"
	"github.com/yaleh/meta-cc/internal/locator"
	"github.com/yaleh/meta-cc/internal/parser"
	"github.com/yaleh/meta-cc/internal/types"
)

// Service encapsulates the analysis pipeline for MCP tool handlers.
type Service struct{}

// New creates a new Service.
func New() *Service {
	return &Service{}
}

// loadData locates session files, parses them, and extracts tool calls.
// It supports "project" (default) and "session" scopes, and an optional
// working_dir override extracted from args.
func (s *Service) loadData(args map[string]interface{}) ([]types.SessionEntry, []types.ToolCall, error) {
	scope := "project"
	if v, ok := args["scope"].(string); ok && v != "" {
		scope = v
	}

	workingDir := ""
	if v, ok := args["working_dir"].(string); ok {
		workingDir = v
	}
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = "."
		}
	}

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

	var allEntries []types.SessionEntry
	for _, f := range files {
		p := parser.NewSessionParser(f)
		entries, err := p.ParseEntries()
		if err != nil {
			continue // skip malformed files
		}
		allEntries = append(allEntries, entries...)
	}

	toolCalls := types.ExtractToolCalls(allEntries)
	return allEntries, toolCalls, nil
}

func intArg(args map[string]interface{}, key string) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return 0
}

func marshalResult(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(data), nil
}

// AnalyzeBugs implements the analyze_bugs MCP tool.
func (s *Service) AnalyzeBugs(args map[string]interface{}) (string, error) {
	entries, toolCalls, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.AnalyzeBugs(entries, toolCalls, intArg(args, "limit"))
	if err != nil {
		return "", fmt.Errorf("analyze bugs failed: %w", err)
	}
	return marshalResult(result)
}

// AnalyzeErrors implements the analyze_errors MCP tool.
func (s *Service) AnalyzeErrors(args map[string]interface{}) (string, error) {
	entries, toolCalls, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.AnalyzeErrors(entries, toolCalls, intArg(args, "limit"))
	if err != nil {
		return "", fmt.Errorf("failed to analyze errors: %w", err)
	}
	return marshalResult(result)
}

// QualityScan implements the quality_scan MCP tool.
func (s *Service) QualityScan(args map[string]interface{}) (string, error) {
	entries, toolCalls, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.QualityScan(entries, toolCalls)
	if err != nil {
		return "", fmt.Errorf("quality scan failed: %w", err)
	}
	return marshalResult(result)
}

// GetWorkPatterns implements the get_work_patterns MCP tool.
func (s *Service) GetWorkPatterns(args map[string]interface{}) (string, error) {
	entries, toolCalls, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.GetWorkPatterns(entries, toolCalls)
	if err != nil {
		return "", fmt.Errorf("get work patterns failed: %w", err)
	}
	return marshalResult(result)
}

// GetTimeline implements the get_timeline MCP tool.
func (s *Service) GetTimeline(args map[string]interface{}) (string, error) {
	entries, _, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.GetTimeline(entries, intArg(args, "limit"))
	if err != nil {
		return "", fmt.Errorf("get timeline failed: %w", err)
	}
	return marshalResult(result)
}

// GetTechDebt implements the get_tech_debt MCP tool.
func (s *Service) GetTechDebt(args map[string]interface{}) (string, error) {
	entries, toolCalls, err := s.loadData(args)
	if err != nil {
		return "", fmt.Errorf("failed to load session data: %w", err)
	}
	result, err := analyzer.GetTechDebt(entries, toolCalls)
	if err != nil {
		return "", fmt.Errorf("get tech debt failed: %w", err)
	}
	return marshalResult(result)
}

// AnalysisService is the interface implemented by *Service.
// It allows cmd/mcp-server to use a mock in tests.
type AnalysisService interface {
	AnalyzeBugs(args map[string]interface{}) (string, error)
	AnalyzeErrors(args map[string]interface{}) (string, error)
	QualityScan(args map[string]interface{}) (string, error)
	GetWorkPatterns(args map[string]interface{}) (string, error)
	GetTimeline(args map[string]interface{}) (string, error)
	GetTechDebt(args map[string]interface{}) (string, error)
}

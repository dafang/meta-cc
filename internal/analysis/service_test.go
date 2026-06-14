package analysis_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaleh/meta-cc/internal/analysis"
	"github.com/yaleh/meta-cc/internal/analyzer"
	"github.com/yaleh/meta-cc/internal/parser"
)

var _ analysis.AnalysisService = (*analysis.Service)(nil)

// ---------------------------------------------------------------------------
// Stub analyzer implementations: no real session files required
// ---------------------------------------------------------------------------

type stubErrorAnalyzer struct {
	result *analyzer.ErrorAnalysisResult
	err    error
}

func (s *stubErrorAnalyzer) AnalyzeErrors(_ []parser.SessionEntry, _ []parser.ToolCall, _ int) (*analyzer.ErrorAnalysisResult, error) {
	return s.result, s.err
}

type stubBugAnalyzer struct {
	result *analyzer.BugAnalysisResult
	err    error
}

func (s *stubBugAnalyzer) AnalyzeBugs(_ []parser.SessionEntry, _ []parser.ToolCall, _ int) (*analyzer.BugAnalysisResult, error) {
	return s.result, s.err
}

type stubQualityScanner struct {
	result *analyzer.QualityScanResult
	err    error
}

func (s *stubQualityScanner) QualityScan(_ []parser.SessionEntry, _ []parser.ToolCall) (*analyzer.QualityScanResult, error) {
	return s.result, s.err
}

type stubWorkPatternsAnalyzer struct {
	result *analyzer.WorkPatternsResult
	err    error
}

func (s *stubWorkPatternsAnalyzer) GetWorkPatterns(_ []parser.SessionEntry, _ []parser.ToolCall) (*analyzer.WorkPatternsResult, error) {
	return s.result, s.err
}

type stubTimelineAnalyzer struct {
	result *analyzer.TimelineResult
	err    error
}

func (s *stubTimelineAnalyzer) GetTimeline(_ []parser.SessionEntry, _ int) (*analyzer.TimelineResult, error) {
	return s.result, s.err
}

type stubTechDebtAnalyzer struct {
	result *analyzer.TechDebtResult
	err    error
}

func (s *stubTechDebtAnalyzer) GetTechDebt(_ []parser.SessionEntry, _ []parser.ToolCall) (*analyzer.TechDebtResult, error) {
	return s.result, s.err
}

// setupEmptyProjectDir creates a project directory with an empty session file so
// loadData returns an empty slice without error (locator requires at least one file).
func setupEmptyProjectDir(t *testing.T) string {
	t.Helper()
	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex-home"))
	projectPath := t.TempDir()
	absProject, err := filepath.Abs(projectPath)
	require.NoError(t, err)
	resolvedProject, err := filepath.EvalSymlinks(absProject)
	require.NoError(t, err)
	hash := strings.ReplaceAll(resolvedProject, "\\", "-")
	hash = strings.ReplaceAll(hash, "/", "-")
	hash = strings.ReplaceAll(hash, ":", "-")
	sessionDir := filepath.Join(projectsRoot, hash)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	// Create an empty JSONL file so the locator finds at least one session file.
	emptyFile := filepath.Join(sessionDir, "empty-session.jsonl")
	require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0o644))
	return projectPath
}

func TestService_AnalyzeBugs(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.BugAnalysisResult{
		TotalPairs: 2,
		Patterns: []analyzer.BugPattern{
			{ErrorSignature: "Read:file not found", FixCount: 2, Recurrences: 1, Examples: []string{"example"}},
		},
	}

	stub := &stubBugAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{BugAnalyzer: stub})

	out, err := svc.AnalyzeBugs(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "patterns")
	assert.Contains(t, result, "total_pairs")
	assert.Equal(t, float64(2), result["total_pairs"])
}

func TestService_AnalyzeErrors(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.ErrorAnalysisResult{
		TotalErrors: 5,
		ByTool: []analyzer.ToolErrorGroup{
			{ToolName: "Bash", Count: 3, Examples: []string{"exit 1"}},
			{ToolName: "Read", Count: 2, Examples: []string{"file not found"}},
		},
	}

	stub := &stubErrorAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{ErrorAnalyzer: stub})

	out, err := svc.AnalyzeErrors(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "by_tool")
}

func TestService_QualityScan(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.QualityScanResult{
		Dimensions: []analyzer.QualityDimension{
			{Name: "error_rate", Score: 0.9, RawValue: "1/10"},
			{Name: "retry_rate", Score: 0.8, RawValue: "2/10"},
		},
	}

	stub := &stubQualityScanner{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{QualityScanner: stub})

	out, err := svc.QualityScan(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "dimensions")
}

func TestService_GetWorkPatterns(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.WorkPatternsResult{
		ContextSwitches: 3,
		PeakHour:        14,
		ToolFrequency: []analyzer.ToolCount{
			{ToolName: "Bash", Count: 10},
		},
	}

	stub := &stubWorkPatternsAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{WorkPatterns: stub})

	out, err := svc.GetWorkPatterns(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	assert.NotEmpty(t, out)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "tool_frequency")
}

func TestService_GetTimeline(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.TimelineResult{
		TotalSpan: "1h30m",
		Events: []analyzer.TimelineEvent{
			{Type: "user_message", Summary: "Fix bug"},
		},
	}

	stub := &stubTimelineAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{Timeline: stub})

	out, err := svc.GetTimeline(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "events")
}

func TestService_GetTechDebt(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.TechDebtResult{
		OpenIssues: 2,
		Markers: []analyzer.MarkerCount{
			{Label: "TODO", Count: 2},
		},
	}

	stub := &stubTechDebtAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{TechDebt: stub})

	out, err := svc.GetTechDebt(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	assert.NotEmpty(t, out)
}

func TestService_WithStubErrorAnalyzer(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	expected := &analyzer.ErrorAnalysisResult{
		TotalErrors: 3,
		ByTool: []analyzer.ToolErrorGroup{
			{ToolName: "Bash", Count: 3, Examples: []string{"exit 1"}},
		},
	}

	stub := &stubErrorAnalyzer{result: expected}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{ErrorAnalyzer: stub})

	out, err := svc.AnalyzeErrors(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var got analyzer.ErrorAnalysisResult
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, 3, got.TotalErrors)
	require.Len(t, got.ByTool, 1)
	assert.Equal(t, "Bash", got.ByTool[0].ToolName)
}

func TestService_WithStubErrorAnalyzer_Error(t *testing.T) {
	projectPath := setupEmptyProjectDir(t)

	stub := &stubErrorAnalyzer{err: assert.AnError}
	svc := analysis.NewWithAnalyzers(analysis.Analyzers{ErrorAnalyzer: stub})

	_, err := svc.AnalyzeErrors(map[string]interface{}{"working_dir": projectPath})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to analyze errors")
}

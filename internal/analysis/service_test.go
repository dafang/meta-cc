package analysis_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaleh/meta-cc/internal/analysis"
	"github.com/yaleh/meta-cc/internal/analyzer"
	"github.com/yaleh/meta-cc/internal/types"
)

// setupTestProjectDir creates a fake META_CC_PROJECTS_ROOT with the given JSONL
// file under the hash-based directory for projectPath (replicates locator logic).
// Returns the projectPath to pass as working_dir.
func setupTestProjectDir(t *testing.T, sourceJSONL string) string {
	t.Helper()

	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)

	projectPath := t.TempDir()
	absProject, err := filepath.Abs(projectPath)
	require.NoError(t, err)
	hash := strings.ReplaceAll(absProject, "/", "-")

	sessionDir := filepath.Join(projectsRoot, hash)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))

	src, err := os.Open(sourceJSONL)
	require.NoError(t, err)
	defer src.Close()
	dst, err := os.Create(filepath.Join(sessionDir, "test-session.jsonl"))
	require.NoError(t, err)
	defer dst.Close()
	_, err = io.Copy(dst, src)
	require.NoError(t, err)

	return projectPath
}

var _ analysis.AnalysisService = (*analysis.Service)(nil)

func TestService_AnalyzeBugs(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.AnalyzeBugs(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "patterns")
	assert.Contains(t, result, "total_pairs")
}

func TestService_AnalyzeErrors(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.AnalyzeErrors(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "by_tool")
}

func TestService_QualityScan(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.QualityScan(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "dimensions")
}

func TestService_GetWorkPatterns(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.GetWorkPatterns(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.NotEmpty(t, out)
}

func TestService_GetTimeline(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.GetTimeline(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.Contains(t, result, "events")
}

func TestService_GetTechDebt(t *testing.T) {
	testJSONL, err := filepath.Abs("../../cmd/mcp-server/test.jsonl")
	require.NoError(t, err)
	if _, err := os.Stat(testJSONL); err != nil {
		t.Skip("test.jsonl not available")
	}

	projectPath := setupTestProjectDir(t, testJSONL)
	svc := analysis.New()
	out, err := svc.GetTechDebt(map[string]interface{}{"working_dir": projectPath})
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &result))
	assert.NotEmpty(t, out)
}

// ---------------------------------------------------------------------------
// Stub-based tests: no real session files required
// ---------------------------------------------------------------------------

type stubErrorAnalyzer struct {
	result *analyzer.ErrorAnalysisResult
	err    error
}

func (s *stubErrorAnalyzer) AnalyzeErrors(_ []types.SessionEntry, _ []types.ToolCall, _ int) (*analyzer.ErrorAnalysisResult, error) {
	return s.result, s.err
}

// setupEmptyProjectDir creates a project directory with an empty session file so
// loadData returns an empty slice without error (locator requires at least one file).
func setupEmptyProjectDir(t *testing.T) string {
	t.Helper()
	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)
	projectPath := t.TempDir()
	absProject, err := filepath.Abs(projectPath)
	require.NoError(t, err)
	hash := strings.ReplaceAll(absProject, "/", "-")
	sessionDir := filepath.Join(projectsRoot, hash)
	require.NoError(t, os.MkdirAll(sessionDir, 0o755))
	// Create an empty JSONL file so the locator finds at least one session file.
	emptyFile := filepath.Join(sessionDir, "empty-session.jsonl")
	require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0o644))
	return projectPath
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

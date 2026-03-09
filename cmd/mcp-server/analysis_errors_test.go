package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAnalysisTestProjectDir creates a fake META_CC_PROJECTS_ROOT with the given
// JSONL file under the hash-based directory for projectPath.
// Returns the projectPath to pass as working_dir.
func setupAnalysisTestProjectDir(t *testing.T, sourceJSONL string) string {
	t.Helper()

	// Create the fake projects root
	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)

	// Create a temp "project" directory
	projectPath := t.TempDir()

	// The locator converts projectPath to a hash via pathToHash (/ -> -)
	// We replicate the same logic: replace "/" with "-"
	absProject, err := filepath.Abs(projectPath)
	require.NoError(t, err)
	hash := strings.ReplaceAll(absProject, "/", "-")

	sessionDir := filepath.Join(projectsRoot, hash)
	err = os.MkdirAll(sessionDir, 0755)
	require.NoError(t, err)

	// Copy the source JSONL into the session directory
	dst := filepath.Join(sessionDir, "test-session.jsonl")
	src, err := os.Open(sourceJSONL)
	require.NoError(t, err)
	defer src.Close()

	dstFile, err := os.Create(dst)
	require.NoError(t, err)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, src)
	require.NoError(t, err)

	return projectPath
}

// TestAnalyzeErrorsToolRegistered verifies analyze_errors appears in getToolDefinitions()
func TestAnalyzeErrorsToolRegistered(t *testing.T) {
	tools := getToolDefinitions()
	found := false
	for _, tool := range tools {
		if tool.Name == "analyze_errors" {
			found = true
			break
		}
	}
	assert.True(t, found, "analyze_errors tool should be registered in getToolDefinitions()")
}

// TestAnalyzeErrorsToolExecution loads test.jsonl and verifies valid JSON output with total_errors field
func TestAnalyzeErrorsToolExecution(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"working_dir": projectPath,
	}

	output, err := executeAnalyzeErrorsTool(nil, args)
	require.NoError(t, err, "executeAnalyzeErrorsTool should not return error")
	require.NotEmpty(t, output, "output should not be empty")

	// Verify output is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	// Verify total_errors field exists
	_, hasTotalErrors := result["total_errors"]
	assert.True(t, hasTotalErrors, "result should have total_errors field")
}

// TestAnalyzeErrorsToolLimit verifies limit parameter restricts examples per group
func TestAnalyzeErrorsToolLimit(t *testing.T) {
	testJSONL, err := filepath.Abs("test.jsonl")
	require.NoError(t, err)
	_, err = os.Stat(testJSONL)
	require.NoError(t, err, "test.jsonl must exist")

	projectPath := setupAnalysisTestProjectDir(t, testJSONL)

	args := map[string]interface{}{
		"limit":       float64(1),
		"working_dir": projectPath,
	}

	output, err := executeAnalyzeErrorsTool(nil, args)
	require.NoError(t, err, "executeAnalyzeErrorsTool should not return error with limit=1")
	require.NotEmpty(t, output, "output should not be empty")

	// Verify output is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "output should be valid JSON")

	// Verify by_tool groups have at most 1 example
	if byTool, ok := result["by_tool"].([]interface{}); ok {
		for i, group := range byTool {
			g, ok := group.(map[string]interface{})
			require.True(t, ok, "group %d should be an object", i)
			if examples, ok := g["examples"].([]interface{}); ok {
				assert.LessOrEqual(t, len(examples), 1, "group %d should have at most 1 example with limit=1", i)
			}
		}
	}
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetQueryBaseDirWithWorkingDir tests that getQueryBaseDir uses workingDir
// when provided instead of os.Getwd().
func TestGetQueryBaseDirWithWorkingDir(t *testing.T) {
	// Setup: create a mock session directory structure using the same pattern
	// as setupTestSessionDir but returning the session dir for assertion.
	testData := `{"type":"user","message":{"content":"hello"}}`

	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)

	projectPath := t.TempDir()
	resolvedPath, err := filepath.EvalSymlinks(projectPath)
	require.NoError(t, err)

	projectHash := pathToHash(resolvedPath)
	sessionDir := filepath.Join(projectsRoot, projectHash)
	err = os.MkdirAll(sessionDir, 0755)
	require.NoError(t, err)

	sessionFile := filepath.Join(sessionDir, "test-session.jsonl")
	err = os.WriteFile(sessionFile, []byte(testData), 0644)
	require.NoError(t, err)

	t.Run("explicit workingDir uses provided path for project scope", func(t *testing.T) {
		baseDir, err := getQueryBaseDir("project", projectPath)
		require.NoError(t, err)
		assert.Equal(t, sessionDir, baseDir,
			"getQueryBaseDir should use workingDir to find sessions")
	})

	t.Run("explicit workingDir uses provided path for session scope", func(t *testing.T) {
		baseDir, err := getQueryBaseDir("session", projectPath)
		require.NoError(t, err)
		assert.Equal(t, sessionDir, baseDir,
			"getQueryBaseDir should use workingDir for session scope too")
	})

	t.Run("empty workingDir falls back to CWD", func(t *testing.T) {
		originalWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(originalWd)
			require.NoError(t, err)
		}()

		err = os.Chdir(projectPath)
		require.NoError(t, err)

		baseDir, err := getQueryBaseDir("project", "")
		require.NoError(t, err)
		assert.Equal(t, sessionDir, baseDir,
			"empty workingDir should fall back to CWD")
	})
}

// TestExecuteQueryWithWorkingDir tests that executeQuery threads workingDir
// through to getQueryBaseDir.
func TestExecuteQueryWithWorkingDir(t *testing.T) {
	testEntries := []map[string]interface{}{
		{"type": "user", "message": map[string]interface{}{"content": "hello"}},
		{"type": "assistant", "message": map[string]interface{}{"content": "hi"}},
	}

	projectsRoot := t.TempDir()
	t.Setenv("META_CC_PROJECTS_ROOT", projectsRoot)

	projectPath := t.TempDir()
	resolvedPath, err := filepath.EvalSymlinks(projectPath)
	require.NoError(t, err)

	projectHash := pathToHash(resolvedPath)
	sessionDir := filepath.Join(projectsRoot, projectHash)
	err = os.MkdirAll(sessionDir, 0755)
	require.NoError(t, err)

	sessionFile := filepath.Join(sessionDir, "test-session.jsonl")
	var lines []string
	for _, entry := range testEntries {
		data, _ := json.Marshal(entry)
		lines = append(lines, string(data))
	}
	err = os.WriteFile(sessionFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
	require.NoError(t, err)

	executor := NewToolExecutor()

	t.Run("executeQuery with explicit workingDir finds session data", func(t *testing.T) {
		result, err := executor.executeQuery("project", ".", 0, projectPath)
		require.NoError(t, err)
		assert.Len(t, result.Entries, 2, "should find 2 entries via workingDir")
	})

	t.Run("executeQuery with empty workingDir uses CWD fallback", func(t *testing.T) {
		originalWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(originalWd)
			require.NoError(t, err)
		}()

		err = os.Chdir(projectPath)
		require.NoError(t, err)

		result, err := executor.executeQuery("project", ".", 0, "")
		require.NoError(t, err)
		assert.Len(t, result.Entries, 2, "should find 2 entries via CWD fallback")
	})
}

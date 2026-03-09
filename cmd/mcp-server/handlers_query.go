package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/yaleh/meta-cc/internal/locator"
)

// TimeRange specifies optional lower and upper bounds for timestamp filtering.
// A nil pointer means "no bound" (open-ended).
type TimeRange struct {
	Since *time.Time
	Until *time.Time
}

// parseTimeRange parses since/until strings (RFC3339) into a TimeRange.
// Empty string means no bound. Non-RFC3339 values return an error.
func parseTimeRange(sinceStr, untilStr string) (TimeRange, error) {
	var tr TimeRange
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid since value %q: must be RFC3339 (e.g. 2026-03-07T00:00:00Z)", sinceStr)
		}
		tr.Since = &t
	}
	if untilStr != "" {
		t, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			return TimeRange{}, fmt.Errorf("invalid until value %q: must be RFC3339 (e.g. 2026-03-09T00:00:00Z)", untilStr)
		}
		tr.Until = &t
	}
	return tr, nil
}

// handleQuery and handleQueryRaw deleted in Phase 27 Stage 27.1
// These tools were removed to simplify the query interface
// Users should use the 10 shortcut query tools instead

// executeQuery is an internal helper for convenience tools
// It executes a jq query and returns results as QueryResult
// This allows proper JSONL formatting by response adapters
// workingDir specifies the project directory for session lookup;
// empty string ("") means use os.Getwd() as fallback (backward compatible).
func (e *ToolExecutor) executeQuery(scope string, jqFilter string, limit int, workingDir string) (QueryResult, error) {
	return e.executeQueryWithTimeRange(scope, jqFilter, limit, workingDir, TimeRange{})
}

// executeQueryWithTimeRange is like executeQuery but applies time-range filtering before jq execution.
// tr.Since and tr.Until are optional (nil = no bound).
func (e *ToolExecutor) executeQueryWithTimeRange(scope string, jqFilter string, limit int, workingDir string, tr TimeRange) (QueryResult, error) {
	// Get base directory using pipeline infrastructure
	baseDir, err := getQueryBaseDir(scope, workingDir)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get base directory: %w", err)
	}

	// Create query executor
	executor := NewQueryExecutor(baseDir)

	// Compile expression
	code, err := executor.compileExpression(jqFilter)
	if err != nil {
		return QueryResult{}, fmt.Errorf("invalid jq expression: %w", err)
	}

	// Get all JSONL files in directory
	files, err := getJSONLFiles(baseDir)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to list JSONL files: %w", err)
	}

	if len(files) == 0 {
		return QueryResult{}, fmt.Errorf("no JSONL files found in %s", baseDir)
	}

	// Execute query with streaming and time range filtering
	ctx := context.Background()
	result := executor.streamFilesWithTimeRange(ctx, files, code, limit, tr)

	// Return QueryResult directly
	// Response adapters will handle serialization (inline or file_ref)
	return result, nil
}

// getQueryBaseDir returns the base directory for the given scope.
// For session scope: returns directory of most recently modified session file.
// For project scope: returns directory containing all session files.
// workingDir specifies the project directory for session lookup;
// empty string ("") means use os.Getwd() as fallback (backward compatible).
func getQueryBaseDir(scope, workingDir string) (string, error) {
	// Determine effective project path: use workingDir if non-empty, else CWD
	projectPath := workingDir
	if projectPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		projectPath = cwd
	}

	loc := locator.NewSessionLocator()

	// Session scope: return directory of most recently modified session file
	if scope == "session" {
		// Use FromProjectPath to find the newest session file
		sessionFile, err := loc.FromProjectPath(projectPath)
		if err != nil {
			return "", fmt.Errorf("failed to locate current session: %w", err)
		}

		// Return the directory containing the session file
		return filepath.Dir(sessionFile), nil
	}

	// Project scope: use SessionLocator to find all session files
	// This matches the behavior of buildPipelineOptions + SessionPipeline.Load

	// AllSessionsFromProject returns the list of session files
	// We need to return the directory containing those files
	sessionFiles, err := loc.AllSessionsFromProject(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to locate project sessions: %w", err)
	}

	if len(sessionFiles) == 0 {
		return "", fmt.Errorf("no sessions found for project: %s", projectPath)
	}

	// All session files should be in the same directory
	// Return the directory of the first session file
	return filepath.Dir(sessionFiles[0]), nil
}

// loadTurnsForSession reads all JSONL files in baseDir and returns the turns
// (entries) that belong to sessionID. Each JSONL file is scanned for entries
// where obj["sessionId"] == sessionID. Returns nil, nil if no entries are found.
func loadTurnsForSession(baseDir, sessionID string) ([]interface{}, error) {
	files, err := getJSONLFiles(baseDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		var turns []interface{}

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		// Allow long lines (up to 10 MB per line)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				continue
			}
			sid, _ := obj["sessionId"].(string)
			if sid == sessionID {
				turns = append(turns, obj)
			}
		}
		f.Close()

		if len(turns) > 0 {
			return turns, nil
		}
	}

	return nil, nil
}

// getJSONLFiles returns all .jsonl files in a directory (non-recursive)
// Files are sorted by modification time (newest first) to prioritize recent sessions
func getJSONLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Collect file info with modification times
	type fileInfo struct {
		path    string
		modTime int64 // Unix timestamp for easier sorting
	}
	var fileInfos []fileInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".jsonl" {
			fullPath := filepath.Join(dir, entry.Name())

			// Get file stat for modification time
			info, err := entry.Info()
			if err != nil {
				// Skip files we can't stat
				continue
			}

			fileInfos = append(fileInfos, fileInfo{
				path:    fullPath,
				modTime: info.ModTime().Unix(),
			})
		}
	}

	// Sort by modification time (newest first = descending order)
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].modTime > fileInfos[j].modTime
	})

	// Extract paths
	var files []string
	for _, fi := range fileInfos {
		files = append(files, fi.path)
	}

	return files, nil
}

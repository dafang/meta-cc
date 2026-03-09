package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	mcerrors "github.com/yaleh/meta-cc/internal/errors"
)

// Session cache variables (moved from capabilities.go for Phase 45.1)
var (
	sessionCacheDir     string
	sessionCacheInitErr error
	sessionCacheOnce    sync.Once
)

// getSessionCacheDir returns the session-scoped cache directory
// Creates temp directory on first call, reuses for subsequent calls in same session
func getSessionCacheDir() (string, error) {
	sessionCacheOnce.Do(func() {
		// Try to get session ID from environment
		sessionID := os.Getenv("CLAUDE_CODE_SESSION_ID")
		if sessionID == "" {
			// Fallback: use process ID
			sessionID = fmt.Sprintf("mcp-%d", os.Getpid())
		}

		// Create session temp directory
		tempBase := os.TempDir()
		sessionDir := filepath.Join(tempBase, fmt.Sprintf("claude-session-%s", sessionID))

		// Create cache directory within session dir
		cacheDir := filepath.Join(sessionDir, ".meta-cc-capabilities")

		slog.Debug("initializing session cache",
			"session_id", sessionID,
			"cache_dir", cacheDir,
		)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			slog.Error("failed to create session cache directory",
				"cache_dir", cacheDir,
				"error", err.Error(),
				"error_type", "io_error",
			)
			sessionCacheInitErr = fmt.Errorf("failed to create session cache directory '%s': %w", cacheDir, mcerrors.ErrFileIO)
			return
		}

		sessionCacheDir = cacheDir
	})

	if sessionCacheInitErr != nil {
		return "", sessionCacheInitErr
	}

	return sessionCacheDir, nil
}

// CleanupSessionCache removes the session cache directory
// Should be called on MCP server shutdown
func CleanupSessionCache() error {
	if sessionCacheDir == "" {
		return nil
	}

	// Remove the entire session directory (parent of cache dir)
	sessionDir := filepath.Dir(sessionCacheDir)
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("failed to cleanup session cache directory '%s': %w", sessionDir, mcerrors.ErrFileIO)
	}

	return nil
}

// TempFileManager manages temporary JSONL files with concurrency safety
type TempFileManager struct {
	mu sync.Mutex
}

var tempFileManager = &TempFileManager{}

// createTempFilePath generates a unique temporary file path
//
// Pattern: /tmp/meta-cc-mcp-{session_hash}-{timestamp}-{query_type}.jsonl
//
// Parameters:
//   - sessionHash: First 8 chars of session ID for grouping
//   - queryType: Tool name (e.g., "query_tools", "get_stats")
//
// Returns:
//   - Absolute path to temp file
func createTempFilePath(sessionHash, queryType string) string {
	// Use nanosecond timestamp for uniqueness
	timestamp := time.Now().UnixNano()

	filename := fmt.Sprintf("meta-cc-mcp-%s-%d-%s.jsonl",
		sessionHash, timestamp, queryType)

	return filepath.Join(os.TempDir(), filename)
}

// writeJSONLFile writes data to a JSONL file
//
// Parameters:
//   - path: Absolute file path
//   - data: Array of records to serialize
//
// Returns:
//   - Error if file creation or write fails
//
// The function:
//  1. Creates parent directories if needed
//  2. Writes each record as a JSON line
//  3. Uses atomic write (temp + rename) for safety
func writeJSONLFile(path string, data []interface{}) error {
	tempFileManager.mu.Lock()
	defer tempFileManager.mu.Unlock()

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, mcerrors.ErrFileIO)
	}

	// Create temp file for atomic write
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	// Write JSONL data
	encoder := json.NewEncoder(file)
	for _, record := range data {
		if err := encoder.Encode(record); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to encode record to %s: %w", tmpPath, mcerrors.ErrParseError)
		}
	}

	// Sync to disk
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	// Close file before rename (required on Windows)
	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file %s to %s: %w", tmpPath, path, mcerrors.ErrFileIO)
	}

	return nil
}

// cleanupOldFiles removes temporary files older than maxAgeDays
//
// Parameters:
//   - maxAgeDays: Maximum age in days (files older than this are removed)
//
// Returns:
//   - []string: List of removed file paths
//   - int64: Total bytes freed
//   - error: If scan fails
//
// The function scans /tmp for meta-cc-mcp-*.jsonl files and removes
// files with modification time older than the threshold.
func cleanupOldFiles(maxAgeDays int) ([]string, int64, error) {
	pattern := filepath.Join(os.TempDir(), "meta-cc-mcp-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to glob files with pattern %s: %w", pattern, mcerrors.ErrFileIO)
	}

	threshold := time.Now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)
	removed := []string{}
	var freedBytes int64

	for _, path := range files {
		stat, err := os.Stat(path)
		if err != nil {
			continue // Skip files we can't stat
		}

		// Check if file is older than threshold
		if stat.ModTime().Before(threshold) {
			size := stat.Size()
			if err := os.Remove(path); err == nil {
				removed = append(removed, path)
				freedBytes += size
			}
		}
	}

	return removed, freedBytes, nil
}

// executeCleanupTool handles the cleanup_temp_files MCP tool
func executeCleanupTool(args map[string]interface{}) (string, error) {
	maxAgeDays := getIntParam(args, "max_age_days", 7)

	removed, freedBytes, err := cleanupOldFiles(maxAgeDays)
	if err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"removed_count": len(removed),
		"freed_bytes":   freedBytes,
		"files":         removed,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

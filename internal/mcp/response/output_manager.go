package response

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

// Session cache variables
var (
	sessionCacheDir     string
	sessionCacheInitErr error
	sessionCacheOnce    sync.Once
)

// GetSessionCacheDir returns the session-scoped cache directory
func GetSessionCacheDir() (string, error) {
	sessionCacheOnce.Do(func() {
		sessionID := os.Getenv("CLAUDE_CODE_SESSION_ID")
		if sessionID == "" {
			sessionID = fmt.Sprintf("mcp-%d", os.Getpid())
		}

		tempBase := os.TempDir()
		sessionDir := filepath.Join(tempBase, fmt.Sprintf("claude-session-%s", sessionID))
		cacheDir := filepath.Join(sessionDir, ".meta-cc-capabilities")

		slog.Debug("initializing session cache",
			"session_id", sessionID,
			"cache_dir", cacheDir,
		)

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
func CleanupSessionCache() error {
	if sessionCacheDir == "" {
		return nil
	}

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

var defaultTempFileManager = &TempFileManager{}

// CreateTempFilePath generates a unique temporary file path
func CreateTempFilePath(sessionHash, queryType string) string {
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("meta-cc-mcp-%s-%d-%s.jsonl", sessionHash, timestamp, queryType)
	return filepath.Join(os.TempDir(), filename)
}

// WriteJSONLFile writes data to a JSONL file
func WriteJSONLFile(path string, data []interface{}) error {
	defaultTempFileManager.mu.Lock()
	defer defaultTempFileManager.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, mcerrors.ErrFileIO)
	}

	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	encoder := json.NewEncoder(file)
	for _, record := range data {
		if err := encoder.Encode(record); err != nil {
			file.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to encode record to %s: %w", tmpPath, mcerrors.ErrParseError)
		}
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to sync file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close file %s: %w", tmpPath, mcerrors.ErrFileIO)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename file %s to %s: %w", tmpPath, path, mcerrors.ErrFileIO)
	}

	return nil
}

// CleanupOldFiles removes temporary files older than maxAgeDays
func CleanupOldFiles(maxAgeDays int) ([]string, int64, error) {
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
			continue
		}

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

// ExecuteCleanupTool handles the cleanup_temp_files MCP tool
func ExecuteCleanupTool(args map[string]interface{}) (string, error) {
	maxAgeDays := getIntParam(args, "max_age_days", 7)

	removed, freedBytes, err := CleanupOldFiles(maxAgeDays)
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

// getIntParam is a local helper
func getIntParam(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultVal
}

package analyzer

import (
	"sort"

	"github.com/yaleh/meta-cc/internal/query/turnindex"
	"github.com/yaleh/meta-cc/internal/types"
)

type fileAccessStats struct {
	file          string
	readCount     int
	editCount     int
	writeCount    int
	totalAccesses int
	firstAccess   int64
	lastAccess    int64
}

// DetectFileChurn detects files with frequent access
func DetectFileChurn(entries []types.SessionEntry, threshold int) FileChurnAnalysis {
	// Extract file access events
	fileAccess := make(map[string]*fileAccessStats)

	toolCalls := types.ExtractToolCalls(entries)
	for _, tc := range toolCalls {
		// Extract file path
		filePath := extractFileFromToolCall(tc)
		if filePath == "" {
			continue
		}

		// Get action type
		action := types.FileActionType(tc.ToolName)
		if action == "" {
			continue
		}

		// Get timestamp
		timestamp := getToolCallTimestamp(entries, tc.UUID)

		// Initialize or update stats
		if _, exists := fileAccess[filePath]; !exists {
			fileAccess[filePath] = &fileAccessStats{
				file:        filePath,
				firstAccess: timestamp,
				lastAccess:  timestamp,
			}
		}

		stats := fileAccess[filePath]
		stats.totalAccesses++

		switch action {
		case "Read":
			stats.readCount++
		case "Edit":
			stats.editCount++
		case "Write":
			stats.writeCount++
		}

		if timestamp < stats.firstAccess {
			stats.firstAccess = timestamp
		}
		if timestamp > stats.lastAccess {
			stats.lastAccess = timestamp
		}
	}

	// Filter by threshold and build result
	var highChurnFiles []FileChurnDetail
	for _, stats := range fileAccess {
		if stats.totalAccesses >= threshold {
			timeSpan := 0
			if stats.lastAccess > stats.firstAccess {
				timeSpan = int((stats.lastAccess - stats.firstAccess) / 60)
			}

			highChurnFiles = append(highChurnFiles, FileChurnDetail{
				File:          stats.file,
				ReadCount:     stats.readCount,
				EditCount:     stats.editCount,
				WriteCount:    stats.writeCount,
				TotalAccesses: stats.totalAccesses,
				TimeSpanMin:   timeSpan,
				FirstAccess:   stats.firstAccess,
				LastAccess:    stats.lastAccess,
			})
		}
	}

	// Sort by total accesses (descending)
	sort.Slice(highChurnFiles, func(i, j int) bool {
		return highChurnFiles[i].TotalAccesses > highChurnFiles[j].TotalAccesses
	})

	return FileChurnAnalysis{
		HighChurnFiles: highChurnFiles,
	}
}

func getToolCallTimestamp(entries []types.SessionEntry, uuid string) int64 {
	for _, entry := range entries {
		if entry.UUID == uuid {
			return turnindex.ParseTimestamp(entry.Timestamp)
		}
	}
	return 0
}

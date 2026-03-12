// Package turnindex provides shared helpers for building turn indices from
// session entries. It is a neutral sub-package that may be imported by any
// package in internal/query/ without creating import cycles.
package turnindex

import (
	"time"

	"github.com/yaleh/meta-cc/internal/parser"
)

// BuildTurnIndex creates a map of UUID to turn number.
// Only entries that represent messages (IsMessage() == true) are indexed.
func BuildTurnIndex(entries []parser.SessionEntry) map[string]int {
	index := make(map[string]int)
	turn := 0
	for _, entry := range entries {
		if entry.IsMessage() {
			index[entry.UUID] = turn
			turn++
		}
	}
	return index
}

// GetToolCallTimestamp finds the Unix timestamp for a tool call identified by uuid.
// Returns 0 if the UUID is not found or if the timestamp cannot be parsed.
func GetToolCallTimestamp(entries []parser.SessionEntry, uuid string) int64 {
	for _, entry := range entries {
		if entry.UUID == uuid {
			return parseTimestamp(entry.Timestamp)
		}
	}
	return 0
}

// parseTimestamp parses an RFC3339Nano timestamp string to Unix seconds.
func parseTimestamp(ts string) int64 {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0
	}
	return t.Unix()
}

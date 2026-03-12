package analyzer

import (
	"sort"
	"strings"

	"github.com/yaleh/meta-cc/internal/query/turnindex"
	"github.com/yaleh/meta-cc/internal/types"
)

type toolCallWithTurn struct {
	toolName string
	turn     int
	uuid     string
	filePath string
	command  string
}

// DetectToolSequences detects repeated tool call sequences
func DetectToolSequences(entries []types.SessionEntry, minLength, minOccurrences int) SequenceAnalysis {
	// Build turn index
	turnIdx := turnindex.BuildTurnIndex(entries)

	// Extract tool calls with turn numbers
	toolCalls := extractToolCallsWithTurns(entries, turnIdx)

	// Sort by turn
	sort.Slice(toolCalls, func(i, j int) bool {
		return toolCalls[i].turn < toolCalls[j].turn
	})

	// Find all sequences
	sequences := findAllSequences(toolCalls, minLength, minOccurrences, entries)

	return SequenceAnalysis{
		Sequences: sequences,
	}
}

func extractToolCallsWithTurns(entries []types.SessionEntry, turnIdx map[string]int) []toolCallWithTurn {
	var result []toolCallWithTurn

	toolCalls := types.ExtractToolCalls(entries)
	for _, tc := range toolCalls {
		if turn, ok := turnIdx[tc.UUID]; ok {
			result = append(result, toolCallWithTurn{
				toolName: tc.ToolName,
				turn:     turn,
				uuid:     tc.UUID,
				filePath: extractFileFromToolCall(tc),
				command:  extractCommandFromToolCall(tc),
			})
		}
	}

	return result
}

func findAllSequences(toolCalls []toolCallWithTurn, minLength, minOccurrences int, entries []types.SessionEntry) []types.SequencePattern {
	sequenceMap := make(map[string][]types.SequenceOccurrence)

	// Try sequences of different lengths
	maxLen := 5
	if maxLen > len(toolCalls) {
		maxLen = len(toolCalls)
	}

	for seqLen := minLength; seqLen <= maxLen; seqLen++ {
		for i := 0; i <= len(toolCalls)-seqLen; i++ {
			// Extract sequence
			tools := make([]string, seqLen)
			for j := 0; j < seqLen; j++ {
				tools[j] = toolCalls[i+j].toolName
			}

			// Create pattern string
			pattern := strings.Join(tools, " → ")

			// Build occurrence with tool details
			var toolsInSeq []types.ToolInSequence
			for j := 0; j < seqLen; j++ {
				tc := toolCalls[i+j]
				toolsInSeq = append(toolsInSeq, types.ToolInSequence{
					Turn:    tc.turn,
					Tool:    tc.toolName,
					File:    tc.filePath,
					Command: tc.command,
				})
			}

			occurrence := types.SequenceOccurrence{
				StartTurn: toolCalls[i].turn,
				EndTurn:   toolCalls[i+seqLen-1].turn,
				Tools:     toolsInSeq,
			}

			sequenceMap[pattern] = append(sequenceMap[pattern], occurrence)
		}
	}

	// Filter by minimum occurrences and build result
	var result []types.SequencePattern
	for pattern, occurrences := range sequenceMap {
		if len(occurrences) >= minOccurrences {
			// Calculate length
			length := len(strings.Split(pattern, " → "))

			// Calculate time span
			timeSpan := calculateSequenceTimeSpan(occurrences, entries)

			result = append(result, types.SequencePattern{
				Pattern:     pattern,
				Length:      length,
				Count:       len(occurrences),
				Occurrences: occurrences,
				TimeSpanMin: timeSpan,
			})
		}
	}

	// Sort by count (descending), then by length (descending)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Length > result[j].Length
	})

	return result
}

func extractFileFromToolCall(tc types.ToolCall) string {
	fileParams := []string{"file_path", "notebook_path", "path"}

	for _, param := range fileParams {
		if val, ok := tc.Input[param]; ok {
			if filePath, ok := val.(string); ok && filePath != "" {
				return filePath
			}
		}
	}

	return ""
}

func extractCommandFromToolCall(tc types.ToolCall) string {
	if tc.ToolName == "Bash" {
		if val, ok := tc.Input["command"]; ok {
			if cmd, ok := val.(string); ok {
				// Return first line only for preview
				lines := strings.Split(cmd, "\n")
				if len(lines) > 0 {
					return lines[0]
				}
			}
		}
	}
	return ""
}

func calculateSequenceTimeSpan(occurrences []types.SequenceOccurrence, entries []types.SessionEntry) int {
	if len(occurrences) == 0 {
		return 0
	}

	var minTs, maxTs int64

	for _, occ := range occurrences {
		// Find timestamps for turns in this occurrence
		for _, entry := range entries {
			ts := turnindex.ParseTimestamp(entry.Timestamp)
			if ts == 0 {
				continue
			}

			// Check if this entry is part of the occurrence
			for range occ.Tools {
				if entry.UUID != "" && ts > 0 {
					// Update min/max
					if minTs == 0 || ts < minTs {
						minTs = ts
					}
					if ts > maxTs {
						maxTs = ts
					}
					break
				}
			}
		}
	}

	if minTs == 0 || maxTs == 0 {
		return 0
	}

	return int((maxTs - minTs) / 60)
}

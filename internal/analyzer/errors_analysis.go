package analyzer

import (
	"sort"
	"time"

	"github.com/yaleh/meta-cc/internal/parser"
)

// TimeRange represents the start and end of a time window
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ToolErrorGroup groups errors by tool name
type ToolErrorGroup struct {
	ToolName string   `json:"tool_name"`
	Count    int      `json:"count"`
	Examples []string `json:"examples"`
}

// ErrorTypeGroup groups errors by signature (tool+error hash)
type ErrorTypeGroup struct {
	Signature string   `json:"signature"`
	Count     int      `json:"count"`
	Examples  []string `json:"examples"`
}

// ErrorAnalysisResult holds the full error analysis output
type ErrorAnalysisResult struct {
	TimeRange   TimeRange        `json:"time_range"`
	TotalErrors int              `json:"total_errors"`
	ByTool      []ToolErrorGroup `json:"by_tool"`
	ByType      []ErrorTypeGroup `json:"by_type"`
}

// AnalyzeErrors analyzes tool call errors and groups them by tool and error type.
// limit controls the maximum number of example messages per group (0 = no limit).
func AnalyzeErrors(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*ErrorAnalysisResult, error) {
	result := &ErrorAnalysisResult{}

	// Calculate TimeRange from entries
	for _, e := range entries {
		t, err := time.Parse("2006-01-02T15:04:05.000Z", e.Timestamp)
		if err != nil {
			// Try standard RFC3339
			t, err = time.Parse(time.RFC3339, e.Timestamp)
			if err != nil {
				continue
			}
		}
		if result.TimeRange.Start.IsZero() || t.Before(result.TimeRange.Start) {
			result.TimeRange.Start = t
		}
		if result.TimeRange.End.IsZero() || t.After(result.TimeRange.End) {
			result.TimeRange.End = t
		}
	}

	// Filter errors
	toolGroupMap := make(map[string]*ToolErrorGroup)
	typeGroupMap := make(map[string]*ErrorTypeGroup)

	for _, tc := range toolCalls {
		if tc.Status != "error" && tc.Error == "" {
			continue
		}
		result.TotalErrors++

		// Group by tool name
		tg, ok := toolGroupMap[tc.ToolName]
		if !ok {
			tg = &ToolErrorGroup{ToolName: tc.ToolName}
			toolGroupMap[tc.ToolName] = tg
		}
		tg.Count++
		if limit <= 0 || len(tg.Examples) < limit {
			tg.Examples = append(tg.Examples, tc.Error)
		}

		// Group by error signature
		sig := CalculateErrorSignature(tc.ToolName, tc.Error)
		eg, ok := typeGroupMap[sig]
		if !ok {
			eg = &ErrorTypeGroup{Signature: sig}
			typeGroupMap[sig] = eg
		}
		eg.Count++
		if limit <= 0 || len(eg.Examples) < limit {
			eg.Examples = append(eg.Examples, tc.Error)
		}
	}

	// Convert maps to sorted slices
	for _, g := range toolGroupMap {
		result.ByTool = append(result.ByTool, *g)
	}
	sort.Slice(result.ByTool, func(i, j int) bool {
		if result.ByTool[i].Count == result.ByTool[j].Count {
			return result.ByTool[i].ToolName < result.ByTool[j].ToolName
		}
		return result.ByTool[i].Count > result.ByTool[j].Count
	})

	for _, g := range typeGroupMap {
		result.ByType = append(result.ByType, *g)
	}
	sort.Slice(result.ByType, func(i, j int) bool {
		if result.ByType[i].Count == result.ByType[j].Count {
			return result.ByType[i].Signature < result.ByType[j].Signature
		}
		return result.ByType[i].Count > result.ByType[j].Count
	})

	return result, nil
}

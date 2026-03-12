package analyzer

import (
	"sort"

	"github.com/yaleh/meta-cc/internal/query/turnindex"
	"github.com/yaleh/meta-cc/internal/types"
)

// DetectIdlePeriods detects idle periods in the session
func DetectIdlePeriods(entries []types.SessionEntry, thresholdMin int) IdlePeriodAnalysis {
	// Build turn index
	turnIdx := turnindex.BuildTurnIndex(entries)

	// Extract all entries with timestamps (both user and assistant)
	type entryWithTurn struct {
		entry types.SessionEntry
		turn  int
	}

	var entriesWithTurns []entryWithTurn
	for _, entry := range entries {
		if turn, ok := turnIdx[entry.UUID]; ok {
			entriesWithTurns = append(entriesWithTurns, entryWithTurn{
				entry: entry,
				turn:  turn,
			})
		}
	}

	// Sort by turn
	sort.Slice(entriesWithTurns, func(i, j int) bool {
		return entriesWithTurns[i].turn < entriesWithTurns[j].turn
	})

	// Find idle periods
	var idlePeriods []IdlePeriod
	thresholdSec := float64(thresholdMin * 60)

	for i := 0; i < len(entriesWithTurns)-1; i++ {
		current := entriesWithTurns[i]
		next := entriesWithTurns[i+1]

		currentTs := turnindex.ParseTimestamp(current.entry.Timestamp)
		nextTs := turnindex.ParseTimestamp(next.entry.Timestamp)

		if currentTs == 0 || nextTs == 0 {
			continue
		}

		gapSec := float64(nextTs - currentTs)
		if gapSec >= thresholdSec {
			// Found an idle period
			period := IdlePeriod{
				StartTurn:      current.turn,
				EndTurn:        next.turn,
				DurationMin:    gapSec / 60,
				StartTimestamp: currentTs,
				EndTimestamp:   nextTs,
			}

			// Add context
			period.ContextBefore = extractTurnContext(current.entry, current.turn)
			period.ContextAfter = extractTurnContext(next.entry, next.turn)

			idlePeriods = append(idlePeriods, period)
		}
	}

	return IdlePeriodAnalysis{
		IdlePeriods: idlePeriods,
	}
}

func extractTurnContext(entry types.SessionEntry, turn int) *TurnContext {
	ctx := &TurnContext{
		Turn: turn,
		Role: entry.Type,
	}

	if entry.Message != nil {
		// Extract tool info
		for _, block := range entry.Message.Content {
			if block.Type == "tool_use" && block.ToolUse != nil {
				ctx.Tool = block.ToolUse.Name
			} else if block.Type == "tool_result" && block.ToolResult != nil {
				ctx.Status = block.ToolResult.Status
				if ctx.Status == "" && block.ToolResult.Error != "" {
					ctx.Status = "error"
				}
			} else if block.Type == "text" && block.Text != "" {
				// Extract preview (first 100 chars)
				preview := block.Text
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				ctx.Preview = preview
			}
		}
	}

	return ctx
}

package analyzer

import (
	"fmt"
	"sort"
	"time"

	"github.com/yaleh/meta-cc/internal/types"
)

// TimelineEvent represents a single event in the session timeline.
type TimelineEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	Summary    string    `json:"summary"`
	DurationMs int64     `json:"duration_ms"`
}

// TimelineResult holds the chronological event list and total span.
type TimelineResult struct {
	Events    []TimelineEvent `json:"events"`
	TotalSpan string          `json:"total_span"`
}

// entryTypeLabel maps raw entry types to human-readable event types.
func entryTypeLabel(t string) string {
	switch t {
	case "user":
		return "user_message"
	case "assistant":
		return "assistant_message"
	default:
		return t
	}
}

// entryToTimestamp parses an entry timestamp into time.Time.
func entryToTimestamp(ts string) time.Time {
	formats := []string{
		"2006-01-02T15:04:05.000Z",
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}

// formatSpan formats a duration as "Xh Ym", "Ym", or "Xs".
func formatSpan(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%ds", s)
}

// GetTimeline converts session entries to a sorted, merged timeline.
func GetTimeline(entries []types.SessionEntry, limit int) (*TimelineResult, error) {
	if len(entries) == 0 {
		return &TimelineResult{Events: []TimelineEvent{}, TotalSpan: "0s"}, nil
	}

	// Convert to events and sort by timestamp.
	type raw struct {
		ts      time.Time
		etype   string
		summary string
	}

	rawEvents := make([]raw, 0, len(entries))
	for _, e := range entries {
		ts := entryToTimestamp(e.Timestamp)
		label := entryTypeLabel(e.Type)
		summary := label
		rawEvents = append(rawEvents, raw{ts: ts, etype: label, summary: summary})
	}

	sort.Slice(rawEvents, func(i, j int) bool {
		return rawEvents[i].ts.Before(rawEvents[j].ts)
	})

	// Merge consecutive events of the same type.
	merged := []TimelineEvent{}
	i := 0
	for i < len(rawEvents) {
		start := rawEvents[i]
		j := i + 1
		for j < len(rawEvents) && rawEvents[j].etype == start.etype {
			j++
		}
		count := j - i
		end := rawEvents[j-1]
		durMs := end.ts.Sub(start.ts).Milliseconds()

		summary := start.summary
		if count > 1 {
			summary = fmt.Sprintf("%s (x%d)", start.etype, count)
		}

		merged = append(merged, TimelineEvent{
			Timestamp:  start.ts,
			Type:       start.etype,
			Summary:    summary,
			DurationMs: durMs,
		})
		i = j
	}

	// Apply limit.
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	// Calculate total span.
	span := "0s"
	if len(rawEvents) >= 2 {
		first := rawEvents[0].ts
		last := rawEvents[len(rawEvents)-1].ts
		span = formatSpan(last.Sub(first))
	}

	return &TimelineResult{Events: merged, TotalSpan: span}, nil
}

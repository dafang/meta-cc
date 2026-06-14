// Package stats provides JSONL statistics generation functions extracted from
// internal/query. These are pure stdlib functions with no internal dependencies.
package stats

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// GenerateStats generates simple statistics from JSONL data grouped by tool name.
func GenerateStats(jsonlData string) (string, error) {
	lines := strings.Split(strings.TrimSpace(jsonlData), "\n")
	counts := make(map[string]int)

	for _, line := range lines {
		if line == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		key := "unknown"
		if tool, ok := obj["tool"].(string); ok {
			key = tool
		} else if toolName, ok := obj["ToolName"].(string); ok {
			key = toolName
		}

		counts[key]++
	}

	var output strings.Builder
	for key, count := range counts {
		statObj := map[string]interface{}{
			"key":   key,
			"count": count,
		}
		jsonBytes, _ := json.Marshal(statObj)
		output.Write(jsonBytes)
		output.WriteString("\n")
	}

	return output.String(), nil
}

// GenerateTimestampStats generates time-bucketed statistics (by hour) from JSONL data.
// It outputs a summary line followed by per-hour bucket lines, sorted chronologically.
// Records with unparseable timestamps are skipped (non-fatal).
// Returns empty string for empty input.
func GenerateTimestampStats(jsonlData string) (string, error) {
	trimmed := strings.TrimSpace(jsonlData)
	if trimmed == "" {
		return "", nil
	}

	lines := strings.Split(trimmed, "\n")
	hourCounts := make(map[string]int)
	sessions := make(map[string]bool)
	total := 0
	var minTS, maxTS time.Time

	for _, line := range lines {
		if line == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		tsStr, _ := obj["timestamp"].(string)
		if tsStr == "" {
			continue
		}

		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue
		}

		hourKey := ts.UTC().Format("2006-01-02T15")
		hourCounts[hourKey]++
		total++

		if sessionID, ok := obj["sessionId"].(string); ok && sessionID != "" {
			sessions[sessionID] = true
		}

		if minTS.IsZero() || ts.Before(minTS) {
			minTS = ts
		}
		if maxTS.IsZero() || ts.After(maxTS) {
			maxTS = ts
		}
	}

	if total == 0 {
		return "", nil
	}

	var output strings.Builder

	summary := map[string]interface{}{
		"total":         total,
		"session_count": len(sessions),
		"time_range": map[string]interface{}{
			"from": minTS.UTC().Format(time.RFC3339),
			"to":   maxTS.UTC().Format(time.RFC3339),
		},
	}
	summaryBytes, _ := json.Marshal(summary)
	output.Write(summaryBytes)
	output.WriteString("\n")

	hours := make([]string, 0, len(hourCounts))
	for h := range hourCounts {
		hours = append(hours, h)
	}
	sort.Strings(hours)

	for _, hour := range hours {
		bucket := map[string]interface{}{
			"hour":  hour,
			"count": hourCounts[hour],
		}
		bucketBytes, _ := json.Marshal(bucket)
		output.Write(bucketBytes)
		output.WriteString("\n")
	}

	return output.String(), nil
}

// sessionGroup holds accumulated data for one session during GroupBySession.
type sessionGroup struct {
	SessionID  string
	MatchCount int
	FirstMatch string
	LastMatch  string
	Turns      []interface{}
}

// GroupBySession groups a slice of parsed user-message entries by session.
// It supports both camelCase "sessionId" (raw data) and snake_case "session_id"
// (post-content_summary) field names. Output objects use snake_case "session_id".
// First-seen order of sessions is preserved.
func GroupBySession(entries []interface{}) []interface{} {
	var order []string
	groups := make(map[string]*sessionGroup)

	for _, entry := range entries {
		obj, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}

		sessionID, _ := obj["session_id"].(string)
		if sessionID == "" {
			sessionID, _ = obj["sessionId"].(string)
		}
		if sessionID == "" {
			sessionID = "unknown"
		}

		ts, _ := obj["timestamp"].(string)

		if _, exists := groups[sessionID]; !exists {
			order = append(order, sessionID)
			groups[sessionID] = &sessionGroup{
				SessionID:  sessionID,
				FirstMatch: ts,
				LastMatch:  ts,
			}
		}

		g := groups[sessionID]
		if ctx, _ := obj["context"].(bool); !ctx {
			g.MatchCount++
		}
		g.Turns = append(g.Turns, entry)
		if ts != "" && (g.FirstMatch == "" || ts < g.FirstMatch) {
			g.FirstMatch = ts
		}
		if ts != "" && ts > g.LastMatch {
			g.LastMatch = ts
		}
	}

	result := make([]interface{}, 0, len(order))
	for _, id := range order {
		g := groups[id]
		result = append(result, map[string]interface{}{
			"session_id":  g.SessionID,
			"match_count": g.MatchCount,
			"first_match": g.FirstMatch,
			"last_match":  g.LastMatch,
			"turns":       g.Turns,
		})
	}
	return result
}

// splitJSONLLines splits a JSONL string into non-empty lines.
func splitJSONLLines(jsonlData string) []string {
	trimmed := strings.TrimSpace(jsonlData)
	if trimmed == "" {
		return nil
	}
	var lines []string
	for _, line := range strings.Split(trimmed, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// GenerateSessionStats generates per-session statistics from JSONL data (rawData, camelCase).
// Output: summary line first, then one line per session ordered by first_match ascending.
func GenerateSessionStats(jsonlData string) (string, error) {
	type sessionAgg struct {
		SessionID string
		Count     int
		First     time.Time
		Last      time.Time
	}

	var order []string
	sessions := make(map[string]*sessionAgg)
	var overallFirst, overallLast time.Time
	firstOverall := true

	for _, line := range splitJSONLLines(jsonlData) {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		sessionID, _ := obj["sessionId"].(string)
		tsStr, _ := obj["timestamp"].(string)

		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			ts, err = time.Parse("2006-01-02T15:04:05.000Z", tsStr)
		}
		if err != nil {
			continue
		}

		if _, exists := sessions[sessionID]; !exists {
			order = append(order, sessionID)
			sessions[sessionID] = &sessionAgg{SessionID: sessionID, First: ts, Last: ts}
		}
		s := sessions[sessionID]
		s.Count++
		if ts.Before(s.First) {
			s.First = ts
		}
		if ts.After(s.Last) {
			s.Last = ts
		}

		if firstOverall || ts.Before(overallFirst) {
			overallFirst = ts
			firstOverall = false
		}
		if overallLast.IsZero() || ts.After(overallLast) {
			overallLast = ts
		}
	}

	if len(sessions) == 0 {
		return "", nil
	}

	sort.Slice(order, func(i, j int) bool {
		return sessions[order[i]].First.Before(sessions[order[j]].First)
	})

	totalMatches := 0
	for _, s := range sessions {
		totalMatches += s.Count
	}

	var output strings.Builder

	summary := map[string]interface{}{
		"total_sessions": len(sessions),
		"total_matches":  totalMatches,
		"time_range": map[string]interface{}{
			"from": overallFirst.UTC().Format(time.RFC3339),
			"to":   overallLast.UTC().Format(time.RFC3339),
		},
	}
	summaryBytes, _ := json.Marshal(summary)
	output.Write(summaryBytes)
	output.WriteString("\n")

	for _, id := range order {
		s := sessions[id]
		durationMin := int(s.Last.Sub(s.First).Minutes())
		sess := map[string]interface{}{
			"session_id":       s.SessionID,
			"match_count":      s.Count,
			"first_match":      s.First.UTC().Format(time.RFC3339),
			"last_match":       s.Last.UTC().Format(time.RFC3339),
			"duration_minutes": durationMin,
		}
		sessBytes, _ := json.Marshal(sess)
		output.Write(sessBytes)
		output.WriteString("\n")
	}

	return output.String(), nil
}

# Query UX Improvements Proposal

**Status**: Draft
**Date**: 2026-03-09
**Author**: Claude Code Analysis

## Background

During retrospective analysis of 48-hour session history using `query_user_messages`, three friction points were observed that required significant manual effort to work around:

1. **Time filtering via jq is unreliable**: The recommended workaround is to add a jq expression like `.[] | select(.timestamp >= "2026-03-07T00:00:00Z")`. This is unreliable because jq string comparison is lexicographic and depends on consistent UTC formatting. Callers receive un-filtered results with no warning when the format doesn't match.

2. **`session_id` not surfaced in summary mode**: Each raw JSONL entry already contains `sessionId` (from the CC session format). However, when `content_summary: true` is used, only `turn_sequence`, `timestamp`, and `content_preview` are returned — `sessionId` is silently dropped. Callers who want to group by session must either use full output (large) or infer session boundaries heuristically.

3. **`stats_first` is tool-centric and meaningless for user messages**: The current `GenerateStats()` function (`internal/query/jq.go`) groups records by their `tool` or `ToolName` field. User message records have neither field, so every record is bucketed as `"unknown"`. Calling `stats_first: true` on `query_user_messages` returns useless stats.

---

## Architecture Context

Understanding the actual call path is essential to placing fixes correctly.

### Actual `query_user_messages` call chain

```
MCP client
  → server.go: handleToolsCall("query_user_messages", args)
  → executor.go: ExecuteTool → handleQueryUserMessages(cfg, scope, args)
  → handlers_convenience.go: executeQuery(scope, jqFilter, limit, workingDir)
  → handlers_query.go: executor.streamFiles(ctx, files, compiledJQ, limit)
```

The function `RunUserMessagesQuery` in `internal/query/messages.go` is **not on this path**. It exists as a standalone function but is not called by any MCP handler. Fixes must target `handlers_convenience.go` and `handlers_query.go`, not `internal/query/messages.go`.

### `stats_first` / `stats_only` architecture

`GenerateStats()` in `internal/query/jq.go` is a shared generic function called by `buildStatsFirstResponse()` in `executor.go` for **all** tools without awareness of which tool is being served. It groups by `tool`/`ToolName` field. For user messages (no such field), this produces only `{"key": "unknown", "count": N}`.

### `sessionId` in raw data

`SessionEntry.SessionID` (`json:"sessionId"`) is populated from the CC JSONL format. When `executeQuery()` streams raw entries through jq, `sessionId` is present on every record and **already accessible** via `.sessionId` in user-supplied jq filters. The field is not missing — it is dropped only in the `content_summary` output path.

---

## Proposed Changes

### Change 1: Native `since` / `until` Parameters (Go-level time filter)

**Affected tools**: `query_user_messages`, `query_timestamps`, `query_conversation_flow` (all tools that operate on time-ordered records)

**Problem**: jq string comparison on timestamps is unreliable. A Go-level `time.Parse` filter is correct by definition and returns an actionable error on malformed input.

**New parameters** (added to `handleQueryUserMessages` and shared via common args extraction):

| Parameter | Type   | Example |
|-----------|--------|---------|
| `since`   | string | `"2026-03-07T00:00:00Z"` |
| `until`   | string | `"2026-03-09T00:00:00Z"` |

**Behavior**:
- Both are optional and independent.
- Malformed value → explicit error: `invalid since value: cannot parse "2026-03-07" as RFC3339`.
- Filtering happens **before** jq in Go, on raw parsed entries.
- Existing `jq_filter` continues to work; `since`/`until` are a pre-filter.

**Implementation location**:

`handlers_query.go`: Extend `executeQuery()` to accept a `TimeRange` struct (or add `since`/`until time.Time` parameters). Filter entries in `streamFiles()` by `entry.timestamp` before feeding to jq.

Alternatively, introduce `executeQueryWithOptions(opts queryOptions)` where `queryOptions` embeds the existing parameters plus time bounds, keeping the existing `executeQuery()` signature as a backward-compatible wrapper.

`handlers_convenience.go`: In `handleQueryUserMessages()`, extract and parse `since`/`until` from `args`, pass to the extended `executeQuery`.

**Error handling**: `time.Parse(time.RFC3339, value)` — return error immediately on parse failure. Do not fall through to jq.

**Complexity: Low.** The filter logic is a simple time comparison loop in `streamFiles()`. No new packages required.

---

### Change 2: Surface `session_id` in `content_summary` Mode

**Affected**: `query_user_messages` (and any tool using `content_summary: true`)

**Problem**: `sessionId` is present in every raw JSONL record and is already accessible via `.sessionId` in jq. However, the `content_summary` output mode silently omits it, which forces callers who want session grouping to either:
- Fetch full output (expensive for large result sets), or
- Accept that summary results cannot be grouped by session.

**Fix**: Add `session_id` to the `content_summary` output record. The value is taken directly from `.sessionId` of the raw entry.

**Before** (`content_summary: true` output — 4 fields):
```json
{"turn_sequence": 12, "uuid": "04305e6d-...", "timestamp": "2026-03-09T11:16:30Z", "content_preview": "修改上面的方案..."}
```

**After** (5 fields):
```json
{"session_id": "d3ea683a-a5f6-430a-8fe3-cd3a55cd247f", "turn_sequence": 12, "uuid": "04305e6d-...", "timestamp": "2026-03-09T11:16:30Z", "content_preview": "修改上面的方案..."}
```

**Implementation location**: `cmd/mcp-server/filters.go`, function `ApplyContentSummary()` (lines ~91–123). This function builds the 4-field summary map `{turn_sequence, uuid, timestamp, content_preview}` for each entry. Add `session_id` sourced from the entry's `sessionId` key.

**Documentation**: The tool description for `query_user_messages` should explicitly list `.sessionId` as an available field for use in `jq_filter` expressions, since callers cannot discover it without reading the source or raw output.

**Non-change**: No new pipeline infrastructure needed. No changes to `SessionEntry`, `parser`, or `session.go`. `sessionId` is already in the data.

**Complexity: Low.** Adding one field to an existing transform.

---

### Change 3: Tool-Aware Stats for `stats_first` / `stats_only`

**Affected**: `query_user_messages` (and more broadly: all non-tool-call tools)

**Problem**: `GenerateStats()` in `jq.go` is hardcoded to group by `tool`/`ToolName`. For user messages, it produces only `{"key": "unknown", "count": 200}` — useless.

**Root cause**: `buildStatsFirstResponse()` in `executor.go` calls `GenerateStats()` without knowing the tool type. The stats function has no fallback for records that don't have a `tool` field.

**Fix**: Make `GenerateStats()` fall back to time-based bucketing when no `tool`/`ToolName` field is present. Specifically:

- If all (or most) records lack a `tool` field, group by **hour** using the `timestamp` field.
- Return additional summary fields alongside the existing `{key, count}` format.

**Proposed enhanced output for user messages** (`stats_only: true`):

```
{"total": 200, "session_count": 12, "time_range": {"from": "2026-03-08T06:01Z", "to": "2026-03-09T13:49Z"}}
{"hour": "2026-03-08T06", "count": 8}
{"hour": "2026-03-08T09", "count": 6}
{"hour": "2026-03-09T03", "count": 9}
...
```

**Detection logic** (no LLM): In `GenerateStats()`, after parsing records, check if any record has a `tool` or `ToolName` key. If fewer than 10% do, switch to time-based bucketing. This is a deterministic heuristic.

**Implementation location**: `internal/query/jq.go` — `GenerateStats()`. Add a second pass that checks for the presence of `tool`/`ToolName` and conditionally switches to timestamp-based grouping.

Alternatively, add a separate `GenerateTimestampStats()` function and call it from `buildStatsFirstResponse()` based on tool name — which is available in `executor.go` at call time.

**The cleaner approach** (preferred): In `executor.go`, `buildStatsFirstResponse()` already receives `toolName`. Add a conditional:
```go
if toolName == "query_user_messages" || toolName == "query_conversation_flow" {
    stats, _ = querypkg.GenerateTimestampStats(jsonlData)
} else {
    stats, _ = querypkg.GenerateStats(jsonlData)
}
```

This avoids heuristics entirely, keeps `GenerateStats()` unchanged, and is explicit about which tools need what kind of stats.

**Complexity: Low-Medium.** Adding `GenerateTimestampStats()` and one conditional in `executor.go`.

---

## Non-Goals

- **`query_sequence` / workflow detection**: Out of scope for this proposal.
- **LLM-based semantic classification**: meta-cc does not use LLM. All computation is deterministic.
- **Modifying the JSONL format or `SessionEntry`**: `sessionId` is already present. No parser changes needed.
- **`by_hour` as configurable granularity**: Fixed hour bucketing is sufficient for 1–48h windows. A `stats_interval` parameter is deferred.

---

## Impact Assessment

| Change | Files Changed | Estimated LoC | Breaking Change |
|--------|---------------|---------------|-----------------|
| `since`/`until` params | `handlers_query.go`, `query_executor.go`, `handlers_convenience.go`, `tools.go` | ~120 | No (additive) |
| `session_id` in summary | `filters.go`, `filters_test.go`, `tools.go` | ~20 | No (additive field) |
| Tool-aware stats | `jq.go` (new fn), `jq_test.go`, `executor.go` | ~80 | No (existing `GenerateStats` unchanged) |

All three changes are additive. Existing callers are unaffected.

---

## Success Criteria

1. `query_user_messages(since: "2026-03-07T00:00:00Z")` returns only records with `timestamp >= 2026-03-07T00:00:00Z`, verified by test with a fixture containing records both inside and outside the window.
2. A malformed `since` value (e.g., `"2026-03-07"`, no time component) returns an explicit error message, not empty results.
3. `query_user_messages(content_summary: true)` output includes `session_id` in each record.
4. `query_user_messages(stats_only: true)` returns time-bucketed stats (by hour), not `{"key": "unknown", "count": N}`.
5. `query_tool_errors(stats_only: true)` continues to return tool-name-grouped stats (existing behavior unchanged).

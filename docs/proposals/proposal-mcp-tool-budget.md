# MCP Tool Budget: query_* Consolidation and Edit-Sequence Addition

> Status: Draft (rev 1)
> Scope: Consolidate the fragmented query_* tool group (10 tools → 3); then add
>        query_edit_sequences; target ≤ 17 tools total
> Branch: `feat/mcp-tool-budget` (future)

---

## Background

meta-cc currently exposes **21 MCP tools** (as implemented in `proposal-project-split.md`,
Phase 41–46). Combined with archguard's 23 tools, the total loaded at session start is
approximately **44 tool schemas**.

### Established constraints

Per `proposal-project-split.md` Section 4 (the authoritative reference for meta-cc):

> "The MCP specification (2025-11-25) defines no hard limit on tool count. Client-side
> limits vary: **Cursor enforces 40, VS Code 128**. Claude Code activates automatic Tool
> Search (deferred loading) when tool definitions exceed **10% of the context window**,
> so 21 tools is well within safe operating range."

The complementary archguard constraint (from `docs/adr/006-mcp-tool-design-standards.md`
and `proposal-multi-paradigm-mcp-tools.md`):

> "Budget unit is tokens, not tool count. Claude Code warns when MCP tool definitions
> exceed **25,000 tokens**."

Claude Code's deferred loading (ToolSearch) is a real mitigating mechanism: it activates
before the absolute limit is reached. However, deferred loading adds a round-trip (the
LLM must call ToolSearch to fetch a tool's schema before using it), so it is a fallback,
not a design target.

### The structural problem: fragmentation, not quantity

The issue with meta-cc's current tool set is not that it has too many tools by absolute
count — 21 tools is within safe range per the documented constraints above. The issue is
that 10 of those 21 tools (`query_*`) are **structurally fragmented**: they perform the
same fundamental operation (filter JSONL by event type) but are exposed as separate tools
with near-identical schemas. This wastes token budget on redundant schema definitions and
creates selection noise when the LLM has to choose between `query_tool_errors`,
`query_system_errors`, `query_tool_blocks`, and `query_conversation_flow` for what is
conceptually one query operation.

### Usage Concentration

Session analysis of the archguard development project shows meta-cc usage is highly
concentrated:

| Tool | Session calls |
|---|---|
| `query_user_messages` | 20 |
| `query_tool_blocks` | 17 |
| `query_summaries` | 7 |
| `analyze_errors` | 3 |
| `get_work_patterns` | 3 |
| `query_tools` | 3 |
| `quality_scan` | 1 |
| `query_conversation_flow` | 1 |
| **13 tools** | **0** |

Thirteen tools — 62% of the set — have zero recorded calls in this project's history.

---

## Problem: Structural Fragmentation of `query_*`

The 10 `query_*` tools (`query_tool_errors`, `query_token_usage`, `query_system_errors`,
`query_file_snapshots`, `query_timestamps`, `query_summaries`, `query_tool_blocks`,
`query_tools`, `query_user_messages`, `query_conversation_flow`) all perform the same
fundamental operation: **filter session JSONL by event type and return matching records**.

The only difference between them is which event type they default to. They share:
- The same underlying data source (JSONL tool_use and text blocks)
- The same pagination / limit mechanism
- The same session-scope parameter
- The same inline / file_ref hybrid output

Exposing them as 10 separate tools means:
1. The LLM sees 10 near-identical tool schemas, increasing selection noise
2. Cross-type queries require multiple round-trips instead of one call
3. Adding a new event type requires a new tool (accumulation by default)

The root cause is **early-phase API design**: each new signal type became a new tool
rather than a parameter of one flexible query tool.

---

## Proposed Consolidation: 10 → 3

Replace the 10 `query_*` tools with 3 semantically distinct tools:

### Tool 1: `query_session_content`

Replaces: `query_summaries`, `query_tool_blocks`, `query_user_messages`,
           `query_conversation_flow`

These all query **message-level content** from the session JSONL — the back-and-forth
between user, assistant, and tools.

```go
type QuerySessionContentInput struct {
    Role     string   `json:"role"`     // "user" | "assistant" | "tool" | "all" (default: "all")
    Contains string   `json:"contains"` // substring filter (optional)
    Limit    int      `json:"limit"`    // max results (default: 50)
    Scope    string   `json:"scope"`    // "project" | "session" (default: "project")
}
```

The `role` parameter replaces the per-role tool split. `role="tool"` covers `query_tool_blocks`;
`role="user"` covers `query_user_messages`; `role="all"` covers `query_conversation_flow`;
no `role` filter with `contains="## Summary"` covers `query_summaries`.

### Tool 2: `query_session_signals`

Replaces: `query_tool_errors`, `query_token_usage`, `query_system_errors`,
           `query_timestamps`, `query_tools`

These all query **diagnostic signals** derived from the JSONL — errors, performance data,
tool usage statistics, timeline events.

```go
type QuerySessionSignalsInput struct {
    Type    string `json:"type"`    // "errors" | "tokens" | "system_errors" | "timestamps" | "tool_stats" | "all"
    Limit   int    `json:"limit"`   // max results (default: 50)
    Scope   string `json:"scope"`   // "project" | "session" (default: "project")
}
```

`type="errors"` covers `query_tool_errors`; `type="tokens"` covers `query_token_usage`;
`type="system_errors"` covers `query_system_errors`; `type="timestamps"` covers
`query_timestamps`; `type="tool_stats"` covers `query_tools`.

### Tool 3: `query_file_activity`

Replaces: `query_file_snapshots`

File-centric queries stand apart from message-content and signal queries: they group data
by file path rather than by message role or signal type. This tool is named to reflect
its purpose clearly, and structured to support the forthcoming `query_edit_sequences`
output in a complementary rather than overlapping way.

```go
type QueryFileActivityInput struct {
    Files   []string `json:"files"`   // filter by file path (optional, returns all if empty)
    Type    string   `json:"type"`    // "snapshots" | "edits" | "reads" | "all" (default: "all")
    Limit   int      `json:"limit"`   // max results per file (default: 20)
    Scope   string   `json:"scope"`   // "project" | "session" (default: "project")
}
```

`type="snapshots"` covers `query_file_snapshots`. The additional `type` values enable
future extension without new tool slots.

---

## Tools Retained Unchanged

The non-`query_*` tools are lower-frequency but semantically distinct. They stay as-is:

| Tool | Rationale |
|---|---|
| `analyze_errors` | Analysis (structured output), not raw query |
| `analyze_bugs` | Analysis (structured output), not raw query |
| `get_session_directory` | Utility — returns path for downstream tools |
| `get_session_metadata` | Metadata — session ID, date, project root |
| `get_tech_debt` | Aggregate report, distinct output shape |
| `get_timeline` | Chronological view, distinct output shape |
| `get_work_patterns` | Behavioral aggregate, distinct output shape |
| `quality_scan` | Cross-session scan, distinct purpose |
| `inspect_session_files` | File listing, utility |
| `cleanup_temp_files` | Maintenance, utility |
| `execute_stage2_query` | Two-stage large-output pipeline |

---

## New Tool Addition: `query_edit_sequences`

After the consolidation, add `query_edit_sequences` as designed in
`proposal-edit-sequence-tool.md`. This tool returns chronologically ordered Read/Edit
events per file with Pattern A/B/C classification — a capability that does not exist in
any current tool and cannot be approximated by the consolidated `query_file_activity`.

`query_file_activity` returns what happened (snapshots); `query_edit_sequences` returns
the temporal shape of how the LLM interacted with a file (sequences + pattern hint).
These are complementary, not overlapping.

---

## Net Result

| State | Tool count |
|---|---|
| Current | 22 |
| After removing 10 `query_*` tools | 12 |
| After adding 3 consolidated query tools | 15 |
| After adding `query_edit_sequences` | **16** |

Combined with archguard's target of 20 tools: **36 tools total** (down from 45).

### Proposed Tool Inventory (post-change)

**Content Queries** *(consolidated)*
- `query_session_content` — messages by role (user / assistant / tool / all)
- `query_session_signals` — diagnostic signals (errors / tokens / timestamps / tool_stats)
- `query_file_activity` — file-centric snapshots and access records

**Behavioral Analysis** *(new)*
- `query_edit_sequences` — ordered Read/Edit timeline + Pattern A/B/C hint per file

**Aggregate Analysis**
- `analyze_errors` — structured error analysis
- `analyze_bugs` — bug pattern detection
- `get_tech_debt` — tech debt report
- `get_timeline` — session timeline
- `get_work_patterns` — behavioral aggregate
- `quality_scan` — cross-session quality scan

**Utilities**
- `get_session_directory` — session path lookup
- `get_session_metadata` — session metadata
- `inspect_session_files` — list JSONL files
- `cleanup_temp_files` — temp file cleanup
- `execute_stage2_query` — two-stage output pipeline

---

## Compatibility

The 10 `query_*` tools are called by:
1. **External LLM sessions** — no code dependency; the LLM selects tools by name at runtime.
   The new tool names are more descriptive and the parameter semantics are equivalent.
2. **Integration tests** — any tests that call `query_tool_errors`, etc. by name must be
   updated to use the new tool names and `type` parameters.
3. **Documentation** — `docs/guides/mcp-query-tools.md` must be rewritten.

No internal Go code calls these tools directly (they are MCP handlers, not library
functions). The consolidation is a handler rename + parameter addition, not a logic rewrite:
the underlying filtering logic per event type is preserved and routed via the `type`
parameter.

### Migration mapping (for test updates)

| Old tool call | New equivalent |
|---|---|
| `query_tool_errors({})` | `query_session_signals({type:"errors"})` |
| `query_token_usage({})` | `query_session_signals({type:"tokens"})` |
| `query_system_errors({})` | `query_session_signals({type:"system_errors"})` |
| `query_timestamps({})` | `query_session_signals({type:"timestamps"})` |
| `query_tools({})` | `query_session_signals({type:"tool_stats"})` |
| `query_summaries({})` | `query_session_content({role:"assistant", contains:"## Summary"})` |
| `query_tool_blocks({})` | `query_session_content({role:"tool"})` |
| `query_user_messages({})` | `query_session_content({role:"user"})` |
| `query_conversation_flow({})` | `query_session_content({role:"all"})` |
| `query_file_snapshots({})` | `query_file_activity({type:"snapshots"})` |

---

## Plan

| Phase | Work |
|---|---|
| 1 | Implement `query_session_content` handler with `role` parameter routing |
| 2 | Implement `query_session_signals` handler with `type` parameter routing |
| 3 | Implement `query_file_activity` handler with `type` parameter routing |
| 4 | Remove 10 old `query_*` tool registrations from `internal/mcp/tools/tools.go` |
| 5 | Update integration tests: replace old tool call names with new equivalents |
| 6 | Implement `query_edit_sequences` (see `proposal-edit-sequence-tool.md`) |
| 7 | Rewrite `docs/guides/mcp-query-tools.md` with new tool set |
| 8 | Bump schema version in tool manifest |

Phases 1–5 (consolidation) and Phase 6 (`query_edit_sequences`) are independent
and can be delivered separately.

---

## Relationship to Other Proposals

```
proposal-edit-sequence-tool.md       ← ADDS query_edit_sequences (Phase 6)
proposal-doc-session-signals.md      ← EXTENDS query_edit_sequences with doc signals
```

The consolidation (Phases 1–5) does not depend on either of those proposals and can
ship first.

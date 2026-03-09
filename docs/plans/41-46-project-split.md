# Plan 41–46: Project Split — `yaleh/baime` Creation and meta-cc Refactoring

## Overview

Implement the project split described in [docs/proposals/proposal-project-split.md](../proposals/proposal-project-split.md).

**Six phases**:

| Phase | Scope | Key deliverable |
|-------|-------|-----------------|
| 41 | Create `yaleh/baime` repository | baime published on Claude Code plugin marketplace |
| 42 | `analyze_errors` + `quality_scan` MCP tools | 2 Go analysis tools live in meta-cc |
| 43 | `get_work_patterns` + `get_timeline` MCP tools | 2 Go analysis tools live in meta-cc |
| 44 | `analyze_bugs` + `get_tech_debt` MCP tools | 2 Go analysis tools live in meta-cc |
| 45 | Remove capability loading system | ~3,349 lines of Go deleted; `/meta` removed |
| 46 | Plugin pruning → meta-cc 3.0.0 | Skills/agents removed; version 3.0.0 released |

**Phase dependencies**:

```
Phase 41 (baime) ───────────────────────────────────────────► verified published before Phase 45
Phase 42 (analyze_errors + quality_scan) ──────────────────┐
Phase 43 (get_work_patterns + get_timeline) ───────────────┤► all complete before Phase 45
Phase 44 (analyze_bugs + get_tech_debt) ───────────────────┘
Phase 45 (remove capability loading) ──────────────────────► complete before Phase 46
Phase 46 (plugin pruning → 3.0.0)
```

**Parallel execution of Phases 42–44**: Data-layer stages (42.1, 43.1, 44.1) can run in parallel git worktrees since they create files in different locations (`errors_analysis.go`, `work_patterns.go`, `bugs_analysis.go`). However, MCP registration stages (42.2, 43.2, 44.2 etc.) all modify `cmd/mcp-server/tools.go` and `cmd/mcp-server/executor.go` — these must be merged sequentially to avoid conflicts in those two files. Designate one integration commit after all data layers are complete.

**Architectural invariant**: Go code owns data extraction and aggregation; returns `[]byte` JSON. Claude owns semantic interpretation and visualization rendering.

**Type convention**: All new analyzer functions accept `[]parser.SessionEntry` (from `internal/parser/`) and `[]parser.ToolCall`, matching the existing `CalculateStats`, `DetectToolSequences`, and `DetectErrorPatterns` function signatures in `internal/analyzer/`.

**Development methodology**: TDD throughout. Data-layer stages write tests that **will not compile** until the implementation file is created — this is correct Go TDD behavior. Each stage begins with a BUILD FAIL; implementation follows to achieve passing tests.

**Code limits**: Phase ≤500 lines, Stage ≤200 lines of code modifications.

---

## Phase 41: Create `yaleh/baime` Repository

**Goal**: Establish the `yaleh/baime` plugin repository with all migrated content from meta-cc, validate it as a standalone plugin, and publish to the Claude Code plugin marketplace.

**Note**: This phase operates in a new GitHub repository (`github.com/yaleh/baime`). All stage work targets that repo. Completion of Phase 41 is tracked by the public availability of the plugin — Phase 45 must not begin until the plugin is installable via `/plugin marketplace add yaleh/baime`.

**Estimated content**: ~250 lines (manifests + CI config); skill/agent content copied from meta-cc

### Stage Dependencies

```
41.1 (repo structure + plugin manifests — initial counts)
  └──► 41.2 (copy 18 skills + 5 agents)
  └──► 41.3 (migrate 2 capabilities → 1 new agent + 1 new skill; update counts to final)
  └──► 41.4 (CI + YAML frontmatter validation + marketplace publish)
```

---

### Stage 41.1 — Repository Structure and Plugin Manifests

**Goal**: Create the `yaleh/baime` repo with standard plugin structure, initial `plugin.json`, and `marketplace.json`.

**TDD sequence**:

1. Create repo `yaleh/baime` with directory structure:
   ```
   .
   ├── .claude-plugin/
   │   └── marketplace.json
   ├── .claude/
   │   ├── .claude-plugin/
   │   │   └── plugin.json
   │   ├── agents/
   │   └── skills/
   └── README.md
   ```
2. Write `marketplace.json` (name: `baime`, version: `1.0.0`; agent and skill counts intentionally omitted until Stage 41.3 finalises them)
3. Write `plugin.json` with empty `agents` and `skills` arrays; no `mcpServers` field
4. Write minimal validation script asserting JSON validity and version parity
5. Run validation — expect PASS

**Note**: Agent and skill counts in the manifests are placeholder (`0`) at this stage and will be updated in Stages 41.2 and 41.3 to reflect the actual files present.

**Key files** (new repo):
- `.claude-plugin/marketplace.json` — CREATE
- `.claude/.claude-plugin/plugin.json` — CREATE

**Acceptance criteria**:
- Both manifests are valid JSON with version parity
- No `mcpServers` field in `plugin.json`
- Validation script passes

**Line budget**: ≤100 lines

---

### Stage 41.2 — Copy Skills and Agents from meta-cc

**Goal**: Populate `yaleh/baime` with all 18 skills and 5 published agents.

**TDD sequence**:

1. Write count-validation assertions: 18 skill directories each containing `SKILL.md`, exactly 5 agent `.md` files
2. Copy `.claude/skills/*` from meta-cc into `yaleh/baime/.claude/skills/`
3. Copy 5 published agents (`stage-executor`, `project-planner`, `iteration-executor`, `iteration-prompt-designer`, `knowledge-extractor`) into `yaleh/baime/.claude/agents/`
4. Verify `feature-developer.md` and `phase-planner-executor.md` are NOT present
5. Update `plugin.json` and `marketplace.json`: `agents` = 5, `skills` = 18
6. Run validation — expect PASS

**Key files** (new repo):
- `.claude/skills/*` — COPY FROM meta-cc (18 directories)
- `.claude/agents/*.md` — COPY FROM meta-cc (5 files)
- `.claude/.claude-plugin/plugin.json` — UPDATE

**Acceptance criteria**:
- 18 skill directories with `SKILL.md`, 5 agent `.md` files
- `feature-developer.md` and `phase-planner-executor.md` absent
- `plugin.json` arrays reflect actual counts
- Validation passes

**Line budget**: ≤60 lines (manifest updates only; content is copied)

---

### Stage 41.3 — Migrate Capabilities to baime

**Goal**: Author `workflow-coach.md` agent (from `meta-coach`) and `next-step-generation/SKILL.md` (from `meta-next`). Absorb `meta-prompt` methodology into `methodology-bootstrapping`. Update manifests to final counts (6 agents, 19 skills).

**Note**: `meta-coach.md` is 332 lines of instructions with 8 explicit `mcp_meta_cc.*` tool call sites. `workflow-coach.md` must be authored from scratch as a **standalone** coaching agent that degrades gracefully when meta-cc is not installed. This is a significant authoring task, not a copy-and-edit.

**TDD sequence**:

1. Write validation assertions: `workflow-coach.md` present and contains no hard `mcp_meta_cc` dependency markers; `next-step-generation/SKILL.md` present and contains no `mcp_` tool calls
2. Author `workflow-coach.md`:
   - Describe coaching methodology without MCP calls in the core flow
   - Add optional enrichment section: "If meta-cc is installed, you may call `mcp_meta_cc.query_conversation_flow()` etc. to ground recommendations in session data; if not available, skip and proceed with general coaching"
3. Author `next-step-generation/SKILL.md` based on `meta-next` (no MCP execution — pure reasoning)
4. Add prompt-refinement section to `methodology-bootstrapping/SKILL.md`
5. Update `plugin.json` and `marketplace.json` to final counts: 6 agents, 19 skills
6. Run validation — expect PASS

**Key files** (new repo):
- `.claude/agents/workflow-coach.md` — CREATE
- `.claude/skills/next-step-generation/SKILL.md` — CREATE
- `.claude/skills/methodology-bootstrapping/SKILL.md` — UPDATE
- `.claude/.claude-plugin/plugin.json` — UPDATE (6 agents, 19 skills)

**Acceptance criteria**:
- `workflow-coach.md` has no unconditional `mcp_meta_cc.*` calls; optional enrichment section is clearly delineated
- `next-step-generation/SKILL.md` has zero `mcp_` references
- Final counts: 6 agents, 19 skills
- Validation passes

**Line budget**: ≤350 lines (`workflow-coach.md` alone will be ~250 lines; new skill ~60 lines; manifest updates ~20 lines)

---

### Stage 41.4 — CI, YAML Validation, and Marketplace Publication

**Goal**: Set up CI with JSON and YAML frontmatter validation; publish to Claude Code plugin marketplace.

**TDD sequence**:

1. Write `scripts/validate-plugin.sh`:
   - JSON validity and version parity (already tested in 41.1)
   - **YAML frontmatter validation**: for each `SKILL.md` and agent `.md`, assert `name`, `description` fields present and non-empty using `python3 -c "import yaml; ..."` or `yq`
   - Agent count = 6, skill count = 19
2. Run validation — expect PASS
3. Set up `.github/workflows/ci.yml`: run `validate-plugin.sh` on push/PR
4. Write `README.md` with tagline, install instructions, and cross-reference to meta-cc
5. Submit plugin to Claude Code plugin marketplace

**Key files** (new repo):
- `scripts/validate-plugin.sh` — CREATE
- `.github/workflows/ci.yml` — CREATE
- `README.md` — CREATE

**Acceptance criteria**:
- YAML frontmatter validation catches missing `name` or `description` fields
- CI passes on push
- Plugin installable via `/plugin marketplace add yaleh/baime`

**Line budget**: ≤150 lines

---

## Phase 42: `analyze_errors` + `quality_scan` MCP Tools

**Goal**: Implement the first two Go analysis tools. Both follow the same integration pattern, which is established here and reused in Phases 43 and 44.

**Session data loading pattern** (established in Stage 42.2 and reused in all subsequent MCP registration stages):

```go
// In cmd/mcp-server/handlers_analysis.go (new file)
func loadEntriesAndToolCalls(cfg *config.Config, args map[string]interface{}) (
    []parser.SessionEntry, []parser.ToolCall, error) {
    // 1. Resolve scope from args["scope"] defaulting to "project"
    // 2. Locate JSONL file(s) via cfg and working directory
    // 3. Call parser.ParseSessionFile() or equivalent
    // 4. Call parser.ExtractToolCalls(entries)
    // 5. Return entries, toolCalls, nil
}
```

All new analysis tool handlers call `loadEntriesAndToolCalls` first, then pass the result to the relevant `internal/analyzer` function.

**Estimated code**: ~420–460 lines (implementation + tests, including the shared loader)

### Stage Dependencies

```
42.1 (analyze_errors data layer: internal/analyzer)
  └──► 42.2 (analyze_errors MCP registration + loadEntriesAndToolCalls helper)
42.3 (quality_scan data layer + MCP registration)   [independent of 42.1]
─── merge all into main before Phase 45 ───
```

---

### Stage 42.1 — `analyze_errors`: Data Layer

**Goal**: Implement error aggregation in `internal/analyzer/errors_analysis.go` using `[]parser.SessionEntry`.

**TDD sequence**:

1. Create shared test helper `internal/analyzer/testutil_test.go` with `makeEntries(...)` helper that constructs `[]parser.SessionEntry` fixtures — this file is reused by all subsequent analyzer test files:
   ```go
   func makeEntries(toolName string, status string, errMsg string) []parser.SessionEntry { ... }
   ```
2. Write `internal/analyzer/errors_analysis_test.go`:
   - Test: groups tool errors by tool name → correct counts per tool
   - Test: groups errors by error type substring → correct counts per type
   - Test: surfaces up to N example messages per group
   - Test: returns correct time range from entry timestamps
   - Test: empty session → zero counts, no error
3. Run `go test ./internal/analyzer/...` — expect **BUILD FAIL** (`analyzer.AnalyzeErrors` undefined)
4. Implement `internal/analyzer/errors_analysis.go`:
   ```go
   type ErrorAnalysisResult struct {
       TimeRange   TimeRange        `json:"time_range"`
       TotalErrors int              `json:"total_errors"`
       ByTool      []ToolErrorGroup `json:"by_tool"`
       ByType      []ErrorTypeGroup `json:"by_type"`
   }
   func AnalyzeErrors(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*ErrorAnalysisResult, error)
   ```
   Reuse `CalculateErrorSignature()` from `errors.go` for type grouping.
5. Run tests — expect PASS
6. Run `make commit`

**Key files**:
- `internal/analyzer/testutil_test.go` — CREATE (shared fixture helper)
- `internal/analyzer/errors_analysis.go` — CREATE
- `internal/analyzer/errors_analysis_test.go` — CREATE

**Acceptance criteria**:
- All 5 unit tests pass; no semantic classification in Go code
- `make commit` passes

**Line budget**: ≤200 lines (testutil ~40 + test file ~100 + impl ~60)

---

### Stage 42.2 — `analyze_errors`: MCP Registration

**Goal**: Create `cmd/mcp-server/handlers_analysis.go` with the shared `loadEntriesAndToolCalls` helper and the `analyze_errors` handler; register the tool.

**TDD sequence**:

1. Write `cmd/mcp-server/analysis_errors_test.go`:
   - Test: tool present in `getToolDefinitions()` return value
   - Test: calling tool against test JSONL returns valid JSON with `total_errors` field
   - Test: `limit` parameter is respected
2. Run tests — expect **BUILD FAIL**
3. Create `cmd/mcp-server/handlers_analysis.go`:
   - Implement `loadEntriesAndToolCalls(cfg, args)` helper (~40 lines)
   - Implement `executeAnalyzeErrorsTool(cfg, args)` handler (~30 lines)
4. Add tool definition to `cmd/mcp-server/tools.go` using `buildTool(...)` (~25 lines)
5. Add case to `executeSpecialTool` in `cmd/mcp-server/executor.go` (~15 lines), calling `recordToolSuccess`/`recordToolFailure`
6. Run tests — expect PASS
7. Run `make commit`

**Key files**:
- `cmd/mcp-server/handlers_analysis.go` — CREATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)
- `cmd/mcp-server/analysis_errors_test.go` — CREATE (~70 lines)

**Acceptance criteria**:
- `analyze_errors` appears in MCP `tools/list` response
- Returns valid `ErrorAnalysisResult` JSON from test session data
- `make commit` passes

**Line budget**: ≤200 lines

---

### Stage 42.3 — `quality_scan`: Full Implementation

**Goal**: Implement `quality_scan` data layer and MCP registration in a single stage.

**TDD sequence**:

1. Write `internal/analyzer/quality_analysis_test.go` (reuses `testutil_test.go`):
   - Test: error_rate = error_count / total_tool_calls (value in [0,1])
   - Test: retry_rate = retried_operations / total_operations
   - Test: tool_diversity = unique_tools / max(unique_tools_seen, 1)
   - Test: completion_rate = successful_calls / total_calls
   - Test: all dimensions present in result
2. Run tests — expect **BUILD FAIL**
3. Implement `internal/analyzer/quality_analysis.go`:
   ```go
   type QualityScanResult struct {
       Dimensions []QualityDimension `json:"dimensions"`
   }
   type QualityDimension struct {
       Name     string  `json:"name"`
       Score    float64 `json:"score"`    // 0.0–1.0
       RawValue string  `json:"raw_value"` // human-readable, e.g. "12/47"
   }
   func QualityScan(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*QualityScanResult, error)
   ```
4. Add handler `executeQualityScanTool` to `handlers_analysis.go`
5. Add tool definition to `tools.go` (+25 lines), add case to `executor.go` (+15 lines)
6. Run all tests — expect PASS
7. Run `make commit`

**Key files**:
- `internal/analyzer/quality_analysis.go` — CREATE
- `internal/analyzer/quality_analysis_test.go` — CREATE
- `cmd/mcp-server/handlers_analysis.go` — UPDATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)

**Acceptance criteria**:
- All dimension scores in [0.0, 1.0]
- `make commit` passes

**Line budget**: ≤200 lines

---

## Phase 43: `get_work_patterns` + `get_timeline` MCP Tools

**Goal**: Implement two statistical aggregation tools. `get_timeline` returns structured JSON only — ASCII rendering is delegated to Claude.

**Estimated code**: ~420–450 lines

### Stage Dependencies

```
43.1 (get_work_patterns data layer)
  └──► 43.2 (get_work_patterns MCP registration)
43.3 (get_timeline data layer + MCP registration)   [independent of 43.1]
─── merge into main before Phase 45 ───
```

---

### Stage 43.1 — `get_work_patterns`: Data Layer

**TDD sequence**:

1. Write `internal/analyzer/work_patterns_test.go`:
   - Test: tool frequency histogram sorted descending by count
   - Test: hourly activity array has exactly 24 elements
   - Test: context-switch count increases when tool calls switch between different files within 5 minutes
   - Test: empty session returns zero values, no error
2. Run tests — expect **BUILD FAIL**
3. Implement `internal/analyzer/work_patterns.go`:
   ```go
   type WorkPatternsResult struct {
       ToolFrequency  []ToolCount `json:"tool_frequency"`
       HourlyActivity [24]int     `json:"hourly_activity"`
       ContextSwitches int        `json:"context_switches"`
       PeakHour       int         `json:"peak_hour"`
   }
   func GetWorkPatterns(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*WorkPatternsResult, error)
   ```
4. Run tests — expect PASS
5. Run `make commit`

**Key files**:
- `internal/analyzer/work_patterns.go` — CREATE
- `internal/analyzer/work_patterns_test.go` — CREATE

**Line budget**: ≤200 lines

---

### Stage 43.2 — `get_work_patterns`: MCP Registration

**TDD sequence**: Write integration test → add handler to `handlers_analysis.go` → register in `tools.go` (+25 lines) and `executor.go` (+15 lines) → run `make commit`.

**Key files**:
- `cmd/mcp-server/handlers_analysis.go` — UPDATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)
- `cmd/mcp-server/analysis_patterns_test.go` — CREATE (~60 lines)

**Acceptance criteria**: Tool registered; returns valid JSON. `make commit` passes.

**Line budget**: ≤120 lines

---

### Stage 43.3 — `get_timeline`: Full Implementation

**Goal**: Return chronological event JSON. No ASCII rendering in Go — Claude renders visualization.

**TDD sequence**:

1. Write `internal/analyzer/timeline_test.go`:
   - Test: events sorted by timestamp ascending
   - Test: consecutive same-type events merged, `duration_ms` set
   - Test: each event has `type`, `summary`, `timestamp`, `duration_ms` fields
   - Test: `limit` parameter caps number of events returned
   - Test: empty session returns result with empty events list, no error
2. Run tests — expect **BUILD FAIL**
3. Implement `internal/analyzer/timeline.go`:
   ```go
   type TimelineResult struct {
       Events    []TimelineEvent `json:"events"`
       TotalSpan string          `json:"total_span"` // e.g. "2h 34m"
   }
   type TimelineEvent struct {
       Timestamp  time.Time `json:"timestamp"`
       Type       string    `json:"type"`       // "tool_call", "user_message", "error", etc.
       Summary    string    `json:"summary"`    // short description only; no ASCII
       DurationMs int64     `json:"duration_ms"`
   }
   func GetTimeline(entries []parser.SessionEntry, limit int) (*TimelineResult, error)
   ```
   Reuse `parseTimestamp()` from `workflow.go` where available.
4. Add handler to `handlers_analysis.go`; register in `tools.go` (+25 lines) and `executor.go` (+15 lines)
5. Run all tests — expect PASS
6. Run `make commit`

**Key files**:
- `internal/analyzer/timeline.go` — CREATE
- `internal/analyzer/timeline_test.go` — CREATE
- `cmd/mcp-server/handlers_analysis.go` — UPDATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)

**Acceptance criteria**:
- Events time-ordered; `duration_ms` populated for merged groups
- No ASCII string rendering anywhere in Go code
- `make commit` passes

**Line budget**: ≤200 lines

---

## Phase 44: `analyze_bugs` + `get_tech_debt` MCP Tools

**Goal**: Implement the remaining two analysis tools.

**Estimated code**: ~430–470 lines

### Stage Dependencies

```
44.1 (analyze_bugs data layer)
  └──► 44.2 (analyze_bugs MCP registration)
44.3 (get_tech_debt data layer + MCP registration)   [independent]
─── merge into main before Phase 45 ───
```

---

### Stage 44.1 — `analyze_bugs`: Data Layer

**TDD sequence**:

1. Write `internal/analyzer/bugs_analysis_test.go`:
   - Test: error followed by successful same-tool call within 3 turns = fix pair
   - Test: identical error signatures across multiple occurrences = recurrence > 1
   - Test: patterns sorted by recurrence descending
   - Test: empty or error-free session returns empty patterns list, no error
2. Run tests — expect **BUILD FAIL**
3. Implement `internal/analyzer/bugs_analysis.go`:
   ```go
   type BugAnalysisResult struct {
       Patterns   []BugPattern `json:"patterns"`
       TotalPairs int          `json:"total_pairs"`
   }
   type BugPattern struct {
       ErrorSignature string   `json:"error_signature"` // from CalculateErrorSignature()
       FixCount       int      `json:"fix_count"`
       Recurrences    int      `json:"recurrences"`
       Examples       []string `json:"examples"`
   }
   func AnalyzeBugs(entries []parser.SessionEntry, toolCalls []parser.ToolCall, limit int) (*BugAnalysisResult, error)
   ```
   Reuse `CalculateErrorSignature()` from `errors.go`.
4. Run tests — expect PASS
5. Run `make commit`

**Key files**:
- `internal/analyzer/bugs_analysis.go` — CREATE
- `internal/analyzer/bugs_analysis_test.go` — CREATE

**Line budget**: ≤200 lines

---

### Stage 44.2 — `analyze_bugs`: MCP Registration

**TDD sequence**: Write integration test → add handler → register in `tools.go` (+25 lines) and `executor.go` (+15 lines) → `make commit`.

**Key files**:
- `cmd/mcp-server/handlers_analysis.go` — UPDATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)
- `cmd/mcp-server/analysis_bugs_test.go` — CREATE (~60 lines)

**Line budget**: ≤120 lines

---

### Stage 44.3 — `get_tech_debt`: Full Implementation

**Goal**: Marker-based debt detection using file snapshot entries. Semantic classification delegated to Claude.

**File snapshot note**: File snapshot data is stored as raw JSON entries with `type == "file-history-snapshot"` in the JSONL. These are NOT typed as `parser.SessionEntry` fields. The implementation must filter for `entry.Type == "file-history-snapshot"` and unmarshal the raw content field separately to access file path and content.

**TDD sequence**:

1. Write `internal/analyzer/tech_debt_test.go`:
   - Test: detects `TODO`, `FIXME`, `HACK`, `XXX` markers in file snapshot content via regex
   - Test: counts markers per file; returns hotspot files sorted by count descending
   - Test: detects same tool error recurring without fix across multiple tool calls (open issues proxy)
   - Test: empty session returns zero counts, no error
2. Run tests — expect **BUILD FAIL**
3. Implement `internal/analyzer/tech_debt.go`:
   ```go
   type TechDebtResult struct {
       Markers      []MarkerCount `json:"markers"`
       HotspotFiles []FileDebt    `json:"hotspot_files"`
       OpenIssues   int           `json:"open_issues"`
   }
   type MarkerCount struct {
       Label string `json:"label"` // "TODO", "FIXME", etc.
       Count int    `json:"count"`
   }
   type FileDebt struct {
       File        string `json:"file"`
       MarkerCount int    `json:"marker_count"`
   }
   func GetTechDebt(entries []parser.SessionEntry, toolCalls []parser.ToolCall) (*TechDebtResult, error)
   ```
   Parse file snapshot raw content via `json.Unmarshal` on the entry's raw bytes for `type == "file-history-snapshot"` entries.
4. Add handler to `handlers_analysis.go`; register in `tools.go` (+25 lines) and `executor.go` (+15 lines)
5. Run all tests — expect PASS
6. Run `make commit`

**Key files**:
- `internal/analyzer/tech_debt.go` — CREATE
- `internal/analyzer/tech_debt_test.go` — CREATE
- `cmd/mcp-server/handlers_analysis.go` — UPDATE
- `cmd/mcp-server/tools.go` — UPDATE (+25 lines)
- `cmd/mcp-server/executor.go` — UPDATE (+15 lines)

**Acceptance criteria**:
- Marker detection is regex-based, language-agnostic
- File snapshot raw JSON parsed correctly
- No semantic debt classification in Go
- `make commit` passes

**Line budget**: ≤200 lines

---

## Phase 45: Remove Capability Loading System

**Goal**: Delete the dynamic capability loading system (~3,349 lines of Go), remove `list_capabilities` and `get_capability` MCP tools, and remove the `/meta` command.

**Prerequisite gate**: Before Stage 45.1 begins, verify:
1. `yaleh/baime` is publicly reachable at `github.com/yaleh/baime`
2. `workflow-coach.md` agent is present in the repo
3. Plugin installable via `/plugin marketplace add yaleh/baime` in Claude Code

If any check fails, do not proceed with Phase 45.

**Estimated net change**: −~3,349 lines of Go removed; −~120 lines of Markdown (`meta.md`); +~30 lines of relocation work

### Stage Dependencies

```
45.1 (relocate CleanupSessionCache + remove capability tools from executor)
  └──► 45.2 (delete capabilities.go + all capabilities test files)
  └──► 45.3 (delete capability Markdown content + release artifact references)
```

45.2 and 45.3 are independent of each other after 45.1.

---

### Stage 45.1 — Relocate `CleanupSessionCache` and Remove Capability Tool Cases (Atomic)

**Goal**: Two changes in one commit: (1) move `CleanupSessionCache()` out of `capabilities.go` so it can be safely deleted; (2) remove the `list_capabilities` and `get_capability` case blocks from `executor.go`.

These must be a single atomic commit because the `make commit` hook will catch undefined references if they are split.

**TDD sequence**:

1. Confirm `make push` passes as baseline
2. Move `CleanupSessionCache()` and `getSessionCacheDir()` from `capabilities.go` to `cmd/mcp-server/temp_file_manager.go` (they logically belong there)
3. Verify `main.go` line 53's call to `CleanupSessionCache()` resolves to the new location — update import if needed
4. Remove `case "list_capabilities":` block (~12 lines) from `executor.go`
5. Remove `case "get_capability":` block (~12 lines) from `executor.go`
6. Remove `list_capabilities` and `get_capability` tool definitions from `tools.go` (~60 lines)
7. Run `make build` — expect PASS (capabilities.go still exists with its other functions; executor no longer calls them)
8. Run `make commit` — expect PASS

**Key files**:
- `cmd/mcp-server/temp_file_manager.go` — UPDATE (add `CleanupSessionCache`, `getSessionCacheDir`)
- `cmd/mcp-server/capabilities.go` — UPDATE (remove `CleanupSessionCache`, `getSessionCacheDir`)
- `cmd/mcp-server/executor.go` — UPDATE (remove 2 case blocks, ~24 lines)
- `cmd/mcp-server/tools.go` — UPDATE (remove 2 tool definitions, ~60 lines)

**Acceptance criteria**:
- `make build` passes: no undefined references
- `list_capabilities` and `get_capability` absent from MCP `tools/list`
- `CleanupSessionCache()` callable from `main.go` (now from `temp_file_manager.go`)
- `make commit` passes

**Line budget**: ≤150 lines (relocations + removals)

---

### Stage 45.2 — Delete `capabilities.go` and All Capability Test Files

**Goal**: Delete ~3,349 lines of capability loading code and tests.

**TDD sequence**:

1. Delete files:
   - `cmd/mcp-server/capabilities.go`
   - `cmd/mcp-server/capabilities_test.go`
   - `cmd/mcp-server/capabilities_http_test.go`
   - `cmd/mcp-server/capabilities_integration_test.go`
   - `cmd/mcp-server/capabilities_cache_test.go`
2. Remove any remaining import of capability-related packages from `main.go` or `server.go` if they exist (should be none after Stage 45.1, but verify)
3. Run `make build` — expect PASS
4. Run `make test` — expect PASS (no test references to deleted types)
5. Run `make commit` — expect PASS

**Key files**: all 5 files above — DELETE

**Acceptance criteria**:
- `go build ./...` succeeds
- `go test ./...` passes
- `make commit` passes

**Line budget**: Mechanical deletions; ≤20 lines of remaining file updates

---

### Stage 45.3 — Delete Capability Content and Release Artifact References

**Goal**: Remove capability Markdown files, `/meta` command, `capabilities-latest.tar.gz` from release pipeline, and `META_CC_CAPABILITY_SOURCES` env var from CI/tooling references.

**TDD sequence**:

1. Delete `capabilities/commands/*.md` (21 files) and the `capabilities/` directory
2. Delete `.claude/commands/meta.md`
3. Remove `capabilities-latest.tar.gz` build step from `.github/workflows/release.yml`
4. Remove capability tarball target from `Makefile`
5. Remove `META_CC_CAPABILITY_SOURCES` from CI scripts and any tooling scripts that reference it (do NOT update CLAUDE.md or README here — that is deferred to Stage 46.4)
6. Run `make build` — expect PASS
7. Run `make commit` — expect PASS

**Key files**:
- `capabilities/` — DELETE (entire directory)
- `.claude/commands/meta.md` — DELETE
- `.github/workflows/release.yml` — UPDATE (remove capabilities tarball step)
- `Makefile` — UPDATE (remove capabilities tarball target)

**Acceptance criteria**:
- `capabilities/` directory absent
- `.claude/commands/meta.md` absent
- Release workflow produces no `capabilities-latest.tar.gz`
- `make commit` passes

**Line budget**: ≤80 lines of file modifications (deletions are mechanical)

---

## Phase 46: Plugin Pruning → meta-cc 3.0.0

**Goal**: Remove skills and agents; update manifests and all tooling; update all documentation including CLAUDE.md; bump to 3.0.0 and release.

**Ordering rationale**: Stage 46.1 updates manifests and scripts FIRST (so pre-commit hooks reflect new expected counts), THEN Stage 46.2 deletes the actual files. This avoids committing a state where hooks assert 18 skills but files are already gone.

**Estimated code**: ~400–450 lines of configuration and documentation updates

### Stage Dependencies

```
46.1 (update manifests + sync scripts + CI scripts — hooks updated first)
  └──► 46.2 (delete skills + agents from .claude/ and dist/)
  └──► 46.3 (update Makefile + release tooling)
All three ──► 46.4 (CLAUDE.md + README + docs + 3.0.0 bump + release)
```

---

### Stage 46.1 — Update Manifests, Sync Scripts, and CI Hooks

**Goal**: Update all manifest files and scripts so pre-commit hooks reflect the new expected counts (0 skills, 0 agents, 3 commands) BEFORE the files are deleted in Stage 46.2.

**TDD sequence**:

1. Update `.claude/.claude-plugin/plugin.json`:
   - Remove `skills` array entirely
   - Remove `agents` array entirely
   - Update `commands` to 3 entries (`/prompt-find`, `/prompt-list`, `/prompt-show`)
2. Update `.claude-plugin/marketplace.json`:
   - Remove agent declarations
   - Remove `meta.md` command entry
3. Update `scripts/ci/test-plugin-json.sh`: expected skill count = 0, agent count = 0, command count = 3
4. Update `scripts/sync-plugin-files.sh`: remove skills, agents, and capabilities copy logic
5. Run `bash scripts/ci/test-plugin-json.sh` — expect PASS (counts match new manifests; actual files still present but not referenced)
6. Run `make commit` — expect PASS

**Key files**:
- `.claude/.claude-plugin/plugin.json` — UPDATE
- `.claude-plugin/marketplace.json` — UPDATE
- `scripts/ci/test-plugin-json.sh` — UPDATE (expected counts)
- `scripts/sync-plugin-files.sh` — UPDATE

**Acceptance criteria**:
- `plugin.json` contains only `commands` (3) and `mcpServers`
- `test-plugin-json.sh` passes with 0 skills, 0 agents, 3 commands
- `make commit` passes (skills/agents files still present at this point — that is expected)

**Line budget**: ≤150 lines

---

### Stage 46.2 — Delete Skills and Agents from Plugin Content

**Goal**: Remove skill and agent content files now that manifests are already updated.

**TDD sequence**:

1. Delete `.claude/skills/` directory (18 subdirectories)
2. Delete `.claude/agents/*.md` (all 7 files: 5 published + 2 dev-only)
3. Delete `dist/skills/` directory
4. Delete `dist/agents/*.md`
5. Run `bash scripts/ci/test-plugin-json.sh` — expect PASS (manifests already correct from Stage 46.1)
6. Run `make build` — expect PASS
7. Run `make commit` — expect PASS

**Key files**: all skill/agent directories — DELETE

**Acceptance criteria**:
- No skill or agent files remain under `.claude/` or `dist/`
- `test-plugin-json.sh` passes
- `make commit` passes

**Line budget**: Mechanical deletions; ≤10 lines of updates

---

### Stage 46.3 — Update Makefile and Release Tooling

**Goal**: Remove skill/agent/capabilities references from build and release infrastructure.

**TDD sequence**:

1. Update `Makefile` `bundle-release` target: remove skills/agents/capabilities copy steps
2. Update `scripts/ci/smoke-tests.sh`: remove all skill count, agent count, and capability assertions
3. Update `scripts/release/release.sh` if it references skills or capabilities
4. Run `make push` — expect PASS

**Key files**:
- `Makefile` — UPDATE (bundle-release target)
- `scripts/ci/smoke-tests.sh` — UPDATE
- `scripts/release/release.sh` — UPDATE if needed

**Acceptance criteria**:
- `make push` passes all quality gates
- No script references to skills, agents, or `capabilities-latest.tar.gz`

**Line budget**: ≤150 lines

---

### Stage 46.4 — CLAUDE.md, Documentation Update, 3.0.0 Bump, and Release

**Goal**: Comprehensively update all user-facing and developer-facing documentation to reflect the post-3.0.0 state. Bump version and release.

**TDD sequence**:

1. Audit `CLAUDE.md` — update or remove all references to:
   - Skills (FAQ entries, plugin development section)
   - Agents (FAQ entries)
   - `/meta` command (all examples and workflow descriptions)
   - `capabilities/commands/` directory
   - `META_CC_CAPABILITY_SOURCES` environment variable
   - Unified Meta Command section
   - Links to `docs/guides/capabilities.md` and `docs/reference/unified-meta-command.md`
   - Replace plugin development workflow section with streamlined: binary + 3 commands only
2. Update `README.md`:
   - Remove skills section and all skill counts
   - Remove agents section
   - Update Quick Install: 3 commands (not 4); remove skills bullet
   - Add companion note linking to `yaleh/baime`
   - Update Key Features list
3. Update `docs/tutorials/installation.md`: remove any remaining skills/agents/`/meta` references
4. Update `docs/guides/troubleshooting.md` if needed
5. Bump version to `3.0.0` in `plugin.json` and `marketplace.json`
6. Update `CHANGELOG.md` with breaking change notice (skills removed → `yaleh/baime`; agents removed → `yaleh/baime`; `/meta` removed; 6 new analysis tools added)
7. Run `make push` — expect PASS
8. Create git tag `v3.0.0` and publish GitHub release

**Key files**:
- `CLAUDE.md` — UPDATE (comprehensive audit)
- `README.md` — UPDATE
- `docs/tutorials/installation.md` — UPDATE
- `.claude/.claude-plugin/plugin.json` — UPDATE (version 3.0.0)
- `.claude-plugin/marketplace.json` — UPDATE (version 3.0.0)
- `CHANGELOG.md` — UPDATE

**Acceptance criteria**:
- `CLAUDE.md` contains no references to skills, agents, `/meta`, or `capabilities/`
- README cross-references `yaleh/baime`
- Version is `3.0.0` in both manifest files
- `make push` passes
- GitHub release `v3.0.0` published

**Line budget**: ≤200 lines

---

## Summary

| Phase | Net lines | Key constraint |
|-------|-----------|----------------|
| 41 | +600 (new repo) | Phase 45 blocked until baime is verified published |
| 42 | +460 | Establish `loadEntriesAndToolCalls` pattern here |
| 43 | +450 | `get_timeline` returns JSON only, no ASCII |
| 44 | +450 | `get_tech_debt` must handle raw file-snapshot JSON |
| 45 | −10,019 removed, +150 relocation | `CleanupSessionCache` must be relocated in 45.1 before deletion in 45.2 |
| 46 | −content, +450 config/docs | Manifests updated in 46.1 before files deleted in 46.2 |

**meta-cc net change**: remove ~10,019 lines (capability Go + Markdown), add ~1,360 lines (6 analysis tools + tests + integration), net −~8,659 lines.

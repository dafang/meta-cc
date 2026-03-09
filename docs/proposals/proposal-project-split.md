# Proposal: Split meta-cc into Two Focused Projects

**Status**: Implemented
**Date**: 2026-03-09
**Implemented**: 2026-03-09 (Phases 41–46)
**Author**: Yale Huang

---

## 1. Problem Statement

meta-cc currently bundles three distinct concerns into a single plugin:

1. **Session history retrieval** — A Go MCP server that parses Claude Code JSONL session files and exposes 17 tools (10 convenience query + 4 metadata + 1 utility + 2 capability-loading tools, for a pre-split total of 17).

2. **Analytical capability workflows** — 21 `meta-*` Markdown workflow templates, dynamically loaded from GitHub/tar.gz at runtime, routed by a `/meta` natural-language dispatcher. These call MCP query tools and rely on Claude's reasoning to produce results.

3. **Software development methodologies** — 18 validated BAIME-derived skills and 7 general-purpose development agents (stage-executor, project-planner, iteration-executor, etc.).

These three concerns have different users, different update cadences, and different dependencies. Bundling them creates noise, bloat, and architectural complexity.

### Symptoms

- Users installing meta-cc for MCP session analysis receive ~1.5MB of methodology skills they may not need.
- Users who want development methodology tools must also install and run a Go MCP binary.
- The dynamic capability loading system (~3,296 lines of Go + tests) implements "list files + read file" with network download, cache, SHA validation, and three source types — significant over-engineering for static Markdown content.
- The default capability source is a GitHub release `.tar.gz`; in restricted network environments this silently fails.
- Skills and agents have no runtime dependency on meta-cc MCP, yet appear as if they are meta-cc-specific.
- The project README conflates three different elevator pitches.

---

## 2. Proposed Split

### 2.1 Project A: meta-cc (refined scope)

**Tagline**: _Query your Claude Code session history via MCP._

**Core principle**: MCP is the only interface. Every tool either retrieves session data or aggregates it into structured results. **Go code handles data extraction and aggregation; Claude handles semantic interpretation and visualization.** No dynamic loading. No LLM workflow templates bundled.

**Retains**:

| Category | Contents |
|----------|----------|
| MCP Server | Go binary (`meta-cc-mcp`), 15 session query/metadata tools |
| New analysis tools | 6 Go-implemented aggregation tools returning structured JSON (see Section 3) |
| Commands | `/prompt-find`, `/prompt-list`, `/prompt-show` |

**Note on `/prompt-*` commands**: These three commands have **zero MCP dependency** — they operate purely on `.meta-cc/prompts/library/` via local filesystem. They are retained in meta-cc but do not require the MCP server to function.

**Removes**:
- All 21 `meta-*` capability Markdown files and the `capabilities/` directory
- `/meta` command and the dynamic capability loading system (~3,296 lines)
- `list_capabilities` and `get_capability` MCP tools
- All 18 skills and all 7 agents (moved to Project B or deleted)

**Result**: Binary-only plugin. Single responsibility: "retrieve and aggregate Claude Code session data."

---

### 2.2 Project B: `yaleh/baime`

**Tagline**: _Validated software development methodologies for Claude Code._

**Core principle**: Pure Markdown. No binary. No MCP dependency. Works standalone. Can optionally call meta-cc MCP tools when both plugins are installed together.

**Contains**:

| Category | Contents |
|----------|----------|
| Skills | 18 validated methodology skills + `next-step-generation` (migrated from `meta-next`) = **19 skills** |
| Agents | 5 published development workflow agents + `workflow-coach` (migrated from `meta-coach`) = **6 agents** |

#### Skills inventory

| Skill | Domain |
|-------|--------|
| `methodology-bootstrapping` | BAIME framework (absorbs `meta-prompt` methodology) |
| `testing-strategy` | TDD, coverage-driven gap closure |
| `ci-cd-optimization` | Quality gates, release automation |
| `error-recovery` | 13-category taxonomy, diagnostic workflows |
| `dependency-health` | Security-first, batch remediation |
| `knowledge-transfer` | Progressive learning paths, onboarding |
| `technical-debt-management` | SQALE methodology, prioritization |
| `code-refactoring` | Test-driven refactoring |
| `cross-cutting-concerns` | Error handling, logging, configuration |
| `observability-instrumentation` | Logs, metrics, traces |
| `api-design` | 6 validated patterns |
| `documentation-management` | Templates, patterns, automation |
| `agent-prompt-evolution` | Agent specialization tracking |
| `baseline-quality-assessment` | Rapid convergence enablement |
| `rapid-convergence` | 3-4 iteration methodology development |
| `retrospective-validation` | Historical data validation |
| `subagent-prompt-construction` | Compact Claude Code subagent prompts |
| `build-quality-gates` | Quality enforcement for build/CI |

#### Agents inventory

| Agent | Role |
|-------|------|
| `stage-executor` | Executes project plan stages with validation |
| `project-planner` | Generates TDD-based development plans |
| `iteration-executor` | Executes BAIME experiment iterations |
| `iteration-prompt-designer` | Designs ITERATION-PROMPTS.md files |
| `knowledge-extractor` | Extracts BAIME experiments into skills |
| `workflow-coach` | Standalone coaching agent; optionally enriches with meta-cc data when available |

`feature-developer` and `phase-planner-executor` are dev-only agents used in meta-cc's own development workflow. They were **removed** (not migrated).

---

## 3. Capability Refactoring

The 21 `meta-*` capability Markdown files are disposed of in three ways: converted to Go MCP tools, migrated to `yaleh/baime`, or deleted.

### 3.1 Design contract for new Go MCP tools

**Critical principle**: Go tools perform **data extraction and statistical aggregation only**. They return structured JSON. Claude provides semantic interpretation, contextual reasoning, and presentation (including any visualizations). This preserves Claude's reasoning strengths while making data gathering deterministic and testable.

```
Session JSONL
    ↓
Go tool (deterministic aggregation → structured JSON)
    ↓
Claude (semantic interpretation → user-facing analysis)
```

This means tools like `analyze_errors` do NOT classify errors semantically — they count, group, and return raw data with examples. Claude categorizes intent and draws conclusions.

### 3.2 → New MCP analysis tools in meta-cc (Go implementation)

| New MCP Tool | Replaces | Go implementation scope | Returns (actual JSON fields) |
|---|---|---|---|
| `analyze_errors` | `meta-errors` | Group tool errors by tool name and error signature; count patterns; surface examples | `{time_range, total_errors, by_tool: [{tool_name, count, examples}], by_type: [{signature, count, examples}]}` |
| `quality_scan` | `meta-quality-scan` | Multi-dimension session metrics (error rate, completion rate, retry rate, tool diversity) | `{dimensions: [{name, score, raw_value}]}` |
| `get_work_patterns` | `meta-habits` + `meta-focus-analyzer` | Tool usage frequency histogram; hourly activity array (24 elements); context-switch detection | `{tool_frequency: [{tool_name, count}], hourly_activity: [24]int, context_switches, peak_hour}` |
| `get_timeline` | `meta-timeline` | Chronological event sequence with timestamps and durations | `{events: [{timestamp, type, summary, duration_ms}], total_span}` — Claude renders visualization |
| `analyze_bugs` | `meta-bugs` | Error→fix turn-pair extraction; recurrence detection | `{patterns: [{error_signature, fix_count, recurrences, examples}], total_pairs}` |
| `get_tech_debt` | `meta-tech-debt` | TODO/FIXME/HACK/XXX marker counts per file; unresolved errors as open-issue proxy | `{markers: [{label, count}], hotspot_files: [{file, marker_count}], open_issues}` |

**Implementation notes**:

- `get_timeline` returns JSON only. ASCII art rendering is delegated to Claude, eliminating ~800 lines of Go string-manipulation code.
- `get_tech_debt` uses regex marker detection over **tool call outputs** (Read/Edit/Write/Bash result text), not raw file-history-snapshot entries. File snapshot parsing was skipped because `parser.ParseEntries()` filters those entries out; scanning tool outputs yields equivalent marker coverage. Semantic debt classification is delegated to Claude.
- `analyze_errors` uses tool call metadata (status, error messages) from JSONL — not semantic analysis of user message text. This is a scope reduction from `meta-errors` which attempted full semantic classification; the reduction is intentional.
- Data-layer functions live in `internal/analyzer/` (one file per tool: `errors_analysis.go`, `quality_analysis.go`, `work_patterns.go`, `timeline.go`, `bugs_analysis.go`, `tech_debt.go`). MCP handlers and the shared `loadEntriesAndToolCalls` helper are consolidated in `cmd/mcp-server/handlers_analysis.go`.

**Estimated implementation**: ~2,000–2,500 lines of new Go (implementation + tests across 6 files). Existing `internal/analyzer/` and `internal/stats/` packages provide data models and can be extended.

### 3.3 → Migrated to `yaleh/baime` (2 capabilities)

These capabilities are LLM reasoning workflows that require no session data to function. Their value is methodological, not data-retrieval.

| Capability | Destination | Form | MCP dependency |
|---|---|---|---|
| `meta-coach` | `yaleh/baime` | Agent: `workflow-coach` — general workflow optimization coaching; may optionally call meta-cc MCP tools when available | None (optional enrichment) |
| `meta-next` | `yaleh/baime` | Skill: `next-step-generation` — generates ready-to-use continuation prompts; explicitly no MCP execution | None |

`meta-prompt` is absorbed into the existing `methodology-bootstrapping` skill (prompt refinement is a core BAIME practice already covered there).

### 3.4 → Deleted (11 capabilities)

| Capability | Reason |
|---|---|
| `meta-guide` | Duplicates `meta-coach`; superseded by the BAIME `workflow-coach` agent |
| `meta-viz` | Helper with no standalone value; ASCII rendering belongs in Claude responses |
| `meta-architecture` | Primarily LLM reasoning over code structure, not session data aggregation |
| `meta-project-bootstrap` | One-time project initialization; unrelated to session analysis |
| `meta-doc-sync` | Only meaningful for projects following meta-cc's own `docs/` conventions |
| `meta-doc-gaps` | Same |
| `meta-doc-structure` | Same |
| `meta-doc-health` | Same |
| `meta-doc-links` | Same |
| `meta-doc-evolution` | Same |
| `meta-doc-usage` | Same |

The seven `meta-doc-*` capabilities are only useful for projects structured like meta-cc itself. They are too niche for a general-purpose plugin and have no counterpart in typical Claude Code projects.

### 3.5 Consequence: removal of dynamic loading system

| Removed component | Lines |
|---|---|
| `capabilities.go` (GitHub/package/local loading, caching, extraction) | ~1,171 |
| `capabilities_test.go` | ~1,398 |
| `capabilities_http_test.go` | ~309 |
| `capabilities_integration_test.go` | ~418 |
| `capabilities_cache_test.go` | ~53 |
| `list_capabilities` + `get_capability` MCP tools | ~50 |
| `/meta` slash command | ~120 |
| `capabilities/commands/*.md` (21 files) | ~6,500 |
| **Total removed** | **~10,019 lines** |

**Migration note** (completed): `CleanupSessionCache()` was defined in `capabilities.go` and called from `main.go`. It was relocated to `temp_file_manager.go` in Phase 45.1 (atomically with the removal of capability tool cases from `executor.go`) before `capabilities.go` was deleted in Phase 45.2.

The `capabilities-latest.tar.gz` release artifact is also eliminated.

**Net code change**: removed ~10,019 lines (capability files + loading system), added ~1,360 lines (6 analysis tools + tests + handlers). Net reduction of ~8,659 lines.

---

## 4. MCP Tool Count

### Before refactoring (17 total)

| Group | Tools | Count |
|---|---|---|
| Convenience query | `query_user_messages`, `query_tools`, `query_tool_errors`, `query_token_usage`, `query_conversation_flow`, `query_system_errors`, `query_file_snapshots`, `query_timestamps`, `query_summaries`, `query_tool_blocks` | 10 |
| Metadata / config | `get_session_directory`, `inspect_session_files`, `get_session_metadata`, `execute_stage2_query` | 4 |
| Utility | `cleanup_temp_files` | 1 |
| Capability loading (removed) | `list_capabilities`, `get_capability` | 2 |
| **Total** | | **17** |

### After refactoring (21 total — as implemented)

Remove 2 capability-loading tools. Add 6 analysis tools. Net: **17 − 2 + 6 = 21 tools**.

| Group | Tools | Count |
|---|---|---|
| Convenience query | (same 10 as above) | 10 |
| Metadata / config | (same 4 as above) | 4 |
| Utility | `cleanup_temp_files` | 1 |
| Analysis (new) | `analyze_errors`, `quality_scan`, `get_work_patterns`, `get_timeline`, `analyze_bugs`, `get_tech_debt` | 6 |
| **Total** | | **21** |

**On MCP tool count limits**: The MCP specification (2025-11-25) defines no hard limit on tool count. Client-side limits vary: Cursor enforces 40, VS Code 128. Claude Code activates automatic Tool Search (deferred loading) when tool definitions exceed 10% of the context window, so 21 tools is well within safe operating range.

---

## 5. Content Boundary Summary

| Content | Destination | Reason |
|---------|-------------|--------|
| MCP server Go binary | meta-cc | Core product |
| 15 existing session query/metadata tools | meta-cc | Core retrieval layer |
| 6 new analysis tools (Go) | meta-cc | Deterministic data aggregation |
| `/prompt-*` commands (3) | meta-cc | Local filesystem ops, zero MCP dependency |
| `/meta` command | **Removed** | No longer needed without capability routing |
| `list_capabilities`, `get_capability` tools | **Removed** | No longer needed |
| Dynamic loading system (~3,296 lines) | **Removed** | Replaced by static Go tools |
| 12 deleted capabilities | **Deleted** | See Section 3.4 |
| 2 reasoning capabilities | **→ baime** (as `workflow-coach` agent + `next-step-generation` skill) | See Section 3.3 |
| 18 skills | **→ baime** | No meta-cc dependency |
| 5 published agents | **→ baime** | No meta-cc dependency |
| 2 dev-only agents | **Removed** | Not migrated |

---

## 6. Project B: Name and Location

**Name**: `baime` (`yaleh/baime` on GitHub)

BAIME (Bootstrapped AI Methodology Engineering) is the unifying framework from which all skills and agents derive. Using it as the project name makes the brand explicit and the purpose clear to practitioners familiar with the methodology.

---

## 7. Relationship Between Projects

- The two projects are **independent** — neither requires the other at runtime.
- baime agents (specifically `workflow-coach`) may optionally call meta-cc MCP tools when available, but degrade gracefully without them.
- meta-cc README will mention `baime` as a companion for development methodology tooling.
- `baime` README will mention meta-cc as a companion for session history analysis.
- No shared code, no shared CI, no cross-repo version dependencies.

---

## 8. Migration Plan

### Phase 1: Create `yaleh/baime` repository

1. Create new repo `yaleh/baime` with `plugin.json`, `marketplace.json`.
2. Copy 18 skills and 5 published agents from meta-cc (excluding `feature-developer` and `phase-planner-executor`).
3. Migrate `meta-coach` as a new `workflow-coach` agent.
4. Migrate `meta-next` as a new `next-step-generation` skill.
5. Absorb `meta-prompt` into `methodology-bootstrapping` skill.
6. Set up minimal CI (JSON/Markdown lint, plugin.json validation).
7. Publish to Claude Code plugin marketplace.

### Phase 2: Implement new Go analysis tools in meta-cc

1. Data-layer functions implemented in `internal/analyzer/`: `errors_analysis.go`, `quality_analysis.go`, `work_patterns.go`, `timeline.go`, `bugs_analysis.go`, `tech_debt.go`.
2. Shared session loader `loadEntriesAndToolCalls` and all MCP handlers consolidated in `cmd/mcp-server/handlers_analysis.go`.
3. Tools registered in `tools.go` and `executor.go`; test files per tool in `cmd/mcp-server/analysis_*_test.go`.
4. Remove `capabilities.go` and all associated test files (~3,349 lines).
5. Remove `list_capabilities` and `get_capability` from `tools.go` and `executor.go`.
6. Remove `capabilities-latest.tar.gz` from release artifacts and CI.

### Phase 3: Prune meta-cc plugin content → 3.0.0

1. Delete `capabilities/` directory (21 Markdown files).
2. Remove `/meta` slash command from `.claude/commands/`.
3. Remove all 18 skills from `.claude/skills/` and `dist/skills/`.
4. Remove all 7 agents from `.claude/agents/` and `dist/agents/`.
5. Update `plugin.json`: remove `skills`, `agents`; update `commands` to 3 (`/prompt-*` only).
6. Update `marketplace.json`: remove agent declarations.
7. Update `sync-plugin-files.sh`: remove skills/agents/capabilities copy logic.
8. Update `README.md` and docs: reflect new scope, link to `yaleh/baime`.
9. Bump to **3.0.0** (breaking: capabilities, skills, and agents no longer bundled).

### Phase 4: Update CI and release tooling in meta-cc

1. Update `test-plugin-json.sh`: expected skill count = 0, agent count = 0, command count = 3.
2. Update `scripts/ci/smoke-tests.sh`: remove skill/agent/capability assertions.
3. Update `Makefile` bundle-release target: remove skills/agents/capabilities copy steps and `capabilities-latest.tar.gz`.
4. Update `scripts/release/bump-plugin-version.sh` and hooks.

Phases 2 and 3 can proceed in parallel.

---

## 9. Impact Assessment

### meta-cc after refactoring (3.0.0)

| Metric | Before | After |
|--------|--------|-------|
| Total MCP tools | 17 | 21 |
| Lines removed | — | ~10,019 (capability files + loading system) |
| Lines added | — | ~1,360 (6 analysis tools + tests + handlers) |
| Net lines | — | −~8,659 |
| Plugin size | ~1.8MB (binary + skills + capabilities) | ~0.2MB (binary only) |
| `plugin.json` skills | 18 | 0 |
| `plugin.json` agents | 5 (published) | 0 |
| `plugin.json` commands | 4 | 3 (`/prompt-*` only) |
| Runtime network dependency | Yes (capability download) | None |
| Elevator pitch | "Session analysis + methodology skills" | "Query Claude Code session history via MCP" |

### `yaleh/baime` (new)

| Metric | Value |
|--------|-------|
| Plugin size | ~1.5MB (19 skills + 6 agents) |
| Binary required | None |
| MCP server required | None (meta-cc optional for enriched coaching) |
| Works without meta-cc | ✓ |
| Applicable to any Claude Code project | ✓ |

---

## 10. Decisions

| Question | Decision |
|----------|----------|
| Project B name | `baime` (`yaleh/baime`) |
| dev-only agents (`feature-developer`, `phase-planner-executor`) | **Removed** — not migrated |
| meta-cc version after split | **3.0.0** (breaking change) |
| Capability architecture | **Path B** — Go analysis tools returning structured JSON; Claude handles semantic interpretation |
| 12 non-core capabilities | **Deleted** directly |
| 2 reasoning capabilities | **Migrated to `yaleh/baime`** |
| `meta-prompt` capability | **Absorbed** into `methodology-bootstrapping` skill in baime |
| `/meta` command | **Removed** — no routing needed without dynamic capabilities |
| ASCII visualization (`meta-timeline`) | Delegated to Claude; Go tool returns structured JSON |

---

## 11. As Implemented

Executed in six phases (Phases 41–46). Phase 41 (baime creation) ran in parallel with Phases 42–44 (Go tool implementation). Phase 45 (capability loading removal) ran after all analysis tools were verified live and `yaleh/baime` was confirmed publicly reachable. Phase 46 (plugin pruning → 3.0.0) followed Phase 45 completion.

The architectural invariant held throughout: **Go code owns data; Claude owns interpretation.**

### Key deviations from original design

| Design intent | As implemented | Reason |
|---|---|---|
| 6 new Go files in `cmd/mcp-server/` | Data layer in `internal/analyzer/`; single `handlers_analysis.go` in `cmd/mcp-server/` | Cleaner separation: analyzer package owns logic, MCP layer owns HTTP/JSON dispatch |
| `get_tech_debt`: regex over file-history-snapshot entries | Regex over tool call outputs (Read/Edit/Write/Bash `.Output` field) | `parser.ParseEntries()` filters out snapshot entries; tool output scanning yields equivalent marker coverage |
| 12 capabilities deleted | 11 capabilities deleted (correct count — the "12" claim in the original design was an off-by-one) | n/a |
| ~2,000–2,500 lines of new Go | ~1,360 lines of new Go | Consolidating handlers into one file and reusing existing `internal/analyzer/` types reduced duplication |

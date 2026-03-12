# Proposal: Architecture Cleanup — Phases 81–86

**Date**: 2026-03-12
**Status**: Reviewed
**Author**: Claude Code (automated analysis)
**Scope**: Internal package boundary hygiene, god-object decomposition, duplicate elimination

---

## Background

Phases 78–80 completed the `query` sub-package reorganisation (extracting `turnindex`, `sequences`, and `assistant` sub-packages). The architectural scan that preceded those phases also surfaced six lower-priority issues that were deferred. With Phase 80 merged, these issues are the primary source of maintainability debt in the codebase:

| ID | Severity | Issue |
|----|----------|-------|
| C-1 | Critical | Analyzer interfaces reference `parser` types instead of `types` |
| C-2 | Critical | `ExecuteSpecialTool` is an 11-case god function in a 472-line file (21 `case` clauses total across two switches) |
| H-2 | High | `buildTurnIndex`/`parseTimestamp`/`getActionType` duplicated across packages |
| H-3 | High | `internal/analyzer/workflow.go` is a 479-line god file |
| H-4 | High | `internal/mcp/pipeline` injects function types to work around an import cycle |
| M-6 | Medium | `internal/parser` is a half-finished alias shim; 44 production + 42 test files still import it |

---

## Issues in Detail

### C-1 — Analyzer Interfaces Leak `parser` Types

**File**: `internal/analyzer/interfaces.go`

All six public interfaces (`BugAnalyzer`, `ErrorAnalyzer`, `QualityScanner`, `WorkPatternsAnalyzer`, `TimelineAnalyzer`, `TechDebtAnalyzer`) declare parameters as `[]parser.SessionEntry` and `[]parser.ToolCall`. Since `internal/parser` is now nothing but an alias shim re-exporting `internal/types`, callers are forced to import `internal/parser` solely to satisfy the interface signature, even though the concrete types live in `internal/types`.

The fix is one-line per interface parameter: replace `parser.SessionEntry` → `types.SessionEntry` and `parser.ToolCall` → `types.ToolCall` throughout `interfaces.go` and its `DefaultAnalyzer` method receivers. The `import "parser"` in that file can then be removed. All implementations already work because the types are identity aliases.

**Risk**: Low. Type aliases guarantee identical underlying types; the compiler will accept the substitution without changes to callers that pass `parser.SessionEntry` values (since `parser.SessionEntry = types.SessionEntry`).

---

### C-2 — God Function `ExecuteSpecialTool`

**File**: `internal/mcp/executor/executor.go` (472 lines)

`ExecuteSpecialTool` is an 11-arm `switch` statement (plus a separate 10-arm switch for the query convenience tools, totalling 21 `case` clauses in the file). Every new MCP tool requires editing this 472-line file. The repeated pattern (call handler → `ClassifyError` → `RecordToolFailure`/`RecordToolSuccess` → marshal → return) is copy-pasted for each case. This violates the Open/Closed Principle and makes the file exceed the project's 200-line stage limit.

**Proposed fix — handler registry**:

1. Define a `SpecialToolHandler` interface (or function type) in a new file `internal/mcp/executor/handler.go`:
   ```go
   type SpecialToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)
   ```
2. Build a `map[string]SpecialToolHandler` registry, populated in `internal/mcp/executor/registry.go`.
3. `ExecuteSpecialTool` becomes a 10-line lookup-and-dispatch function.
4. Each group of related handlers (analysis, query, cleanup) lives in its own file.

The execution lifecycle (classify error, record metrics) is extracted into a shared `executeWithMetrics` helper so it is not repeated.

**Risk**: Medium. The refactor touches `ExecuteSpecialTool` and all its callers indirectly via the registry. Existing unit tests cover the happy path; additional tests for the registry dispatch and the metrics wrapper should be added.

---

### H-2 — Duplicate Utility Functions

**Locations** (full inventory after codebase scan):

| Function | Files |
|----------|-------|
| `parseTimestamp` | `internal/analyzer/workflow.go:443`, `internal/query/turnindex/turnindex.go:38` (private), `internal/query/context.go:182` |
| `buildTurnIndex` | `internal/analyzer/workflow.go:248` — canonical version is `turnindex.BuildTurnIndex` |
| `getActionType` | `internal/analyzer/workflow.go:419`, `internal/query/file_access.go:121` |

Additionally, `internal/analyzer/work_patterns.go` calls `parseTimestamp(tc.Timestamp)` on a `ToolCall.Timestamp` field (line 80), relying on the private copy in `workflow.go`.

**Root cause**: `turnindex.parseTimestamp` is unexported (lowercase), making it invisible to other packages. `internal/query/context.go` therefore carries its own copy, and `internal/analyzer/workflow.go` maintains a third.

**Fix**:
1. Export `parseTimestamp` from `turnindex` as `ParseTimestamp` (rename the function).
2. Update `turnindex.GetToolCallTimestamp` to call `ParseTimestamp` internally.
3. Remove the private `buildTurnIndex` and `parseTimestamp` from `workflow.go`; import `internal/query/turnindex` instead.
4. Update `work_patterns.go` to call `turnindex.ParseTimestamp`.
5. Remove the private `parseTimestamp` from `internal/query/context.go`; call `turnindex.ParseTimestamp`.
6. Move `getActionType` to `internal/types` (as `FileActionType`) so both `analyzer` and `query` can use it without a cycle.

**Risk**: Low. The function logic is identical across all copies. Exporting from `turnindex` is a non-breaking additive change; existing internal callers continue to work.

---

### H-3 — `workflow.go` God File (479 lines, 18 functions)

**File**: `internal/analyzer/workflow.go`

The file contains at least five distinct responsibilities:

| Concern | Functions |
|---------|-----------|
| Tool sequence detection | `DetectToolSequences`, `extractToolCallsWithTurns`, `findAllSequences`, `calculateSequenceTimeSpan` |
| File churn analysis | `DetectFileChurn`, `fileAccessStats` type |
| Idle period detection | `DetectIdlePeriods` |
| Turn/timestamp indexing | `buildTurnIndex`, `parseTimestamp`, `getToolCallTimestamp` |
| Context extraction | `extractTurnContext`, `extractFileFromToolCall`, `extractCommandFromToolCall`, `getActionType` |

**Proposed split**:
- `internal/analyzer/sequences.go` — tool sequence detection (≤150 lines)
- `internal/analyzer/churn.go` — file churn analysis (≤100 lines)
- `internal/analyzer/idle.go` — idle period detection (≤80 lines)
- Timestamp/turn helpers removed (delegated to `internal/query/turnindex` — see H-2)
- File/command extraction helpers moved to a shared location or kept in a small `helpers.go`

Each new file stays comfortably within the 200-line stage limit.

**Risk**: Low. Pure refactor, no behaviour change. Existing tests in `workflow_test.go` cover the public API.

---

### H-4 — `pipeline` Package Inverted Dependency (Design Smell)

**File**: `internal/mcp/pipeline/pipeline.go` (lines 26–30)

`AdaptResponseFunc` and `SerializeResponseFunc` are injected as function-type parameters into `BuildStatsFirstResponse` and `BuildStandardResponse`. The comment in the file says explicitly: *"injected to avoid an import cycle with cmd/mcp-server"*.

The root cause is that `AdaptResponse` and `SerializeResponse` live in `internal/mcp/response`, and `pipeline` could import them directly. The injection was added when these functions lived in `cmd/mcp-server`, but they have since been moved to `internal/mcp/response`.

**Fix**:
1. Remove `AdaptResponseFunc` and `SerializeResponseFunc` type declarations from `pipeline.go`.
2. Import `internal/mcp/response` directly in `pipeline.go`.
3. Update callers (`internal/mcp/executor/executor.go`) to stop injecting the function literals.

**Risk**: Low-medium. Removing the injection simplifies call sites but requires verifying no import cycle is reintroduced. Since `response` does not import `pipeline`, the cycle risk is absent.

---

### M-6 — Half-finished `parser` → `types` Migration

**File**: `internal/parser/aliases.go`

`internal/parser` is now a pure alias shim:
```go
type SessionEntry = types.SessionEntry
type ToolCall     = types.ToolCall
// ...
var ExtractToolCalls = types.ExtractToolCalls
```

**44 production Go files** and 42 test files (86 total, excluding worktrees) still import `internal/parser`. The migration goal is to make all production code import `internal/types` directly and deprecate `internal/parser`.

**Constraints**:
- Actual count of non-test Go files importing `internal/parser`: **44 files** (confirmed via `grep -rl`, excluding worktrees). Test files: 42 more.
- The project's 500-line/phase and 200-line/stage limits mean this migration must be batched. A realistic approach is ~15–20 production files per stage, each requiring approximately 1 import-line change.
- Test files should be migrated in a separate pass to avoid oversized stages.
- The `internal/parser` package must not be deleted until all imports are redirected. After migration, `aliases.go` can be replaced with a deprecation comment file guarded by a build tag.
- C-1 (Phase 83) must complete before M-6, since C-1 removes `parser` from the most-visible interface file.

**Risk**: Low per-file (type aliases are identical), but high aggregate risk if done carelessly. Each stage must run `make commit` to catch regressions immediately.

---

## Solution Design

### Ordering Rationale

The issues should be resolved in this order:

1. **H-2 first** — eliminates duplicates before H-3 splits the file (if H-3 went first, the duplicates would move into new files).
2. **H-3 after H-2** — splitting `workflow.go` is clean once duplicates are removed.
3. **C-1 and H-4 in parallel** — independent of each other and of H-2/H-3.
4. **C-2** — executor refactor is independent but benefits from a clean state after H-4 removes the injected function types.
5. **M-6 last** — largest mechanical change; benefits from C-1 being done (fewer `parser` imports needed after interface fix).

### Phase Overview

| Phase | Issue(s) | Description |
|-------|----------|-------------|
| 81 | H-2 | Deduplicate `buildTurnIndex`/`parseTimestamp`/`getActionType` |
| 82 | H-3 | Split `workflow.go` into focused files |
| 83 | C-1 | Fix analyzer interfaces to use `types` not `parser` |
| 84 | H-4 | Remove injected function types from `pipeline` |
| 85 | C-2 | Refactor `ExecuteSpecialTool` to a handler registry |
| 86 | M-6 | Complete `parser` → `types` migration (production files) |

---

## Tradeoff Analysis

### Handler Registry (C-2)

**Pro**: OCP compliance, each tool handler is independently testable, new tools require zero changes to `executor.go`.
**Con**: Indirection; harder to see at a glance which tools are registered (mitigated by `registry.go` being the single source of truth).
**Alternative**: Extract into multiple `executeXxx` methods on `ToolExecutor`. Rejected: still requires editing the switch statement to add a new case.

### Full `parser` Removal (M-6)

**Pro**: Eliminates the cognitive load of the double-import question.
**Con**: Large mechanical change; high risk of merge conflicts if other work touches the same files.
**Alternative**: Keep `internal/parser` as a permanent thin facade. Rejected: the facade is already causing C-1 (interface leakage) and will continue to confuse contributors.

### Import `response` Directly from `pipeline` (H-4)

**Pro**: Removes function injection boilerplate, makes data flow explicit.
**Con**: Slightly tighter coupling between `pipeline` and `response`.
**Alternative**: Extract a `Serializer` interface in `pipeline`. Rejected: over-engineering; the dependency is stable and unidirectional.

---

## Risks

| Risk | Mitigation |
|------|------------|
| C-2 registry breaks tool dispatch | Existing executor integration tests catch missing registrations |
| M-6 import substitution introduces cycle | `go build ./...` in each stage CI step |
| H-3 split introduces private symbol visibility issue | All split files remain in `package analyzer` |
| H-4 direct import creates cycle | Verified: `response` does not (transitively) import `pipeline` |
| H-2 exporting `ParseTimestamp` from `turnindex` breaks callers | No external callers; only internal callers (all in `internal/`) |
| H-2 `query/context.go` has a third private `parseTimestamp` | Must be migrated in same phase as `workflow.go` copy |
| M-6 scope is larger than originally estimated (129 production files) | Batched stages of ≤20 files; single-pass `gofmt` per stage |

---

## References

- `internal/analyzer/interfaces.go` — C-1 source
- `internal/mcp/executor/executor.go` — C-2 source
- `internal/analyzer/workflow.go` — H-2, H-3 source
- `internal/mcp/pipeline/pipeline.go` — H-4 source
- `internal/parser/aliases.go` — M-6 source
- `internal/query/turnindex/` — existing centralised turn index (Phase 80)
- `internal/mcp/response/adapter.go` — contains `AdaptResponse`/`SerializeResponse` (H-4 target)
- `docs/plans/78-80-query-architecture-cleanup.md` — preceding plan

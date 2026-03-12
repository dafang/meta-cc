# Plan 76–77: Clean Up Duplicate Types in `cmd/mcp-server`

**Status**: Completed
**Proposal**: [docs/proposals/proposal-cleanup-mcp-server-duplicates.md](../proposals/proposal-cleanup-mcp-server-duplicates.md)

---

## Overview

Phase 74-75 extracted `QueryExecutor`, `JQRunner`, and related helpers from `cmd/mcp-server` into `internal/mcp/query` and `internal/mcp/executor`. The extraction left behind a residue in `cmd/mcp-server` that was never cleaned up: a duplicate `JQRunner` interface, a `QueryExecutor` wrapper struct, five type aliases, and a duplicated query execution path in `handlers_query.go`.

This plan removes those duplicates in two phases:

| Phase | Scope | Key deliverable |
|---|---|---|
| 76 | Migrate tests off local wrappers; delete `query_executor.go` and `query_runner.go` | `cmd/mcp-server` has no `QueryExecutor` wrapper or duplicate `JQRunner` |
| 77 | Migrate `handlers_query.go` to delegate to `internal/mcp/executor`; remove remaining residue | ArchGuard no longer reports duplicate-type signals for `JQRunner` and `QueryExecutor` |

**Phase dependencies**: Phase 77 depends on Phase 76.

```text
Phase 76 → Phase 77
```

---

## Phase 76: Delete Wrapper Types and Migrate Tests

### Objectives

Remove the `cmd/mcp-server.QueryExecutor` wrapper struct, the duplicate `cmd/mcp-server.JQRunner` interface, and all five type aliases. Migrate tests to use canonical types from `internal/mcp/query` directly.

### Acceptance Criteria

- `cmd/mcp-server/query_executor.go` is deleted entirely
- `cmd/mcp-server/query_runner.go` is deleted entirely
- All test files in `cmd/mcp-server` compile and pass against `querypkg.*` types without any local alias
- `grep -r "JQRunner" cmd/` returns no results
- `make commit` passes with no new test failures

### Stages

#### Stage 76.1 — Pre-migration: Audit alias and wrapper references

- Change budget: ≤50 lines (test changes only; no production code changed)
- Tasks:
  - Run `grep -rn "executor\.cache\|compileExpression\|buildExpression\|streamFiles\|processFile\|ExpressionCache\|parsedTimeRange\|QueryResult\|QueryRequest\|QueryResponse\|parseTimeRange" cmd/mcp-server/ --include="*.go"` and record all call sites
  - In `cmd/mcp-server/query_executor_test.go`: replace `executor.cache` with `executor.Cache` (accessing the public `Cache` field of `querypkg.QueryExecutor`)
  - Confirm the change compiles: `go build ./cmd/mcp-server/...`
- Files:
  - `cmd/mcp-server/query_executor_test.go` — `executor.cache` → `executor.Cache` (line 158)
- Tests:
  - Run `go test ./cmd/mcp-server/...` to confirm existing tests still pass before any deletion
- Exit criteria:
  - No `executor.cache` references remain in `cmd/mcp-server/`
  - `make dev` passes

#### Stage 76.2 — Delete wrapper files and migrate test files simultaneously

- Change budget: ≤200 lines
- Tasks:
  - Delete `cmd/mcp-server/query_executor.go` entirely (removes: type aliases, `QueryExecutor` wrapper struct, lowercase delegator methods, `JQRunner` interface, compile-time assertion)
  - Delete `cmd/mcp-server/query_runner.go` entirely (removes: `RunQuery` and `RunQueryWithTimeRange` shadow methods)
  - In `cmd/mcp-server/query_executor_test.go`:
    - Add `import querypkg "github.com/yaleh/meta-cc/internal/mcp/query"` if not present
    - Replace `ExpressionCache` → `querypkg.ExpressionCache`
    - Replace `parsedTimeRange` → `querypkg.ParsedTimeRange`
    - Replace lowercase wrapper calls (`executor.compileExpression`, `executor.buildExpression`, `executor.streamFiles`, `executor.processFile`, `executor.processFileWithTimeRange`) with their exported equivalents on `querypkg.QueryExecutor` (`executor.CompileExpression`, etc.)
  - In `cmd/mcp-server/query_runner_test.go` — two distinct groups require different treatment:
    - **Mock interface tests (lines 10–69)**: Replace `JQRunner` with `querypkg.JQRunner`, `QueryResult` → `querypkg.QueryResult`, `parsedTimeRange` → `querypkg.ParsedTimeRange`. Evaluate whether `mockJQRunner` tests (`TestJQRunner_InterfaceCompliance`, `TestMockJQRunner`, `TestMockJQRunner_WithTimeRange`, `TestMockJQRunner_Error`) add value without the local interface; if not, delete these four tests.
    - **Functional tests (lines 71–131)**: `TestQueryExecutor_RunQuery`, `TestQueryExecutor_RunQuery_InvalidFilter`, `TestQueryExecutor_RunQueryWithTimeRange` must be **migrated, not deleted**. Update: `NewQueryExecutor(...)` → `querypkg.NewQueryExecutor(...)` and `parseTimeRange(...)` (line 118) → `querypkg.ParseTimeRange(...)`.
  - In `cmd/mcp-server/handlers_query.go`:
    - Remove the comment on line 10 that references the deleted `query_executor.go` (`parsedTimeRange and parseTimeRange are defined in query_executor.go`)
    - Adjust any import that referenced the now-deleted aliases
- Files:
  - `cmd/mcp-server/query_executor.go` — **delete**
  - `cmd/mcp-server/query_runner.go` — **delete**
  - `cmd/mcp-server/query_executor_test.go` — migrate type references and method names
  - `cmd/mcp-server/query_runner_test.go` — migrate functional tests (`NewQueryExecutor` → `querypkg.NewQueryExecutor`, `parseTimeRange` → `querypkg.ParseTimeRange`); evaluate and optionally delete mock interface tests
  - `cmd/mcp-server/handlers_query.go` — remove stale comment referencing deleted `query_executor.go`
- Tests:
  - `go vet ./cmd/mcp-server/...` after deletion to confirm no missing method references
  - `go test ./cmd/mcp-server/...`
  - `grep -r "JQRunner" cmd/` must return no results
  - `make commit`
- Exit criteria:
  - `query_executor.go` and `query_runner.go` are gone
  - All tests in `cmd/mcp-server` pass
  - `make commit` is green

### Phase 76 Validation

- `make commit` passes
- `grep -r "JQRunner" cmd/` returns no output
- `grep -rn "ExpressionCache\|parsedTimeRange\|QueryResult\|QueryRequest\|QueryResponse" cmd/mcp-server/*.go` returns only `querypkg.`-qualified references (no bare aliases)
- `go vet ./cmd/mcp-server/...` returns no errors

---

## Phase 77: Migrate `handlers_query.go` and Remove Remaining Residue

### Objectives

Eliminate the duplicated query execution path in `cmd/mcp-server/handlers_query.go` by delegating to `internal/mcp/executor`. Remove the import-guard sentinel in `executor.go` if it is no longer needed. Confirm ArchGuard no longer reports the duplicate-type signals.

### Acceptance Criteria

- `cmd/mcp-server.ToolExecutor.executeQuery` and `executeQueryWithTimeRange` in `handlers_query.go` delegate to `e.ToolExecutor.ExecuteQuery` and `e.ToolExecutor.ExecuteQueryWithTimeRange` from `internal/mcp/executor`
- The `NewQueryExecutor` call at line 33 of `handlers_query.go` is removed
- `var _ mcquery.QueryResult` sentinel in `executor.go` is evaluated and removed if no longer required
- `make commit` passes with no new test failures
- ArchGuard no longer reports `JQRunner` or `QueryExecutor` as duplicate-type signals

### Stages

#### Stage 77.1 — Migrate `executeQuery` and `executeQueryWithTimeRange` in `handlers_query.go`

- Change budget: ≤100 lines
- Tasks:
  - Replace the body of `executeQuery` (lines 21–42 of `handlers_query.go`) with a single call to `e.ToolExecutor.ExecuteQuery(ctx, args)`
  - Replace the body of `executeQueryWithTimeRange` (lines 44–52 of `handlers_query.go`) with a single call to `e.ToolExecutor.ExecuteQueryWithTimeRange(ctx, args)`
  - Remove the `NewQueryExecutor(baseDir)` call (line 33) — it is no longer referenced after the body is replaced
  - Confirm the `querypkg` import in `handlers_query.go` is still required by the remaining wrapper functions (`getQueryBaseDir`, `getJSONLFiles`, `loadTurnsForSession`); do not remove the import if those functions still call `querypkg.*`
  - Run `go build ./cmd/mcp-server/...` to confirm no unused import or missing method errors
- Files:
  - `cmd/mcp-server/handlers_query.go` — replace `executeQuery` and `executeQueryWithTimeRange` bodies; remove `NewQueryExecutor` call
- Tests:
  - `go test ./cmd/mcp-server/...` — all handler tests must pass
  - Focus on `handlers_query_test.go`, `handlers_query_session_scope_test.go`, `handlers_query_workingdir_test.go`
  - `make commit`
- Exit criteria:
  - `handlers_query.go` no longer calls `NewQueryExecutor`
  - `go build ./cmd/mcp-server/...` is clean
  - `make commit` passes

#### Stage 77.2 — Remove import-guard sentinel and final cleanup

- Change budget: ≤50 lines
- Tasks:
  - In `cmd/mcp-server/executor.go` line 106: evaluate whether `var _ mcquery.QueryResult` is still needed
    - If `executor.go` has no remaining direct use of `mcquery` types after Phase 76 changes, delete the sentinel and the `mcquery` import alias
    - If other code in `executor.go` still references `mcquery` types (e.g., `toolPipelineConfig`), leave the sentinel and document the reason
  - Run `go build ./cmd/mcp-server/...` and `go vet ./cmd/mcp-server/...`
  - Run ArchGuard scan and confirm no remaining duplicate-type signals for `JQRunner` or `QueryExecutor`
  - Update any architecture documentation that still describes the pre-cleanup state
- Files:
  - `cmd/mcp-server/executor.go` — remove sentinel (line 106) and `mcquery` import if no longer needed
- Tests:
  - `go test ./cmd/mcp-server/...`
  - `make commit`
- Exit criteria:
  - `make commit` passes
  - ArchGuard reports no duplicate-type signals for `JQRunner` or `QueryExecutor`
  - No dead import-guard sentinels remain

#### Stage 77.3 — Post-cleanup documentation and open-question resolution

- Change budget: ≤50 lines
- Tasks:
  - Resolve Open Question 1 from the proposal: decide whether `query_runner_test.go` mock tests should be migrated to `internal/mcp/query` package tests; if yes, add them there with a `var _ querypkg.JQRunner = (*mockJQRunner)(nil)` assertion; if no, confirm deletion was correct
  - Resolve Open Question 3 from the proposal: record a decision on whether to track a follow-on Phase 78 to close the remaining `cmd/mcp-server → internal/mcp/query` direct import (blocked by `handlers_stage1.go`, `handlers_stage2.go`, and any remaining sentinel in `executor.go`)
  - Update `docs/proposals/proposal-cleanup-mcp-server-duplicates.md` status to `Implemented`
  - Update this plan's **Status** field to `Completed`
- Files:
  - `docs/proposals/proposal-cleanup-mcp-server-duplicates.md` — status update
  - `docs/plans/76-77-cleanup-mcp-server-duplicates.md` — status update
  - Optionally: `internal/mcp/query/*_test.go` — add migrated mock interface tests if decided in step 1
- Tests:
  - `go test ./internal/mcp/query/...` if new tests are added
  - `make commit`
- Exit criteria:
  - Proposal status is `Implemented`
  - Plan status is `Completed`
  - Any new tests in `internal/mcp/query` pass

### Phase 77 Validation

- `make commit` passes
- `cmd/mcp-server/handlers_query.go` calls `e.ToolExecutor.ExecuteQuery` / `e.ToolExecutor.ExecuteQueryWithTimeRange` (no local `NewQueryExecutor` invocation)
- `grep -rn "NewQueryExecutor" cmd/mcp-server/handlers_query.go` returns no results
- ArchGuard: `JQRunner` and `QueryExecutor` duplicate-type signals resolved

---

## Test Strategy

- **TDD where applicable**: Stage 76.2 migrates existing tests to the canonical types. No new production logic is introduced, so no new test-first stubs are needed. Confirm `go test` passes at each sub-step before proceeding to the next file.
- **Mechanical rename verification**: After each file migration, run `go vet ./cmd/mcp-server/...` to catch missing method signatures before running the full test suite.
- **Coverage requirement**: No coverage regression is acceptable. The moved test coverage (operating on `querypkg.QueryExecutor` directly) must exercise the same code paths as before. Target ≥80% for `cmd/mcp-server` and `internal/mcp/query`.
- **Integration smoke test**: After Phase 77 Stage 77.1, run a focused integration case for `tools/call` with a query tool to confirm the delegated path returns the same result shape.

---

## Dependencies

- **Depends on**: Phase 74-75 (`internal/mcp/query` and `internal/mcp/executor` must exist with canonical types and methods). These are already completed as of the current main branch.
- **Blocks**: A potential Phase 78 that would fully close the `cmd/mcp-server → internal/mcp/query` import by migrating `handlers_stage1.go` and `handlers_stage2.go` behind `internal/mcp/executor`.
- **No internal cross-phase dependency inversion**: Phase 77 may not start until Phase 76 `make commit` is green.

---

## Non-Goals (from proposal)

- Changing `JQRunner`, `QueryExecutor`, or `ExpressionCache` behavior in `internal/mcp/query`
- Modifying `internal/mcp/executor` logic
- Changing MCP wire behavior (tool names, parameters, response shapes)
- Fully eliminating the `cmd/mcp-server → internal/mcp/query` import (blocked by `handlers_stage1.go` and `handlers_stage2.go`; requires a follow-on phase)
- Resolving other technical debt in `cmd/mcp-server` beyond the duplicate types described here

---

## Total Estimates

| Stage | Description | Estimated LOC |
|---|---|---|
| 76.1 | Migrate `executor.cache` → `executor.Cache` in test | ~10 |
| 76.2 | Delete wrapper files; migrate both test files (including functional tests in `query_runner_test.go`) | ~170 |
| 77.1 | Replace `executeQuery`/`executeQueryWithTimeRange` bodies | ~40 |
| 77.2 | Remove import-guard sentinel; ArchGuard validation | ~20 |
| 77.3 | Documentation and open-question resolution | ~30 |
| **Total** | 5 stages across 2 phases | **~270 lines** |

Well within the phase limit (≤500 lines) and per-stage limit (≤200 lines).

---

## Validation Checklist

- [ ] `cmd/mcp-server/query_executor.go` deleted
- [ ] `cmd/mcp-server/query_runner.go` deleted
- [ ] `grep -r "JQRunner" cmd/` returns no output
- [ ] `grep -rn "executor\.cache" cmd/mcp-server/` returns no output
- [ ] All type alias references in test files use `querypkg.` qualifier
- [ ] `query_runner_test.go` functional tests (`TestQueryExecutor_RunQuery*`) migrated to use `querypkg.NewQueryExecutor` and `querypkg.ParseTimeRange`
- [ ] `cmd/mcp-server/handlers_query.go` delegates `executeQuery`/`executeQueryWithTimeRange` to `internal/mcp/executor`
- [ ] `NewQueryExecutor` call removed from `handlers_query.go`
- [ ] `var _ mcquery.QueryResult` sentinel in `executor.go` evaluated (removed or justified)
- [ ] `make commit` passes after each stage
- [ ] ArchGuard: `JQRunner` and `QueryExecutor` duplicate-type signals resolved
- [ ] Proposal status updated to `Implemented`

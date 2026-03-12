# Proposal: Remove Duplicate Types in `cmd/mcp-server` After Phase 74-75 Extraction

**Status**: Draft
**Date**: 2026-03-12
**Related**: [proposal-mcp-server-package-split.md](proposal-mcp-server-package-split.md)

---

## Background

Phase 74-75 successfully extracted `QueryExecutor`, `JQRunner`, and related helpers from `cmd/mcp-server` into `internal/mcp/query` and `internal/mcp/executor`. The internal packages now own the canonical implementations.

However, the extraction left behind a residue in `cmd/mcp-server` that was never cleaned up:

### Duplicate Type: `JQRunner`

`internal/mcp/query.JQRunner` is the canonical interface declared in `internal/mcp/query/query.go` (lines 383–390):

```go
type JQRunner interface {
    RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error)
    RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr ParsedTimeRange) (QueryResult, error)
}
```

`cmd/mcp-server` re-declares a structurally identical interface in `cmd/mcp-server/query_executor.go` (lines 69–73):

```go
type JQRunner interface {
    RunQuery(ctx context.Context, files []string, filter, transform string, limit int) (QueryResult, error)
    RunQueryWithTimeRange(ctx context.Context, files []string, filter, transform string, limit int, tr parsedTimeRange) (QueryResult, error)
}
```

Both have two identical methods. The local `parsedTimeRange` is itself a type alias for `querypkg.ParsedTimeRange` (line 19), so the two interfaces are structurally indistinguishable.

`query_runner_test.go` carries a second compile-time assertion for this interface (`var _ JQRunner = (*QueryExecutor)(nil)` at line 25), independent of the one in `query_executor.go` (line 76). Both assertions become dead code once the wrapper is removed.

### Duplicate Type: `QueryExecutor`

`internal/mcp/query.QueryExecutor` is the canonical struct with 8 exported methods (`BuildExpression`, `CompileExpression`, `StreamFiles`, `StreamFilesWithTimeRange`, `ProcessFile`, `ProcessFileWithTimeRange`, `RunQuery`, `RunQueryWithTimeRange`).

`cmd/mcp-server.QueryExecutor` in `query_executor.go` (lines 22–27) wraps the canonical struct via struct embedding and re-exposes the same methods as unexported lowercase delegators (`buildExpression`, `compileExpression`, `streamFiles`, `streamFilesWithTimeRange`, `processFile`, `processFileWithTimeRange`). The public `RunQuery` / `RunQueryWithTimeRange` methods are split into a separate file (`query_runner.go`) but are equally trivial one-liner delegations. The wrapper adds no logic whatsoever.

**[CORRECTION]** The proposal originally stated the wrapper "exposes the same 8 methods as unexported lowercase delegators." This is imprecise. Because `cmd/mcp-server.QueryExecutor` embeds `*querypkg.QueryExecutor`, **all 8 exported methods of the canonical struct are promoted automatically** onto the wrapper. The lowercase delegators (`buildExpression` etc.) are *additional* unexported aliases added for test backward compatibility. The `RunQuery` and `RunQueryWithTimeRange` in `query_runner.go` **shadow** the promoted methods with identical one-liner bodies, making them redundant even as shadows.

### Concrete Coupling: `cache *querypkg.ExpressionCache`

`cmd/mcp-server.QueryExecutor` carries an explicit unexported field `cache *querypkg.ExpressionCache` (line 26), a concrete struct pointer copied from `inner.Cache` at construction time (line 34). This field exists solely to satisfy `query_executor_test.go` line 158, which accesses `executor.cache.Get(expr)` directly.

This is a concrete-type coupling: a command package directly exposing the internal cache field of an internal struct for white-box testing.

**[NOTE]** The canonical `querypkg.ExpressionCache` struct already has a public `Cache` field (line 118 of `internal/mcp/query/query.go`), so tests can call `executor.Cache.Get(expr)` on the canonical type directly without any accessor addition. `ExpressionCache.Get()` is also a public method. Stage 1 (Option B) is therefore over-engineered: the white-box coupling can be eliminated with zero changes to `internal/mcp/query` by replacing `executor.cache` with `executor.Cache` in the test after the wrapper is removed.

### Type Aliases for Internal Types

`query_executor.go` re-exports five internal types as aliases in the `main` package (lines 11–19):

- `ExpressionCache = querypkg.ExpressionCache`
- `QueryRequest = querypkg.QueryRequest`
- `QueryResponse = querypkg.QueryResponse`
- `QueryResult = querypkg.QueryResult`
- `parsedTimeRange = querypkg.ParsedTimeRange`

**[CORRECTION]** The original proposal listed "four" aliases; the actual count is **five** (including `parsedTimeRange`). The aliases inflate the local namespace and make the package boundary opaque, but their removal cascades into a significant number of test file changes (see Scope section below).

### Additional Wrappers Not Mentioned in the Original Proposal

**[MISSING FROM ORIGINAL]** The original proposal scoped the cleanup to `query_executor.go` and `query_runner.go`. Code inspection reveals two additional files with the same wrapper pattern that must be addressed in the same phase:

1. **`cmd/mcp-server/handlers_query.go`** (lines 21–52): Declares `executeQuery` and `executeQueryWithTimeRange` as methods on `cmd/mcp-server.ToolExecutor`. These are **exact duplicates** of `internal/mcp/executor.ExecuteQuery` and `internal/mcp/executor.ExecuteQueryWithTimeRange` (lines 442–471 of `internal/mcp/executor/executor.go`). Both implementations call `NewQueryExecutor(baseDir)`, compile the filter, list files, and call `StreamFilesWithTimeRange`. The `cmd/mcp-server` versions differ only in calling the local `NewQueryExecutor` wrapper instead of `mcquery.NewQueryExecutor` directly.

2. **`cmd/mcp-server/handlers_stage1.go`** (entire file) and **`cmd/mcp-server/handlers_stage2.go`** (entire file): These files contain only single-line delegators to `querypkg.*` functions. They are not duplicates in the type-duplication sense, but they represent the same boundary-violation pattern (direct `cmd/mcp-server → internal/mcp/query` coupling) and will remain after the `query_executor.go` cleanup unless explicitly addressed.

3. **`cmd/mcp-server/executor.go` line 106**: `var _ mcquery.QueryResult` — an import-guard sentinel that keeps the `mcquery` import alive. This is residue from the Phase 74-75 migration. If the `cmd/mcp-server → internal/mcp/query` direct import path is ever to be closed, this sentinel is the last blocker in `executor.go`.

---

## Current Dependency Relationship

The actual import graph (verified by code inspection) is:

```
cmd/mcp-server → internal/mcp/query     (files: query_executor.go, handlers_query.go,
                                                 handlers_stage1.go, handlers_stage2.go,
                                                 executor.go)
cmd/mcp-server → internal/mcp/executor  (files: executor.go, handlers_convenience.go)
internal/mcp/executor → internal/mcp/query
```

**[CORRECTION]** The original proposal stated "5 import references" for `cmd/mcp-server → internal/mcp/query`. The actual count is **5 source files** that carry the import, not 5 references within a single file. This is a meaningfully larger footprint than the proposal implied.

The desired long-term topology is:

```
cmd/mcp-server → internal/mcp/executor (thin wiring only)
internal/mcp/executor → internal/mcp/query
```

However, the direct `cmd/mcp-server → internal/mcp/query` path **cannot be fully closed in this phase** because `handlers_stage1.go` and `handlers_stage2.go` call `querypkg.*` functions that have no corresponding surface in `internal/mcp/executor`. Moving those functions behind the executor is out of scope (see Non-Goals). The achievable goal in this phase is removing the `query_executor.go` and `query_runner.go` duplicates and the redundant `executeQuery`/`executeQueryWithTimeRange` wrapper in `handlers_query.go`.

---

## Goals

- Remove the duplicate `JQRunner` interface from `cmd/mcp-server`.
- Remove the duplicate `QueryExecutor` wrapper struct from `cmd/mcp-server`.
- Remove the `cache *querypkg.ExpressionCache` concrete coupling from the wrapper.
- Migrate `cmd/mcp-server.ToolExecutor.executeQuery` and `executeQueryWithTimeRange` in `handlers_query.go` to delegate to `e.ToolExecutor.ExecuteQuery` / `ExecuteQueryWithTimeRange` from `internal/mcp/executor`.
- Remove the five type aliases from `query_executor.go`.
- Preserve all existing test behavior.

## Non-Goals

- Changing `JQRunner`, `QueryExecutor`, or `ExpressionCache` behavior in `internal/mcp/query`.
- Modifying `internal/mcp/executor` logic.
- Changing MCP wire behavior (tool names, parameters, response shapes).
- Fully eliminating the `cmd/mcp-server → internal/mcp/query` import (blocked by `handlers_stage1.go`, `handlers_stage2.go`, and the import-guard sentinel in `executor.go`; a follow-on phase is required).
- Resolving other technical debt in `cmd/mcp-server` beyond the duplicate types described here.

---

## Proposed Design

### Stage 1: Migrate tests off the `cache` field

`query_executor_test.go` line 158 directly accesses `executor.cache` to test LRU cache hit rate. This white-box coupling must be resolved before the wrapper can be removed.

**[REVISED — Option A is now preferred]**: The canonical `querypkg.QueryExecutor` already exposes a public `Cache *ExpressionCache` field, and `ExpressionCache.Get()` is a public method. Adding a `CacheLen()` accessor (Option B from the original proposal) is unnecessary over-engineering. The test in `query_executor_test.go` should be migrated to operate on `querypkg.QueryExecutor` directly after removing the wrapper, referencing `executor.Cache.Get(expr)` instead of `executor.cache.Get(expr)`. No changes to `internal/mcp/query/query.go` are required.

The `TestExpressionCache` test (lines 78–140) directly instantiates `ExpressionCache` by value (via the `ExpressionCache` type alias). After alias removal, this test must import `querypkg` explicitly and use `querypkg.ExpressionCache`. This is a mechanical change with no behavioral impact.

### Stage 2: Remove the `QueryExecutor` wrapper struct and `query_runner.go`

Once tests no longer reference `executor.cache`, the wrapper in `cmd/mcp-server/query_executor.go` is deleted:

- `cmd/mcp-server/query_executor.go`: delete entirely. The type aliases, the wrapper struct, the lowercase method wrappers, the `JQRunner` interface, and its compile-time assertion all disappear together.
- `cmd/mcp-server/query_runner.go`: delete entirely. The two methods it declares are promoted automatically from the embedded `*querypkg.QueryExecutor` — they are redundant even before the wrapper exists.
- `cmd/mcp-server/query_runner_test.go` has two distinct groups that require different treatment:
  - **Mock interface tests (lines 10–69)**: the `mockJQRunner` struct, `var _ JQRunner` assertions, and four mock tests (`TestJQRunner_InterfaceCompliance`, `TestMockJQRunner`, `TestMockJQRunner_WithTimeRange`, `TestMockJQRunner_Error`) will need `JQRunner` and `parsedTimeRange` types resolved from `querypkg` after alias removal. Evaluate whether the mock tests add value once the local `JQRunner` interface is gone; they may be deleted or migrated to `internal/mcp/query` package tests.
  - **Functional tests (lines 71–131)**: `TestQueryExecutor_RunQuery`, `TestQueryExecutor_RunQuery_InvalidFilter`, and `TestQueryExecutor_RunQueryWithTimeRange` use `NewQueryExecutor`, `parseTimeRange`, and the wrapper's `RunQuery`/`RunQueryWithTimeRange`. These tests must be **migrated** (not deleted): update `NewQueryExecutor(...)` → `querypkg.NewQueryExecutor(...)` and `parseTimeRange(...)` → `querypkg.ParseTimeRange(...)`. The functional coverage they provide is valuable and should be preserved.
- All tests in `query_executor_test.go` that call lowercase wrappers (`executor.compileExpression`, `executor.buildExpression`, `executor.streamFiles`, `executor.processFile`, `executor.processFileWithTimeRange`) must be updated to call the promoted exported methods (`executor.CompileExpression`, etc.) on `querypkg.QueryExecutor` directly.

### Stage 3: Migrate `handlers_query.go` wrapper methods

**[NEW — not in original proposal]** `cmd/mcp-server.ToolExecutor.executeQuery` and `executeQueryWithTimeRange` are exact duplicates of `internal/mcp/executor.ToolExecutor.ExecuteQuery` and `ExecuteQueryWithTimeRange`. The migration:

- Replace the bodies of `executeQuery` and `executeQueryWithTimeRange` in `handlers_query.go` with delegations to `e.ToolExecutor.ExecuteQuery` and `e.ToolExecutor.ExecuteQueryWithTimeRange` respectively.
- Remove the `NewQueryExecutor` call at line 33 of `handlers_query.go`.
- This eliminates the last non-alias, non-sentinel use of `querypkg` in `handlers_query.go`. After this stage, `handlers_query.go` no longer needs to import `internal/mcp/query` directly (the `querypkg` delegator functions `getQueryBaseDir`, `getJSONLFiles`, `loadTurnsForSession` at lines 54–67 still wrap `querypkg.*` calls; these are covered by the Non-Goals — fully closing this import requires a follow-on phase).

### Stage 4: Remove type aliases

The five type aliases (`ExpressionCache`, `QueryRequest`, `QueryResponse`, `QueryResult`, `parsedTimeRange`) can be removed from `query_executor.go` (which is deleted in Stage 2). All references in test files must be updated to use `querypkg.` qualifiers:

- `query_executor_test.go`: `ExpressionCache` → `querypkg.ExpressionCache`, `parsedTimeRange` → `querypkg.ParsedTimeRange`
- `query_runner_test.go`: `QueryResult` → `querypkg.QueryResult`, `parsedTimeRange` → `querypkg.ParsedTimeRange`
- Any other test file in `cmd/mcp-server` that references these aliases via the type alias shorthand

**[IMPORTANT]** Stage 4 is **not independent** of Stage 2. Since `query_executor.go` is the file that declares the aliases, deleting the file in Stage 2 automatically removes the aliases. Stage 4 is the test-file migration work that must accompany Stage 2, not a separate follow-on stage.

### Revised Execution Order

The original four-stage sequencing is incorrect because Stage 4 cannot be deferred past Stage 2 (deleting `query_executor.go` immediately breaks all alias references). The correct sequence is:

1. **Stage 1**: Migrate `executor.cache` references in tests to `executor.Cache` (if keeping tests on the wrapper type) or plan for the test rewrite in Stage 2.
2. **Stage 2**: Delete `query_executor.go` and `query_runner.go`. Simultaneously update all test files that used the aliases or lowercase wrappers. Run `make commit` to validate.
3. **Stage 3**: Migrate `handlers_query.go` to delegate to `internal/mcp/executor`. Run `make commit`.
4. **Stage 4 (optional)**: Clean up `var _ mcquery.QueryResult` sentinel in `executor.go` and evaluate whether `query_runner_test.go` mock tests should be deleted or migrated.

### Resulting File Changes

| File | Action |
|---|---|
| `cmd/mcp-server/query_executor.go` | **Delete entirely** |
| `cmd/mcp-server/query_runner.go` | **Delete entirely** |
| `cmd/mcp-server/query_executor_test.go` | Migrate: `executor.cache` → `executor.Cache`; lowercase wrappers → exported methods; `ExpressionCache` → `querypkg.ExpressionCache` |
| `cmd/mcp-server/query_runner_test.go` | Two groups: (1) migrate or delete mock tests (`mockJQRunner`, `TestJQRunner_*`, `TestMockJQRunner*`) — evaluate whether coverage is needed at all; (2) **must migrate** functional tests (`TestQueryExecutor_RunQuery`, `TestQueryExecutor_RunQuery_InvalidFilter`, `TestQueryExecutor_RunQueryWithTimeRange`) — update `NewQueryExecutor` → `querypkg.NewQueryExecutor`, `parseTimeRange` → `querypkg.ParseTimeRange` |
| `cmd/mcp-server/handlers_query.go` | Migrate `executeQuery`/`executeQueryWithTimeRange` to delegate to `e.ToolExecutor.ExecuteQuery`/`ExecuteQueryWithTimeRange` |
| `cmd/mcp-server/executor.go` (line 106) | Evaluate: `var _ mcquery.QueryResult` sentinel — may be removed in a follow-on cleanup phase |
| `internal/mcp/query/query.go` | **No changes required** (Option B accessor is unnecessary) |

---

## Trade-off Analysis

### Keeping the wrappers (status quo)

**Pros**: No test churn. Existing passing tests remain unchanged.

**Cons**:
- ArchGuard will continue to flag duplicate types on every scan, creating noise.
- `cmd/mcp-server` retains a direct compile-time dependency on `internal/mcp/query` in 5 source files even though the intended boundary is `internal/mcp/executor`.
- The `cache *querypkg.ExpressionCache` field leaks an internal struct pointer into `package main`, a concrete coupling that blocks future abstraction.
- `handlers_query.go` duplicates the full query execution pipeline that already exists in `internal/mcp/executor`, meaning any bug fix or optimization in the canonical path must be applied twice.
- Adding any future abstraction layer under `internal/mcp/query` requires changing both the canonical type and the `cmd/mcp-server` wrappers.

### Removing wrappers (this proposal)

**Pros**:
- Eliminates the duplicate-type ArchGuard signal.
- Closes the `query_executor.go`/`query_runner.go` duplicate code completely.
- Eliminates the duplicated query execution path in `handlers_query.go`.
- Removes the concrete `*ExpressionCache` coupling from `package main`.
- Makes `cmd/mcp-server` strictly thinner, consistent with the long-term direction in `proposal-mcp-server-package-split.md`.

**Cons**:
- Test migration requires touching `query_executor_test.go` and `query_runner_test.go` with mechanical but non-trivial rename work.
- Diff will be non-trivial despite being a pure relocation with no behavioral change.
- The `cmd/mcp-server → internal/mcp/query` direct import is **not fully eliminated** by this proposal: `handlers_stage1.go`, `handlers_stage2.go`, and the sentinel in `executor.go` keep the import alive.

---

## Risks

**Test regression risk (low)**: The wrapper methods are pure delegations and the canonical `querypkg.QueryExecutor` methods are semantically identical. Moving tests to use the canonical type directly carries negligible logic risk, but requires careful line-by-line validation of every lowercase → uppercase rename.

**Test scope underestimation risk (medium — REVISED UPWARD)**: The original proposal identified only `query_executor_test.go` as requiring changes. In reality, `query_runner_test.go` also requires changes (it uses `JQRunner`, `parsedTimeRange`, and `QueryResult` from the aliases). If any convenience-tool test file in `cmd/mcp-server` references type aliases or lowercase wrapper methods, it must also be updated. A full grep pass before starting is required.

**Incomplete goal risk (medium — NEW)**: The original proposal states a goal of "Replace direct `internal/mcp/query` references in `cmd/mcp-server` with references through `internal/mcp/executor`." This is **not fully achievable** in a single phase without also migrating `handlers_stage1.go` and `handlers_stage2.go`, which is out of scope. The proposal should explicitly state that the direct import is only *partially* reduced, not eliminated.

**Partial removal risk (medium)**: If only the `JQRunner` duplicate is removed but the `QueryExecutor` wrapper is kept, or vice versa, the codebase ends up with an inconsistent half-state. Stages 1 and 2 must be completed atomically in a single commit.

**Import cycle risk (none)**: The proposed changes reduce import references. No new imports are introduced.

**`ExpressionCache.Len()` risk (none — REVISED)**: The original proposal warned about adding a `CacheLen()` accessor. Since we are using the existing `Cache` public field and `Get()` method directly, no new surface area is added to `internal/mcp/query`.

---

## Testing and Validation

1. Before starting: run `grep -rn "executor\.cache\|compileExpression\|buildExpression\|streamFiles\|processFile\|ExpressionCache\|parsedTimeRange\|QueryResult\|QueryRequest\|QueryResponse" cmd/mcp-server/ --include="*.go"` to produce a complete list of alias/wrapper references that must be updated.
2. `make commit` must pass after Stage 2 (file deletions + test migration) with no new test failures.
3. After Stage 2, run `go vet ./cmd/mcp-server/...` to confirm no missing method references.
4. After Stage 2, run `grep -r "JQRunner" cmd/` and confirm no remaining references.
5. After Stage 3, run `go build ./cmd/mcp-server/...` to confirm no unused import or compile errors in `handlers_query.go`.
6. Re-run ArchGuard after the final stage and confirm the `JQRunner` and `QueryExecutor` duplicate-type signals are resolved.

---

## Open Questions

1. **`query_runner_test.go` mock tests**: The `mockJQRunner` type and its tests (`TestJQRunner_InterfaceCompliance`, `TestMockJQRunner`, etc.) exist to verify that a mock satisfies the local `JQRunner` interface. Once the local interface is removed, these tests lose their purpose. Should they be deleted, or migrated to `internal/mcp/query` to test `querypkg.JQRunner` directly? The latter adds test coverage for the canonical interface, which currently has none in `internal/mcp/query`.

2. **`var _ mcquery.QueryResult` sentinel in `executor.go` (line 106)**: This is not a duplicate type — it is an import-guard. It keeps the `mcquery` import live after an earlier refactor removed direct uses of `mcquery` in `executor.go`. If all `mcquery` references are removed from `executor.go`, this sentinel should be deleted. If `executor.go` still needs `mcquery` types (e.g., for `toolPipelineConfig`), verify the sentinel is still necessary before removing it.

3. **Follow-on phase scope**: Fully closing the `cmd/mcp-server → internal/mcp/query` direct import requires migrating `handlers_stage1.go` (6 wrapper functions + 1 type alias) and `handlers_stage2.go` (1 wrapper function) behind `internal/mcp/executor`. This is feasible but constitutes additional surface area beyond this proposal. Should it be tracked as a distinct Phase 76 item?

4. **`parseTimeRange` free function in `query_executor.go` (line 39)**: This function (`func parseTimeRange(sinceStr, untilStr string) (parsedTimeRange, error)`) wraps `querypkg.ParseTimeRange`. It is called from `handlers_query.go` (comment on line 10 referencing it) and from `query_runner_test.go` line 118 (`TestQueryExecutor_RunQueryWithTimeRange`). After `query_executor.go` is deleted, all callers must be updated to call `querypkg.ParseTimeRange` directly. Note: `handlers_query.go` itself does not call `parseTimeRange` in production code — the comment on line 10 references it, but the actual call sites in `cmd/mcp-server` that must be updated are: `query_runner_test.go` line 118 only. `handlers_query.go` will have its `executeQuery`/`executeQueryWithTimeRange` bodies replaced entirely in Stage 3, eliminating any implicit dependency.

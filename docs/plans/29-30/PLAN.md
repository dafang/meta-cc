# Phase 29–30: Query Reliability Improvements — Implementation Plan

**Status**: Pending approval
**Date**: 2026-03-08
**Proposal**: [docs/proposals/query-reliability-improvements.md](../../proposals/query-reliability-improvements.md)

---

## Overview

This plan implements the four improvements described in the proposal:

| Problem | Description | Severity | Phase |
|---------|-------------|----------|-------|
| P1 | Unknown MCP parameters silently ignored; scope values not validated | Medium | 29 |
| P2 | Large session files silently skipped (buffer limit + swallowed errors) | High | 29 |
| P3 | Project path resolved from MCP server CWD; no `working_dir` parameter | High | 30 |
| P4 | No content length filtering in `query_user_messages` | Low | 30 |

**Development methodology**: TDD throughout. Each stage begins with failing tests; implementation follows to make tests pass.

**Code limits**: Phase ≤500 lines, Stage ≤200 lines.

---

## Phase 29: Query Reliability Foundation

**Goal**: Fix the two correctness issues (P2, P1) that cause silent query failures.

**Estimated code**: ~360–430 lines (tests + implementation)

**Budget note**: The ≤500 line per-phase limit is a guideline; these phases address High-severity correctness bugs and the cascading signature changes (Stages 29.2, 30.2) may push toward the ceiling. Each stage's contingency split (29.2a/b, 30.2a/b) provides an escape valve if implementation exceeds the per-stage 200-line limit.

### Stage Dependencies

```
29.1 (buffer constant + size)
  └─► 29.2 (error surfacing)        [same files, sequential]

29.3 (schema wiring)
  └─► 29.4 (dispatch validation)    [same files, sequential]

29.1 ──┐
       ├── can run in parallel
29.3 ──┘
```

Stages 29.1–29.2 and Stages 29.3–29.4 are independent tracks that can be developed concurrently. Within each track, stages are sequential.

---

### Stage 29.1 — Buffer Constant: Tests + Fix

**Problem addressed**: P2 (buffer limit 1MB < observed 1.8MB; duplicate constants in `query_executor.go` and `reader.go`)

**TDD sequence**:

1. Write a test in `query_executor_test.go` that feeds `processFile` a synthetic JSONL line of exactly 1.5 MB. Assert the test currently fails (error returned from scanner).
2. Write a test asserting that a 4 MB line is processed without error (expected to fail before the fix).
3. Define the shared constant in `internal/parser/` (e.g., as `const MaxScannerLineBytes = 4 * 1024 * 1024` in `reader.go` or a new file `internal/parser/constants.go`). Import this constant into `query_executor.go` to replace the local `const maxCapacity`. This direction of dependency (`cmd/mcp-server` importing from `internal/parser`) is consistent with existing patterns in the codebase where `handlers_query.go` already imports from `internal/locator`.
4. Replace the hardcoded `1024 * 1024` in `query_executor.go` and `2 * 1024 * 1024` in `reader.go` with the shared constant set to `4 * 1024 * 1024`.
5. Run tests; all should pass.

**Key files**: `query_executor.go`, `internal/parser/reader.go`, new constant file, `query_executor_test.go`

**Acceptance criteria**:
- `processFile` handles lines up to 4 MB without error.
- Both `query_executor.go` and `reader.go` reference the same named constant.
- `make commit` passes.

**Estimated lines**: ~60–80 (10 constant definition + 2 substitutions + 50–70 test lines)

---

### Stage 29.2 — Error Surfacing in `streamFiles`

**Problem addressed**: P2 (errors from `processFile` silently swallowed; callers cannot distinguish "no data" from "data existed but was unreadable")

**Depends on**: Stage 29.1 (same files)

**Decision: Use a `QueryResult` wrapper struct approach.** Rather than extending `QueryResponse` (which is used only internally in `query_executor.go`), introduce a `QueryResult` wrapper that carries both entries and warnings through the call chain: `type QueryResult struct { Entries []interface{}; Warnings []string }`. This wrapper flows from `streamFiles` → `executeQuery` → convenience handlers → `buildResponse` in `executor.go`. The `buildResponse` function extracts `Entries` for the main response body and appends `QueryResult.Warnings` as a top-level `warnings` field in the MCP response.

**TDD sequence**:

1. Write tests for `streamFiles` using a fixture directory containing one valid JSONL file and one file with a line that will trigger a read error (e.g., a file that cannot be opened, simulated by permissions, or a too-long line). Assert that:
   - Results from the valid file are returned in `QueryResult.Entries`.
   - `QueryResult.Warnings` is non-empty and identifies the skipped file path and error reason.
2. Update `streamFiles` to return `(QueryResult, error)` instead of `[]interface{}`. Accumulate file errors into `QueryResult.Warnings` rather than silently continuing; add a `slog.Warn` call for each skipped file.
3. Update `executeQuery` to return `(QueryResult, error)` and propagate the `QueryResult` from `streamFiles`.
4. Update all 10 convenience handlers to return `(QueryResult, error)` instead of `([]interface{}, error)`.
5. Update `buildResponse` in `executor.go` to accept `QueryResult` instead of `parsedData []interface{}`. `buildResponse` extracts `result.Entries` as `parsedData []interface{}` and passes it to the existing sub-methods (`buildStatsOnlyResponse`, `buildStatsFirstResponse`, `buildStandardResponse`) unchanged — those three sub-methods do NOT change their signatures and continue to accept `[]interface{}`. After getting the sub-method result, `buildResponse` wraps the final response to include a `warnings` field alongside the existing response fields before returning the serialized output. This means all three response paths (stats-only, stats-first, standard) carry warnings, and the `warnings` field is added at the `buildResponse` level — NOT inside `adaptResponse`.
6. Run all tests; confirm `make commit` passes.

**Key files**: `query_executor.go`, `handlers_convenience.go`, `executor.go`, `response_adapter.go` (if serialization changes needed), `query_executor_test.go`

**Acceptance criteria**:
- `streamFiles` with mixed valid/unreadable files returns partial results AND a `QueryResult.Warnings` list identifying skipped file paths and error reasons.
- `slog.Warn` is called for each skipped file.
- MCP response for any tool call includes a `warnings` field (empty array `[]` when no files were skipped; populated with file path + error strings when files were skipped).
- Callers can detect partial results by inspecting the `warnings` field.
- `make commit` passes.

**Estimated lines**: ~170–210 (signature changes: `streamFiles`, `executeQuery`, 10 handlers, `buildResponse` = ~50–60 lines; tests = ~120–150 lines). If implementation exceeds 200 lines, split into Stage 29.2a (`streamFiles` → `executeQuery` → 10 handler signature changes only, with basic tests) and Stage 29.2b (`buildResponse` warnings integration + full tests).

---

### Stage 29.3 — Schema Accessor for Dispatch Validation

**Problem addressed**: P1 (tool schemas in `tools.go` are documentation-only; dispatch layer has no accessor to retrieve per-tool schema at runtime)

**Architectural context**: The dispatch layer currently has no per-tool schema accessor. `getToolDefinitions()` returns all tool schemas as a flat list used only for the `tools/list` response. The key architectural change in this stage is introducing `getToolSchemaByName(name string) (ToolSchema, error)` — a function that the dispatch path can call to retrieve a specific tool's schema by name at runtime. The full validation logic (rejecting unknown parameters) is Stage 29.4; this stage only makes the schema accessible.

**Scope**: Schema validation applies only to query tools dispatched through the main switch statement in `ExecuteTool` (query_user_messages, query_tools, etc.). Special tools handled by `executeSpecialTool` (cleanup_temp_files, list_capabilities, get_capability, get_session_directory, inspect_session_files, execute_stage2_query, get_session_metadata) are out of scope for Stage 29.3 and 29.4 — they each perform their own parameter handling and are exempt from the central dispatch validation introduced here.

**TDD sequence**:

1. Implement `getToolSchemaByName(name string) (ToolSchema, error)` in `tools.go`. This is the deliverable of Stage 29.3: it builds an index of tool schemas from `getToolDefinitions()` and returns the schema for a named tool, or an error if the tool name is not found.
2. Write tests that call `getToolSchemaByName` directly and verify: it returns correct schema properties for known tool names (e.g., verifying that `query_user_messages` schema declares a `pattern` property); it returns an error for unknown tool names (e.g., `"query_nonexistent"`).
3. Wire a call to `getToolSchemaByName` into `ExecuteTool` in `executor.go` at the point **after** `executeSpecialTool` returns `handled=false` and **before** the main query tool switch statement. At this stage, the retrieved schema is not yet used for validation; the purpose is to confirm that schema retrieval is integrated into the dispatch path for query tools only.

   **Relationship to existing unknown-tool handling**: The current `ExecuteTool` switch already handles unknown tool names in a `default` case (`return fmt.Errorf("unknown tool %s in executor: %w", toolName, mcerrors.ErrUnknownTool)`). Stage 29.3 adds `getToolSchemaByName` BEFORE the switch as an independent validation layer — positioned after the `executeSpecialTool` check so special tools bypass it. If the schema lookup returns an error for an unknown query tool name, the switch `default` case becomes dead code and should be removed. The acceptance criterion "fails early for unknown tool names" means the schema check fires before the switch dispatch, not after.

4. Write a test that verifies the schema IS retrieved at dispatch time: a call with a valid tool name succeeds, and a call with a non-existent tool name returns an error from `getToolSchemaByName` before the switch statement is reached; the switch `default` case is removed as it becomes redundant.
5. Run tests; confirm `make commit` passes.

**Key files**: `tools.go`, `executor.go`, `executor_test.go`

**Acceptance criteria**:
- `getToolSchemaByName` is implemented and returns correct schema properties for known tools, and an error for unknown tool names.
- `ExecuteTool` calls `getToolSchemaByName` before the switch statement; a call with an unrecognized tool name returns an error from `getToolSchemaByName` before the switch is reached; the switch `default` case is removed as it becomes redundant.
- No existing tool tests broken.
- `make commit` passes.

**Estimated lines**: ~80–100 (30–40 accessor + wiring + 50–60 tests)

---

### Stage 29.4 — Dispatch Validation + Scope Value Check

**Problem addressed**: P1 (unknown parameter keys not rejected; invalid scope values silently default)

**Depends on**: Stage 29.3

**TDD sequence**:

0. **Before writing tests**: Audit all call sites of `ExecuteTool` in the codebase to confirm no internal callers pass parameters that are not declared in the tool's schema. If any internal call sites pass undeclared parameters (e.g., test helpers or internal wrappers), document them and add an allowlist mechanism before enabling strict validation — otherwise the validation will break internal callers before it ever rejects external ones.
1. Extend the tests from Stage 29.3:
   - Unknown key: expect error listing the unrecognized key and valid alternatives.
   - Valid key with correct value: expect success (regression).
   - `scope: "sessions"` (invalid): expect error listing `"project"` and `"session"` as valid values.
   - `scope: "session"` (valid): expect success.
2. Implement central unknown-key validation in the dispatch path using the schema accessor from Stage 29.3. Compare incoming `args` keys against the schema's `Properties` map; return a structured error for any unrecognized key.
3. Implement scope value validation in `ExecuteTool`, immediately after the `scope := determineScope(...)` call (line 174 in executor.go) and before the `executeSpecialTool` check. Check that scope is one of `{"project", "session"}`; return an informative error otherwise. Placing it here means all tools — both special tools and query tools — receive scope validation at dispatch time, without modifying `determineScope` itself (which should remain a pure extraction function).
4. Run full test suite; fix any regressions from tools that may be passing unexpected keys internally.
5. Confirm `make commit` passes.

**Key files**: `executor.go` or `server.go`, `handlers_query.go`, `executor_test.go`, `handlers_convenience_test.go`

**Acceptance criteria**:
- Calling any tool with an unrecognized key returns an explicit error (not empty results).
- Calling any tool with `scope: "sessions"` returns an explicit error naming valid values.
- All existing tool calls with correct parameters continue to work.
- `make commit` passes.

**Estimated lines**: ~100–120 (50–60 validation logic + 50–60 additional tests)

---

### Phase 29 Verification

After all four stages:

1. Run `make push` (full check: format + build + lint + tests + coverage).
2. Manual smoke test: call `query_user_messages` with `{"match": "foo"}` via MCP client; confirm error response naming `match` as unknown.
3. Manual smoke test: call `query_user_messages` with `{"scope": "sessions"}`; confirm error naming valid values.
4. Manual smoke test against a real archguard-scale session directory: confirm queries return results (previously returned empty due to oversized file skipping).
5. Confirm that queries returning partial results include warning information.

---

## Phase 30: Cross-Project Support + Enhancement

**Goal**: Add `working_dir` parameter (P3), content length filtering (P4), and synchronize documentation.

**Estimated code**: ~340–440 lines (tests + implementation + docs)

**Budget note**: The ≤500 line per-phase limit is a guideline; these phases address High-severity correctness bugs and the cascading signature changes (Stages 29.2, 30.2) may push toward the ceiling. Each stage's contingency split (29.2a/b, 30.2a/b) provides an escape valve if implementation exceeds the per-stage 200-line limit.

### Stage Dependencies

```
30.1 (getQueryBaseDir + executeQuery signatures)
  └─► 30.2 (10 handlers + 10 schemas)    [sequential, cascading changes]

30.3 (content length filtering)           [independent, can parallel with 30.1–30.2]

30.4 (documentation sync)                [independent, can parallel with all above]
```

Stages 30.3 and 30.4 are independent of the `working_dir` track and can proceed concurrently.

---

### Stage 30.1 — `working_dir`: Core Function Signatures

**Problem addressed**: P3 (core plumbing — `getQueryBaseDir` and `executeQuery` currently hardcode `os.Getwd()`)

**Note**: `scope` is extracted from `args` at the dispatch level in `executor.go` (`determineScope`) before `executeQuery` is called. Handlers already receive `scope` as a parameter. The `working_dir` extraction follows the same pattern: extracted in the handler from `args`, passed as a parameter to `executeQuery`. External callers of `executeQuery` (if any) must supply both `scope` and `workingDir`.

**TDD sequence**:

1. Write tests for `getQueryBaseDir` that:
   - Pass an explicit `workingDir` pointing to a test fixtures directory; assert the returned path uses the fixture directory.
   - Pass an empty `workingDir`; assert CWD fallback behavior (backward compatible).
2. Tests fail because the function does not accept `workingDir`.
3. Extend `getQueryBaseDir(scope string)` → `getQueryBaseDir(scope, workingDir string)`. When `workingDir` is non-empty, use it instead of `os.Getwd()`.
4. Update `executeQuery` signature to accept and thread `workingDir` through to `getQueryBaseDir`.
5. Update all existing call sites of `executeQuery` to pass `""` (empty = CWD fallback) — at this stage all 10 convenience handlers pass empty string; behavior is unchanged.
6. Run tests; confirm `make commit` passes.

**Key files**: `handlers_query.go`, `query_executor_test.go` or new `handlers_query_test.go`

**Acceptance criteria**:
- `getQueryBaseDir` and `executeQuery` accept `workingDir` parameter.
- Empty `workingDir` preserves existing CWD-based behavior.
- All existing tests pass unchanged.
- `make commit` passes.

**Estimated lines**: ~80–100 (30–40 signature changes + 50–60 tests)

---

### Stage 30.2 — `working_dir`: Handler + Schema Propagation

**Problem addressed**: P3 (expose `working_dir` in all 10 MCP tool schemas; extract and forward in all 10 handlers)

**Depends on**: Stage 30.1

**Scope clarification**: `working_dir` is added only to the 10 convenience query tools that call `executeQuery` (`query_user_messages` through `query_tool_blocks`). Other MCP tools that accept a `scope` parameter — specifically `get_session_directory` and `get_session_metadata` — are out of scope for this stage. Those tools use different dispatch paths (via `handleGetSessionDirectory` and `handleGetSessionMetadata` in `executor.go`) and would require separate investigation and work.

**TDD sequence**:

1. Write parameterized tests for each of the 10 convenience handlers, asserting that:
   - When `working_dir` is passed in `args`, it is forwarded to `executeQuery`.
   - When `working_dir` is absent, behavior is unchanged.
   - Integration test: invoke `query_user_messages` with `working_dir` set to a test fixtures directory; confirm only sessions from that directory are queried.
2. Update all 10 convenience handlers to extract `working_dir` via `getStringParam(args, "working_dir", "")` and pass it to `executeQuery`.
3. Update all 10 tool schema definitions in `tools.go` to declare `working_dir` as an optional string property with description: `"Override working directory for session lookup. Defaults to MCP server CWD."`.
4. Run full test suite; confirm all handlers behave correctly.
5. Confirm `make commit` passes.

**Key files**: `handlers_convenience.go`, `tools.go`, `handlers_convenience_test.go`

**Acceptance criteria**:
- All 10 convenience query tools accept `working_dir` without error.
- `working_dir` is forwarded to session directory resolution.
- Tool schemas declare `working_dir` (verifiable via `tools/list` response).
- Integration test with alternate directory passes.
- `make commit` passes.

**Estimated lines**: ~180–220 (70–90 handler + schema changes + 110–130 tests). If implementation exceeds 200 lines, split into Stage 30.2a (handler extraction, 10 handlers × ~3 lines each) and Stage 30.2b (schema updates, 10 schemas × ~5 lines each).

---

### Stage 30.3 — Content Length Filtering

**Problem addressed**: P4 (`query_user_messages` has no way to filter messages by character count, only truncate)

**Can run in parallel with** Stages 30.1–30.2.

**TDD sequence**:

1. Write tests for `handleQueryUserMessages` that:
   - Pass `{min_content_length: 10}`: assert generated jq filter includes a length lower-bound predicate.
   - Pass `{max_content_length: 100}`: assert filter includes an upper-bound predicate.
   - Pass both: assert both predicates are present.
   - Pass neither: assert filter is unchanged (regression).
   - Pass `{content_type: "string", min_content_length: 20}`: assert filter applies to string content length (character count).
   - Pass `{content_type: "array", min_content_length: 2}`: assert implementation returns an error (or the documented fallback behavior), not silent filtering. Content length filtering is meaningful for string content only — for array content, `| length` returns element count (1–3 for structured tool result messages), which is not a character count and is not a useful proxy for content length. The schema description for `min_content_length` and `max_content_length` must document this limitation explicitly.
2. Extend `handleQueryUserMessages` to extract `min_content_length` and `max_content_length` via `getIntParam`, and append jq length predicates to the filter when non-zero.
3. Add `min_content_length` and `max_content_length` to `query_user_messages` schema in `tools.go` with descriptions clarifying: (a) the distinction from `max_message_length` (truncation vs. filtering), and (b) that length filtering applies to string content only — specifying this in combination with `content_type: "array"` returns an error.
4. Write an integration test confirming that combined use of `max_message_length` (truncation) and `max_content_length` (filtering) works correctly and independently.
5. Confirm `make commit` passes.

**Key files**: `handlers_convenience.go`, `tools.go`, `handlers_convenience_test.go`

**Acceptance criteria**:
- `min_content_length` and `max_content_length` filter by content length before returning results (string content only).
- Specifying length filters together with `content_type: "array"` returns an error or documented behavior, not silent filtering.
- Schema descriptions document the string-only limitation and the distinction from `max_message_length` (truncation vs. filtering).
- `max_message_length` (truncation) continues to work independently and can be combined with length filters.
- `make commit` passes.

**Estimated lines**: ~90–110 (25–30 handler logic + 10–15 schema + 55–65 tests)

---

### Stage 30.4 — Documentation Sync

**Can run in parallel with** Stages 30.1–30.3.

**Documents to update**:

1. **`docs/guides/mcp.md`** (MCP reference): Add entries for:
   - New `working_dir` parameter (available on all 10 convenience tools).
   - New `min_content_length` / `max_content_length` parameters on `query_user_messages`.
   - Warning output format for partial results (P2 fix).
   - Note on parameter validation: unknown parameters now return errors.

2. **`docs/examples/mcp-query-cookbook.md`** (or equivalent): Add examples:
   - Querying a specific project's sessions via `working_dir`.
   - Filtering messages by length to remove template noise.
   - Detecting partial results from the warning field.

3. **`CLAUDE.md`** (FAQ section): Add/update:
   - Q: Why am I getting "unknown parameter" errors? → parameter validation is now strict.
   - Q: How do I query a different project's sessions? → use `working_dir`.
   - Q: What does the warning field in query results mean? → partial results due to unreadable files.

4. **`docs/proposals/query-reliability-improvements.md`**: Update `Status` from `Draft for Review` to `Implemented` and add implementation notes referencing the phase.

5. **`docs/core/plan.md`**: Add Phases 29–30 to the phase overview table and status section.

**Acceptance criteria**:
- All mentioned documents updated and consistent with implementation.
- No references to old behavior (e.g., silent parameter ignoring) without a note that behavior changed.
- `make commit` passes (no code changes, only docs).

**Estimated lines**: ~80–120 doc lines changed across multiple files

---

### Phase 30 Verification

After all stages complete:

1. Run `make push` (full suite).
2. Manual smoke test: call `query_user_messages` with `working_dir` pointing to a different project; confirm sessions from that project are returned.
3. Manual smoke test: call `query_user_messages` with `{min_content_length: 50, max_content_length: 300}`; confirm only messages within the range are returned.
4. Confirm `tools/list` MCP response includes `working_dir` on all 10 tools and `min/max_content_length` on `query_user_messages`.
5. Review updated documentation for consistency with Phase 29 behavior changes (parameter validation errors).

---

## Overall Acceptance Criteria

All of the following must be true before the implementation is considered complete:

| Criterion | Verification |
|-----------|-------------|
| Lines up to 4 MB processed without error | Unit test in `query_executor_test.go` |
| Buffer constant unified (single definition) | Compile-time: both files import same constant |
| Partial results include warning for skipped files | Unit test + MCP response inspection |
| Unknown parameters return explicit error | Unit test via `ExecuteTool` |
| Invalid scope values return explicit error | Unit test |
| `working_dir` accepted by all 10 tools | Schema inspection + integration test |
| `working_dir` overrides CWD for session resolution | Integration test with alternate directory |
| `min/max_content_length` filter by character count (string) | Unit test + integration test |
| `min/max_content_length` and `max_message_length` are independent | Combined-use integration test |
| All existing tool tests pass unchanged | `make push` |
| Test coverage ≥ 80% maintained | `make test-coverage` |
| Documentation updated and consistent | Manual review |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `QueryResult.Warnings` field (new) is not yet surfaced in existing callers that only read `Entries` | Low | Low | Additive approach: `warnings` is a new top-level field in MCP response; existing callers that ignore unknown fields are unaffected |
| Central parameter validation rejects parameters passed by internal code paths | Low | High | Audit all `ExecuteTool` call sites before implementing; add allowlist for internal-only calls if needed |
| Cascading signature changes in Stage 30.2 introduce regressions | Medium | Medium | Staged approach: Stage 30.1 establishes signatures + empty-string defaults first; Stage 30.2 wires actual values |
| `working_dir` with relative paths or symlinks behaves unexpectedly | Low | Low | Normalize to absolute path using `filepath.Abs` at extraction time |

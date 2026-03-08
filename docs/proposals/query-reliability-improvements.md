# Query Reliability Improvements Proposal

**Status**: Draft for Review
**Date**: 2026-03-08
**Author**: Claude Code Analysis

## Background

During cross-project usage of meta-cc (specifically in the archguard project), several categories of query reliability problems were identified. These issues cause silent failures—callers receive empty results with no indication that anything went wrong, making debugging extremely difficult.

The problems were discovered when attempting to use `query_user_messages` to search recent plan/proposal-related messages across a project's sessions.

---

## Problem 1: Unknown Parameters Silently Ignored

### Description

MCP tool handlers extract parameters from the `args` map using `getStringParam`, `getIntParam`, and `getBoolParam`. Any key not explicitly accessed by the handler is silently ignored. There is no validation that the caller's parameter set is a subset of the tool's declared schema.

The tool schemas defined in `tools.go` are documentation only: they are served to MCP clients via `tools/list` for self-description purposes, but they play no role at runtime. The dispatch path in `server.go` (`handleToolsCall` → `executor.ExecuteTool`) passes the raw `arguments` map directly to handlers with no validation step. There is no central dispatch layer that compares incoming parameter keys against the schema definitions.

### Impact

- A caller using `--match` instead of `--pattern` receives empty results with no error.
- Typos in parameter names are undetectable from the caller's side.
- Debugging requires reading source code to discover the correct parameter name.
- The same silent-failure applies to the `scope` parameter: any value other than `"project"` or `"session"` is silently treated as `"project"` (the default in `getStringParam`), meaning callers who misspell `"session"` as `"sessions"` silently get project-scope results. This is covered here because it has the same root cause: no validation at dispatch.

### Proposed Fix

Add a runtime validation layer in the dispatch path. Before invoking any handler, compare the keys present in `args` against the set of properties declared in that tool's schema. If any key is unrecognized, return an explicit error listing the unknown parameter and the valid options.

This validation must be implemented centrally—either in `handleToolsCall` or in `ExecuteTool`—so all handlers benefit automatically rather than requiring per-handler changes. This requires the dispatch layer to have access to the schema definitions at runtime (currently the schema definitions in `tools.go` are only used for the `tools/list` response), so the fix also needs to make the schema available to the validation logic.

For the `scope` parameter specifically, valid values (`"project"` and `"session"`) should be checked explicitly; unrecognized values should be rejected with an informative error rather than silently defaulting.

**Implementation dependency**: Enabling strict parameter validation will reject any parameter used by a handler but not declared in the tool's schema. A known instance is `content_type` in `query_user_messages`: the handler extracts it via `getStringParam(args, "content_type", "string")` but it is not declared in the schema in `tools.go`. This must be remediated (by adding `content_type` to the schema) before or alongside the validation rollout — see Problem 4.

**Complexity: Medium-to-High.** The validation logic itself is straightforward, but wiring it requires making the schema definitions available at dispatch time (currently they are only used to populate `tools/list`) and ensuring the validation works correctly for all parameter types including optional standard parameters shared across tools.

---

## Problem 2: Large Session Files Silently Skipped

### Description

`processFile` in the query executor uses a `bufio.Scanner` with a 1 MB per-line buffer cap. Claude Code session files can contain lines exceeding this limit (observed: a ~1.8 MB session file in archguard). When `scanner.Scan()` encounters a line exceeding the cap, it stops and returns `bufio.ErrTooLong`.

`processFile` correctly surfaces this error to its caller. However, `streamFiles`—which iterates over all session files—silently swallows the error:

```
// Log error but continue processing other files
continue
```

The comment acknowledges logging intent, but no log call exists. The result: any session file with an oversized line is entirely skipped, the caller gets partial or empty results, and no warning is emitted.

There is a secondary inconsistency: `reader.go` (the session parser used elsewhere) allocates a 2 MB buffer (`const maxCapacity = 2 * 1024 * 1024`), while `query_executor.go` allocates only 1 MB (`const maxCapacity = 1024 * 1024`). These magic constants are duplicated across two files. They should be unified into a shared named constant (e.g., in an `internal/` package or a shared file) to prevent the two values from drifting apart again.

### Impact

- Project-scope queries silently drop entire sessions containing long lines.
- The caller has no way to distinguish "no matching data" from "data existed but was unreadable."
- In real usage this manifested as completely empty query results despite matching data existing.

### Proposed Fix

Three changes:

1. **Increase buffer limit**: Raise `maxCapacity` in `query_executor.go` from 1 MB to at least 4 MB. The 2 MB limit in `reader.go` was introduced in an earlier phase and already covers the observed 1.8 MB case. The 4 MB recommendation for `query_executor.go` reflects: (1) the observed 1.8 MB peak; (2) 2× headroom for future growth; and (3) alignment with the existing `reader.go` practice of using 2 MB as baseline, while going higher to ensure the query path never becomes the bottleneck. Memory impact is negligible — 4 MB per open file, and files are not held simultaneously.

2. **DRY the buffer constant**: Replace the duplicated magic number in both `query_executor.go` and `reader.go` with a shared named constant to ensure the two components stay aligned. The constant should be defined in `internal/parser/` (e.g., in `reader.go` or a new `internal/parser/constants.go`) and imported into `query_executor.go`. This direction of dependency (`cmd/mcp-server` importing from `internal/parser`) is consistent with existing patterns in the codebase.

3. **Surface errors to callers**: When `processFile` returns an error, instead of silently continuing, accumulate the error information and include it in the tool's response. The current `QueryResponse` struct only has `Entries []interface{}`; surfacing per-file errors requires extending the response format. The chosen approach is to introduce a `QueryResult` wrapper struct returned by `executeQuery`: `type QueryResult struct { Entries []interface{}; Warnings []string }`. `QueryResult.Warnings` is populated by `streamFiles` when files are skipped, flows through `executeQuery` and the convenience handlers, and is appended to the final output by `buildResponse` in `executor.go` as a top-level `warnings` field in the MCP response (an empty array when no files were skipped). The `warnings` field is added at the `buildResponse` level — not inside `adaptResponse` — so that all three response paths (stats-only, stats-first, standard) carry warnings consistently. This additive approach avoids disrupting existing callers who only inspect the data entries.

---

## Problem 3: Project Path Resolved from MCP Server CWD

### Description

`getQueryBaseDir` resolves the project path by calling `os.Getwd()`. When meta-cc runs as a long-lived MCP server process (the standard deployment), its working directory is set at startup—typically the directory where the IDE or shell launched the server, not the project the user is currently working in.

None of the 10 convenience query tools (`query_user_messages`, `query_tools`, `query_tool_errors`, etc.) expose a `working_dir` or `project_path` parameter in their schemas. A caller cannot specify which project's sessions to query.

This means:
- Project-scope queries may resolve to the wrong session directory.
- There is no way to query sessions from a specific project without restarting the MCP server in that project's directory.
- The `--project` flag pattern that users naturally attempt has no effect (see Problem 1).

### Impact

- Cross-project meta-cc usage (e.g., using meta-cc installed in one project to analyze another) is effectively impossible.
- CWD-dependent resolution is fragile across IDE environments and launch configurations.

### Proposed Fix

Add an optional `working_dir` parameter to all query tools that accept a `scope` parameter. When provided, `getQueryBaseDir` uses this path instead of `os.Getwd()`. When absent, the current behavior (CWD fallback) is preserved for backward compatibility.

**Actual scope of changes required**:

- `getQueryBaseDir(scope string)` currently takes only `scope`. Its signature must be extended to accept `workingDir string` (or an options struct).
- `executeQuery(scope string, jqFilter string, limit int)` calls `getQueryBaseDir` directly; its signature must be updated to thread `workingDir` through.
- All 10 convenience handlers (`handleQueryUserMessages`, `handleQueryTools`, `handleQueryToolErrors`, `handleQueryTokenUsage`, `handleQueryConversationFlow`, `handleQuerySystemErrors`, `handleQueryFileSnapshots`, `handleQueryTimestamps`, `handleQuerySummaries`, `handleQueryToolBlocks`) call `executeQuery` and must each extract the new parameter from `args` and forward it.
- All 10 tool schemas in `tools.go` must be updated to declare the new `working_dir` property.

The `SessionLocator` already supports arbitrary paths via `LocateOptions.ProjectPath` and `AllSessionsFromProject(projectPath string)`; the fix bridges the MCP layer to that existing capability without changing the locator itself.

**Note**: Other MCP tools that accept a `scope` parameter — specifically `get_session_directory` and `get_session_metadata` — are not in scope for this change. Those tools use different dispatch paths in `executor.go` (`handleGetSessionDirectory` and `handleGetSessionMetadata`) and would require separate investigation to determine the right extension point.

**Interaction between `working_dir` and `scope`**: When both are provided, `working_dir` controls which project's session directory is used, and `scope` controls whether to query a single session or all sessions within that directory. The combination is valid and well-defined: `scope: "session"` with `working_dir` returns results from the most recently modified session file in the specified project directory; `scope: "project"` with `working_dir` queries all sessions in that directory. When `working_dir` is absent, both scope values fall back to using CWD as today.

**Complexity: Medium-to-High.** The logic at each point is simple, but the change is a cascading signature update across approximately 13 functions and all 10 tool schema definitions.

---

## Problem 4: No Content Length Filtering

### Description

This is a usability gap rather than a correctness bug. User messages in Claude Code sessions span from one-line commands to multi-kilobyte prompts (e.g., skill invocation templates that start with `λ(intent)`). When searching for intent-revealing messages, short operator commands and long template wrappers both match, creating high noise in results.

There is currently no way to filter by message content length.

The existing `max_message_length` parameter in `query_user_messages` performs **truncation** of long messages (cutting content to N characters before returning it). This is distinct from **filtering**: truncation changes what is returned for matching messages; filtering excludes messages that fall outside a length range entirely.

### Impact

- Callers must post-process results externally (e.g., via Python scripts) to exclude noise.
- Common filtering patterns (e.g., "show me messages between 20 and 500 characters") require raw jq expressions, bypassing convenience tools.

### Proposed Fix

Add optional `min_content_length` and `max_content_length` integer parameters to `query_user_messages`. When provided, add a jq predicate filtering on content length to the generated filter. These parameters should default to `0` (no limit) to preserve existing behavior.

The names `min_content_length` / `max_content_length` are chosen to avoid confusion with the existing `max_message_length` parameter:
- `max_message_length` (existing): truncation—limits the number of characters returned in each message's content field.
- `min_content_length` / `max_content_length` (proposed): filtering—excludes messages whose content length falls outside the specified range before returning results.

Using `min_length` / `max_length` (as in an earlier draft) would conflict with the established naming pattern around `max_message_length` and create ambiguity about which operation is being performed.

**Content type consideration**: `query_user_messages` already distinguishes string content (`content_type: "string"`) from array content (`content_type: "array"`). The `| length` jq operator has different semantics for each: for strings it returns character count; for arrays it returns element count (typically 1–3 for structured tool result messages, not a character count). Length filtering is therefore most meaningful for string content. The implementation should document this limitation; if array content filtering is needed, a separate mechanism (e.g., summing text field lengths within the array) would be required.

**Schema gap**: The `content_type` parameter is already used by the handler (`getStringParam(args, "content_type", "string")` in `handleQueryUserMessages`) but is not declared in the `query_user_messages` tool schema in `tools.go`. Once Problem 1's parameter validation is implemented (rejecting undeclared parameters), `content_type` will be rejected as unknown. The implementation of Problem 4 must add `content_type` to the schema alongside `min_content_length` and `max_content_length`.

---

## Validation Approach

Each fix can be validated independently:

**Problem 1 (unknown parameters and scope validation)**:
- Unit test: pass `{match: "foo"}` to `handleQueryUserMessages`; expect an error response naming `match` as unknown.
- Unit test: pass `{scope: "sessions"}` (invalid value); expect an error response listing valid values.
- Regression test: pass `{pattern: "foo"}` and `{scope: "session"}` and confirm they still work correctly.

**Problem 2 (large file handling)**:
- Create a synthetic JSONL test fixture containing one line exceeding 1 MB.
- Unit test `processFile`: confirm it returns a wrapped error (not a panic or empty result).
- Unit test `streamFiles` with mixed valid/oversized files: confirm results contain data from valid files and the response includes a warning indicator (via the chosen surfacing mechanism) for the oversized file.
- Increase buffer size test: confirm lines up to 4 MB are processed without error.
- Constant sharing test: confirm that the shared constant is used in both `query_executor.go` and `reader.go` (compile-time check).

**Problem 3 (CWD-dependent resolution)**:
- Unit test `getQueryBaseDir` with an explicit `working_dir` argument pointing to a test fixtures directory; confirm it uses that path rather than CWD.
- Integration test: invoke `query_user_messages` with `working_dir` set to an alternate project directory; confirm sessions from that directory are queried.
- Test that all 10 convenience tools extract and forward `working_dir` correctly.

**Problem 4 (content length filtering)**:
- Unit test: pass `{min_content_length: 10, max_content_length: 100}` and confirm the generated jq filter includes length predicates.
- Integration test with fixture data: confirm only messages within the specified length range are returned.
- Test that `max_message_length` (truncation) and `max_content_length` (filtering) operate independently and can be combined.

---

## Non-Goals

- This proposal does not address CLI flag parsing (meta-cc's primary interface is MCP, not CLI).
- This proposal does not redesign the session locator or introduce multi-project session aggregation.
- No changes to the JSONL schema or session file format.
- No changes to the query tool output format beyond the addition of optional warning/error fields for Problem 2.

---

## Summary

| # | Problem | Severity | Fix Complexity |
|---|---------|----------|----------------|
| 1 | Unknown parameters silently ignored; scope values not validated | Medium | Medium-to-High |
| 2 | Large files silently skipped (bufio limit + swallowed errors + duplicated constants + no response error channel) | High | Medium |
| 3 | Project path from MCP server CWD; no working_dir parameter (cascading changes across 13 functions + 10 schemas) | High | Medium-to-High |
| 4 | No content length filtering (distinct from existing truncation) | Low | Low |

Problems 2 and 3 are the root cause of the complete query failures observed in practice. Problem 1 compounds the difficulty of diagnosing those failures. Problem 4 is a quality-of-life improvement for the common use case of filtering conversational user messages from template noise.

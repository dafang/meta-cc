# Proposal: Split cmd/mcp-server God Package

## Problem

`cmd/mcp-server` contains 43 source files in a single `package main`, totaling ~15,500 lines.
It violates the Single Responsibility Principle by combining six distinct concerns:

1. **MCP protocol dispatch** (`server.go`, `main.go`) — JSON-RPC parsing, method routing
2. **Tool registry and schema** (`tools.go`) — tool definitions, schema validation
3. **Executor orchestration** (`executor.go`) — per-tool dispatch, metric recording, pipeline config
4. **Query execution** (`query_executor.go`, `handlers_query.go`, `handlers_convenience.go`, `handlers_stage1.go`, `handlers_stage2.go`) — jq compilation, streaming, session directory resolution
5. **Output / file-ref infrastructure** (`output_mode.go`, `response_adapter.go`, `file_reference.go`, `temp_file_manager.go`, `filters.go`) — inline vs. file_ref selection, JSONL writing, content filtering
6. **Capabilities** (`capabilities.go`) — package download, cache, frontmatter parsing
7. **Observability** (`tracing.go`, `metrics.go`, `logging.go`) — OTel tracing, Prometheus metrics, structured logging

Being a `package main` makes all of these untestable as exported packages and prevents reuse elsewhere in the codebase.

## Goals

- Move each concern into its own `internal/` package with a clear API
- Reduce `cmd/mcp-server/` to only `main.go` and `server.go` (startup, signal handling, JSON-RPC dispatch)
- Maintain 100% behavioural compatibility — no change to MCP protocol, tool names, or output formats
- Preserve test coverage (all existing tests continue to pass, adapted to the new package paths)
- Each new package is independently unit-testable

## Non-Goals

- Changing the MCP protocol or wire format
- Adding new tools or capabilities
- Changing the public interface seen by Claude Code clients
- Splitting `internal/query` (already well-structured)
- Changing the Go module path or binary name

## Proposed Target Architecture

```
cmd/mcp-server/
  main.go      — startup: load config, init observability, run signal handler
  server.go    — JSON-RPC read loop + method dispatch (handleRequest, writeResponse, writeError)

internal/
  mcptools/    — tool definitions (Tool, ToolSchema, Property types; getToolDefinitions; schema index)
  executor/    — ToolExecutor: per-tool dispatch, pipeline config, jq helpers, validation
  mcpquery/    — QueryExecutor, ExpressionCache, QueryResult, getQueryBaseDir, getJSONLFiles
  fileref/     — OutputModeConfig, adaptResponse, FileReference, TempFileManager, filters
  capabilities/ — CapabilitySource, CapabilityIndex, load/cache/fetch, session cache dir
  observability/ — InitTracing, InitLogger, metrics registration and recording functions
```

### Package Responsibilities

#### `internal/mcptools`
Exports: `Tool`, `ToolSchema`, `Property`, `GetToolDefinitions() []Tool`, `GetToolSchemaByName(name string) (ToolSchema, error)`.
Corresponds to the current `tools.go`.
No external dependencies beyond the standard library and `internal/config`.

#### `internal/executor`
Exports: `ToolExecutor`, `NewToolExecutor()`, `(*ToolExecutor).ExecuteTool(cfg, toolName, args) (string, error)`.
Contains: `toolPipelineConfig`, `newToolPipelineConfig`, helper functions `getStringParam`, `getBoolParam`, `getIntParam`, `getFloatParam`, `validateArgKeys`, `determineScope`, `injectWarnings`, `buildResponse` and its variants.
Depends on: `internal/config`, `internal/errors`, `internal/query` (for `GenerateStats`), `internal/mcptools`, `internal/mcpquery`, `internal/fileref`, `internal/capabilities`, `internal/observability`.

#### `internal/mcpquery`
Exports: `QueryExecutor`, `QueryResult`, `NewQueryExecutor(baseDir string) *QueryExecutor`, `(*QueryExecutor).ExecuteQuery(scope, jqFilter string, limit int, workingDir string) (QueryResult, error)`, `GetQueryBaseDir(scope, workingDir string) (string, error)`, `GetJSONLFiles(dir string) ([]string, error)`.
Contains: `ExpressionCache`, `compileExpression`, `streamFiles`, `getJSONLFiles`, `getQueryBaseDir`.
Depends on: `internal/locator`, `internal/parser`, `github.com/itchyny/gojq`.

#### `internal/fileref`
Exports: `FileReference`, `OutputModeConfig`, `DefaultOutputModeConfig()`, `AdaptResponse(cfg, data, params, toolName) (interface{}, error)`, `TruncateMessageContent(messages []interface{}, maxLen int) []interface{}`, `ApplyContentSummary(messages []interface{}) []interface{}`.
Contains: `calculateOutputSize`, `selectOutputMode`, `generateFileReference`, `writeJSONLFile`, `cleanupOldFiles`, `executeCleanupTool`, `extractFields`, `generateSummary`.
Depends on: `internal/config`, `internal/errors`.

#### `internal/capabilities`
Exports: `CapabilitySource`, `CapabilityMetadata`, `CapabilityIndex`, `ExecuteListCapabilitiesTool(cfg, args) (string, error)`, `ExecuteGetCapabilityTool(cfg, args) (string, error)`, `CleanupSessionCache() error`.
Contains: all of `capabilities.go` logic (parse, load, cache, GitHub fetch, package download/extract).
Depends on: `internal/config`, `internal/errors`.

#### `internal/observability`
Exports: `InitTracing() (func(), error)`, `GetTracer() trace.Tracer`, `GetTraceID(ctx) string`, `GetSpanID(ctx) string`, `InitLogger(cfg)`, `NewRequestLogger(toolName) (*slog.Logger, string)`, `ClassifyError(err) string`, metric recording functions (`RecordRequest`, `RecordToolCall`, `RecordError`, etc.), `GetErrorSeverity(errorType) string`, `ClassifyResourceError(err) string`, `ClassifyTimeoutError(err) string`, `StartResourceMonitoring(interval)`.
Depends on: `internal/config`, `github.com/prometheus/client_golang/prometheus`, OpenTelemetry packages.

## Key Design Decisions

### Why `internal/mcptools` not `internal/mcp/tools`?
Flat internal packages are simpler and consistent with the existing `internal/` structure (`internal/query`, `internal/locator`, etc.). A nested `internal/mcp/` directory would require all sub-packages to import each other which creates circular dependencies.

### Why keep `server.go` in `cmd/mcp-server`?
`server.go` contains the JSON-RPC types (`JSONRPCRequest`, `JSONRPCResponse`) and the dispatch loop that is inherently tied to the binary's stdin/stdout protocol. It is only 292 lines and has no reuse value outside the binary. Moving it to an `internal/` package would gain nothing.

### Handlers remain in `cmd/mcp-server` initially
`handlers_stage1.go`, `handlers_stage2.go`, `handlers_convenience.go` are thin dispatch functions that call into `internal/mcpquery` and `internal/executor`. They can stay in `cmd/mcp-server` as they contain no reusable logic and their move would be a follow-up refactor.

### `internal/observability` Prometheus init() side effect
`metrics.go` uses `init()` to register Prometheus metrics. Moving it to `internal/observability` means the init() runs when the package is imported, which is the existing behaviour. No change required.

### Circular dependency prevention
The dependency graph is strictly layered:
```
cmd/mcp-server → internal/executor → internal/mcptools
                                   → internal/mcpquery → internal/locator
                                                        → internal/parser
                                   → internal/fileref → internal/config
                                   → internal/capabilities
                                   → internal/observability
```
No internal package imports `cmd/mcp-server` or another package at the same level that would create a cycle.

## Migration Approach

### Move strategy: copy-then-delete
1. Create new `internal/X` package
2. Copy existing code with package name changed and exported identifiers promoted
3. Update `cmd/mcp-server` files to import `internal/X` and use the exported API
4. Run `make commit` to validate
5. Delete the now-redundant code from `cmd/mcp-server`

### Phase ordering (each phase ≤500 lines)
The migration is broken into phases that can each be independently validated:

- **Phase 31** — `internal/observability` (tracing + logging + metrics, ~400 source lines)
- **Phase 32** — `internal/mcptools` (tool schema registry, ~427 source lines)
- **Phase 33** — `internal/fileref` (output mode + file ref + temp file + filters, ~430 source lines)
- **Phase 34** — `internal/mcpquery` (query executor + base dir resolution, ~500 source lines)
- **Phase 35** — `internal/capabilities` (capability loading/caching, ~1171 source lines → split across stages)
- **Phase 36** — `internal/executor` (tool executor orchestration, ~523 source lines)
- **Phase 37** — Clean up `cmd/mcp-server`, remove scaffolding, validate final state

## Alternatives Considered

### Alternative A: Leave code in `cmd/mcp-server` but add sub-packages
Go does not support sub-packages in `package main`. All files in `cmd/mcp-server/` must belong to `package main`. So sub-packages are not possible without moving to `internal/`.

### Alternative B: `pkg/` instead of `internal/`
Using `pkg/` would make packages importable by external consumers. These packages contain MCP server internals that are not part of any public API. `internal/` is the correct choice.

### Alternative C: Single large `internal/mcp` package
This would be a smaller improvement — still a god package, just named differently. Individual responsibility packages are the correct solution.

## Open Questions

1. Should `handlers_stage1.go`, `handlers_stage2.go`, and `handlers_convenience.go` be moved to `internal/executor` in Phase 36, or remain in `cmd/mcp-server` long-term? The proposal defers this to Phase 36 review.
2. The `sessionCapabilityCache` global in `capabilities.go` uses package-level globals with a mutex. Should this become an injected dependency rather than a singleton? Deferred to Phase 35 design.
3. Should `internal/mcptools` and `internal/executor` be renamed to `internal/tools` and `internal/toolexec` to avoid the `mcp` prefix given they are MCP-specific anyway? Both naming conventions are acceptable; `mcptools` and `mcpquery` are chosen to prevent collision with `internal/query`.

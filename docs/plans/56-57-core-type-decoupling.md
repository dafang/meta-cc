# Plan 56–57: Core Type Decoupling

**Status**: Planned
**Proposal**: [docs/proposals/proposal-core-type-decoupling.md](../proposals/proposal-core-type-decoupling.md)

---

## Overview

Two phases addressing the two structural violations identified by archguard:

| Phase | Scope | Key deliverable |
|---|---|---|
| 56 | Move domain types from `internal/parser` to `internal/types` | Dependency inversion resolved; `internal/types` is true domain layer |
| 57 | Create `internal/analysis` service; update `cmd/mcp-server` | Entry point no longer imports `internal/parser` or `internal/analyzer` |

**Phase dependencies**: Phase 57 depends on Phase 56 (types must be in `internal/types` before `internal/analysis` can use them).

---

## Phase 56: Move Core Domain Types

**Goal**: `internal/types` owns all shared domain types; `internal/parser` imports `internal/types` (not vice versa).

### Stage 56.1: Create type definitions in `internal/types`

**TDD**: Write/update tests in `internal/types/` first.

Files to create:
- `internal/types/session.go` — `SessionEntry`, `Message`, `ContentBlock`, `ToolUse`, `ToolResult` (with all custom JSON methods)
- `internal/types/toolcall.go` — `ToolCall`, `ExtractToolCalls()`
- `internal/types/constants.go` — `MaxScannerLineBytes`

Test coverage: add `internal/types/session_test.go`, `internal/types/toolcall_test.go` verifying JSON serialization round-trips.

Estimated: ~150 lines (type definitions + tests)

### Stage 56.2: Update `internal/parser` to depend on `internal/types`

- `internal/parser/types.go` → replace with `internal/parser/aliases.go`:
  ```go
  package parser
  import "github.com/yaleh/meta-cc/internal/types"
  type SessionEntry = types.SessionEntry
  type Message = types.Message
  type ContentBlock = types.ContentBlock
  type ToolUse = types.ToolUse
  type ToolResult = types.ToolResult
  type ToolCall = types.ToolCall
  const MaxScannerLineBytes = types.MaxScannerLineBytes
  ```
- `internal/parser/tools.go` → replace body with alias: `var ExtractToolCalls = types.ExtractToolCalls`
- `internal/parser/reader.go` — add `import "github.com/yaleh/meta-cc/internal/types"`, no API changes
- `internal/parser/constants.go` — delete (constant moved to types)

Run `make dev` to verify compilation.

Estimated: ~30 lines

### Stage 56.3: Update all consumers to import `internal/types` instead of `internal/parser` for types

**No logic changes** — only import path updates for type references.

Files to update (type reference changes only):

| Package | File | Change |
|---|---|---|
| `internal/types` | `loader.go` | Remove `import "internal/parser"`; types now in same package |
| `internal/analyzer` | all `*.go` | `parser.SessionEntry` → `types.SessionEntry`, `parser.ToolCall` → `types.ToolCall` |
| `internal/filter` | `pagination.go` | `parser.ToolCall` → `types.ToolCall` |
| `internal/stats` | all `*.go` | `parser.SessionEntry` → `types.SessionEntry`, `parser.ToolCall` → `types.ToolCall` |
| `internal/query` | all `*.go` | `parser.SessionEntry` → `types.SessionEntry`, `parser.ToolCall` → `types.ToolCall` |
| `internal/query/resources` | all `*.go` | same |
| `pkg/output` | all `*.go` | same |
| `pkg/pipeline` | `session.go` | `[]parser.SessionEntry` → `[]types.SessionEntry`, `parser.ToolCall` → `types.ToolCall` |
| `cmd/mcp-server` | `query_executor.go` | `parser.MaxScannerLineBytes` → `types.MaxScannerLineBytes` |
| `cmd/mcp-server` | `handlers_analysis.go` | `parser.SessionEntry` → `types.SessionEntry`, `parser.ToolCall` → `types.ToolCall` |

Run `make commit` after this stage.

Estimated: ~80 lines of import/type reference changes

**Phase 56 validation**: `make commit` passes; `internal/types` no longer imports `internal/parser`.

---

## Phase 57: Create `internal/analysis` Service Package

**Goal**: `cmd/mcp-server` no longer imports `internal/parser` or `internal/analyzer`.

### Stage 57.1: Create `internal/analysis` package with tests

**TDD first**: Create `internal/analysis/service_test.go` with table-driven tests for each method (mock or integration using test fixtures).

Files to create:
- `internal/analysis/service.go` — `Service` struct + 6 public methods wrapping `loadEntriesAndToolCalls` + `analyzer.*` calls
- `internal/analysis/service_test.go` — tests

The `loadEntriesAndToolCalls` helper moves from `cmd/mcp-server/handlers_analysis.go` to `internal/analysis/service.go` (unexported `loadData` method on `Service`).

```go
package analysis

import (
    "github.com/yaleh/meta-cc/internal/analyzer"
    "github.com/yaleh/meta-cc/internal/config"
    "github.com/yaleh/meta-cc/internal/locator"
    "github.com/yaleh/meta-cc/internal/parser"
    "github.com/yaleh/meta-cc/internal/types"
)

type Service struct{ cfg *config.Config }
func New(cfg *config.Config) *Service
func (s *Service) loadData(args map[string]interface{}) ([]types.SessionEntry, []types.ToolCall, error)
func (s *Service) AnalyzeBugs(args map[string]interface{}) (string, error)
// ... 5 more methods
```

Estimated: ~130 lines

### Stage 57.2: Update `cmd/mcp-server/handlers_analysis.go`

- Replace `loadEntriesAndToolCalls` + direct analyzer calls with `analysis.New(cfg).AnalyzeBugs(args)` etc.
- Remove imports: `internal/parser`, `internal/analyzer`
- Add import: `internal/analysis`

Estimated: ~40 lines

Run `make commit`.

**Phase 57 validation**: `make commit` passes; `cmd/mcp-server` no longer imports `internal/parser` or `internal/analyzer`.

---

## Parallel Execution Strategy

Stages within each phase are sequential (dependency chain). Phases themselves are sequential (57 depends on 56).

```
Stage 56.1  →  Stage 56.2  →  Stage 56.3
                                   ↓
                             Stage 57.1  →  Stage 57.2
```

No worktrees needed (sequential single-branch execution).

---

## Total Estimates

| Phase | Stages | Estimated LOC |
|---|---|---|
| 56 | 56.1, 56.2, 56.3 | ~260 lines |
| 57 | 57.1, 57.2 | ~170 lines |
| **Total** | 5 stages | **~430 lines** |

Within phase limit (≤500) and per-stage limit (≤200).

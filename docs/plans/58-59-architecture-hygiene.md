# Plan 58–59: Architecture Hygiene

**Status**: Completed 2026-03-10
**Proposal**: [docs/proposals/proposal-architecture-hygiene.md](../proposals/proposal-architecture-hygiene.md)

---

## Overview

Two phases addressing five structural issues identified by archguard analysis:

| Phase | Scope | Key deliverable |
|---|---|---|
| 58 | Facade completion + file inspector extraction + experiments isolation | `cmd/mcp-server` has zero `internal/parser` imports |
| 59 | Remove unused `cfg` from analysis.Service + add AnalysisService interface | `analysis.Service` is zero-dependency; handlers testable via interface |

**Phase dependencies**: Phase 59 depends on Phase 58 (no blocking dep, but sequential for safety).

---

## Phase 58: Facade Completion and Hygiene

**Goal**: Eliminate all remaining structural coupling violations with minimal blast radius.

### Stage 58.1: Fix `query_executor.go` parser import

**Change**: Replace `parser.MaxScannerLineBytes` with `types.MaxScannerLineBytes` in
`cmd/mcp-server/query_executor.go`. Remove the `internal/parser` import line.

Files:
- `cmd/mcp-server/query_executor.go` — change 1 import + 4 constant references

```go
// before
import "github.com/yaleh/meta-cc/internal/parser"
buf := make([]byte, parser.MaxScannerLineBytes)

// after
import "github.com/yaleh/meta-cc/internal/types"
buf := make([]byte, types.MaxScannerLineBytes)
```

Run `make dev` to verify compilation.

Estimated: ~10 lines

### Stage 58.2: Rename `experiments/` → `_experiments/`

Go build tools ignore directories starting with `_`. Rename the directory; update the single
reference in `scripts/ci/check-changelog-updated.sh`.

Files:
- `mv experiments/ _experiments/` (shell rename)
- `scripts/ci/check-changelog-updated.sh` — update path string `^experiments/` → `^_experiments/`

Run `make dev` to verify `go build ./...` still compiles cleanly.

Estimated: ~2 lines

### Stage 58.3: Extract `file_inspector.go` to `internal/query/files`

`file_inspector.go` has zero internal imports (pure stdlib). Extract it to a new subpackage
consistent with the existing `internal/query/resources` pattern.

Files to create:
- `internal/query/files/file_inspector.go` — copy content, change `package query` → `package files`
- `internal/query/files/file_inspector_test.go` — copy from `internal/query/file_inspector_test.go`

Files to update:
- `cmd/mcp-server/handlers_stage1.go` — add `internal/query/files` import for `InspectFiles`;
  keep existing `internal/query` import (still needed for `LoadTemplates`, `QueryTemplate`)
- `internal/query/file_inspector.go` — delete (replaced by subpackage)
- `internal/query/file_inspector_test.go` — delete (replaced by subpackage test)

Verify callers: `grep -rn "query\.InspectFiles\|query\.InspectionResult\|query\.FileMetadata\|query\.RecordSample" cmd/`

Run `make commit` after this stage.

Estimated: ~50 lines (move + import updates)

**Phase 58 validation**: `make commit` passes; `cmd/mcp-server` has zero `internal/parser` imports.

---

## Phase 59: Interface Extraction and Dead Field Removal

**Goal**: `analysis.Service` has no unused dependencies; `handlers_analysis.go` is independently testable.

### Stage 59.1: Remove unused `cfg` from `analysis.Service`

Audit confirms `Service.cfg` is stored but never read by any method.

Files:
- `internal/analysis/service.go` — remove `cfg *config.Config` field; change `New(cfg *config.Config)` to `New()`
- `internal/analysis/service_test.go` — update any `analysis.New(cfg)` calls to `analysis.New()`
- `cmd/mcp-server/handlers_analysis.go` — remove `cfg *config.Config` parameter from all `execute*Tool` functions; change `analysis.New(cfg)` to `analysis.New()`
- `cmd/mcp-server/executor.go` — update calls from `executeAnalyzeBugsTool(cfg, args)` to `executeAnalyzeBugsTool(args)` (and similarly for the 5 other analysis tool functions)

```go
// before
type Service struct{ cfg *config.Config }
func New(cfg *config.Config) *Service { return &Service{cfg: cfg} }

// after
type Service struct{}
func New() *Service { return &Service{} }
```

Run `make dev`.

Estimated: ~30 lines

### Stage 59.2: Add `AnalysisService` interface to `internal/analysis`

Add the interface in `internal/analysis/service.go`. Update `cmd/mcp-server/executor.go` to store
an `analysis.AnalysisService` field on `ToolExecutor` and inject it at construction, so that
`executeSpecialTool` uses the injected service rather than constructing a new one per call.

**TDD**: Add interface compliance assertion; existing service_test.go tests serve as integration coverage.

Files:
- `internal/analysis/service.go` — append `AnalysisService` interface after `Service` type
- `internal/analysis/service_test.go` — add: `var _ AnalysisService = (*Service)(nil)`
- `cmd/mcp-server/executor.go` — add `analysisSvc analysis.AnalysisService` field to `ToolExecutor`;
  initialise in `NewToolExecutor` (or wherever the executor is constructed) with `analysis.New()`;
  replace `executeAnalyzeBugsTool(args)` etc. calls with `e.analysisSvc.AnalyzeBugs(args)` etc.
- `cmd/mcp-server/handlers_analysis.go` — delete entirely (execute*Tool wrappers are now redundant;
  their bodies move inline to executor.go call sites, which become single-line method calls)

```go
// internal/analysis/service.go (append)
type AnalysisService interface {
    AnalyzeBugs(args map[string]interface{})       (string, error)
    AnalyzeErrors(args map[string]interface{})     (string, error)
    QualityScan(args map[string]interface{})       (string, error)
    GetWorkPatterns(args map[string]interface{})   (string, error)
    GetTimeline(args map[string]interface{})       (string, error)
    GetTechDebt(args map[string]interface{})       (string, error)
}
```

```go
// cmd/mcp-server/executor.go — ToolExecutor gains one new field
type ToolExecutor struct {
    // ... existing fields ...
    analysisSvc analysis.AnalysisService
}
```

Run `make commit`.

Estimated: ~40 lines (interface + ToolExecutor field + inline dispatch + delete wrappers)

**Phase 59 validation**: `make commit` passes; `analysis.Service` has no fields; `ToolExecutor`
holds `AnalysisService`; `handlers_analysis.go` deleted; compliance test `var _ AnalysisService = (*Service)(nil)` passes.

---

## Parallel Execution Strategy

Stages 58.1 and 58.2 are independent — can run in parallel (separate worktrees).
Stage 58.3 depends only on 58.1 passing compilation.
Stage 59.1 must complete before 59.2 (interface depends on cleaned-up struct).

```
58.1 (parser import fix)  ──┐
58.2 (experiments rename) ──┤→ Stage 58.3 (file_inspector extraction) → Stage 59.1 → Stage 59.2
```

Two worktrees for 58.1 + 58.2 in parallel; merge before 58.3.

---

## Total Estimates

| Stage | Description | Estimated LOC |
|---|---|---|
| 58.1 | Fix parser import in query_executor | ~10 |
| 58.2 | Rename experiments/ → _experiments/ | ~5 |
| 58.3 | Extract file_inspector to subpackage | ~50 |
| 59.1 | Remove unused cfg from analysis.Service | ~30 |
| 59.2 | Add AnalysisService interface + tests | ~60 |
| **Total** | 5 stages | **~155 lines** |

Well within phase limit (≤500) and per-stage limit (≤200).

---

## Validation Checklist

- [ ] `cmd/mcp-server` imports: zero references to `internal/parser`
- [ ] `internal/query/files` package exists with `InspectFiles` function
- [ ] `analysis.Service` struct has zero fields
- [ ] `analysis.New()` takes no arguments
- [ ] `AnalysisService` interface exported from `internal/analysis`
- [ ] `var _ AnalysisService = (*Service)(nil)` compilation test passes
- [ ] `_experiments/` directory; no Go files in module graph from experiments
- [ ] `make commit` passes (all tests green, lint clean)

# Proposal: Architecture Hygiene — Five Structural Improvements

**Status**: Draft
**Date**: 2026-03-10
**Author**: Yale Huang

---

## 1. Problem Statement

Archguard analysis (2026-03-10) identified five structural issues after the completion of Phases 56–57:

### 1.1 P1 — `cmd/mcp-server` Still Imports `internal/parser`

Despite Phase 57 introducing `internal/analysis` as a facade, `cmd/mcp-server/query_executor.go`
retains a direct import of `internal/parser` solely for the constant `MaxScannerLineBytes`:

```go
// cmd/mcp-server/query_executor.go:15
import "github.com/yaleh/meta-cc/internal/parser"
// ...
buf := make([]byte, parser.MaxScannerLineBytes)  // lines 188, 189, 268, 269
```

Phase 56 already moved `MaxScannerLineBytes` to `internal/types` (with an alias in `internal/parser`).
The fix is a one-line import change — the alias was kept precisely for this migration.

### 1.2 P2 — `internal/query` is Oversized

`internal/query` contains 22 files (3,418 LOC implementation, 5,000+ LOC tests). `file_inspector.go`
(201 LOC) is a self-contained file-system inspection utility with **zero internal imports** — it uses
only stdlib (`bufio`, `encoding/json`, `os`, `time`). It is a natural candidate for extraction to
`internal/query/files`, consistent with the existing `internal/query/resources` pattern.

The broader query package (file_access, sequences, project_state, etc.) shares `buildTurnIndex`
from `context.go` and requires a more involved decomposition; that is deferred to a future proposal.

### 1.3 P3 — `internal/analysis.Service` Has No Interface

`cmd/mcp-server/handlers_analysis.go` calls `analysis.New(cfg).AnalyzeBugs(args)` etc. against the
concrete `*analysis.Service`. There is no interface, so the handler layer cannot be unit-tested with
a mock. This is the same structural gap that Phase 57 intended to close at the package level.

### 1.4 P4 — `internal/analysis.Service` Stores Unused `*config.Config`

Archguard flags `internal/analysis/service.go:21` as a `concreteUsageRisk`. Code audit reveals
that `Service.cfg` is **never read** by any method:

```go
type Service struct {
    cfg *config.Config  // stored but never accessed in AnalyzeBugs, QualityScan, etc.
}
```

The field was likely added in Phase 57 in anticipation of future config-driven behavior. Storing
an unused dependency bloats the constructor API and prevents callers from using the service without
a real `*config.Config`. The correct fix is to **remove `cfg` entirely** from the struct and the
constructor signature, not to wrap it in an interface.

### 1.5 P5 — `experiments/` Pollutes the Production Module Graph

`experiments/bootstrap-001-doc-methodology/data/` contains Go source files included in the module.
Archguard reports this directory as a production package. Go build tools ignore directories whose
names begin with `_`; renaming to `_experiments/` cleanly excludes all experimental code from
`go build`, `go test ./...`, and static analysis without any code changes.

---

## 2. Proposed Solution

### 2.1 Phase 58 — Facade Completion and Hygiene

Three changes with minimal blast radius:

**58.1** Update `cmd/mcp-server/query_executor.go`: replace `parser.MaxScannerLineBytes` with
`types.MaxScannerLineBytes`. Remove `internal/parser` import from `query_executor.go`.

**58.2** Rename `experiments/` → `_experiments/`. Update `.gitignore` if needed.

**58.3** Extract `internal/query/file_inspector.go` → `internal/query/files/file_inspector.go`.
Update the single caller (`cmd/mcp-server/handlers_inspect.go` or equivalent) to import
`internal/query/files`. No logic changes.

### 2.2 Phase 59 — Interface Extraction

**59.1** Remove `cfg *config.Config` from `internal/analysis.Service` (dead field):

```go
// before
type Service struct{ cfg *config.Config }
func New(cfg *config.Config) *Service { return &Service{cfg: cfg} }

// after
type Service struct{}
func New() *Service { return &Service{} }
```

Update all call sites in `cmd/mcp-server/handlers_analysis.go`:
`analysis.New(cfg).AnalyzeBugs(args)` → `analysis.New().AnalyzeBugs(args)`.
Remove the now-unused `cfg *config.Config` parameter from the `execute*Tool` functions if
`cfg` is not used elsewhere in those functions.

**59.2** Add `AnalysisService` interface to `internal/analysis`:

```go
// internal/analysis/service.go (or interface.go)
type AnalysisService interface {
    AnalyzeBugs(args map[string]interface{})       (string, error)
    AnalyzeErrors(args map[string]interface{})     (string, error)
    QualityScan(args map[string]interface{})       (string, error)
    GetWorkPatterns(args map[string]interface{})   (string, error)
    GetTimeline(args map[string]interface{})       (string, error)
    GetTechDebt(args map[string]interface{})       (string, error)
}
```

`*Service` already satisfies this interface; no implementation changes needed.
`cmd/mcp-server/handlers_analysis.go` stores the service as `AnalysisService` instead of `*Service`.

---

## 3. Constraints

- **No functional changes**: All 21 MCP tool behaviors preserved.
- **Backward compatibility**: `analysis.New` signature change is internal-only (no external API).
- **Test coverage ≥ 80%**: Existing tests must pass. New interface methods require mock-based tests.
- **Phase limit**: ≤ 500 lines total per phase; ≤ 200 lines per stage.

---

## 4. Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| `experiments/` rename breaks existing paths in scripts/CI | Low | Search all scripts and docs for `experiments/` before rename |
| `internal/query/files` extraction misses callers of `file_inspector.go` | Low | `grep -rn "query.*InspectSession\|query.*RecordSample"` before moving |
| `cfg` removal breaks callers that rely on `analysis.New(cfg)` signature | Low | Audit all callers in `cmd/mcp-server/`; update `execute*Tool` to remove `cfg` param if unused |
| `AnalysisService` interface drift (new methods added later without updating interface) | Medium | Interface lives in the same file as `Service`; compilation enforces consistency |

---

## 5. Out of Scope

- Full decomposition of `internal/query` (deferred; requires `buildTurnIndex` refactor).
- Converting `internal/analyzer` free functions to struct methods (deferred; `internal/analysis.Service` already provides the service facade).
- Reducing `cmd/mcp-server` package size (deferred to a dedicated package-split phase).

---

## 6. Expected Impact

| Metric | Before | After |
|---|---|---|
| `cmd/mcp-server` → `internal/parser` dependency | 1 ref (constant only) | **0** |
| `internal/query` file count | 22 | 20 (2 moved to subpackage) |
| `analysis.Service` unused dependency | Stores unused `*config.Config` | **No dependencies — zero-value constructible** |
| `handlers_analysis.go` testability | Requires real `*analysis.Service` | Accepts any `AnalysisService` mock |
| Experimental code in module graph | Yes | **No** |
| Archguard `concreteUsageRisk` count | 1 | **0** |

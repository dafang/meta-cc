# Proposal: Core Type Decoupling — Move Domain Types out of internal/parser

**Status**: Implemented (2026-03-10, Phases 56–57)
**Date**: 2026-03-10
**Author**: Yale Huang

---

## 1. Problem Statement

Archguard analysis identifies two structural violations:

### 1.1 `internal/parser` is a Hub Package (Dependency Inversion)

`internal/parser` is imported by 9 out of 13 packages. More critically, `internal/types` — which should define shared domain types — currently imports `internal/parser` to reference `SessionEntry` and `ToolCall`. This is a **dependency inversion**: the type layer depends on the parser layer.

```
internal/types/loader.go  →  import "internal/parser"  ← INVERSION
```

Any change to `SessionEntry`, `Message`, `ContentBlock`, `ToolCall`, or `ToolResult` propagates across 9 packages. New packages needing domain types must pull in the JSON parsing machinery unnecessarily.

**Packages importing `internal/parser`:**
```
cmd/mcp-server           (2 direct refs)
internal/analyzer        (9 refs)
internal/filter          (3 refs)
internal/query           (13 refs)
internal/query/resources (2 refs)
internal/stats           (4 refs)
internal/types           (1 ref — the inversion)
pkg/output               (6 refs)
pkg/pipeline             (1 ref)
```

### 1.2 `cmd/mcp-server` Bypasses the Abstraction Layer

`cmd/mcp-server/handlers_analysis.go` directly imports both `internal/parser` and `internal/analyzer`. The entry point orchestrates file location, parsing, and analysis itself instead of delegating to a service layer:

```go
// handlers_analysis.go
import (
    "github.com/yaleh/meta-cc/internal/analyzer"  // bypass
    "github.com/yaleh/meta-cc/internal/parser"     // bypass
)
```

`cmd/mcp-server/query_executor.go` additionally imports `internal/parser` only for the constant `MaxScannerLineBytes`, which has no semantic relationship to query execution.

---

## 2. Proposed Solution

### 2.1 Move Core Domain Types to `internal/types`

Extract from `internal/parser` into `internal/types`:

| Source | Type / Symbol | Target |
|---|---|---|
| `internal/parser/types.go` | `SessionEntry`, `Message`, `ContentBlock`, `ToolUse`, `ToolResult` | `internal/types/session.go` |
| `internal/parser/tools.go` | `ToolCall`, `ExtractToolCalls()` | `internal/types/toolcall.go` |
| `internal/parser/constants.go` | `MaxScannerLineBytes` | `internal/types/constants.go` |

**After migration:**
- `internal/parser` only contains `SessionParser`, `ParseEntries()`, `ParseEntriesFromContent()` — pure parsing logic — and imports `internal/types` for the types it returns.
- `internal/types` holds all shared domain types with zero parser dependency. `loader.go` no longer needs its `import "internal/parser"`.
- All existing consumers update references from `parser.SessionEntry` → `types.SessionEntry`, etc.
- **Backward compatibility**: type aliases in `internal/parser` ensure no external breakage:
  ```go
  // internal/parser/aliases.go (new file)
  type SessionEntry = types.SessionEntry
  type ToolCall = types.ToolCall
  // ... etc.
  ```

### 2.2 Create `internal/analysis` Service Package

Introduce a new `internal/analysis` package that encapsulates the full pipeline: locate → parse → analyze. This mirrors the existing pattern of `internal/query` for query operations.

```
cmd/mcp-server/handlers_analysis.go
    └─ imports: internal/analysis, internal/config
           └─ internal/analysis (new)
               ├─ imports: internal/locator, internal/parser, internal/analyzer, internal/types
               └─ exports: AnalysisService (or package-level functions)
```

**`internal/analysis` API:**
```go
type Service struct{ cfg *config.Config }

func New(cfg *config.Config) *Service

func (s *Service) AnalyzeBugs(args map[string]interface{}) (string, error)
func (s *Service) AnalyzeErrors(args map[string]interface{}) (string, error)
func (s *Service) QualityScan(args map[string]interface{}) (string, error)
func (s *Service) GetTechDebt(args map[string]interface{}) (string, error)
func (s *Service) GetTimeline(args map[string]interface{}) (string, error)
func (s *Service) GetWorkPatterns(args map[string]interface{}) (string, error)
```

`handlers_analysis.go` becomes a thin MCP dispatcher: create `*analysis.Service`, call the appropriate method, return the result.

**Why not wrappers in `internal/query`?**
`internal/query` is already the largest package (22 files, 115 functions). Mixing query and analysis operations increases its scope further. A dedicated `internal/analysis` package mirrors the clean separation between `internal/query` (query) and `internal/analyzer` (analysis computation), with `internal/analysis` as the service facade.

---

## 3. Constraints

- **No functional changes**: All 21 MCP tool behaviors are preserved.
- **Backward compatibility**: Type aliases in `internal/parser` cover any external consumers.
- **Test coverage ≥ 80%**: Existing tests must pass unmodified. New `internal/analysis` service must have tests covering the 6 public methods.
- **Phase limit**: ≤ 500 lines total across both phases; ≤ 200 lines per stage.

---

## 4. Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| JSON custom serialization on `Message`, `ContentBlock`, `ToolResult` uses `encoding/json` from the `parser` package — no unexported helpers, safe to move | Verified (low) | No action needed; methods move with the types |
| `ExtractToolCalls` references `ToolUse`, `ToolResult` — all types co-located in `parser/types.go` | High (resolved by moving all together) | Move `ExtractToolCalls` to `internal/types/toolcall.go` alongside types it references |
| Circular import: `internal/types` → `internal/parser` (current) must not become `internal/parser` → `internal/types` → `internal/parser` | Low | `internal/parser/aliases.go` uses `= types.X` syntax; `internal/types` has no parser import |
| `internal/types/loader.go` interface methods return `[]parser.SessionEntry` and `[]parser.ToolCall` — after migration, these become `[]types.SessionEntry`, `[]types.ToolCall` — all implementers (`pkg/pipeline.SessionPipeline`) must update | Medium | Part of Stage 56.3 update sweep |
| `internal/analysis` adds a new `config.Config` dependency path — verify no import cycle | Low | `internal/analysis` imports `internal/config`; `internal/config` imports nothing internal |

---

## 5. Out of Scope

- Moving `internal/analyzer` result types (`BugAnalysisResult`, etc.) to `internal/types`.
- Adding interfaces to `internal/analyzer`.
- Fixing `pkg/pipeline.SessionPipeline` concrete field `[]types.SessionEntry`.
- Reducing size of `internal/query`.

---

## 6. Expected Impact

| Metric | Before | After |
|---|---|---|
| Packages importing `internal/parser` | 9 | 2–3 (`internal/analysis`, `pkg/pipeline`; parser itself gone) |
| Packages importing `internal/types` | ~5 | ~10 (becomes true domain layer) |
| `cmd/mcp-server` imports of `internal/parser` | 2 | 0 |
| `cmd/mcp-server` imports of `internal/analyzer` | 1 | 0 |
| Dependency inversion (`types` → `parser`) | Yes | Resolved |
| New packages | 0 | 1 (`internal/analysis`) |

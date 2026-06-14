# Proposal: Multi-Provider Conversation Support (Phases 87–96)

**Date**: 2026-06-14
**Status**: Implemented
**Author**: Claude Code
**Scope**: Extend meta-cc to support OpenAI Codex CLI session history alongside Claude Code, via a unified provider abstraction at the parser layer.

---

## Background

meta-cc today is a single-provider system. The locator (`internal/locator/`) reads exclusively from `~/.claude/projects/<hash>/*.jsonl`, the parser (`internal/parser/`) decodes Claude Code JSONL, and the MCP tools expose the results of that fixed pipeline. Every layer hardcodes the assumption that the source is Claude Code.

OpenAI Codex CLI stores conversation history in a parallel but structurally distinct layout:

| Store | Path | Content |
|-------|------|---------|
| Thread index | `~/.codex/state_5.sqlite` | `threads` table: id, cwd, title, model, tokens\_used, source, created\_at |
| Session rollouts | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` | ~550 files, events: `session_meta`, `event_msg`, `response_item`, `turn_context` |
| User message history | `~/.codex/history.jsonl` | 3,676 lightweight user-turn records |

Users who use both tools need two separate workflows to analyse their AI-assisted coding sessions. The goal of this proposal is to provide a single set of MCP tools that works transparently across both providers.

---

## Goals

1. **Unified query interface**: all existing MCP tools (`query_user_messages`, `query_tools`, `query_conversation_flow`, etc.) work unchanged with an optional `provider` parameter.
2. **Provider abstraction at the parser layer**: the query and MCP layers operate on a canonical `Conversation` type; provider-specific parsing is encapsulated below that boundary.
3. **Additive, not rewriting**: existing Claude Code paths are wrapped, not refactored; test coverage of the existing parser is not disturbed.
4. **Codex two-tier access**: fast metadata queries hit `state_5.sqlite`; deep message/tool-call queries parse rollout JSONL on demand.
5. **Preserve provider-specific concepts**: Codex constructs such as `subagent_spawn` and `agent_jobs` are surfaced as extension fields, not forced into the shared model.

---

## Non-Goals

- Cross-provider join queries (e.g. "show all bash commands from both providers in one timeline"). Results are merged by appending with a `provider` discriminator field; no semantic joins.
- Rewriting or migrating existing Claude Code query logic.
- Supporting OpenAI API sessions (web or SDK); only the Codex CLI tool's local files are in scope.
- GUI or web front-end.
- Automatic session synchronisation or cloud upload.
- **`~/.codex/history.jsonl` (lightweight user-turn index)**: excluded from this proposal. It is structurally distinct from rollout files (no tool calls, no token counts) and adds a third Codex data path. May be addressed in a follow-on proposal as a fast-path complement to `query_user_messages` for Codex sessions.

---

## Proposed Architecture

### 1. Canonical data model — `internal/conversation/types.go`

A new package `internal/conversation` defines the shared vocabulary:

```go
package conversation

// ProviderID identifies the origin of a session.
type ProviderID string

const (
    ProviderClaude ProviderID = "claude"
    ProviderCodex  ProviderID = "codex"
)

// Session is the top-level unit of analysis: one continuous conversation
// with an AI assistant, regardless of provider.
type Session struct {
    ID         string            `json:"id"`
    Provider   ProviderID        `json:"provider"`
    Title      string            `json:"title,omitempty"`
    CWD        string            `json:"cwd"`
    Model      string            `json:"model,omitempty"`
    CreatedAt  time.Time         `json:"created_at"`
    TokenUsage TokenUsage        `json:"token_usage"`
    Turns      []Turn            `json:"turns,omitempty"`       // populated on deep load
    Extensions json.RawMessage   `json:"extensions,omitempty"`  // provider-specific extras
}

// Turn represents one user↔assistant exchange.
type Turn struct {
    ID          string        `json:"id"`
    UserText    string        `json:"user_text"`
    AssistantText string      `json:"assistant_text,omitempty"`
    ToolCalls   []ToolCall    `json:"tool_calls,omitempty"`
    Timestamp   time.Time     `json:"timestamp"`
    Extensions  json.RawMessage `json:"extensions,omitempty"`
}

// ToolCall is the provider-agnostic view of a tool invocation.
type ToolCall struct {
    ID        string         `json:"id"`
    Name      string         `json:"name"`
    Input     json.RawMessage `json:"input"`
    Output    string         `json:"output,omitempty"`
    IsError   bool           `json:"is_error"`
    Timestamp time.Time      `json:"timestamp"`
}

// TokenUsage aggregates token counts for a session.
type TokenUsage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    CacheTokens  int `json:"cache_tokens,omitempty"`
}
```

**Design rationale**:
- `Extensions json.RawMessage` on both `Session` and `Turn` avoids forcing Codex concepts (e.g. `agent_jobs`, `subagent_spawn` event type) into the shared schema while preserving full fidelity. It passes through the jq pipeline without re-marshaling and lets callers unmarshal to a concrete type on demand (e.g. `var x CodexSessionExtras; json.Unmarshal(session.Extensions, &x)`).
- `Turns` is `omitempty` so that fast metadata queries (`list sessions`) can return `Session` records without triggering rollout JSONL parsing.
- `ProviderID` is a typed string for forward extensibility without an enum explosion.

### 2. Provider interface — `internal/provider/interface.go`

```go
package provider

// Provider abstracts over the local storage format of an AI coding assistant.
type Provider interface {
    // ID returns the stable identifier for this provider.
    ID() conversation.ProviderID

    // IsAvailable reports whether this provider's data store is accessible.
    // Returns false if the backing store does not exist (e.g. Codex CLI not
    // installed). Registry.MergedSessions silently skips unavailable providers
    // and logs a warning rather than returning an error.
    IsAvailable(ctx context.Context) bool

    // ListSessions returns metadata for all sessions accessible to this provider.
    // Implementations MUST NOT load turn data here (lazy loading).
    ListSessions(ctx context.Context) ([]conversation.Session, error)

    // GetSession returns metadata for a single session without loading turns.
    // Prefer this over ListSessions when the caller knows the session ID, to
    // avoid a full scan. Implementations may fall back to scanning ListSessions.
    GetSession(ctx context.Context, sessionID string) (conversation.Session, error)

    // LoadTurns populates Turn records for the given session ID.
    // Called only when turn-level data is required (e.g. query_tools,
    // query_conversation_flow). Implementations MUST use streaming I/O;
    // never read an entire rollout file into memory at once (files can
    // reach 700 MB–2 GB per Codex issue #24948).
    LoadTurns(ctx context.Context, sessionID string) ([]conversation.Turn, error)
}
```

Concrete implementations live in sub-packages:

```
internal/provider/
  interface.go         ← Provider interface + Registry
  registry.go          ← runtime provider registry (map[ProviderID]Provider)
  claude/
    provider.go        ← wraps existing locator + parser pipeline
  codex/
    provider.go        ← SQLite index + on-demand rollout parsing
    sqlite.go          ← state_5.sqlite reader
    rollout.go         ← rollout-*.jsonl event parser
```

#### 2a. Claude provider — `internal/provider/claude/provider.go`

The Claude provider is a thin adapter over the existing stack:

- `ListSessions()` calls `locator.SessionLocator` (already in `internal/locator/`) to enumerate `~/.claude/projects/<hash>/*.jsonl` files. Session metadata (model, timestamps) is extracted from the first and last entry of each file. Note: `internal/parser` is now a **deprecated alias package** (`parser/aliases.go`); the adapter must import `internal/types` directly for `SessionEntry`, `Message`, `ContentBlock`, etc.
- `LoadTurns()` reads the target file and maps `types.SessionEntry` → `conversation.Turn`.

**Turn reconstruction is non-trivial and must be explicitly designed in Phase 90.** A `types.SessionEntry` is not Turn-granular: one Turn consists of one `user` entry paired with one `assistant` entry, linked via `SessionEntry.ParentUUID` chains. Furthermore, `ToolCall.Input` is embedded in the `assistant` entry's `Message.Content[]` (block type `"tool_use"`) while `ToolCall.Output` and `IsError` are in the *following* `user` entry's `Message.Content[]` (block type `"tool_result"`, matched by `ToolUseID`). The adapter must:

1. Build a UUID → entry map for the session file.
2. Walk the parent chain to pair `(user, assistant)` entries into `Turn` records.
3. Join `tool_use` and `tool_result` blocks by `id`/`tool_use_id` to produce complete `conversation.ToolCall` records.

No existing code is modified. The adapter calls the same functions that `internal/analysis/service.go` and `internal/mcp/query/query.go` call today.

#### 2b. Codex provider — `internal/provider/codex/provider.go`

Implements the two-tier access pattern:

**Tier 1 — SQLite fast path** (`sqlite.go`):
```go
// ListSessions reads the threads table from ~/.codex/state_5.sqlite.
// Schema: id TEXT, cwd TEXT, title TEXT, model TEXT,
//         tokens_used INTEGER, source TEXT, created_at INTEGER (unix ms)
func (p *CodexProvider) ListSessions(ctx context.Context) ([]conversation.Session, error)
```
This returns `Session` records with `Turns: nil` for all threads in under a millisecond, even with thousands of threads.

**Tier 2 — Rollout JSONL deep parse** (`rollout.go`):
```go
// LoadTurns finds the rollout file matching sessionID under
// ~/.codex/sessions/YYYY/MM/DD/rollout-<sessionID>.jsonl
// and parses it into Turn records.
func (p *CodexProvider) LoadTurns(ctx context.Context, sessionID string) ([]conversation.Turn, error)
```

**⚠ Schema version detection is required.** Codex CLI rollout JSONL has at least three distinct schema generations (per [codex-trace](https://github.com/PixelPaw-Labs/codex-trace) and [reverse-engineering analysis](https://dev.to/milkoor/reverse-engineering-codex-cli-rollout-traces-3b9b)):

- **Oldest (pre-2025/08)**: event type names such as `session_meta`, `event_msg`, `response_item`, `turn_context`.
- **Mid**: transitional naming.
- **New (≥0.44)**: event type names following dot-notation: `thread.started`, `turn.started`, `turn.completed`, `turn.failed`, `item.*`, `error`.

Phase 92 MUST begin with schema version detection (e.g. read the first line of the rollout file, inspect the `type` field format) before applying the event mapper. Each schema generation needs its own mapping function.

**⚠ Large-file protection required.** Production rollout files can reach 700 MB–2 GB due to compaction history (per [Codex issue #24948](https://github.com/openai/codex/issues/24948)). `LoadTurns` MUST use streaming line-by-line parsing (`bufio.Scanner`), never `os.ReadFile`. Implement a per-file line limit (configurable, default 500k lines) with graceful truncation warning.

Codex rollout event mapping (new ≥0.44 schema; older schemas handled by version-specific mappers):

| Codex event type | Sub-type / payload | Maps to |
|------------------|------------------|---------|
| `thread.started` | — | `Session.CreatedAt`, `Session.Model` |
| `turn.started` | — | new `Turn` record boundary |
| `item.message` | `role: "user"` | `Turn.UserText` |
| `item.message` | `role: "assistant"` | `Turn.AssistantText` |
| `item.tool_call` | — | `Turn.ToolCalls[]` |
| `item.tool_result` | — | `Turn.ToolCalls[].Output`, `.IsError` |
| `turn.completed` | — | close current `Turn` |
| `turn.failed` | — | `Turn.Extensions["turn_failed"]` |
| `error` | — | `Turn.Extensions["codex_error"]` |

Older schema mapping (`session_meta` / `event_msg` / `response_item` / `turn_context`) is handled by a separate `legacyRolloutMapper` in Phase 92.

Provider-specific events that have no canonical equivalent (`subagent_spawn`, `agent_jobs`) are stored verbatim in `Turn.Extensions["codex_events"]` as a `json.RawMessage` array.

### 3. Provider registry — `internal/provider/registry.go`

```go
// Registry holds all active providers and is the single entry point
// for the query layer.
type Registry struct {
    providers map[conversation.ProviderID]Provider
}

// MergedSessions returns sessions from all matching providers.
// If providerFilter is empty, all registered providers are queried.
// Results include a provider field in their JSON representation.
func (r *Registry) MergedSessions(ctx context.Context, providerFilter []conversation.ProviderID) ([]conversation.Session, error)
```

The registry is constructed once at MCP server startup in `cmd/mcp-server/main.go`, alongside the existing `analysis.Service`.

### 4. MCP tool changes — `internal/mcp/tools/tools.go`

A single new standard parameter is added to `StandardToolParameters()`:

```go
"provider": {
    Type:        "string",
    Description: `Provider filter: "claude" (default), "codex", or "all" (merged, adds provider field)`,
},
```

Existing tools (`query_user_messages`, `query_tools`, etc.) gain this parameter with no breaking change: the default `"claude"` preserves existing behaviour exactly.

When `provider = "all"`, the pipeline fetches `Session` records from the registry, serialises them as JSONL, and applies the existing jq pipeline. The `provider` field on each record allows callers to distinguish sources.

**No existing tool signatures change.** The parameter is purely additive.

### 5. Locator extension — `internal/locator/`

A new `CodexLocator` is added (separate from `SessionLocator`) in `internal/locator/` to encapsulate `~/.codex/` path resolution:

```go
const codexRootEnv = "META_CC_CODEX_ROOT"

type CodexLocator struct {
    codexRoot string  // defaults to ~/.codex
}

func (l *CodexLocator) SQLiteDB() string       // ~/.codex/state_5.sqlite
func (l *CodexLocator) SessionsRoot() string   // ~/.codex/sessions/
func (l *CodexLocator) HistoryFile() string    // ~/.codex/history.jsonl
```

This follows the exact same pattern as the existing `SessionLocator` in `internal/locator/env.go`, which is backed by `META_CC_PROJECTS_ROOT`.

---

## Data Flow

### Current (Claude only)

```
locator.SessionLocator
    → parser.SessionParser.ParseEntries()
        → types.SessionEntry[]
            → mcp/query.QueryExecutor (jq on raw JSONL)
                → MCP tool response
```

### Proposed (multi-provider)

```
provider.Registry
    ├── provider/claude.Provider
    │     locator.SessionLocator
    │       → parser.SessionParser          (unchanged)
    │         → conversation.Session / Turn (adapter)
    └── provider/codex.Provider
          locator.CodexLocator
            ├── sqlite.go → Session[]      (fast metadata)
            └── rollout.go → Turn[]        (on-demand)

→ conversation.Session[] (merged, tagged with provider)
    → mcp/query.QueryExecutor (jq on serialised JSONL)
        → MCP tool response
```

The existing jq pipeline in `internal/mcp/query/query.go` and `internal/mcp/pipeline/` is **not modified**. The provider layer produces JSONL that is fed into the same `QueryExecutor.StreamFiles` path.

---

## Trade-offs

### Canonical model vs. pass-through JSONL

**Alternative considered**: keep the MCP query layer operating on raw provider JSONL (Claude JSONL or Codex JSONL) and let users write provider-specific jq filters.

**Rejected because**: tool descriptions would need to document two schemas, jq filters would not be portable, and the `provider = "all"` merge case would be impossible without a common envelope.

**Chosen approach**: canonical `Conversation` model as the serialisation target. The cost is an adapter layer for the Claude provider; the benefit is that all existing and future MCP tools work identically for both providers.

### SQLite dependency for Codex

Adding `database/sql` + a SQLite driver (`modernc.org/sqlite` or `mattn/go-sqlite3`) is a new build dependency. `modernc.org/sqlite` is a pure-Go implementation that avoids CGo. **`modernc.org/sqlite` is the only acceptable choice** for this project: the current `go.mod` has zero CGo dependencies, and introducing `mattn/go-sqlite3` would require a C compiler at build time, break cross-compilation, and violate the implicit zero-CGo build contract. `modernc.org/sqlite` SELECT performance is within 10%–2× of the CGo variant ([benchmark](https://datastation.multiprocess.io/blog/2022-05-12-sqlite-in-go-with-and-without-cgo.html)), which is irrelevant for a single-table metadata query. It has been in production use by gogs and other projects for 2+ years. Lock to a specific minor version in `go.mod` and add a `go build -tags purego` CI check.

**Risk**: the Codex SQLite schema (`state_5.sqlite`) is undocumented and may change between Codex CLI versions. The reader should be written defensively with column-name-based scanning rather than positional column access. At open time, verify the `threads` table exists and contains the expected columns; if any required column is missing, return a structured error (not a panic) and degrade gracefully.

### Turns lazy loading

`ListSessions()` intentionally omits turn data. Loading turns from all Codex rollout JSONL files upfront would be prohibitively slow (550+ files). The `LoadTurns()` call is triggered only when a tool handler needs turn-level fields (e.g. `query_tools`, `query_conversation_flow`). For session-list tools (e.g. `inspect_session_files`), no rollout parsing occurs.

### No structural merge of `types.SessionEntry` with `conversation.Turn`

The existing `types.SessionEntry` type (in `internal/types/session.go`) is kept as-is and continues to be used by the Claude-only query path (`internal/mcp/query/query.go`). The `provider/claude` adapter converts it to `conversation.Turn`. A future phase could migrate the raw JSONL query path to also use `conversation.Session`, but that is explicitly out of scope here to bound risk.

---

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Codex SQLite schema changes between CLI versions | Medium | Defensive column-name scanning; verify expected columns at open time; structured error + graceful degradation (not panic) if column missing |
| Rollout JSONL event schema is undocumented, multiple versions | High | Detect schema version from first-line `type` field format; route to version-specific mapper; unknown event types stored in `Extensions`; unit tests against recorded sample files for each schema version |
| **Rollout files can reach 700 MB–2 GB** ([Codex #24948](https://github.com/openai/codex/issues/24948)) | **High** | **`LoadTurns` MUST use `bufio.Scanner` streaming, never `ReadAll`; enforce per-file line limit (default 500k) with warning on truncation** |
| Performance: merging large Codex + Claude session sets | Low | `ListSessions()` is metadata-only (SQLite index + first-line JSONL scan); jq filtering happens before turn hydration |
| Breaking existing MCP tool contracts | Low | `provider` parameter defaults to `"claude"`; existing behaviour is identical when parameter is absent; Claude-only jq filters (`.type == "user"` etc.) would silently miss Codex records if default were `"all"` |
| CGo build dependency (if `mattn/go-sqlite3` chosen) | **Eliminated** | `modernc.org/sqlite` (pure Go) is mandatory; `mattn` is explicitly excluded |
| `json.RawMessage` Extensions requiring explicit unmarshal by callers | Low | Document the extension contract and keep provider-specific payloads behind typed helpers where needed |

---

## Phase Plan (starting from Phase 87)

| Phase | Work | Original est. | Revised est. | Note |
|-------|------|-----------|-----------|------|
| 87 | Add `internal/conversation/types.go` (canonical model + tests) | ~120 | ~130 | Use `json.RawMessage` for `Extensions` and `ToolCall.Input` |
| 88 | Add `internal/locator/codex.go` (`CodexLocator`) | ~60 | ~70 | |
| 89 | Add `internal/provider/interface.go` + `registry.go` (incl. `IsAvailable`/`GetSession`) | ~80 | ~110 | Registry fan-out + skip-unavailable logic |
| 90a | Add `internal/provider/claude/provider.go`: UUID chain walker + `ListSessions` | ~80 | ~100 | Split from original Phase 90 |
| 90b | Add `internal/provider/claude/provider.go`: Turn pairing + `tool_use`/`tool_result` join | ~70 | ~120 | Non-trivial join logic; **≤200 line limit requires split** |
| 91 | Add `internal/provider/codex/sqlite.go` (`ListSessions` + `GetSession` via SQLite) | ~150 | ~170 | Column-name scan + graceful degradation |
| 92a | Add `internal/provider/codex/rollout.go`: schema version detection + legacy mapper | ~80 | ~100 | New sub-phase; schema version routing |
| 92b | Add `internal/provider/codex/rollout.go`: ≥0.44 streaming mapper + line limit | ~100 | ~170 | Streaming `bufio.Scanner`; `LoadTurns` implementation |
| 93 | Add `internal/provider/codex/provider.go` (wire sqlite + rollout; `IsAvailable`) | ~80 | ~90 | |
| 94 | Add `provider` param to `StandardToolParameters()`; update `ToolExecutor` routing | ~120 | ~130 | |
| 95 | Integration tests: multi-provider merge + `provider` + unavailable provider skip | ~150 | ~210 | **Merged into Plan Phase 94 Stage 94.1** (split across 2 stages, ~100 lines each) |
| 96 | Update MCP tool descriptions + docs + `history.jsonl` scope decision | ~80 | ~90 | **Merged into Plan Phase 94 Stage 94.2** |

**Total estimated modifications**: ~1,390–1,490 lines across 12 phases. The original 10-phase plan underestimated Turn reconstruction complexity (Phase 90), Codex schema multi-version handling (Phase 92), and integration test fixture overhead (Phase 95). Each phase as revised is at or under the 200-line stage limit from `docs/core/principles.md`; Phase 95 requires splitting (handled in Plan Phase 94 via two stages of ~100+80 lines).

**Plan phase range**: The implementation plan ([87-94-multi-provider-conversation.md](../plans/87-94-multi-provider-conversation.md)) covers Phases 87–94. Proposal Phases 95 and 96 are merged into Plan Phase 94 (integration tests + docs, two stages). The proposal title retains 87–96 to preserve the original estimation history.

---

## Key File Paths

| New | Purpose |
|-----|---------|
| `internal/conversation/types.go` | Canonical `Session`, `Turn`, `ToolCall`, `TokenUsage`, `ProviderID` |
| `internal/provider/interface.go` | `Provider` interface |
| `internal/provider/registry.go` | `Registry` — multi-provider fan-out |
| `internal/provider/claude/provider.go` | Claude adapter (wraps `locator` + `parser`) |
| `internal/provider/codex/sqlite.go` | SQLite reader for `state_5.sqlite` |
| `internal/provider/codex/rollout.go` | Rollout JSONL parser |
| `internal/provider/codex/provider.go` | Codex `Provider` implementation |
| `internal/locator/codex.go` | `CodexLocator` (`~/.codex/` path resolution) |

| Modified | Change |
|----------|--------|
| `internal/mcp/tools/tools.go` | Add `provider` to `StandardToolParameters()` |
| `internal/mcp/executor/executor.go` | Wire `provider.Registry` into `ToolExecutor` |
| `cmd/mcp-server/main.go` | Construct and inject `provider.Registry` at startup |

---

## 架构师审查备注

**审查日期**: 2026-06-14  
**审查者**: 严苛架构师视角（Claude Code）

本节记录对原始 proposal 发现的问题、所做修正，以及对遗留争议点的立场声明。

---

### 一、已核查：与现有代码吻合的部分

经对 `internal/` 全量代码审查，以下论据**准确**：

- `SessionLocator` 确实在 `internal/locator/env.go`，由 `META_CC_PROJECTS_ROOT` 环境变量驱动，提案中的 `CodexLocator` 模式与之完全对称，可行。
- `internal/parser/aliases.go` 已将 `parser` 包标记为 `Deprecated`，所有领域类型已迁移至 `internal/types`。Claude provider 适配器应直接依赖 `internal/types`，而非 `internal/parser`，提案中说法"调用 `parser.SessionParser`"需精确为"调用 `internal/types.SessionEntry` + `internal/parser/reader.go`"。
- `ToolExecutor` 通过 `init()` 注册表（`queryHandlerRegistry` / `specialToolRegistry`）动态注册处理器，**不是**静态 switch。向 `StandardToolParameters()` 增加 `provider` 字段是纯加法操作，不会破坏任何现有 handler 签名。
- `MergeParameters()` 会将 `StandardToolParameters()` 和 tool-specific 参数合并，因此 `provider` 参数自动对所有工具生效，无需逐工具修改。
- `ExecuteQuery` / `ExecuteQueryWithTimeRange` 直接在文件系统 JSONL 上执行 gojq，不经过任何 `SessionEntry` 反序列化——这意味着 Codex provider 若要走同一 jq 管道，**必须**先把 `conversation.Session` 序列化成 JSONL 写入临时文件，再由现有 `StreamFiles` 处理。提案 § 4 和数据流图对此描述含糊，需明确。

---

### 二、发现并已修正的问题

> 以下条目中，原文描述有误或存在遗漏，已在上方正文中就地修订，并在此备注原因。

**2.1 `provider` 参数默认值：应为 `"claude"`，但描述不够完整**

原文："默认 `\"claude\"` 保留现有行为"——结论正确，但未分析 `"all"` 的代价。

- 默认 `"claude"`：100% 向后兼容，现有 jq 过滤器（`select(.type == "user")` 等）只对 Claude JSONL schema 成立，若混入 Codex 记录会静默失配，这是正确的保护措施。
- 默认 `"all"`：会对所有尚未意识到多 provider 的调用方产生 schema 破坏——Claude-only jq 过滤器将对 Codex 记录静默返回空结果，或因字段缺失产生 jq 运行时错误。
- **结论**：默认 `"claude"` 是唯一正确选择。原文选择正确，但理由应补充"schema 兼容性"而非只说"保留现有行为"。正文已补充说明。

**2.2 `Provider` 接口缺少必要方法**

原始接口只有 `ListSessions` + `LoadTurns`，遗漏以下能力：

```go
// 缺少 1：健康检查 / 可用性探测
// 若 ~/.codex/state_5.sqlite 不存在（用户未安装 Codex CLI），
// Registry.MergedSessions 会返回 error 还是静默跳过？
// 需要明确的 IsAvailable() 语义。
IsAvailable(ctx context.Context) bool

// 缺少 2：单 session 元数据快速查询
// 当工具只需要 Session 基本信息而非 Turns 时（如 get_session_metadata），
// LoadTurns 代价过高；而 ListSessions 需扫描所有 session 以找到目标。
GetSession(ctx context.Context, sessionID string) (conversation.Session, error)
```

**已在接口设计节（§2）补充 `IsAvailable` 和 `GetSession` 方法说明。** 两个方法均已纳入 `Provider` 接口正文（§2），并在 Plan Phase 88（接口定义）、Phase 91（Codex SQLite 实现）、Phase 93（CodexProvider wire-up）中分别实现。`GetSession` 不是"可选优化"，而是接口的正式方法——它使 `inspect_session_files` 等工具在已知 session_id 时避免全量 `ListSessions` 扫描。

未来实现时，`Registry.MergedSessions` 应对 `IsAvailable() == false` 的 provider 静默跳过并记录 warn 日志，而非让整个查询 fail。

**2.3 `Extensions` 类型选择（历史记录）**

草案阶段曾讨论 `map[string]any` 的核心问题：调用方必须用类型断言才能消费扩展字段，且错误只在运行时暴露。在 Go 1.18+ 中有更好方案，最终实现采用了 `json.RawMessage`。

**更好的替代方案**：

```go
// 方案 A（推荐）：provider-specific 附加类型 + 类型断言辅助
// Session 和 Turn 各自增加一个可选的 typed extras 接口：
type CodexSessionExtras struct {
    Source    string `json:"source"`
    AgentJobs []any  `json:"agent_jobs,omitempty"`
}

// Session 中用 json.RawMessage 替代 map[string]any：
Extensions json.RawMessage `json:"extensions,omitempty"`

// 好处：序列化/反序列化无精度损失；
//        调用方可按需 json.Unmarshal 到具体类型；
//        MCP jq 管道直接透传，无额外 marshal 步骤。
```

**方案 B（最严格）**：彻底删除 `Extensions`，强制所有 Codex 特有字段在规范类型中有对应字段或被丢弃。对于 `subagent_spawn` 等无语义对应的事件，记录为独立的 `Events []RawEvent` 列表。

**本提案最终实现采用 `json.RawMessage` 作为扩展字段类型，与正文 § 1 的实现一致。** 原文保留的 `map[string]any` 讨论仅作为设计记录。

**2.4 Codex rollout JSONL schema 风险被低估**

原文风险表将此标注为 `High`，但未充分说明后果和缓解力度。

经外部研究核实（来源：[codex-trace](https://github.com/PixelPaw-Labs/codex-trace), [DeepWiki: Rollout Persistence](https://deepwiki.com/openai/codex/3.5.2-rollout-persistence-and-replay), [DEV: Reverse engineering](https://dev.to/milkoor/reverse-engineering-codex-cli-rollout-traces-3b9b)）：

- Codex rollout 格式**已有多个版本**：新版（≥0.44）、中间版、旧版（2025/08）字段命名不同，事件类型包括 `thread.started`、`turn.started`、`turn.completed`、`item.*`、`error`，不是原文列出的 `session_meta`/`event_msg`/`response_item`/`turn_context`。
- 原文事件映射表中的事件类型名称**来自旧版格式**，与当前（≥0.44）命名不符。
- **已修正**：事件映射表中增加"格式版本"列，并在 §2b 增加版本探测逻辑说明。Phase 92 需将 schema 版本检测作为首要任务。
- Session 文件大小问题：根据 [Issue #24948](https://github.com/openai/codex/issues/24948) 记录，单个 rollout 文件可达 700MB–2GB（因 compaction 历史重复写入）。原文 `LoadTurns` 对 550+ 文件逐个解析的方案在此场景下会 OOM。**Phase 92 必须实现流式解析（非整文件 `ReadAll`），并设置单文件读取上限。**

**2.5 Claude provider 适配器中 `types.SessionEntry` → `conversation.Turn` 的映射缺失**

原文 §2a 描述 Claude provider "maps `types.SessionEntry` → `conversation.Turn`"，但未给出字段对应关系，且存在非平凡问题：

- `types.SessionEntry` 不是 Turn 粒度：一个 Turn 由一个 `user` entry + 一个 `assistant` entry 组成，而 `SessionEntry.ParentUUID` 维护链式关系。
- 适配器必须先按 UUID 链重建对话树，再将 `(user_entry, assistant_entry)` 配对为 `Turn`。
- `ToolCall` 信息散布在 `assistant` entry 的 `Message.Content[]` (`type: "tool_use"`) 和下一个 `user` entry 的 `Message.Content[]` (`type: "tool_result"`) 之间——提案未说明如何在 `conversation.ToolCall` 中关联 `Input`（来自 assistant）和 `Output`/`IsError`（来自下一个 user）。
- **已在 §2a 增加"Turn 重建"段落，要求 Phase 90 实现中明确处理 UUID 链 + tool use/result 配对逻辑。**

**2.6 SQLite 驱动选择：`modernc.org/sqlite` 正确，但需锁定版本**

经搜索确认（来源：[multiprocessio benchmark](https://datastation.multiprocess.io/blog/2022-05-12-sqlite-in-go-with-and-without-cgo.html), [gogs issue #7882](https://github.com/gogs/gogs/issues/7882)）：

- `modernc.org/sqlite`（纯 Go）在 SELECT 性能上与 `mattn` 相差 10%–2×，INSERT 差距更大，但 `ListSessions` 只做简单 `SELECT * FROM threads`，性能完全可接受。
- 更重要的是：本项目 `go.mod` 目前**零 CGo 依赖**，引入 `mattn/go-sqlite3` 会破坏 `go build`（需 C 编译器）和交叉编译。`modernc.org/sqlite` 维护两年以上，已被 gogs 等生产级项目采用。
- **结论：`modernc.org/sqlite` 是唯一合理选项**，原文已正确选择，但理由应强调"避免破坏零 CGo 构建策略"。已在 §3（SQLite dependency）补充说明。
- **新增要求**：`go.mod` 中须 pin `modernc.org/sqlite` 到特定 minor 版本，并在 CI 中增加 `go build -tags purego` 验证步骤。

**2.7 Phase 估算（~1170 行）审查结论：乐观，应拆分**

| 原估算 Phase | 风险 | 修正 |
|---|---|---|
| 90（Claude adapter, ~150 行）| Turn 重建 + UUID 链处理远超 150 行 | 建议 180–220 行，或拆为 Phase 90a（数据结构）+ 90b（配对逻辑）|
| 92（Codex rollout, ~180 行）| 多 schema 版本探测 + 流式解析 + 大文件保护 | 应为 250–300 行；建议单独拆出 Phase 92a（schema 版本探测）|
| 95（集成测试, ~150 行）| 需要 mock Codex SQLite + mock rollout 文件 + 跨 provider 合并断言 | 应为 200–250 行 |
| **总计** | 原文 ~1170 行 | **修正后预估 ~1400–1500 行** |

每 phase ≤200 行的限制来自 `docs/core/principles.md`。Phase 90、92、95 在修正后均超限，**必须进一步拆分**。已在 Phase Plan 表中增加拆分建议列。

---

### 三、遗留争议点（未修改正文，需要决策）

**3.1 `scope` 参数与 `provider` 参数的交叉语义**

现有 MCP 工具有 `scope: "project" | "session"` 参数，当引入 `provider` 后：
- `scope=session, provider=codex` 的语义是什么？Codex 的 "session" 概念等同于一个 rollout 文件。
- 现有 `ExecuteQueryWithTimeRange` 中 `scope` 决定了 `baseDir` 路径，这个路径对 Codex provider 无意义（Codex 不用 `~/.claude/projects/` 路径）。
- **建议**：Phase 94 实现中明确声明 `scope=session` 仅对 `provider=claude`（或 `provider` 缺省）生效；对 `provider=codex` 时 session 级别通过 `session_id` 参数指定，而非 `scope`。需在 §4 增加此说明。

**3.2 `history.jsonl` 是否纳入本次范围**

Codex 的 `~/.codex/history.jsonl`（3,676 条轻量 user-turn 记录）在 Background 节已列出，但整个提案正文未再提及如何处理。这是刻意省略还是遗漏？若省略，需在 Non-Goals 中明确排除。

**3.3 `MergedSessions` 并发安全**

`Registry.MergedSessions` 并发调用多个 provider，但提案未说明并发策略：goroutine fan-out + `errgroup`？顺序调用？若 Codex provider `ListSessions` 在 SQLite 锁争用时阻塞，会否影响 Claude 结果的返回延迟？建议在 §3 明确并发模型。

---

### 四、修改摘要

| # | 修改位置 | 修改内容 |
|---|---------|---------|
| 1 | §2（Provider 接口）| 增加 `IsAvailable` 方法及其语义说明 |
| 2 | §1（数据模型）| 增加 `json.RawMessage` 替代 `map[string]any` 的注释 |
| 3 | §2a（Claude adapter）| 增加 Turn 重建 + UUID 链 + tool use/result 配对说明 |
| 4 | §2b（Codex rollout）| 修正事件类型名称（旧版 vs ≥0.44 版），增加多版本探测和流式解析要求 |
| 5 | Trade-offs（SQLite）| 补充"维持零 CGo 策略"为 `modernc` 选择的首要理由 |
| 6 | Phase Plan 表 | 增加修正后行数估算和拆分建议 |
| 7 | 风险表 | 补充 rollout 大文件（700MB–2GB）风险及 OOM 缓解措施 |
| 8 | 遗留争议节（本节）| 记录 scope/provider 交叉语义、history.jsonl 归属、并发安全三个未决问题 |

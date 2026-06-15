# Feature Overview

meta-cc is a provider-aware MCP server for local coding-agent history analysis. It supports Claude Code and Codex through a shared session model, then exposes query and analysis tools to the host.

## Host Support

| Host | Data source | Integration files |
|------|-------------|-------------------|
| Claude Code | `~/.claude/projects/<project-hash>/*.jsonl` | `plugin-src/.claude-plugin/`, `plugin-src/.mcp.json`, `plugin-src/commands/` |
| Codex | `${META_CC_CODEX_ROOT:-~/.codex}/state_5.sqlite` plus rollout JSONL files referenced by `threads.rollout_path` | `plugin-src/.codex-plugin/`, `plugin-src/.codex-mcp.json`, `plugin-src/skills/` |

The `provider` parameter controls which history is queried:

- `claude`: Claude Code only. This is the default for backward compatibility.
- `codex`: Codex only.
- `all`: merge both providers and include provider-tagged records.

## MCP Tools

meta-cc exposes 21 MCP tools.

### Convenience Queries

- `query_user_messages`: search user messages by regex
- `query_tools`: query assistant tool calls
- `query_tool_errors`: query failed tool results
- `query_token_usage`: query assistant token usage
- `query_conversation_flow`: query user/assistant turns
- `query_tool_blocks`: query `tool_use` or `tool_result` blocks
- `query_timestamps`: query timestamped records
- `query_system_errors`: query Claude Code API system errors
- `query_file_snapshots`: query Claude Code file history snapshots
- `query_summaries`: query Claude Code session summaries

Claude-only record types return empty results for Codex when Codex has no equivalent local record.

### Analysis Tools

- `analyze_errors`: aggregate tool errors by tool and type
- `analyze_bugs`: detect error-fix pairs and recurring patterns
- `quality_scan`: compute error, retry, diversity, and completion dimensions
- `get_work_patterns`: summarize tool frequency, hourly activity, and context switches
- `get_timeline`: build chronological session events
- `get_tech_debt`: detect TODO/FIXME/HACK markers and unresolved error signals

### Two-Stage Query Tools

- `get_session_directory`: locate a session directory and aggregate metadata
- `inspect_session_files`: inspect selected JSONL files
- `execute_stage2_query`: run jq-style filter/sort/transform on selected files
- `get_session_metadata`: return schema hints, file info, and query templates

### Utilities

- `cleanup_temp_files`: remove old temporary MCP output files
- `list_capabilities`: list packaged prompt/command capabilities
- `get_capability`: retrieve a capability by name/type

## Provider-Aware Normalization

Codex rollout records are normalized into the same conversation model used for Claude Code:

- user and assistant messages
- function/custom tool calls
- function/custom tool outputs
- token count events when present
- session metadata from SQLite

This lets the same MCP tools answer questions such as:

```text
Which tools do I use most often?
Show my work patterns and peak hours
Find user messages mentioning "refactor"
Analyze recent tool errors
Show token usage for recent assistant turns
```

## Prompt Library

meta-cc provides matching prompt-library workflows in both hosts.

Claude Code slash commands:

- `/prompt-find`
- `/prompt-list`
- `/prompt-show`

Codex skills:

- `$prompt-find`
- `$prompt-list`
- `$prompt-show`

Both read `.meta-cc/prompts/library/` in the current project and parse Markdown frontmatter fields such as `id`, `title`, `category`, `keywords`, `usage_count`, `updated`, and `status`.

## Output Modes

MCP responses use hybrid output:

- small results return inline
- large results are written to temporary JSONL files and returned as `file_ref`

This keeps natural host conversations usable while preserving complete result sets.

## Verification

General checks:

```text
Which tools do I use most often?
Find user messages mentioning "release"
Show token usage for recent assistant turns
```

Codex-specific check:

```text
Use provider=codex and show my work patterns
```

Development E2E:

```bash
make test-e2e-codex
```

The Codex E2E test creates an isolated Codex home, installs Codex skills and plugin metadata, creates SQLite and rollout fixtures, and verifies MCP calls with `provider: "codex"`.

## See Also

- [MCP Guide](../guides/mcp.md)
- [MCP Query Tools Reference](../guides/mcp-query-tools.md)
- [Integration Guide](../guides/integration.md)
- [Examples](../tutorials/examples.md)
- [JSONL Schema Reference](jsonl-schema.md)

# MCP Server Guide

meta-cc provides a Model Context Protocol (MCP) server for Claude Code and Codex session analysis. It exposes 21 tools for querying transcript history, running jq over selected session files, and computing higher-level analysis such as error patterns and work patterns.

## Host Support

| Host | Transcript root | Integration files |
|------|-----------------|-------------------|
| Claude Code | `~/.claude/projects/<project-hash>/` | `.mcp.json`, `.claude-plugin/plugin.json`, slash commands |
| Codex | `$CODEX_HOME/sessions` or `~/.codex/sessions` | `.codex-mcp.json`, `.codex-plugin/plugin.json`, skills |

Codex uses a different JSONL schema from Claude Code. The MCP server normalizes Codex messages, tool calls, tool outputs, and token counts into the same message/tool shape used by the existing query and analysis tools.

## Configuration

### Claude Code

The release archive includes `.mcp.json`; the installer merges it into `~/.claude/mcp.json`.

Manual configuration:

```json
{
  "mcpServers": {
    "meta-cc": {
      "command": "meta-cc-mcp",
      "args": []
    }
  }
}
```

### Codex

The release archive includes `.codex-plugin/plugin.json` and `.codex-mcp.json`. The installer copies them under `~/.codex/plugins/meta-cc/`.

Use `CODEX_HOME` to target a custom Codex home:

```bash
CODEX_HOME=/tmp/codex ./install.sh
```

## Tool Catalog

### Convenience Query Tools

| Tool | Purpose | Codex support |
|------|---------|---------------|
| `query_user_messages` | Search user messages by regex | Yes |
| `query_tools` | Query assistant tool calls | Yes |
| `query_tool_errors` | Query failed tool results | Yes |
| `query_token_usage` | Query token usage | Yes |
| `query_conversation_flow` | Query user/assistant flow | Yes |
| `query_tool_blocks` | Query `tool_use` or `tool_result` blocks | Yes |
| `query_timestamps` | Query timestamped records | Yes |
| `query_system_errors` | Query Claude Code API system errors | Host-specific empty |
| `query_file_snapshots` | Query Claude Code file snapshots | Host-specific empty |
| `query_summaries` | Query Claude Code summaries | Host-specific empty |

Examples:

```javascript
query_user_messages({
  pattern: "bug|fix",
  limit: 20
})

query_tools({
  tool: "exec_command",
  working_dir: "/path/to/project",
  limit: 50
})

query_token_usage({
  stats_first: true,
  limit: 20
})
```

### Two-Stage Query Tools

Use these for large sessions, targeted file selection, or custom jq:

| Tool | Purpose |
|------|---------|
| `get_session_directory` | Locate the session directory and aggregate file metadata |
| `inspect_session_files` | Inspect selected JSONL files |
| `execute_stage2_query` | Run jq-style filter/sort/transform on selected files |
| `get_session_metadata` | Return schema hints, file info, and query templates |

Example:

```javascript
const dir = await get_session_directory({scope: "project"})

const results = await execute_stage2_query({
  files: ["/path/to/session.jsonl"],
  filter: 'select(.type == "assistant") | select(.message | has("usage"))',
  transform: '{timestamp, usage: .message.usage}',
  limit: 20
})
```

`execute_stage2_query` runs after host normalization, so filters written for normalized message/tool records work on both Claude Code and Codex sessions.

### Analysis Tools

| Tool | Purpose |
|------|---------|
| `analyze_errors` | Aggregate tool errors by tool and type |
| `analyze_bugs` | Detect error-fix pairs and recurring bug patterns |
| `quality_scan` | Compute quality dimensions |
| `get_work_patterns` | Tool frequency, hourly activity, and context switches |
| `get_timeline` | Chronological session events |
| `get_tech_debt` | TODO/FIXME/HACK markers and unresolved errors |

Examples:

```javascript
get_work_patterns({
  working_dir: "/path/to/project"
})

analyze_errors({
  scope: "project",
  limit: 10
})
```

### Utility Tools

| Tool | Purpose |
|------|---------|
| `cleanup_temp_files` | Remove old temporary MCP output files |
| `list_capabilities` | List prompt/command capabilities |
| `get_capability` | Retrieve one capability by name/type |

## Standard Parameters

Most query tools accept:

| Parameter | Type | Description |
|-----------|------|-------------|
| `scope` | string | `project` (default) or `session` |
| `working_dir` | string | Override project path used for session lookup |
| `limit` | number | Maximum results; default is no limit |
| `stats_only` | boolean | Return aggregate statistics only |
| `stats_first` | boolean | Return stats followed by details |
| `output_format` | string | `jsonl` or `tsv` |
| `inline_threshold_bytes` | number | Threshold for inline vs file reference output |

Time-aware tools also accept RFC3339 `since` and `until`.

## Output Modes

The MCP server uses hybrid output:

- Inline responses for small results.
- `file_ref` responses for large results written to temporary JSONL files.

This lets the host inspect summaries first and read large result files only when needed.

## Verification

Ask either host:

```text
Which tools do I use most often?
Find user messages mentioning "refactor"
Show token usage for recent turns
```

For Codex-specific verification, `query_tools`, `query_user_messages`, `query_tool_errors`, `query_token_usage`, `get_work_patterns`, and `execute_stage2_query` are covered by `make test-e2e-codex`.

## Troubleshooting

### No sessions found

- Check `working_dir` points at the project whose transcripts you want.
- For Claude Code, verify `~/.claude/projects/<project-hash>/` exists.
- For Codex, verify `${CODEX_HOME:-$HOME/.codex}/sessions` contains JSONL files and that the transcript references the project path.

### Tool returns empty on Codex

Some tools query Claude Code-only record types:

- `query_file_snapshots`
- `query_summaries`
- `query_system_errors`

Empty results are expected for Codex unless Codex adds equivalent transcript records.

## See Also

- [MCP Query Tools Reference](mcp-query-tools.md)
- [Two-Stage Query Guide](two-stage-query-guide.md)
- [JSONL Schema Reference](../reference/jsonl-schema.md)
- [Installation Guide](../tutorials/installation.md)

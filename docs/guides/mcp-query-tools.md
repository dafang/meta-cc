# MCP Query Tools Reference

meta-cc exposes 21 MCP tools for Claude Code and Codex session analysis. Claude Code transcripts are read from `~/.claude/projects/`; Codex transcripts are read from `$CODEX_HOME/sessions` or `~/.codex/sessions` and normalized into the same message/tool schema before queries run.

## Host Support

| Host | Session root | Notes |
|------|--------------|-------|
| Claude Code | `~/.claude/projects/<project-hash>/` | Native schema already matches meta-cc message/tool records |
| Codex | `$CODEX_HOME/sessions` or `~/.codex/sessions` | Date-based JSONL files matched by project path in transcript content |

Codex normalization covers:

- `response_item.payload.type == "message"` -> user/assistant entries
- `function_call` and `custom_tool_call` -> assistant `tool_use` blocks
- `function_call_output` and `custom_tool_call_output` -> user `tool_result` blocks
- `event_msg.payload.type == "token_count"` -> assistant entries with `message.usage`

Claude-specific records without Codex equivalents remain host-specific: `file-history-snapshot`, top-level `summary`, and `system` records with `subtype: "api_error"`.

## Tool Catalog

### Convenience Queries

These tools scan the current project by default and accept `scope`, `working_dir`, `limit`, `stats_only`, `stats_first`, and output parameters where applicable.

| Tool | Purpose | Claude Code | Codex |
|------|---------|-------------|-------|
| `query_user_messages` | Search user messages by regex | Yes | Yes |
| `query_tools` | Query assistant tool calls | Yes | Yes |
| `query_tool_errors` | Query failed tool results | Yes | Yes |
| `query_token_usage` | Query assistant token usage | Yes | Yes |
| `query_conversation_flow` | Query user/assistant turns | Yes | Yes |
| `query_tool_blocks` | Query `tool_use` or `tool_result` blocks | Yes | Yes |
| `query_timestamps` | Query timestamped records | Yes | Yes |
| `query_system_errors` | Query Claude API system errors | Yes | Host-specific empty |
| `query_file_snapshots` | Query Claude file history snapshots | Yes | Host-specific empty |
| `query_summaries` | Query Claude session summaries | Yes | Host-specific empty |

Examples:

```javascript
query_user_messages({
  pattern: "refactor|migration",
  scope: "project",
  limit: 20
})

query_tools({
  tool: "exec_command",
  working_dir: "/path/to/project",
  limit: 50
})

query_tool_errors({
  scope: "session",
  stats_first: true
})

query_token_usage({
  stats_first: true,
  limit: 20
})
```

### Two-Stage Query Tools

Use these when you need file selection control or custom jq.

| Tool | Purpose |
|------|---------|
| `get_session_directory` | Locate session directory and aggregate metadata |
| `inspect_session_files` | Inspect selected JSONL files for record counts, time ranges, and samples |
| `execute_stage2_query` | Run jq-style filter/sort/transform on selected files |
| `get_session_metadata` | Return schema hints, file info, and query templates |

Example workflow:

```javascript
const dir = await get_session_directory({
  scope: "project"
})

const inspection = await inspect_session_files({
  files: ["/path/to/session.jsonl"],
  include_samples: true
})

const results = await execute_stage2_query({
  files: ["/path/to/session.jsonl"],
  filter: 'select(.type == "assistant") | select(.message | has("usage"))',
  transform: '{timestamp, usage: .message.usage}',
  limit: 20
})
```

`execute_stage2_query` receives normalized records, so common filters such as `select(.type == "user")`, `select(.type == "assistant")`, and tool block queries work on both Claude Code and Codex transcripts.

### Analysis Tools

| Tool | Purpose | Claude Code | Codex |
|------|---------|-------------|-------|
| `analyze_errors` | Aggregate tool errors by tool and type | Yes | Yes |
| `analyze_bugs` | Detect error-fix pairs and recurring bug patterns | Yes | Yes |
| `quality_scan` | Compute error, retry, diversity, and completion dimensions | Yes | Yes |
| `get_work_patterns` | Tool frequency, hourly activity, context switches | Yes | Yes |
| `get_timeline` | Chronological session events | Yes | Yes |
| `get_tech_debt` | TODO/FIXME/HACK markers and unresolved errors | Yes | Yes |

Example:

```javascript
get_work_patterns({
  working_dir: "/path/to/project"
})

analyze_errors({
  scope: "project",
  limit: 10
})
```

### Cleanup and Capabilities

| Tool | Purpose |
|------|---------|
| `cleanup_temp_files` | Remove old temporary MCP output files |
| `list_capabilities` | List prompt/command capabilities |
| `get_capability` | Retrieve a capability by name/type |

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

`query_user_messages` also supports `pattern`, `content_type`, length filters, `context_turns`, `group_by_session`, and RFC3339 `since` / `until`.

## Output Mode

meta-cc uses hybrid output:

- Small responses are returned inline.
- Large responses are written to a temporary file and returned as `file_ref`.

This keeps MCP responses usable for both short interactive questions and large project-level scans.

## Common Recipes

Find user prompts about a topic:

```javascript
query_user_messages({
  pattern: "release|deploy",
  limit: 20
})
```

Count tool usage:

```javascript
get_work_patterns({})
```

Inspect token usage:

```javascript
query_token_usage({
  stats_first: true
})
```

Run custom jq on selected files:

```javascript
execute_stage2_query({
  files: ["/path/to/session.jsonl"],
  filter: 'select(.type == "user" and (.message.content | type == "string"))',
  transform: '{timestamp, content: .message.content}',
  limit: 100
})
```

## See Also

- [MCP Server Guide](mcp.md)
- [Two-Stage Query Guide](two-stage-query-guide.md)
- [JSONL Schema Reference](../reference/jsonl-schema.md)

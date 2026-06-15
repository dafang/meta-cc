# Meta-CC Claude Code and Codex Integration Guide

This guide explains which meta-cc integration to use in Claude Code or Codex.

## Integration Surface

| Integration | Claude Code | Codex | Best for |
|-------------|-------------|-------|----------|
| MCP server | Yes | Yes | Session history queries and analysis |
| Prompt-library commands | `/prompt-find`, `/prompt-list`, `/prompt-show` | Native skills: `prompt-find`, `prompt-list`, `prompt-show` | Reusing saved prompts |
| Plugin metadata | Claude Code marketplace/archive plugin | Codex plugin metadata under `~/.codex/plugins/meta-cc/` | Host-native packaging |

The MCP server is the primary integration. It exposes 21 tools for querying and analyzing Claude Code and Codex session history through a provider-aware layer.

## Data Sources

| Provider | Source | Notes |
|----------|--------|-------|
| `claude` | `~/.claude/projects/<project-hash>/*.jsonl` | Default provider for backward compatibility. |
| `codex` | `${META_CC_CODEX_ROOT:-~/.codex}/state_5.sqlite` and rollout JSONL files referenced by `threads.rollout_path` | `~/.codex/history.jsonl` is intentionally not used. |
| `all` | Both providers | Results include a provider tag where applicable. |

Use `working_dir` to query a project path different from the current process working directory.

## Installation Paths

### Claude Code Marketplace

Use this when you only need Claude Code integration:

```bash
/plugin marketplace add yaleh/meta-cc
/plugin install meta-cc
```

Restart Claude Code. The plugin provides the MCP server configuration and prompt-library slash commands.

### Archive Install

Use this for Claude Code and Codex together:

```bash
./install.sh
```

The archive installer:

- copies `meta-cc-mcp` to `~/.local/bin/`
- installs Claude Code slash commands under `~/.claude/commands/`
- merges Claude Code MCP configuration into `~/.claude/mcp.json`
- installs Codex skills under `~/.codex/skills/`
- installs Codex plugin metadata under `~/.codex/plugins/meta-cc/`

Install one host only:

```bash
INSTALL_CLAUDE=0 ./install.sh  # Codex files only
INSTALL_CODEX=0 ./install.sh   # Claude Code files only
```

Install prompt-library commands and skills without the MCP binary:

```bash
./install-skills.sh
```

## MCP Usage

Ask naturally in either host:

```text
Which tools do I use most often?
Show my work patterns and peak hours
Find user messages mentioning "migration"
Analyze recent errors
Show token usage for recent assistant turns
```

For Codex-specific checks:

```text
Use provider=codex and show recent tool usage
Use provider=codex and find user messages mentioning "release"
Use provider=codex and show token usage
```

For cross-host analysis:

```text
Use provider=all and compare tool usage across Claude Code and Codex
Use provider=all and analyze recent error patterns
```

Common MCP calls:

```javascript
get_work_patterns({
  provider: "all",
  working_dir: "/path/to/project"
})
```

```javascript
query_tools({
  provider: "codex",
  tool: "exec_command",
  limit: 20
})
```

```javascript
query_user_messages({
  provider: "claude",
  pattern: "test|refactor",
  limit: 20
})
```

See [MCP Query Tools Reference](mcp-query-tools.md) for the full catalog.

## Prompt Library

The prompt library lives in the current project's `.meta-cc/prompts/library/` directory.

Claude Code slash commands:

```text
/prompt-list
/prompt-list sort=usage
/prompt-find release checklist
/prompt-show phase-execution-001
```

Codex skills:

```text
$prompt-list
$prompt-list sort=usage
$prompt-find release checklist
$prompt-show phase-execution-001
```

The commands and skills parse the same Markdown files and frontmatter fields: `id`, `title`, `category`, `keywords`, `usage_count`, `updated`, and `status`.

## Choosing A Method

| Task | Use |
|------|-----|
| Ask about session history, tools, errors, or token usage | MCP tools |
| Browse saved prompts in Claude Code | `/prompt-list` |
| Browse saved prompts in Codex | `$prompt-list` |
| Search reusable prompts in Claude Code | `/prompt-find <keywords>` |
| Search reusable prompts in Codex | `$prompt-find <keywords>` |
| View a saved prompt in Claude Code | `/prompt-show <id>` |
| View a saved prompt in Codex | `$prompt-show <id>` |
| Validate Codex support in development | `make test-e2e-codex` |

## Verification

### Claude Code

1. Restart Claude Code.
2. Ask: `Which tools do I use most often?`
3. Run: `/prompt-list`

### Codex

1. Restart Codex.
2. Ask: `Use provider=codex and show my work patterns`
3. Run: `$prompt-list`

### Development E2E

```bash
make test-e2e-codex
```

This creates an isolated Codex home, installs the Codex files, creates SQLite and rollout fixtures, and calls the MCP server over JSON-RPC with `provider: "codex"`.

## Troubleshooting

### MCP Tools Are Not Called

- Confirm the host has loaded the MCP server configuration.
- Ask more directly: `Use the meta-cc MCP server to show recent tool usage`.
- Check the binary is available:

```bash
which meta-cc-mcp
```

### Claude Code Prompt Commands Not Found

```bash
ls ~/.claude/commands/prompt-list.md
ls ~/.claude/commands/prompt-find.md
ls ~/.claude/commands/prompt-show.md
```

Restart Claude Code after installing.

### Codex Skills Not Found

```bash
ls ~/.codex/skills/prompt-list/SKILL.md
ls ~/.codex/skills/prompt-find/SKILL.md
ls ~/.codex/skills/prompt-show/SKILL.md
```

Restart Codex after installing.

### Codex MCP Query Returns No Sessions

Check the Codex index:

```bash
sqlite3 ~/.codex/state_5.sqlite \
  "select cwd, rollout_path from threads order by updated_at desc limit 10;"
```

If Codex data is stored outside `~/.codex`, set:

```bash
export META_CC_CODEX_ROOT=/path/to/codex-home
```

The `cwd` in the `threads` table must match the project path you query with `working_dir`.

## Related Documentation

- [Installation Guide](../tutorials/installation.md)
- [Examples](../tutorials/examples.md)
- [MCP Guide](mcp.md)
- [MCP Query Tools Reference](mcp-query-tools.md)
- [JSONL Schema Reference](../reference/jsonl-schema.md)

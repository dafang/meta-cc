# meta-cc

[![CI](https://github.com/yaleh/meta-cc/actions/workflows/ci.yml/badge.svg)](https://github.com/yaleh/meta-cc/actions)
[![License](https://img.shields.io/github/license/yaleh/meta-cc)](LICENSE)
[![Release](https://img.shields.io/github/v/release/yaleh/meta-cc)](https://github.com/yaleh/meta-cc/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yaleh/meta-cc)](go.mod)
[![Plugin Marketplace](https://img.shields.io/badge/Claude_Code-Plugin_Marketplace-blue)](https://github.com/yaleh/meta-cc)

**Meta-cognition tool for Claude Code** - Analyze session history, detect patterns, optimize workflows.

> **Note**: Skills and agents from previous versions have been moved to [yaleh/baime](https://github.com/yaleh/baime). meta-cc 3.0.0 focuses exclusively on session history analysis via MCP tools.

---

## What is meta-cc?

meta-cc helps you understand and improve your Claude Code workflows through:

- **Autonomous analysis** - Claude automatically queries session data via MCP tools
- **21 MCP tools** - Error analysis, quality scanning, work patterns, timelines, bug detection, and more
- **Prompt library** - Save, search, and reuse optimized prompts with 3 slash commands

**Zero configuration required** - works out of the box with Claude Code.

---

## Quick Install

### Method 1: Plugin Marketplace (Recommended)

```bash
/plugin marketplace add yaleh/meta-cc
/plugin install meta-cc
```

Restart Claude Code — that's it. The MCP server is automatically configured via `.mcp.json` bundled in the plugin.

The meta-cc plugin includes:
- **3 Slash Commands** - `/prompt-find`, `/prompt-list`, `/prompt-show` for prompt library management
- **21 MCP Tools** - Session data analysis with two-stage query architecture (v2.1)

### Method 2: Archive Install (Alternative)

**Full install** (MCP server + slash commands):

```bash
# Linux/macOS (one-liner)
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-linux-amd64.tar.gz | tar xz
cd meta-cc-plugin-linux-amd64
./install.sh
```

The archive installer copies the binary and integration files, and automatically merges the MCP server configuration into `~/.claude/mcp.json`.

**Slash commands only** (no binary required, any platform):

```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-skills-latest.tar.gz | tar xz
cd meta-cc-skills-*/
./install-skills.sh
```

**MCP server binary only** (for CI/Docker/PATH installs):

```bash
# Download the bare binary for your platform, e.g. Linux amd64:
curl -LO https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-mcp-linux-amd64
chmod +x meta-cc-mcp-linux-amd64
INSTALL_DIR=~/.local/bin bash scripts/install/install-mcp.sh meta-cc-mcp-linux-amd64
```

**Other platforms**: See [Installation Guide](docs/tutorials/installation.md) for macOS (Apple Silicon), Windows, and manual installation.

### Verify Installation

In Claude Code, ask naturally:

```
"Show me all Bash errors in this project"
"Which tools do I use most often?"
"Find user messages mentioning 'refactor'"
```

**Troubleshooting**: See [Installation Guide](docs/tutorials/installation.md#troubleshooting) for common issues.

---

## Quick Start

### Autonomous Analysis (MCP)

Just ask Claude naturally - MCP tools are invoked automatically:

```
"Show me all Bash errors in this project"
"Find user messages mentioning 'refactor'"
"Which tools do I use most often?"
"Scan session quality and show me scores"
"Show my work patterns and peak hours"
"Find bug fix pairs in my session"
```

**Unified query interface with 21 MCP tools and jq filtering**:

```javascript
// Core query tool - unified interface
query({
  resource: "tools",
  filter: {tool_status: "error"},
  jq_filter: '.[] | select(.tool_name == "Bash")'
})

// Convenience tools - optimized for common queries
query_tool_errors({limit: 10})
query_token_usage({stats_first: true})
query_conversation_flow({scope: "session"})

// Raw jq - maximum flexibility for power users
query_raw({
  jq_expression: '.[] | group_by(.tool_name) | map({tool: .[0].tool_name, count: length})'
})

// New analysis tools (3.0.0)
analyze_errors({})          // Aggregate errors by tool and type
quality_scan({})             // Compute error/retry/diversity scores
get_work_patterns({})        // Hourly activity and context switches
get_timeline({})             // Chronological session events
analyze_bugs({})             // Error-fix pairs and recurring patterns
get_tech_debt({})            // TODO/FIXME markers and unresolved errors
```

**Key Features**:
- **Hybrid Output Mode**: Auto-switches between inline (<8KB) and file_ref (≥8KB)
- **jq Integration**: Native jq filtering for complex queries
- **No Limits by Default**: Returns all results, relies on hybrid mode
- **21 Tools**: 2 core + 8 convenience + 7 legacy + 3 utility + 6 analysis tools

**Resources**:
- [MCP Query Tools Reference](docs/guides/mcp-query-tools.md) - Complete tool documentation
- [MCP Query Cookbook](docs/examples/mcp-query-cookbook.md) - 25+ practical examples
- [MCP v2.0 Migration Guide](docs/guides/mcp-v2-migration.md) - Upgrade from v1.x

### Prompt Library (Slash Commands)

Save and reuse your best prompts with 3 built-in commands:

```bash
/prompt-find phase execution      # Search by keywords
/prompt-list sort=usage           # Browse all (sorted by use)
/prompt-show phase-execution-001  # View full prompt details
```

---

## Documentation

### Getting Started

- **[Installation Guide](docs/tutorials/installation.md)** - Detailed setup for all platforms
- **[Quick Start Tutorial](docs/tutorials/examples.md)** - Step-by-step examples
- **[Troubleshooting](docs/guides/troubleshooting.md)** - Common issues and solutions

### Integration

- **[MCP Guide](docs/guides/mcp.md)** - Complete MCP tool reference (21 tools)
- **[Integration Guide](docs/guides/integration.md)** - MCP and Slash Commands

### Advanced

- **[JSONL Reference](docs/reference/jsonl.md)** - Output format and jq patterns
- **[Feature Overview](docs/reference/features.md)** - Advanced features and capabilities

### Development

- **[Contributing Guide](CONTRIBUTING.md)** - Development workflow and guidelines
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community standards

### For Claude Code

- **[CLAUDE.md](CLAUDE.md)** - Project instructions for Claude Code development
- **[Design Principles](docs/core/principles.md)** - Core constraints and architecture
- **[Implementation Plan](docs/core/plan.md)** - Development roadmap

**Complete documentation map**: [DOCUMENTATION_MAP.md](docs/DOCUMENTATION_MAP.md)

---

## Key Features

- **21 MCP tools** - Autonomous session data analysis with two-stage query architecture
- **3 Slash Commands** - Prompt library management (`/prompt-find`, `/prompt-list`, `/prompt-show`)
- **Advanced analytics** - jq-based filtering, aggregation, time series
- **Error analysis** - Aggregate tool errors by name and type
- **Quality scanning** - Error/retry/diversity/completion dimensions
- **Work pattern detection** - Tool frequency, hourly activity, context switches
- **Timeline visualization** - Chronological session events as JSON
- **Bug detection** - Error-fix pairs and recurring patterns
- **Tech debt tracking** - TODO/FIXME markers and unresolved errors
- **File operation tracking** - Identify hotspots and churn
- **Zero dependencies** - Single binary MCP server
- **Prompt Learning System** - Save, search, and reuse optimized prompts with project-specific intelligence

---

## Development

### Prerequisites

- Go 1.21 or later
- make

### Build from Source

```bash
git clone https://github.com/yaleh/meta-cc.git
cd meta-cc
make build
```

### Development Workflow (3-Tier)

Use the optimized 3-tier workflow for efficient development:

```bash
make dev           # Quick dev build (format + build, <10s)
make commit        # Pre-commit validation (workspace + tests, <60s)
make push          # Full check before push (all checks + lint, <120s)
```

**Workflow**:
1. **Iterate**: Use `make dev` for fast feedback during development
2. **Commit**: Run `make commit` to validate before committing
3. **Push**: Run `make push` for full verification before pushing to remote

### Run Tests

```bash
make test           # Unit tests (fast)
make test-all       # Including E2E tests (~30s)
make test-coverage  # With coverage report
```

**Coverage Requirement**: Maintain ≥80% test coverage for all code changes.

---

## Platform Support

- Linux (amd64, arm64)
- macOS (Intel, Apple Silicon)
- Windows (amd64)

---

## Contributing

We welcome contributions! Please see:

- **[Contributing Guide](CONTRIBUTING.md)** - Development process and guidelines
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Community standards

---

## License

MIT License - See [LICENSE](LICENSE) file for details.

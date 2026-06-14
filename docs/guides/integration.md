# Meta-CC Claude Code and Codex Integration Guide

A practical guide to choosing and using the right integration method for your meta-cognition workflow.

## Overview

meta-cc provides host-native integrations for Claude Code and Codex, all powered by the MCP server:

- **MCP Tools**: Direct access to query and analysis tools through the host's autonomous tool calling
- **Claude Code Slash Commands**: Quick prompt-library workflows (`/prompt-find`, `/prompt-list`, `/prompt-show`)
- **Codex Skills**: Native Codex prompt-library workflows (`prompt-find`, `prompt-list`, `prompt-show`)

### Quick Comparison

| Feature | MCP Server | Claude Commands | Codex Skills |
|---------|-----------|-----------------|--------------|
| **Invocation** | Automatic host tool calls | Manual (`/command`) | Natural-language skill trigger |
| **Context** | Main conversation | Main conversation | Main conversation |
| **Parameters** | Structured schema | Positional (`$1`, `$2`) | Natural language |
| **Best for** | Data queries and analysis | Repeated Claude Code workflows | Repeated Codex workflows |

**👉 [Jump to Decision Framework](#decision-framework)** to find the best method for your task.

---

## Quick Start: MCP Server Setup

For most users, the **MCP Server** provides the best balance of power and convenience.

### Step 1: Install and Build

```bash
git clone https://github.com/yaleh/meta-cc.git
cd meta-cc
make build-mcp
cp meta-cc-mcp ~/.local/bin/
```

### Step 2: Configure Claude Code

```bash
# Quick setup
claude mcp add meta-cc --transport stdio meta-cc-mcp --scope user

# Or manual configuration - edit claude_desktop_config.json
{
  "mcpServers": {
    "meta-cc": {
      "type": "stdio",
      "command": "meta-cc-mcp",
      "args": []
    }
  }
}
```

### Step 2b: Configure Codex

Archive installs include `.codex-plugin/plugin.json`, `.codex-mcp.json`, and `skills/`. The installer copies Codex skills to `~/.codex/skills/` by default:

```bash
CODEX_HOME=~/.codex ./install-skills.sh
```

For MCP, use the bundled `.codex-mcp.json` as the server template if your Codex plugin loader does not import it automatically.

### Step 3: Test Integration

```
@meta-cc get_session_stats
@meta-cc query_tools --limit=10
@meta-cc query_user_messages --pattern=".*error.*"
```

**You're ready!** Claude Code or Codex can now call meta-cc tools when you ask questions about your session data.

---

## Core Differences

### Context Isolation

**Main Conversation (MCP & Slash)**:
- ✅ Full history access - can reference previous messages
- ✅ Context continuity - Claude remembers earlier analysis
- ⚠️ Context pollution - tool calls accumulate in history
- ⚠️ Token consumption - each call adds to context length

**Independent Context (Subagent)**:
- ✅ No pollution - main conversation stays focused
- ✅ Specialized reasoning - dedicated context for deep analysis
- ❌ No shared history - each invocation starts fresh
- ❌ Limited continuity - cannot reference main conversation details

**When it matters**:
- Need to correlate with earlier conversation → MCP/Slash (main context)
- Deep multi-step analysis → Subagent (independent context)
- Keep main conversation clean → Subagent

### Invocation Models

**MCP - Autonomous Tool Selection**:
```
User: "What's my session error rate?"
  ↓
Claude: [Decides to call get_session_stats]
  ↓
Result: {"ErrorRate": 0.0, "ErrorCount": 0, ...}
  ↓
Response: "Your error rate is 0%, with 0 errors detected."
```

**Pros**: Natural UX, no command memorization, flexible
**Cons**: Less predictable, less control over parameters

**Slash Commands - Explicit Execution**:
```
User: /meta-stats
  ↓
Executes: meta-cc parse stats --output md
  ↓
Output: [Formatted markdown table]
```

**Pros**: Fully predictable, fast, scriptable
**Cons**: Must remember commands, limited flexibility

**Subagent - Delegated Conversation**:
```
User: "@meta-coach I feel stuck, help analyze my workflow"
  ↓
@meta-coach: "Let me gather data first..."
  ↓
[Multi-turn dialogue with tool calls and reasoning]
  ↓
@meta-coach: "Here's what I found and recommend..."
```

**Pros**: Conversational, adaptive, keeps main chat clean
**Cons**: Slower, no memory between sessions

### Execution Models

**MCP - Data Source**:
- Raw data retrieval
- Main conversation Claude does interpretation
- Single tool call per invocation
- Can combine with other tools

**Slash Commands - Pre-Programmed Workflow**:
- Pre-defined logic (Bash scripts)
- Can include multiple meta-cc commands
- Output is pre-formatted (markdown/json)
- Claude's role is minimal (display + optional context)

**Subagent - Independent Analyst**:
- Has own personality and methodology
- Can reason across multiple tool calls
- Supports back-and-forth dialogue
- Returns only high-level summary to main conversation

---

## Decision Framework

### Task Type Decision Tree

```
┌─────────────────────────────────────┐
│ What do you need to do?             │
└─────────────────┬───────────────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
    [Simple       [Complex multi-step
     data query]   analysis]
        │                   │
        ├── Is it a one-time    │
        │   question?            │
        │   YES → MCP            │
        │   NO ↓                 │
        │                        │
        ├── Will you repeat      ├── Do you know exactly
        │   this often?          │   what steps to take?
        │   YES → Slash Command  │   YES → Slash Command
        │   NO → MCP             │   NO ↓
                                 │
                                 ├── Do you need guidance
                                 │   or exploration?
                                 │   YES → Subagent
                                 │   NO → MCP (multiple calls)
```

**Quick decision rules**:

1. **Just want data, ask naturally** → MCP
2. **Repeat the same workflow often** → Slash Command
3. **Don't know what's wrong, need help** → Subagent
4. **Multi-step with known steps** → Slash Command
5. **Multi-step with unknown steps** → Subagent

### Use Case Scenarios Matrix

| Scenario | Best Method | Why | Alternative |
|----------|-------------|-----|-------------|
| **Quick stats check** | MCP or Slash | Fast, no ceremony | Either works |
| **Daily workflow automation** | Slash Command | Predictable, repeatable | - |
| **Debugging repeated errors** | Subagent | Needs exploration | Slash + manual |
| **Cross-project comparison** | MCP Tools | Native support | - |
| **Learning optimization** | Subagent | Educational, conversational | - |
| **Ad-hoc exploration** | MCP | Natural questions | - |
| **Implementing fixes** | Subagent | Can create files/configs | Manual |

### Anti-Patterns

**❌ Don't Use MCP When**:
1. You need exactly the same analysis every time → Use Slash Command
2. You need multi-step reasoning → Use Subagent
3. Building automation/scripts → Use meta-cc CLI directly

**❌ Don't Use Slash Commands When**:
1. Workflow isn't well-defined yet → Use Subagent to explore first
2. Need adaptive behavior based on results → Use Subagent
3. Only use it once → Just ask Claude (uses MCP)

**❌ Don't Use Subagent When**:
1. Just need quick data → Use MCP tools or Slash (faster)
2. Need same exact output format → Use Slash Command
3. Want to reference main conversation → Stay in main context (MCP)
4. Track progress across sessions → MCP query tools handle multi-session data

---

## Best Practices

### Combining Integration Methods

The three methods work together:

**Pattern 1: Slash Command → Calls MCP**
- Use case: Fixed workflow leveraging MCP data access
- Benefit: Combines predictability with seamless integration

**Pattern 2: Subagent → Calls MCP Tools**
- Use case: Complex analysis requiring reasoning and data
- Benefit: Subagent's reasoning + MCP tools' structured data

**Pattern 3: MCP as Foundation, Others as Shortcuts**
- Strategy: Start with MCP → Identify common patterns → Create Slash Commands → Add Subagent for guidance
- Benefit: Organic growth based on actual usage

### Minimizing Context Pollution

**Solutions**:

1. **Use Slash Commands for bulk operations**
   - Bad: Claude calls MCP 20 times for different sessions
   - Good: `/meta-compare-all` (script loops)

2. **Use Subagent for exploratory deep dives**
   - Bad: Long back-and-forth in main conversation
   - Good: `@meta-coach` (keeps main clean)

3. **Be explicit about output format**
   - Better: "Get stats as JSON" (MCP with output_format)
   - Good: `/meta-stats` (pre-configured)

### Choosing Output Format

**JSON** - Best for:
- Programmatic processing, piping to tools, precision

**Markdown** - Best for:
- Human readability, slash commands, interpretation

**Recommendation by method**:
- **MCP**: JSON (Claude interprets)
- **Slash**: Markdown (better UX)
- **Subagent**: JSON (subagent reasons over it)

### Creating Custom Integrations

**When to Create Slash Command**:
1. Run same meta-cc command >3 times
2. Workflow has clear, fixed steps
3. Want consistent output format

**Template**:
```markdown
---
name: my-custom-check
description: [Your description]
---

# My Custom Check

Query session data using MCP tools.

## Instructions

Use the following MCP tools:
- query_tools for tool call analysis
- get_session_stats for statistics
- query_user_messages for message search

Analyze the results and provide insights.
```

---

## Troubleshooting

### MCP Tools Not Being Called

**Symptoms**: Ask for stats but Claude doesn't use MCP tool

**Solutions**:
- Question too indirect → Rephrase explicitly: "Get my session statistics"
- MCP server not connected → Check `claude mcp list`
- Tool description too vague → Update tool schema

### Prompt Commands Or Skills Not Found

**Claude Code checklist**:
1. File exists at `.claude/commands/prompt-list.md`
2. Restarted Claude Code after installing
3. Using `/prompt-list`, `/prompt-find`, or `/prompt-show`

**Codex checklist**:
1. File exists at `.codex/skills/prompt-list/SKILL.md`
2. Restarted Codex after installing
3. Asking Codex to use `prompt-list`, `prompt-find`, or `prompt-show`

### Subagent Not Understanding Context

**Remember**: Subagent has independent context!

**Solution**:
```
Don't: "@meta-coach why did that error happen?"
       (doesn't know which error)

Do: "@meta-coach I just got an error in my auth module.
     Can you analyze recent errors and help debug?"
```

---

## Quick Reference

### Command Cheat Sheet

**MCP Tools** (mention naturally in conversation):
```
"Get session statistics"          → get_session_stats
"Analyze error patterns"          → analyze_errors
"Show tool usage"                 → extract_tools
```

**Slash Commands**:
```bash
/meta-stats              # Session overview
/meta-errors [window]    # Error analysis (default=20)
/meta-timeline [limit]   # Chronological tool calls (default=50)
/meta-compare <path>     # Compare with another project
/meta-help               # Show all commands
```

**Subagent**:
```bash
@meta-coach [question]   # Start analysis conversation

# Example questions:
@meta-coach How's my workflow efficiency?
@meta-coach I'm stuck, help analyze what's wrong
@meta-coach Compare current session with best practices
```

### Decision Quick Lookup

| I want to... | Use this |
|--------------|----------|
| Check error rate quickly | MCP or `/meta-stats` |
| Analyze repeated errors | `/meta-errors 30` |
| Understand why I'm inefficient | `@meta-coach` |
| Compare two projects | `/meta-compare <path>` |
| Get help optimizing | `@meta-coach` |
| See recent tool usage | MCP or `/meta-timeline` |
| Automate daily checks | Create Slash Command |
| Explore unknown problem | `@meta-coach` |
| Get exact same report | Slash Command |

---

## Next Steps

### For New Users

1. **Start with MCP**: Ask questions naturally
2. **Learn Slash Commands**: Use `/meta-help`
3. **Try @meta-coach**: When you need guidance

### For Advanced Users

1. **Create Custom Slash Commands**: Automate workflows
2. **Extend @meta-coach**: Add domain-specific analysis
3. **Combine Methods**: Use MCP + Slash + Subagent together

---

## Related Documentation

- **[meta-cc README](../../README.md)**: Installation and CLI reference
- **[Examples & Usage](../tutorials/examples.md)**: Step-by-step setup guides
- **[Troubleshooting Guide](troubleshooting.md)**: Common issues and solutions
- **[MCP Output Modes](../archive/mcp-output-modes.md)**: Detailed MCP usage
- **[Technical Proposal](../architecture/proposals/meta-cognition-proposal.md)**: Architecture deep dive

---

*Last updated: 2025-10-12*

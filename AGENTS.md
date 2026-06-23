# meta-cc AGENTS

This file is read by Codex (and Claude Code) to find capability anchors.

## meta-cc Session Analysis & Reflection

For analyzing Claude Code session history, debugging recurring errors, reflecting on your workflow patterns, or querying past messages and tool calls, use the **meta-cc-insights** skill and the meta-cc MCP tools.

Quick triggers to invoke meta-cc:
- "Why do I keep hitting this error?" → use `analyze_bugs`
- "How did I approach this last time?" → use `query_user_messages` or `query_tools`
- "Let's reflect on this phase" → use `quality_scan`
- "Show me what I did today" → use `get_timeline`

All tools support `provider: "claude"`, `provider: "codex"`, or `provider: "all"` to query across both platforms.

For a full reference, see the `meta-cc-insights` skill or run `/prompt-list` to browse the prompt library.

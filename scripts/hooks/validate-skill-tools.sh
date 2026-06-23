#!/bin/bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

SKILL_FILE="${SKILL_FILE:-plugin-src/skills/meta-cc-insights/SKILL.md}"

ALL_TOOLS="query_tool_errors query_token_usage query_conversation_flow query_system_errors query_file_snapshots query_timestamps query_summaries query_tool_blocks query_tools query_user_messages cleanup_temp_files get_session_directory inspect_session_files execute_stage2_query analyze_errors analyze_bugs quality_scan get_work_patterns get_session_metadata get_timeline get_tech_debt"

MISSING=""
for tool in $ALL_TOOLS; do
  if ! grep -q "$tool" "$SKILL_FILE"; then
    MISSING="$MISSING $tool"
  fi
done

if [ -n "$MISSING" ]; then
  echo "ERROR: SKILL missing tools:$MISSING"
  exit 1
fi

echo "✓ SKILL includes all 21 tools"

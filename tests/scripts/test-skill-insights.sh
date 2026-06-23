#!/bin/bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

echo "=== test-skill-insights: Phase A ==="

# Phase A assertions
test -f plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: SKILL.md not found"; exit 1; }
echo "✓ SKILL.md exists"

grep -qE '^name:[[:space:]]*meta-cc-insights' plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: name not found"; exit 1; }
echo "✓ name: meta-cc-insights"

grep -qE '^description:[[:space:]]*\S' plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: description missing"; exit 1; }
echo "✓ description exists"

! grep -qE '^description:.*调用 meta-cc' plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: description uses invocation, not trigger"; exit 1; }
echo "✓ description uses trigger language"

# Check representative tools
for tool in query_tool_errors quality_scan analyze_bugs get_work_patterns get_timeline execute_stage2_query; do
  grep -q "$tool" plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: $tool not mentioned"; exit 1; }
done
echo "✓ representative tools present"

grep -q 'provider' plugin-src/skills/meta-cc-insights/SKILL.md || { echo "FAIL: provider not mentioned"; exit 1; }
echo "✓ provider mentioned"

echo "=== test-skill-insights: Phase A PASS ==="

echo -e "\n=== test-skill-insights: Phase B ==="

test -x scripts/hooks/validate-skill-tools.sh || { echo "FAIL: validate-skill-tools not executable"; exit 1; }
echo "✓ validate-skill-tools executable"

bash scripts/hooks/validate-skill-tools.sh || { echo "FAIL: validate-skill-tools failed"; exit 1; }
echo "✓ validate-skill-tools passes"

echo "✓ Phase B PASS"

echo -e "\n=== test-skill-insights: Phase C ==="

grep -q '"skills"' plugin-src/.claude-plugin/plugin.json || { echo "FAIL: Claude manifest missing skills"; exit 1; }
echo "✓ Claude manifest has skills"

jq . plugin-src/.claude-plugin/plugin.json > /dev/null || { echo "FAIL: Claude manifest invalid JSON"; exit 1; }
jq . .claude-plugin/marketplace.json > /dev/null || { echo "FAIL: marketplace invalid JSON"; exit 1; }
echo "✓ JSON validity"

grep -q 'meta-cc-insights' scripts/install/install-skills.sh || { echo "FAIL: install-skills missing meta-cc-insights"; exit 1; }
echo "✓ install-skills includes new skill"

grep -qE 'skills' scripts/install/install-skills.sh || { echo "FAIL: install-skills missing Claude skill install"; exit 1; }
echo "✓ install-skills installs skills"

grep -q 'meta-cc-insights' scripts/sync-plugin-files.sh || { echo "FAIL: sync-plugin-files missing meta-cc-insights"; exit 1; }
echo "✓ sync-plugin-files checks new skill"

echo "✓ Phase C PASS"

echo -e "\n=== test-skill-insights: Phase D ==="

grep -q 'meta-cc-insights' scripts/ci/smoke-tests-skills.sh || { echo "FAIL: smoke-tests missing meta-cc-insights"; exit 1; }
echo "✓ smoke-tests includes new skill"

grep -q 'meta-cc-insights' AGENTS.md || { echo "FAIL: AGENTS.md missing meta-cc-insights"; exit 1; }
echo "✓ AGENTS.md has anchor"

test -f docs/guides/capability-awareness.md || { echo "FAIL: docs/guides/capability-awareness.md missing"; exit 1; }
echo "✓ docs exist"

grep -q 'Codex' docs/guides/capability-awareness.md && grep -q 'Claude' docs/guides/capability-awareness.md || { echo "FAIL: docs must mention both Claude and Codex"; exit 1; }
echo "✓ docs mention both providers"

echo "=== test-skill-insights: ALL PASS ==="

#!/bin/bash
# Smoke tests for skills package
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

test_result() {
    if [ $1 -eq 0 ]; then
        echo "  ✓ PASS: $2"
    else
        echo "  ✗ FAIL: $2"
        FAILED=true
    fi
}

echo "=== meta-cc skills package smoke tests ==="
echo ""

FAILED=false

# Create a temporary work dir
WORKDIR=$(mktemp -d /tmp/meta-cc-smoke-XXXXXX)
trap 'rm -rf "$WORKDIR"' EXIT

cd "$PROJECT_ROOT"

# 1. Check that plugin-src/skills/ directory exists
echo "[1/4] Checking plugin-src/ structure..."
test -d plugin-src/skills
test_result $? "plugin-src/skills/ exists"

# Check all skills exist
for skill in prompt-find prompt-list prompt-show meta-cc-insights; do
    test -f plugin-src/skills/$skill/SKILL.md
    test_result $? "plugin-src/skills/$skill/SKILL.md exists"
done
echo ""

# 2. Check that .codex-plugin/plugin.json has skills reference
echo "[2/4] Checking Codex plugin.json..."
grep -q '"skills":' plugin-src/.codex-plugin/plugin.json
test_result $? "Codex plugin.json declares skills"
jq . plugin-src/.codex-plugin/plugin.json >/dev/null
test_result $? "Codex plugin.json is valid JSON"
echo ""

# 3. Check that .claude-plugin/plugin.json has skills reference
echo "[3/4] Checking Claude plugin.json..."
grep -q '"skills":' plugin-src/.claude-plugin/plugin.json
test_result $? "Claude plugin.json declares skills"
jq . plugin-src/.claude-plugin/plugin.json >/dev/null
test_result $? "Claude plugin.json is valid JSON"
echo ""

# 4. Verify that the validate-skill-tools script runs
echo "[4/4] Checking validate-skill-tools script..."
test -x scripts/hooks/validate-skill-tools.sh
test_result $? "validate-skill-tools exists and executable"
bash scripts/hooks/validate-skill-tools.sh
test_result $? "validate-skill-tools passes"
echo ""

if [ "$FAILED" = true ]; then
    echo "❌ Smoke tests FAILED"
    exit 1
else
    echo "✅ All smoke tests PASSED"
fi

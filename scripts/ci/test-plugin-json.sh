#!/bin/bash
# Validate plugin.json and marketplace.json consistency
# Used by CI and pre-commit hooks
set -e

ERRORS=0

check() {
    local desc="$1"
    local result="$2"
    if [ "$result" = "pass" ]; then
        echo "  ✓ PASS: $desc"
    else
        echo "  ✗ FAIL: $desc"
        ERRORS=$((ERRORS + 1))
    fi
}

PLUGIN_JSON=".claude/.claude-plugin/plugin.json"
MARKETPLACE_JSON=".claude-plugin/marketplace.json"

echo "=== Plugin JSON Validation ==="
echo ""

# 1. plugin.json exists and is valid JSON
if [ ! -f "$PLUGIN_JSON" ]; then
    check "plugin.json exists at $PLUGIN_JSON" "fail"
    ERRORS=$((ERRORS + 1))
else
    check "plugin.json exists at $PLUGIN_JSON" "pass"
    if jq . "$PLUGIN_JSON" > /dev/null 2>&1; then
        check "plugin.json is valid JSON" "pass"
    else
        check "plugin.json is valid JSON" "fail"
    fi
fi

# 2. marketplace.json does NOT contain strict: false
if jq -e '.plugins[0].strict == false' "$MARKETPLACE_JSON" > /dev/null 2>&1; then
    check "marketplace.json does NOT have strict: false" "fail"
else
    check "marketplace.json does NOT have strict: false" "pass"
fi

# 3. Version parity
if [ -f "$PLUGIN_JSON" ]; then
    MARKET_VER=$(jq -r '.plugins[0].version' "$MARKETPLACE_JSON")
    PLUGIN_VER=$(jq -r '.version' "$PLUGIN_JSON")
    if [ "$MARKET_VER" = "$PLUGIN_VER" ]; then
        check "Version parity: marketplace=$MARKET_VER, plugin=$PLUGIN_VER" "pass"
    else
        check "Version parity: marketplace=$MARKET_VER plugin=$PLUGIN_VER" "fail"
    fi
fi

# 4. plugin.json declares expected content counts
if [ -f "$PLUGIN_JSON" ]; then
    CMD_COUNT=$(jq '.commands | length' "$PLUGIN_JSON" 2>/dev/null || echo 0)
    AGENT_COUNT=$(jq '.agents | length' "$PLUGIN_JSON" 2>/dev/null || echo 0)
    SKILL_COUNT=$(jq '.skills | length' "$PLUGIN_JSON" 2>/dev/null || echo 0)

    if [ "$CMD_COUNT" -eq 4 ]; then
        check "plugin.json declares 4 commands (got $CMD_COUNT)" "pass"
    else
        check "plugin.json declares 4 commands (got $CMD_COUNT)" "fail"
    fi

    if [ "$AGENT_COUNT" -eq 5 ]; then
        check "plugin.json declares 5 agents (got $AGENT_COUNT)" "pass"
    else
        check "plugin.json declares 5 agents (got $AGENT_COUNT)" "fail"
    fi

    if [ "$SKILL_COUNT" -eq 18 ]; then
        check "plugin.json declares 18 skills (got $SKILL_COUNT)" "pass"
    else
        check "plugin.json declares 18 skills (got $SKILL_COUNT)" "fail"
    fi
fi

# 5. marketplace.json commands array matches plugin.json
if [ -f "$PLUGIN_JSON" ]; then
    MKT_CMD_COUNT=$(jq '.plugins[0].commands | length' "$MARKETPLACE_JSON" 2>/dev/null || echo 0)
    if [ "$MKT_CMD_COUNT" -eq 4 ]; then
        check "marketplace.json declares 4 commands (got $MKT_CMD_COUNT)" "pass"
    else
        check "marketplace.json declares 4 commands (got $MKT_CMD_COUNT)" "fail"
    fi
fi

echo ""
if [ "$ERRORS" -eq 0 ]; then
    echo "✓ All plugin JSON validations passed"
    exit 0
else
    echo "✗ $ERRORS validation(s) failed"
    exit 1
fi

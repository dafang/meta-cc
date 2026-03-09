#!/bin/bash
# Pre-commit hook: Validate marketplace.json and plugin.json schema
# Ensures both files have valid JSON structure and consistent versions

set -e

ERRORS=0

MARKETPLACE_JSON=".claude-plugin/marketplace.json"
PLUGIN_JSON=".claude-plugin/plugin.json"

# Validate marketplace.json
if jq -e '.plugins[0].version' "$MARKETPLACE_JSON" >/dev/null 2>&1; then
    : # valid
else
    echo "ERROR: Invalid marketplace.json structure"
    echo "Required field .plugins[0].version is missing or invalid"
    ERRORS=$((ERRORS + 1))
fi

# Validate plugin.json exists and is valid
if [ -f "$PLUGIN_JSON" ]; then
    if ! jq . "$PLUGIN_JSON" > /dev/null 2>&1; then
        echo "ERROR: $PLUGIN_JSON is not valid JSON"
        ERRORS=$((ERRORS + 1))
    fi

    # Check version parity
    MARKET_VER=$(jq -r '.plugins[0].version' "$MARKETPLACE_JSON")
    PLUGIN_VER=$(jq -r '.version' "$PLUGIN_JSON")
    if [ "$MARKET_VER" != "$PLUGIN_VER" ]; then
        echo "ERROR: Version mismatch: marketplace=$MARKET_VER plugin=$PLUGIN_VER"
        echo "       Run: ./scripts/release/bump-plugin-version.sh to sync versions"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "ERROR: $PLUGIN_JSON does not exist"
    echo "       Expected plugin manifest at $PLUGIN_JSON"
    ERRORS=$((ERRORS + 1))
fi

if [ "$ERRORS" -gt 0 ]; then
    exit 1
fi

exit 0

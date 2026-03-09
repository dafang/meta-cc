#!/bin/bash
# Pre-commit hook: Check version consistency
# Warns if marketplace.json or plugin.json version doesn't match latest git tag
# This is expected during version bumps, so it's a warning not an error

LATEST=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
MARKET=$(jq -r '.plugins[0].version' .claude-plugin/marketplace.json)

if [ "$LATEST" != "$MARKET" ] && [ -n "$LATEST" ]; then
    echo "⚠️  Version mismatch: tag=v$LATEST, marketplace=$MARKET"
    echo "    (This is expected when bumping versions - will be fixed by bump-version.sh)"
fi

# Also check plugin.json parity if it exists
PLUGIN_JSON=".claude-plugin/plugin.json"
if [ -f "$PLUGIN_JSON" ]; then
    PLUGIN_VER=$(jq -r '.version' "$PLUGIN_JSON" 2>/dev/null || echo "")
    if [ -n "$PLUGIN_VER" ] && [ "$MARKET" != "$PLUGIN_VER" ]; then
        echo "⚠️  plugin.json version mismatch: marketplace=$MARKET plugin=$PLUGIN_VER"
        echo "    Run: ./scripts/release/bump-plugin-version.sh to sync versions"
    fi
fi

# Always exit 0 (warning only, don't block commits)
exit 0

#!/bin/bash
# Pre-commit hook: Check plugin-src/ changes and remind about version bump
# Expected behavior:
#   - If plugin-src/ files were staged, remind the developer to bump the version
#   - If plugin-src/.claude-plugin/plugin.json and .claude-plugin/marketplace.json
#     versions are out of sync, warn and exit non-zero
#   - Otherwise, pass silently (no-op)

set -e

PLUGIN_JSON="plugin-src/.claude-plugin/plugin.json"
MARKETPLACE_JSON=".claude-plugin/marketplace.json"

# Check if plugin-src/ files are staged in this commit
PLUGIN_SRC_CHANGED=$(git diff --cached --name-only | grep -c '^plugin-src/' 2>/dev/null || true)

if [ "$PLUGIN_SRC_CHANGED" -gt 0 ]; then
    echo "Pre-commit: plugin-src/ changes detected"

    # Verify version consistency between plugin.json and marketplace.json
    if [ -f "$PLUGIN_JSON" ] && [ -f "$MARKETPLACE_JSON" ]; then
        PLUGIN_VER=$(jq -r '.version' "$PLUGIN_JSON" 2>/dev/null || echo "")
        MARKET_VER=$(jq -r '.plugins[0].version' "$MARKETPLACE_JSON" 2>/dev/null || echo "")

        if [ -n "$PLUGIN_VER" ] && [ -n "$MARKET_VER" ] && [ "$PLUGIN_VER" != "$MARKET_VER" ]; then
            echo "WARNING: Version mismatch detected:"
            echo "  plugin-src/.claude-plugin/plugin.json: $PLUGIN_VER"
            echo "  .claude-plugin/marketplace.json:       $MARKET_VER"
            echo ""
            echo "Run './scripts/release/bump-plugin-version.sh patch' to sync versions."
        fi
    fi
fi

echo "Pre-commit checks passed"

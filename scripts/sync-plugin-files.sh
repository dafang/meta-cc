#!/bin/bash
# Prepare plugin files for release packaging
# Usage:
#   ./sync-plugin-files.sh          - Sync files
#   ./sync-plugin-files.sh --verify - Verify sync (don't modify files)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DIST_DIR="$PROJECT_ROOT/dist"
CAPABILITIES_DIR="$PROJECT_ROOT/capabilities"

# Parse arguments
VERIFY_MODE=false
if [ "$1" = "--verify" ]; then
    VERIFY_MODE=true
    echo "=== Plugin File Sync Verification ==="
    echo ""
else
    echo "Preparing plugin files for release packaging..."
fi

if [ "$VERIFY_MODE" = true ]; then
    # VERIFY MODE: Check that sync was done correctly
    echo "[1/3] Verifying dist/ structure..."
    if [ ! -d "$DIST_DIR/commands" ] || [ ! -d "$DIST_DIR/agents" ]; then
        echo "❌ ERROR: Plugin file sync failed - dist/ directory not created"
        exit 1
    fi
    echo "✓ dist/ structure verified"
    echo ""

    echo "[2/3] Checking file count..."
    DIST_CMD_COUNT=$(find "$DIST_DIR/commands" -name "*.md" 2>/dev/null | wc -l)
    EXPECTED_COUNT=4

    if [ "$DIST_CMD_COUNT" -ne "$EXPECTED_COUNT" ]; then
        echo "❌ ERROR: Command file count mismatch: expected $EXPECTED_COUNT, got $DIST_CMD_COUNT"
        exit 1
    fi
    echo "✓ File count verified: $DIST_CMD_COUNT command file(s)"
    echo ""

    echo "[3/3] Verifying file content..."
    for cmd in meta prompt-find prompt-list prompt-show; do
        if [ ! -f "$DIST_DIR/commands/${cmd}.md" ]; then
            echo "❌ ERROR: ${cmd}.md not found in dist/commands/"
            exit 1
        fi
    done
    echo "✓ All 4 commands verified"
    echo ""

    echo "✅ Plugin file sync verification passed"
else
    # SYNC MODE: Perform the sync
    # Verify source directories exist
    if [ ! -f "$PROJECT_ROOT/.claude/commands/meta.md" ]; then
        echo "ERROR: .claude/commands/meta.md not found"
        exit 1
    fi

    # Create dist directories (clean agents and commands to remove stale files)
    mkdir -p "$DIST_DIR/commands" "$DIST_DIR/agents" "$DIST_DIR/skills"
    rm -f "$DIST_DIR/agents/"*.md 2>/dev/null || true
    rm -f "$DIST_DIR/commands/"*.md 2>/dev/null || true

    # Copy published commands
    echo "  Copying published commands from .claude/commands/..."
    PUBLISHED_COMMANDS="meta prompt-find prompt-list prompt-show"
    for cmd in $PUBLISHED_COMMANDS; do
        if [ -f "$PROJECT_ROOT/.claude/commands/${cmd}.md" ]; then
            cp "$PROJECT_ROOT/.claude/commands/${cmd}.md" "$DIST_DIR/commands/"
        else
            echo "  WARNING: Expected command not found: .claude/commands/${cmd}.md"
        fi
    done

    # Copy only published agents (not dev-only ones like feature-developer, phase-planner-executor)
    echo "  Copying published agents from .claude/agents/..."
    PUBLISHED_AGENTS="iteration-executor iteration-prompt-designer knowledge-extractor project-planner stage-executor"
    for agent in $PUBLISHED_AGENTS; do
        if [ -f "$PROJECT_ROOT/.claude/agents/${agent}.md" ]; then
            cp "$PROJECT_ROOT/.claude/agents/${agent}.md" "$DIST_DIR/agents/"
        else
            echo "  WARNING: Expected agent not found: .claude/agents/${agent}.md"
        fi
    done

    # Copy skills directory with all supporting files
    echo "  Copying skills from .claude/skills/..."
    if [ -d "$PROJECT_ROOT/.claude/skills" ]; then
        cp -r "$PROJECT_ROOT/.claude/skills/"* "$DIST_DIR/skills/"
        SKILL_COUNT=$(find "$DIST_DIR/skills" -name "SKILL.md" 2>/dev/null | wc -l)
        SKILL_FILES=$(find "$DIST_DIR/skills" -type f 2>/dev/null | wc -l)
        echo "    ✓ Copied $SKILL_COUNT skills ($SKILL_FILES total files)"
    fi

    # Count files
    CMD_COUNT=$(find "$DIST_DIR/commands" -name "*.md" 2>/dev/null | wc -l)
    AGENT_COUNT=$(find "$DIST_DIR/agents" -name "*.md" 2>/dev/null | wc -l)
    SKILL_COUNT=$(find "$DIST_DIR/skills" -name "SKILL.md" 2>/dev/null | wc -l)

    echo "✓ Plugin files synced to $DIST_DIR/"
    echo "✓ Total: $CMD_COUNT command, $AGENT_COUNT agents, $SKILL_COUNT skills"
    echo "  Note: 13 capability files distributed separately in capabilities-latest.tar.gz"
fi

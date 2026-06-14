#!/bin/bash
# Build the platform-independent skills-only package
#
# Usage:
#   ./build-skills-package.sh --version <version> --output <dir>
#
# Output:
#   <dir>/meta-cc-skills-<version>.tar.gz
#
# Contents:
#   meta-cc-skills-<version>/
#     commands/        (slash command .md files)
#     skills/          (Codex skills)
#     lib/             (meta-utils.sh)
#     install-skills.sh
#     README-skills.md

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

VERSION=""
OUTPUT_DIR=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        *)
            echo "Usage: $0 --version <version> --output <dir>"
            exit 1
            ;;
    esac
done

if [ -z "$VERSION" ] || [ -z "$OUTPUT_DIR" ]; then
    echo "Usage: $0 --version <version> --output <dir>"
    exit 1
fi

PKG_NAME="meta-cc-skills-${VERSION}"
STAGING_DIR="$(mktemp -d)"
trap "rm -rf $STAGING_DIR" EXIT

PKG_STAGING="$STAGING_DIR/$PKG_NAME"
mkdir -p "$PKG_STAGING/commands" "$PKG_STAGING/skills" "$PKG_STAGING/lib"

echo "Building skills package: $PKG_NAME"
echo ""

# Copy slash commands from dist/ (synced from plugin-src/commands/)
# Fall back to plugin-src/commands/ if dist/ not available
if [ -d "$PROJECT_ROOT/dist/commands" ]; then
    CMD_SRC="$PROJECT_ROOT/dist/commands"
else
    CMD_SRC="$PROJECT_ROOT/plugin-src/commands"
fi

echo "[1/5] Copying Claude Code commands from $CMD_SRC..."
for f in "$CMD_SRC"/*.md; do
    [ -f "$f" ] || { echo "ERROR: No .md files found in $CMD_SRC"; exit 1; }
    cp "$f" "$PKG_STAGING/commands/"
done
CMD_COUNT=$(ls "$PKG_STAGING/commands/"*.md 2>/dev/null | wc -l)
echo "  Copied $CMD_COUNT command files"

echo "[2/5] Copying Codex skills..."
SKILL_SRC="$PROJECT_ROOT/plugin-src/skills"
if [ ! -d "$SKILL_SRC" ]; then
    echo "ERROR: plugin-src/skills not found"
    exit 1
fi
cp -R "$SKILL_SRC"/. "$PKG_STAGING/skills/"
SKILL_COUNT=$(find "$PKG_STAGING/skills" -name "SKILL.md" 2>/dev/null | wc -l)
if [ "$SKILL_COUNT" -eq 0 ]; then
    echo "ERROR: No Codex SKILL.md files found in $SKILL_SRC"
    exit 1
fi
echo "  Copied $SKILL_COUNT Codex skill(s)"

# Copy lib/meta-utils.sh
echo "[3/5] Copying lib/meta-utils.sh..."
if [ ! -f "$PROJECT_ROOT/lib/meta-utils.sh" ]; then
    echo "ERROR: lib/meta-utils.sh not found at $PROJECT_ROOT/lib/meta-utils.sh"
    exit 1
fi
cp "$PROJECT_ROOT/lib/meta-utils.sh" "$PKG_STAGING/lib/"

# Copy install-skills.sh
echo "[4/5] Copying install-skills.sh..."
if [ ! -f "$PROJECT_ROOT/scripts/install/install-skills.sh" ]; then
    echo "ERROR: scripts/install/install-skills.sh not found"
    exit 1
fi
cp "$PROJECT_ROOT/scripts/install/install-skills.sh" "$PKG_STAGING/"
chmod +x "$PKG_STAGING/install-skills.sh"

# Create README-skills.md
echo "[5/5] Generating README-skills.md..."
cat > "$PKG_STAGING/README-skills.md" <<EOF
# meta-cc Skills Package

This package contains the platform-independent prompt-library integrations for meta-cc.

## Contents

- \`commands/\` — Claude Code slash commands (prompt-find, prompt-list, prompt-show)
- \`skills/\` — Codex skills (prompt-find, prompt-list, prompt-show)
- \`lib/\` — Shared shell utilities
- \`install-skills.sh\` — Installer script

## Installation

\`\`\`bash
tar -xzf meta-cc-skills-${VERSION}.tar.gz
cd meta-cc-skills-${VERSION}
./install-skills.sh
\`\`\`

By default, Claude Code commands are installed to \`~/.claude/commands/\`
and Codex skills are installed to \`~/.codex/skills/\`.
Override the destinations with \`CLAUDE_DIR\` and \`CODEX_HOME\`:

\`\`\`bash
CLAUDE_DIR=/path/to/.claude CODEX_HOME=/path/to/.codex ./install-skills.sh
\`\`\`

To install one host only:

\`\`\`bash
INSTALL_CODEX=0 ./install-skills.sh
INSTALL_CLAUDE=0 ./install-skills.sh
\`\`\`

## Usage

After Claude Code installation, restart Claude Code and use:
- \`/prompt-find <keywords>\` — search saved prompts
- \`/prompt-list\` — browse prompt library
- \`/prompt-show <id>\` — view prompt details

After Codex installation, restart Codex and ask for the matching skill by name:
- \`prompt-find\` — search saved prompts
- \`prompt-list\` — browse prompt library
- \`prompt-show\` — view prompt details

## MCP Server

This package does NOT include the MCP server binary.
For the full install (binary + commands), use the platform-specific
\`meta-cc-plugin-{platform}.tar.gz\` package instead.
EOF

# Create archive
mkdir -p "$OUTPUT_DIR"
ARCHIVE="$OUTPUT_DIR/${PKG_NAME}.tar.gz"
tar -czf "$ARCHIVE" -C "$STAGING_DIR" "$PKG_NAME"

echo ""
echo "✓ Package built: $ARCHIVE"
echo "  Contents:"
tar -tzf "$ARCHIVE" | sed 's/^/    /'

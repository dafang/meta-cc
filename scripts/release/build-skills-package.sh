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
mkdir -p "$PKG_STAGING/commands" "$PKG_STAGING/lib"

echo "Building skills package: $PKG_NAME"
echo ""

# Copy slash commands from dist/ (synced from plugin-src/commands/)
# Fall back to plugin-src/commands/ if dist/ not available
if [ -d "$PROJECT_ROOT/dist/commands" ]; then
    CMD_SRC="$PROJECT_ROOT/dist/commands"
else
    CMD_SRC="$PROJECT_ROOT/plugin-src/commands"
fi

echo "[1/4] Copying commands from $CMD_SRC..."
for f in "$CMD_SRC"/*.md; do
    [ -f "$f" ] || { echo "ERROR: No .md files found in $CMD_SRC"; exit 1; }
    cp "$f" "$PKG_STAGING/commands/"
done
CMD_COUNT=$(ls "$PKG_STAGING/commands/"*.md 2>/dev/null | wc -l)
echo "  Copied $CMD_COUNT command files"

# Copy lib/meta-utils.sh
echo "[2/4] Copying lib/meta-utils.sh..."
if [ ! -f "$PROJECT_ROOT/lib/meta-utils.sh" ]; then
    echo "ERROR: lib/meta-utils.sh not found at $PROJECT_ROOT/lib/meta-utils.sh"
    exit 1
fi
cp "$PROJECT_ROOT/lib/meta-utils.sh" "$PKG_STAGING/lib/"

# Copy install-skills.sh
echo "[3/4] Copying install-skills.sh..."
if [ ! -f "$PROJECT_ROOT/scripts/install/install-skills.sh" ]; then
    echo "ERROR: scripts/install/install-skills.sh not found"
    exit 1
fi
cp "$PROJECT_ROOT/scripts/install/install-skills.sh" "$PKG_STAGING/"
chmod +x "$PKG_STAGING/install-skills.sh"

# Create README-skills.md
echo "[4/4] Generating README-skills.md..."
cat > "$PKG_STAGING/README-skills.md" <<EOF
# meta-cc Skills Package

This package contains the platform-independent slash commands for meta-cc.

## Contents

- \`commands/\` — Claude Code slash commands (prompt-find, prompt-list, prompt-show)
- \`lib/\` — Shared shell utilities
- \`install-skills.sh\` — Installer script

## Installation

\`\`\`bash
tar -xzf meta-cc-skills-${VERSION}.tar.gz
cd meta-cc-skills-${VERSION}
./install-skills.sh
\`\`\`

By default, commands are installed to \`~/.claude/commands/\`.
Override with the \`CLAUDE_DIR\` environment variable:

\`\`\`bash
CLAUDE_DIR=/path/to/.claude ./install-skills.sh
\`\`\`

## Usage

After installation, restart Claude Code and use:
- \`/prompt-find <keywords>\` — search saved prompts
- \`/prompt-list\` — browse prompt library
- \`/prompt-show <id>\` — view prompt details

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

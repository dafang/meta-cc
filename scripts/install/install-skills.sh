#!/bin/bash
# meta-cc skills installer
#
# Installs Claude Code slash commands and utilities.
# Does NOT install the MCP server binary.
#
# Usage:
#   ./install-skills.sh
#
# Environment:
#   CLAUDE_DIR   Target Claude config dir (default: ~/.claude)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="${CLAUDE_DIR:-${HOME}/.claude}"

info()  { echo -e "${GREEN}✓${NC} $1"; }
warn()  { echo -e "${YELLOW}⚠${NC} $1"; }
error_exit() { echo -e "${RED}ERROR: $1${NC}" >&2; exit 1; }

install_commands() {
    local cmd_src="$SCRIPT_DIR/commands"
    local cmd_dst="$CLAUDE_DIR/commands"

    [ -d "$cmd_src" ] || error_exit "commands/ directory not found at $cmd_src"

    mkdir -p "$cmd_dst"
    local count=0
    for f in "$cmd_src"/*.md; do
        [ -f "$f" ] || continue
        cp "$f" "$cmd_dst/"
        count=$((count + 1))
    done

    [ "$count" -gt 0 ] || error_exit "No .md command files found in $cmd_src"
    info "Installed $count slash commands to $cmd_dst"
}

install_lib() {
    local lib_src="$SCRIPT_DIR/lib"
    local lib_dst="$CLAUDE_DIR/lib"

    [ -d "$lib_src" ] || { warn "lib/ not found, skipping"; return; }

    mkdir -p "$lib_dst"
    cp -r "$lib_src"/. "$lib_dst/"
    info "Installed lib/ utilities to $lib_dst"
}

verify_installation() {
    local ok=true
    for cmd in prompt-find prompt-list prompt-show; do
        if [ ! -f "$CLAUDE_DIR/commands/${cmd}.md" ]; then
            warn "Expected command not found: $CLAUDE_DIR/commands/${cmd}.md"
            ok=false
        fi
    done
    $ok && info "Verification passed"
}

main() {
    echo "Installing meta-cc skills..."
    echo "  Target: $CLAUDE_DIR"
    echo ""

    install_commands
    install_lib
    verify_installation

    echo ""
    echo "Installation complete!"
    echo "Restart Claude Code to load the slash commands."
    echo ""
}

main "$@"

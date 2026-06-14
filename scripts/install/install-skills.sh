#!/bin/bash
# meta-cc skills installer
#
# Installs Claude Code slash commands, Codex skills, and shared utilities.
# Does NOT install the MCP server binary.
#
# Usage:
#   ./install-skills.sh
#
# Environment:
#   CLAUDE_DIR       Target Claude config dir (default: ~/.claude)
#   CODEX_HOME       Target Codex home dir (default: ~/.codex)
#   CODEX_DIR        Alias for CODEX_HOME
#   INSTALL_CLAUDE   Install Claude Code commands (default: 1)
#   INSTALL_CODEX    Install Codex skills (default: 1)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="${CLAUDE_DIR:-${HOME}/.claude}"
CODEX_HOME="${CODEX_HOME:-${CODEX_DIR:-${HOME}/.codex}}"
INSTALL_CLAUDE="${INSTALL_CLAUDE:-1}"
INSTALL_CODEX="${INSTALL_CODEX:-1}"

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

install_codex_skills() {
    local skill_src="$SCRIPT_DIR/skills"
    local skill_dst="$CODEX_HOME/skills"

    [ -d "$skill_src" ] || error_exit "skills/ directory not found at $skill_src"

    mkdir -p "$skill_dst"
    local count=0
    for skill in "$skill_src"/*; do
        [ -d "$skill" ] || continue
        cp -R "$skill" "$skill_dst/"
        count=$((count + 1))
    done

    [ "$count" -gt 0 ] || error_exit "No Codex skills found in $skill_src"
    info "Installed $count Codex skills to $skill_dst"
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

    if [ "$INSTALL_CLAUDE" != "0" ]; then
        for cmd in prompt-find prompt-list prompt-show; do
            if [ ! -f "$CLAUDE_DIR/commands/${cmd}.md" ]; then
                warn "Expected command not found: $CLAUDE_DIR/commands/${cmd}.md"
                ok=false
            fi
        done
        if [ ! -f "$CLAUDE_DIR/lib/meta-utils.sh" ]; then
            warn "Expected lib not found: $CLAUDE_DIR/lib/meta-utils.sh"
            ok=false
        fi
    fi

    if [ "$INSTALL_CODEX" != "0" ]; then
        for skill in prompt-find prompt-list prompt-show; do
            if [ ! -f "$CODEX_HOME/skills/${skill}/SKILL.md" ]; then
                warn "Expected Codex skill not found: $CODEX_HOME/skills/${skill}/SKILL.md"
                ok=false
            fi
        done
    fi

    $ok && info "Verification passed"
}

main() {
    echo "Installing meta-cc skills..."
    echo "  Claude Code target: $CLAUDE_DIR"
    echo "  Codex target:       $CODEX_HOME"
    echo ""

    if [ "$INSTALL_CLAUDE" != "0" ]; then
        install_commands
        install_lib
    else
        warn "Skipping Claude Code command install (INSTALL_CLAUDE=0)"
    fi

    if [ "$INSTALL_CODEX" != "0" ]; then
        install_codex_skills
    else
        warn "Skipping Codex skill install (INSTALL_CODEX=0)"
    fi

    verify_installation

    echo ""
    echo "Installation complete!"
    echo "Restart Claude Code and/or Codex to load the integrations."
    echo ""
}

main "$@"

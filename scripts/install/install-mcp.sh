#!/bin/bash
# meta-cc MCP server installer
#
# Installs a bare meta-cc-mcp binary to INSTALL_DIR.
# Does NOT install slash commands or skills.
#
# Usage:
#   ./install-mcp.sh <path-to-binary>
#
# Environment:
#   INSTALL_DIR   Target directory (default: ~/.local/bin)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}✓${NC} $1"; }
warn()  { echo -e "${YELLOW}⚠${NC} $1"; }
error_exit() { echo -e "${RED}ERROR: $1${NC}" >&2; exit 1; }

BINARY_SRC="$1"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

if [ -z "$BINARY_SRC" ]; then
    echo "Usage: $0 <path-to-meta-cc-mcp-binary>"
    echo ""
    echo "Environment:"
    echo "  INSTALL_DIR   Target directory (default: ~/.local/bin)"
    exit 1
fi

[ -f "$BINARY_SRC" ] || error_exit "Binary not found: $BINARY_SRC"
[ -r "$BINARY_SRC" ] || error_exit "Binary not readable: $BINARY_SRC"

install_binary() {
    mkdir -p "$INSTALL_DIR"

    local dest="$INSTALL_DIR/meta-cc-mcp"
    cp "$BINARY_SRC" "$dest"
    chmod +x "$dest"
    info "Installed meta-cc-mcp to $dest"
}

verify_installation() {
    local dest="$INSTALL_DIR/meta-cc-mcp"
    [ -f "$dest" ]  || error_exit "Binary missing after install: $dest"
    [ -x "$dest" ]  || error_exit "Binary not executable after install: $dest"
    info "Verification passed"
}

suggest_path() {
    # Only suggest if INSTALL_DIR is not already in PATH
    if ! echo "$PATH" | tr ':' '\n' | grep -qxF "$INSTALL_DIR"; then
        echo ""
        warn "Add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
}

main() {
    echo "Installing meta-cc MCP server..."
    echo "  Source:  $BINARY_SRC"
    echo "  Target:  $INSTALL_DIR"
    echo ""

    install_binary
    verify_installation
    suggest_path

    echo ""
    echo "Installation complete!"
    echo "Configure MCP: claude mcp add meta-cc meta-cc-mcp"
    echo ""
}

main

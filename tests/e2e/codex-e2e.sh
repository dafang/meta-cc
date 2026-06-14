#!/bin/bash
# Codex-focused E2E test for meta-cc.
# Verifies the Codex package/install surface and the MCP server's Codex
# transcript discovery path against a real JSON-RPC tool call.
#
# Usage: ./tests/e2e/codex-e2e.sh [binary_path]

set -euo pipefail

BINARY="${1:-./bin/meta-cc-mcp}"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

fail() {
    echo -e "${RED}FAILED:${NC} $1" >&2
    exit 1
}

pass() {
    echo -e "  ${GREEN}PASS${NC} - $1"
}

require_file() {
    [ -f "$1" ] || fail "missing file: $1"
}

require_dir() {
    [ -d "$1" ] || fail "missing directory: $1"
}

get_json_response() {
    local input="$1"
    echo "$input" | grep -E '^\s*\{' | grep '"jsonrpc"' | head -1 || true
}

send_request() {
    local request="$1"
    local raw_output
    if command -v timeout >/dev/null 2>&1; then
        raw_output=$(printf '%s\n' "$request" | timeout 8 "$BINARY" 2>&1 || true)
    elif command -v gtimeout >/dev/null 2>&1; then
        raw_output=$(printf '%s\n' "$request" | gtimeout 8 "$BINARY" 2>&1 || true)
    elif command -v python3 >/dev/null 2>&1; then
        raw_output=$(REQUEST_PAYLOAD="$request" python3 - 8 "$BINARY" <<'PY' 2>&1 || true
import os
import subprocess
import sys

seconds = int(sys.argv[1])
binary = sys.argv[2]
payload = os.environ["REQUEST_PAYLOAD"] + "\n"

try:
    proc = subprocess.run(
        [binary],
        input=payload,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        timeout=seconds,
        check=False,
    )
    print(proc.stdout, end="")
except subprocess.TimeoutExpired as exc:
    if exc.stdout:
        print(exc.stdout, end="")
    print("__META_CC_TIMEOUT__")
PY
)
    else
        raw_output=$(printf '%s\n' "$request" | "$BINARY" 2>&1 || true)
    fi
    local response
    response=$(get_json_response "$raw_output")
    if [ -z "$response" ]; then
        echo "$raw_output" | sed 's/^/RAW: /' >&2
    fi
    echo "$response"
}

if [ ! -f "$BINARY" ]; then
    fail "binary not found: $BINARY"
fi

if ! command -v jq >/dev/null 2>&1; then
    fail "jq is required"
fi

echo "=========================================="
echo "Codex E2E Test"
echo "=========================================="
echo "Binary:      $BINARY"
echo "Project dir: $PROJECT_DIR"
echo "=========================================="
echo ""

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

CODEX_HOME="$TMP_DIR/codex-home"
CLAUDE_DIR="$TMP_DIR/claude-home"
INSTALL_DIR="$TMP_DIR/bin"
SESSION_ID="codex-e2e-session"
UNIQUE_MESSAGE="codex-e2e-message-$RANDOM-$(date +%s)"
SESSION_DIR="$CODEX_HOME/sessions/2026/06/14"
SESSION_FILE="$SESSION_DIR/$SESSION_ID.jsonl"

echo -e "${BLUE}Test 1: Install Codex skills package into temp CODEX_HOME${NC}"
rm -rf "$TMP_DIR/skills-package"
bash "$PROJECT_DIR/scripts/release/build-skills-package.sh" \
    --version v0.0.0-codex-e2e \
    --output "$TMP_DIR/skills-package" >/dev/null

tar -xzf "$TMP_DIR/skills-package/meta-cc-skills-v0.0.0-codex-e2e.tar.gz" -C "$TMP_DIR"
INSTALL_CODEX=1 INSTALL_CLAUDE=0 CODEX_HOME="$CODEX_HOME" CLAUDE_DIR="$CLAUDE_DIR" \
    bash "$TMP_DIR/meta-cc-skills-v0.0.0-codex-e2e/install-skills.sh" >/dev/null

for skill in prompt-find prompt-list prompt-show; do
    require_file "$CODEX_HOME/skills/$skill/SKILL.md"
done
if find "$CLAUDE_DIR" -type f 2>/dev/null | grep -q .; then
    fail "INSTALL_CLAUDE=0 wrote files under CLAUDE_DIR"
fi
pass "Codex skills installed under temp CODEX_HOME without Claude writes"
echo ""

echo -e "${BLUE}Test 2: Install full archive Codex plugin files into temp CODEX_HOME${NC}"
FULL_PKG="$TMP_DIR/full-package"
mkdir -p "$FULL_PKG/bin" "$FULL_PKG/commands" "$FULL_PKG/skills" "$FULL_PKG/lib" \
    "$FULL_PKG/.claude-plugin" "$FULL_PKG/.codex-plugin"
cp "$BINARY" "$FULL_PKG/bin/meta-cc-mcp"
cp -r "$PROJECT_DIR/plugin-src/commands/." "$FULL_PKG/commands/"
cp -r "$PROJECT_DIR/plugin-src/skills/." "$FULL_PKG/skills/"
cat > "$FULL_PKG/lib/mcp-config.json" <<'EOF'
{
  "mcpServers": {
    "meta-cc": {
      "command": "meta-cc-mcp",
      "args": []
    }
  }
}
EOF
cp "$PROJECT_DIR/plugin-src/.claude-plugin/plugin.json" "$FULL_PKG/.claude-plugin/plugin.json"
cp "$PROJECT_DIR/plugin-src/.mcp.json" "$FULL_PKG/.mcp.json"
cp "$PROJECT_DIR/plugin-src/.codex-plugin/plugin.json" "$FULL_PKG/.codex-plugin/plugin.json"
cp "$PROJECT_DIR/plugin-src/.codex-mcp.json" "$FULL_PKG/.codex-mcp.json"
cp "$PROJECT_DIR/scripts/install/install.sh" "$FULL_PKG/install.sh"

(
    cd "$FULL_PKG"
    INSTALL_DIR="$INSTALL_DIR" CLAUDE_DIR="$CLAUDE_DIR" CODEX_HOME="$CODEX_HOME" \
        bash ./install.sh >/dev/null
)

require_file "$INSTALL_DIR/meta-cc-mcp"
require_file "$CODEX_HOME/plugins/meta-cc/.codex-plugin/plugin.json"
require_file "$CODEX_HOME/plugins/meta-cc/.codex-mcp.json"
jq -e '.skills == "./skills/" and .mcpServers == "./.codex-mcp.json"' \
    "$CODEX_HOME/plugins/meta-cc/.codex-plugin/plugin.json" >/dev/null
jq -e '.mcpServers["meta-cc"].command == "./bin/meta-cc-mcp"' \
    "$CODEX_HOME/plugins/meta-cc/.codex-mcp.json" >/dev/null
pass "Full archive installs Codex plugin manifest and MCP template"
echo ""

echo -e "${BLUE}Test 3: Query Codex transcript through real MCP JSON-RPC${NC}"
mkdir -p "$SESSION_DIR"
cat > "$SESSION_FILE" <<EOF
{"type":"user","sessionId":"$SESSION_ID","cwd":"$PROJECT_DIR","timestamp":"2026-06-14T06:00:00Z","message":{"role":"user","content":"$UNIQUE_MESSAGE"}}
{"type":"assistant","sessionId":"$SESSION_ID","cwd":"$PROJECT_DIR","timestamp":"2026-06-14T06:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"ack"}]}}
EOF

REQUEST=$(jq -nc --arg cwd "$PROJECT_DIR" \
    '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_session_directory","arguments":{"scope":"project","working_dir":$cwd}}}')
RESPONSE=$(HOME="$TMP_DIR/home" CODEX_HOME="$CODEX_HOME" META_CC_PROJECTS_ROOT= \
    send_request "$REQUEST")
[ -n "$RESPONSE" ] || fail "no JSON-RPC response for get_session_directory"
DIR=$(echo "$RESPONSE" | jq -r '.result.content[0].text | fromjson | .directory')
FILE_COUNT=$(echo "$RESPONSE" | jq -r '.result.content[0].text | fromjson | .file_count')
[ "$DIR" = "$SESSION_DIR" ] || fail "expected Codex session dir $SESSION_DIR, got $DIR"
[ "$FILE_COUNT" = "1" ] || fail "expected one Codex session file, got $FILE_COUNT"
pass "get_session_directory resolved CODEX_HOME/sessions project transcript"

REQUEST=$(jq -nc --arg cwd "$PROJECT_DIR" --arg pattern "$UNIQUE_MESSAGE" \
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"query_user_messages","arguments":{"scope":"project","working_dir":$cwd,"pattern":$pattern,"limit":5}}}')
RESPONSE=$(HOME="$TMP_DIR/home" CODEX_HOME="$CODEX_HOME" META_CC_PROJECTS_ROOT= \
    send_request "$REQUEST")
[ -n "$RESPONSE" ] || fail "no JSON-RPC response for query_user_messages"
echo "$RESPONSE" | jq -e '.result.content[0].text | contains("'"$UNIQUE_MESSAGE"'")' >/dev/null \
    || fail "query_user_messages did not return the Codex transcript message"
pass "query_user_messages returned data from the Codex transcript"
echo ""

echo "=========================================="
echo "Codex E2E Test Complete"
echo "=========================================="

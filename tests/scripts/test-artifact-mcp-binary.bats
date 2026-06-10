#!/usr/bin/env bats
#
# TDD tests for bare MCP binary artifact and install-mcp.sh
#
# Contract:
#   - scripts/install/install-mcp.sh installs a bare binary
#   - The bare binary is a standalone executable, no archive needed
#
# Run: bats tests/scripts/test-artifact-mcp-binary.bats

setup() {
    export TEST_DIR="$(mktemp -d)"
    export ORIGINAL_DIR="$(pwd)"
    cd "$ORIGINAL_DIR"
}

teardown() {
    cd "$ORIGINAL_DIR"
    rm -rf "$TEST_DIR"
}

# ----------------------------------------------------------------
# 1. install-mcp.sh script exists
# ----------------------------------------------------------------

@test "install-mcp.sh: script exists" {
    [ -f "scripts/install/install-mcp.sh" ]
}

@test "install-mcp.sh: script is executable" {
    [ -x "scripts/install/install-mcp.sh" ]
}

@test "install-mcp.sh: shows usage when no args given" {
    run bash scripts/install/install-mcp.sh
    # Should either succeed with usage or fail with a helpful message
    [[ "$output" =~ "Usage:" || "$output" =~ "usage:" || "$output" =~ "BINARY" || "$status" -ne 0 ]]
}

# ----------------------------------------------------------------
# 2. install-mcp.sh installs binary correctly
# ----------------------------------------------------------------

@test "install-mcp.sh: installs binary to INSTALL_DIR" {
    # Create a fake binary for testing (same arch, so just cp a shell script)
    FAKE_BINARY="$TEST_DIR/meta-cc-mcp-fake"
    printf '#!/bin/sh\necho "meta-cc-mcp test"\n' > "$FAKE_BINARY"
    chmod +x "$FAKE_BINARY"

    INSTALL_DIR="$TEST_DIR/bin"
    run env INSTALL_DIR="$INSTALL_DIR" bash scripts/install/install-mcp.sh "$FAKE_BINARY"
    [ "$status" -eq 0 ]
    [ -f "$INSTALL_DIR/meta-cc-mcp" ]
}

@test "install-mcp.sh: installed binary is executable" {
    FAKE_BINARY="$TEST_DIR/meta-cc-mcp-fake"
    printf '#!/bin/sh\necho "meta-cc-mcp test"\n' > "$FAKE_BINARY"
    chmod +x "$FAKE_BINARY"

    INSTALL_DIR="$TEST_DIR/bin"
    env INSTALL_DIR="$INSTALL_DIR" bash scripts/install/install-mcp.sh "$FAKE_BINARY"
    [ -x "$INSTALL_DIR/meta-cc-mcp" ]
}

@test "install-mcp.sh: fails if binary path does not exist" {
    run env INSTALL_DIR="$TEST_DIR/bin" bash scripts/install/install-mcp.sh "/nonexistent/path/meta-cc-mcp"
    [ "$status" -ne 0 ]
}

@test "install-mcp.sh: idempotent (safe to run twice)" {
    FAKE_BINARY="$TEST_DIR/meta-cc-mcp-fake"
    printf '#!/bin/sh\necho "meta-cc-mcp test"\n' > "$FAKE_BINARY"
    chmod +x "$FAKE_BINARY"

    INSTALL_DIR="$TEST_DIR/bin"
    env INSTALL_DIR="$INSTALL_DIR" bash scripts/install/install-mcp.sh "$FAKE_BINARY"
    run env INSTALL_DIR="$INSTALL_DIR" bash scripts/install/install-mcp.sh "$FAKE_BINARY"
    [ "$status" -eq 0 ]
}

# ----------------------------------------------------------------
# 3. CI smoke tests script for skills package
# ----------------------------------------------------------------

@test "smoke-tests-skills.sh: script exists" {
    [ -f "scripts/ci/smoke-tests-skills.sh" ]
}

@test "smoke-tests-skills.sh: script is executable" {
    [ -x "scripts/ci/smoke-tests-skills.sh" ]
}

@test "smoke-tests-skills.sh: requires arguments" {
    run bash scripts/ci/smoke-tests-skills.sh
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Usage:" || "$output" =~ "ERROR" ]]
}

@test "smoke-tests-skills.sh: validates package file exists" {
    run bash scripts/ci/smoke-tests-skills.sh "v1.0.0" "$TEST_DIR/nonexistent.tar.gz"
    [ "$status" -ne 0 ]
}

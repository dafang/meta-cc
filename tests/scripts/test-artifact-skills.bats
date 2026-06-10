#!/usr/bin/env bats
#
# TDD tests for skills-only artifact
#
# These tests define the contract for:
#   - scripts/release/build-skills-package.sh
#   - scripts/install/install-skills.sh
#   - meta-cc-skills-{version}.tar.gz package structure
#
# Run: bats tests/scripts/test-artifact-skills.bats

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
# 1. Build script exists
# ----------------------------------------------------------------

@test "build-skills-package.sh: script exists" {
    [ -f "scripts/release/build-skills-package.sh" ]
}

@test "build-skills-package.sh: script is executable" {
    [ -x "scripts/release/build-skills-package.sh" ]
}

# ----------------------------------------------------------------
# 2. Skills package builds successfully
# ----------------------------------------------------------------

@test "build-skills-package.sh: builds without error" {
    run bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    [ "$status" -eq 0 ]
}

@test "build-skills-package.sh: creates tarball in output dir" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" ]
}

# ----------------------------------------------------------------
# 3. Package structure
# ----------------------------------------------------------------

@test "skills package: extracts to versioned directory" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"

    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"
    [ -d "$TEST_DIR/meta-cc-skills-v0.0.0-test" ]
}

@test "skills package: contains commands/ directory" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -d "$TEST_DIR/meta-cc-skills-v0.0.0-test/commands" ]
}

@test "skills package: contains prompt-find.md" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test/commands/prompt-find.md" ]
}

@test "skills package: contains prompt-list.md" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test/commands/prompt-list.md" ]
}

@test "skills package: contains prompt-show.md" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test/commands/prompt-show.md" ]
}

@test "skills package: contains lib/meta-utils.sh" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test/lib/meta-utils.sh" ]
}

@test "skills package: contains install-skills.sh" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -f "$TEST_DIR/meta-cc-skills-v0.0.0-test/install-skills.sh" ]
}

@test "skills package: install-skills.sh is executable" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    [ -x "$TEST_DIR/meta-cc-skills-v0.0.0-test/install-skills.sh" ]
}

@test "skills package: does NOT contain binary files" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    ! find "$TEST_DIR/meta-cc-skills-v0.0.0-test" -name "meta-cc-mcp*" | grep -q .
}

# ----------------------------------------------------------------
# 4. install-skills.sh script
# ----------------------------------------------------------------

@test "install-skills.sh: script exists" {
    [ -f "scripts/install/install-skills.sh" ]
}

@test "install-skills.sh: script is executable" {
    [ -x "scripts/install/install-skills.sh" ]
}

@test "install-skills.sh: installs commands to CLAUDE_DIR" {
    # Build and extract package
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    PKG_DIR="$TEST_DIR/meta-cc-skills-v0.0.0-test"
    CLAUDE_DIR="$TEST_DIR/dot-claude"
    mkdir -p "$CLAUDE_DIR"

    run env CLAUDE_DIR="$CLAUDE_DIR" bash "$PKG_DIR/install-skills.sh"
    [ "$status" -eq 0 ]
    [ -f "$CLAUDE_DIR/commands/prompt-find.md" ]
    [ -f "$CLAUDE_DIR/commands/prompt-list.md" ]
    [ -f "$CLAUDE_DIR/commands/prompt-show.md" ]
}

@test "install-skills.sh: installs lib/meta-utils.sh" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    PKG_DIR="$TEST_DIR/meta-cc-skills-v0.0.0-test"
    CLAUDE_DIR="$TEST_DIR/dot-claude"
    mkdir -p "$CLAUDE_DIR"

    env CLAUDE_DIR="$CLAUDE_DIR" bash "$PKG_DIR/install-skills.sh"
    [ -f "$CLAUDE_DIR/lib/meta-utils.sh" ]
}

@test "install-skills.sh: idempotent (safe to run twice)" {
    bash scripts/release/build-skills-package.sh \
        --version "v0.0.0-test" \
        --output "$TEST_DIR"
    tar -xzf "$TEST_DIR/meta-cc-skills-v0.0.0-test.tar.gz" -C "$TEST_DIR"

    PKG_DIR="$TEST_DIR/meta-cc-skills-v0.0.0-test"
    CLAUDE_DIR="$TEST_DIR/dot-claude"
    mkdir -p "$CLAUDE_DIR"

    env CLAUDE_DIR="$CLAUDE_DIR" bash "$PKG_DIR/install-skills.sh"
    run env CLAUDE_DIR="$CLAUDE_DIR" bash "$PKG_DIR/install-skills.sh"
    [ "$status" -eq 0 ]
}

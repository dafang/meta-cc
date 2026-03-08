# Plan 38–40: Standard Plugin Packaging

## Overview

Align meta-cc's plugin packaging with the official Claude Code plugin ecosystem standard. This plan implements **Option M** (Minimal Changes) from the proposal, with Option F (Full Restructuring) deferred to a follow-up plan.

Reference proposal: [docs/proposals/proposal-standard-plugin-packaging.md](../proposals/proposal-standard-plugin-packaging.md)

**Five gaps addressed:**

| # | Gap | Severity | Phase |
|---|-----|----------|-------|
| G1 | No `plugin.json` at correct location; `strict: false` conflict | High | 38 |
| G2 | `.mcp.json` missing — MCP server not wired into plugin lifecycle | High | 38 |
| G3 | Dev-only agents leak into releases; 3 commands missing from dist | Medium | 39 |
| G4 | Archive layout mismatch (`source: "./.claude"` vs flat root) | High | 39 |
| G5 | `uninstall.sh` broken (MCP orphan + agent glob) | Medium | 40 |

**Development methodology**: TDD throughout. Each stage begins with failing tests; implementation follows to make tests pass.

**Code limits**: Phase ≤500 lines, Stage ≤200 lines.

**Prerequisites**: PoC validations (Section 3 of proposal) must be completed before implementation begins:
- **PoC 1** (`${CLAUDE_PLUGIN_ROOT}` + `.mcp.json` format) gates Stage 38.2
- **PoC 2** (`strict: false` removal behavior) gates Stage 38.1
- **PoC 3** (command frontmatter) and **PoC 4** (directory vs explicit paths) are informational

PoCs should be executed as a pre-phase activity, not a numbered stage, since they require a separate test plugin repo and manual Claude Code interaction.

---

## Phase 38: Plugin Manifest and MCP Integration

**Goal**: Create `plugin.json` at the correct location, remove `strict: false`, add `.mcp.json`, and fix all version management scripts.

**Estimated code**: ~350–450 lines (tests + implementation)

### Stage Dependencies

```
PoC 2 (strict mode) ──► 38.1 (plugin.json + strict: false removal)
                            └──► 38.2 (.mcp.json)  [requires PoC 1]
                            └──► 38.3 (bump scripts + version sync)
                            └──► 38.4 (validation hooks)
```

Stages 38.1 is the foundation; 38.2–38.4 depend on it but are independent of each other.

---

### Stage 38.1 — Create `plugin.json`, Update `marketplace.json`, Remove `strict: false` (Atomic)

**Problem addressed**: G1 — No `plugin.json`; `strict: false` prevents adding one. Additionally, `marketplace.json` only declares 1 of 4 commands (see proposal Section 1.13).

**Prerequisite**: PoC 2 confirms that removing `strict: false` and adding `plugin.json` causes Claude Code to read content definitions from `plugin.json`.

**TDD sequence**:

1. Write `scripts/ci/test-plugin-json.sh` — a validation script that:
   - Checks `.claude/.claude-plugin/plugin.json` exists and is valid JSON
   - Checks `marketplace.json` does NOT contain `"strict": false`
   - Checks version parity between `plugin.json` and `marketplace.json`
   - Checks `plugin.json` declares exactly 4 commands, 5 agents, 18 skills
   - Checks `marketplace.json` `commands` array matches `plugin.json`
2. Run test — expect FAIL (plugin.json doesn't exist yet)
3. Create `.claude/.claude-plugin/plugin.json` with content from proposal Section 2.1.1 (includes all 4 commands: `meta`, `prompt-find`, `prompt-list`, `prompt-show`)
4. Update `.claude-plugin/marketplace.json`: remove `"strict": false`, add missing commands (`prompt-find.md`, `prompt-list.md`, `prompt-show.md`) to `commands` array
5. Run test — expect PASS
6. Run `make commit`

**Key files**:
- `.claude/.claude-plugin/plugin.json` — CREATE
- `.claude-plugin/marketplace.json` — UPDATE (remove `strict: false` + add 3 missing commands)

**Acceptance criteria**:
- `.claude/.claude-plugin/plugin.json` exists with correct schema
- `marketplace.json` no longer contains `"strict": false`
- `marketplace.json` declares all 4 commands (not just `meta.md`)
- Version in both files matches
- `commands/agents/skills` arrays in `plugin.json` match `marketplace.json`
- `make commit` passes

**CRITICAL**: All changes must be in a single atomic commit. Between removing `strict: false` and creating `plugin.json`, the plugin is in an undefined state.

**Line budget**: ≤180 lines (JSON content + test script)

---

### Stage 38.2 — Add `.mcp.json` for MCP Auto-Start

**Problem addressed**: G2 — MCP server not wired into plugin lifecycle.

**Prerequisite**: PoC 1 confirms `${CLAUDE_PLUGIN_ROOT}` resolves correctly AND determines the correct `.mcp.json` format (flat vs `mcpServers` wrapper — see proposal Section 1.14).

**TDD sequence**:

1. Add test to `scripts/ci/test-plugin-json.sh` that validates:
   - `.claude/.mcp.json` exists and is valid JSON
   - Contains a `meta-cc` server entry (at top level or under `mcpServers`, per PoC 1 result)
   - Command path uses `${CLAUDE_PLUGIN_ROOT}`
2. Run test — expect FAIL
3. Create `.claude/.mcp.json` using the format confirmed by PoC 1
4. Add MCP server reference to `.claude/.claude-plugin/plugin.json` (field name per PoC 1)
5. Run test — expect PASS
6. Run `make commit`

**Key files**:
- `.claude/.mcp.json` — CREATE
- `.claude/.claude-plugin/plugin.json` — UPDATE (add MCP reference)

**Acceptance criteria**:
- `.claude/.mcp.json` valid with `meta-cc` server entry in PoC-verified format
- `plugin.json` references `.mcp.json`
- `make commit` passes

**Line budget**: ≤80 lines

---

### Stage 38.3 — Fix Bump Scripts and `release.sh`

**Problem addressed**: Both bump scripts read non-existent `.claude-plugin/plugin.json` (wrong path). `release.sh` only updates `marketplace.json`.

**TDD sequence**:

1. Write tests (in existing test infrastructure or as shell assertions) that:
   - Run `bump-plugin-version.sh` in dry-run/check mode — currently fails with file-not-found
   - Verify `release.sh` would update both version files
2. Fix `scripts/hooks/plugin-version-bump.sh`:
   - Line 52: `.claude-plugin/plugin.json` → `.claude/.claude-plugin/plugin.json`
   - Lines 66-67 (update), line 74 (git add): same path fix
3. Fix `scripts/release/bump-plugin-version.sh`:
   - Line 45: same path fix
   - Lines 81-82 (update), line 94 (git add): same path fix
4. Fix `scripts/release/release.sh`:
   - Add `plugin.json` update after existing `marketplace.json` update
   - Add version parity verification step
5. Run `make commit`

**Key files**:
- `scripts/hooks/plugin-version-bump.sh` — UPDATE
- `scripts/release/bump-plugin-version.sh` — UPDATE
- `scripts/release/release.sh` — UPDATE

**Acceptance criteria**:
- `plugin-version-bump.sh` reads/writes `.claude/.claude-plugin/plugin.json`
- `bump-plugin-version.sh` reads/writes `.claude/.claude-plugin/plugin.json`
- `release.sh` updates both `marketplace.json` and `plugin.json`, verifies parity
- `make commit` passes

**Line budget**: ≤120 lines

---

### Stage 38.4 — Update Validation Hooks

**Problem addressed**: `validate-marketplace.sh` only validates `marketplace.json`; `check-version-sync.sh` ignores `plugin.json`.

**TDD sequence**:

1. Verify current hooks pass (baseline)
2. Add `plugin.json` existence and validity check to `scripts/hooks/validate-marketplace.sh`
3. Add version parity check (marketplace vs plugin)
4. Add `plugin.json` parity check to `scripts/hooks/check-version-sync.sh`
5. Run `make commit`

**Key files**:
- `scripts/hooks/validate-marketplace.sh` — UPDATE
- `scripts/hooks/check-version-sync.sh` — UPDATE

**Acceptance criteria**:
- `validate-marketplace.sh` checks `plugin.json` exists, is valid JSON, and versions match
- `check-version-sync.sh` compares git tag against both `marketplace.json` and `plugin.json`
- `make commit` passes

**Line budget**: ≤80 lines

---

## Phase 39: Release Pipeline Fixes

**Goal**: Fix agent filtering, archive layout, and release workflow to produce standard-compliant packages.

**Estimated code**: ~300–400 lines (tests + implementation)

### Stage Dependencies

```
38.1 (plugin.json exists)
  └──► 39.1 (agent filtering)
  └──► 39.2 (release workflow + archive layout)
  └──► 39.3 (Makefile bundle-release + smoke tests)
```

Stages 39.1–39.3 depend on Phase 38 but are independent of each other.

---

### Stage 39.1 — Fix Agent and Command Sync in `sync-plugin-files.sh`

**Problem addressed**: G3 — Dev-only agents leak into release archives. Additionally, 3 slash commands (`prompt-find`, `prompt-list`, `prompt-show`) are never synced to `dist/`.

**TDD sequence**:

1. Add smoke tests to `scripts/ci/smoke-tests.sh`:
   - `test_no_dev_agents()`: checks `feature-developer.md` and `phase-planner-executor.md` NOT in archive
   - `test_all_commands()`: checks all 4 commands present in archive (update existing count check from 1 to 4)
2. Run existing smoke tests against a local bundle — expect FAIL (dev agents present, commands missing)
3. Fix `scripts/sync-plugin-files.sh`:
   - Replace wildcard agent copy (line 76) with explicit published agent list
   - Add explicit command sync for all 4 published commands
   ```
   PUBLISHED_AGENTS="iteration-executor iteration-prompt-designer knowledge-extractor project-planner stage-executor"
   PUBLISHED_COMMANDS="meta prompt-find prompt-list prompt-show"
   ```
4. Rebuild bundle, run smoke tests — expect PASS
5. Run `make commit`

**Key files**:
- `scripts/sync-plugin-files.sh` — UPDATE
- `scripts/ci/smoke-tests.sh` — UPDATE

**Acceptance criteria**:
- Only 5 published agents in `dist/agents/`
- All 4 commands in `dist/commands/`
- `feature-developer.md` and `phase-planner-executor.md` NOT in dist or archive
- Smoke tests pass (agent filter + command count)
- `make commit` passes

**Line budget**: ≤120 lines

---

### Stage 39.2 — Update Release Workflow for `plugin.json`, `.mcp.json`, and Archive Source Rewrite

**Problem addressed**: G4 — Archive layout mismatch. `plugin.json` and `.mcp.json` not included in archive. Archive `marketplace.json` says `source: "./.claude"` but archive has flat layout.

**TDD sequence**:

1. Add smoke tests to `scripts/ci/smoke-tests.sh` (from proposal Section 2.1.10):
   - `test_plugin_json()`: checks `plugin.json` in archive, version parity
   - `test_mcp_json()`: checks `.mcp.json` in archive, contains `meta-cc` server
2. Update `.github/workflows/release.yml` "Create plugin packages" step:
   - Copy `plugin.json` from `.claude/.claude-plugin/` to `$PKG_DIR/.claude-plugin/`
   - Copy `.mcp.json` from `.claude/` to `$PKG_DIR/`
   - Rewrite archive `marketplace.json` `source` from `"./.claude"` to `"."`
3. Run smoke tests against rebuilt archive — expect PASS
4. Run `make commit`

**Key files**:
- `.github/workflows/release.yml` — UPDATE
- `scripts/ci/smoke-tests.sh` — UPDATE

**Acceptance criteria**:
- Archive contains `.claude-plugin/plugin.json` with correct version
- Archive contains `.mcp.json` with `meta-cc` server entry
- Archive `marketplace.json` has `source: "."` and declares all 4 commands
- Archive contains all 4 commands in `commands/`
- All existing smoke tests continue to pass
- `make commit` passes

**Line budget**: ≤120 lines

---

### Stage 39.3 — Fix `make bundle-release` Missing Skills

**Problem addressed**: `Makefile` `bundle-release` target doesn't create `skills/` or copy skills, diverging from CI workflow.

**TDD sequence**:

1. Run `make bundle-release` and verify `skills/` is missing in output — confirms the gap
2. Update `Makefile` `bundle-release` target:
   - Add `skills/` to `mkdir -p` list
   - Add `cp -r $(DIST_DIR)/skills/* $$BUNDLE_DIR/skills/` after existing copies
   - Also copy `plugin.json` and `.mcp.json` into bundle (matching CI workflow)
3. Run `make bundle-release` and verify `skills/` populated
4. Run `make commit`

**Key files**:
- `Makefile` — UPDATE

**Acceptance criteria**:
- `make bundle-release` output includes `skills/` with all 18 skill directories
- Bundle also includes `.claude-plugin/plugin.json` and `.mcp.json`
- `make commit` passes

**Line budget**: ≤60 lines

---

## Phase 40: Cleanup and Validation

**Goal**: Fix uninstall script, retire legacy MCP config, and validate end-to-end.

**Estimated code**: ~200–300 lines

### Stage Dependencies

```
39.* (release pipeline fixed)
  └──► 40.1 (uninstall.sh)
  └──► 40.2 (retire lib/mcp-config.json)
  └──► 40.3 (end-to-end validation)
```

---

### Stage 40.1 — Fix `uninstall.sh`

**Problem addressed**: G5 — Agent removal uses `meta-*` glob (doesn't match actual names like `iteration-executor.md`). MCP cleanup skipped entirely.

**TDD sequence**:

1. Write a test script that:
   - Creates a mock Claude directory with known agent files
   - Runs uninstall.sh in dry-run or test mode
   - Verifies all known agents are removed
   - Verifies MCP config entry is removed
2. Fix agent removal (lines 48-53, 56-61): replace `meta-*` glob with explicit agent name list
3. Fix MCP cleanup (lines 66-68): add `jq del(.mcpServers["meta-cc"])` logic per proposal Section 2.1.9
4. Run test — expect PASS
5. Run `make commit`

**Key files**:
- `scripts/install/uninstall.sh` — UPDATE

**Acceptance criteria**:
- Uninstall removes all 5 published agent files by name
- Uninstall removes `meta-cc` from `~/.claude/mcp.json`
- Graceful fallback if `jq` not available
- `make commit` passes

**Line budget**: ≤100 lines

---

### Stage 40.2 — Retire `lib/mcp-config.json`

**Problem addressed**: `lib/mcp-config.json` is the legacy MCP config, superseded by `.mcp.json`.

**Steps**:

1. Add deprecation notice to `lib/mcp-config.json`:
   ```json
   {
     "_deprecated": "Use .mcp.json in plugin root instead. This file will be removed in a future release.",
     "mcpServers": { "meta-cc": { "command": "meta-cc-mcp" } }
   }
   ```
2. Update `install.sh` to prefer `.mcp.json` when available, fall back to `lib/mcp-config.json`
3. Run `make commit`

**Key files**:
- `lib/mcp-config.json` — UPDATE
- `scripts/install/install.sh` — UPDATE (conditional)

**Acceptance criteria**:
- `lib/mcp-config.json` has deprecation notice
- `install.sh` uses `.mcp.json` if present
- `make commit` passes

**Line budget**: ≤60 lines

---

### Stage 40.3 — End-to-End Validation and Documentation

**Problem addressed**: Validate all changes work together; update documentation.

**Steps**:

1. Run full smoke test suite: `bash scripts/ci/smoke-tests.sh`
2. Run `make push` (full check including lint)
3. Verify `claude plugin marketplace add` works from repo (if PoC environment available)
4. Update skill namespace migration note in CHANGELOG
5. Verify all success criteria from proposal Section 11 (Option M)

**Acceptance criteria**:
- All smoke tests pass (existing + new `plugin.json`, `.mcp.json`, agent filter tests)
- `make push` passes
- `plugin.json` version equals `marketplace.json` version
- No dev-only agents in release archives
- `uninstall.sh` removes MCP registration and agent files
- Both bump scripts work correctly with new path

**Line budget**: ≤80 lines (documentation only; no new code)

---

## Implementation Notes

### Atomic commit requirement

Stage 38.1 MUST be a single atomic commit containing both:
- Creation of `.claude/.claude-plugin/plugin.json`
- Removal of `"strict": false` from `.claude-plugin/marketplace.json`

Between these two changes, the plugin is in an undefined state.

### Missing commands discovery

During architect review, 3 slash commands (`prompt-find`, `prompt-list`, `prompt-show`)
were found in `.claude/commands/` but missing from `marketplace.json` and `dist/commands/`.
These were added in Phase 28 but never declared or synced. Stage 38.1 adds them to both
manifests; Stage 39.1 adds them to the sync pipeline.

### Skill namespacing

With `plugin.json` declaring `"name": "meta-cc"`, skills become namespaced as `meta-cc:skill-name`. This is a user-visible change. Include migration note in release CHANGELOG.

### Version parity enforcement

After Phase 38, three mechanisms enforce `plugin.json` ↔ `marketplace.json` version parity:
1. `scripts/hooks/plugin-version-bump.sh` (pre-commit hook)
2. `scripts/hooks/validate-marketplace.sh` (validation hook)
3. `scripts/release/release.sh` (release automation)

### Option F follow-up

This plan implements Option M only. Option F (full restructuring: `plugin/` as canonical root, remove `dist/`, clean `.claude/`) is deferred to a separate plan after Option M is validated.

# Proposal: Standardize Plugin Packaging to Claude Code Official Spec

**Status**: Draft for Review
**Date**: 2026-03-08
**Author**: Claude Code Analysis

## Executive Summary

This proposal aligns meta-cc's plugin packaging with the official Claude Code plugin ecosystem
standard. Four rounds of architectural review have progressively corrected the diagnosis.

The most critical finding: **`plugin.json` must live inside the plugin source directory at
`<plugin-source>/.claude-plugin/plugin.json`**, not alongside `marketplace.json` at the
repo root. With the current `source: "./.claude"`, the correct location is
`.claude/.claude-plugin/plugin.json`. Furthermore, the current `marketplace.json` sets
`"strict": false`, which means the marketplace entry IS the complete plugin definition.
Adding a `plugin.json` while `strict: false` is set causes a **loading conflict**.

The actual gaps are:

1. **No `plugin.json` at the correct location** — and adding one requires removing
   `strict: false` from the marketplace entry first.
2. **`.mcp.json` missing** — MCP server not wired into plugin lifecycle.
3. **Archive layout mismatch** — release archive has content at root but `marketplace.json`
   says `source: "./.claude"`. Archive-based `claude plugin install` is broken.
4. **Dev-only agents leak into releases** — `sync-plugin-files.sh` copies all `*.md` without
   filtering.
5. **`uninstall.sh` leaves orphaned MCP config**.

The proposal offers two implementation options:
- **Option M (Minimal)**: Fix `plugin.json` (at correct location), add `.mcp.json`, fix
  archive layout, fix agent filtering. No structural reorganization.
- **Option F (Full)**: Introduce `plugin/` as dedicated plugin root, eliminate `dist/`,
  clean `.claude/`.

---

## 1. Current State: Accurate Assessment

### 1.1 Release Archive Structure (Binary Already Bundled)

`.github/workflows/release.yml` lines 61–94 produce:

```
meta-cc-plugin-${VERSION}-${PLATFORM}/
├── bin/
│   └── meta-cc-mcp              ← binary ALREADY included
├── .claude-plugin/
│   └── marketplace.json         ← no plugin.json yet
├── commands/                    ← flat layout from dist/
├── agents/                      ← ALL agents including dev-only (bug)
├── skills/
├── lib/
│   └── mcp-config.json
├── install.sh
├── uninstall.sh
├── README.md
└── LICENSE
```

The binary is already co-located. The archive layout (flat `commands/agents/skills` at root)
is structurally correct for a standard plugin. However, **dev-only agents are included** (see 1.8).

### 1.2 `plugin.json` Location: The Correct Architecture

Per the official Claude Code plugin spec (confirmed against `anthropics/claude-code` repo):

```
repo-root/
├── .claude-plugin/
│   └── marketplace.json         ← MARKETPLACE level (catalog)
└── <plugin-source>/             ← referenced by marketplace source field
    ├── .claude-plugin/
    │   └── plugin.json          ← PLUGIN level (manifest)
    ├── commands/
    ├── agents/
    └── skills/
```

`marketplace.json` at the repo root is the **catalog**. `plugin.json` lives inside each
**plugin source directory** at `<source>/.claude-plugin/plugin.json`.

With `source: "./.claude"`, the correct path is:
```
.claude/.claude-plugin/plugin.json     ← CORRECT
.claude-plugin/plugin.json             ← WRONG (this is the marketplace level)
```

All previous versions of this proposal had the location wrong.

### 1.3 `strict: false` — The Current Plugin Definition Mode

The current `marketplace.json` sets `"strict": false` on the plugin entry (line 36). Per the
official docs:

> **`strict: false`**: The marketplace entry is the entire definition. If the plugin also has
> a `plugin.json` that declares components, that's a conflict and the plugin fails to load.

This means:
1. The current `marketplace.json` already IS the complete plugin definition — `commands`,
   `agents`, `skills` arrays in the marketplace entry are the canonical source.
2. Adding a `plugin.json` with `strict: false` still set will **break plugin loading**.
3. To use `plugin.json`, we must first remove `"strict": false` (or set `"strict": true`).

**Implication for the proposal**: We cannot "just add `plugin.json`". We must coordinate
three changes atomically: (a) create `plugin.json` at the correct location, (b) remove
`strict: false` from marketplace.json, (c) ensure the two manifests don't conflict.

### 1.4 Archive-Based `claude plugin install` Is Currently Broken

`marketplace.json` has `"source": "./.claude"`. In the release archive there is no `.claude/`
directory — content is at the archive root. Archive-based `claude plugin install` fails.

### 1.5 Two Broken Bump Scripts (Latent Bug)

| Script | Trigger | Broken line |
|--------|---------|-------------|
| `scripts/hooks/plugin-version-bump.sh` | Pre-commit hook, automatic on `.claude/commands/` or `.claude/agents/` changes | Line 52: `jq -r '.version' .claude-plugin/plugin.json` |
| `scripts/release/bump-plugin-version.sh` | Manual invocation | Line 45: same `jq` call |

Both read `.claude-plugin/plugin.json` which does not exist. The hook script has `set -e`
(line 5) so the missing-file error at line 52 kills the commit before the staged-files guard
(lines 36-44) can fire.

**Additional issue**: Both scripts read/write `.claude-plugin/plugin.json` — the wrong
location per the spec (Section 1.2). Once we create `plugin.json` at the correct path
(`.claude/.claude-plugin/plugin.json`), these scripts must be updated to reference the
correct path.

### 1.6 `marketplace.json` Uses Explicit Paths (Not Auto-Discovery)

```json
"commands": ["./commands/meta.md"],
"agents":   ["./agents/iteration-executor.md", ...],
"skills":   ["./skills/agent-prompt-evolution", ...]
```

Note: `marketplace.json` only declares 1 command (`meta.md`), but `.claude/commands/`
contains 4 (see Section 1.13). The 3 undeclared commands are invisible to plugin users.

The proposal previously claimed `plugin.json` can use directory-level auto-discovery. While
the official spec documents this, the proven working pattern is explicit enumeration. Use
explicit paths in `plugin.json` until auto-discovery is verified by PoC.

### 1.7 `/meta` Command Depends on MCP Server and Capabilities

`commands/meta.md` calls `mcp_meta_cc.list_capabilities()`. This requires:
1. MCP server running (blocked by missing `.mcp.json`)
2. Capabilities content available via `META_CC_CAPABILITY_SOURCES`

Integrating capabilities into the plugin package is **out of scope** for this proposal.

### 1.8 Dev-Only Agents Leak into Releases

`sync-plugin-files.sh` line 76: `cp .claude/agents/*.md $(DIST_DIR)/agents/` copies ALL
agents. Currently `dist/agents/` contains 6 agents (feature-developer was added in HEAD
but hasn't been synced yet; phase-planner-executor IS in dist).

The `marketplace.json` lists only 5 agents explicitly, but the release archive contains
all `.md` files from `dist/agents/` — the marketplace declaration and the archive contents
disagree. When `strict: false` is set, only the marketplace-listed agents are loaded, so
this is not a functional bug. But it ships unnecessary files and will become a real bug
once `strict: false` is removed.

### 1.13 Three Slash Commands Missing from Marketplace Declaration

`.claude/commands/` contains **4 commands**, but `marketplace.json` only declares 1:

| Command | In `.claude/commands/` | In `marketplace.json` | In `dist/commands/` |
|---------|----------------------|----------------------|---------------------|
| `meta.md` | Yes | Yes | Yes |
| `prompt-find.md` | Yes | **No** | **No** |
| `prompt-list.md` | Yes | **No** | **No** |
| `prompt-show.md` | Yes | **No** | **No** |

These 3 commands were added in Phase 28 (prompt optimization learning) but never declared
in `marketplace.json` or synced to `dist/`. Currently they work in the development repo
because `.claude/commands/` is auto-discovered as the project's own commands directory.
But they are NOT available when the plugin is installed externally.

Once `strict: false` is removed and `plugin.json` becomes authoritative, all 4 commands
must be explicitly declared or they will be invisible to plugin users.

### 1.14 `.mcp.json` Format Uncertainty

Official plugin `.mcp.json` examples show a **flat format** (servers at top level):
```json
{
  "server-name": {
    "command": "${CLAUDE_PLUGIN_ROOT}/servers/binary",
    "args": []
  }
}
```

This differs from the project-level `.mcp.json` format which uses a `mcpServers` wrapper.
The exact format for plugin `.mcp.json` must be verified in PoC 1 before shipping.

### 1.9 `make bundle-release` Missing Skills

`Makefile` `bundle-release` target (line 391) creates `bin commands agents .claude-plugin lib`
but **no `skills/` directory** and does not copy skills. This diverges from the GitHub Actions
workflow which includes skills (line 68+81). Local-built bundles are incomplete.

### 1.10 `release.sh` Only Updates `marketplace.json`, Not `plugin.json`

Once `plugin.json` exists, `release.sh` will update `marketplace.json` version but leave
`plugin.json` stale. The manual `bump-plugin-version.sh` already handles both files (at the
wrong path — needs updating too).

### 1.11 `uninstall.sh` Leaves Orphaned MCP Config

`scripts/install/uninstall.sh` lines 66–68 skip MCP cleanup. Also, lines 48–53 and 56–61
remove files matching `meta-*` glob pattern, which won't match files like `iteration-executor.md`
or `project-planner.md`. The uninstaller is largely non-functional for current agent names.

### 1.12 `check-version-sync.sh` Only Checks `marketplace.json`

`scripts/hooks/check-version-sync.sh` compares git tag against `marketplace.json` only.
Adding `plugin.json` creates a third version source this hook ignores.

---

## 2. Proposed Changes

### 2.1 Shared Changes (Required for Both Options)

#### 2.1.1 Remove `strict: false` and Create `plugin.json` (Atomic)

This is the highest-priority change and must be atomic (single commit).

**Step 1**: Remove `"strict": false` from `.claude-plugin/marketplace.json` plugin entry.

**Step 2**: Create `.claude/.claude-plugin/plugin.json`:

```json
{
  "name": "meta-cc",
  "version": "2.3.6",
  "description": "Meta-Cognition tool for Claude Code: session history analysis, workflow optimization, and 18 validated BAIME methodology skills.",
  "author": {
    "name": "Yale Huang",
    "email": "yaleh@ieee.org",
    "url": "https://github.com/yaleh"
  },
  "license": "MIT",
  "homepage": "https://github.com/yaleh/meta-cc",
  "repository": "https://github.com/yaleh/meta-cc",
  "keywords": [
    "workflow-analysis", "session-history", "productivity", "metacognition",
    "analytics", "optimization", "methodologies", "testing-strategy",
    "ci-cd", "error-recovery", "refactoring", "technical-debt", "skills"
  ],
  "mcpServers": "./.mcp.json",
  "commands": [
    "./commands/meta.md",
    "./commands/prompt-find.md",
    "./commands/prompt-list.md",
    "./commands/prompt-show.md"
  ],
  "agents": [
    "./agents/iteration-executor.md",
    "./agents/iteration-prompt-designer.md",
    "./agents/knowledge-extractor.md",
    "./agents/project-planner.md",
    "./agents/stage-executor.md"
  ],
  "skills": [
    "./skills/agent-prompt-evolution",
    "./skills/api-design",
    "./skills/baseline-quality-assessment",
    "./skills/build-quality-gates",
    "./skills/ci-cd-optimization",
    "./skills/code-refactoring",
    "./skills/cross-cutting-concerns",
    "./skills/dependency-health",
    "./skills/documentation-management",
    "./skills/error-recovery",
    "./skills/knowledge-transfer",
    "./skills/methodology-bootstrapping",
    "./skills/observability-instrumentation",
    "./skills/rapid-convergence",
    "./skills/retrospective-validation",
    "./skills/subagent-prompt-construction",
    "./skills/technical-debt-management",
    "./skills/testing-strategy"
  ]
}
```

All paths are relative to the plugin source root (`.claude/`).

**Step 3**: Update `marketplace.json` `commands` array to include all 4 commands (currently
only `meta.md` is declared; `prompt-find.md`, `prompt-list.md`, `prompt-show.md` are
missing — see Section 1.13).

**Step 4**: Verify `marketplace.json` `commands/agents/skills` arrays are consistent with
`plugin.json`. With `strict: true` (or `strict` absent), `plugin.json` becomes the
authoritative source for plugin content. The `marketplace.json` arrays become supplementary
metadata for marketplace display.

**CRITICAL**: Steps 1–3 must be in a single atomic commit. Between removing `strict: false`
and creating `plugin.json`, the plugin is in an undefined state.

#### 2.1.2 Add `.mcp.json`

Place at `.claude/.mcp.json` (inside the plugin source).

**Format note**: Official plugin `.mcp.json` examples use a **flat format** (servers at
top level, no `mcpServers` wrapper), unlike the project-level `.mcp.json` which uses
`"mcpServers": {...}`. PoC 1 must verify the correct format. Candidate formats:

**Flat format** (per official plugin examples):
```json
{
  "meta-cc": {
    "command": "${CLAUDE_PLUGIN_ROOT}/bin/meta-cc-mcp",
    "args": [],
    "env": {}
  }
}
```

**Wrapped format** (per project-level `.mcp.json` convention):
```json
{
  "mcpServers": {
    "meta-cc": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/meta-cc-mcp",
      "args": [],
      "env": {}
    }
  }
}
```

**`${CLAUDE_PLUGIN_ROOT}` MUST be verified before this ships** (see Section 3, PoC 1).

**Windows note**: The binary is `meta-cc-mcp.exe` on Windows. Verify whether Claude Code
auto-appends `.exe` or whether the `.mcp.json` needs to handle this.

#### 2.1.3 Fix Bump Scripts: Correct `plugin.json` Path

**Both scripts** currently read/write `.claude-plugin/plugin.json` (wrong path). Update to
`.claude/.claude-plugin/plugin.json`:

`scripts/hooks/plugin-version-bump.sh` line 52:
```bash
# OLD:
CURRENT_VERSION=$(jq -r '.version' .claude-plugin/plugin.json)
# NEW:
CURRENT_VERSION=$(jq -r '.version' .claude/.claude-plugin/plugin.json)
```

Same for lines 66-67 (update), and line 74 (git add).

`scripts/release/bump-plugin-version.sh` line 45:
```bash
# OLD:
CURRENT=$(jq -r '.version' .claude-plugin/plugin.json)
# NEW:
CURRENT=$(jq -r '.version' .claude/.claude-plugin/plugin.json)
```

Same for lines 81-82 (update), and line 94 (git add).

#### 2.1.4 Fix Version Sync in `release.sh`

Add `plugin.json` update after existing `marketplace.json` update:
```bash
# After marketplace.json update:
jq --arg ver "$VERSION_NUM" '.version = $ver' \
  .claude/.claude-plugin/plugin.json > .claude/.claude-plugin/plugin.json.tmp
mv .claude/.claude-plugin/plugin.json.tmp .claude/.claude-plugin/plugin.json
echo "✓ plugin.json updated to $VERSION_NUM"
```

Add verification step:
```bash
MARKET_VER=$(jq -r '.plugins[0].version' .claude-plugin/marketplace.json)
PLUGIN_VER=$(jq -r '.version' .claude/.claude-plugin/plugin.json)
if [ "$MARKET_VER" != "$PLUGIN_VER" ]; then
  echo "❌ ERROR: Version mismatch: marketplace=$MARKET_VER plugin=$PLUGIN_VER"
  exit 1
fi
```

#### 2.1.5 Update Validation Hooks

**`scripts/hooks/validate-marketplace.sh`** — add `plugin.json` checks:
```bash
# Add after existing marketplace.json check:
PLUGIN_JSON=".claude/.claude-plugin/plugin.json"
if [ ! -f "$PLUGIN_JSON" ]; then
    echo "ERROR: $PLUGIN_JSON does not exist"
    exit 1
fi
MARKET_VER=$(jq -r '.plugins[0].version' .claude-plugin/marketplace.json)
PLUGIN_VER=$(jq -r '.version' "$PLUGIN_JSON")
if [ "$MARKET_VER" != "$PLUGIN_VER" ]; then
    echo "ERROR: Version mismatch: marketplace=$MARKET_VER plugin=$PLUGIN_VER"
    exit 1
fi
```

**`scripts/hooks/check-version-sync.sh`** — add `plugin.json` parity check:
```bash
# Add after existing marketplace check:
PLUGIN_VER=$(jq -r '.version' .claude/.claude-plugin/plugin.json 2>/dev/null)
if [ "$MARKET" != "$PLUGIN_VER" ] && [ -n "$PLUGIN_VER" ]; then
    echo "⚠️  plugin.json version mismatch: marketplace=$MARKET plugin=$PLUGIN_VER"
fi
```

#### 2.1.6 Fix Agent and Command Filtering in `sync-plugin-files.sh`

Replace the wildcard agent copy (line 76) with an explicit list matching `plugin.json`:
```bash
# OLD:
cp .claude/agents/*.md $(DIST_DIR)/agents/ 2>/dev/null || true

# NEW: Copy only published agents
PUBLISHED_AGENTS="iteration-executor iteration-prompt-designer knowledge-extractor project-planner stage-executor"
for agent in $PUBLISHED_AGENTS; do
    cp ".claude/agents/${agent}.md" "$(DIST_DIR)/agents/" 2>/dev/null || true
done
```

Also sync the 3 missing commands to `dist/commands/`:
```bash
# Add to command sync section:
PUBLISHED_COMMANDS="meta prompt-find prompt-list prompt-show"
for cmd in $PUBLISHED_COMMANDS; do
    cp ".claude/commands/${cmd}.md" "$(DIST_DIR)/commands/" 2>/dev/null || true
done
```

#### 2.1.7 Fix `make bundle-release` Missing Skills

Add skills to the Makefile `bundle-release` target (line 391):
```makefile
mkdir -p $$BUNDLE_DIR/bin $$BUNDLE_DIR/commands $$BUNDLE_DIR/agents $$BUNDLE_DIR/skills $$BUNDLE_DIR/.claude-plugin $$BUNDLE_DIR/lib;
```
And add (after line 398):
```makefile
cp -r $(DIST_DIR)/skills/* $$BUNDLE_DIR/skills/ 2>/dev/null || true; \
```

#### 2.1.8 Include `plugin.json` and `.mcp.json` in Release Archive

The GitHub Actions workflow (line 78) copies `.claude-plugin/*` from the repo root, which
only includes `marketplace.json`. Since `plugin.json` now lives inside `.claude/.claude-plugin/`,
it must be separately copied into the archive:

```yaml
# In "Create plugin packages" step:
# Copy plugin.json from plugin source into archive's .claude-plugin/
cp .claude/.claude-plugin/plugin.json $PKG_DIR/.claude-plugin/

# Copy .mcp.json to archive root
cp .claude/.mcp.json $PKG_DIR/
```

**Archive `marketplace.json` source rewrite**: The archive has content at root (not under
`.claude/`). Rewrite `source` for archive context:
```yaml
jq '.plugins[0].source = "."' $PKG_DIR/.claude-plugin/marketplace.json \
  > $PKG_DIR/.claude-plugin/marketplace.json.tmp
mv $PKG_DIR/.claude-plugin/marketplace.json.tmp $PKG_DIR/.claude-plugin/marketplace.json
```

With `source: "."`, the archive's `plugin.json` should be at
`$PKG_DIR/.claude-plugin/plugin.json` (which is where we just copied it).

#### 2.1.9 Fix `uninstall.sh`

Two fixes needed:

**a) MCP cleanup** (replace lines 66-68):
```bash
MCP_CONFIG="${HOME}/.claude/mcp.json"
if [ -f "$MCP_CONFIG" ] && command -v jq >/dev/null 2>&1; then
    if jq -e '.mcpServers["meta-cc"]' "$MCP_CONFIG" >/dev/null 2>&1; then
        jq 'del(.mcpServers["meta-cc"])' "$MCP_CONFIG" > "$MCP_CONFIG.tmp"
        mv "$MCP_CONFIG.tmp" "$MCP_CONFIG"
        info "MCP server registration removed"
    fi
else
    warn "Could not auto-remove MCP config. Manually remove 'meta-cc' from $MCP_CONFIG"
fi
```

**b) Agent removal glob** (lines 56-61 remove `meta-*` but agents aren't prefixed `meta-`):
```bash
# OLD:
if ls "$CLAUDE_DIR/agents/meta-"* >/dev/null 2>&1; then
    rm -f "$CLAUDE_DIR/agents/meta-"* 2>/dev/null || true

# NEW: Remove known agent files by name
for agent in iteration-executor iteration-prompt-designer knowledge-extractor project-planner stage-executor; do
    rm -f "$CLAUDE_DIR/agents/${agent}.md" 2>/dev/null || true
done
info "Agent files removed"
```

Same fix for commands (line 48-53): the `/meta` command is `meta.md`, which the `meta-*` glob
does match, but only by coincidence.

#### 2.1.10 Add Smoke Tests for `plugin.json` and `.mcp.json`

```bash
test_plugin_json() {
    # Check plugin.json in archive's .claude-plugin/
    MANIFEST="$EXTRACT_DIR/.claude-plugin/plugin.json"
    if [ ! -f "$MANIFEST" ]; then
        echo "  ✗ FAIL: .claude-plugin/plugin.json not found in archive"
        return 1
    fi
    PLUGIN_VER=$(jq -r '.version' "$MANIFEST")
    MARKET_VER=$(jq -r '.plugins[0].version' "$EXTRACT_DIR/.claude-plugin/marketplace.json")
    if [ "$PLUGIN_VER" != "$MARKET_VER" ]; then
        echo "  ✗ FAIL: Version mismatch: plugin=$PLUGIN_VER marketplace=$MARKET_VER"
        return 1
    fi
    echo "  ✓ PASS: plugin.json valid, version=$PLUGIN_VER"
}

test_mcp_json() {
    if [ ! -f "$EXTRACT_DIR/.mcp.json" ]; then
        echo "  ✗ FAIL: .mcp.json not found"
        return 1
    fi
    if ! jq -e '.mcpServers["meta-cc"]' "$EXTRACT_DIR/.mcp.json" >/dev/null 2>&1; then
        echo "  ✗ FAIL: .mcp.json missing meta-cc server"
        return 1
    fi
    echo "  ✓ PASS: .mcp.json valid"
}

test_no_dev_agents() {
    for agent in feature-developer phase-planner-executor; do
        if [ -f "$EXTRACT_DIR/agents/${agent}.md" ]; then
            echo "  ✗ FAIL: Dev-only agent ${agent}.md found in archive"
            return 1
        fi
    done
    echo "  ✓ PASS: No dev-only agents in archive"
}
```

#### 2.1.11 Retire `lib/mcp-config.json`

Add deprecation notice. Remove in a future release after `.mcp.json` is validated.

---

### 2.2 Option M: Minimal Changes

Implement Section 2.1 steps only. No structural reorganization.

**Summary of changes**:
- Remove `strict: false` from `marketplace.json`
- Create `.claude/.claude-plugin/plugin.json`
- Create `.claude/.mcp.json`
- Fix both bump scripts (correct `plugin.json` path)
- Update `release.sh` (version sync)
- Update validation hooks (parity checks)
- Fix agent filtering in `sync-plugin-files.sh`
- Fix `make bundle-release` missing skills
- Update release workflow (copy `plugin.json`, `.mcp.json`, rewrite `source` in archive)
- Fix `uninstall.sh` (MCP cleanup + agent glob)
- Add smoke tests

**What does NOT change**:
- `.claude/` remains the plugin source directory
- `dist/` and `sync-plugin-files.sh` remain
- `marketplace.json` `source: "./.claude"` remains for repo-based install
- Dev workflow is unchanged

**Residual issue**: `.claude/` dual-role problem remains. Deferred to Option F.

---

### 2.3 Option F: Full Restructuring

Build on Option M. Introduce `plugin/` as the canonical plugin root.

**Plugin content inventory**:
- **Published agents** (5): `iteration-executor`, `iteration-prompt-designer`,
  `knowledge-extractor`, `project-planner`, `stage-executor`
- **Dev-only agents** (2, NOT shipped): `feature-developer`, `phase-planner-executor`
- **Published skills** (18): all current `.claude/skills/` entries
- **Published commands** (4): `meta`, `prompt-find`, `prompt-list`, `prompt-show`

**Target structure**:
```
meta-cc/
├── .claude-plugin/
│   └── marketplace.json         # source → "./plugin" (updated)
├── plugin/                      # NEW: clean plugin source
│   ├── .claude-plugin/
│   │   └── plugin.json          # CORRECT location: inside plugin source
│   ├── .mcp.json
│   ├── commands/
│   │   └── meta.md
│   ├── agents/                  # 5 published agents only
│   └── skills/                  # 18 published skills
├── .claude/                     # dev settings only (NOT shipped)
│   ├── agents/
│   │   ├── feature-developer.md
│   │   └── phase-planner-executor.md
│   └── settings.local.json
└── dist/                        # REMOVED
```

**Phase F1: Create `plugin/` (atomic commit)**
1. Create `plugin/` with `.claude-plugin/plugin.json`, `.mcp.json`, `commands/`, `agents/`
   (published only), `skills/`
2. Update `.claude-plugin/marketplace.json`: `source` → `"./plugin"`, remove `strict: false`
3. Verify `claude plugin install` works

**Phase F2: Remove `dist/` (atomic commit)**
4. Update release workflow: replace `dist/` with `plugin/`
5. Delete `dist/`, `scripts/sync-plugin-files.sh`, Makefile `sync-plugin-files` target
6. Update `make bundle-release` to use `plugin/` as source

**Phase F3: Dev cleanup**
7. Remove `.claude/commands/`, `.claude/skills/`, published agents from `.claude/agents/`
8. Update `CLAUDE.md`, `docs/guides/plugin-development.md`
9. Update bump scripts to read from `plugin/.claude-plugin/plugin.json`
10. Remove `lib/mcp-config.json`

---

## 3. Prerequisites: PoC Validation

**PoC 1: `${CLAUDE_PLUGIN_ROOT}` resolution and `.mcp.json` format**

Create a minimal test plugin, install via `claude plugin install`. Verify:
1. Does `${CLAUDE_PLUGIN_ROOT}` resolve? To what path?
2. Does `.mcp.json` auto-start the MCP server?
3. Which `.mcp.json` format works: flat (`{"server-name": {...}}`) or wrapped
   (`{"mcpServers": {"server-name": {...}}}`)?
4. Windows `.exe` handling?

**PoC 2: `strict: true` (or absent) behavior**

Verify that removing `strict: false` and adding `plugin.json` causes the plugin manager
to read content definitions from `plugin.json` rather than `marketplace.json`. Confirm no
loading conflict.

**PoC 3: Command frontmatter format**

`meta.md` has `name: meta` frontmatter. Verify it loads as a command from `commands/`
without being confused with a skill.

**PoC 4: Explicit path vs. directory path**

Test `"commands": "./commands"` vs `"commands": ["./commands/meta.md"]`. Only switch to
directory paths if confirmed working.

---

## 4. Skill Namespacing and Migration

With `plugin.json` declaring `"name": "meta-cc"`, skills become namespaced:

| Current | Post-migration |
|---------|----------------|
| `/testing-strategy` | `/meta-cc:testing-strategy` |
| `/error-recovery` | `/meta-cc:error-recovery` |
| `/meta` | `/meta-cc:meta` |

Include migration note in release CHANGELOG. The `install.sh` fallback path continues to
install unnamespaced skills to `~/.claude/skills/`.

---

## 5. User Experience Impact

### Before

```bash
curl -LO .../meta-cc-plugin-linux-amd64.tar.gz
tar -xzf meta-cc-plugin-linux-amd64.tar.gz && cd meta-cc-plugin-linux-amd64
./install.sh
claude mcp add meta-cc meta-cc-mcp   # manual MCP registration
```

### After

```bash
claude plugin marketplace add yaleh/meta-cc
claude plugin install meta-cc@yaleh/meta-cc   # MCP auto-registered

# Capabilities still require manual setup (out of scope):
export META_CC_CAPABILITY_SOURCES="yaleh/meta-cc@main/commands"
```

---

## 6. Files Changed Summary

### Option M

| File | Action | Notes |
|------|--------|-------|
| `.claude/.claude-plugin/plugin.json` | CREATE | Plugin manifest at correct location |
| `.claude/.mcp.json` | CREATE | MCP auto-start |
| `.claude-plugin/marketplace.json` | UPDATE | Remove `strict: false`; add 3 missing commands |
| `scripts/hooks/plugin-version-bump.sh` | UPDATE | Fix `plugin.json` path to `.claude/.claude-plugin/` |
| `scripts/release/bump-plugin-version.sh` | UPDATE | Fix `plugin.json` path to `.claude/.claude-plugin/` |
| `scripts/release/release.sh` | UPDATE | Also updates `plugin.json` + parity check |
| `scripts/hooks/validate-marketplace.sh` | UPDATE | Add `plugin.json` checks |
| `scripts/hooks/check-version-sync.sh` | UPDATE | Add `plugin.json` parity |
| `scripts/sync-plugin-files.sh` | UPDATE | Filter dev-only agents; sync all 4 commands |
| `scripts/ci/smoke-tests.sh` | UPDATE | Add `plugin.json`, `.mcp.json`, agent filter tests |
| `scripts/install/uninstall.sh` | UPDATE | MCP cleanup + fix agent glob |
| `.github/workflows/release.yml` | UPDATE | Copy `plugin.json`, `.mcp.json`; rewrite archive `source` |
| `Makefile` | UPDATE | Fix `bundle-release` missing skills |
| `lib/mcp-config.json` | UPDATE | Add deprecation notice |

### Option F Additional

| File | Action | Notes |
|------|--------|-------|
| `plugin/` | CREATE | Clean plugin source (Phase F1) |
| `plugin/.claude-plugin/plugin.json` | MOVE | From `.claude/.claude-plugin/` (Phase F1) |
| `.claude-plugin/marketplace.json` | UPDATE | `source: "./plugin"` (Phase F1) |
| `.claude/commands/` | REMOVE | Phase F3 |
| `.claude/skills/` | REMOVE | Phase F3 |
| `.claude/agents/*.md` (published) | REMOVE | Phase F3 |
| `dist/` | REMOVE | Phase F2 |
| `scripts/sync-plugin-files.sh` | REMOVE | Phase F2 |
| `.github/workflows/release.yml` | UPDATE | Use `plugin/` (Phase F2) |
| `Makefile` | UPDATE | Remove `sync-plugin-files` (Phase F2) |
| `lib/mcp-config.json` | REMOVE | Phase F3 |
| `CLAUDE.md`, `README.md` | UPDATE | Phase F3 |

---

## 7. Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Removing `strict: false` changes how Claude Code resolves plugin content | High | Medium | PoC 2 required; verify content arrays are read from `plugin.json` when `strict` is absent |
| `${CLAUDE_PLUGIN_ROOT}` resolves incorrectly | High | Medium | PoC 1 required before `.mcp.json` ships |
| `plugin.json` + `marketplace.json` content arrays conflict | High | Low | Keep arrays identical in both files; add CI parity check |
| Archive `source` rewrite creates divergence between repo and archive | Medium | Medium | Test archive install path explicitly in smoke tests |
| Namespace change breaks existing users | Medium | High | CHANGELOG notice; `install.sh` path unaffected |
| Phase F1/F2 atomicity failures | High | Medium | Feature branch; single-commit merges; tag only after |
| `/meta` silent failure without capabilities | Medium | High | Document in README; consider graceful detection |
| Windows `.exe` path handling in `.mcp.json` | Medium | Medium | PoC 1 on Windows |
| `plugin.json` and `marketplace.json` version drift | Medium | Low | Three hooks enforce parity; `release.sh` updates both |

---

## 8. Alternatives Considered

**Option X: Keep `strict: false`, skip `plugin.json` entirely**
Continue using `marketplace.json` as the sole plugin definition. This works but prevents
namespacing, lifecycle management, and `.mcp.json` integration. Rejected: does not advance
toward the standard.

**Option Y: npm Distribution**
Handles platform binaries cleanly but adds Node.js dependency. Deferred.

---

## 9. Out of Scope

- Capabilities distribution (making `/meta` work out of the box).
- Per-platform marketplace entries for binary distribution.
- npm packaging.

---

## 10. Open Questions

1. **`${CLAUDE_PLUGIN_ROOT}` resolution path**: Exact directory when installed via
   `claude plugin install`? (Blocks `.mcp.json`.)

2. **`strict: false` → `strict: true` behavior**: What exactly changes when `strict` is
   removed? Does `plugin.json` fully override `marketplace.json` content arrays, or are
   they merged? (Blocks 2.1.1.)

3. **Directory vs. explicit paths in `plugin.json`**: Does `"commands": "./commands"` work
   equivalently to `"commands": ["./commands/meta.md"]`? (Blocks format choice.)

4. **`meta.md` frontmatter and command loading**: Is it correctly loaded as a command
   (not a skill)? (PoC 3.)

5. **`extraKnownMarketplaces` in `.claude/settings.json`**: Auto-prompts contributors.
   Desired for team; noisy for reference-only cloners.

6. **Archive `plugin.json` location**: When the archive is extracted and used as a plugin
   source with `source: "."`, does the plugin manager look for `plugin.json` at
   `.claude-plugin/plugin.json` within the extracted directory? (The archive currently has
   `.claude-plugin/` at root containing only `marketplace.json`.)

---

## 11. Success Criteria

### Option M

- [ ] `scripts/hooks/plugin-version-bump.sh` works on `.claude/` file commits
- [ ] `scripts/release/bump-plugin-version.sh` runs without error
- [ ] `claude plugin marketplace add yaleh/meta-cc` + `claude plugin install` works from repo
- [ ] MCP server starts automatically on plugin load (after PoC 1)
- [ ] 18 skills discoverable as `meta-cc:skill-name`
- [ ] No dev-only agents in release archives
- [ ] `uninstall.sh` removes MCP registration and agent files
- [ ] `plugin.json` version equals `marketplace.json` version (enforced at 3 points)
- [ ] Smoke tests pass including new checks

### Option F Additional

- [ ] `dist/` removed; `plugin/` is the single source
- [ ] `make push` passes
- [ ] Dev-only agents exist only in `.claude/agents/`
- [ ] `plugin.json` at `plugin/.claude-plugin/plugin.json`

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-08 | Claude Code | Initial proposal |
| 1.1 | 2026-03-08 | Claude Code | Fix skills format; remove meta skill; exclude dev-only agents |
| 1.2 | 2026-03-08 | Claude Code | Correct false diagnosis on release archive; broken bump scripts; PoC requirements; Option M vs F |
| 1.3 | 2026-03-08 | Claude Code | Archive layout mismatch; two broken bump scripts; explicit paths; uninstall.sh; smoke tests |
| 1.4 | 2026-03-08 | Claude Code | **Critical**: Fix `plugin.json` location (must be inside plugin source at `.claude/.claude-plugin/plugin.json`, not repo root). Discover `strict: false` conflict — must remove before adding `plugin.json`. Dev-only agents leak into releases. `make bundle-release` missing skills. `uninstall.sh` agent glob broken. `check-version-sync.sh` incomplete. Add PoC 2 for strict mode behavior. |
| 1.5 | 2026-03-08 | Claude Code | **Discovery**: 3 slash commands (`prompt-find`, `prompt-list`, `prompt-show`) exist in `.claude/commands/` but not in `marketplace.json` or `dist/` — invisible to plugin users. `plugin.json` must declare all 4 commands. `.mcp.json` format uncertainty: flat vs `mcpServers` wrapper — added to PoC 1. Fixed `sync-plugin-files.sh` line reference (76 not 366). Added command sync to 2.1.6. |

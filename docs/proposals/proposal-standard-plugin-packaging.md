# Proposal: Standardize Plugin Packaging to Claude Code Official Spec

**Status**: Draft for Review
**Date**: 2026-03-08
**Author**: Claude Code Analysis

## Executive Summary

This proposal aligns meta-cc's plugin packaging with the official Claude Code plugin ecosystem
standard. The release archive structure is already closer to the standard than it first appears:
it already bundles the MCP binary alongside flat `commands/`, `agents/`, `skills/` directories.
The actual gaps are narrower and more precise:

1. **No `plugin.json`** — `claude plugin install` cannot parse the plugin; the
   `bump-plugin-version.sh` script that already references `plugin.json` is currently broken.
2. **No `.mcp.json`** — MCP server is not wired into the plugin lifecycle; users must register
   it manually.
3. **Plugin source in `.claude/`** — mixing dev-only files with distributable content; `dist/`
   exists only to compensate for this.

The proposal provides two implementation options of different scope:
- **Option M (Minimal)**: Add `plugin.json` and `.mcp.json` to the existing `.claude/` plugin
  source. No structural reorganization. Fixes the broken bump script and enables
  `claude plugin install` with minimal disruption.
- **Option F (Full)**: Introduce `plugin/` as a dedicated plugin root, remove `dist/`, clean
  up the dual-role `.claude/`. Maximally aligned with the standard.

---

## 1. Accurate Current State Assessment

### 1.1 Release Archive Structure (Already Correct)

Contrary to naive inspection, the GitHub Actions release workflow
(`.github/workflows/release.yml` lines 61–94) already produces a release archive with the
binary co-located with plugin content:

```
meta-cc-plugin-${VERSION}-${PLATFORM}/
├── bin/
│   └── meta-cc-mcp              ← binary ALREADY included (lines 70–75)
├── .claude-plugin/
│   └── marketplace.json         ← no plugin.json yet
├── commands/                    ← from dist/ (via sync-plugin-files.sh)
├── agents/                      ← from dist/
├── skills/                      ← from dist/ (all 18 skills)
├── lib/
│   └── mcp-config.json          ← legacy MCP config
├── install.sh
├── uninstall.sh
├── README.md
└── LICENSE
```

The archive layout already matches the standard plugin structure. The `bin/meta-cc-mcp` binary
is already present at `${PLUGIN_ROOT}/bin/meta-cc-mcp`. The only things missing from this
archive are `plugin.json` and `.mcp.json`.

### 1.2 The `dist/` Directory: Limited Scope Problem

`dist/` is not a general "mess" — it solves a specific problem: the release archive needs flat
`commands/agents/skills/` directories, but those files live under `.claude/` in the repo.
`sync-plugin-files.sh` produces the flat layout. This is the exact problem a dedicated
`plugin/` directory would eliminate, but `dist/` makes it work today.

### 1.3 Broken `bump-plugin-version.sh`

`scripts/release/bump-plugin-version.sh` line 45 reads:
```bash
CURRENT=$(jq -r '.version' .claude-plugin/plugin.json)
```
But `.claude-plugin/plugin.json` does not exist. The script fails on first invocation.
This is the most immediate bug this proposal must fix, regardless of which option is chosen.

### 1.4 Missing `plugin.json` — Precise Consequences

Without `plugin.json`:
- `claude plugin install` cannot identify, namespace, or load the plugin
- Skills have no `meta-cc:` namespace prefix
- `bump-plugin-version.sh` is broken (see 1.3)
- `.mcp.json` cannot be referenced from a manifest

### 1.5 Missing `.mcp.json` — MCP Server Not Auto-Started

Currently users must manually run `claude mcp add meta-cc meta-cc-mcp` after installation.
With `.mcp.json` pointing to `${CLAUDE_PLUGIN_ROOT}/bin/meta-cc-mcp`, the plugin manager
starts the MCP server automatically. The release archive already has the binary at the right
relative path.

### 1.6 The `/meta` Command Depends on MCP Server AND Capabilities

The `/meta` slash command (`commands/meta.md` line 16) calls `mcp_meta_cc.list_capabilities()`.
This means:
1. The MCP server must be running (blocked by missing `.mcp.json` — see 1.5)
2. The MCP server must have access to capability content files from `capabilities/`

`capabilities/` is distributed as a **separate** `capabilities-latest.tar.gz` archive, not
inside the plugin package. Users must set `META_CC_CAPABILITY_SOURCES` to point to the
unpacked capabilities directory. Without this, `/meta` will run but return no capabilities.

**Scope note**: Integrating `capabilities/` into the plugin package is out of scope for this
proposal. The proposal explicitly targets the plugin binary/manifest/MCP wiring layer. The
capabilities distribution model is addressed separately.

### 1.7 `.claude/` Dual-Role Problem

`.claude/` serves two purposes:
1. **Dev settings for this repo**: `settings.local.json`, `worktrees/`, `experiments/`
2. **Plugin distribution source**: `commands/`, `agents/`, `skills/`

`marketplace.json` currently points `"source": "./.claude"`, so the entire `.claude/`
directory including dev-only files is the plugin root. `dist/` exists to compensate.

---

## 2. Proposed Changes

### 2.1 Shared Changes (Both Options)

These changes are required regardless of option chosen.

#### 2.1.1 Create `.claude-plugin/plugin.json`

This is the highest-priority fix: it unblocks `bump-plugin-version.sh` and enables
`claude plugin install`.

Per the official spec, `commands`, `agents`, and `skills` fields accept directory paths.
All entries in the specified directory are auto-discovered. Enumeration of individual files
is not required.

**For Option M** (`.claude/` remains the plugin source, `marketplace.json` source = `./.claude`):
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
  "commands": "./commands",
  "agents": "./agents",
  "skills": "./skills"
}
```

**For Option F** (repo root is plugin source, all content under `plugin/`):
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
  "mcpServers": "./plugin/.mcp.json",
  "commands": "./plugin/commands",
  "agents": "./plugin/agents",
  "skills": "./plugin/skills"
}
```

#### 2.1.2 Add `.mcp.json`

**For Option M** — place at `.claude/.mcp.json` (inside the plugin source directory):
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

**For Option F** — place at `plugin/.mcp.json`.

`${CLAUDE_PLUGIN_ROOT}` is resolved by Claude Code to the plugin's cache directory at runtime.
**This variable must be verified empirically before Phase 3** (see Section 4, PoC requirement).
The release archive already places the binary at `bin/meta-cc-mcp` relative to the archive
root, so the path is structurally correct.

#### 2.1.3 Include `plugin.json` in Release Archive

Update `.github/workflows/release.yml` step "Create plugin packages" (line 78):
```yaml
# Existing:
cp -r .claude-plugin/* $PKG_DIR/.claude-plugin/

# This already copies plugin.json once it exists — no change to workflow needed.
```

No workflow change is required: line 78 already copies all files from `.claude-plugin/`.
Once `plugin.json` is created in `.claude-plugin/`, it will be included automatically.

#### 2.1.4 Add `.mcp.json` to Release Archive

For the archive-based install path, `.mcp.json` must be at the archive root (alongside
`commands/`, `agents/`, `skills/`). Add to the release workflow:

```yaml
# After existing cp commands in "Create plugin packages" step:
cp plugin/.mcp.json $PKG_DIR/   # Option F
# or:
cp .claude/.mcp.json $PKG_DIR/  # Option M
```

#### 2.1.5 Retire `lib/mcp-config.json`

With `.mcp.json` wired into the plugin, `lib/mcp-config.json` (the legacy manual MCP config)
becomes redundant. It should remain in the archive for one release cycle as a fallback for
users on older Claude Code versions that do not support the plugin system, then be removed.
Update `lib/mcp-config.json` to add a deprecation notice in its next release.

#### 2.1.6 Version Sync: `plugin.json` + `marketplace.json`

`bump-plugin-version.sh` already updates both files atomically (lines 79–89). Once
`plugin.json` is created, the script works as intended. No changes needed to the script.

`release.sh` (line 132) only updates `marketplace.json`. Update it to also update
`plugin.json`:
```bash
# After existing marketplace.json update:
jq --arg ver "$VERSION_NUM" '.version = $ver' .claude-plugin/plugin.json > .claude-plugin/plugin.json.tmp
mv .claude-plugin/plugin.json.tmp .claude-plugin/plugin.json
echo "✓ plugin.json updated to $VERSION_NUM"
```

Note: The Go binary version is set via git tag ldflags at build time — it is independent
and does not need to match the plugin version file.

---

### 2.2 Option M: Minimal Changes

**Scope**: Add `plugin.json` and `.mcp.json` to `.claude/`. No structural changes.

**Additional steps beyond 2.1**:

1. Create `.claude/.mcp.json` (Section 2.1.2, Option M content)
2. Create `.claude-plugin/plugin.json` (Section 2.1.1, Option M content — paths relative
   to `.claude/` since `source: "./.claude"` in marketplace.json stays unchanged)
3. Update `release.sh` to also bump `plugin.json` (Section 2.1.6)
4. Update `validate-marketplace.sh` to also check `plugin.json` version equality

**What does NOT change**:
- `.claude/` structure
- `dist/` and `sync-plugin-files.sh`
- `marketplace.json` source field (remains `./.claude`)
- Release workflow (`.claude-plugin/` already copied at line 78)
- Dev workflow

**Result**: `bump-plugin-version.sh` works; `claude plugin install meta-cc@yaleh/meta-cc`
installs skills/agents/command with MCP auto-start. The `.claude/` dual-role issue remains
but is deferred.

---

### 2.3 Option F: Full Restructuring

**Scope**: Introduce `plugin/` as dedicated plugin root, separate from `.claude/` dev settings,
and eliminate `dist/`.

**Plugin content inventory**:
- **Published agents** (5): `iteration-executor`, `iteration-prompt-designer`,
  `knowledge-extractor`, `project-planner`, `stage-executor`
- **Dev-only agents** (2, NOT shipped): `feature-developer`, `phase-planner-executor`
- **Published skills** (18): `agent-prompt-evolution`, `api-design`,
  `baseline-quality-assessment`, `build-quality-gates`, `ci-cd-optimization`,
  `code-refactoring`, `cross-cutting-concerns`, `dependency-health`,
  `documentation-management`, `error-recovery`, `knowledge-transfer`,
  `methodology-bootstrapping`, `observability-instrumentation`, `rapid-convergence`,
  `retrospective-validation`, `subagent-prompt-construction`, `technical-debt-management`,
  `testing-strategy`
- **Published command** (1): `meta` (slash command, NOT a skill — stays in `commands/`)

**Target structure**:
```
meta-cc/                         # GitHub repo root = marketplace root
├── .claude-plugin/
│   ├── marketplace.json         # source → "." (updated)
│   └── plugin.json              # NEW
├── plugin/                      # NEW: canonical plugin root
│   ├── .mcp.json                # NEW
│   ├── commands/
│   │   └── meta.md
│   ├── agents/                  # 5 published agents only
│   │   ├── iteration-executor.md
│   │   ├── iteration-prompt-designer.md
│   │   ├── knowledge-extractor.md
│   │   ├── project-planner.md
│   │   └── stage-executor.md
│   └── skills/                  # 18 published skills
│       └── */SKILL.md
├── bin/                         # compiled binaries (built by CI)
│   └── meta-cc-mcp
├── .claude/                     # dev settings only (NOT shipped)
│   ├── agents/                  # dev-only agents
│   │   ├── feature-developer.md
│   │   └── phase-planner-executor.md
│   └── settings.local.json
└── dist/                        # REMOVED
```

**Additional steps beyond 2.1** (after Option M is stable):

**Phase F1: Create `plugin/` (atomic)**
1. Create `plugin/commands/`, `plugin/agents/`, `plugin/skills/`
2. Copy (not move yet) published content from `.claude/` to `plugin/`
3. Create `plugin/.mcp.json`
4. Create `.claude-plugin/plugin.json` (Option F variant, `source: "."`)
5. Update `.claude-plugin/marketplace.json` `source` to `"."`

**CRITICAL**: Steps 4 and 5 must be in the same commit. Between step 4 and 5 there is no
valid state: the marketplace points to `.claude/` but `plugin.json` references `./plugin/`.

6. Verify `claude plugin install` works end-to-end before proceeding

**Phase F2: Build pipeline (atomic)**
7. Update `.github/workflows/release.yml`:
   - Remove `bash scripts/sync-plugin-files.sh` step (line 36)
   - Replace `dist/` references with `plugin/` (lines 79–81)
   - Add `cp plugin/.mcp.json $PKG_DIR/`
8. Delete `scripts/sync-plugin-files.sh`
9. Remove `sync-plugin-files` Makefile target
10. Run dry-run release to verify

**CRITICAL**: Steps 7–9 must be in the same commit and released atomically. Between removing
`sync-plugin-files.sh` (step 8) and updating the workflow (step 7), any tag-triggered release
will fail. Work on a feature branch; only merge once all steps are complete.

**Phase F3: Dev cleanup**
11. Remove `.claude/commands/` (now in `plugin/commands/`)
12. Remove `.claude/skills/` (now in `plugin/skills/`)
13. Remove published agents from `.claude/agents/` (keep dev-only agents)
14. Update `CLAUDE.md`, `docs/guides/plugin-development.md` to reference `plugin/`
15. Add CI check: verify no duplicate files between `.claude/` and `plugin/`

---

## 3. User Experience Impact

### Before

```bash
# Download release archive
curl -LO https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-linux-amd64.tar.gz
tar -xzf meta-cc-plugin-linux-amd64.tar.gz
cd meta-cc-plugin-linux-amd64

# Install files
./install.sh

# Register MCP server manually
claude mcp add meta-cc meta-cc-mcp

# Restart Claude Code
```

### After Option M or F

```bash
# Add marketplace (one-time per machine)
claude plugin marketplace add yaleh/meta-cc

# Install — plugin files + MCP server auto-registered
claude plugin install meta-cc@yaleh/meta-cc

# Configure capabilities (still required; out of scope for this proposal)
export META_CC_CAPABILITY_SOURCES="yaleh/meta-cc@main/commands"
```

---

## 4. Prerequisites: PoC Validation Before Phase 3

Before implementing any CI pipeline changes, validate the following empirically.
These are unknowns that, if wrong, invalidate the entire MCP integration approach.

**PoC 1: `${CLAUDE_PLUGIN_ROOT}` resolution**

In a test plugin, create `.mcp.json`:
```json
{"mcpServers": {"test": {"command": "${CLAUDE_PLUGIN_ROOT}/bin/echo-server"}}}
```
Place a trivial binary at `bin/echo-server`. Install the test plugin via
`claude plugin install`. Verify the MCP server starts. Confirm the resolved path.

**PoC 2: Directory auto-discovery for `commands`, `agents`, `skills`**

Verify that `"commands": "./commands"` in `plugin.json` auto-discovers all `.md` files in
the directory (not just `SKILL.md`-style frontmatter). The `/meta` command uses `name: meta`
frontmatter that resembles a skill; confirm it is treated as a command when placed in
`commands/` vs. `skills/`.

**PoC 3: Namespace behavior**

Verify that skills are invokable as `/meta-cc:testing-strategy` after `plugin.json`
declares `"name": "meta-cc"`. Verify the migration impact on existing users.

---

## 5. Skill Namespacing and Migration

With `plugin.json` declaring `"name": "meta-cc"`, skills become namespaced:

| Current invocation | Post-migration |
|--------------------|----------------|
| `/testing-strategy` | `/meta-cc:testing-strategy` |
| `/error-recovery` | `/meta-cc:error-recovery` |
| `/meta` | `/meta-cc:meta` |

**Breaking change**: Users who installed via the old bash script have unnamespaced skills
in `~/.claude/skills/`. Switching to `claude plugin install` requires relearning invocations.

**Migration strategy**: For one release cycle, provide a note in install output:
```
Note: Skills are now namespaced as /meta-cc:skill-name
      (e.g., /meta-cc:testing-strategy instead of /testing-strategy)
```

---

## 6. Implementation Comparison

| Dimension | Option M (Minimal) | Option F (Full) |
|-----------|-------------------|-----------------|
| Files changed | ~5 files | ~20+ files |
| Risk | Low | Medium |
| Breaks dev workflow | No | Temporarily during F1 |
| Removes `dist/` | No | Yes (Phase F2) |
| Fixes broken `bump-plugin-version.sh` | Yes | Yes |
| Enables `claude plugin install` | Yes | Yes |
| Separates dev/dist content | No | Yes |
| CI window risk | None | Yes (Phase F2 must be atomic) |
| Requires PoC validation first | Yes (MCP path) | Yes (MCP path) |

**Recommendation**: Implement Option M first. It fixes the immediate bugs and enables the
standard install path with minimal risk. Option F can follow as a separate PR once Option M
is validated in production.

---

## 7. Files Changed Summary

### Option M Only

| File | Action | Notes |
|------|--------|-------|
| `.claude-plugin/plugin.json` | CREATE | Fixes `bump-plugin-version.sh`; enables `claude plugin install` |
| `.claude/.mcp.json` | CREATE | MCP server auto-start |
| `scripts/release/release.sh` | UPDATE | Also bump `plugin.json` version |
| `scripts/hooks/validate-marketplace.sh` | UPDATE | Check `plugin.json` version parity |
| `lib/mcp-config.json` | UPDATE | Add deprecation notice |

### Option F Additional (after Option M)

| File | Action | Notes |
|------|--------|-------|
| `plugin/` | CREATE | New canonical plugin root |
| `plugin/.mcp.json` | MOVE | From `.claude/.mcp.json` |
| `.claude-plugin/marketplace.json` | UPDATE | `source: "."` |
| `.claude-plugin/plugin.json` | UPDATE | Paths now reference `./plugin/` |
| `.claude/commands/` | REMOVE | Moved to `plugin/commands/` |
| `.claude/skills/` | REMOVE | Moved to `plugin/skills/` |
| `.claude/agents/*.md` (published) | REMOVE | Moved to `plugin/agents/` |
| `dist/` | REMOVE | Obsolete |
| `scripts/sync-plugin-files.sh` | REMOVE | Replaced by direct `plugin/` |
| `.github/workflows/release.yml` | UPDATE | Replace `dist/` with `plugin/`, add `.mcp.json` copy |
| `Makefile` | UPDATE | Remove `sync-plugin-files` target |
| `CLAUDE.md` | UPDATE | Plugin development paths |
| `README.md` | UPDATE | New install instructions |
| `lib/mcp-config.json` | REMOVE | Superseded by `.mcp.json` |

---

## 8. Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| `${CLAUDE_PLUGIN_ROOT}` resolves incorrectly | High | Medium | PoC 1 required before any CI changes |
| `commands` directory discovery doesn't work for `meta.md` frontmatter format | Medium | Low | PoC 2; if needed, migrate `meta.md` to SKILL.md format |
| Namespace change breaks existing users | Medium | High | One-release migration notice; document in CHANGELOG |
| Phase F2 CI window: tag between removing `dist/` and updating workflow | High | Medium | Execute on feature branch; merge as single commit; release only after merge |
| `.claude/` and `plugin/` drift during Phase F1 (pre-F3) | Medium | High | Add CI lint: `scripts/ci/check-plugin-sync.sh` that errors if same filename exists in both dirs |
| `release.sh` version sync misses `plugin.json` | Medium | Low | Add verification step in release.sh that both files match after update |
| `/meta` command non-functional without capabilities | Medium | High | Document clearly in README; add detection in plugin post-load message |

---

## 9. Out of Scope

The following are explicitly out of scope and should be addressed in separate proposals:

- **Capabilities distribution**: Integrating `capabilities/` into the plugin package so
  `/meta` works immediately after `claude plugin install`. This requires either bundling
  20+ capability files or implementing runtime fetch within the plugin.
- **npm distribution**: Publishing `meta-cc` as an npm package for broader ecosystem reach.
- **Per-platform marketplace entries**: Enabling different `source` entries per OS/arch
  so platform-specific binaries can be distributed via the marketplace mechanism.

---

## 10. Open Questions

1. **`${CLAUDE_PLUGIN_ROOT}` exact resolution path**: What directory does this variable
   resolve to when a plugin is installed via `claude plugin install`? Is it versioned
   (e.g., `~/.claude/plugins/cache/meta-cc@2.3.6/`) or unversioned? The install script
   (Option B fallback) must write the binary to the exact same path.

2. **`commands` field with single file or directory**: Does `"commands": "./commands"` work
   when the directory contains only one `.md` file? Or must it list the file explicitly?
   Confirm `meta.md`'s YAML frontmatter (`name: meta`) is parsed correctly as a command
   (not confused with a skill).

3. **`extraKnownMarketplaces` in `.claude/settings.json`**: Adding this auto-prompts all
   contributors when they open the repo. Is this desired for a dev-facing repo? May cause
   noise for users who clone for reference only. Decision needed before Phase 5 documentation.

4. **Capabilities without `META_CC_CAPABILITY_SOURCES`**: What does `/meta` display when
   the MCP server is running but no capabilities are configured? Should the command degrade
   gracefully with a helpful message, or fail silently?

---

## 11. Success Criteria

### Option M

- [ ] `bump-plugin-version.sh` runs without error
- [ ] `claude plugin marketplace add yaleh/meta-cc` succeeds
- [ ] `claude plugin install meta-cc@yaleh/meta-cc` installs command, 5 agents, 18 skills
- [ ] All 18 skills discoverable as `meta-cc:skill-name`
- [ ] MCP server starts automatically on plugin load (after PoC validates the binary path)
- [ ] `plugin.json` version always equals `marketplace.json` version (enforced by hooks and `release.sh`)
- [ ] Existing `install.sh` users are unaffected

### Option F (additional)

- [ ] `dist/` directory no longer exists in repo
- [ ] `make push` passes with no `dist/` references
- [ ] Dev-only agents (`feature-developer`, `phase-planner-executor`) not installed by plugin
- [ ] CI lint check catches any `.claude/` ↔ `plugin/` duplicate files

---

## Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-03-08 | Claude Code | Initial proposal based on industry standard research |
| 1.1 | 2026-03-08 | Claude Code | Fix skills format (directory path); remove meta skill; exclude dev-only agents; simplify plugin.json |
| 1.2 | 2026-03-08 | Claude Code | Correct false diagnosis: release archive already bundles binary and skills. Identify broken bump-plugin-version.sh as immediate fix. Add PoC validation requirement. Distinguish Option M (minimal) vs Option F (full). Document `/meta` → MCP → capabilities dependency chain. Fix Phase F2 atomicity requirement. Address lib/ fate and three-file version sync. |

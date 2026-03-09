# Proposal: Split meta-cc into Two Focused Projects

**Status**: Approved
**Date**: 2026-03-09
**Author**: Yale Huang

---

## 1. Problem Statement

meta-cc currently bundles two distinct value propositions into a single plugin:

1. **Session history analysis** — A Go MCP server that parses Claude Code JSONL session files and exposes 20 query tools, plus 21 capabilities and 4 slash commands that use those tools for workflow analysis.

2. **Software development methodologies** — 18 validated BAIME-derived skills (testing strategy, CI/CD, error recovery, etc.) and 7 general-purpose development agents (stage-executor, project-planner, iteration-executor, etc.).

These two value propositions have different users, different update cadences, and different dependencies. Bundling them together creates noise for users who only want one, and complicates maintenance.

### Symptoms

- Users installing meta-cc for MCP session analysis receive ~1.5MB of methodology skills they may not need.
- Users who want the development methodology tools must also install and run a Go MCP binary server.
- Version bumps to the MCP server trigger plugin re-installs that rebuild the skills cache unnecessarily.
- The project README conflates two different elevator pitches.
- Skills and agents are general-purpose (no dependency on meta-cc MCP), yet appear as if they are meta-cc-specific.

---

## 2. Proposed Split

### 2.1 Project A: meta-cc (refined scope)

**Tagline**: _Analyze your Claude Code session history via MCP._

**Core principle**: MCP is the primary interface. Everything in this project either produces session data or consumes it.

**Retains**:

| Category | Contents |
|----------|----------|
| MCP Server | Go binary (`meta-cc-mcp`), 20 query tools, session JSONL parser |
| Capabilities | 21 `meta-*` capabilities (all call MCP tools for analysis) |
| Commands | `/meta` (capability router), `/prompt-find`, `/prompt-list`, `/prompt-show` |

**Removes**: All 18 skills and all 7 agents (moved to Project B).

**Result**: Lean plugin — binary + capabilities + 4 commands. Single responsibility: "understand your Claude Code work history."

---

### 2.2 Project B: New Project (Claude Code Development Methodologies)

**Tagline**: _Validated software development methodologies for Claude Code._

**Core principle**: Pure Markdown. No binary. No MCP dependency. Works standalone.

**Contains**:

| Category | Contents |
|----------|----------|
| Skills | All 18 validated methodology skills |
| Agents | All 7 development workflow agents |

#### Skills inventory

| Skill | Domain |
|-------|--------|
| `methodology-bootstrapping` | BAIME framework |
| `testing-strategy` | TDD, coverage-driven gap closure |
| `ci-cd-optimization` | Quality gates, release automation |
| `error-recovery` | 13-category taxonomy, diagnostic workflows |
| `dependency-health` | Security-first, batch remediation |
| `knowledge-transfer` | Progressive learning paths, onboarding |
| `technical-debt-management` | SQALE methodology, prioritization |
| `code-refactoring` | Test-driven refactoring |
| `cross-cutting-concerns` | Error handling, logging, configuration |
| `observability-instrumentation` | Logs, metrics, traces |
| `api-design` | 6 validated patterns |
| `documentation-management` | Templates, patterns, automation |
| `agent-prompt-evolution` | Agent specialization tracking |
| `baseline-quality-assessment` | Rapid convergence enablement |
| `rapid-convergence` | 3-4 iteration methodology development |
| `retrospective-validation` | Historical data validation |
| `subagent-prompt-construction` | Compact Claude Code subagent prompts |
| `build-quality-gates` | Quality enforcement for build/CI |

#### Agents inventory

| Agent | Role |
|-------|------|
| `stage-executor` | Executes project plan stages with validation |
| `project-planner` | Generates TDD-based development plans |
| `iteration-executor` | Executes BAIME experiment iterations |
| `iteration-prompt-designer` | Designs ITERATION-PROMPTS.md files |
| `knowledge-extractor` | Extracts BAIME experiments into skills |

`feature-developer` and `phase-planner-executor` are dev-only agents used in meta-cc's own development workflow. They will be **removed** (not migrated) during the split.

**Result**: Pure content plugin. No binary to build or install. Can be installed by anyone using Claude Code regardless of whether they use meta-cc.

---

## 3. Content Boundary Analysis

### 3.1 Clear-cut assignments

| Content | Assignment | Reason |
|---------|-----------|--------|
| MCP server Go binary | meta-cc | Core product |
| Capabilities (`meta-*.md`) | meta-cc | All call MCP tools |
| `/meta` command | meta-cc | Routes to capabilities |
| `/prompt-*` commands | meta-cc | Operate on `.meta-cc/prompts/library/` |
| `testing-strategy` skill | Project B | No meta-cc dependency |
| `ci-cd-optimization` skill | Project B | No meta-cc dependency |
| `stage-executor` agent | Project B | No meta-cc dependency |
| `project-planner` agent | Project B | No meta-cc dependency |

### 3.2 Borderline items

| Content | Issue | Decision |
|---------|-------|----------|
| `retrospective-validation` | Uses "historical data" — could suggest meta-cc MCP, but actually refers to any historical record | → Project B. The skill is general; users can source historical data independently. |
| `agent-prompt-evolution` | Emerged from BAIME experiments run on meta-cc | → Project B. Tracks agent evolution patterns, not session data. |
| `knowledge-extractor` agent | Extracts BAIME experiments into skills; indirectly related to meta-cc development workflow | → Project B. No runtime dependency on MCP tools. |
| `feature-developer` agent | Dev-only in meta-cc | → **Removed**. Not migrated. |
| `phase-planner-executor` agent | Dev-only in meta-cc | → **Removed**. Not migrated. |

---

## 4. Project B: Name and Location

**Name**: `baime` (`yaleh/baime` on GitHub)

BAIME (Bootstrapped AI Methodology Engineering) is the unifying framework from which all skills and agents derive. Using it as the project name makes the brand explicit and the purpose clear to practitioners familiar with the methodology.

---

## 5. Relationship Between Projects

- The two projects are **independent** — neither requires the other at runtime.
- meta-cc README will mention `baime` as a companion ("if you want development methodology tools, see [yaleh/baime](https://github.com/yaleh/baime)").
- `baime` README will mention meta-cc as a companion ("if you want session history analysis, see [yaleh/meta-cc](https://github.com/yaleh/meta-cc)").
- No shared code, no shared CI, no cross-repo version dependencies.

---

## 6. Migration Plan

### Phase 1: Create `yaleh/baime` repository

1. Create new repo `yaleh/baime` with `plugin.json`, `marketplace.json`.
2. Copy 18 skills and 5 published agents from meta-cc (excluding `feature-developer` and `phase-planner-executor`).
3. Set up minimal CI (JSON/Markdown lint, plugin.json validation).
4. Publish to Claude Code plugin marketplace.

### Phase 2: Prune meta-cc → 3.0.0

1. Remove all 18 skills from `.claude/skills/` and `dist/skills/`.
2. Remove all 7 agents from `.claude/agents/` and `dist/agents/` (5 published + 2 dev-only).
3. Update `plugin.json`: remove `skills` and `agents` arrays.
4. Update `marketplace.json`: remove agent declarations.
5. Update `sync-plugin-files.sh`: remove skills/agents copy logic.
6. Update `README.md` and docs: reflect new scope, link to `yaleh/baime`.
7. Bump to **3.0.0** (breaking: skills and agents no longer bundled with meta-cc).

### Phase 3: Update CI and release tooling in meta-cc

1. Update `test-plugin-json.sh`: set expected skill count to 0, agent count to 0.
2. Update `scripts/ci/smoke-tests.sh`: remove skill/agent assertions.
3. Update `Makefile` bundle-release target: remove skills/agents copy steps.
4. Update `scripts/release/bump-plugin-version.sh` and hooks if needed.

---

## 7. Impact Assessment

### meta-cc after split (3.0.0)

| Metric | Before | After |
|--------|--------|-------|
| Plugin size | ~1.8MB (binary + skills) | ~0.3MB (binary + capabilities) |
| `plugin.json` skills | 18 | 0 |
| `plugin.json` agents | 5 (published) | 0 |
| `plugin.json` commands | 4 | 4 |
| Elevator pitch | "Session analysis + methodology skills" | "Session history analysis via MCP" |

### `yaleh/baime` (new)

| Metric | Value |
|--------|-------|
| Plugin size | ~1.5MB (18 skills + 5 agents) |
| Binary required | None |
| MCP server required | None |
| Works without meta-cc | ✓ |
| Applicable to any Claude Code project | ✓ |
| Agents published | 5 (`stage-executor`, `project-planner`, `iteration-executor`, `iteration-prompt-designer`, `knowledge-extractor`) |

---

## 8. Decisions

| Question | Decision |
|----------|----------|
| Project B name | `baime` (`yaleh/baime`) |
| dev-only agents (`feature-developer`, `phase-planner-executor`) | **Removed** — not migrated to either project |
| meta-cc version after split | **3.0.0** (breaking change) |

---

## 9. Recommendation

Proceed with the split in two phases: create `yaleh/baime` first and validate it as a standalone plugin, then prune meta-cc and release 3.0.0.

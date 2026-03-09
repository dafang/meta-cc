# Plugin 安装标准化方案

## 1. 背景与问题

### 1.1 当前现象

`/mcp` 面板中出现两个 `meta-cc` 条目：

| 条目 | 来源 | 状态 |
|------|------|------|
| `meta-cc · ✔ connected` | `.claude/.mcp.json`（Local MCPs 直接配置） | 运行但冗余 |
| `plugin:meta-cc:meta-cc · ✘ failed` | Claude Code Plugin 系统（Built-in MCPs） | 失败 |

### 1.2 实际目录结构分析

项目存在**两个 plugin 相关目录**：

```
meta-cc/
  .claude-plugin/                     ← 层级 1：Marketplace manifest
    marketplace.json                  ← 声明 "source": "./.claude"
  .claude/                            ← 层级 2：Plugin root（开发模式）
    .claude-plugin/
      plugin.json                     ← 声明 mcpServers: "./.mcp.json", commands: ./commands/
    commands/                         ← 命令文件（相对 plugin root 正确）
    .mcp.json                         ← MCP 配置（相对 plugin root 正确）
    settings.local.json               ← 开发配置
    bin/                              ← 不存在！← 唯一缺失项
```

### 1.3 CLAUDE_PLUGIN_ROOT 的实际解析

`marketplace.json` 中 `"source": "./.claude"` 决定了：

**开发模式（directory marketplace）：**
- `CLAUDE_PLUGIN_ROOT` = `/home/yale/work/meta-cc/.claude/`
- `.mcp.json` → `.claude/.mcp.json` ✅ **已存在，正确**
- `commands/` → `.claude/commands/` ✅ **已存在，正确**
- `bin/meta-cc-mcp` → `.claude/bin/meta-cc-mcp` ❌ **缺失，这是唯一问题**

### 1.4 Bundle/安装后的结构转换

`bundle-release` 在打包时执行关键变换（Makefile 第 372 行）：

```bash
jq '.plugins[0].source = "."' marketplace.json > marketplace.json.tmp
```

将 `"source"` 从 `"./.claude"` 改为 `"."`，使 bundle 根目录成为 plugin root：

```
meta-cc-vX.Y.Z-linux-amd64/         ← bundle root = plugin root（安装后）
  bin/meta-cc-mcp                   ← 二进制（从 cross-compile 拷入）
  commands/                         ← 命令（从 .claude/commands/ 同步）
  .claude-plugin/
    plugin.json                     ← plugin manifest
    marketplace.json                ← source 已被改为 "."
  .mcp.json                         ← 从 .claude/.mcp.json 拷入
  install.sh
```

**安装后（user scope）：**
- `CLAUDE_PLUGIN_ROOT` = `~/.claude/plugins/cache/meta-cc/`（官方文档确认使用 `cache/` 子目录）

### 1.5 当前 Local MCP 能运行的原因

`.claude/.mcp.json` 被 Claude Code 作为项目级 MCP 配置读取，其中 `${CLAUDE_PLUGIN_ROOT}` 由 plugin 系统在环境中设置。实际连接成功可能依赖于此前手工安装的 `~/.local/bin/meta-cc-mcp`（PATH 回退或变量解析副作用）。这属于历史状态残留，不应依赖。

### 1.6 根本原因总结

**plugin 系统失败的唯一直接原因**：`${CLAUDE_PLUGIN_ROOT}/bin/meta-cc-mcp` 即 `.claude/bin/meta-cc-mcp` 不存在。

`.mcp.json` 位置**已经正确**，无需移动。

---

## 2. 标准安装方式调研

### 2.1 Claude Code Plugin 典型安装命令

```bash
claude plugin install meta-cc          # user scope（默认，跨项目可用）
claude plugin install meta-cc --scope project  # project scope（团队共享）
```

### 2.2 Scope 对比

| Scope | CLAUDE_PLUGIN_ROOT | 适用场景 |
|-------|---------------------|----------|
| user（默认） | `~/.claude/plugins/cache/<name>/` | 通用工具（推荐） |
| project | `.claude/plugins/<name>/` | 团队项目专用 |
| local | `.claude/plugins/local/<name>/` | 个人临时 |

> **注意**：user scope 安装到 `~/.claude/plugins/cache/` 子目录（官方文档明确），不是 `plugins/<name>/` 直接路径。

### 2.3 结论

meta-cc 是通用会话分析工具，**user scope 是正确默认值**。

---

## 3. 方案选项

当前架构存在「开发模式 plugin root = `.claude/`」vs「bundle plugin root = 项目根」的两层结构。两种修复路径：

### 选项 A：最小修复（维持现有两层架构）

**变更内容**：仅解决二进制路径问题。

| 变更 | 详情 |
|------|------|
| `make build` | 输出到 `.claude/bin/meta-cc-mcp`（与 plugin root 对齐） |
| `.gitignore` | 添加 `.claude/bin/` |
| E2E 测试 | 默认路径更新为 `.claude/bin/meta-cc-mcp` |

**优点**：改动最小，当前架构逻辑自洽。
**缺点**：`.claude/` 同时承担 Claude Code 配置目录和 plugin root 两个职责，对新贡献者不直观；`make build` 输出到 `.claude/bin/` 违背 Go 项目惯例。

### 选项 B：结构对齐（推荐）

**目标**：开发模式 plugin root 与 bundle plugin root 统一为项目根，消除两层结构。

**变更内容**：

```
meta-cc/                              ← plugin root（开发 + bundle 统一）
  bin/meta-cc-mcp                    ← make build 输出（gitignored）
  .mcp.json                          ← 从 .claude/.mcp.json 移来
  commands/                          ← 命令文件（gitignored，由 sync 生成 or 直接存放）
  .claude-plugin/
    marketplace.json                  ← source 改为 "."（与 bundle 一致）
    plugin.json                       ← 从 .claude/.claude-plugin/ 移来
  .claude/
    commands/                         ← 源文件保留（sync target），或直接删除
    settings.local.json               ← 保留（非 plugin 文件）
    （删除 .mcp.json）
    （删除 .claude-plugin/plugin.json）
```

**关键变更**：

| # | 变更 | 影响 |
|---|------|------|
| 1 | `marketplace.json` `source` 改为 `"."` | plugin root = 项目根，开发与 bundle 一致 |
| 2 | `plugin.json` 移到 `.claude-plugin/plugin.json` | 标准 plugin 布局 |
| 3 | `make build` 输出到 `bin/meta-cc-mcp` | Go 标准惯例，`.gitignore` 已有 `/bin/` |
| 4 | `.mcp.json` 从 `.claude/` 移到项目根 | 与 plugin root 对齐 |
| 5 | `bundle-release` 中 `jq` source 改写可移除 | 开发与 bundle 结构一致，无需变换 |
| 6 | `commands/` 的 source of truth 明确 | 直接在项目根维护，或由 `.claude/commands/` sync |

**优点**：开发与生产一致，结构清晰，符合标准 plugin layout。
**缺点**：变更范围较大，涉及 CI 脚本（`test-plugin-json.sh`）、Makefile（`bundle-release`、`sync-plugin-files`）和目录结构调整。

---

## 4. 连锁影响矩阵

无论选哪个选项，都需要同步更新以下位置：

### 选项 A 影响范围

| 文件 | 当前 | 改为 |
|------|------|------|
| `Makefile` build target | `-o $(MCP_BINARY_NAME)` | `-o .claude/bin/$(MCP_BINARY_NAME)` |
| `Makefile` clean target | `rm -f $(MCP_BINARY_NAME)` | `rm -f .claude/bin/$(MCP_BINARY_NAME)` |
| `Makefile` test-e2e-mcp | `./$(MCP_BINARY_NAME)` | `./.claude/bin/$(MCP_BINARY_NAME)` |
| `tests/e2e/mcp-e2e-simple.sh:10` | `BINARY="${1:-./meta-cc-mcp}"` | `BINARY="${1:-./.claude/bin/meta-cc-mcp}"` |
| `tests/e2e/mcp-e2e-test.sh:10` | `BINARY="${1:-./meta-cc-mcp}"` | `BINARY="${1:-./.claude/bin/meta-cc-mcp}"` |
| `tests/e2e/capability-type-eue.sh:8` | `BINARY="${1:-./meta-cc-mcp}"` | `BINARY="${1:-./.claude/bin/meta-cc-mcp}"` |
| `tests/e2e/capability-type-simple.sh:7` | `BINARY="${1:-./meta-cc-mcp}"` | `BINARY="${1:-./.claude/bin/meta-cc-mcp}"` |
| `.gitignore` | （无） | 添加 `.claude/bin/` |

### 选项 B 影响范围（在 A 基础上增加）

| 文件 | 当前 | 改为 |
|------|------|------|
| `.claude-plugin/marketplace.json` | `"source": "./.claude"` | `"source": "."` |
| `Makefile` build target | 同 A | `-o bin/$(MCP_BINARY_NAME)` |
| `Makefile` test-e2e-mcp | 同 A | `./bin/$(MCP_BINARY_NAME)` |
| `Makefile bundle-release:372` | `jq source = "."` 改写 | **可删除**（结构已一致） |
| `Makefile bundle-release:371` | `cp .claude/.mcp.json` | `cp .mcp.json` |
| `Makefile bundle-release:370` | `cp .claude/.claude-plugin/plugin.json` | `cp .claude-plugin/plugin.json` |
| `scripts/ci/test-plugin-json.sh:19` | `PLUGIN_JSON=".claude/.claude-plugin/plugin.json"` | `PLUGIN_JSON=".claude-plugin/plugin.json"` |
| `scripts/ci/test-plugin-json.sh:92` | `MCP_JSON=".claude/.mcp.json"` | `MCP_JSON=".mcp.json"` |
| E2E 测试默认路径 | 同 A | `./bin/meta-cc-mcp` |
| `.gitignore` | `/bin/` 已存在 | 无需修改 |

### 已知独立问题（与本方案无关，需单独修复）

`install.sh` 第 87 行要求 `agents/` 目录存在，但 `bundle-release` 不打包 `agents/`，导致从 bundle 执行 `install.sh` 必然失败。这是现有 bug，应单独修复。

---

## 5. 推荐实施路径

### 近期（选项 B，分两步）

**Step 1（最小可验证）**：仅修复 binary 路径，验证 plugin 系统恢复正常

按选项 A 修改，重启 Claude Code，确认 `plugin:meta-cc:meta-cc · ✔ connected`。

**Step 2（结构对齐）**：消除两层结构

完成选项 B 的剩余变更，使开发模式与 bundle 模式结构一致。

---

## 6. 安装工作流（目标状态）

### 6.1 端用户安装（user scope）

```bash
claude plugin install meta-cc          # 从 marketplace（未来）
claude plugin install ./bundle-dir --scope user  # 从本地 bundle
```

安装后：`CLAUDE_PLUGIN_ROOT` = `~/.claude/plugins/cache/meta-cc/`

### 6.2 开发者工作流（选项 B 实施后）

```bash
make build     # 构建 bin/meta-cc-mcp
# 重启 Claude Code，plugin:meta-cc:meta-cc 显示 ✔ connected
```

`settings.local.json` 中的 `extraKnownMarketplaces + enabledPlugins` 配置保持不变。

### 6.3 project scope（团队场景）

```bash
claude plugin install meta-cc --scope project
```

---

## 7. 验收标准

**Step 1 完成后：**
- [ ] `/mcp` 面板 `plugin:meta-cc:meta-cc` 显示 `✔ connected`
- [ ] MCP 工具调用正常（如 `get_session_stats`）
- [ ] `make test-e2e-mcp` 通过

**Step 2 完成后（选项 B）：**
- [ ] `/mcp` 面板无重复 `meta-cc` 条目
- [ ] `make commit` 全部检查通过（含 `test-plugin-json.sh`）
- [ ] `make bundle-release VERSION=vX.Y.Z` 生成包结构正确，无需 `jq source` 变换
- [ ] 从 bundle 安装后 `claude plugin install` 正常加载

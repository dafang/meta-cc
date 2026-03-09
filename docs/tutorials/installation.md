# Installation Guide

## Method 1: Plugin Marketplace (Recommended)

Install meta-cc directly from within Claude Code:

```bash
/plugin marketplace add yaleh/meta-cc
/plugin install meta-cc
```

Then restart Claude Code. The plugin system handles everything:
- Installs slash commands (`/prompt-find`, `/prompt-list`, `/prompt-show`)
- Configures the MCP server automatically via `.mcp.json` (no manual `claude mcp add` needed)

## Method 2: Archive Install

Download a platform-specific release archive and run the included installer.

### Linux (x86_64)
```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-linux-amd64.tar.gz | tar xz
cd meta-cc-plugin-linux-amd64
./install.sh
```

### Linux (ARM64)
```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-linux-arm64.tar.gz | tar xz
cd meta-cc-plugin-linux-arm64
./install.sh
```

### macOS (Intel)
```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-darwin-amd64.tar.gz | tar xz
cd meta-cc-plugin-darwin-amd64
./install.sh
```

### macOS (Apple Silicon)
```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-darwin-arm64.tar.gz | tar xz
cd meta-cc-plugin-darwin-arm64
./install.sh
```

### Windows (x86_64)

**Using Git Bash (Recommended):**
```bash
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-windows-amd64.tar.gz | tar xz
cd meta-cc-plugin-windows-amd64
./install.sh
```

**Manual Download:**
1. Download `meta-cc-plugin-windows-amd64.tar.gz` from [GitHub Releases](https://github.com/yaleh/meta-cc/releases/latest)
2. Extract the archive using 7-Zip or similar tool
3. Open Git Bash in the extracted directory
4. Run `./install.sh`

The archive installer:
- Copies the `meta-cc-mcp` binary to `~/.local/bin/`
- Copies slash commands to `~/.claude/commands/`
- Automatically merges MCP server configuration into `~/.claude/mcp.json`

## Manual Installation

If the automated installer fails, follow these steps:

### 1. Download Archive

```bash
# Download plugin package for your platform
curl -L https://github.com/yaleh/meta-cc/releases/latest/download/meta-cc-plugin-<platform>.tar.gz | tar xz
cd meta-cc-plugin-<platform>
```

### 2. Install Binary

**Linux/macOS:**
```bash
mkdir -p ~/.local/bin
cp bin/meta-cc-mcp ~/.local/bin/meta-cc-mcp
chmod +x ~/.local/bin/meta-cc-mcp
```

**Windows:**
```bash
mkdir -p ~/.local/bin
cp bin/meta-cc-mcp.exe ~/.local/bin/meta-cc-mcp.exe
```

### 3. Install Claude Code Files

The archive uses a flat layout with `commands/` at the top level:

```bash
mkdir -p ~/.claude/commands

# Copy slash commands
cp commands/* ~/.claude/commands/
```

### 4. Configure MCP

The archive includes a `.mcp.json` file. If you have `jq` installed, merge it automatically:

```bash
jq -s '.[0] * .[1]' ~/.claude/mcp.json .mcp.json > /tmp/mcp-merged.json && mv /tmp/mcp-merged.json ~/.claude/mcp.json
```

Otherwise, manually add to `~/.claude/mcp.json`:

```json
{
  "mcpServers": {
    "meta-cc": {
      "command": "meta-cc-mcp",
      "args": []
    }
  }
}
```

If you already have other MCP servers configured, add the `"meta-cc"` entry to the existing `"mcpServers"` object.

## Verification

After installation, verify the setup:

```bash
# Check binary location
which meta-cc-mcp

# Verify binary is executable
ls -l ~/.local/bin/meta-cc-mcp
```

**In Claude Code:**

1. **Test MCP Tools**: In conversation, ask "What are my recent tool usage patterns?"
2. **Test Slash Commands**: Type `/prompt-list` and press Enter

## Troubleshooting

### Binary not found

**Issue**: `meta-cc-mcp: command not found`

**Solution**: Add `~/.local/bin` to PATH:

```bash
# For bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# For zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# For fish
fish_add_path ~/.local/bin
```

**Windows (Git Bash)**:
```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bash_profile
source ~/.bash_profile
```

### MCP server not connecting

**Issue**: MCP server fails to start or times out

**Solutions**:

1. **Check MCP logs** in Claude Code settings (Settings → MCP)
2. **Verify binary is executable**:
   ```bash
   ls -l ~/.local/bin/meta-cc-mcp
   chmod +x ~/.local/bin/meta-cc-mcp
   ```
3. **Test MCP server manually**:
   ```bash
   meta-cc-mcp
   # Should start and wait for JSON-RPC messages
   # Press Ctrl+C to exit
   ```
4. **Check MCP configuration**:
   ```bash
   cat ~/.claude/mcp.json
   # Verify meta-cc entry exists and is valid JSON
   ```

### Slash commands not working

**Issue**: Slash commands not recognized in Claude Code

**Solutions**:

1. **Restart Claude Code** after installation
2. **Verify command files exist**:
   ```bash
   ls ~/.claude/commands/prompt-*.md
   ```
3. **Check command permissions**:
   ```bash
   chmod +r ~/.claude/commands/prompt-find.md
   ```
4. **Check Claude Code settings** to ensure slash commands are enabled

### Installation fails on macOS

**Issue**: macOS blocks execution due to Gatekeeper

**Solutions**:

1. **Allow unsigned binary**:
   ```bash
   xattr -d com.apple.quarantine ~/.local/bin/meta-cc-mcp
   ```
2. **Or use System Settings**:
   - Go to System Settings → Privacy & Security
   - Allow the binary to run

### Permission denied errors

**Issue**: Permission errors during installation

**Solutions**:

1. **Ensure write permissions**:
   ```bash
   mkdir -p ~/.local/bin ~/.claude/commands
   chmod u+w ~/.local/bin ~/.claude
   ```
2. **Check disk space**:
   ```bash
   df -h ~
   ```
3. **Run without sudo** (installation should not require root)

### Windows-specific issues

**Issue**: Installation fails on Windows

**Solutions**:

1. **Use Git Bash** (not PowerShell or CMD)
2. **Check PATH in Git Bash**:
   ```bash
   echo $PATH | tr ':' '\n' | grep local
   ```
3. **Verify .exe extensions**:
   ```bash
   ls -l ~/.local/bin/meta-cc-mcp.exe
   ```

## Uninstallation

To remove meta-cc:

### Using uninstall script

```bash
cd meta-cc-plugin-<platform>
./uninstall.sh
```

The uninstall script removes the binary, all slash commands, and automatically removes the `meta-cc` entry from `~/.claude/mcp.json`.

### Manual uninstallation

```bash
# Remove binary
rm ~/.local/bin/meta-cc-mcp

# Remove Claude Code files
rm ~/.claude/commands/prompt-find.md
rm ~/.claude/commands/prompt-list.md
rm ~/.claude/commands/prompt-show.md

# Remove meta-cc from MCP configuration
jq 'del(.mcpServers["meta-cc"])' ~/.claude/mcp.json > /tmp/mcp.json && mv /tmp/mcp.json ~/.claude/mcp.json
```

## Upgrading

To upgrade to a newer version:

1. **Download new version** using the Quick Install commands above
2. **Run install.sh** - it will overwrite existing binaries
3. **Restart Claude Code** to load the new version

The installer preserves your MCP configuration and existing settings.

## Platform-Specific Notes

### Linux

- **Distributions**: Tested on Ubuntu 22.04+, Debian 11+, Fedora 38+
- **Dependencies**: None (statically compiled binaries)
- **systemd**: Not required (MCP server runs on-demand)

### macOS

- **Versions**: Tested on macOS 12 (Monterey) and later
- **Gatekeeper**: See "Installation fails on macOS" troubleshooting
- **Homebrew**: Not required (standalone binaries)

### Windows

- **Requirements**: Git Bash (part of Git for Windows)
- **PowerShell**: Not supported (use Git Bash)
- **WSL**: Not required (native Windows binaries)

## Getting Help

If you encounter issues not covered in this guide:

1. **Check existing issues**: [GitHub Issues](https://github.com/yaleh/meta-cc/issues)
2. **Create new issue**: Include:
   - Operating system and version
   - Installation method used
   - Complete error messages
   - Output of `meta-cc-mcp --version` (if binary runs)
3. **Community support**: See [Discussions](https://github.com/yaleh/meta-cc/discussions)

## Next Steps

After successful installation:

1. **Read the documentation**: [Getting Started](../../README.md)
2. **Browse prompts**: `/prompt-list` to see saved prompts, `/prompt-find <keywords>` to search
3. **Learn MCP tools**: See [MCP Guide](../guides/mcp.md)
4. **Ask naturally**: "Show me my recent tool errors" or "What are my work patterns?"

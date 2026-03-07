# Claude Marketplace Plugin Method

This guide documents the plugin architecture and method used to distribute Trindex as a Claude Marketplace plugin. This pattern can be reused for other MCP-based tools.

## Overview

The marketplace plugin method consists of four key files that work together to provide:
- Plugin metadata and versioning
- Automatic binary installation (OS/arch detection)
- MCP server configuration
- Environment variable support

```
.claude-plugin/
├── marketplace.json          # Marketplace listing & metadata
└── plugin/
    ├── plugin.json           # Plugin information
    ├── .mcp.json             # MCP server configuration
    └── postInstall.sh        # Auto-installation script
```

## File Specifications

### 1. `marketplace.json` - Plugin Registry Entry

The marketplace listing that appears in Claude's plugin browser.

```json
{
  "name": "dbehnke",
  "owner": {
    "name": "David Behnke"
  },
  "metadata": {
    "description": "Trindex — persistent semantic memory for AI agents",
    "homepage": "https://github.com/dbehnke/trindex"
  },
  "plugins": [
    {
      "name": "trindex",
      "version": "1.0.7",
      "source": "./.claude-plugin/plugin",
      "description": "Persistent semantic memory via MCP — hybrid vector + full-text search across sessions"
    }
  ]
}
```

**Key fields:**
- `owner.name` - Your name/organization
- `metadata.homepage` - GitHub repo or project homepage
- `plugins[].name` - Plugin identifier (lowercase, no spaces)
- `plugins[].version` - Semantic version (must match `plugin.json` and `.mcp.json`)
- `plugins[].source` - Relative path to plugin directory
- `plugins[].description` - Short description (~80 chars)

### 2. `plugin.json` - Plugin Metadata

Contains plugin information (name, version, author, keywords, etc.).

```json
{
  "name": "trindex",
  "version": "1.0.7",
  "description": "Persistent semantic memory for AI agents via MCP. Stores and recalls memories using hybrid vector + full-text search (pgvector + Postgres). Proxies to a running trindex server.",
  "author": {
    "name": "Dave Behnke"
  },
  "repository": "https://github.com/dbehnke/trindex",
  "license": "MIT",
  "keywords": ["memory", "semantic", "mcp", "persistence", "pgvector"]
}
```

**Key fields:**
- `version` - Must match marketplace.json version
- `keywords` - Helps users discover your plugin
- `repository` - Link to source code
- `license` - Choose appropriate license (MIT, BSD, BSL, etc.)

### 3. `.mcp.json` - MCP Server Configuration

Configures how Claude launches the MCP server. Uses environment variable substitution via `${VAR_NAME}` syntax.

```json
{
  "mcpServers": {
    "trindex": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/trindex",
      "args": ["mcp"],
      "env": {
        "TRINDEX_URL": "${TRINDEX_URL:-http://localhost:9636}",
        "TRINDEX_API_KEY": "${TRINDEX_API_KEY:-}"
      }
    }
  }
}
```

**Special variables:**
- `${CLAUDE_PLUGIN_ROOT}` - Plugin installation directory (auto-set by Claude)
- `${VARIABLE_NAME:-default}` - Use env var or fallback to default
- `${VARIABLE_NAME}` - Required; Claude will prompt user if not set

**Command considerations:**
- Path must be executable binary or script
- `args` - Arguments passed to the command
- `env` - Environment variables available to the process

### 4. `postInstall.sh` - Auto-Installation Script

Runs automatically after plugin is installed. Downloads and installs the binary.

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="dbehnke/trindex"
INSTALL_DIR="${CLAUDE_PLUGIN_ROOT}/bin"
BINARY="${INSTALL_DIR}/trindex"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

# Get latest release version from GitHub API
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
TARBALL="trindex_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

echo "Installing trindex ${VERSION} (${OS}/${ARCH})..."
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" | tar -xz -C "$INSTALL_DIR" trindex
chmod +x "$BINARY"
echo "trindex installed to ${BINARY}"
```

**Key features:**
- **OS/Arch detection** - Supports darwin (macOS), linux
- **Architecture mapping** - Handles x86_64→amd64, arm64/aarch64
- **GitHub API** - Fetches latest release version automatically
- **Fallback to release** - No version hardcoding; always latest
- **Direct download** - Minimal dependencies (only curl, tar)

## Release and Versioning

Version sync is critical. All three files must use the same version:

```
marketplace.json    → plugins[].version
plugin.json         → version
.mcp.json           → (implicit in released binary)
```

### Automated Version Updates

Use the provided script to update all version fields:

```bash
./scripts/update-marketplace-version.sh v1.0.8
```

This script:
- Accepts version with or without `v` prefix
- Updates `marketplace.json` and `plugin.json`
- Creates backups (`.bak`) then removes them

### Release Workflow

1. **Create GitHub release** with tag `v1.0.8`
2. **Build binaries** for each platform:
   ```
   trindex_darwin_arm64.tar.gz
   trindex_darwin_amd64.tar.gz
   trindex_linux_arm64.tar.gz
   trindex_linux_amd64.tar.gz
   ```
3. **Upload to release** - Attach binaries to GitHub release
4. **Update version files**:
   ```bash
   ./scripts/update-marketplace-version.sh v1.0.8
   ```
5. **Commit and push** - Version bump commit
6. **Submit to marketplace** - Claude marketplace auto-detects new version from repo

## Requirements for Binary Releases

The `postInstall.sh` script expects:

1. **GitHub releases** with semantic version tags (v1.0.0, v1.0.1, etc.)
2. **Tarball naming convention**: `{binary-name}_{os}_{arch}.tar.gz`
   - OS: `darwin` or `linux`
   - Arch: `amd64` or `arm64`
3. **Binary inside tarball** - Single file at root: `trindex`
4. **Public repository** - Accessible without authentication

### Example: Using GoReleaser for automatic builds

Trindex uses GoReleaser to automate multi-platform builds (`.goreleaser.yml`):

```yaml
builds:
  - main: ./cmd/trindex
    binary: trindex
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64

archives:
  - name_template: "trindex_{{ .Os }}_{{ .Arch }}"
```

This automatically creates:
- `trindex_linux_amd64.tar.gz`
- `trindex_linux_arm64.tar.gz`
- `trindex_darwin_amd64.tar.gz`
- `trindex_darwin_arm64.tar.gz`

**Release workflow** (`.github/workflows/release.yml`):
1. Tag commit with `v1.0.8`
2. GitHub Actions triggers GoReleaser
3. Binaries are built and uploaded to GitHub releases
4. `postInstall.sh` auto-detects and downloads the correct binary

## Environment Variables

Users can configure the plugin via environment variables. These are prompted by Claude or set in `~/.claude/env.toml`.

**Common patterns:**

```bash
# Direct endpoint
TRINDEX_URL=https://brain.example.com

# With API key
TRINDEX_API_KEY=sk-abc123...

# With authentication
TRINDEX_URL=https://brain.example.com
TRINDEX_API_KEY=sk-abc123...
```

The `.mcp.json` uses the **bash parameter expansion** syntax:

```json
"env": {
  "TRINDEX_URL": "${TRINDEX_URL:-http://localhost:9636}",
  "TRINDEX_API_KEY": "${TRINDEX_API_KEY:-}"
}
```

- `${VAR:-default}` - Use VAR if set, else use default
- `${VAR}` - Required (Claude will prompt if not set)

## Testing Locally

### 1. Install via Claude Code CLI

```bash
claude mcp add trindex --command "/path/to/plugin/.claude-plugin/plugin/.mcp.json"
```

### 2. Or install from directory

```bash
claude mcp add trindex --command "$(pwd)/.claude-plugin/plugin/.mcp.json"
```

### 3. Set environment variables

```bash
export TRINDEX_URL=http://localhost:9636
export TRINDEX_API_KEY=your-key
```

### 4. Test MCP connection

```bash
./trindex doctor
```

## Advantages of This Pattern

1. **User-friendly installation** - One-click from marketplace, no manual downloads
2. **Always latest** - postInstall.sh fetches latest release automatically
3. **Cross-platform** - Supports macOS (Intel & ARM) and Linux (Intel & ARM)
4. **Configurable** - Environment variables allow user customization
5. **Version managed** - Single source of truth in marketplace.json
6. **No dependencies** - Only requires curl and tar (both standard)
7. **Reusable** - Can apply this pattern to any Go binary tool

## Adapting for Your Tool

To use this pattern for another project:

1. **Create directory structure** - Copy `.claude-plugin/` folder
2. **Update marketplace.json** - Change owner, plugin name, description
3. **Update plugin.json** - Update name, version, author, keywords
4. **Update .mcp.json**:
   - Change `trindex` keys to your tool name
   - Update command path and args
   - Add/remove environment variables as needed
5. **Update postInstall.sh**:
   - Change REPO to your GitHub path
   - Change BINARY name
   - Change TARBALL name pattern
6. **Build releases** - Create GitHub releases with correctly named tarballs
7. **Test locally** - Use Claude Code CLI to test before submitting to marketplace

## Troubleshooting

**Plugin won't install:**
- Check GitHub releases exist with correct naming
- Verify binary is executable (`chmod +x`)
- Ensure tarball contains binary at root level

**MCP connection fails:**
- Run `./trindex doctor` to verify configuration
- Check `TRINDEX_URL` is correct
- Verify API server is running on specified port

**Wrong architecture installed:**
- Check `uname -m` output
- Verify tarball naming matches detection logic
- Test postInstall.sh directly: `bash postInstall.sh`

**Version mismatch errors:**
- Run `./scripts/update-marketplace-version.sh` with correct version
- Verify all three files were updated
- Check git diff to confirm changes

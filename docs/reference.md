# Reference

Environment settings, options, troubleshooting, and other supplementary information.

[← Back to README](../README.md)

---

## Two Environments

| Environment | Purpose | When to Use |
|-------------|---------|-------------|
| **DevContainer** (`.devcontainer/`) | Primary development with VS Code | Daily development |
| **CLI Sandbox** (`cli_sandbox/`) | Alternative/recovery | When DevContainer is broken |

**Why two environments?**

The CLI Sandbox serves as a **recovery alternative**.

If Dev Container configuration breaks:
1. VS Code can't start the Dev Container
2. Claude Code doesn't work either
3. Can't get AI help to fix the configuration → **stuck**

With `cli_sandbox/`:
1. Even if Dev Container is broken
2. You can start AI from the host
   - `./cli_sandbox/claude.sh` (Claude Code)
   - `./cli_sandbox/gemini.sh` (Gemini CLI)
3. Have AI fix the Dev Container configuration

```bash
./cli_sandbox/claude.sh   # or
./cli_sandbox/gemini.sh
# Have AI fix the broken DevContainer configuration
```

---

## Project Name Customization

By default, the DevContainer project name is `<parent-directory-name>_devcontainer` (e.g., `workspace_devcontainer`).

To set a custom project name, create a `.devcontainer/.env` file:

```bash
# Copy .env.example
cp .devcontainer/.env.example .devcontainer/.env
```

`.env` file contents:
```bash
COMPOSE_PROJECT_NAME=ai-sandbox
```

This makes container and volume names more readable:
- Container: `ai-sandbox-ai-sandbox-1`
- Volume: `ai-sandbox_node-home`

> **Note:** The `.env` file is in `.gitignore`, so each developer can have their own settings.

---

## Startup Validation

Both AI Sandbox environments (DevContainer and CLI Sandbox) automatically run the following checks on every startup:

| Check | What It Does |
|-------|-------------|
| AI settings merge | Automatically combines `.claude/settings.json` from subprojects |
| Configuration consistency | Verifies secret hiding settings match between DevContainer and CLI Sandbox |
| Secret hiding verification | Confirms `.env` and `secrets/` are actually hidden from AI |
| Sync check | Ensures files blocked in AI settings are also hidden in docker-compose |
| Template updates | Notifies if a newer template version is available |

If any issues are found, warnings are displayed and you can review them before continuing. <ins>This ensures you never work with a misconfigured environment without knowing.</ins>

### Output Options

You can control the output verbosity:

| Mode | Flag | Output |
|------|------|--------|
| Quiet | `--quiet` or `-q` | Warnings and errors only (minimal) |
| Summary | `--summary` or `-s` | Concise summary |
| Verbose | (none, default) | Detailed output with decorations |

**CLI Sandbox examples:**
```bash
# Minimal output (warnings only)
./cli_sandbox/ai_sandbox.sh --quiet

# Concise summary
./cli_sandbox/ai_sandbox.sh --summary
```

**Environment variable:**
```bash
# Set default verbosity
export STARTUP_VERBOSITY=quiet  # or: summary, verbose
```

**Configuration file:** `.sandbox/config/startup.conf`
```bash
# Default verbosity for all startup scripts
STARTUP_VERBOSITY="verbose"

# URLs used in "see README for details" messages
README_URL="README.md"
README_URL_JA="README.ja.md"  # Used when LANG=ja_JP*

# Backup retention count per label (0 = unlimited)
BACKUP_KEEP_COUNT=0
```

Backups created by sync scripts are stored in `.sandbox/backups/`. To limit retention:

```bash
# Keep only the last 10
BACKUP_KEEP_COUNT=10

# Temporarily override via environment variable
BACKUP_KEEP_COUNT=10 .sandbox/scripts/sync-secrets.sh
```

---

## Excluding Files from Sync Warnings

Startup scripts check that files blocked in `.claude/settings.json` are also hidden in `docker-compose.yml`. To exclude specific patterns (like `.example` files) from warnings, edit `.sandbox/config/sync-ignore`:

```gitignore
# Exclude example/template files from sync warnings
**/*.example
**/*.sample
**/*.template
```

This uses gitignore-style patterns. Files matching these patterns won't trigger "not configured in docker-compose.yml" warnings.

---

## Running Multiple DevContainers

If you need completely isolated DevContainer environments (e.g., different client projects), use `COMPOSE_PROJECT_NAME` to create separate instances.

<details>
<summary>Methods and home directory sharing</summary>

### Method A: Separate via .env file (Recommended)

Set different project names in `.devcontainer/.env`:

```bash
COMPOSE_PROJECT_NAME=client-a
```

In another workspace:

```bash
COMPOSE_PROJECT_NAME=client-b
```

### Method B: Separate via command line

Start DevContainers with different project names:

```bash
# Project A
COMPOSE_PROJECT_NAME=client-a docker-compose up -d

# Project B (creates separate volumes)
COMPOSE_PROJECT_NAME=client-b docker-compose up -d
```

> ⚠️ **Note:** Different project names create separate volumes, so the home directory (credentials, settings, history) won't be shared automatically. See "Home directory copy" below.

### Method C: Share home directory via bind mount

To automatically share the home directory across all instances, change `docker-compose.yml` to use bind mounts:

```yaml
volumes:
  # Bind mounts instead of named volumes
  - ~/.ai-sandbox/home:/home/node
  - ~/.ai-sandbox/gcloud:/home/node/.config/gcloud
```

**Pros:**
- Auto-shared home directory across all instances
- Easy backup (just copy the host directory)

**Cons:**
- Depends on host directory structure
- May need UID/GID adjustments on Linux hosts

### Home Directory Export/Import

You can backup or migrate the home directory (credentials, settings, history):

```bash
# Export entire workspace (both devcontainer and cli_sandbox)
./.sandbox/host-tools/copy-credentials.sh --export /path/to/workspace ~/backup

# Export from specific docker-compose.yml
./.sandbox/host-tools/copy-credentials.sh --export .devcontainer/docker-compose.yml ~/backup

# Import to workspace
./.sandbox/host-tools/copy-credentials.sh --import ~/backup /path/to/workspace
```

**Note:** If target volumes don't exist, start the environment once first to create them.

Use cases:
- Check `~/.claude/` usage data
- Backup settings
- Migrate credentials to a new workspace
- Troubleshooting

</details>

---

## Uninstalling DockMCP

If DockMCP is no longer needed, delete the binary from its install location:

```bash
rm ~/go/bin/dkmcp
# or
rm /usr/local/bin/dkmcp
```

---

## Troubleshooting

### DockMCP Connection

If Claude Code doesn't recognize DockMCP tools:

1. **Check VS Code ports panel** - Stop if DockMCP's port (default 8080) is being forwarded
2. **Verify DockMCP is running** - `curl http://localhost:8080/health` (on host OS)
3. **Try MCP reconnect** - In Claude Code, run `/mcp` and select "Reconnect"
4. **Fully restart VS Code** (Cmd+Q / Alt+F4) - If Reconnect doesn't help

### Fallback: Using dkmcp client in AI Sandbox

If the MCP protocol isn't working (Claude Code or Gemini can't connect), you can use `dkmcp client` commands directly in the AI Sandbox as a fallback.

> **Note:** Even when `/mcp` shows "✔ connected", MCP tools may fail with "Client not initialized" error. This may be caused by session management timing issues in VS Code extensions (Claude Code, Gemini Code Assist, etc.). In this case:
> 1. First try `/mcp` → "Reconnect" (quickest solution)
> 2. If that doesn't work, AI uses `dkmcp client` commands as fallback
> 3. As a last resort, fully restart VS Code to re-establish the connection

**Setup (first time only):**

Install dkmcp inside AI Sandbox:
```bash
cd /workspace/dkmcp
make install
```

> **Note:** Go environment is enabled by default. After installation, if you want to reduce image size, comment out the `features` block in `.devcontainer/devcontainer.json` and rebuild.

**Usage:**
```bash
# List containers
dkmcp client list

# Get logs
dkmcp client logs securenote-api

# Execute command
dkmcp client exec securenote-api "npm test"
```

> **About `--url`:** Defaults to `http://host.docker.internal:8080`. If you changed the server port in `dkmcp.yaml`, specify it explicitly via the `--url` flag or `DOCKMCP_SERVER_URL` environment variable.
> ```bash
> dkmcp client list --url http://host.docker.internal:9090
> # or
> export DOCKMCP_SERVER_URL=http://host.docker.internal:9090
> ```

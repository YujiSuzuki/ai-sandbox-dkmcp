# Customization Guide

There are three ways to get started with this project. Choose the one that fits your situation.

[← Back to README](../README.md)

---

## How to Get This Project

| Method | GitHub account | How to update |
|---|---|---|
| **Use this template** | Required | See [Updating Guide](updating.md) |
| **git clone** | Not required | `git pull origin main` |
| **ZIP download** | Not required | Download new ZIP, apply changes manually |

### Option 1: Use as a GitHub Template (recommended)

On GitHub, click **"Use this template"** → **"Create a new repository"**.

The created repository will have:
- All template files (without this repo's commit history)
- A fresh Git history
- Independent from upstream (no automatic sync)

Then clone your new repository:

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

Since template repositories have no automatic upstream connection, an **update notification feature** is included. It checks for new GitHub releases at AI Sandbox startup. See [Updating Guide](updating.md) for how to apply updates.

### Option 2: Direct Clone

If you want to track upstream changes directly with Git (e.g., for contributing):

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

Updates are available via `git pull origin main`.

### Option 3: ZIP Download

If you don't use Git, download the ZIP from GitHub (**"Code"** → **"Download ZIP"**) and extract it.

Note: With this method, you'll need to download a new ZIP and manually apply changes when updating. The built-in update notification will still alert you when new versions are available.

---

## Project Customization

Whether you used the template or cloned directly, follow these steps to customize the environment.

### AI-assisted setup

Since this is an AI Sandbox, you can ask your AI assistant to handle most of the customization. Open the AI Sandbox and describe your project:

> "Customize this template for my project. My projects are:
> - `my-api/` (Node.js API with `.env` and `secrets/` directory)
> - `my-web/` (React frontend, no secrets)
>
> Container names: `my-api`, `my-web`. Allowed commands for my-api: `npm test`, `npm run lint`"

The AI will edit docker-compose.yml, create dkmcp.yaml, update AI configuration files, and run validation scripts. You only need to rebuild the DevContainer and start DockMCP yourself.

The sections below describe each step for manual setup.

### Replace demo-apps with your projects

```bash
# Remove demo apps (or keep them as reference)
rm -rf demo-apps demo-apps-ios

# Add your projects
git clone https://github.com/your-org/your-api.git
git clone https://github.com/your-org/your-web.git
```

### Configure secret hiding

Edit both **`.devcontainer/docker-compose.yml`** and **`cli_sandbox/docker-compose.yml`**:

```yaml
services:
  ai-sandbox:
    volumes:
      # Hide secret files (mount to /dev/null)
      - /dev/null:/workspace/your-api/.env:ro
      - /dev/null:/workspace/your-api/config/secrets.json:ro

    tmpfs:
      # Make secret directories empty
      - /workspace/your-api/secrets:ro
      - /workspace/your-api/keys:ro
```

**Key points:**
- `.env` files → mount to `/dev/null`
- `secrets/` directories → `tmpfs` + `:ro` for empty directories
- Keep both docker-compose.yml files in sync

**Automatic validation:**

These checks run automatically at startup:
1. `validate-secrets.sh` - Verifies secrets are actually hidden (auto-reads paths from docker-compose.yml)
2. `compare-secret-config.sh` - Warns if DevContainer and CLI configurations differ
3. `check-secret-sync.sh` - Warns if files blocked in AI settings are not hidden in docker-compose.yml
   - Supports: `.claude/settings.json`, `.aiexclude`, `.geminiignore`
   - Note: `.gitignore` is intentionally **not supported** — it contains many non-secret patterns (`node_modules/`, `dist/`, `*.log`) that would create noise. List only secrets explicitly in AI-specific files.

**Manual sync tool:** If `check-secret-sync.sh` reports unconfigured files, run `.sandbox/scripts/sync-secrets.sh` to interactively add them. Use option `4` (preview) to check settings without modifying files.

**Recommended first-time setup flow:**
```bash
# 1. Enter container without AI (AI won't auto-start)
./cli_sandbox/ai_sandbox.sh

# 2. Inside container: interactively sync secret settings
.sandbox/scripts/sync-secrets.sh

# 3. Exit and rebuild DevContainer
exit
# Then open DevContainer in VS Code
```

This ensures secret settings are complete before AI accesses any files.

Detection rules:
- `/dev/null:/workspace/...` in volumes → secret file
- `/workspace/...:ro` in tmpfs → secret directory

### DockMCP configuration

Copy and edit **`dkmcp/configs/dkmcp.example.yaml`**:

```bash
cp dkmcp/configs/dkmcp.example.yaml dkmcp.yaml
```

```yaml
security:
  mode: "moderate"

  # Change to your container names
  allowed_containers:
    - "your-api-*"
    - "your-web-*"
    - "your-db-*"

  # Configure allowed commands
  exec_whitelist:
    "your-api":
      - "npm test"
      - "npm run lint"
      - "python manage.py test"
    "your-db":
      - "psql -c 'SELECT 1'"
```

For stricter configuration:

```yaml
security:
  mode: "strict"  # Read-only (logs, inspect, stats)

  allowed_containers:
    - "prod-*"      # Production containers only

  exec_whitelist: {}  # No command execution
```

For multiple instances and more, see [dkmcp/README.md "Server Startup"](../dkmcp/README.md#running-multiple-instances).

### AI assistant configuration

Edit these files so AI assistants correctly understand your project structure and secret policies.

**Automatically applied (no action needed):**

If subprojects already have `.claude/settings.json`, they are auto-merged at AI Sandbox startup (`merge-claude-settings.sh`). No need to create new ones.

**Files that need editing:**

| File | Content | Action |
|------|---------|--------|
| `CLAUDE.md` | Project description for Claude Code | Remove demo-specific content, rewrite for your project |
| `GEMINI.md` | Project description for Gemini Code Assist | Same as above |
| `.aiexclude` | Gemini Code Assist secret patterns | Add your secret paths as needed |
| `.geminiignore` | Gemini CLI secret patterns | Same as above |

**CLAUDE.md / GEMINI.md editing guidelines:**

- **Keep**: DockMCP MCP Tools usage, security architecture overview, environment separation (What Runs Where)
- **Rewrite**: Project structure, Common Tasks examples
- **Remove**: SecureNote demo-specific content, demo scenarios

### Plugins for multi-repo setups

When using Claude Code plugins with multi-repo setups (each project as an independent Git repository), some configuration is needed. See [Plugin Guide](plugins.md) for details.

> **Note**: This section is Claude Code-specific. Not available for Gemini Code Assist.

### Rebuild DevContainer

```bash
# Open Command Palette in VS Code (Cmd/Ctrl + Shift + P)
# Run "Dev Containers: Rebuild Container"
```

### Verify

```bash
# Verify secret files are hidden inside AI Sandbox
cat your-api/.env
# → Empty or "No such file"

# Verify container access via DockMCP
# Ask Claude Code "Show me the container list"
# Ask Claude Code "Show me your-api logs"
```

### Checklist

- [ ] Configure secret files in `.devcontainer/docker-compose.yml`
- [ ] Apply same configuration in `cli_sandbox/docker-compose.yml`
- [ ] Set container names in `dkmcp.yaml`
- [ ] Set allowed commands in `dkmcp.yaml`
- [ ] Edit `CLAUDE.md` / `GEMINI.md` for your project
- [ ] Add secret paths to `.aiexclude` / `.geminiignore` (if needed)
- [ ] Rebuild DevContainer
- [ ] Verify secret files are hidden
- [ ] Verify log access via DockMCP

# Customization Guide

This repository is designed as a **GitHub template repository**. You can create your own project from this template.

[← Back to README](../README.md)

---

## Use as a Template

### Step 1: Create from template

On GitHub, click **"Use this template"** → **"Create a new repository"**.

The created repository will have:
- All template files (without this repo's commit history)
- A fresh Git history
- Independent from upstream (no automatic sync)

### Step 2: Clone your new repository

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

### Check for Updates

Repositories created from the template cannot automatically receive upstream updates, so an **update notification feature** is included. It checks for new GitHub releases at AI Sandbox startup and notifies you if a new version is available.

<details>
<summary>Notification examples and configuration details</summary>

**How it works:**
- By default, checks **all releases including pre-releases** so you can receive bug fixes and improvements quickly
- On first startup, it only records the latest version without showing notifications
- On subsequent checks, if a new version is found, a notification like this appears:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 Update Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Current version:  v0.1.0
  Latest version:   v0.2.0

  How to update:
    1. Check release notes for changes
    2. Manually apply necessary changes

  Release notes:
    https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**How to apply updates:**
1. Check [release notes](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases) for changes
2. Manually apply necessary changes to your project

**Configuration file:** `.sandbox/config/template-source.conf`
```bash
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"            # "all" = including pre-releases, "stable" = stable releases only
CHECK_UPDATES="true"           # "false" to disable
CHECK_INTERVAL_HOURS="24"      # Check interval (0 = every time)
```

| `CHECK_CHANNEL` | Behavior | Use Case |
|---|---|---|
| `"all"` (default) | Checks all releases including pre-releases | Want bug fixes and improvements ASAP |
| `"stable"` | Checks stable releases only | Only want to track stable milestones |

</details>

---

## Alternative: Direct Clone

If you want to track upstream changes with Git (e.g., for contributing):

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

---

## Project Customization

Whether you used the template or cloned directly, follow these steps to customize the environment.

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

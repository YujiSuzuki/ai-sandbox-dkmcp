# Updating Guide

How to apply updates from the original template to your project.

[← Back to README](../README.md)

---

## Automatic Update Notifications

This project checks for new releases on startup. When a new version is available, you'll see a notification:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 Update Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Current version:  v1.1.0
  Latest version:   v1.2.0

  💡 You can ask your AI assistant to help
     Example: "Please update to the latest version"

  Release notes:
    https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Configuration

Update checks are configured in `.sandbox/config/template-source.conf`:

```bash
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_UPDATES="true"           # "false" to disable
CHECK_INTERVAL_HOURS="24"      # Check interval (0 = every time)
CHECK_CHANNEL="all"            # "all" = including pre-releases, "stable" = stable releases only
```

| `CHECK_CHANNEL` | Behavior | Use Case |
|---|---|---|
| `"all"` (default) | Checks all releases including pre-releases | Want bug fixes and improvements ASAP |
| `"stable"` | Checks stable releases only | Only want to track stable milestones |

---

## Quick Update (with AI Assistance)

The easiest way to update is to ask your AI assistant:

```
You: "Please update to the latest version"
```

Your AI assistant will:
1. Check what changed in the new version
2. Detect any conflicts with your customizations
3. Explain the changes and potential impacts
4. Apply the update after your confirmation
5. Rebuild necessary components (SandboxMCP, etc.)

This works for both clone and template users — the AI assistant will determine your setup and choose the right approach.

---

## Manual Update

The procedure depends on how you set up this project.

### If you cloned this repository directly

```bash
# 1. Check what's new
git fetch origin main
git log HEAD..origin/main --oneline

# 2. Pull changes
git pull origin main

# 3. Rebuild (see "After Updating" section below)
```

### If you downloaded the ZIP

1. Download the latest ZIP from [the repository](https://github.com/YujiSuzuki/ai-sandbox-dkmcp) (**"Code"** → **"Download ZIP"**)
2. Compare the new files with your current project and apply relevant changes manually
3. Focus on infrastructure directories: `.sandbox/`, `dkmcp/`, `.devcontainer/`, `cli_sandbox/`

### If you created from the GitHub template

Template repositories have **no automatic upstream connection**. Choose one of these approaches:

#### Option A: Check release notes and apply manually (simplest)

1. Check [release notes](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases) for changes
2. Apply relevant changes to your project manually

Best for: small or infrequent updates, projects that have diverged significantly.

#### Option B: Add upstream remote and merge

Add the original repository as a remote and pull changes:

```bash
# One-time setup: add the template repo as "upstream"
git remote add upstream https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git

# Fetch and merge updates
git fetch upstream main
git merge upstream/main
```

> **Note:** If your project has diverged from the template, you may encounter merge conflicts that need to be resolved manually.

Best for: projects that stay relatively close to the template structure.

#### Option C: Automated sync with GitHub Actions

Use [actions-template-sync](https://github.com/AndreasAugustin/actions-template-sync) to automatically receive updates as pull requests:

```yaml
# .github/workflows/template-sync.yml
name: Template Sync
on:
  schedule:
    - cron: "0 0 * * 0"  # Weekly
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: AndreasAugustin/actions-template-sync@v2
        with:
          source_repo_path: YujiSuzuki/ai-sandbox-dkmcp
          upstream_branch: main
```

This creates a PR whenever the template has new changes, so you can review and merge at your own pace.

Best for: teams that want to stay up-to-date with minimal effort.

---

## About Demo App Changes

If you've already deleted `demo-apps/` and `demo-apps-ios/` (as most users do after customizing), you can safely ignore any demo app changes in updates.

- **Option A (manual):** Simply skip demo-related changes in the release notes
- **Option B (upstream merge):** Conflicts on deleted demo files are expected — resolve by keeping your deletion (`git rm` the conflicting files, then continue the merge)
- **Option C (GitHub Actions):** The generated PR may include demo app changes — just discard those files from the PR

The important parts of updates are the infrastructure files: `.sandbox/`, `dkmcp/`, `.devcontainer/`, and `cli_sandbox/`.

---

## After Updating

Regardless of how you applied the update, you may need to rebuild components.

### Rebuild SandboxMCP (if `.sandbox/sandbox-mcp/` changed)

```bash
# Inside AI Sandbox
cd .sandbox/sandbox-mcp
make clean && make register
```

### Rebuild DockMCP (if `dkmcp/` changed)

```bash
# On Host OS (not in AI Sandbox)
cd dkmcp
make install
```

### Restart VS Code or reconnect MCP

- **Quick**: Run `/mcp` → "Reconnect" in Claude Code
- **Full**: Restart VS Code (Cmd+Q on macOS / Alt+F4 on Windows/Linux)

### How to tell what needs rebuilding

Check the update diff to see which directories were affected:

```bash
# For clone users (before pulling)
git diff HEAD..origin/main --stat

# For template users (after merging)
git diff HEAD~1 --stat
```

| Changed directory | Action needed |
|---|---|
| `.sandbox/sandbox-mcp/` | Rebuild SandboxMCP |
| `dkmcp/` | Rebuild DockMCP (host OS) |
| `.devcontainer/` | Rebuild DevContainer |
| `.sandbox/scripts/` | No rebuild needed (used directly) |

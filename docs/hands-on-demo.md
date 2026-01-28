# Hands-On: Experience Secret Hiding

[Back to README](../README.md#try-the-demo)

This project uses **two hiding mechanisms**:

| Method | Effect | Use Case |
|--------|--------|----------|
| Docker mount | The file itself is invisible | `.env`, certificates, etc. |
| `.claude/settings.json` | Claude Code denies access | Secrets in source code |

---

## Method 1: Hiding via Docker Mounts

This hands-on walks you through both the **normal state** and a **misconfiguration** scenario.

### Step 1: Verify Normal State

First, confirm that secret files are properly hidden with the current configuration.

```bash
# Run inside DevContainer
# Check the iOS app's Config directory (should appear empty)
ls -la demo-apps-ios/SecureNote/Config/

# Check the Firebase config file (should be empty or missing)
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

If the directory is empty or file contents are empty, hiding is working correctly.

### Step 2: Experience a Misconfiguration

Next, intentionally comment out settings to see what happens when hiding is misconfigured.

1. Edit `.devcontainer/docker-compose.yml` and comment out the iOS-related secret settings:

```yaml
    volumes:
      # ...
      # Hide iOS app Firebase config file
      # - /dev/null:/workspace/demo-apps-ios/SecureNote/GoogleService-Info.plist:ro  # ← commented out

    tmpfs:
      # ...
      # Make iOS app config directory empty
      # - /workspace/demo-apps-ios/SecureNote/Config:ro  # ← commented out
```

2. Rebuild the DevContainer:
   - VS Code: `Cmd+Shift+P` → "Dev Containers: Rebuild Container"

### Step 3: Check Startup Warnings

After rebuilding, you'll see warnings like these in the terminal:

**Warning 1: Configuration mismatch between DevContainer and CLI Sandbox**
```
Warning: Secret configurations differ

Please synchronize both docker-compose.yml files:
  /workspace/.devcontainer/docker-compose.yml
  /workspace/cli_sandbox/docker-compose.yml
```

**Warning 2: Out of sync with .claude/settings.json**
```
Warning: The following files are not configured in docker-compose.yml:

   demo-apps-ios/SecureNote/GoogleService-Info.plist

These files are blocked in .claude/settings.json but not hidden
via volume mounts in docker-compose.yml.

To fix:
  Edit docker-compose.yml manually
  Or run: .sandbox/scripts/sync-secrets.sh
```

> **Key point:** Startup validation scripts run multiple checks to detect misconfigurations. This catches problems before AI can access any files.

### Step 4: Confirm Secrets Are Exposed

With the misconfiguration in place, check the secret file contents:

```bash
# Config directory contents are now visible
cat demo-apps-ios/SecureNote/Config/Debug.xcconfig

# Firebase config file contents are also visible
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

The misconfiguration has exposed files that should be hidden, and structural access controls are no longer effective.

### Step 5: Restore Settings

Uncomment the lines and rebuild the DevContainer to return to the normal state.

> **Summary:** Docker mount-based secret settings must be kept in sync across both DevContainer and CLI Sandbox. Misconfigurations are detected at startup and trigger warnings.

---

## Method 2: Restrictions via .claude/settings.json (Safety net + Docker mount target suggestions)

When subproject `.claude/settings.json` files define blocked files, there are two benefits:

  1. **Safety net**
    - Claude Code cannot read those files (protection even if Docker mount configuration is missing)
  2. **Docker mount target suggestions**
    - `sync-secrets.sh` reads these definitions and assists with reflecting them in Docker mount settings

In other words, `.claude/settings.json` is the source of truth for what should be hidden, and Docker mounts are derived from it.

```bash
# Example: Secrets.swift exists as a file, but...
ls demo-apps-ios/SecureNote/Secrets.swift

# Claude Code cannot read it (permission error)
```

### Syncing to Docker Mounts

To reflect `.claude/settings.json` definitions in Docker mounts:

```bash
# Sync interactively (choose which files to add)
.sandbox/scripts/sync-secrets.sh

# Options:
#   1) Add all
#   2) Confirm individually
#   3) Don't add any
#   4) Preview (dry run) ← check settings without changing files
```

> **Recommendation:** Use option `4` to preview first, then `2` to add only what you need.

### How Merging Works

```
demo-apps-ios/.claude/settings.json  ─┐
demo-apps/.claude/settings.json      ─┼─→ /workspace/.claude/settings.json
(other subprojects)                  ─┘     (merged result)
```

- **Source**: Each subproject's `.claude/settings.json` (committed to the repository)
- **Result**: `/workspace/.claude/settings.json` (not in the repository)
- **Timing**: Automatically executed at DevContainer startup

**Merge conditions:**

| State | Behavior |
|-------|----------|
| `/workspace/.claude/settings.json` doesn't exist | Merge and create |
| Exists but no manual changes | Re-merge |
| **Exists with manual changes** | Don't merge; preserve manual changes |

> If you manually edit `/workspace/.claude/settings.json`, it won't be overwritten on next startup. To reset, delete the file and restart.

```bash
# Check source files (in the repository)
cat demo-apps-ios/.claude/settings.json

# Check merged result (created at DevContainer startup)
cat /workspace/.claude/settings.json
```

> Merging is performed by `.sandbox/scripts/merge-claude-settings.sh`.

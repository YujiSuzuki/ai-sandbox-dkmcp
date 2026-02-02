# Hands-on Guide

Exercises to experience the security features firsthand.

[← Back to README](../README.md)

---

## About git status in the Demo Environment

This template force-tracks demo secret files (`.env`, files in `secrets/`) with `git add -f` so you can experience secret hiding. As a result, hidden files appear as "deleted" in git status inside the AI Sandbox.

When applying to your own project, you'll add secret files to `.gitignore`, so this issue won't occur.

To suppress git status display in the demo environment, use `skip-worktree`:

```bash
# Check if skip-worktree is already set
git ls-files -v | grep ^S

# Exclude hidden files from git status
git update-index --skip-worktree <file>

# To undo
git update-index --no-skip-worktree <file>
```

---

## Try Secret Hiding

This project uses **two hiding mechanisms**:

| Method | Effect | Use Case |
|--------|--------|----------|
| Docker mount | File itself is invisible | `.env`, certificates, etc. |
| `.claude/settings.json` | Claude Code denies access | Secrets within source code |

---

### Method 1: Docker Mount Hiding

Experience both the **normal state** and a **misconfiguration state**.

#### Step 1: Verify normal state

First, confirm that secret files are properly hidden with the current configuration.

```bash
# Run inside AI Sandbox
# Check iOS app Config directory (should appear empty)
ls -la demo-apps-ios/SecureNote/Config/

# Check Firebase config file (should be empty or not found)
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

If the directory is empty or the file contents are empty, hiding is working correctly.

#### Step 2: Experience a misconfiguration

Intentionally comment out settings to experience a misconfiguration state.

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

2. Rebuild DevContainer:
   - VS Code: `Cmd+Shift+P` → "Dev Containers: Rebuild Container"

#### Step 3: Check startup warnings

After rebuilding, warnings like these will appear in the terminal:

**Warning 1: Configuration difference between DevContainer and CLI Sandbox**
```
⚠️  Secret configurations differ

Please sync both docker-compose.yml files:
  📄 /workspace/.devcontainer/docker-compose.yml
  📄 /workspace/cli_sandbox/docker-compose.yml
```

**Warning 2: Out of sync with .claude/settings.json**
```
⚠️  The following files are not configured in docker-compose.yml:

   📄 demo-apps-ios/SecureNote/GoogleService-Info.plist

These files are blocked in .claude/settings.json but
not configured in docker-compose.yml volume mounts.

To fix:
  Manually edit docker-compose.yml
  Or run: .sandbox/scripts/sync-secrets.sh
```

> 💡 **Key point:** Startup validation scripts perform multiple checks to detect misconfigurations. This lets you catch issues before AI accesses any files.

#### Step 4: Confirm secrets are exposed

With the misconfiguration in place, check the secret file contents:

```bash
# Config directory contents are visible
cat demo-apps-ios/SecureNote/Config/Debug.xcconfig

# Firebase config file contents are also visible
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

Due to the misconfiguration, files that should be hidden are exposed inside the container, and structural access restrictions are not in effect.

#### Step 5: Restore the configuration

Uncomment the settings and rebuild to restore the normal state.

> 📝 **Summary:** Docker mount secret settings must be synchronized across both AI Sandbox environments (DevContainer and CLI Sandbox). Misconfigurations are detected at startup and warnings are displayed.

---

### Method 2: .claude/settings.json Restrictions (Safety Net + Docker Mount Suggestions)

When blocked files are defined in each subproject's `.claude/settings.json`, it provides two benefits:

  1. **Safety net**
    - Claude Code cannot read those files (protection even if Docker mount configuration is missed)
  2. **Docker mount suggestions**
    - `sync-secrets.sh` reads these definitions and assists with reflecting them in Docker mount configuration

In other words, `.claude/settings.json` is the source of truth for what should be hidden, and Docker mounts are derived from it.

```bash
# Example: Secrets.swift exists as a file, but...
ls demo-apps-ios/SecureNote/Secrets.swift

# Claude Code cannot read it (permission error)
```

**Syncing to Docker mounts:**

To reflect `.claude/settings.json` definitions in Docker mounts:

```bash
# Interactive sync (choose which files to add)
.sandbox/scripts/sync-secrets.sh

# Options:
#   1) Add all
#   2) Confirm individually
#   3) Don't add
#   4) Preview (dry run) ← check settings without changes
```

> 💡 **Recommended:** Use option `4` to preview first, then `2` to add only what you need.

**How merging works:**

```
demo-apps-ios/.claude/settings.json  ─┐
demo-apps/.claude/settings.json      ─┼─→ /workspace/.claude/settings.json
(other subprojects)                  ─┘     (merged result)
```

- **Source**: Each subproject's `.claude/settings.json` (committed to repo)
- **Result**: `/workspace/.claude/settings.json` (not in repo)
- **Timing**: Automatically executed at AI Sandbox startup

**Merge conditions:**

| State | Behavior |
|-------|----------|
| `/workspace/.claude/settings.json` doesn't exist | Created by merging |
| Exists with no manual changes | Re-merged |
| **Exists with manual changes** | Not overwritten — manual changes preserved |

> 💡 If you manually edit `/workspace/.claude/settings.json`, it won't be overwritten on next startup. To reset, delete the file and restart.

```bash
# Check source (in repo)
cat demo-apps-ios/.claude/settings.json

# Check merged result (created at AI Sandbox startup)
cat /workspace/.claude/settings.json
```

> 📝 Merging is done by `.sandbox/scripts/merge-claude-settings.sh`.

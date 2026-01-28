# AI Sandbox Environment + DockMCP

[日本語版はこちら](README.ja.md)


This project is a development environment template designed to minimize security risks while fully leveraging AI's ability to analyze systems holistically.

All you need is Docker (or OrbStack) and VS Code. (VS Code is not strictly required — the [CLI-only environment](#two-environments) also works.)


- **Cross-project development** — Let AI work across multiple codebases (mobile, API, web) in a single environment
- **Structural isolation of secrets** — Hide `.env` files and secret keys from AI via volume mounts, while real containers use them normally
- **Cross-container access via DockMCP** — AI can check logs and run tests in other containers


This project is intended for local development environments and is not designed for production use. For limitations, see "[Issues This Environment Does Not Address](#issues-this-environment-does-not-address)" and "[FAQ](#faq)."



> [!NOTE]
> **Using DockMCP standalone is not recommended.** When running AI on the host OS, the AI has the same permissions as the user, so routing through DockMCP provides no benefit. While remote host access is conceivable, there is currently no authentication mechanism. For standalone setup, see [dkmcp/README.md](dkmcp/README.md).


---

# Table of Contents

- [Problems This Environment Solves](#problems-this-environment-solves)
- [Use Cases](#use-cases)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Project Structure](#project-structure)
- [Security Features](#security-features)
- [Try the Demo](#try-the-demo)
- [Two Environments](#two-environments)
- [Advanced Usage](#advanced-usage)
- [Adapting to Your Own Project](#adapting-to-your-own-project)
- [Supported AI Tools](#supported-ai-tools)
- [FAQ](#faq)
- [Documentation](#documentation)






# Problems This Environment Solves

## Structural Protection of Secrets
Running AI on the host OS is convenient, but it's difficult to prevent access to `.env` files and secret keys. This environment isolates AI inside a sandbox container and uses volume mount controls to structurally enforce the boundary: **"code is visible, but secrets are not."**

## Cross-Project Integration
Investigating bugs that occur in the "gaps" between repositories is hard work even for human engineers. This environment consolidates multiple projects into a single workspace so AI can see the entire system. Per-subproject settings and secret hiding states are automatically verified at startup by dedicated scripts.

## Cross-Container Operations via DockMCP
DockMCP compensates for the trade-off of sandboxing — the inability to access other containers.
Based on security policies, it grants AI **"permission to read logs and run tests in other containers."** This enables AI to autonomously investigate issues like API-frontend integration failures across the entire system.



## Issues This Environment Does Not Address

**Network Restrictions**

If you need to prevent AI from accessing arbitrary external domains, consider introducing a whitelist-based proxy.

References:
- [Docker Compose Networking](https://docs.docker.com/compose/networking/)
- [Squid Proxy](http://www.squid-cache.org/Doc/)

> See also: [Anthropic's official sandbox environment](https://github.com/anthropics/claude-code/tree/main/.devcontainer) for firewall configuration examples.




# Use Cases

### 1. Microservices Development
```
workspace/
├── mobile-app/     ← Flutter/React Native
├── api-gateway/    ← Node.js
├── auth-service/   ← Go
└── db-admin/       ← Python
```

AI supports all services without exposing API keys.

### 2. Full-Stack Projects
```
workspace/
├── frontend/       ← React
├── backend/        ← Django
└── workers/        ← Celery tasks
```

AI can edit frontend code while checking backend logs.

### 3. Legacy + New
```
workspace/
├── legacy-php/     ← Old codebase
└── new-service/    ← Modern rewrite
```

AI understands both and assists with migration.

## Prerequisites

### Sandbox: Secure sandbox + secret hiding
- **Docker and Docker Compose** (Docker Desktop or OrbStack)
- **VS Code** with the **[Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)** extension

### Sandbox + DockMCP: Secure sandbox + secret hiding with AI cross-container access
- All of the above, plus:
- **DockMCP** - Install via one of these methods:
  - Download a pre-built binary from [GitHub Releases](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)
  - Build from source (can be built inside DevContainer)
- An MCP-compatible AI assistant CLI (e.g., `claude` CLI, Gemini in Agent mode)

## Architecture Overview

```
Host OS
├── DockMCP Server (:8080)
│   ├── HTTP/SSE API for AI
│   ├── Security policy enforcement
│   └── Container access gateway
│
└── Docker Engine
    ├── DevContainer (AI environment)
    │   ├── Claude Code / Gemini
    │   └── secrets/ → empty (hidden)
    │
    ├── API Container
    │   └── secrets/ → real files
    │
    └── Web Container
```

**Data flow:** AI (DevContainer) → DockMCP (:8080) → Other containers

### How Secret File Hiding Works

**Key insight:** Because AI runs inside a DevContainer, Docker volume mounts can hide secret files.

```
Host OS
├── demo-apps/securenote-api/.env  ← actual file
│
├── DevContainer (AI execution environment)
│   └── AI tries to read .env
│       → mounted from /dev/null, so it appears empty
│
└── API Container (runtime environment)
    └── Node.js app reads .env
        → reads normally
```

**Result:**
- AI cannot read secret files (security ensured)
- Applications can read secret files (functionality preserved)
- AI can still check logs and run tests via DockMCP

### Benefits of DevContainer Isolation

Running AI inside a DevContainer also restricts access to host OS files.

```
Host OS
├── /etc/            ← inaccessible to AI
├── ~/.ssh/          ← inaccessible to AI
├── ~/Documents/     ← inaccessible to AI
├── ~/other-project/ ← inaccessible to AI
├── ~/secret-memo/   ← inaccessible to AI
│
└── DevContainer
    └── /workspace/   ← only this is visible
        ├── demo-apps/
        ├── dkmcp/
        └── ...
```

**Benefits:**
- Cannot touch host OS system files
- Cannot access other projects
- Cannot access SSH keys or credentials (`~/.ssh/`)
- No risk of accidentally modifying the host OS

> **Git warning:** Inside the DevContainer, hidden files (`.env`, files in `secrets/`) appear as "deleted." Running `git commit -a` or `git add .` could accidentally commit file deletions. Perform git operations on the host, or explicitly specify files with `git add` inside the DevContainer.








# Quick Start

> **💡 Language setup (optional):** Before opening DevContainer (or cli_sandbox), run on host OS:
> ```bash
> .sandbox/scripts/init-env-files.sh -i
> ```
> Select `1) English` or `2) 日本語` for terminal output language.
> (Can also be run inside the container)

### Option A: Sandbox Only

If you only need a secure sandbox with secret hiding:

```bash
# 1. Open in VS Code
code .

# 2. Reopen in container (Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container")
```

**That's it!** AI can access code in `/workspace`, but `.env` and `secrets/` directories are hidden.

**What's protected:**
- `.env` files → mounted as empty
- `secrets/` directories → appear empty
- Host OS files → completely inaccessible

### Option B: Sandbox + DockMCP

If AI needs to check logs or run tests in other containers, use DockMCP:

#### Step 1: Start DockMCP (on host OS)

```bash
# Install DockMCP (installs to ~/go/bin/)
cd dkmcp
make install

# Start the server
dkmcp serve --config configs/dkmcp.example.yaml
```

> **Note:** Use `make install` instead of `make build`. This installs the binary to `$GOPATH/bin` rather than the workspace (which is visible from DevContainer but won't run there).

> **Important:** If you restart the DockMCP server, SSE connections are dropped. The AI assistant needs to reconnect. In Claude Code, run `/mcp` → "Reconnect."

#### Step 2: Open DevContainer

```bash
code .
# Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"
```

#### Step 3: Connect Claude Code to DockMCP

Inside the DevContainer:

```bash
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```

**Restart VS Code** to activate the MCP connection.

#### Step 4 (Recommended): Custom Domain Setup

For a more realistic development experience, set up custom domains:

**On host OS — add to `/etc/hosts`:**
```bash
# macOS/Linux
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts

# Windows (run Notepad as administrator)
# Edit: C:\Windows\System32\drivers\etc\hosts
# Add: 127.0.0.1 securenote.test api.securenote.test
```

> **Note:** The DevContainer automatically resolves custom domains to the host via `extra_hosts` in `docker-compose.yml`. No additional configuration is needed inside the container.

#### Step 5 (Optional): Try with Demo Apps

> You need to prepare `.env` and key files. See [demo-apps/README.md](demo-apps/README.md) for details.

```bash
# On host OS — start the demo apps
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d --build
```

**Access:**
- Web: http://securenote.test:8000
- API: http://api.securenote.test:8000/api/health

**From DevContainer (AI can test with curl):**
```bash
curl http://api.securenote.test:8000/api/health
curl http://securenote.test:8000
```

Now AI can:
- Check logs: "Show me the logs for securenote-api"
- Run tests: "Run npm test in securenote-api"
- Test connectivity: curl with custom domains
- Secrets remain protected

---

### Troubleshooting: DockMCP Connection

If Claude Code doesn't recognize DockMCP tools:

1. **Verify DockMCP is running**: `curl http://localhost:8080/health` (on host OS)
2. **Try MCP reconnect** — Run `/mcp` in Claude Code and select "Reconnect"
3. **Fully restart VS Code** (Cmd+Q / Alt+F4) — if Reconnect doesn't help

### Fallback: Using dkmcp client Inside DevContainer

If the MCP protocol isn't working (Claude Code or Gemini can't connect), you can use `dkmcp client` commands directly inside the DevContainer as a fallback.

> **Note:** Even if `/mcp` shows "connected," MCP tools may fail with a "Client not initialized" error. This may be caused by session management timing issues in VS Code extensions (Claude Code, Gemini Code Assist, etc.). In that case:
> 1. Try `/mcp` → "Reconnect" first (simplest fix)
> 2. If that doesn't work, AI can use `dkmcp client` commands as a fallback
> 3. As a last resort, fully restart VS Code to re-establish the connection

**Setup (first time only):**

Install dkmcp inside the DevContainer:
```bash
cd /workspace/dkmcp
make install
```

> **Note:** The Go environment is enabled by default. After installation, you can comment out the `features` block in `.devcontainer/devcontainer.json` and rebuild to reduce image size.

**Usage:**
```bash
# List containers
dkmcp client list --url http://host.docker.internal:8080

# Get logs
dkmcp client logs --url http://host.docker.internal:8080 securenote-api

# Execute commands
dkmcp client exec --url http://host.docker.internal:8080 securenote-api "npm test"
```





# Commands

| Command | Where to Run | Description |
|---------|-------------|-------------|
| `dkmcp serve` | Host OS | Start the DockMCP server |
| `dkmcp list` | Host OS | List accessible containers |
| `dkmcp client list` | DevContainer | List containers via HTTP |
| `dkmcp client logs <container>` | DevContainer | Get logs via HTTP |
| `dkmcp client exec <container> "cmd"` | DevContainer | Execute commands via HTTP |

> For detailed command options, see [dkmcp/README.md](dkmcp/README.md#cli-commands)





# Project Structure

```
workspace/
├── .sandbox/               # Shared sandbox infrastructure
│   ├── Dockerfile          # Container image definition
│   └── scripts/            # Shared scripts (validate-secrets, check-secret-sync, sync-secrets)
│
├── .devcontainer/          # VS Code Dev Container configuration
│   ├── docker-compose.yml  # Secret hiding configuration
│   └── devcontainer.json   # VS Code integration settings (extensions, port control, etc.)
│
├── cli_sandbox/            # CLI Sandbox (alternative environment)
│   ├── claude.sh           # Run Claude Code from terminal
│   ├── gemini.sh           # Run Gemini CLI from terminal
│   ├── ai_sandbox.sh       # General shell (for debugging without AI)
│   └── docker-compose.yml
│
├── dkmcp/               # MCP server for container access
│   ├── cmd/dkmcp/
│   ├── internal/
│   ├── configs/
│   └── README.md
│
├── demo-apps/              # Server-side projects
│   ├── securenote-api/     # Node.js backend
│   ├── securenote-web/     # React frontend
│   └── docker-compose.demo.yml
│
└── demo-apps-ios/          # iOS app project
    ├── SecureNote/         # SwiftUI source code
    ├── SecureNote.xcodeproj
    └── README.md
```





# Security Features

### 1. Secret Hiding

Secrets are hidden from AI using Docker volume mounts:

```yaml
# .devcontainer/docker-compose.yml
volumes:
  # Hide secret files
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  # Hide secret directories
  - /workspace/demo-apps/securenote-api/secrets:ro
```

**Result:**
- AI sees empty files/directories
- Real containers have access to actual secrets
- Development works as normal!

### 2. Controlled Container Access

DockMCP enforces security policies:

```yaml
# dkmcp.yaml
security:
  mode: "moderate"  # strict | moderate | permissive

  allowed_containers:
    - "demo-*"
    - "project_*"

  exec_whitelist:
    "securenote-api":
      - "npm test"
      - "npm run lint"
```

For details on file blocking (`blocked_paths`), auto-import from Claude Code / Gemini settings, and more, see the [dkmcp/README.md "Configuration Reference"](dkmcp/README.md#configuration-reference).

### 3. Basic Sandbox Protection

- **Non-root user**: Runs as the `node` user
- **Limited sudo**: Only package managers (apt, npm, pip)
- **Credential persistence**: Named volumes for `.claude/`, `.config/gcloud/`

> **Security note: npm/pip3 sudo risks**
>
> Allowing sudo for npm/pip3 could be exploited through malicious packages. A malicious postinstall script can execute arbitrary code with elevated privileges.
>
> **Mitigation options:**
> 1. Remove npm/pip3 from sudoers (edit `.sandbox/Dockerfile`)
> 2. Use the `npm install --ignore-scripts` flag
> 3. Pre-install required packages in the Dockerfile
> 4. Set `ignore-scripts=true` in `.npmrc`

### 4. Output Masking (Defense in Depth)

Even if secrets appear in logs or command output, DockMCP automatically masks them:

```
# Raw log output
DATABASE_URL=postgres://user:secret123@db:5432/app

# What AI sees (after masking)
DATABASE_URL=[MASKED]db:5432/app
```

Passwords, API keys, Bearer tokens, database URLs with credentials, and more are detected by default. For configuration details, see [dkmcp/README.md "Output Masking"](dkmcp/README.md#output-masking).





# Try the Demo

### Hands-On: Experience Secret Hiding

This project uses **two hiding mechanisms**:

| Method | Effect | Use Case |
|--------|--------|----------|
| Docker mount | The file itself is invisible | `.env`, certificates, etc. |
| `.claude/settings.json` | Claude Code denies access | Secrets in source code |

---

**Method 1: Hiding via Docker Mounts**

This hands-on walks you through both the **normal state** and a **misconfiguration** scenario.

#### Step 1: Verify Normal State

First, confirm that secret files are properly hidden with the current configuration.

```bash
# Run inside DevContainer
# Check the iOS app's Config directory (should appear empty)
ls -la demo-apps-ios/SecureNote/Config/

# Check the Firebase config file (should be empty or missing)
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

If the directory is empty or file contents are empty, hiding is working correctly.

#### Step 2: Experience a Misconfiguration

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

#### Step 3: Check Startup Warnings

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

#### Step 4: Confirm Secrets Are Exposed

With the misconfiguration in place, check the secret file contents:

```bash
# Config directory contents are now visible
cat demo-apps-ios/SecureNote/Config/Debug.xcconfig

# Firebase config file contents are also visible
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

The misconfiguration has exposed files that should be hidden, and structural access controls are no longer effective.

#### Step 5: Restore Settings

Uncomment the lines and rebuild the DevContainer to return to the normal state.

> **Summary:** Docker mount-based secret settings must be kept in sync across both DevContainer and CLI Sandbox. Misconfigurations are detected at startup and trigger warnings.

---

**Method 2: Restrictions via .claude/settings.json (Safety net + Docker mount target suggestions)**

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

**Syncing to Docker Mounts:**

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

**How Merging Works:**

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

---

### Demo Scenario 1: Secret Isolation

```bash
# From inside DevContainer (AI tries but fails)
$ cat demo-apps/securenote-api/secrets/jwt-secret.key
(empty)

# But ask Claude Code:
"Check if the API can access its secrets"

# Claude queries via DockMCP:
$ curl http://localhost:8080/api/demo/secrets-status

# The response proves the API has access to secrets:
{
  "secretsLoaded": true,
  "proof": {
    "jwtSecretLoaded": true,
    "jwtSecretPreview": "super-sec***"
  }
}
```

### Demo Scenario 2: Cross-Container Development

```bash
# Simulate a bug: Login fails on the web app

# Ask Claude Code:
"Login is failing. Can you check the API logs?"

# Claude fetches logs via DockMCP:
dkmcp.get_logs("securenote-api", { tail: "50" })

# Error found in logs:
"JWT verification failed - invalid secret"

# Ask Claude Code:
"Please run the API tests to verify"

# Claude runs tests via DockMCP:
dkmcp.exec_command("securenote-api", "npm test")

# Problem identified and fixed!
```

### Demo Scenario 3: Multi-Project Workspace

This workspace contains:
- **Backend API** (demo-apps/securenote-api)
- **Web Frontend** (demo-apps/securenote-web)
- **iOS App** (demo-apps-ios/)

What Claude Code can do:
- View all source code (investigate issues across app and server boundaries)
- Check logs for any container (via DockMCP)
- Run tests across projects
- Debug cross-container issues






# Two Environments

| Environment | Purpose | When to Use |
|-------------|---------|-------------|
| **DevContainer** (`.devcontainer/`) | Primary development in VS Code | Day-to-day development |
| **CLI Sandbox** (`cli_sandbox/`) | Alternative / Recovery | When DevContainer is broken |

### Why Two Environments?

It serves as a **recovery alternative**.

If the DevContainer configuration breaks:
1. VS Code can't start the DevContainer
2. Claude Code won't work either
3. You can't get AI help to fix the configuration → **stuck**

With `cli_sandbox/`:
1. Even if the DevContainer is broken
2. You can launch AI from the host
   - `./cli_sandbox/claude.sh` (Claude Code)
   - `./cli_sandbox/gemini.sh` (Gemini CLI)
3. Have AI fix the DevContainer configuration

```bash
./cli_sandbox/claude.sh   # or
./cli_sandbox/gemini.sh
# Have AI fix the broken DevContainer config
```






# Advanced Usage

### Using Plugins (Multi-Repo Setup)

When using Claude Code plugins in a multi-repo setup (each project is an independent Git repository), special handling is required. See the [Plugins Guide](docs/plugins.md) for details.

> **Note**: This section is specific to Claude Code. Not applicable to Gemini Code Assist.

### Custom DockMCP Configuration

```yaml
# dkmcp.yaml
security:
  mode: "strict"  # read-only (logs, inspect, stats)

  allowed_containers:
    - "prod-*"      # production containers only

  exec_whitelist: {}  # no command execution
```

For running multiple instances and other details, see [dkmcp/README.md "Server Startup"](dkmcp/README.md#running-multiple-instances).

### Customizing the Project Name

By default, the DevContainer project name is `<parent-directory-name>_devcontainer` (e.g., `workspace_devcontainer`).

To set a custom project name, create a `.devcontainer/.env` file:

```bash
# Copy from the example
cp .devcontainer/.env.example .devcontainer/.env
```

Contents of the `.env` file:
```bash
COMPOSE_PROJECT_NAME=ai-sandbox
```

This makes container and volume names more readable:
- Container: `ai-sandbox-ai-sandbox-1`
- Volume: `ai-sandbox_node-home`

> **Note:** The `.env` file is in `.gitignore`, so each developer can have their own settings.

### Startup Output Options

The DevContainer and CLI Sandbox run validation scripts at startup. You can control how much output they produce:

| Mode | Flag | Output |
|------|------|--------|
| Quiet | `--quiet` or `-q` | Warnings and errors only (minimal) |
| Default | (none) | Problem explanation + action required |
| Verbose | `--verbose` or `-v` | Full detailed output with decorations |

**CLI Sandbox example:**
```bash
# Minimal output (warnings only)
./cli_sandbox/ai_sandbox.sh --quiet

# Full details (useful for troubleshooting)
./cli_sandbox/ai_sandbox.sh --verbose
```

**Environment variable:**
```bash
# Set default verbosity
export STARTUP_VERBOSITY=quiet  # or: default, verbose
```

**Configuration file:** `.sandbox/config/startup.conf`
```bash
# Default verbosity for all startup scripts
STARTUP_VERBOSITY="default"

# README URL for "See README for details" messages
README_URL="README.md"
README_URL_JA="README.ja.md"  # Used when LANG=ja_JP*
```

### Excluding Files from Sync Warnings

The startup scripts check if files blocked in `.claude/settings.json` are also hidden in `docker-compose.yml`. To exclude certain patterns (like `.example` files) from these warnings, edit `.sandbox/config/sync-ignore`:

```gitignore
# Exclude example/template files from sync warnings
**/*.example
**/*.sample
**/*.template
```

This uses gitignore-style patterns. Files matching these patterns will not trigger "missing from docker-compose.yml" warnings.

### Running Multiple DevContainer Instances

If you need fully isolated DevContainer environments (e.g., different client projects), use `COMPOSE_PROJECT_NAME` to create separate instances.

#### Method A: Isolate via .env file (recommended)

Set different project names in `.devcontainer/.env`:

```bash
COMPOSE_PROJECT_NAME=client-a
```

In another workspace:

```bash
COMPOSE_PROJECT_NAME=client-b
```

#### Method B: Isolate via command line

Launch DevContainers with different project names:

```bash
# Project A
COMPOSE_PROJECT_NAME=client-a docker-compose up -d

# Project B (separate volumes will be created)
COMPOSE_PROJECT_NAME=client-b docker-compose up -d
```

> **Note:** Different project names create different volumes, so the home directory (credentials, settings, history) is not automatically shared. See "Copying the Home Directory" below.

#### Method C: Share Home Directory via Bind Mounts

To automatically share the home directory across all instances, change `docker-compose.yml` to use bind mounts:

```yaml
volumes:
  # Use bind mounts instead of named volumes
  - ~/.ai-sandbox/home:/home/node
  - ~/.ai-sandbox/gcloud:/home/node/.config/gcloud
```

**Pros:**
- Automatic sharing of home directory across all instances
- Easy to back up (just copy the host directory)

**Cons:**
- Depends on host directory structure
- May require UID/GID adjustments on Linux hosts

#### Exporting/Importing the Home Directory

You can back up or migrate the home directory (credentials, settings, history) to another workspace:

```bash
# Export entire workspace (both devcontainer and cli_sandbox)
./.sandbox/scripts/copy-credentials.sh --export /path/to/workspace ~/backup

# Export from a specific docker-compose.yml
./.sandbox/scripts/copy-credentials.sh --export .devcontainer/docker-compose.yml ~/backup

# Import into a workspace
./.sandbox/scripts/copy-credentials.sh --import ~/backup /path/to/workspace
```

**Note:** If the target volumes don't exist, you need to start the environment once first to create them.

Use cases:
- Check `~/.claude/` usage data
- Back up settings
- Migrate credentials to a new workspace
- Troubleshooting






# Adapting to Your Own Project

Here's how to use this repository as a template for your own project.

### Step 1: Clone the Repository

```bash
git clone https://github.com/your-username/ai-sandbox-environment.git
cd ai-sandbox-environment
```

### Step 2: Replace demo-apps with Your Projects

```bash
# Remove demo apps (or keep them for reference)
rm -rf demo-apps demo-apps-ios

# Place your own projects
git clone https://github.com/your-org/your-api.git
git clone https://github.com/your-org/your-web.git
```

### Step 3: Configure Secret File Hiding

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

The following checks run automatically at startup:
1. `validate-secrets.sh` — Verifies that secret hiding is actually working (auto-reads paths from docker-compose.yml)
2. `compare-secret-config.sh` — Warns if DevContainer and CLI configurations differ
3. `check-secret-sync.sh` — Warns if files blocked in AI settings are not hidden in docker-compose.yml
   - Supports: `.claude/settings.json`, `.aiexclude`, `.geminiignore`
   - Note: `.gitignore` is intentionally **not** supported — it contains many non-secret patterns (`node_modules/`, `dist/`, `*.log`) that would create noise. Use AI-specific files to explicitly list secrets only.

**Manual sync tool:** If `check-secret-sync.sh` reports unconfigured files, run `.sandbox/scripts/sync-secrets.sh` to add them interactively. Use option `4` (preview) to check settings without modifying files.

**Recommended initial setup flow:**
```bash
# 1. Enter the container without AI (AI doesn't auto-start)
./cli_sandbox/ai_sandbox.sh

# 2. Inside the container: sync secret settings interactively
.sandbox/scripts/sync-secrets.sh

# 3. Exit and rebuild DevContainer
exit
# Then open DevContainer in VS Code
```

This ensures secret configuration is complete before AI accesses any files.

Detection rules:
- `volumes` entries with `/dev/null:/workspace/...` → secret files
- `tmpfs` entries with `/workspace/...:ro` → secret directories

### Step 4: DockMCP Configuration

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

### Step 5: Rebuild DevContainer

```bash
# Open Command Palette in VS Code (Cmd/Ctrl + Shift + P)
# Run "Dev Containers: Rebuild Container"
```

### Step 6: Verify

```bash
# Confirm secret files are hidden inside DevContainer
cat your-api/.env
# → Empty or "No such file"

# Confirm DockMCP can access containers
# Ask Claude Code "Show me the container list"
# Ask Claude Code "Show me the logs for your-api"
```

### Checklist

- [ ] Configure secret files in `.devcontainer/docker-compose.yml`
- [ ] Apply the same configuration in `cli_sandbox/docker-compose.yml`
- [ ] Set container names in `dkmcp.yaml`
- [ ] Configure allowed commands in `dkmcp.yaml`
- [ ] Rebuild DevContainer
- [ ] Verify secret files are hidden
- [ ] Verify log access works via DockMCP





# Supported AI Tools

- **Claude Code** (Anthropic) — Full MCP support
- **Gemini Code Assist** (Google) — MCP support in Agent mode (configure MCP in `.gemini/settings.json`)
- **Gemini CLI** (Google) — Terminal-based (check the official site for MCP and IDE integration status)
- **Cline** (VS Code extension) — MCP integration (likely supported; not verified)





# FAQ

**Q: Why can't I ask AI to run `docker-compose up/down`?**
A: This is by design. There is a deliberate separation of responsibilities: AI "observes and suggests," humans "execute infrastructure operations." See [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) for details.

**Q: Do I need to use DockMCP?**
A: No. The sandbox works without DockMCP. DockMCP enables cross-container access.

**Q: Is it safe for production use?**
A: **No, not recommended.** DockMCP has no authentication mechanism and is designed for local development environments. Avoid using it on production or internet-facing servers. Use at your own risk.

**Q: Can I use a different secret management solution?**
A: Yes! This approach can be combined with other secret management methods.

**Q: Does it work on Windows?**
A: Yes. It works on Windows/macOS/Linux with Docker Desktop.





# Documentation

- [DockMCP Documentation](dkmcp/README.md) — MCP server setup and usage
- [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) — Why DockMCP doesn't support container lifecycle operations
- [Plugins Guide](docs/plugins.md) — Claude Code plugins in multi-repo setups
- [Demo Apps Guide](demo-apps/README.md) — How to run the SecureNote demo
- [CLI Sandbox Guide](cli_sandbox/README.md) — Terminal-based sandbox
- [CLAUDE.md](CLAUDE.md) — Guide for AI assistants

## License

MIT License - See [LICENSE](LICENSE)

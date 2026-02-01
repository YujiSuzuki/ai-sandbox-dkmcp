# AI Sandbox Environment + DockMCP

[日本語版 README はこちら](README.ja.md)


This project is a development environment template designed to minimize security risks while fully leveraging AI's ability to analyze and work across your entire codebase.


- **Cross-project development** — Let AI work across multiple codebases (mobile, API, web) in a single environment
- **Structural isolation of secrets** — Hide `.env` files and secret keys from AI via volume mounts, while real containers use them normally
- **Cross-container access via DockMCP** — AI can check logs and run tests in other containers

All you need is Docker (or OrbStack) and VS Code.

VS Code is not required. [You can also use the CLI-only environment.](#two-environments)
```
 AI Sandbox (concept)
  ├── DevContainer environment (VS Code integration)
  └── CLI Sandbox environment (terminal-based)
```

Help output after launching AI Sandbox:

```
node@sandbox:/workspace$ .sandbox/scripts/help.sh

🚀 AI Sandbox Help
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

■ Getting Started
  Open DevContainer or start CLI Sandbox. That's it.
  Secret hiding is applied automatically.

■ Auto-run on startup (no need to run manually)

  Verify secrets are properly hidden:
    .sandbox/scripts/validate-secrets.sh

  Check if AI config and docker-compose are in sync:
    .sandbox/scripts/check-secret-sync.sh

■ Run manually when needed (suggested based on results above)

  Interactively fix sync issues:
    .sandbox/scripts/sync-secrets.sh

■ DockMCP (Cross-Container Access)

  Start DockMCP server on host OS:
    cd dkmcp && make install && dkmcp serve

  Connect from AI Sandbox:
    claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

  Once connected, AI can check logs, run tests, etc. automatically.

  Show all scripts:
    .sandbox/scripts/help.sh --list

```

This project is intended for local development environments and is not designed for production use. For limitations, see "[Issues this environment does not address](#issues-this-environment-does-not-address)" and "[FAQ](#faq)".



> [!NOTE]
> **Using DockMCP standalone is not recommended.** When running AI on the host OS, AI has the same permissions as the user, so there is no benefit to going through DockMCP. While remote host access is conceivable, there is currently no authentication mechanism, meaning anyone can access it. For standalone setup, see [dkmcp/README.md](dkmcp/README.md).


---

# Table of Contents

- [Real-world problems this environment solves](#real-world-problems-this-environment-solves)
- [Use cases](#use-cases)
- [Quick start](#quick-start)
- [Commands](#commands)
- [Project structure](#project-structure)
- [Security features](#security-features)
- [Hands-on tutorial](#hands-on-tutorial)
- [Adapting to your own project](#adapting-to-your-own-project)
  - [Using as a template](#using-as-a-template)
    - [Checking for updates](#checking-for-updates)
  - [Or clone directly](#alternative-clone-directly)
  - [Customizing your project](#customizing-your-project)
- [Reference](#reference)
- [Supported AI tools](#supported-ai-tools)
- [FAQ](#faq)
- [Documentation](#documentation)





# Real-world problems this environment solves

## Structural protection of secrets
Running AI on the host OS is convenient, but preventing access to `.env` files and secret keys is difficult. This environment isolates AI inside a sandbox container and uses volume mount controls to structurally enforce the boundary: **"code is visible, but secret files are not."**

## Cross-project integration
Investigating issues that occur in the "gaps" between repositories (e.g., app-server coordination) is heavy lifting even for human engineers. This environment consolidates multiple projects into a single workspace, giving AI a bird's-eye view of the entire system. Per-subproject settings and secret hiding configurations are automatically validated at startup by dedicated scripts.

## Cross-container operations via DockMCP
DockMCP compensates for the trade-off of sandboxing — the inability to access other containers. Based on security policies, it grants AI permissions such as **"read logs from other containers and run tests."** This enables AI to autonomously investigate issues across the entire system, including coordination problems between API and frontend.



## Issues this environment does not address

**Network restrictions**

If you want to prevent AI from accessing arbitrary external domains, consider introducing a whitelist-based proxy.

Reference documentation:
- [Docker Compose networking](https://docs.docker.com/compose/networking/)
- [Squid proxy](http://www.squid-cache.org/Doc/)

> Note: [Anthropic's official sandbox environment](https://github.com/anthropics/claude-code/tree/main/.devcontainer) also includes firewall configuration examples.




# Use cases

## Development scenarios where AI Sandbox + DockMCP is useful

### 1. Microservices development
```
workspace/
├── mobile-app/     ← Flutter/React Native
├── api-gateway/    ← Node.js
├── auth-service/   ← Go
└── db-admin/       ← Python
```

AI supports all services across the board without exposing API keys.

### 2. Full-stack projects
```
workspace/
├── frontend/       ← React
├── backend/        ← Django
└── workers/        ← Celery tasks
```

AI can edit frontend code while checking backend logs.

### 3. Legacy + new
```
workspace/
├── legacy-php/     ← Old codebase
└── new-service/    ← Modern rewrite
```

AI understands both and assists with migration.

---

# Quick start

## Prerequisites

### Sandbox: Secure sandbox + secret hiding
- **Docker and Docker Compose** (Docker Desktop or OrbStack)
- **VS Code** with the **[Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)** extension

### Sandbox + DockMCP: Secure sandbox + secret hiding with AI cross-container access
- All of the above, plus:
- **DockMCP** - Install via one of the following methods:
  - Download a pre-built binary from [GitHub Releases](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases)
  - Build from source (can be built inside the AI Sandbox)
- An MCP-compatible AI assistant CLI (e.g., `claude` CLI, Gemini in Agent mode)

## Architecture overview

```
┌───────────────────────────────────────────────────┐
│ Host OS                                           │
│                                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ DockMCP Server                               │ │
│  │  HTTP/SSE API for AI                         ←─────┐
│  │  Security policy enforcement                 │ │   │
│  │  Container access gateway                    │ │   │
│  │                                              │ │   │
│  └────────────────────↑─────────────────────────┘ │   │
│                       │ :8080                     │   │
│  ┌────────────────────│─────────────────────────┐ │   │
│  │ Docker Engine      │                         │ │   │
│  │                    │                         │ │   │
│  │   AI Sandbox  ←────┘                         │ │   │
│  │    └─ Claude Code / Gemini                   │ │   │
│  │       secrets/ → empty (hidden)              │ │   │
│  │                                              │ │   │
│  │   API Container    ←───────────────────────────────┘
│  │    └─ secrets/ → real files                  │ │   │
│  │                                              │ │   │
│  │   Web Container    ←───────────────────────────────┘
│  │                                              │ │
│  └──────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────┘
```

<details>
<summary>View as tree</summary>

**Data flow:** AI (AI Sandbox) → DockMCP (:8080) → other containers

```
Host OS
├── DockMCP Server (:8080)
│   ├── HTTP/SSE API for AI
│   ├── Security policy enforcement
│   └── Container access gateway
│
└── Docker Engine
    ├── AI Sandbox (AI environment)
    │   ├── Claude Code / Gemini
    │   └── secrets/ → empty (hidden)
    │
    ├── API Container
    │   └── secrets/ → real files
    │
    └── Web Container
```

</details>


#### Why can secret files be hidden?

**Key point:** Because AI runs inside the AI Sandbox, Docker volume mounts can hide secret files.

```
Host OS
├── demo-apps/securenote-api/.env  ← actual file exists
│
├── AI Sandbox (AI execution environment)
│   └── AI tries to read .env
│       → appears empty because it's mounted to /dev/null
│
└── API Container (runtime environment)
    └── Node.js app reads .env
        → reads normally
```

**Result:**
- AI cannot read secret files (security ensured)
- Apps can read secret files (functionality preserved)
- AI can still check logs and run tests via DockMCP

#### Benefits of AI Sandbox isolation

By running AI inside the AI Sandbox, access to host OS files is also restricted.

```
Host OS
├── /etc/            ← inaccessible to AI
├── ~/.ssh/          ← inaccessible to AI
├── ~/Documents/     ← inaccessible to AI
├── ~/other-project/ ← inaccessible to AI
├── ~/secret-memo/   ← inaccessible to AI
│
└── AI Sandbox
    └── /workspace/   ← only this is visible
        ├── demo-apps/
        ├── dkmcp/
        └── ...
```

**Benefits:**
- Cannot touch host OS system files
- Cannot touch other projects
- Cannot touch SSH keys or credentials (`~/.ssh/`)
- No risk of accidentally modifying the host OS

> [!NOTE]
> **About git status in the demo environment:** This template force-tracks demo secret files with `git add -f`, so they appear as "deleted" in git status inside the AI Sandbox. When applying to your own project, secret files go in `.gitignore`, so this issue does not occur. See the [hands-on tutorial](#hands-on-tutorial) for workarounds.


<details>
<summary>Language setup (optional)</summary>

> **💡 Language setup (optional):** Before opening DevContainer (or cli_sandbox), run on host OS:
> ```bash
> .sandbox/scripts/init-host-env.sh -i
> ```
> Select `1) English` or `2) 日本語` for terminal output language.
> (Can also be run inside the container)
</details>



## Option A: Sandbox

If you only need a secure sandbox with secret hiding:

```bash
# 1. Open in VS Code
code .

# 2. Reopen in container (Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container")
```

<details>
<summary>If the <code>code</code> command is not found</summary>

**Opening from VS Code's menu:**
Select "File → Open Folder" and choose this folder.

**Installing the `code` command (macOS):**
Open the Command Palette (Cmd+Shift+P) in VS Code and run `Shell Command: Install 'code' command in PATH`. Restart your terminal and the `code` command will be available.

> Reference: [Visual Studio Code on macOS - Official documentation](https://code.visualstudio.com/docs/setup/mac)

</details>



<details>
<summary>For CLI Sandbox environment (terminal-based)</summary>

```bash
   ./cli_sandbox/claude.sh # (Claude Code)
   ./cli_sandbox/gemini.sh # (Gemini CLI)
```

</details>

**That's it!** AI can access code in `/workspace`, but `.env` and `secrets/` directories are hidden.


**What's protected:**
- `demo-apps/securenote-api/.env` → mounted as empty
- `demo-apps/securenote-api/secrets/` → appears empty
- Host OS files → inaccessible from inside the AI Sandbox




## Option B: Sandbox + DockMCP

If AI needs to check logs or run tests in other containers, use DockMCP:

### Step 1: Start the DockMCP server (on host OS)

If you have a Go environment on your host OS, install as follows. (For Go environment setup, see the [Go official site](https://go.dev/dl/).)


```bash
# Install DockMCP (installs to ~/go/bin/)
cd dkmcp
make install

# Start the server
dkmcp serve --config configs/dkmcp.example.yaml
```

> **Note:** Use `make install` instead of `make build`. This installs the binary to `$GOPATH/bin` rather than the workspace (which is visible but non-functional from the AI Sandbox).

<details>
<summary>If you don't have a Go environment on the host OS, you can build and install from inside the AI Sandbox</summary>

The AI Sandbox includes a Go environment, so you can cross-build binaries for the host OS.
The host OS type is auto-detected at container startup, so no OS specification is needed.

**1. Build inside the AI Sandbox:**

```bash
cd /workspace/dkmcp
make build-host
```

The built binary is output to `dkmcp/dist/`. This directory is also visible from the host OS.

> If auto-detection doesn't work, you can specify manually:
> `make build-host HOST_OS=darwin HOST_ARCH=arm64`

**2. Install on the host OS:**

```bash
cd <path-to-this-repo>/dkmcp

# Install to ~/go/bin (if you have a Go environment)
make install-host DEST=~/go/bin

# Install to /usr/local/bin (if you don't have a Go environment)
make install-host DEST=/usr/local/bin
```

</details>


### Step 2: Open the DevContainer

```bash
code .
# Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"
```

### Step 3: Connect Claude Code to DockMCP

In the AI Sandbox shell:

```bash
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```


In Claude Code:

Run `/mcp` → "Reconnect".

> **Important:** If you restart the DockMCP server, SSE connections are dropped. You need to reconnect from the AI assistant side. In Claude Code, run `/mcp` → "Reconnect".


### Step 4 (recommended): Custom domain setup

For a more realistic development experience, set up custom domains:

**On the host OS — add to `/etc/hosts`:**
```bash
# macOS/Linux
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts

# Windows (run Notepad as Administrator)
# Edit: C:\Windows\System32\drivers\etc\hosts
# Add: 127.0.0.1 securenote.test api.securenote.test
```

> **Note:** The AI Sandbox automatically resolves custom domains to the host via `extra_hosts` in `docker-compose.yml`. No additional configuration is needed inside the container.

### Step 5 (optional): Try the demo apps

#### Start the demo apps (on host OS)

```bash
# On host OS — start the demo apps
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d --build
```


**Access:**
- Web: http://securenote.test:8000
- API: http://api.securenote.test:8000/api/health

**From the AI Sandbox (AI can test with curl):**
```bash
curl http://api.securenote.test:8000/api/health
curl http://securenote.test:8000
```


#### Ask AI to access other containers

- Check logs:
  - "Show me the logs from securenote-api"

- Run tests:
  - "Run npm test in securenote-api"

- Health check:
  - "Run curl http://api.securenote.test:8000/api/health"

- Secrets are still protected:
  - "Check if secret files are accessible"


---

## Troubleshooting: DockMCP connection

If Claude Code doesn't recognize DockMCP tools:

1. **Check VS Code's Ports panel** — If DockMCP's port (default 8080) is being forwarded, stop it
2. **Verify DockMCP is running** — `curl http://localhost:8080/health` (on host OS)
3. **Try MCP reconnect** — Run `/mcp` in Claude Code and select "Reconnect"
4. **Fully restart VS Code** (Cmd+Q / Alt+F4) — if Reconnect doesn't resolve it


## Fallback: Using dkmcp client inside the AI Sandbox

If the MCP protocol doesn't work (Claude Code or Gemini can't connect), you can use `dkmcp client` commands directly inside the AI Sandbox as a fallback.

> **Note:** Even when `/mcp` shows "✔ connected", MCP tools may fail with a "Client not initialized" error. This may be caused by session management timing issues in the VS Code extension (Claude Code, Gemini Code Assist, etc.). In this case:
> 1. First try `/mcp` → "Reconnect" (quickest solution)
> 2. If that doesn't work, AI can use `dkmcp client` commands as a fallback
> 3. As a last resort, fully restart VS Code to re-establish the connection

**Setup (first time only):**

Install dkmcp inside the AI Sandbox:
```bash
cd /workspace/dkmcp
make install
```

> **Note:** The Go environment is enabled by default. After installation, if you want to reduce image size, you can comment out the `features` block in `.devcontainer/devcontainer.json` and rebuild.

**Usage:**
```bash
# List containers
dkmcp client list

# Get logs
dkmcp client logs securenote-api

# Execute commands
dkmcp client exec securenote-api "npm test"
```

> **About `--url`:** Connects to `http://host.docker.internal:8080` by default. If you change the server port in `dkmcp.yaml`, specify the URL explicitly via the `--url` flag or the `DOCKMCP_SERVER_URL` environment variable.
> ```bash
> dkmcp client list --url http://host.docker.internal:9090
> # or
> export DOCKMCP_SERVER_URL=http://host.docker.internal:9090
> ```





# Commands

| Command | Where to run | Description |
|---------|-------------|-------------|
| `dkmcp serve` | Host OS | Start the DockMCP server |
| `dkmcp list` | Host OS | List accessible containers |
| `dkmcp client list` | AI Sandbox | List containers via HTTP |
| `dkmcp client logs <container>` | AI Sandbox | Get logs via HTTP |
| `dkmcp client exec <container> "cmd"` | AI Sandbox | Execute commands via HTTP |

> For detailed command options, see [dkmcp/README.md](dkmcp/README.md#cli-commands)





# Project structure

Shared infrastructure is in `.sandbox/`, the two sandbox environments are in `.devcontainer/` and `cli_sandbox/`, the MCP server is in `dkmcp/`, and demo apps are in `demo-apps/` and `demo-apps-ios/`.

<details>
<summary>View directory tree</summary>

```
workspace/
├── .sandbox/               # Shared sandbox infrastructure
│   ├── Dockerfile          # Container image definition
│   └── scripts/            # Shared scripts (validate-secrets, check-secret-sync, sync-secrets)
│
├── .devcontainer/          # VS Code Dev Container settings
│   ├── docker-compose.yml  # Secret hiding configuration
│   └── devcontainer.json   # VS Code integration settings (extensions, port control, etc.)
│
├── cli_sandbox/             # CLI Sandbox (alternative environment)
│   ├── claude.sh           # Run Claude Code from terminal
│   ├── gemini.sh           # Run Gemini CLI from terminal
│   ├── ai_sandbox.sh       # General-purpose shell (for debugging without AI)
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

</details>

In practice, delete the demo apps `demo-apps/` and `demo-apps-ios/` and replace them with your own projects.


# Security features

## 1. Secret hiding

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
- Real containers access actual secrets
- Development works as normal!

**How it works in practice:**

```bash
# From inside the AI Sandbox (AI's attempt fails)
$ cat demo-apps/securenote-api/secrets/jwt-secret.key
(empty)

# But ask Claude Code:
"Check if the API can access its secrets"

# Claude uses DockMCP to query:
$ curl http://localhost:8080/api/demo/secrets-status

# Response proves the API has its secrets:
{
  "secretsLoaded": true,
  "proof": {
    "jwtSecretLoaded": true,
    "jwtSecretPreview": "super-sec***"
  }
}
```

## 2. Controlled container access

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

For details on blocking sensitive file paths within containers (`blocked_paths`), auto-importing from Claude Code / Gemini settings, and more, see [dkmcp/README.md "Configuration reference"](dkmcp/README.md#configuration-reference).

**How it works in practice — cross-container debugging:**

```bash
# Simulate a bug: can't log in on the web app

# Ask Claude Code:
"Login is failing. Can you check the API logs?"

# Claude uses DockMCP to fetch logs:
dkmcp.get_logs("securenote-api", { tail: "50" })

# Finds an error in the logs:
"JWT verification failed - invalid secret"

# Ask Claude Code:
"Please run the API tests to check"

# Claude runs tests via DockMCP:
dkmcp.exec_command("securenote-api", "npm test")

# Problem identified and fixed!
```

## 3. Basic sandbox protection

- **Non-root user**: Runs as `node` user
- **Limited sudo**: Package managers only (apt, npm, pip)
- **Credential persistence**: Named volumes for `.claude/`, `.config/gcloud/`

> ⚠️ **Security note: npm/pip3 sudo risks**
>
> Allowing sudo for npm/pip3 can be exploited through malicious packages. Malicious postinstall scripts can execute arbitrary code with elevated privileges.
>
> **Mitigation options:**
> 1. Remove npm/pip3 from sudoers (edit `.sandbox/Dockerfile`)
> 2. Use the `npm install --ignore-scripts` flag
> 3. Pre-install required packages in the Dockerfile
> 4. Set `ignore-scripts=true` in `.npmrc`

## 4. Output masking (defense in depth)

Even if secrets appear in logs or command output, DockMCP automatically masks them:

```
# Raw log output
DATABASE_URL=postgres://user:secret123@db:5432/app

# What AI sees (after masking)
DATABASE_URL=[MASKED]db:5432/app
```

Detects passwords, API keys, Bearer tokens, database URLs with credentials, and more by default. For configuration details, see [dkmcp/README.md "Output masking"](dkmcp/README.md#output-masking).

## Multi-project workspace

These security features enable safely working with multiple projects in a single workspace.

Example from this demo environment:
- **Backend API** (demo-apps/securenote-api)
- **Web frontend** (demo-apps/securenote-web)
- **iOS app** (demo-apps-ios/)

What AI can do:
- Read all source code (enabling investigation of app-server coordination issues)
- Check logs from any container (via DockMCP)
- Run tests across projects
- Debug cross-container issues
- **Never touch any secrets**





# Hands-on tutorial

Exercises to experience the security features firsthand.

## About git status in the demo environment

This template force-tracks demo secret files (`.env`, files in `secrets/`) with `git add -f` so you can experience secret hiding. As a result, hidden files appear as "deleted" when viewing git status inside the AI Sandbox.

Normally, when applying to your own project, secret files go in `.gitignore`, so this issue does not occur.

To suppress git status display for the demo environment, use `skip-worktree`:

```bash
# Check if skip-worktree is already set
git ls-files -v | grep ^S

# Exclude hidden files from git status
git update-index --skip-worktree <file>

# To undo
git update-index --no-skip-worktree <file>
```

---

## Experience secret hiding

This project uses **two hiding mechanisms**:

| Method | Effect | Use case |
|--------|--------|----------|
| Docker mount | File itself is invisible | `.env`, certificates, etc. |
| `.claude/settings.json` | Claude Code denies access | Secrets within source code |

---

**🔹 Method 1: Hiding via Docker mounts**

This hands-on exercise lets you experience both the **normal state** and a **misconfigured state** of secret settings.

### Step 1: Verify the normal state

First, confirm that secret files are properly hidden with the current settings.

```bash
# Run inside the AI Sandbox
# Check the iOS app's Config directory (should appear empty)
ls -la demo-apps-ios/SecureNote/Config/

# Check the Firebase config file (should be empty or not found)
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

If the directory is empty or the file content is empty, the hiding is working correctly.

### Step 2: Experience a misconfiguration

Next, intentionally comment out settings to experience a misconfigured state.

1. Edit `.devcontainer/docker-compose.yml` and comment out the iOS-related secret settings:

```yaml
    volumes:
      # ...
      # Hide iOS app Firebase config file
      # - /dev/null:/workspace/demo-apps-ios/SecureNote/GoogleService-Info.plist:ro  # ← comment out

    tmpfs:
      # ...
      # Make iOS app config directory empty
      # - /workspace/demo-apps-ios/SecureNote/Config:ro  # ← comment out
```

2. Rebuild the DevContainer:
   - VS Code: `Cmd+Shift+P` → "Dev Containers: Rebuild Container"

### Step 3: Check startup warnings

After rebuilding, the terminal displays warnings like:

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

These files are blocked in .claude/settings.json but are not
configured as volume mounts in docker-compose.yml.

Action:
  Edit docker-compose.yml manually
  Or run: .sandbox/scripts/sync-secrets.sh
```

> 💡 **Key point:** Startup validation scripts perform multiple checks and detect misconfigurations. This allows you to notice problems before AI accesses any files.

### Step 4: Confirm that secrets are exposed

With the misconfiguration in place, check the secret file contents:

```bash
# Config directory contents are visible
cat demo-apps-ios/SecureNote/Config/Debug.xcconfig

# Firebase config file contents are also visible
cat demo-apps-ios/SecureNote/GoogleService-Info.plist
```

Due to the misconfiguration, files that should be hidden are exposed inside the container, and structural access controls are not in effect.

### Step 5: Restore the settings

Uncomment the settings and rebuild to return to the normal state.

> 📝 **Summary:** Docker mount secret settings must be kept in sync across both AI Sandbox environments (DevContainer and CLI Sandbox). Misconfigurations are detected at startup and warnings are displayed.

---

**🔹 Method 2: Restrictions via .claude/settings.json (safety net + Docker mount target suggestions)**

When subprojects define blocked files in their `.claude/settings.json`, this has two effects:

  1. **Safety net**
    - Claude Code cannot read those files (protection even if Docker mount configuration is missed)
  2. **Docker mount target suggestions**
    - `sync-secrets.sh` reads these definitions and assists with reflecting them in Docker mount settings

In other words, `.claude/settings.json` is the source of truth for secret definitions, and Docker mounts are derived from it.

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
#   4) Preview (dry run) ← check settings without modifying files
```

> 💡 **Recommended:** Check with option `4` (preview) first, then use option `2` to add only what's needed.

**How merging works:**

```
demo-apps-ios/.claude/settings.json  ─┐
demo-apps/.claude/settings.json      ─┼─→ /workspace/.claude/settings.json
(other subprojects)                  ─┘     (merged result)
```

- **Merge sources**: Each subproject's `.claude/settings.json` (committed to the repository)
- **Merge result**: `/workspace/.claude/settings.json` (not in the repository)
- **Timing**: Automatically executed at AI Sandbox startup

**Merge conditions:**

| State | Behavior |
|-------|----------|
| `/workspace/.claude/settings.json` doesn't exist | Created by merging |
| Exists but no manual changes | Re-merged |
| **Exists with manual changes** | Not overwritten — manual changes preserved |

> 💡 If you manually edit `/workspace/.claude/settings.json`, it won't be overwritten on next startup. To reset, delete the file and restart.

```bash
# Check merge sources (in the repository)
cat demo-apps-ios/.claude/settings.json

# Check merge result (created at AI Sandbox startup)
cat /workspace/.claude/settings.json
```

> 📝 Merging is performed by `.sandbox/scripts/merge-claude-settings.sh`.





# Adapting to your own project

This repository is designed as a **GitHub template repository**. You can create your own project from the template.

## Using as a template

### Step 1: Create from the template

On GitHub, click **"Use this template"** → **"Create a new repository"**.

Characteristics of the created repository:
- All files from the template (without this repository's commit history)
- Starts with a fresh Git history
- Independent from the upstream (no automatic syncing)

### Step 2: Clone the new repository

```bash
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo
```

### Checking for updates

Repositories created from the template don't automatically receive upstream updates, so an **update notification feature** is built in. At AI Sandbox startup, it checks GitHub for new releases and notifies you if a new version is available.

<details>
<summary>Notification example and configuration details</summary>

**How it works:**
- By default, it checks **all releases including pre-releases**, so you receive bug fixes and improvements promptly
- On first startup, it only records the latest version and does not display a notification
- From the second check onward, if a new version is found, a notification like the following is displayed:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📦 Update check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Current version:  v0.1.0
  Latest version:   v0.2.0

  How to update:
    1. Check the release notes for changes
    2. Manually apply the necessary changes

  Release notes:
    https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**How to apply updates:**
1. Check the [release notes](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases) for changes
2. Manually apply the necessary changes to your project

**Configuration file:** `.sandbox/config/template-source.conf`
```bash
TEMPLATE_REPO="YujiSuzuki/ai-sandbox-dkmcp"
CHECK_CHANNEL="all"            # "all" = including pre-releases, "stable" = official releases only
CHECK_UPDATES="true"           # "false" to disable
CHECK_INTERVAL_HOURS="24"      # Check interval (0 = every time)
```

| `CHECK_CHANNEL` | Behavior | Use case |
|---|---|---|
| `"all"` (default) | Checks all releases including pre-releases | Get bug fixes and improvements promptly |
| `"stable"` | Checks official releases only | Track only stable milestones |

</details>

---

## Alternative: Clone directly

If you want to track upstream changes via Git (e.g., for contribution purposes):

```bash
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

---

## Customizing your project

Whether you used the template or cloned directly, follow these steps to customize the environment.

### Replace demo-apps with your own projects

```bash
# Remove demo apps (or keep for reference)
rm -rf demo-apps demo-apps-ios

# Place your own projects
git clone https://github.com/your-org/your-api.git
git clone https://github.com/your-org/your-web.git
```

### Configure secret file hiding

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
   - Note: `.gitignore` is intentionally **not supported** — it contains many non-secret patterns (`node_modules/`, `dist/`, `*.log`) that would create noise. Explicitly list only secrets in AI-specific files.

**Manual sync tool:** If `check-secret-sync.sh` reports unconfigured files, run `.sandbox/scripts/sync-secrets.sh` to interactively add them. Option `4` (preview) lets you check settings without modifying files.

**Recommended initial setup flow:**
```bash
# 1. Enter the container without AI (AI doesn't auto-start)
./cli_sandbox/ai_sandbox.sh

# 2. Inside the container: interactively sync secret settings
.sandbox/scripts/sync-secrets.sh

# 3. Exit and rebuild the DevContainer
exit
# Then open the DevContainer in VS Code
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

For a stricter configuration:

```yaml
security:
  mode: "strict"  # Read-only (logs, inspect, stats)

  allowed_containers:
    - "prod-*"      # Production containers only

  exec_whitelist: {}  # No command execution
```

For details on starting multiple instances and more, see [dkmcp/README.md "Server startup"](dkmcp/README.md#starting-multiple-instances).

### AI assistant configuration

Edit the following files so AI assistants correctly understand your project structure and secret policies.

**Automatically applied (no action needed):**

If subprojects already have `.claude/settings.json`, they are automatically merged at AI Sandbox startup (`merge-claude-settings.sh`). No need to create new ones.

**Files that need editing:**

| File | Content | Action |
|------|---------|--------|
| `CLAUDE.md` | Project description for Claude Code | Remove demo-app-specific content and rewrite for your project |
| `GEMINI.md` | Project description for Gemini Code Assist | Same as above |
| `.aiexclude` | Secret patterns for Gemini Code Assist | Add your secret paths as needed |
| `.geminiignore` | Secret patterns for Gemini CLI | Same as above |

**CLAUDE.md / GEMINI.md editing guidelines:**

- **Keep**: DockMCP MCP Tools usage, security architecture overview, environment separation (What Runs Where)
- **Rewrite**: Project structure, Common Tasks examples
- **Remove**: SecureNote demo-specific content, demo scenario descriptions

### Using plugins (multi-repo setups)

When using Claude Code plugins with multi-repo setups (each project is an independent Git repository), some adjustments are needed. See the [plugin guide](docs/plugins.md) for details.

> **Note**: This section is Claude Code-specific. It does not apply to Gemini Code Assist.

### Rebuild the DevContainer

```bash
# Open Command Palette in VS Code (Cmd/Ctrl + Shift + P)
# Run "Dev Containers: Rebuild Container"
```

### Verify

```bash
# Confirm secret files are hidden inside the AI Sandbox
cat your-api/.env
# → empty or "No such file"

# Confirm container access via DockMCP
# Ask Claude Code: "Show me the list of containers"
# Ask Claude Code: "Show me the logs from your-api"
```

### Checklist

- [ ] Configure secret files in `.devcontainer/docker-compose.yml`
- [ ] Apply the same settings in `cli_sandbox/docker-compose.yml`
- [ ] Set container names in `dkmcp.yaml`
- [ ] Configure allowed commands in `dkmcp.yaml`
- [ ] Edit `CLAUDE.md` / `GEMINI.md` for your project
- [ ] Add secret paths to `.aiexclude` / `.geminiignore` (as needed)
- [ ] Rebuild the DevContainer
- [ ] Verify secret files are hidden
- [ ] Verify log access via DockMCP





# Reference

## Two environments

| Environment | Purpose | When to use |
|-------------|---------|-------------|
| **DevContainer** (`.devcontainer/`) | Primary development in VS Code | Day-to-day development |
| **CLI Sandbox** (`cli_sandbox/`) | Alternative / recovery | When DevContainer is broken |

**Why two environments?**

**As a recovery alternative.**

If the Dev Container configuration breaks:
1. VS Code can't start the Dev Container
2. Claude Code can't run either
3. You can't get AI help to fix the configuration → **stuck**

With `cli_sandbox/`:
1. Even if the Dev Container is broken
2. You can start AI from the host
   - `./cli_sandbox/claude.sh` (Claude Code)
   - `./cli_sandbox/gemini.sh` (Gemini CLI)
3. Have AI fix the Dev Container configuration

```bash
./cli_sandbox/claude.sh   # or
./cli_sandbox/gemini.sh
# Have AI fix the broken DevContainer configuration
```

## Customizing the project name

By default, the DevContainer project name is `<parent-directory-name>_devcontainer` (e.g., `workspace_devcontainer`).

To set a custom project name, create a `.devcontainer/.env` file:

```bash
# Copy .env.example
cp .devcontainer/.env.example .devcontainer/.env
```

`.env` file content:
```bash
COMPOSE_PROJECT_NAME=ai-sandbox
```

This makes container and volume names more descriptive:
- Container: `ai-sandbox-ai-sandbox-1`
- Volume: `ai-sandbox_node-home`

> **Note:** The `.env` file is in `.gitignore`, so each developer can have their own settings.

## Startup output options

Both AI Sandbox environments (DevContainer and CLI Sandbox) run validation scripts at startup. You can control the verbosity of the output:

| Mode | Flag | Output |
|------|------|--------|
| Quiet | `--quiet` or `-q` | Warnings and errors only (minimal) |
| Summary | `--summary` or `-s` | Condensed summary |
| Verbose | (none, default) | Detailed output with decorative borders |

**CLI Sandbox examples:**
```bash
# Minimal output (warnings only)
./cli_sandbox/ai_sandbox.sh --quiet

# Condensed summary
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
# Keep only the latest 10
BACKUP_KEEP_COUNT=10

# Temporarily override via environment variable
BACKUP_KEEP_COUNT=10 .sandbox/scripts/sync-secrets.sh
```

## Excluding files from sync warnings

The startup script checks whether files blocked in `.claude/settings.json` are also hidden in `docker-compose.yml`. To exclude certain patterns (such as `.example` files) from warnings, edit `.sandbox/config/sync-ignore`:

```gitignore
# Exclude example/template files from sync warnings
**/*.example
**/*.sample
**/*.template
```

This uses gitignore-style patterns. Files matching these patterns will not trigger "not configured in docker-compose.yml" warnings.

## Running multiple DevContainers

If you need fully isolated DevContainer environments (e.g., different client projects), you can use `COMPOSE_PROJECT_NAME` to create isolated instances.

<details>
<summary>Methods and home directory sharing</summary>

### Method A: Isolate via .env file (recommended)

Set a different project name in `.devcontainer/.env`:

```bash
COMPOSE_PROJECT_NAME=client-a
```

In another workspace:

```bash
COMPOSE_PROJECT_NAME=client-b
```

### Method B: Isolate via command line

Start DevContainers with different project names:

```bash
# Project A
COMPOSE_PROJECT_NAME=client-a docker-compose up -d

# Project B (creates separate volumes)
COMPOSE_PROJECT_NAME=client-b docker-compose up -d
```

> ⚠️ **Note:** Different project names create different volumes, so the home directory (credentials, settings, history) is not shared automatically. See "Home directory export/import" below.

### Method C: Share home directory via bind mount

To automatically share the home directory across all instances, change `docker-compose.yml` to use bind mounts:

```yaml
volumes:
  # Bind mounts instead of named volumes
  - ~/.ai-sandbox/home:/home/node
  - ~/.ai-sandbox/gcloud:/home/node/.config/gcloud
```

**Pros:**
- Automatic home directory sharing across all instances
- Easy backup (just copy the host directory)

**Cons:**
- Depends on host directory structure
- UID/GID adjustment may be needed on Linux hosts

### Home directory export/import

You can back up the home directory (credentials, settings, history) or migrate it to another workspace:

```bash
# Export the entire workspace (both devcontainer and cli_sandbox)
./.sandbox/scripts/copy-credentials.sh --export /path/to/workspace ~/backup

# Export from a specific docker-compose.yml
./.sandbox/scripts/copy-credentials.sh --export .devcontainer/docker-compose.yml ~/backup

# Import to a workspace
./.sandbox/scripts/copy-credentials.sh --import ~/backup /path/to/workspace
```

**Note:** If the target volume doesn't exist, you need to start the environment once first to create the volume.

Use cases:
- Check `~/.claude/` usage data
- Back up settings
- Migrate credentials to a new workspace
- Troubleshooting

</details>

## Uninstalling DockMCP

If you no longer need DockMCP, remove the binary from its install location:

```bash
rm ~/go/bin/dkmcp
# or
rm /usr/local/bin/dkmcp
```



# Supported AI tools

- ✅ **Claude Code** (Anthropic) - Full MCP support
- ✅ **Gemini Code Assist** (Google) - MCP support in Agent mode (configure MCP in `.gemini/settings.json`)
- ✅ **Gemini CLI** (Google) - Terminal-based (MCP integration and IDE integration support status is unclear; refer to the official site)
- ✅ **Cline** (VS Code extension) - MCP integration (likely supported; not verified)





# FAQ

**Q: Why can't I ask AI to run `docker-compose up/down`?**
A: This is by design. There is a deliberate separation of responsibilities: AI "observes and suggests," while humans "execute infrastructure operations." For details, see [DockMCP design philosophy](dkmcp/README.md#design-philosophy).

**Q: Do I need to use DockMCP?**
A: No. The sandbox functions normally without DockMCP. DockMCP enables cross-container access.

**Q: Is it safe to use in production?**
A: **No, this is not recommended.** DockMCP has no authentication mechanism, so it is designed for local development environments. Avoid using it on production environments or internet-facing servers. Use at your own risk.

**Q: Can I use a different secret management system?**
A: Yes! This can be combined with other secret management methods.

**Q: Does it work on Windows?**
A: Yes. It works on Windows/macOS/Linux as long as Docker Desktop is installed.





# Documentation

- [DockMCP documentation](dkmcp/README.md) - MCP server setup and usage
- [DockMCP design philosophy](dkmcp/README.md#design-philosophy) - Why DockMCP does not support container lifecycle operations
- [Plugin guide](docs/plugins.md) - Using Claude Code plugins with multi-repo setups
- [Demo app guide](demo-apps/README.md) - How to run the SecureNote demo
- [CLI Sandbox guide](cli_sandbox/README.md) - Terminal-based sandbox
- [CLAUDE.md](CLAUDE.md) - Instructions for AI assistants

## License

MIT License - See [LICENSE](LICENSE)

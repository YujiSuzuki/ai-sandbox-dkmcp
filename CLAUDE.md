# AI Sandbox Environment with DockMCP - Context for AI Assistants

> **Language policy:** This file must be written in English. AI assistants adding or editing content here should always use English.

This document provides essential context for AI assistants (Claude Code, Gemini Code Assist, etc.) working with this project.

## What This Project Is

This is a **comprehensive AI development environment** that demonstrates:

1. **Safe AI Usage** - Run AI coding assistants in isolated Docker containers
2. **Secret Protection** - Hide sensitive files from AI while maintaining full functionality
3. **Cross-Container Access** - AI can interact with other containers via DockMCP MCP server
4. **Multi-Project Workspaces** - Work on multiple related projects (mobile, API, web) simultaneously

## Key Innovation: DockMCP

**DockMCP** is an MCP (Model Context Protocol) server that runs on the host OS and provides controlled access to Docker containers. This allows AI assistants inside the AI Sandbox to:

- ✅ Check logs from other containers
- ✅ Run tests in other containers
- ✅ Inspect container stats
- ❌ Cannot access secrets (they're hidden via volume mounts)

This solves the common problem: "My API is in a separate container, how can AI help debug it?"

## Project Structure

```
/workspace/
├── .sandbox/               # Shared sandbox infrastructure
│   ├── Dockerfile          # Node.js base with limited sudo
│   ├── backups/            # Backup files from sync scripts (gitignored)
│   ├── config/             # Startup configuration
│   │   ├── startup.conf    # Verbosity settings, README URLs, backup retention
│   │   └── sync-ignore     # Patterns to exclude from sync warnings
│   ├── scripts/            # Shared scripts (run: .sandbox/scripts/help.sh)
│   │   ├── help.sh                   # Show this script list with descriptions
│   │   ├── _startup_common.sh        # Common functions for startup scripts
│   │   ├── validate-secrets.sh       # 🐳 Secret hiding verification
│   │   ├── compare-secret-config.sh  # DevContainer/CLI config diff check
│   │   ├── check-secret-sync.sh      # Check if Claude deny files are hidden in docker-compose
│   │   ├── sync-secrets.sh           # 🐳 Interactive tool to sync secrets to docker-compose
│   │   ├── sync-compose-secrets.sh   # 🐳 Sync secret config between DevContainer/CLI compose
│   │   ├── merge-claude-settings.sh  # Merge subproject .claude/settings.json
│   │   ├── init-host-env.sh           # Host-side init: env files from templates + host OS info
│   │   ├── run-all-tests.sh          # Run all test scripts
│   │   └── test-*.sh                 # Test scripts for each utility
│   ├── host-tools/            # 🖥️ Host-only tool scripts (run on host OS)
│   │   ├── copy-credentials.sh       # Export/Import home directory between compose projects
│   │   ├── demo-build.sh             # Build demo app Docker images
│   │   ├── demo-up.sh                # Start demo apps
│   │   └── demo-down.sh              # Stop demo apps
│   ├── tools/               # Utility tools (extras)
│   │   ├── search-history.go         # Claude Code conversation history search
│   │   └── usage-report.go           # Token usage report (model/period breakdown)
│   └── sandbox-mcp/          # Sandbox-Tools MCP Server (stdio, Go)
│       ├── cmd/sandbox-mcp/        # Entry point
│       └── internal/                 # Implementation
│
├── .devcontainer/          # VS Code Dev Container (AI environment)
│   ├── docker-compose.yml  # ⚠️ Secret hiding configuration
│   └── devcontainer.json   # VS Code DevContainer settings
│
├── cli_sandbox/            # CLI environment (backup)
│   ├── claude.sh           # Run Claude Code from terminal
│   ├── gemini.sh           # Run Gemini CLI from terminal
│   ├── ai_sandbox.sh       # Enter shell
│   └── docker-compose.yml  # ⚠️ Secret hiding configuration
│
├── dkmcp/               # MCP Server (Go)
│   ├── cmd/dkmcp/       # Main entry point
│   ├── internal/           # Core implementation
│   │   ├── mcp/            # MCP server & tools
│   │   ├── docker/         # Docker client wrapper
│   │   ├── security/       # Policy enforcement
│   │   ├── config/         # Configuration management
│   │   ├── cli/            # CLI commands
│   │   └── client/         # DockMCP client
│   └── configs/            # Example configurations
│
├── demo-apps/              # Demo Application (Server-side)
│   ├── securenote-api/     # Node.js API with secrets
│   ├── securenote-web/     # React frontend
│   └── docker-compose.demo.yml
│
└── demo-apps-ios/          # iOS Application
    └── SecureNote/         # SwiftUI source code
```

**Script icons:** 🐳 = Run in container only, 🖥️ = Run on host OS only

## Security Architecture

### 1. Secret Hiding (Volume Mounts)

Secrets are hidden from AI using Docker volume mounts:

```yaml
# In .devcontainer/docker-compose.yml and cli_sandbox/docker-compose.yml
volumes:
  # Hide .env file
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  # Hide secrets directory (appears empty)
  - /workspace/demo-apps/securenote-api/secrets:ro
```

**Result:**
- AI sees empty files/directories
- Real containers (demo-apps) access actual secrets
- Functionality is preserved!

### 2. Controlled Container Access (DockMCP)

DockMCP enforces security policies:

```yaml
# In dkmcp.yaml
security:
  mode: "moderate"  # strict | moderate | permissive

  allowed_containers:
    - "securenote-*"
    - "demo-*"

  exec_whitelist:
    "securenote-api":
      - "npm test"
      - "npm run lint"
```

### 3. Sandbox Protection

- **Non-root user**: Runs as `node` user
- **Limited sudo**: Only `apt`, `npm`, `pip3` allowed
- **No Docker socket**: AI cannot access `/var/run/docker.sock`

**IMPORTANT: What AI CANNOT do in AI Sandbox:**
- ❌ Run `docker` commands (no Docker CLI access)
- ❌ Run `docker-compose` commands (no Docker socket)
- ❌ Start/stop containers directly
- ❌ Build Docker images

**These operations MUST be done on the host OS by the user.**

## Critical Files - Handle with Care

### ⚠️ Security Configuration Files

These files define what AI can and cannot access:

1. **`.devcontainer/docker-compose.yml`** (line 24-29)
   - Defines which secrets are hidden from AI
   - Changes affect AI's visibility

2. **`cli_sandbox/docker-compose.yml`**
   - Same as above for CLI environment

3. **`dkmcp/configs/dkmcp.example.yaml`**
   - Defines which containers AI can access
   - Whitelists allowed commands

4. **`.devcontainer/devcontainer.json`**
   - VS Code DevContainer integration settings
   - **Key configurations:**
     - `features`: Go environment for DockMCP development
     - `customizations.vscode.extensions`: Auto-installed extensions (Claude Code, ESLint, Go, etc.)
     - `otherPortsAttributes`: Disables auto port forwarding (enables direct connection to host DockMCP)
     - `postStartCommand`: Runs settings sync check on startup
   - Changes affect VS Code behavior and available tools

### ✅ Safe to Modify

- Demo application code (`demo-apps/`)
- DockMCP implementation (`dkmcp/internal/`)
- Documentation (`README.md`, `README.ja.md`)
- Shell scripts (with user approval)

### 🤔 Requires User Confirmation

- Adding/removing secret exclusions
- Changing security mode in DockMCP
- Modifying sudoers configuration
- Adding new DockMCP MCP tools

### Understanding `/workspace/.claude/settings.json`

This file controls what you (AI) can read. It's **automatically merged** from subproject settings on AI Sandbox startup.

**Merge behavior:**

| State | What happens |
|-------|--------------|
| File doesn't exist | Created by merging subproject settings |
| Exists, no manual changes | Re-merged from subprojects |
| **Exists with manual changes** | **Not overwritten** - manual changes preserved |

**Important:** If a user asks you to read a file but you get a permission error, check if it's blocked in `/workspace/.claude/settings.json`. To unblock temporarily, the user can edit this file directly (it won't be overwritten if changed manually).

**Source files:**
- `demo-apps-ios/.claude/settings.json`
- `demo-apps/.claude/settings.json`
- Other subproject `.claude/settings.json` files

## Important: Guidance for Users vs AI Assistants

**This document (CLAUDE.md) is for AI assistants.** When users ask questions about:
- Setup and installation
- Troubleshooting connection issues
- How to use this project
- General project guidance

**Direct them to README.md or README.ja.md instead.**

Example:
- User: "How do I start the demo apps?"
  → Response: "Please see README.md for setup instructions"

- User: "Why can't I read the .env file?"
  → Response: "This is explained in docs/architecture.md under 'How Secret Hiding Works'"

- User: "DockMCP isn't connecting"
  → Response: "See docs/reference.md → Troubleshooting section"

Please see README.md (and docs/ for detailed topics) for practical guidance to users.

## Working with This Project

### Startup Verbosity Options

The CLI Sandbox accepts verbosity flags that control startup output:

```bash
./cli_sandbox/ai_sandbox.sh --quiet    # Warnings/errors only (minimal)
./cli_sandbox/ai_sandbox.sh --summary  # Condensed summary
./cli_sandbox/ai_sandbox.sh            # Default: full detailed output
```

The `STARTUP_VERBOSITY` environment variable can also be set (`quiet`, `summary`, `verbose`).

Configuration files in `.sandbox/config/`:
- `startup.conf` - Default verbosity, README URLs for locale-aware messages, backup retention (`BACKUP_KEEP_COUNT`)
- `sync-ignore` - gitignore-style patterns to exclude from sync warnings (e.g., `**/*.example`)

### AI Settings Files for Secret Sync

The `check-secret-sync.sh` script reads patterns from these AI-specific files:
- `.claude/settings.json` - Claude Code
- `.aiexclude` - Gemini Code Assist
- `.geminiignore` - Gemini CLI

**Why `.gitignore` is NOT supported:**

`.gitignore` contains many non-secret patterns (`node_modules/`, `dist/`, `*.log`, `.DS_Store`) that would create noise in sync checks. AI exclusion files should explicitly list only secrets, keeping the intent clear and maintenance easy. If a user asks why `.gitignore` isn't checked, explain this design decision.

### Understanding the Demo

The SecureNote demo application demonstrates:

**Scenario:** A multi-container app with secrets

```
┌─────────────────────────────┐
│ AI Sandbox (AI here)        │
│ - Can read demo app code    │
│ - CANNOT read secrets/      │
│ - CANNOT read .env          │
└─────────────────────────────┘
         │
         │ DockMCP MCP
         ↓
┌─────────────────────────────┐
│ securenote-api container    │
│ - HAS access to secrets/    │
│ - HAS access to .env        │
│ - AI can check logs here    │
└─────────────────────────────┘
```

### Common Tasks

#### 1. User asks: "Start the demo apps"

You should respond:
```
I cannot run docker-compose from inside the AI Sandbox because I don't have
access to the Docker socket. Please run this on your host OS:

cd /path/to/workspace/demo-apps
docker-compose -f docker-compose.demo.yml up -d
```

Do NOT try to:
- Run `docker-compose` inside AI Sandbox (will fail)
- Run `docker` commands (no access)

#### 2. User asks: "Check the API logs"

You should:
```
Use DockMCP MCP tool: get_logs
Container: securenote-api
Tail: 100
```

Do NOT try to:
- Read log files directly (they're in another container)
- Access Docker socket (you don't have access)

#### 3. User asks: "Run the tests"

You should:
```
Use DockMCP MCP tool: exec_command
Container: securenote-api
Command: npm test
```

DockMCP will check if "npm test" is whitelisted, then execute it.

#### 4. User asks: "Read the .env file"

You should:
```
cat demo-apps/securenote-api/.env
```

Result: Empty or error (it's mounted as /dev/null)

Then explain:
"This file is hidden for security. The API container has access to it, but I don't.
This is intentional - it protects secrets while allowing development."

#### 5. User asks: "Why are secrets hidden from you?"

Explain:
"This project demonstrates secure AI development. Secrets (API keys, encryption keys)
are hidden from me using Docker volume mounts. This prevents accidental exposure.

However, I can still help develop because:
- I can see all application code
- I can check logs via DockMCP
- I can run tests via DockMCP
- The actual containers have full access to secrets"

### Updating This Template

When a user asks you to update to the latest version, or when you detect an update is available, follow this procedure:

#### Detecting Updates

**Method 1: Read the state file** (if available)
```bash
cat .sandbox/.state/update-check
# Format: <unix_timestamp>:<version>
# Example: 1705315800:v1.2.0
```

**Method 2: Check directly with git**
```bash
git fetch origin main
git log HEAD..origin/main --oneline
```

**Method 3: Use SandboxMCP tool**
- Use `get_update_status` to check update status

#### Update Procedure

When an update is available, follow these steps:

**1. Check what changed**
```bash
# Fetch latest changes
git fetch origin main

# Show summary of changes
git log HEAD..origin/main --oneline

# Show detailed diff
git diff HEAD..origin/main --stat
```

**2. Identify affected components**

Look for changes in these critical areas:
- `.sandbox/sandbox-mcp/` → SandboxMCP needs rebuild
- `dkmcp/` → DockMCP needs rebuild (user must do this on host OS)
- `.devcontainer/` or `cli_sandbox/` → Container config changed (restart required)
- `.sandbox/scripts/` → Scripts updated (may need re-run)
- `demo-apps/` → Demo apps updated

**3. Detect potential conflicts**

Check if the user has customized these files:
```bash
# Check for local modifications
git status

# Files commonly customized by users:
# - .devcontainer/docker-compose.yml (secret hiding config)
# - cli_sandbox/docker-compose.yml (secret hiding config)
# - dkmcp/configs/dkmcp.yaml (security policy)
# - .claude/settings.json (AI file access)
```

If conflicts are detected, **warn the user** before proceeding.

**4. Explain changes to the user**

Provide a clear summary:
```
📦 Update available: v1.1.0 → v1.2.0

Changes:
- SandboxMCP: Added update check tools
- DockMCP: Improved log masking
- Scripts: Fixed secret sync validation

Impact:
- SandboxMCP will be rebuilt automatically
- DockMCP requires manual rebuild on host OS
- No config file conflicts detected

Risks:
- Low risk: Only tool improvements, no breaking changes
```

**5. Get user confirmation**

Use `AskUserQuestion` if you need clarification on:
- How to handle conflicts (merge vs. overwrite)
- Whether to apply optional changes
- Timing (apply now vs. later)

**6. Apply the update**

```bash
# Pull changes
git pull origin main

# Rebuild SandboxMCP (if affected)
cd /workspace/.sandbox/sandbox-mcp
make clean
make register

# Inform user about DockMCP rebuild (if affected)
# User must run this on host OS:
# cd /workspace/dkmcp && make install
```

**7. Verify and complete**

```bash
# Check that SandboxMCP is rebuilt
ls -lh /workspace/.sandbox/sandbox-mcp/sandbox-mcp

# Inform user to reconnect MCP or restart VS Code
# Option A: /mcp → "Reconnect" (quick)
# Option B: Restart VS Code completely (thorough)
```

**8. Post-update checks**

After the update:
- Verify SandboxMCP tools are available
- Check if DockMCP is still accessible (if used)
- Run basic sanity checks (e.g., `git status`, `ls .sandbox/`)

#### When Updates Are Detected Automatically

The startup script (`check-upstream-updates.sh`) runs on every container start. If you notice a user just started their session and an update notification might have been shown, you can proactively offer help:

```
I noticed this project checks for updates on startup. Would you like me to
check if there are any updates available and help you apply them?
```

However, **do not** check for updates at the start of every conversation unless:
- The user explicitly asks
- You notice they're experiencing issues that might be fixed in a newer version

#### Important Notes

**What you CAN do:**
- ✅ Read state files and check for updates
- ✅ Run `git fetch` and `git diff` to analyze changes
- ✅ Pull updates with `git pull`
- ✅ Rebuild SandboxMCP inside the container
- ✅ Explain changes and risks to the user
- ✅ Build DockMCP **client** inside the container (`cd /workspace/dkmcp && make install`)

**What you CANNOT do:**
- ❌ Rebuild/restart DockMCP **server** (runs on host OS, user must do this)
- ❌ Restart the DevContainer (user must do this)
- ❌ Run `docker` or `docker-compose` commands
- ❌ Modify files without user approval if conflicts exist

**Always prioritize safety:**
- Explain risks before applying updates
- Warn about potential conflicts
- Get explicit user approval for significant changes
- Provide rollback instructions if something goes wrong

## What Runs Where

Understanding the separation between AI Sandbox and Host OS is critical:

### On Host OS (User runs these):
- ✅ `docker-compose up` - Start demo apps
- ✅ `cd dkmcp && make install` - Build and install DockMCP
- ✅ `dkmcp serve` - Run DockMCP MCP server
- ✅ Docker commands (`docker ps`, `docker logs`, etc.)

### In AI Sandbox (AI runs these):
- ✅ Read/edit source code
- ✅ Use DockMCP MCP tools (via HTTP to host)
- ✅ Install Node packages (`npm install`)
- ✅ Run linters, formatters
- ❌ CANNOT run docker/docker-compose (no socket access)

### Via DockMCP MCP (AI uses this):
- ✅ Get container logs
- ✅ Run whitelisted commands in containers
- ✅ Check container stats
- ✅ Inspect container details

## Setting Up DockMCP MCP Connection

For AI assistants to access DockMCP, the MCP server must be configured.

### Step 1: Start DockMCP on Host OS

```bash
# On Host OS (NOT in AI Sandbox)
cd dkmcp
make install  # Builds and installs to $GOPATH/bin (usually ~/go/bin)
dkmcp serve --config configs/dkmcp.example.yaml
```

**Important:** If you restart the DockMCP server, SSE connections are dropped. You (AI assistant) should inform the user to run `/mcp` → "Reconnect" in Claude Code to re-establish the connection.

### Step 2: Configure MCP in AI Sandbox

```bash
# Inside AI Sandbox
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```

This creates `~/.claude.json` with:

```json
{
  "mcpServers": {
    "dkmcp": {
      "type": "sse",
      "url": "http://host.docker.internal:8080/sse"
    }
  }
}
```

**After adding the MCP server, restart VS Code for it to connect.**

### Step 3: Verify Connection

You can verify DockMCP tools are available by checking if you can use tools like `list_containers`, `get_logs`, etc.

### Troubleshooting: DockMCP Connection Issues

If Claude Code does not recognize the DockMCP server even though it's running on the host OS:

1. **Verify DockMCP is running on host OS:**
   ```bash
   # On Host OS (NOT in AI Sandbox)
   curl http://localhost:8080/health
   # Should return 200 OK
   ```

2. **Try MCP Reconnect:**
   - Run `/mcp` in Claude Code, then select "Reconnect"

3. **Restart VS Code completely:**
   - macOS: Press `Cmd + Q` to fully quit
   - Windows/Linux: Press `Alt + F4` or use the menu
   - Reopen VS Code to re-establish the MCP connection

**If issues persist**, verify MCP configuration in AI Sandbox:

```bash
cat ~/.claude.json | jq '.mcpServers.dkmcp'
# Should display: "url": "http://host.docker.internal:8080/sse"
```

### Fallback: Using DockMCP Client Commands

If DockMCP MCP tools are not available (MCP server not recognized, connection issues, etc.), **you (AI assistant) can use the `dkmcp client` command directly** as a fallback.

**When to use this fallback:**
- DockMCP MCP tools (`list_containers`, `get_logs`, etc.) are not appearing in your tool list
- MCP connection errors persist after troubleshooting
- You need to access container logs or run commands but MCP is not working
- **MCP shows "connected" but tools fail with "Client not initialized" error**

**Important:** Even when `/mcp` shows "✔ connected", the MCP tools may fail with "Client not initialized" error. This may be caused by session management timing issues in the VS Code extension (Claude Code, Gemini Code Assist, etc.). In this case:
1. Inform the user that MCP appears connected but tools are failing
2. **Suggest the user try `/mcp` → "Reconnect" first** (this is the quickest solution)
3. If Reconnect doesn't work, use `dkmcp client` commands as a fallback
4. As a last resort, suggest the user restart VS Code completely (Cmd+Q / Alt+F4) to re-establish the MCP connection

**How to use `dkmcp client` (run these yourself via Bash):**

```bash
# List containers
dkmcp client list

# Get logs from a container
dkmcp client logs securenote-api

# Get logs with tail option
dkmcp client logs --tail 50 securenote-api

# Execute a whitelisted command
dkmcp client exec securenote-api "npm test"
```

> **Note on `--url`:** The default URL is `http://host.docker.internal:8080`. If the server port is changed in `dkmcp.yaml`, specify the URL explicitly via the `--url` flag or the `DOCKMCP_SERVER_URL` environment variable.
> ```bash
> dkmcp client list --url http://host.docker.internal:9090
> # or
> export DOCKMCP_SERVER_URL=http://host.docker.internal:9090
> ```

**If `dkmcp` command is not found in AI Sandbox:**

Tell the user:

```
The dkmcp command is not installed in this AI Sandbox. Please run the following:

cd /workspace/dkmcp
make install

This will build and install dkmcp. After installation, I can use the dkmcp client
commands to access container logs and run tests.
```

**Note:** `make install` can be run inside the AI Sandbox (Go is available). The `dkmcp client` commands connect to the DockMCP server running on the host OS via HTTP, so they work even without Docker socket access.

## Two Environment Strategy

This project provides TWO AI Sandbox environments:

1. **DevContainer** (`.devcontainer/`) - Primary
   - VS Code integration
   - Claude Code, Gemini Code Assist
   - Full featured

2. **CLI Sandbox** (`cli_sandbox/`) - Backup
   - Terminal-based (Claude Code, Gemini CLI, etc.)
   - Independent of DevContainer
   - Recovery tool

**Why both?** If DevContainer config breaks, user can still run `./cli_sandbox/claude.sh` to get AI help fixing the DevContainer.

### Detecting Current Environment

To identify which environment you are running in, check the `SANDBOX_ENV` environment variable:

```bash
echo $SANDBOX_ENV
```

| Value | Environment | Description |
|-------|-------------|-------------|
| `devcontainer` | DevContainer | VS Code integrated environment with Go |
| `cli_claude` | CLI Sandbox (Claude) | Terminal-based Claude Code environment |
| `cli_gemini` | CLI Sandbox (Gemini) | Terminal-based Gemini CLI environment |
| `cli_ai_sandbox` | CLI Sandbox (Shell) | Terminal-based general shell environment |

**At the start of a session**, run this check to understand your context:

```bash
echo "Environment: $SANDBOX_ENV"
```

## Multiple DevContainer Instances

If users need to run multiple DevContainer instances (e.g., separate client projects), they can use `COMPOSE_PROJECT_NAME` to create isolated environments.

**Important:** Different `COMPOSE_PROJECT_NAME` creates different volumes, so the home directory (credentials, settings, history) won't be shared automatically.

### Copying Home Directory Between Projects

Use the provided script to copy the home directory from one project to another:

```bash
# On Host OS - Export from workspace
./.sandbox/host-tools/copy-credentials.sh --export /path/to/workspace ~/backup

# Import to another workspace
./.sandbox/host-tools/copy-credentials.sh --import ~/backup /path/to/workspace
```

**Note:** If the target DevContainer is already running, it needs to be restarted for the changes to take effect.

For more details, see docs/reference.md → "Running Multiple DevContainers" section.

## DockMCP MCP Tools

As an AI assistant, you have access to these DockMCP tools:

| Tool | What It Does | Example Use |
|------|--------------|-------------|
| `list_containers` | List accessible containers | "Show me running containers" |
| `get_logs` | Get container logs | "Check API logs" |
| `get_stats` | Get resource stats | "Is the API using too much memory?" |
| `exec_command` | Run whitelisted command | "Run npm test in API" |
| `inspect_container` | Get detailed info | "Show me API container config" |
| `get_allowed_commands` | List whitelisted commands | "What commands can I run?" |
| `get_security_policy` | Get current security policy | "Show security settings" |
| `search_logs` | Search logs for a pattern | "Search for 'error' in logs" |
| `list_files` | List files in container directory | "List files in /app" |
| `read_file` | Read file from container | "Read /app/config.json" |
| `get_blocked_paths` | Get blocked file paths | "Which paths are blocked?" |

**Security:** All operations are checked against the security policy in `dkmcp.yaml`. Output masking automatically hides sensitive data (passwords, API keys, tokens) in logs and command output.

**Note on tool naming:** In MCP implementations, tools appear with prefixed names like `mcp__dkmcp__list_containers`. The base tool names listed above remain the same regardless of prefix.

## SandboxMCP

In addition to DockMCP (host-side), there's **SandboxMCP** that runs **inside the container** via stdio. It helps you discover and use scripts/tools in `.sandbox/`.

### Available Tools

| Tool | What It Does | Example Use |
|------|--------------|-------------|
| `list_scripts` | List scripts in .sandbox/scripts/ | "What scripts are available?" |
| `get_script_info` | Get detailed info about a script | "How do I use validate-secrets.sh?" |
| `run_script` | Execute a container script | "Run validate-secrets.sh" |
| `list_tools` | List tools in .sandbox/tools/ | "What tools are available?" |
| `get_tool_info` | Get detailed info about a tool | "How do I use search-history?" |
| `run_tool` | Execute a tool | "Search my conversation history for 'MCP'" |

### Key Features

- **Auto-registered**: Builds and registers automatically on container startup
- **Host-only rejection**: Scripts like `init-host-env.sh` that require host OS access are rejected with guidance on how to run them on the host OS
- **Script metadata**: Returns description (EN/JA), execution environment (host/container/any), category (utility/test)

### Manual Registration

If not auto-registered, you can register manually:
```bash
cd /workspace/.sandbox/sandbox-mcp
make register    # Build and register
make unregister  # Remove registration
```

### Adding Custom Tools

You can extend SandboxMCP by placing Go files in `.sandbox/tools/`. They are automatically discovered and available to AI assistants via `list_tools`, `get_tool_info`, and `run_tool`.

**Header format** (parsed from comments before `package`, stops at `// ---` separator):

```go
// Short description of the tool (first comment line = description)
//
// Usage:
//   go run .sandbox/tools/my-tool.go [options] <args>
//
// Examples:
//   go run .sandbox/tools/my-tool.go "hello"
//   go run .sandbox/tools/my-tool.go -verbose "world"
//
// --- optional localized description (not parsed by SandboxMCP) ---
//
// ツールの日本語説明（任意）
package main
```

**Parsing rules:**
- First comment line → `description`
- `Usage:` / `Examples:` sections → `usage` / `examples`
- `// ---` line → stops parsing (everything below is for human readers only)
- `package` line → stops parsing

The `// ---` separator is optional. Use it when adding localized descriptions or other notes that should not be parsed by SandboxMCP.

No registration or configuration needed — just drop a `.go` file and AI can use it.

### Adding Custom Scripts

You can also place shell scripts in `.sandbox/scripts/` and they will be automatically discovered via `list_scripts`, `get_script_info`, and `run_script`. Add the following header:

```bash
#!/bin/bash
# my-script.sh
# English description
# ---
# Japanese description (optional, for human readers only, not parsed)
```

The `# ---` separator marks the end of parsed content. Everything after it is ignored by the parser but kept for human readers (like Japanese documentation). This aligns with the Go tools' `// ---` separator pattern.

Since scripts can call other languages (Python, Node.js, etc.), you can build tools in any language, not just Go.

### Using Tools Proactively

As an AI assistant, you can **discover and run** tools in `.sandbox/tools/` via SandboxMCP:

1. **`list_tools`** → See what tools are available
2. **`get_tool_info`** → Read usage and options for a tool
3. **`run_tool`** → Execute the tool with arguments

When a user's request can be fulfilled by an existing tool (e.g., searching conversation history), **run it on their behalf** via `run_tool`. Show the equivalent `go run` command as well, so the user can run it themselves later if they want. For example, if asked "find the conversation where we discussed MCP setup", execute `run_tool` with `search-history.go` and also mention the command line equivalent.

**Cost estimation workflow:** When a user asks about usage cost or plan comparison:
1. Run `usage-report.go -json` (with appropriate period flags) via `run_tool` to get token counts
2. Use `WebFetch` to get current pricing from `https://docs.anthropic.com/en/docs/about-claude/pricing`
3. Calculate API costs from the token data and fetched prices
4. Compare with Pro ($20/mo) and Max ($100/mo, $200/mo) plan pricing
5. Present a clear cost comparison table

### Difference from DockMCP

| | sandbox-mcp | DockMCP |
|---|---|---|
| Location | Inside container | Host OS |
| Transport | stdio | SSE (HTTP) |
| Purpose | Script/tool discovery | Container access |
| Auto-start | By Claude Code | Manual (`dkmcp serve`) |

### Shell-based Help (help.sh)

For users working in shell environments (e.g., `cli_sandbox/ai_sandbox.sh`), there's also a lightweight **help.sh** script that provides a quick one-line summary of all scripts:

```bash
.sandbox/scripts/help.sh
```

**Role separation:**
- **help.sh**: Shell users get a quick one-line summary of each script
- **SandboxMCP**: AI assistants get detailed information (multi-line descriptions, usage, etc.)

This design keeps the shell output concise while providing AI with full context. For detailed information, users can read the header comments in each script directly.

## Development Approach: Test-Driven Development (TDD)

When fixing bugs or implementing features, **always follow TDD (Test-Driven Development)**:

### TDD Workflow

1. **Write test first** - Before implementing or fixing, write a test that:
   - Detects the current bug (for bug fixes)
   - Verifies the expected behavior (for new features)

2. **Verify test fails** - Run the test to confirm it fails as expected:
   - For bug fixes: test should fail, proving the bug exists
   - For new features: test should fail, proving feature is not yet implemented

3. **Implement/Fix** - Write the minimum code to make the test pass

4. **Verify test passes** - Run the test to confirm the fix/implementation works

5. **Run all tests** - Ensure no regressions: `go test ./...` (or equivalent)

### Why TDD?

- **Proves the bug exists** before fixing (avoids fixing non-issues)
- **Proves the fix works** after implementation
- **Prevents regression** in future changes
- **Documents expected behavior** through tests

### Example: Bug Fix with TDD

```
1. User reports: "dkmcp-go-client logs at INFO level instead of DEBUG"

2. Write test first:
   - Create test that sends initialize request with client_name="dkmcp-go-client"
   - Assert: NO INFO level logs should contain "dkmcp-go-client"

3. Run test → FAILS (proving the bug exists)

4. Fix the code (remove duplicate log, add proper level check)

5. Run test → PASSES (proving the fix works)

6. Run all tests → All pass (no regression)
```

### When to Apply TDD

- ✅ Bug fixes (always write test that reproduces the bug first)
- ✅ New features (write tests for expected behavior first)
- ✅ Refactoring (ensure tests exist before changing code)
- ⚠️ Exploratory changes (may write tests after understanding the problem)

### Writing Meaningful Tests

**Tests must exercise real code paths, not duplicate logic.**

❌ **Bad test** (duplicates logic instead of testing real code):
```go
// This just duplicates the condition, doesn't test actual behavior
func TestClientLogLevel(t *testing.T) {
    clientName := "dkmcp-go-client"
    if clientName == "dkmcp-go-client" {
        // This is the same logic as the code being tested!
        expected := "DEBUG"
    }
}
```

✅ **Good test** (uses real handlers, tests actual behavior):
```go
// This sends a real request through the actual handler
func TestClientLogLevel(t *testing.T) {
    server := NewServer(...)
    ts := httptest.NewServer(server.handler)
    // Send real initialize request
    // Capture logs
    // Verify actual log output
}
```

**When unsure if a test is meaningful:**
1. Ask yourself: "Does this test call the actual code being tested?"
2. If the test just duplicates the logic, it's not meaningful
3. **Ask the user for confirmation** before writing tests that seem redundant

**If you must write a seemingly simple test:**
- Explain to the user why it might appear redundant
- Get confirmation before proceeding

## Coding Conventions

### Scaffolding Logs (足場ログ)

When adding temporary debug logs during development, **always add a comment** to indicate they should be removed later:

```go
// TODO: remove after debugging - scaffolding log
// TODO: デバッグ後に削除 - 足場ログ
slog.Debug("checkpoint: data received", "size", len(data))
```

**Why this matters:**
- Without a comment, reviewers can't tell if the log was intentionally kept or forgotten
- Makes cleanup easy: `grep "TODO:.*scaffolding"` finds all temporary logs
- Prevents debug noise from accumulating in production code

**Convention:**
- Use `// TODO: remove after debugging - scaffolding log` (English)
- Or `// TODO: デバッグ後に削除 - 足場ログ` (Japanese)
- Include both if the codebase is bilingual

**When to remove:**
- Before committing (ideal)
- During code review (if missed)
- When the feature is stable and working

### Japanese Documentation

When writing Japanese documentation (comments, README, docs), **do not translate directly from English**. Write naturally in Japanese so the text reads smoothly for native speakers.

**Bad example (unnatural direct translation):**
```
// これはクライアント名とユーザーエージェントから表示名を解決します
```

**Good example (natural Japanese):**
```
// クライアントの表示名を、client_nameとUser-Agentの内容から決定します
```

**Key points:**
- Avoid unnatural word order that mirrors English grammar
- Use natural Japanese particles and sentence structure
- Technical terms (client_name, User-Agent, etc.) can remain in English
- Prioritize clarity and readability over literal accuracy

### Host OS Test Scripts

Test scripts that run on the host OS (e.g., under `dkmcp/scripts/`) can cause real side effects such as port occupation and leftover processes, unlike those inside AI Sandbox. **Display the following information before execution:**

1. **Impact** — Ports used, temporary files created, processes started, etc.
2. **Risk** — Risk level and reasoning (e.g., localhost only, brief duration)
3. **Recovery** — Commands to stop processes, delete temp files, and free ports

Also, **display concrete recovery commands in the failure summary** when tests fail.

**実装例:**
- `dkmcp/scripts/server-log-test.sh` — `show_prerun_info()` と `print_summary()` で表示
- `.sandbox/scripts/test-advanced-features.sh` — `confirm_section()` でセクションごとに表示

## Best Practices for AI Assistants

### DO:
- ✅ Use DockMCP MCP to access other containers
- ✅ Explain when secrets are hidden (don't just say "file not found")
- ✅ Read application code freely
- ✅ Suggest changes to demo apps
- ✅ Help with DockMCP development

### DON'T:
- ❌ Try to bypass secret hiding
- ❌ Suggest removing security configurations without explanation
- ❌ Attempt to access Docker socket directly
- ❌ Modify security files without user approval

### Git Operations in Demo Environment

In this demo project, secret files are force-tracked with `git add -f` to demonstrate the hiding mechanism. As a result, `git status` shows them as "deleted" inside AI Sandbox.

This is demo-specific. In real projects, secrets should be in `.gitignore`, so this issue won't occur. See [docs/hands-on.ja.md](docs/hands-on.ja.md) for details.

### Hidden Files May Appear as Missing

Inside AI Sandbox, secret files are hidden by Docker volume mounts (`/dev/null`) and `tmpfs`. As a result, files that **exist on the host OS** may appear empty or missing from the sandbox. Before concluding that a file does not exist:

1. **Check if the file path is listed in the volume/tmpfs mounts** in `.devcontainer/docker-compose.yml` or `cli_sandbox/docker-compose.yml`
2. **If a file appears empty or missing and matches a hidden path**, it is likely hidden by the sandbox — not actually absent
3. **Ask the user to verify on the host OS** (e.g., `ls -la <path>` or `cat <path>` on the host) since you cannot see the real contents from inside the sandbox

This is especially important when investigating issues related to `.env` files, `secrets/` directories, or any path configured as a hidden mount. Never report these files as "not found" without first considering whether they are sandbox-hidden.

### When User Wants to Customize

When a user wants to adapt this template for their own projects (e.g., "customize this for my project", "set up secret hiding for my app", "configure DockMCP for my containers"), follow this workflow. **Do the work yourself** — don't just list instructions for the user.

#### Step 1: Gather project information

Use `AskUserQuestion` or follow-up questions to collect:

1. **Project paths** — What directories will be in `/workspace/`?
   - Example: `my-api/`, `my-web/`
2. **Secret files** — Which files contain secrets?
   - Example: `.env`, `config/secrets.json`
3. **Secret directories** — Which directories should appear empty?
   - Example: `secrets/`, `keys/`
4. **Container names** — Docker container names for DockMCP?
   - Example: `my-api`, `my-web-*`
5. **Allowed commands** — What commands should AI be able to run per container?
   - Example: `npm test`, `npm run lint`

At minimum, you need project paths and secret files. If the user only mentions some items, ask about the rest.

#### Step 2: Remove demo content

Remove demo-specific entries from both docker-compose files:

- **`.devcontainer/docker-compose.yml`** — Remove volume mounts and tmpfs entries referencing `demo-apps/` and `demo-apps-ios/`, remove `extra_hosts` for `securenote.test`
- **`cli_sandbox/docker-compose.yml`** — Same removals

Keep the general structure (env_file, home volume, resource limits) intact.

#### Step 3: Configure secret hiding

Edit **both** docker-compose files with the user's secret paths:

```yaml
volumes:
  # Secret files → /dev/null
  - /dev/null:/workspace/my-api/.env:ro

tmpfs:
  # Secret directories → empty
  - /workspace/my-api/secrets:ro
```

**Important:** Both `.devcontainer/docker-compose.yml` and `cli_sandbox/docker-compose.yml` must have identical secret hiding configuration.

#### Step 4: Configure DockMCP

```bash
cp dkmcp/configs/dkmcp.example.yaml dkmcp.yaml
```

Edit `dkmcp.yaml`:
- Update `allowed_containers` with user's container name patterns
- Update `exec_whitelist` with allowed commands per container
- Keep security mode as `moderate` unless user requests otherwise

#### Step 5: Update AI configuration

- **`.claude/settings.json`** — Replace demo deny patterns with user's secret file patterns
- **`.aiexclude` / `.geminiignore`** — Update secret patterns for Gemini
- **`CLAUDE.md`** — Rewrite project-specific sections (Project Structure, Common Tasks examples), remove SecureNote demo references
- **`GEMINI.md`** — Same updates in shorter format

#### Step 6: Run validation

```bash
.sandbox/scripts/validate-secrets.sh
.sandbox/scripts/compare-secret-config.sh
.sandbox/scripts/check-secret-sync.sh
```

If `check-secret-sync.sh` reports issues, run `.sandbox/scripts/sync-secrets.sh` to fix.

#### Step 7: Hand off to user

Tell the user what you've done and what they need to do:

```
Configuration complete:
- Secret hiding in both docker-compose.yml files
- DockMCP configuration (dkmcp.yaml)
- AI settings (.claude/settings.json, .aiexclude)
- Updated CLAUDE.md and GEMINI.md

You need to:
1. Rebuild DevContainer (Cmd/Ctrl+Shift+P → "Dev Containers: Rebuild Container")
2. Start DockMCP on host OS:
   cd dkmcp && make install && dkmcp serve --config ../dkmcp.yaml
3. After rebuild, ask me to verify (container list, secret hiding check)
```

#### What you CAN do:
- ✅ Edit both docker-compose.yml files (secret hiding)
- ✅ Copy and edit `dkmcp.yaml`
- ✅ Edit `CLAUDE.md`, `GEMINI.md`, `.claude/settings.json`, `.aiexclude`, `.geminiignore`
- ✅ Run validation scripts

#### What you CANNOT do:
- ❌ Rebuild the DevContainer (user must do via VS Code)
- ❌ Start/restart DockMCP server (runs on host OS)
- ❌ Run `docker` or `docker-compose` commands
- ❌ Add/remove user's project files (user must clone/copy)

## Testing & Verification

### Security Test

User can verify secret hiding works:
```bash
# From inside AI Sandbox
cat demo-apps/securenote-api/secrets/jwt-secret.key
# Should be empty

# But API has access:
curl http://localhost:8080/api/demo/secrets-status
# Should show secrets are loaded
```

### DockMCP Test

User can verify DockMCP works:
```bash
# From AI Sandbox, ask AI:
"Show me logs from securenote-api"
"Run npm test in securenote-api"

# AI uses DockMCP MCP to execute these
```

## Project Goals

This project aims to show:

1. **AI can be useful without seeing secrets**
   - Logs, tests, code review all work
   - Secrets stay protected

2. **Multi-project development is easier**
   - Mobile + API + Web in one workspace
   - AI helps across all of them

3. **Security doesn't block productivity**
   - Proper isolation allows safe AI usage
   - No workflow disruption

## For Users Customizing This Template

This is a **template**. Users should:

1. Replace demo-apps with their actual projects
2. Update secret hiding paths
3. Configure DockMCP for their container names
4. Adjust security policies to their needs

The patterns shown here work for any multi-container setup.

## Summary

**What you are:** An AI assistant working inside a secure AI Sandbox

**What you can do:**
- Read code in `/workspace/`
- Use DockMCP MCP to access other containers
- Use `dkmcp client` commands as a fallback when MCP is unavailable
- Help develop across multiple projects

**What you cannot do:**
- Read files in `secrets/` directories (they're hidden)
- Read `.env` files (they're hidden)
- Access Docker socket directly

**Your mission:**
- Help users develop safely
- Demonstrate that AI + security can coexist
- Show value of cross-container development

For more details, see:
- [README.md](README.md) - User documentation
- [dkmcp/README.md](dkmcp/README.md) - DockMCP details
- [demo-apps/README.md](demo-apps/README.md) - Demo application guide

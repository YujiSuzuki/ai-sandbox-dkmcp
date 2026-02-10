# Architecture Details

Detailed diagrams explaining how AI Sandbox + DockMCP works.

[← Back to README](../README.md)

---

## Overall Architecture

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
│  │    ├─ Claude Code / Gemini                   │ │   │
│  │    ├─ SandboxMCP (stdio)                     │ │   │
│  │    └─ secrets/ → empty (hidden)              │ │   │
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
<summary>Tree format</summary>

**Data flow:** AI (AI Sandbox) → DockMCP (:8080) → Other containers

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
    │   ├── SandboxMCP (stdio)
    │   └── secrets/ → empty (hidden)
    │
    ├── API Container
    │   └── secrets/ → real files
    │
    └── Web Container
```

</details>

---

## How Secret Hiding Works

Since AI runs inside the AI Sandbox, Docker volume mounts can hide secret files.

```
Host OS
├── demo-apps/securenote-api/.env  ← actual file
│
├── AI Sandbox (AI execution environment)
│   └── AI tries to read .env
│       → Mounted to /dev/null, appears empty
│
└── API Container (runtime environment)
    └── Node.js app reads .env
        → Reads normally
```

**Result:**
- AI cannot read secret files (security ensured)
- Apps can read secret files (functionality maintained)
- AI can still check logs and run tests via DockMCP

---

## Benefits of AI Sandbox Isolation

Running AI inside the AI Sandbox also restricts access to host OS files.

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
- Cannot access other projects
- Cannot access SSH keys or credentials (`~/.ssh/`)
- No risk of accidentally modifying the host OS

---

## Security Features in Detail

### 1. Secret Hiding

Hides secrets from AI using Docker volume mounts:

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
- Actual containers can access real secrets
- Development works normally!

**Example in action:**

```bash
# From inside AI Sandbox (AI tries but fails)
$ cat demo-apps/securenote-api/secrets/jwt-secret.key
(empty)

# But ask Claude Code:
"Check if the API can access its secrets"

# Claude queries via DockMCP:
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

For container file blocking (`blocked_paths`), auto-import from Claude Code / Gemini settings, and more, see [dkmcp/README.md "Configuration Reference"](../dkmcp/README.md#configuration-reference).

**Example — Cross-container debugging:**

```bash
# Simulate a bug: Can't log in on web app

# Ask Claude Code:
"Login is failing. Can you check the API logs?"

# Claude gets logs via DockMCP:
dkmcp.get_logs("securenote-api", { tail: "50" })

# Error found in logs:
"JWT verification failed - invalid secret"

# Ask Claude Code:
"Please run the API tests to verify"

# Claude runs tests via DockMCP:
dkmcp.exec_command("securenote-api", "npm test")

# Issue identified and fixed!
```

### 3. Basic Sandbox Protection

- **Non-root user**: Runs as `node` user
- **Limited sudo**: Package managers only (apt, npm, pip)
- **Credential persistence**: Named volumes for `.claude/`, `.config/gcloud/`

> ⚠️ **Security note: npm/pip3 sudo risks**
>
> Allowing sudo for npm/pip3 can be exploited through malicious packages. Malicious postinstall scripts can execute arbitrary code with elevated privileges.
>
> **Mitigation options:**
> 1. Remove npm/pip3 from sudoers (edit `.sandbox/Dockerfile`)
> 2. Use `npm install --ignore-scripts` flag
> 3. Pre-install required packages in Dockerfile
> 4. Set `ignore-scripts=true` in `.npmrc`

### 4. Output Masking (Defense in Depth)

Even if secrets appear in logs or command output, DockMCP automatically masks them:

```
# Raw log output
DATABASE_URL=postgres://user:secret123@db:5432/app

# What AI sees (after masking)
DATABASE_URL=[MASKED]db:5432/app
```

Detects passwords, API keys, Bearer tokens, database URLs with credentials, and more by default. For configuration details, see [dkmcp/README.md "Output Masking"](../dkmcp/README.md#output-masking).

---

## Multi-Project Workspace

These security features enable safely working with multiple projects in a single workspace.

Example in this demo environment:
- **Backend API** (demo-apps/securenote-api)
- **Web Frontend** (demo-apps/securenote-web)
- **iOS App** (demo-apps-ios/)

What AI can do:
- Read all source code (investigate issues across app and server boundaries)
- Check any container's logs (via DockMCP)
- Run tests across projects
- Debug cross-container issues
- **Never touch secrets**

---

## SandboxMCP - In-Container MCP Server

In addition to DockMCP (host-side), **SandboxMCP** runs inside the container.

```
┌─────────────────────────────────────────────────────┐
│ AI Sandbox (inside container)                       │
│                                                     │
│  ┌─────────────────┐      ┌─────────────────────┐  │
│  │ Claude Code     │ ←──→ │ SandboxMCP (stdio)  │  │
│  │ Gemini CLI      │      │                     │  │
│  └─────────────────┘      │ • list_scripts      │  │
│                           │ • get_script_info   │  │
│                           │ • run_script        │  │
│  ┌─────────────────────┐  │ • list_tools        │  │
│  │ .sandbox/scripts/   │  │ • get_tool_info     │  │
│  │ • validate-secrets  │←─│ • run_tool          │  │
│  │ • sync-secrets      │  └─────────────────────┘  │
│  │ • help              │                           │
│  │ • ...               │                           │
│  └─────────────────────┘                           │
└─────────────────────────────────────────────────────┘
```

### DockMCP vs SandboxMCP

| | SandboxMCP | DockMCP |
|---|---|---|
| Location | Inside container | Host OS |
| Transport | stdio | SSE (HTTP) |
| Purpose | Script/tool discovery & execution | Cross-container access |
| Startup | Auto-started by AI CLI | Manual (`dkmcp serve`) |

### 6 MCP Tools

| Tool | Description | Example Use |
|------|-------------|-------------|
| `list_scripts` | List available scripts | "What scripts can I use?" |
| `get_script_info` | Get script details | "How do I use validate-secrets.sh?" |
| `run_script` | Execute a container script | "Run validate-secrets.sh" |
| `list_tools` | List available tools | "What tools are available?" |
| `get_tool_info` | Get tool details | "How do I use search-history?" |
| `run_tool` | Execute a tool | "Search my conversation history for 'MCP'" |

### Host-Only Script Handling

Some scripts (like `copy-credentials.sh`) require Docker socket access and cannot run inside the container.

```
When AI calls run_script("copy-credentials.sh"):

┌────────────────────────────────────────────────────────────┐
│ ❌ This script (copy-credentials.sh) must be run           │
│    on the host OS, not inside the AI Sandbox.              │
│                                                            │
│ To run it on your host machine:                            │
│   .sandbox/scripts/copy-credentials.sh                     │
│                                                            │
│ I cannot execute host-only scripts because the AI Sandbox  │
│ does not have Docker socket access.                        │
└────────────────────────────────────────────────────────────┘
```

**Result:** Clear guidance instead of a confusing error

### Auto-Registration

SandboxMCP automatically builds and registers on container startup:

- **DevContainer**: Runs in `postStartCommand`
- **CLI Sandbox**: Runs in startup script
- **Supports both Claude Code and Gemini CLI**: Registers if CLI is installed

For manual registration:

```bash
cd /workspace/.sandbox/sandbox-mcp
make register    # Build and register
make unregister  # Remove registration
```

### Adding Custom Tools

Place a Go file in `.sandbox/tools/` and SandboxMCP will automatically discover it. The file header (comments before `package`) is parsed to extract metadata. A `// ---` separator line stops parsing, so localized descriptions below it are not included:

```go
// Short description (first comment line becomes the description)
//
// Usage:
//   go run .sandbox/tools/my-tool.go [options] <args>
//
// Examples:
//   go run .sandbox/tools/my-tool.go "hello"
//   go run .sandbox/tools/my-tool.go -verbose "world"
//
// --- optional localized description (not parsed) ---
//
// ツールの日本語説明（任意）
package main
```

```
┌───────────────────────────────────────────────────┐
│ .sandbox/tools/                                   │
│  ├── search-history.go   ← built-in              │
│  └── my-tool.go          ← just drop a file here │
│                                                   │
│ SandboxMCP auto-discovers *.go files              │
│ No registration or configuration needed           │
└───────────────────────────────────────────────────┘
```

AI assistants can then use `list_tools` to find it, `get_tool_info` to read its usage, and `run_tool` to execute it.

### Adding Custom Scripts

You can also place shell scripts in `.sandbox/scripts/` and they will be automatically discovered. Since scripts can call other languages (Python, Node.js, etc.), you can build tools in any language, not just Go.

**Header format:**

```bash
#!/bin/bash
# my-script.sh
# English description (can be multi-line)
# Additional description continues here
# ---
# Japanese description (optional, not parsed)
```

- Line 1: Shebang
- Line 2: Filename
- Line 3+: English description (can span multiple lines, shown to AI in `list_scripts`)
- Line N: `# ---` separator (parsing stops here)
- Line N+1 onwards: Japanese description, etc. (for human readers, not passed to AI)

The `# ---` separator marks the end of parsed content. Everything after it is ignored by the parser but kept for human readers. This aligns with the Go tools' `// ---` separator pattern.

**Usage section (optional):**

If a `Usage:` line appears before the `# ---` separator, it will be displayed by `get_script_info`. The section ends at an empty comment line. This aligns with the Go tools pattern where Usage/Examples come before `// ---`.

```bash
#!/bin/bash
# my-script.sh
# English description
#
# Usage:
#   my-script.sh [options] <args>
#   my-script.sh --verbose "hello"
#
# ---
# 日本語の説明
```

**Skipped files:**

| Pattern | Reason |
|---|---|
| Files starting with `_` | Treated as libraries (e.g., `_startup_common.sh`) |
| `help.sh` | The help script itself is excluded from listings |
| Non-`.sh` files | Not processed |

**Automatic category classification:**

| Filename | Category |
|---|---|
| Starts with `test-` | `test` |
| All others | `utility` |

**Environment classification:**

Scripts are classified into three execution environments. Attempting to run a host-only script via `run_script` returns an error with guidance on how to run it on the host OS.

| Environment | Scripts |
|---|---|
| `host` (host only) | `copy-credentials.sh`, `init-host-env.sh` |
| `container` (container only) | `sync-secrets.sh`, `validate-secrets.sh`, `sync-compose-secrets.sh` |
| `any` (either) | All others |

```
┌───────────────────────────────────────────────────┐
│ .sandbox/scripts/                                 │
│  ├── validate-secrets.sh  ← built-in (container)  │
│  ├── test-*.sh            ← test category         │
│  ├── _startup_common.sh   ← skipped (library)     │
│  └── my-script.sh         ← just drop a file here │
│                                                   │
│ SandboxMCP auto-discovers *.sh files              │
│ No registration or configuration needed           │
└───────────────────────────────────────────────────┘
```

AI assistants can use `list_scripts` to find them, `get_script_info` to read usage, and `run_script` to execute them.

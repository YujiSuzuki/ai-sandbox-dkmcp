# DockMCP Host Access

DockMCP can extend AI's reach beyond containers to the host OS itself. Three features — **Host Tools**, **Container Lifecycle**, and **Host Commands** — let AI perform operations on the host in a controlled, auditable way.

[<- Back to README](../README.md)

---

## Table of Contents

- [Overview](#overview)
- [Host Tools](#host-tools)
- [Container Lifecycle](#container-lifecycle)
- [Host Commands](#host-commands)
- [MCP Tools Reference](#mcp-tools-reference)
- [CLI Commands Reference](#cli-commands-reference)
- [Security Considerations](#security-considerations)

---

## Overview

```
AI Sandbox (container)
  │
  │  MCP / HTTP
  ▼
DockMCP Server (host OS)
  ├── Container access     ← existing (logs, exec, stats, etc.)
  ├── Host Tools           ← NEW: run approved scripts on host
  ├── Container Lifecycle  ← NEW: start/stop/restart containers
  └── Host Commands        ← NEW: run whitelisted CLI commands on host
```

All three features are **off by default** and configured in `dkmcp.yaml` under `host_access` and `security.permissions`.

---

## Host Tools

### What It Does

AI can discover and execute scripts (`.sh`, `.go`, `.py`) placed in `.sandbox/host-tools/` on the host OS. This allows AI to perform host-side operations (like starting demo containers) through pre-written, reviewed scripts rather than raw shell commands.

### Approval Workflow

Tools proposed by AI or developers go through a two-stage process:

```
1. Place script in .sandbox/host-tools/     (staging — inside workspace)
2. Run: dkmcp tools sync                    (review & approve on host)
3. Approved copy goes to ~/.dkmcp/host-tools/<project-id>/
4. AI can now execute the approved version
```

Only the approved copy is executed. If the staging version changes, `dkmcp tools sync` detects the difference and prompts for re-approval.

### Directory Layout

```
~/.dkmcp/host-tools/
├── _common/                    # Shared across all projects
│   └── shared-tool.sh
└── <project-id>/               # Per-project approved tools
    ├── .project                # Project metadata (workspace path, etc.)
    └── demo-build.sh           # Approved tool
```

- **`_common/`** — Tools available to all projects (enabled with `common: true`)
- **`<project-id>/`** — Derived from workspace path; isolates tools per project

### Configuration

```yaml
# dkmcp.yaml
host_access:
  host_tools:
    enabled: true
    approved_dir: "~/.dkmcp/host-tools"
    staging_dirs:
      - ".sandbox/host-tools"
    common: true
    allowed_extensions: [".sh", ".go", ".py"]
    timeout: 60  # seconds
```

### Included Demo Tools

| Tool | Description |
|------|-------------|
| `demo-build.sh` | Build demo app Docker images |
| `demo-up.sh` | Start demo containers (`docker-compose up -d`) |
| `demo-down.sh` | Stop demo containers (`docker-compose down`) |
| `copy-credentials.sh` | Copy home directory between DevContainer projects |

### Writing Your Own Host Tools

Place a script in `.sandbox/host-tools/` with a description header:

```bash
#!/bin/bash
# my-tool.sh
# Short description of what this tool does
#
# Usage:
#   my-tool.sh [options] <args>
#
# Examples:
#   my-tool.sh --verbose build
```

The header is parsed by DockMCP and shown to AI via `list_host_tools` and `get_host_tool_info`.

---

## Container Lifecycle

### What It Does

AI can start, stop, and restart Docker containers using the Docker API directly. This is useful when AI needs to recover a crashed container or apply configuration changes.

### How It Works

- Uses **Docker API** directly (not `docker` CLI execution)
- Requires `lifecycle: true` in security permissions
- Respects `allowed_containers` policy — AI can only manage containers it's allowed to access
- Optional timeout parameter for graceful shutdown

### Configuration

```yaml
# dkmcp.yaml
security:
  permissions:
    lifecycle: true  # default: false
```

### Usage

AI uses MCP tools (`restart_container`, `stop_container`, `start_container`) or the CLI fallback:

```bash
# From AI Sandbox
dkmcp client restart securenote-api
dkmcp client stop securenote-api --timeout 30
dkmcp client start securenote-api
```

---

## Host Commands

### What It Does

AI can execute whitelisted CLI commands on the host OS through DockMCP. This is useful for operations like checking git status, disk usage, or system info without giving AI full shell access.

### How It Works

Commands are controlled by three layers:

1. **Whitelist** — Base command + argument patterns must match
2. **Deny list** — Overrides whitelist for specific dangerous combinations
3. **Dangerous mode** — Separate command set requiring explicit `dangerously=true` flag

### Configuration

```yaml
# dkmcp.yaml
host_access:
  host_commands:
    enabled: true

    # Whitelisted commands (base command → allowed argument patterns)
    whitelist:
      "git":
        - "status"            # exact match
        - "diff *"            # prefix match: diff HEAD, diff --stat, etc.
        - "log --oneline *"   # prefix match
      "df":
        - "-h"
      "free":
        - "-m"

    # Deny list (overrides whitelist)
    # deny:
    #   "git":
    #     - "push --force *"

    # Dangerous mode (requires dangerously=true)
    dangerously:
      enabled: false
      commands:
        "git":
          - "checkout"
          - "pull"
```

### Argument Pattern Matching

| Pattern | Matches | Does Not Match |
|---------|---------|----------------|
| `"status"` | `git status` | `git status --short` |
| `"diff *"` | `git diff`, `git diff HEAD`, `git diff --stat` | — |
| `"-h"` | `df -h` | `df -h /tmp` |

### Built-in Protections

Regardless of whitelist configuration, the following are **always blocked**:

- **Pipes** (`|`) — prevents chaining commands
- **Redirects** (`>`, `<`) — prevents file manipulation
- **Path traversal** (`..`) — prevents escaping workspace
- **Blocked paths** — file arguments checked against `blocked_paths` policy

---

## MCP Tools Reference

| MCP Tool | Description | Feature |
|----------|-------------|---------|
| `list_host_tools` | List available host tools with descriptions | Host Tools |
| `get_host_tool_info` | Get detailed usage/examples for a tool | Host Tools |
| `run_host_tool` | Execute an approved host tool | Host Tools |
| `restart_container` | Restart a container (Docker API) | Lifecycle |
| `stop_container` | Stop a running container (Docker API) | Lifecycle |
| `start_container` | Start a stopped container (Docker API) | Lifecycle |
| `exec_host_command` | Execute a whitelisted host CLI command | Host Commands |

---

## CLI Commands Reference

### Host Tools Management (run on host OS)

```bash
# Review and approve tools from staging directories
dkmcp tools sync

# Show approved tools directory and project info
dkmcp tools list
```

### Host Tools Client (run from AI Sandbox)

```bash
dkmcp client host-tools list
dkmcp client host-tools info <tool-name>
dkmcp client host-tools run <tool-name> [args...]
```

### Container Lifecycle (run from AI Sandbox)

```bash
dkmcp client restart <container> [--timeout <seconds>]
dkmcp client stop <container> [--timeout <seconds>]
dkmcp client start <container>
```

### Host Commands (run from AI Sandbox)

```bash
dkmcp client host-exec "git status"
dkmcp client host-exec --dangerously "git pull"
```

---

## Security Considerations

### Host Tools

- **Approval required** — Tools must be explicitly approved before execution. The staging directory (inside workspace) is writable by AI, but the approved directory (`~/.dkmcp/host-tools/`) is not.
- **Change detection** — SHA256 hashing detects modifications. Changed tools require re-approval.
- **Timeout** — Tool execution has a configurable timeout (default: 60s) to prevent runaway scripts.
- **Extension whitelist** — Only `.sh`, `.go`, `.py` files can be registered as tools.

### Container Lifecycle

- **Opt-in** — Disabled by default (`lifecycle: false`).
- **Container scope** — Respects `allowed_containers` policy.
- **Docker API only** — Uses the Docker API directly, not shell execution.

### Host Commands

- **Whitelist-only** — Only explicitly listed base commands + argument patterns are allowed.
- **Deny overrides** — Deny list takes precedence over whitelist.
- **No pipes/redirects** — `|`, `>`, `<` are always blocked regardless of configuration.
- **No path traversal** — `..` is always blocked.
- **Blocked paths** — File path arguments are checked against the same `blocked_paths` policy used for container file access.
- **Dangerous mode** — Sensitive commands (e.g., `git pull`, `git checkout`) can be placed in a separate `dangerously` section, requiring the caller to explicitly pass `dangerously=true`.

### General

- **Audit logging** — All host access operations are recorded when audit logging is enabled.
- **Output masking** — Sensitive data in tool/command output is masked before returning to AI.
- **Host path masking** — Host OS paths (e.g., `/Users/username/`) are masked to prevent AI from seeing the host user's identity.

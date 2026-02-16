# AI Sandbox Environment with DockMCP - Context for AI Assistants

> **Language policy:** This file must be written in English. AI assistants adding or editing content here should always use English.

This document provides essential behavioral rules for AI assistants. For detailed reference, see [docs/ai-guide.md](docs/ai-guide.md).

## Essential Rules

### Commits and Releases

- **Commits:** Always use `.sandbox/scripts/commit-msg.sh` — do NOT use `git commit -m "..."` directly. Run `get_script_info("commit-msg.sh")` for usage details.
- **Releases:** Always use `.sandbox/scripts/release.sh`. Run `get_script_info("release.sh")` for usage details.

### User Questions

Direct users to documentation — do not explain setup/troubleshooting yourself:
- Setup/installation → `README.md` or `README.ja.md`
- Troubleshooting → `docs/reference.md`
- Architecture → `docs/architecture.md`

### Security Rules

- ❌ Never bypass secret hiding
- ❌ Never modify security files without user approval
- ❌ Never access Docker socket directly
- ✅ Explain when secrets are hidden (don't just say "file not found")
- ✅ Check host tools (`list_host_tools`) before telling the user "I can't do this"

### Hidden Files

Files hidden by Docker volume mounts appear empty or missing. Before reporting "not found":
1. Check if the path is in `.devcontainer/docker-compose.yml` volume/tmpfs mounts
2. If it matches a hidden path, explain it's sandbox-hidden, not actually absent
3. Ask the user to verify on the host OS if needed

### Development Approach

- **TDD:** Always write tests first. Bug fix → reproduce bug in test first. New feature → write expected behavior test first. Run all tests after to prevent regression. See [docs/ai-guide.md](docs/ai-guide.md#tdd-workflow) for detailed steps.
- **Scaffolding logs:** Mark temporary debug logs with `// TODO: remove after debugging - scaffolding log` (or Japanese: `// TODO: デバッグ後に削除 - 足場ログ`)
- **Japanese documentation:** Write naturally in Japanese, don't translate directly from English. Prioritize clarity over literal accuracy.
- **Host OS test scripts:** Display impact/risk/recovery info before execution. See [docs/ai-guide.md](docs/ai-guide.md#host-os-test-scripts).
- **Meaningful tests:** Tests must exercise real code paths, not duplicate logic. See [docs/ai-guide.md](docs/ai-guide.md#writing-meaningful-tests).

---

## What This Project Is

A secure AI development environment demonstrating:
1. **Safe AI Usage** — AI coding assistants in isolated Docker containers
2. **Secret Protection** — Hide sensitive files from AI via volume mounts
3. **Cross-Container Access** — Interact with other containers via DockMCP
4. **Multi-Project Workspaces** — Mobile, API, Web in one workspace

**DockMCP** is an MCP server on the host OS providing controlled container access. It solves: "My API is in a separate container, how can AI help debug it?"

---

## What AI Can and Cannot Do

### Cannot Do
- ❌ Run `docker` or `docker-compose` commands (no Docker socket)
- ❌ Read files in `secrets/` directories (hidden by tmpfs)
- ❌ Read `.env` files (hidden by /dev/null mount)
- ❌ Start/stop containers directly
- ❌ Build Docker images

**These operations MUST be done on the host OS by the user** (or via DockMCP host tools if available).

### Can Do
- ✅ Read/edit source code in `/workspace/`
- ✅ Use DockMCP MCP tools to access other containers
- ✅ Use `dkmcp client` commands as fallback when MCP is unavailable
- ✅ Run DockMCP host tools (`.sandbox/host-tools/`) for Docker operations
- ✅ Install packages (`npm install`)
- ✅ Run linters, formatters

---

## Critical Files

### ⚠️ Requires User Confirmation to Modify

| File | Purpose |
|------|---------|
| `.devcontainer/docker-compose.yml` | Secret hiding configuration |
| `cli_sandbox/docker-compose.yml` | Same for CLI environment |
| `dkmcp/configs/dkmcp.example.yaml` | Container access policy |
| `.devcontainer/devcontainer.json` | VS Code DevContainer settings |

### ✅ Safe to Modify

- Demo application code (`demo-apps/`)
- DockMCP implementation (`dkmcp/internal/`)
- Documentation (`README.md`, `README.ja.md`)
- Shell scripts (with user approval)

### `.claude/settings.json`

Controls what AI can read. Auto-merged from subproject settings on startup. If manually changed, preserved (not overwritten). If you get a permission error reading a file, check if it's blocked here.

---

## Common Tasks

### 1. "Start the demo apps"

Check host tools first via `list_host_tools`. If available, use `run_host_tool` with `demo-build.sh` and `demo-up.sh`. If not available, ask the user to run on host OS, or suggest enabling host tools with `dkmcp serve --sync`.

Do NOT try to run `docker-compose` inside AI Sandbox (will fail).

### 2. "Check the API logs"

Use DockMCP MCP: `get_logs` (container: `securenote-api`, tail: 100). Do NOT try to read log files directly or access Docker socket.

### 3. "Run the tests"

Use DockMCP MCP: `exec_command` (container: `securenote-api`, command: `npm test`). DockMCP checks if the command is whitelisted.

### 4. "Read the .env file"

It will appear empty (hidden by volume mount). Explain: "This file is hidden for security. The API container has access to it, but I don't. This is intentional — it protects secrets while allowing development."

### 5. "Why are secrets hidden from you?"

Explain: Secrets are hidden via Docker volume mounts. AI can still help because it can read all application code, check logs via DockMCP, run tests via DockMCP, and the actual containers have full secret access.

### 6. Committing changes

Use `.sandbox/scripts/commit-msg.sh` to draft and commit. Run `get_script_info("commit-msg.sh")` for usage. Do NOT use `git commit -m "..."` directly.

### 7. DockMCP not connected

If DockMCP MCP tools (`mcp__dkmcp__*`) are not available, proactively check registration and offer setup:

```
.sandbox/scripts/setup-dkmcp.sh --check   # Silent check (exit: 0=ok, 1=not registered, 2=offline)
.sandbox/scripts/setup-dkmcp.sh            # Register if needed + verify connectivity
.sandbox/scripts/setup-dkmcp.sh --status   # Show detailed status
```

If `--check` returns 1 (not registered), offer to run `setup-dkmcp.sh` for the user.
If `--check` returns 2 (registered but offline), troubleshoot in this order:
1. **Check VS Code Ports panel** — stop forwarding port 8080 if listed (most common cause)
2. **Verify DockMCP is running on host**: `curl http://localhost:8080/health`
3. **Try `/mcp` → "Reconnect"** in Claude Code
4. **Restart VS Code completely** (Cmd+Q → reopen)

### 8. Creating a release

Use `.sandbox/scripts/release.sh` to generate release notes and publish. Run `get_script_info("release.sh")` for usage.

---

## DockMCP

DockMCP runs on the host OS and provides controlled container access via MCP.

### MCP Tools

| Tool | What It Does |
|------|--------------|
| `list_containers` | List accessible containers |
| `get_logs` | Get container logs |
| `get_stats` | Get resource usage stats |
| `exec_command` | Run whitelisted command |
| `inspect_container` | Get detailed container info |
| `get_allowed_commands` | List whitelisted commands |
| `get_security_policy` | Get current security policy |
| `search_logs` | Search logs for a pattern |
| `list_files` | List files in container directory |
| `read_file` | Read file from container |
| `get_blocked_paths` | Get blocked file paths |
| `restart_container` | Restart a container |
| `stop_container` | Stop a container |
| `start_container` | Start a container |
| `list_host_tools` | List available host tools |
| `get_host_tool_info` | Get host tool details |
| `run_host_tool` | Execute a host tool |
| `exec_host_command` | Execute whitelisted host command |

Tools appear with `mcp__dkmcp__` prefix. Output masking automatically hides sensitive data (passwords, API keys, tokens).

### Fallback: dkmcp client

If MCP tools are unavailable (connection issues, "Client not initialized" error), use `dkmcp client` commands via Bash. See [docs/ai-guide.md](docs/ai-guide.md#dockmcp-client-fallback) for the full command reference.

If `dkmcp` command is not found, tell the user: `cd /workspace/dkmcp && make install`

For DockMCP setup and troubleshooting, see [docs/ai-guide.md](docs/ai-guide.md#dockmcp-setup-and-troubleshooting).

---

## SandboxMCP

Runs inside the container via stdio. Provides: `list_scripts`, `get_script_info`, `run_script`, `list_tools`, `get_tool_info`, `run_tool`.

| | SandboxMCP | DockMCP |
|---|---|---|
| Location | Inside container | Host OS |
| Transport | stdio | SSE (HTTP) |
| Purpose | Script/tool discovery | Container access |
| Auto-start | By Claude Code | Manual (`dkmcp serve`) |

**Use tools proactively:** When a user's request can be fulfilled by an existing tool (e.g., searching conversation history), run it via `run_tool` and show the equivalent `go run` command.

For adding custom tools/scripts and cost estimation workflow, see [docs/ai-guide.md](docs/ai-guide.md#sandboxmcp-extensions).

---

## Project Structure

```
/workspace/
├── .sandbox/          # Infrastructure (scripts, tools, sandbox-mcp, host-tools)
├── .devcontainer/     # VS Code DevContainer (⚠️ secret hiding config)
├── cli_sandbox/       # CLI environment (backup, ⚠️ secret hiding config)
├── dkmcp/             # DockMCP MCP Server (Go)
├── demo-apps/         # Demo Application (securenote-api, securenote-web)
└── demo-apps-ios/     # iOS Application (SecureNote)
```

For full structure, see [docs/ai-guide.md](docs/ai-guide.md#project-structure-full).

### Environment Detection

```bash
echo $SANDBOX_ENV
# devcontainer | cli_claude | cli_gemini | cli_ai_sandbox
```

---

## Git Operations in Demo Environment

Secret files are force-tracked with `git add -f` for demo purposes, so `git status` shows them as "deleted" inside AI Sandbox. This is expected and demo-specific. In real projects, secrets should be in `.gitignore`.

---

## Reference

For detailed information, read the relevant file when needed:

| Topic | File |
|-------|------|
| DockMCP setup & troubleshooting | [docs/ai-guide.md → DockMCP Setup](docs/ai-guide.md#dockmcp-setup-and-troubleshooting) |
| DockMCP client command reference | [docs/ai-guide.md → Client Fallback](docs/ai-guide.md#dockmcp-client-fallback) |
| Template update procedure | [docs/ai-guide.md → Updating](docs/ai-guide.md#updating-this-template) |
| Template customization workflow | [docs/ai-guide.md → Customization](docs/ai-guide.md#customization-workflow) |
| SandboxMCP extensions | [docs/ai-guide.md → SandboxMCP](docs/ai-guide.md#sandboxmcp-extensions) |
| Writing meaningful tests | [docs/ai-guide.md → Tests](docs/ai-guide.md#writing-meaningful-tests) |
| Host OS test script conventions | [docs/ai-guide.md → Host Scripts](docs/ai-guide.md#host-os-test-scripts) |
| Two environment strategy | [docs/reference.md](docs/reference.md) |
| Security architecture details | [docs/architecture.md](docs/architecture.md) |
| Project customization guide | [docs/customization.md](docs/customization.md) |
| Full project structure | [docs/ai-guide.md → Structure](docs/ai-guide.md#project-structure-full) |

---

## Summary

**What you are:** An AI assistant inside a secure AI Sandbox

**Project goals:**
1. AI can be useful without seeing secrets (logs, tests, code review all work)
2. Multi-project development is easier (Mobile + API + Web in one workspace)
3. Security doesn't block productivity (proper isolation, no workflow disruption)

**Your mission:**
- Help users develop safely
- Use DockMCP for cross-container access
- Protect secrets (explain when hidden, never bypass)
- Follow project conventions (commit-msg.sh, TDD, etc.)

For more details, see:
- [README.md](README.md) — User documentation
- [dkmcp/README.md](dkmcp/README.md) — DockMCP details
- [docs/ai-guide.md](docs/ai-guide.md) — AI reference guide

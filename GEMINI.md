# AI Sandbox Environment with DockMCP - Context for Gemini Code Assist

This document provides essential behavioral rules for Gemini Code Assist. For detailed reference, see [docs/ai-guide.md](docs/ai-guide.md).

## Essential Rules

### Security Rules

- Never bypass secret hiding
- Never modify security files without user approval
- Never access Docker socket directly
- Explain when secrets are hidden (don't just say "file not found")
- Check host tools (`list_host_tools`) before telling the user "I can't do this"

### Hidden Files

Files hidden by Docker volume mounts appear empty or missing. Before reporting "not found":
1. Check if the path is in `.devcontainer/docker-compose.yml` volume/tmpfs mounts
2. If it matches a hidden path, explain it's sandbox-hidden, not actually absent
3. Ask the user to verify on the host OS if needed

### User Questions

Direct users to documentation:
- Setup/installation → `README.md` or `README.ja.md`
- Troubleshooting → `docs/reference.md`
- Architecture → `docs/architecture.md`

### Commits and Releases

- **Commits:** Always use `commit-msg.sh` to draft commit messages collaboratively with the user:
  ```
  .sandbox/scripts/commit-msg.sh
  ```
  Do NOT use `git commit -m "..."` directly — use the script so the user can review and adjust the message.

- **Releases:** Use `release.sh` to generate release notes:
  ```
  .sandbox/scripts/release.sh v0.5.0          # Generate draft
  .sandbox/scripts/release.sh --prev           # Check previous release tone
  .sandbox/scripts/release.sh v0.5.0 --notes-file ReleaseNotes-draft.md  # Publish
  ```

### Development Approach

- **TDD:** Always write tests first. Bug fix → reproduce bug in test first. New feature → write expected behavior test first.
- **Meaningful tests:** Tests must exercise real code paths, not duplicate logic. If unsure, ask the user first.

---

## What This Project Is

A secure AI development environment demonstrating:
1. **Safe AI Usage** — AI coding assistants in isolated Docker containers
2. **Secret Protection** — Hide sensitive files from AI via volume mounts
3. **Cross-Container Access** — Interact with other containers via DockMCP
4. **Multi-Project Workspaces** — Mobile, API, Web in one workspace

---

## What AI Can and Cannot Do

### Cannot Do
- Run `docker` or `docker-compose` commands (no Docker socket)
- Read files in `secrets/` directories (hidden by tmpfs)
- Read `.env` files (hidden by /dev/null mount)
- Start/stop containers directly

**These operations MUST be done on the host OS by the user** (or via DockMCP host tools if available).

### Can Do
- Read/edit source code in `/workspace/`
- Use DockMCP MCP tools to access other containers
- Use `dkmcp client` commands as fallback when MCP is unavailable
- Install packages (`npm install`)
- Run linters, formatters

---

## Critical Files

| File | Purpose |
|------|---------|
| `.devcontainer/docker-compose.yml` | Secret hiding configuration (requires user approval to modify) |
| `cli_sandbox/docker-compose.yml` | Same for CLI environment |
| `dkmcp/configs/dkmcp.example.yaml` | Container access policy |
| `.devcontainer/devcontainer.json` | VS Code DevContainer settings |

---

## Common Tasks

### "Start the demo apps"
Check host tools first via `list_host_tools`. If available, use `run_host_tool` with `demo-build.sh` and `demo-up.sh`. If not, ask user to run on host OS.

### "Check the API logs"
Use DockMCP MCP: `get_logs` (container: `securenote-api`, tail: 100).

### "Run the tests"
Use DockMCP MCP: `exec_command` (container: `securenote-api`, command: `npm test`).

### "Read the .env file"
It will appear empty (hidden by volume mount). Explain: "This file is hidden for security. The API container has access to it, but I don't."

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
| `search_logs` | Search logs for a pattern |
| `list_host_tools` | List available host tools |
| `run_host_tool` | Execute a host tool |

### Fallback: dkmcp client

If MCP tools are unavailable, use `dkmcp client` commands via Bash. See [docs/ai-guide.md](docs/ai-guide.md#dockmcp-client-fallback) for the full command reference.

If `dkmcp` command is not found, tell the user: `cd /workspace/dkmcp && make install`

For DockMCP setup and troubleshooting, see [docs/ai-guide.md](docs/ai-guide.md#dockmcp-setup-and-troubleshooting).

---

## Project Structure

```
/workspace/
├── .sandbox/          # Infrastructure (scripts, tools, sandbox-mcp, host-tools)
├── .devcontainer/     # VS Code DevContainer (secret hiding config)
├── cli_sandbox/       # CLI environment (backup)
├── dkmcp/             # DockMCP MCP Server (Go)
├── demo-apps/         # Demo Application (securenote-api, securenote-web)
└── demo-apps-ios/     # iOS Application (SecureNote)
```

For full structure, see [docs/ai-guide.md](docs/ai-guide.md#project-structure-full).

---

## Reference

| Topic | File |
|-------|------|
| DockMCP setup & troubleshooting | [docs/ai-guide.md → DockMCP Setup](docs/ai-guide.md#dockmcp-setup-and-troubleshooting) |
| DockMCP client command reference | [docs/ai-guide.md → Client Fallback](docs/ai-guide.md#dockmcp-client-fallback) |
| Template update procedure | [docs/ai-guide.md → Updating](docs/ai-guide.md#updating-this-template) |
| Template customization workflow | [docs/ai-guide.md → Customization](docs/ai-guide.md#customization-workflow) |
| Writing meaningful tests | [docs/ai-guide.md → Tests](docs/ai-guide.md#writing-meaningful-tests) |
| Security architecture details | [docs/architecture.md](docs/architecture.md) |
| Project customization guide | [docs/customization.md](docs/customization.md) |

---

## Summary

**What you are:** An AI assistant inside a secure AI Sandbox

**Your mission:**
- Help users develop safely
- Use DockMCP for cross-container access
- Protect secrets (explain when hidden, never bypass)

For more details, see:
- [README.md](README.md) — User documentation
- [dkmcp/README.md](dkmcp/README.md) — DockMCP details
- [docs/ai-guide.md](docs/ai-guide.md) — AI reference guide

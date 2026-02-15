# AI Sandbox Environment - GitHub Copilot Instructions

This document provides essential behavioral rules for GitHub Copilot. For detailed reference, see [../docs/ai-guide.md](../docs/ai-guide.md).

## Security Constraints

### Hidden Files
The following files/directories appear empty due to security measures (Docker volume mounts):
- `demo-apps/securenote-api/.env`
- `demo-apps/securenote-api/secrets/`

This is intentional. The API containers have access to real secrets, but AI assistants don't.

**Important:** If a file appears empty or missing, check whether its path is listed in the volume/tmpfs mounts in `.devcontainer/docker-compose.yml` or `cli_sandbox/docker-compose.yml`. If so, the file is sandbox-hidden. Ask the user to verify on the host OS.

### No Docker Access
You cannot run `docker` or `docker-compose` commands. Tell users to run these on the host OS.

## What AI Can and Cannot Do

### Cannot Do
- Run `docker` or `docker-compose` commands (no Docker socket)
- Read files in `secrets/` directories (hidden by tmpfs)
- Read `.env` files (hidden by /dev/null mount)

### Can Do
- Read/edit source code in `/workspace/`
- Use DockMCP MCP tools to access other containers
- Use `dkmcp client` commands as fallback when MCP is unavailable

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

## Cross-Container Access (DockMCP)

Use DockMCP MCP tools: `list_containers`, `get_logs`, `exec_command`, `inspect_container`, `search_logs`, `list_host_tools`, `run_host_tool`.

### Fallback: dkmcp client

If MCP tools are unavailable, use `dkmcp client` commands via Bash. See [../docs/ai-guide.md](../docs/ai-guide.md#dockmcp-client-fallback) for the full command reference.

If `dkmcp` not found, tell user: `cd /workspace/dkmcp && make install`

For troubleshooting, see [../docs/ai-guide.md](../docs/ai-guide.md#dockmcp-setup-and-troubleshooting).

## Critical Files

- `.devcontainer/docker-compose.yml` — Secret hiding config (requires user approval to modify)
- `cli_sandbox/docker-compose.yml` — CLI secret hiding (must match above)
- `dkmcp/configs/dkmcp.example.yaml` — Container access policy

## Development Approach: TDD

1. **Write test first** — Before implementing, write a test that detects the bug or verifies expected behavior
2. **Verify test fails** — Confirm the test fails (proves the bug exists)
3. **Implement/Fix** — Write the code to make the test pass
4. **Verify test passes** — Confirm the fix works
5. **Run all tests** — Ensure no regressions

Tests must call actual code, not duplicate logic. If unsure whether a test is meaningful, ask the user first.

## Commits and Releases

- **Commits:** Always use `commit-msg.sh` to draft commit messages collaboratively with the user:
  ```
  .sandbox/scripts/commit-msg.sh              # Generate draft
  .sandbox/scripts/commit-msg.sh --log        # Check previous commit style
  # Refine CommitMsg-draft.md together
  .sandbox/scripts/commit-msg.sh --msg-file CommitMsg-draft.md  # Commit
  ```
  Do NOT use `git commit -m "..."` directly — use the script so the user can review and adjust the message.

- **Releases:** Use `release.sh` to generate release notes:
  ```
  .sandbox/scripts/release.sh v0.5.0          # Generate draft
  .sandbox/scripts/release.sh --prev           # Check previous release tone
  .sandbox/scripts/release.sh v0.5.0 --notes-file ReleaseNotes-draft.md  # Publish
  ```

## Guidelines

1. Never suggest bypassing security configurations
2. Explain when files appear empty due to security
3. Guide users to run Docker commands on host OS
4. Use DockMCP tools for cross-container operations
5. Follow existing code patterns in the project

## Reference

For detailed information, see [../docs/ai-guide.md](../docs/ai-guide.md).

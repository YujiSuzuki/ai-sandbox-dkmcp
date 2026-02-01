# AI Sandbox Environment - GitHub Copilot Instructions

## Project Context

This is an AI sandbox environment with DockMCP for safe, multi-project AI development. You are running inside an AI Sandbox with security restrictions.

## Security Constraints

### Hidden Files
The following files/directories appear empty due to security measures:
- `demo-apps/securenote-api/.env`
- `demo-apps/securenote-api/secrets/`

This is intentional. The API containers have access to real secrets, but AI assistants don't.

**Important:** If a file appears empty or missing, check whether its path is listed in the volume/tmpfs mounts in `.devcontainer/docker-compose.yml` or `cli_sandbox/docker-compose.yml`. If so, the file is sandbox-hidden and exists on the host OS. Ask the user to verify on the host side.

### No Docker Access
You cannot run `docker` or `docker-compose` commands. Tell users to run these on the host OS.

## Project Structure

```
/workspace/
├── .devcontainer/      # DevContainer config
├── cli_sandbox/         # CLI backup environment
├── dkmcp/            # MCP Server (Go)
├── demo-apps/          # API + Web demo
└── demo-apps-ios/      # iOS demo app
```

## Cross-Container Access

Use DockMCP MCP tools:
- `list_containers` - List containers
- `get_logs` - Get logs
- `exec_command` - Run whitelisted commands

### Troubleshooting: DockMCP Connection Issues

If MCP tools are not available:

1. Verify DockMCP is running: `curl http://localhost:8080/health` (on Host OS)
2. Try MCP Reconnect: Run `/mcp`, then select "Reconnect"
3. Restart VS Code completely (Cmd+Q / Alt+F4)

**Note:** If the DockMCP server was restarted, SSE connections are dropped. Inform the user to run `/mcp` → "Reconnect" to re-establish the connection.

### Fallback: DockMCP Client Commands

If MCP tools are unavailable, use `dkmcp client` directly:

```bash
dkmcp client list --url http://host.docker.internal:8080
dkmcp client logs --url http://host.docker.internal:8080 securenote-api
dkmcp client exec --url http://host.docker.internal:8080 securenote-api "npm test"
```

If `dkmcp` not found, tell user to run: `cd /workspace/dkmcp && make install`

## Code Conventions

### Node.js (demo-apps)
- ES6+ syntax
- Express.js for API
- React + Vite for frontend
- Jest for testing

### Go (dkmcp)
- Standard Go project layout
- Cobra for CLI
- MCP implementation

### Swift (demo-apps-ios)
- SwiftUI
- @Observable pattern
- Async/await

## Important Files

- `.devcontainer/docker-compose.yml` - Secret hiding config
- `cli_sandbox/docker-compose.yml` - CLI secret hiding
- `dkmcp/configs/dkmcp.example.yaml` - Container access policy

## Development Approach: TDD

When fixing bugs or implementing features, follow TDD (Test-Driven Development):

1. **Write test first** - Before implementing, write a test that detects the bug or verifies expected behavior
2. **Verify test fails** - Confirm the test fails (proves the bug exists)
3. **Implement/Fix** - Write the code to make the test pass
4. **Verify test passes** - Confirm the fix works
5. **Run all tests** - Ensure no regressions

**Writing meaningful tests:** Tests must call actual code, not duplicate logic. If unsure whether a test is meaningful, ask the user first.

## Guidelines

1. Never suggest bypassing security configurations
2. Explain when files appear empty due to security
3. Guide users to run Docker commands on host OS
4. Use DockMCP tools for cross-container operations
5. Follow existing code patterns in the project

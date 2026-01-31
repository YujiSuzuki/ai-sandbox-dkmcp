# AI Sandbox Environment with DockMCP - Context for Gemini Code Assist

This document provides essential context for Gemini Code Assist working with this project.

## What This Project Is

This is a **comprehensive AI development environment** that demonstrates:

1. **Safe AI Usage** - Run AI coding assistants in isolated Docker containers
2. **Secret Protection** - Hide sensitive files from AI while maintaining full functionality
3. **Cross-Container Access** - AI can interact with other containers via DockMCP MCP server
4. **Multi-Project Workspaces** - Work on multiple related projects (mobile, API, web) simultaneously

## Key Innovation: DockMCP

**DockMCP** is an MCP (Model Context Protocol) server that runs on the host OS and provides controlled access to Docker containers. This allows AI assistants inside the AI Sandbox to:

- Check logs from other containers
- Run tests in other containers
- Inspect container stats
- Cannot access secrets (they're hidden via volume mounts)

## Project Structure

```
/workspace/
├── .sandbox/               # Shared sandbox infrastructure
│   ├── Dockerfile          # Node.js base with limited sudo
│   └── scripts/            # Shared scripts (validate-secrets, check-secret-sync)
│
├── .devcontainer/          # VS Code Dev Container (AI environment)
│   ├── docker-compose.yml  # Secret hiding configuration
│   └── devcontainer.json   # VS Code DevContainer settings
│
├── cli_sandbox/            # CLI environment (backup)
│   ├── claude.sh           # Run Claude Code from terminal
│   ├── gemini.sh           # Run Gemini CLI from terminal
│   ├── ai_sandbox.sh       # Enter shell
│   └── docker-compose.yml  # Secret hiding configuration
│
├── dkmcp/               # MCP Server (Go)
│   ├── cmd/dkmcp/       # Main entry point
│   ├── internal/          # Core implementation
│   └── configs/           # Example configurations
│
├── demo-apps/              # Demo Application (Server-side)
│   ├── securenote-api/     # Node.js API with secrets
│   ├── securenote-web/     # React frontend
│   └── docker-compose.demo.yml
│
└── demo-apps-ios/          # iOS Application
    └── SecureNote/         # SwiftUI source code
```

## Security Architecture

### 1. Secret Hiding (Volume Mounts)

Secrets are hidden from AI using Docker volume mounts:

```yaml
# In .devcontainer/docker-compose.yml and cli_sandbox/docker-compose.yml
volumes:
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  - /workspace/demo-apps/securenote-api/secrets:ro
```

**Result:**
- AI sees empty files/directories
- Real containers (demo-apps) access actual secrets
- Functionality is preserved

### 2. Controlled Container Access (DockMCP)

DockMCP enforces security policies:

```yaml
security:
  mode: "moderate"  # strict | moderate | permissive
  allowed_containers:
    - "securenote-*"
  exec_whitelist:
    "securenote-api":
      - "npm test"
      - "npm run lint"
```

### 3. Sandbox Protection

- **Non-root user**: Runs as `node` user
- **Limited sudo**: Only `apt`, `npm`, `pip3` allowed
- **No Docker socket**: AI cannot access `/var/run/docker.sock`

## What AI Can and Cannot Do

### CAN Do:
- Read/edit source code in `/workspace/`
- Use DockMCP MCP tools to access other containers
- Install Node packages (`npm install`)
- Run linters, formatters

### CANNOT Do:
- Run `docker` or `docker-compose` commands (no Docker socket access)
- Read files in `secrets/` directories (they're hidden)
- Read `.env` files (they're hidden)
- Access Docker socket directly

## Important Files

### Security Configuration Files

1. **`.devcontainer/docker-compose.yml`** - Defines which secrets are hidden from AI
2. **`cli_sandbox/docker-compose.yml`** - Same as above for CLI environment
3. **`dkmcp/configs/dkmcp.example.yaml`** - Defines which containers AI can access
4. **`.devcontainer/devcontainer.json`** - VS Code DevContainer settings (extensions, port control)

## Common Tasks

### When user asks to start demo apps:
Tell them to run on the host OS:
```bash
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d
```
You cannot run docker-compose inside DevContainer.

### When user asks to check API logs:
Use DockMCP MCP tool: `get_logs` with container `securenote-api`

### When user asks to run tests:
Use DockMCP MCP tool: `exec_command` with container `securenote-api` and command `npm test`

### When user asks to read .env file:
The file will appear empty. Explain that secrets are hidden for security.

## DockMCP MCP Tools

| Tool | Description |
|------|-------------|
| `list_containers` | List accessible containers |
| `get_logs` | Get container logs |
| `get_stats` | Get resource stats |
| `exec_command` | Run whitelisted command |
| `inspect_container` | Get detailed info |

## Troubleshooting: DockMCP Connection Issues

If DockMCP MCP tools are not available:

1. **Verify DockMCP is running on host OS:** `curl http://localhost:8080/health`
2. **Try MCP Reconnect:** Run `/mcp` in Gemini, then select "Reconnect"
3. **Restart VS Code completely** (Cmd+Q / Alt+F4)

**Note:** If the DockMCP server was restarted, SSE connections are dropped. Inform the user to run `/mcp` → "Reconnect" to re-establish the connection.

### Fallback: Using DockMCP Client Commands

If MCP tools are not available, **you can use `dkmcp client` commands directly** via Bash:

```bash
# List containers
dkmcp client list --url http://host.docker.internal:8080

# Get logs from a container
dkmcp client logs --url http://host.docker.internal:8080 securenote-api

# Get logs with tail option
dkmcp client logs --url http://host.docker.internal:8080 --tail 50 securenote-api

# Execute a whitelisted command
dkmcp client exec --url http://host.docker.internal:8080 securenote-api "npm test"
```

**If `dkmcp` command is not found:**

Tell the user:
```
The dkmcp command is not installed in this DevContainer. Please run:

cd /workspace/dkmcp
make install

After installation, I can use dkmcp client commands to access container logs and run tests.
```

## Development Approach: Test-Driven Development (TDD)

When fixing bugs or implementing features, **always follow TDD**:

### TDD Workflow

1. **Write test first** - Before implementing or fixing, write a test that detects the bug or verifies expected behavior
2. **Verify test fails** - Run the test to confirm it fails (proves the bug exists or feature is missing)
3. **Implement/Fix** - Write the minimum code to make the test pass
4. **Verify test passes** - Confirm the fix/implementation works
5. **Run all tests** - Ensure no regressions

### Why TDD?

- Proves the bug exists before fixing
- Proves the fix works after implementation
- Prevents regression in future changes
- Documents expected behavior through tests

### When to Apply

- Bug fixes: Always write test that reproduces the bug first
- New features: Write tests for expected behavior first
- Refactoring: Ensure tests exist before changing code

### Writing Meaningful Tests

- Tests must exercise real code paths, not duplicate logic
- Bad: Test that copies the same if-condition as the code
- Good: Test that sends real requests through actual handlers
- **If unsure whether a test is meaningful, ask the user first**

## Best Practices

### DO:
- Use DockMCP MCP to access other containers
- Explain when secrets are hidden (don't just say "file not found")
- Read application code freely
- Suggest changes to demo apps

### DON'T:
- Try to bypass secret hiding
- Suggest removing security configurations without explanation
- Attempt to access Docker socket directly
- Modify security files without user approval

## For More Details

- [README.md](README.md) - User documentation
- [dkmcp/README.md](dkmcp/README.md) - DockMCP details
- [demo-apps/README.md](demo-apps/README.md) - Demo application guide

# AI Sandbox Environment + DockMCP

[日本語の README はこちら](README.ja.md)


A secure development environment template for AI coding assistants.

- **Hide secrets from AI** — `.env` files and private keys are invisible to AI, while your apps work normally
- **Work across multiple projects** — Let AI see mobile, API, and web codebases in a single environment
- **Access other containers** — With DockMCP, AI can check logs and run tests in other containers

All you need is **Docker** and **VS Code**. [CLI-only usage is also supported](docs/reference.md#two-environments).

This project is designed for local development environments and is not intended for production use. See "[Limitations](#limitations)" and "[FAQ](#faq)" for details.

> [!NOTE]
> **Using DockMCP standalone is not recommended.** When running AI on the host OS, it already has the same permissions as the user, so there is no benefit to routing through DockMCP. For standalone setup, see [dkmcp/README.md](dkmcp/README.md).


---

# Table of Contents

- [Problems This Solves](#problems-this-solves)
- [Use Cases](#use-cases)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Project Structure](#project-structure)
- [Security Features](#security-features)
- [Supported AI Tools](#supported-ai-tools)
- [FAQ](#faq)
- [Documentation](#documentation)



# Problems This Solves

**Secret protection** — Running AI on the host OS makes it hard to prevent access to `.env` files and private keys. This environment isolates AI in a Docker container, creating a state where **code is visible but secret files are not**.

**Cross-project development** — Investigating issues at the boundary between apps and servers is hard work. This environment combines multiple projects into a single workspace so AI can see the entire system.

**Cross-container access** — Sandboxing prevents access to other containers, but DockMCP solves this. AI can read API container logs and run tests.

## Limitations

**Network restrictions** — If you want to restrict AI's external access, we recommend using Anthropic's official firewall script. See [Network Restrictions Guide](docs/network-firewall.md) for details.


# Use Cases

### Microservice Development
```
workspace/
├── mobile-app/     ← Flutter/React Native
├── api-gateway/    ← Node.js
├── auth-service/   ← Go
└── db-admin/       ← Python
```
AI supports all services without exposing API keys.

### Full-Stack Project
```
workspace/
├── frontend/       ← React
├── backend/        ← Django
└── workers/        ← Celery tasks
```
AI edits frontend code while checking backend logs.

### Legacy + New
```
workspace/
├── legacy-php/     ← Old codebase
└── new-service/    ← Modern rewrite
```
AI understands both and assists with migration.

---

# Quick Start

## Prerequisites

| Setup | Requirements |
|-------|-------------|
| **Sandbox (VS Code)** | Docker + VS Code + [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) |
| **Sandbox (CLI only)** | Docker only |
| **Sandbox + DockMCP** | Either of the above + [DockMCP](https://github.com/YujiSuzuki/ai-sandbox-dkmcp/releases) (or build from source) + MCP-compatible AI CLI |

## How It Works (Overview)

```
AI Sandbox (container)  →  DockMCP (host OS)     →  Other containers (API, DB, etc.)
   AI runs here              Relays access            Log checking, test execution
   Secrets are invisible     Enforces security policy
```

Since AI runs inside a Docker container Sandbox, secret files become completely inaccessible — as if they don't exist. This doesn't hinder development, because AI can still check logs and run tests in other containers through DockMCP.

→ For detailed architecture diagrams, see [Architecture Details](docs/architecture.md)

> **💡 To use Japanese locale:** Before opening DevContainer (or cli_sandbox), run on the host OS:
> ```bash
> .sandbox/scripts/init-host-env.sh -i
> ```
> Select `2) 日本語` to switch terminal output to Japanese.
> (Can also be run from inside the container)


## Option A: Sandbox

If you only need secret hiding (no DockMCP):

```bash
# 1. Open in VS Code
code .

# 2. Reopen in Container (Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container")
```

<details>
<summary>If <code>code</code> command is not found</summary>

**Open from VS Code menu:**
Select "File → Open Folder" and choose this folder.

**Install `code` command (macOS):**
Open the Command Palette (Cmd+Shift+P) and run `Shell Command: Install 'code' command in PATH`. Restart your terminal.

> Reference: [Visual Studio Code on macOS - Official docs](https://code.visualstudio.com/docs/setup/mac)

</details>

<details>
<summary>CLI Sandbox (terminal-based)</summary>

```bash
   ./cli_sandbox/claude.sh # (Claude Code)
   ./cli_sandbox/gemini.sh # (Gemini CLI)
```

</details>

**That's it!** AI can access code in `/workspace`, but `.env` and `secrets/` directories are hidden.


## Option B: Sandbox + DockMCP

If you also want AI to check logs and run tests in other containers:

### Step 1: Start DockMCP server (on host OS)

```bash
cd dkmcp
make install        # Installs to ~/go/bin/
dkmcp serve --config configs/dkmcp.example.yaml
```

> For Go setup, see [Go official site](https://go.dev/dl/). Use `make install`, not `make build`.

<details>
<summary>If you don't have Go on the host OS</summary>

The AI Sandbox includes a Go environment, so you can cross-build binaries for the host OS.

```bash
# Build inside AI Sandbox
cd /workspace/dkmcp
make build-host

# Install on host OS
cd <path-to-this-repo>/dkmcp
make install-host DEST=~/go/bin        # If Go is installed
make install-host DEST=/usr/local/bin  # If Go is not installed
```

</details>

### Step 2: Open DevContainer

```bash
code .
# Cmd+Shift+P / F1 → "Dev Containers: Reopen in Container"
```

### Step 3: Connect Claude Code to DockMCP

In the AI Sandbox shell:

```bash
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse
```

In Claude Code, run `/mcp` → "Reconnect".

> **Important:** If you restart the DockMCP server, `/mcp` → "Reconnect" is required again.

### Step 4 (Recommended): Custom domain setup

```bash
# macOS/Linux — run on host OS
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts
```

> AI Sandbox automatically resolves custom domains via `extra_hosts` in `docker-compose.yml`.

### Step 5 (Optional): Try the demo apps

```bash
# On host OS
cd demo-apps
docker-compose -f docker-compose.demo.yml up -d --build
```

**Access:**
- Web: http://securenote.test:8000
- API: http://api.securenote.test:8000/api/health

**Try asking AI:**
- `Show me the logs for securenote-api`
- `Run npm test in securenote-api`
- `Check if there are any secret files`

→ If connection fails, see [Troubleshooting](docs/reference.md#troubleshooting)

## Next Steps

- **Want to experience the security features?** → [Hands-on Guide](docs/hands-on.md)
- **Ready to use with your own project?** → [Customization Guide](docs/customization.md)

---


# Commands

| Command | Where to Run | Description |
|---------|-------------|-------------|
| `dkmcp serve` | Host OS | Start DockMCP server |
| `dkmcp list` | Host OS | List accessible containers |
| `dkmcp client list` | AI Sandbox | List containers via HTTP |
| `dkmcp client logs <container>` | AI Sandbox | Get logs via HTTP |
| `dkmcp client exec <container> "cmd"` | AI Sandbox | Execute command via HTTP |

> For detailed command options, see [dkmcp/README.md](dkmcp/README.md#cli-commands)



# Project Structure

`.sandbox/` contains shared infrastructure, `.devcontainer/` and `cli_sandbox/` provide two Sandbox environments, `dkmcp/` is the MCP server, and `demo-apps/` and `demo-apps-ios/` are demo applications.

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
│   └── devcontainer.json   # VS Code integration (extensions, port control, etc.)
│
├── cli_sandbox/             # CLI Sandbox (alternative environment)
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
├── demo-apps/              # Server-side project
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

When using this for your own project, delete the demo apps demo-apps/ and demo-apps-ios/. See [Customization Guide](docs/customization.md) for details.


# Security Features

| Feature | What It Does |
|---------|-------------|
| **Secret hiding** | Hides `.env` and `secrets/` from AI via Docker mounts. Apps can read them normally |
| **Container access control** | DockMCP restricts AI's access scope based on security policies |
| **Sandbox protection** | Non-root user, limited sudo, no access to host OS files |
| **Output masking** | DockMCP automatically masks passwords and API keys in logs |

→ For details and configuration, see [Architecture Details](docs/architecture.md)

> [!NOTE]
> **About git status in the demo environment:** This template force-tracks demo secret files with `git add -f`, so they appear as "deleted" in git status inside the AI Sandbox. This won't happen in your own project since you'll add secret files to `.gitignore`. See [Hands-on Guide](docs/hands-on.md) for workarounds.


# Supported AI Tools

- ✅ **Claude Code** (Anthropic) - Full MCP support
- ✅ **Gemini Code Assist** (Google) - MCP support in Agent mode (configure MCP in `.gemini/settings.json`)
- ✅ **Gemini CLI** (Google) - Terminal-based (MCP/IDE integration status unknown — check official docs)
- ✅ **Cline** (VS Code extension) - MCP integration (likely supported, unverified)



# FAQ

**Q: Why can't I ask AI to run `docker-compose up/down`?**
A: This is by design. AI handles "observation and suggestions" while humans handle "infrastructure operations". See [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) for details.

**Q: Do I need to use DockMCP?**
A: No. It works as a regular sandbox without DockMCP. DockMCP enables cross-container access.

**Q: Is it safe for production use?**
A: **No, not recommended.** DockMCP has no authentication, so it's designed for local development only.

**Q: Can I use a different secret management solution?**
A: Yes! It can be combined with other secret management methods.

**Q: Does it work on Windows?**
A: It should work with Docker Desktop, but only macOS has been tested. Linux/Windows are unverified.



# Documentation

| Document | Description |
|----------|-------------|
| [Hands-on Guide](docs/hands-on.md) | Hands-on exercises for security features |
| [Customization Guide](docs/customization.md) | How to adapt this template to your project |
| [Reference](docs/reference.md) | Environment settings, options, troubleshooting |
| [Architecture Details](docs/architecture.md) | Security mechanisms and architecture diagrams |
| [Network Restrictions](docs/network-firewall.md) | How to add firewall to AI Sandbox |
| [DockMCP Documentation](dkmcp/README.md) | MCP server details |
| [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) | Why DockMCP doesn't support container lifecycle operations |
| [Plugin Guide](docs/plugins.md) | Claude Code plugins for multi-repo setups |
| [Demo App Guide](demo-apps/README.md) | Running the SecureNote demo |
| [CLI Sandbox Guide](cli_sandbox/README.md) | Terminal-based sandbox |

## License

MIT License - See [LICENSE](LICENSE)

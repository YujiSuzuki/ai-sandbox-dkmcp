# AI Sandbox Environment + DockMCP

[日本語の README はこちら](README.ja.md)


AI coding agents read everything in your project directory — including `.env` files, API keys, and private certificates. Application-level deny rules can help, but they depend on correct configuration and have [scope limitations](docs/comparison.md). What if the secrets simply didn't exist in AI's filesystem?

This template creates a Docker-based development environment where:

- **Secrets are physically absent** — `.env` files and private keys don't exist in AI's filesystem, not blocked by rules — just not there
- **Misconfigurations are caught automatically** — Startup validation checks that your deny rules and volume mounts are in sync, warning you before AI sees anything
- **Code is fully accessible** — AI can read and edit all source code across multiple projects
- **Other containers are reachable** — With DockMCP, AI can check logs and run tests in other containers safely
- **Helper scripts and tools are discoverable** — Via SandboxMCP, AI automatically discovers and runs scripts and tools in `.sandbox/`

All you need is **Docker** and **VS Code**. [CLI-only usage is also supported](docs/reference.md#two-environments).

This project is designed for local development environments and is not intended for production use. See "[Limitations](#limitations)" and "[FAQ](#faq)" for details.

> [!NOTE]
> **Using DockMCP standalone with CLI tools (Claude Code, Gemini CLI, etc.) is not recommended.** CLI tools running on the host OS can execute `docker` commands directly, so there is no benefit to routing through DockMCP. However, for apps like **Claude Desktop** that can only access external systems via MCP, DockMCP standalone is useful for container operations. For standalone setup, see [dkmcp/README.md](dkmcp/README.md).


---

# Table of Contents

- [Problems This Solves](#problems-this-solves)
- [Use Cases](#use-cases)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [DockMCP Host Access](#dockmcp-host-access)
- [AI Sandbox Tools](#ai-sandbox-tools)
- [Project Structure](#project-structure)
- [Security Features](#security-features)
- [Supported AI Tools](#supported-ai-tools)
- [FAQ](#faq)
- [Documentation](#documentation)

<details>
<summary>📚 Documentation Links (Click to expand)</summary>

### 📖 Getting Started
- [Getting Started Guide](docs/getting-started.md) — Step-by-step setup from zero to a working environment
- [Comparison with Existing Solutions](docs/comparison.md) — How this compares to Claude Code Sandbox, Docker AI Sandboxes, etc.
- [Hands-on Guide](docs/hands-on.md) — Hands-on exercises for security features

### 🔧 Setup & Operations
- [Customization Guide](docs/customization.md) — How to adapt this template to your project
- [Updating Guide](docs/updating.md) — How to apply updates from new template releases
- [Reference](docs/reference.md) — Environment settings, options, troubleshooting

### 🏗️ Architecture
- [Architecture Details](docs/architecture.md) — Security mechanisms and architecture diagrams
- [Network Restrictions](docs/network-firewall.md) — How to add firewall to AI Sandbox

### 📦 Components
- [DockMCP Documentation](dkmcp/README.md) — MCP server details
- [DockMCP Host Access](docs/host-access.md) — Host tools, container lifecycle, and host command execution
- [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) — Graduated access model and AI-human responsibility separation
- [Plugin Guide](docs/plugins.md) — Claude Code plugins for multi-repo setups
- [Demo App Guide](demo-apps/README.md) — Running the SecureNote demo
- [CLI Sandbox Guide](cli_sandbox/README.md) — Terminal-based sandbox

</details>

----

# Problems This Solves

**Secret protection** — Running AI on the host OS makes it hard to prevent access to `.env` files and private keys. This environment isolates AI in a Docker container, creating a state where **code is visible but secret files are not**.

**Cross-project development** — Investigating issues at the boundary between apps and servers is hard work. This environment combines multiple projects into a single workspace so AI can see the entire system.

**Cross-container access** — Sandboxing prevents access to other containers, but DockMCP solves this. AI can read API container logs and run tests.

> **How does this compare to existing tools?** Claude Code Sandboxing and Docker AI Sandboxes are valuable — this project complements them by adding filesystem-level secret hiding and controlled cross-container access. See [Comparison with Existing Solutions](docs/comparison.md) for details.

## Limitations

- **Local development only** — DockMCP has no authentication, so it's designed for local use only
- **Docker required** — The volume mount approach requires a Docker-compatible runtime (Docker Desktop, OrbStack, etc.)
- **Only tested on macOS** — It should work on Linux and Windows, but this is unverified
- **No network restriction by default** — AI can still make outbound HTTP requests. See [Network Restrictions Guide](docs/network-firewall.md) for adding a firewall
- **Not a replacement for production secrets management** — This is a development-time protection layer. Use HashiCorp Vault, AWS Secrets Manager, etc. for production


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

Separately from DockMCP, **SandboxMCP** runs inside the container and lets AI automatically discover and run scripts and tools in `.sandbox/`. See [AI Sandbox Tools](#ai-sandbox-tools) for details.

→ For detailed architecture diagrams, see [Architecture Details](docs/architecture.md)

> [!TIP]
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
dkmcp serve --config configs/dkmcp.example.yaml --sync
```

The `--sync` flag runs the [host tools approval workflow](#host-tools) on startup, so AI can use the bundled demo tools right away. You can omit it if you don't need host tools.

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

### Step 3: Register DockMCP as an MCP server

In the AI Sandbox shell:

```bash
# Claude Code
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

# Gemini CLI
gemini mcp add --transport sse dkmcp http://host.docker.internal:8080/sse
```

For Claude Code, run `/mcp` → "Reconnect".

> **Important:** If you restart the DockMCP server, reconnection is required again.

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

Or, if you approved host tools with `--sync` in Step 1, ask AI instead:
- `Build and start the demo apps` — AI runs `demo-build.sh` and `demo-up.sh` via DockMCP

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
- **Want to detect configuration gaps?** → `.sandbox/scripts/check-secret-sync.sh` (sync check between AI deny settings and docker-compose.yml)

---

## Updating This Template

New versions are checked automatically on startup. When an update is available, you'll see a notification with the version info and a link to the release notes.

**Easiest way:** Ask your AI assistant — `"Please update to the latest version"`. It handles version checks, conflict detection, and rebuilds for you.

**Manual update:** See [Updating Guide](docs/updating.md) for step-by-step instructions (both clone and template users).

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

# DockMCP Host Access

DockMCP can also give AI controlled access to the **host OS** itself — not just other containers. Three capabilities are available, all configurable in `dkmcp.yaml`:

### Host Tools

AI can discover and run scripts placed in `.sandbox/host-tools/`. New tools go through an **approval workflow** — you review them with `dkmcp tools sync` before they become executable.

```
.sandbox/host-tools/         ← AI proposes tools here (staging)
~/.dkmcp/host-tools/<id>/    ← Only approved tools run from here
```

Three demo tools are included: `demo-build.sh`, `demo-up.sh`, `demo-down.sh` — letting AI manage demo app containers through approved scripts rather than raw Docker commands.

### Container Lifecycle

AI can start, stop, and restart containers using the Docker API directly. This is opt-in (`lifecycle: false` by default) and respects the `allowed_containers` policy.

```yaml
# In dkmcp.yaml
security:
  permissions:
    lifecycle: true  # Enable start/stop/restart
```

### Host Commands

AI can execute whitelisted CLI commands on the host OS (e.g., `git status`, `df -h`). Commands are restricted by base command + argument pattern matching, with deny lists and dangerous mode for sensitive operations.

```yaml
# In dkmcp.yaml
host_access:
  host_commands:
    enabled: true
    whitelist:
      "git": ["status", "diff *", "log --oneline *"]
```

> For full configuration details, approval workflow, and security considerations, see [DockMCP Host Access](docs/host-access.md)

# AI Sandbox Tools

## What are AI Sandbox Tools?

The AI Sandbox includes a lightweight MCP server called **SandboxMCP** (stdio). It is automatically built and registered at container startup, enabling AI to discover and run scripts and tools under `.sandbox/`.

| | SandboxMCP | DockMCP |
|---|-----------|---------|
| Runs on | Inside the container (stdio) | Host OS (SSE / HTTP) |
| Purpose | Discover and run scripts/tools | Access other containers |
| Startup | Automatic (container start) | Manual (`dkmcp serve`) |

Just ask AI things like "What scripts are available?" or "Search my conversation history" — SandboxMCP routes it to the right tool automatically.

> [!TIP]
> For SandboxMCP architecture details, see [docs/architecture.md](docs/architecture.md)

## Bundled Tools

Two tools are included out of the box.

### Conversation History Search

A built-in tool lets you search past Claude Code conversations. Just ask your AI — it handles the rest automatically via SandboxMCP.

**What you can ask:**

| Question | What AI does |
|----------|--------------|
| "What did we work on yesterday?" | Searches yesterday's messages and summarizes them |
| "Give me a summary of last week" | Looks up sessions day by day and creates an overview |
| "Did we discuss DockMCP setup?" | Keyword search across past conversations |
| "When did we fix that bug?" | Finds the relevant conversation by date and keyword |
| "Where did this mystery file come from?" | Traces back through past AI session commands to find the cause |

> [!TIP]
> For detailed usage and options, see [docs/search-history.md](docs/search-history.md)

### Token Usage Report

A built-in tool tracks how many tokens you're consuming in Claude Code. It breaks down usage by model and time period, and AI can estimate costs on the fly.

**Example questions you can ask:**

| What you say | What AI does |
|--------------|--------------|
| "How much did I use this week?" | Aggregates last 7 days of token usage by model |
| "Show me last month's usage and cost" | 30-day summary + fetches latest pricing for cost estimate |
| "How does this compare to a Pro plan?" | Calculates API cost and compares with Pro / Max plans |
| "Show me daily breakdown" | Displays per-day token consumption |

**How cost estimation works:**

When you ask about costs, AI fetches the latest pricing from [Anthropic's official pricing page](https://docs.anthropic.com/en/docs/about-claude/pricing) and calculates on the spot, so it adapts easily to pricing changes.

```
You: "What did last month cost?"
    ↓
AI: ① Runs the tool to aggregate token counts
    ② Fetches current pricing from docs.anthropic.com
    ③ Outputs cost breakdown + Pro/Max plan comparison
```

## Adding Your Own Tools and Scripts

### Your Own Tools

Place a Go file in `.sandbox/tools/` and AI will automatically discover it. No configuration needed.

### Your Own Scripts

Place shell scripts in `.sandbox/scripts/` and they will be discovered the same way.
Since scripts can call other languages (Python, Node.js, etc.), you can build tools in any language, not just Go.

> [!TIP]
> Adding a comment header with a description and usage info helps AI understand and use your tool effectively.
> For header format details, see [Architecture Details](docs/architecture.md#adding-custom-tools)


# Project Structure

`.sandbox/` contains shared infrastructure, `.devcontainer/` and `cli_sandbox/` provide two Sandbox environments, `dkmcp/` is the MCP server, and `demo-apps/` and `demo-apps-ios/` are demo applications.

<details>
<summary>View directory tree</summary>

```
workspace/
├── .sandbox/               # Shared sandbox infrastructure
│   ├── Dockerfile          # Container image definition
│   └── scripts/            # Shared scripts
│       ├── validate-secrets.sh    # Verify secret files are hidden
│       ├── check-secret-sync.sh   # Sync check with AI deny settings
│       └── sync-secrets.sh        # Interactively sync settings
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
| **Startup validation** | Automatically checks secret configuration consistency on every startup. Warns if issues are found |

→ For details and configuration, see [Architecture Details](docs/architecture.md). For startup validation details, see [Reference](docs/reference.md#startup-validation)

> [!NOTE]
> **About git status in the demo environment:** This template force-tracks demo secret files with `git add -f`, so they appear as "deleted" in git status inside the AI Sandbox. This won't happen in your own project since you'll add secret files to `.gitignore`. See [Hands-on Guide](docs/hands-on.md) for workarounds.


# Supported AI Tools

- ✅ **Claude Code** (Anthropic) - Full MCP support
- ✅ **Gemini Code Assist** (Google) - MCP support in Agent mode
- ✅ **Gemini CLI** (Google) - MCP support
- ✅ **Cline** (VS Code extension) - MCP integration (likely supported, unverified)



# FAQ

**Q: How is this different from Claude Code's sandboxing or Docker AI Sandboxes?**
A: They're complementary. Claude Code's sandbox restricts execution; Docker AI Sandboxes provide full VM isolation. This project adds filesystem-level secret hiding and cross-container access. Use them together for defense in depth. See [Comparison with Existing Solutions](docs/comparison.md) for details.

**Q: Do I need to use DockMCP?**
A: No. It works as a regular sandbox without DockMCP. DockMCP enables cross-container access.

**Q: Why not just mount the Docker socket so AI can access containers directly?**
A: Docker socket access is essentially host admin privileges — AI could read secrets from any container, bypassing all hiding. DockMCP exists to provide only the operations AI needs (logs, tests) in a safe, controlled way. See [Architecture Details](docs/architecture.md#5-why-no-docker-socket-access) for details.

**Q: Can AI run `docker-compose up/down`?**
A: Not directly — but AI can run approved host tools (e.g., `demo-up.sh`, `demo-down.sh`) that wrap these commands. Raw `docker-compose` and image builds remain human-only, while host tools provide controlled access through human-reviewed scripts. See [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) for details.

**Q: Can I use a different secret management solution?**
A: Yes! This can be combined with HashiCorp Vault, AWS Secrets Manager, or other tools. This project handles development-time protection; use dedicated tools for production.



# Documentation

| Document | Description |
|----------|-------------|
| [Getting Started Guide](docs/getting-started.md) | Step-by-step setup from zero to a working environment |
| [Comparison with Existing Solutions](docs/comparison.md) | How this compares to Claude Code Sandbox, Docker AI Sandboxes, etc. |
| [Hands-on Guide](docs/hands-on.md) | Hands-on exercises for security features |
| [Customization Guide](docs/customization.md) | How to adapt this template to your project |
| [Reference](docs/reference.md) | Environment settings, options, troubleshooting |
| [Architecture Details](docs/architecture.md) | Security mechanisms and architecture diagrams |
| [Network Restrictions](docs/network-firewall.md) | How to add firewall to AI Sandbox |
| [DockMCP Documentation](dkmcp/README.md) | MCP server details |
| [DockMCP Host Access](docs/host-access.md) | Host tools, container lifecycle, and host command execution |
| [DockMCP Design Philosophy](dkmcp/README.md#design-philosophy) | Graduated access model and AI-human responsibility separation |
| [Plugin Guide](docs/plugins.md) | Claude Code plugins for multi-repo setups |
| [Demo App Guide](demo-apps/README.md) | Running the SecureNote demo |
| [CLI Sandbox Guide](cli_sandbox/README.md) | Terminal-based sandbox |

> **Note:** `docs/ai-guide.md` is a reference guide for AI assistants (referenced from CLAUDE.md and GEMINI.md). Users don't need to read it.

## License

MIT License - See [LICENSE](LICENSE)

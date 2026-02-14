# Getting Started Guide

A step-by-step walkthrough from zero to a working AI Sandbox + DockMCP setup.

[← Back to README](../README.md)

---

## Who This Guide Is For

- You're familiar with Docker and VS Code but new to this project
- You want to understand how AI Sandbox and DockMCP fit together
- You want to get things running first, then explore

**Estimated time:** 15–30 minutes (5 minutes without DockMCP)

---

## Overview

Setup has three stages. Go as far as you need.

```
Steps 1–3: Get AI Sandbox running (required)
    ↓
Steps 4–6: Connect DockMCP (recommended)
    ↓
Steps 7–8: Try the demo apps (optional)
```

| Stage | What You Get |
|-------|-------------|
| **Sandbox only** | AI can read/write code. Secret files are hidden |
| **+ DockMCP** | AI can also check logs and run tests in other containers |
| **+ Demo apps** | Try all features with the included demo |

---

## Step 1: Check Prerequisites

Make sure the following are installed.

| Tool | Verify | Install |
|------|--------|---------|
| **Docker** | `docker --version` | [Docker Desktop](https://www.docker.com/products/docker-desktop/) or [OrbStack](https://orbstack.dev/) |
| **VS Code** | `code --version` | [Visual Studio Code](https://code.visualstudio.com/) |
| **Dev Containers extension** | Check in VS Code extensions | [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) |

If you plan to use DockMCP, you'll also need:

| Tool | Verify | Install |
|------|--------|---------|
| **Go** (1.24+) | `go version` | [go.dev](https://go.dev/dl/) |

> [!TIP]
> **No Go on host?** The AI Sandbox includes a Go environment, so you can cross-compile the host binary from inside the container (explained in Step 4).

### Expected Result

```bash
$ docker --version
Docker version 27.x.x, build xxxxxxx   # Any version shown = OK

$ code --version
1.9x.x                                  # Any version shown = OK
```

Also confirm Docker Desktop (or OrbStack) is **running**.

---

## Step 2: Get the Repository

```bash
# Option A: From template (click "Use this template" on GitHub, then clone)
git clone https://github.com/your-username/your-new-repo.git
cd your-new-repo

# Option B: Direct clone
git clone https://github.com/YujiSuzuki/ai-sandbox-dkmcp.git
cd ai-sandbox-dkmcp
```

> [!TIP]
> For more options, see [Customization Guide](customization.md).

### Expected Result

```
your-repo/
├── .devcontainer/
├── .sandbox/
├── dkmcp/
├── demo-apps/
└── README.md
```

If you see this directory structure, you're good.

---

## Step 3: Start the DevContainer

```bash
code .
```

Once VS Code opens:

1. A notification **"Reopen in Container"** appears in the bottom right → click it
2. If no notification appears → `Cmd+Shift+P` (macOS) / `Ctrl+Shift+P` (Windows/Linux) → **"Dev Containers: Reopen in Container"**

The first launch builds the container, which takes a few minutes.

### What Happens on First Startup

During DevContainer startup, these processes run automatically:

1. Docker image build (first time only; cached on subsequent starts)
2. Container startup
3. AI settings merge (consolidates `.claude/settings.json` from subprojects)
4. Secret config validation (checks `docker-compose.yml` settings are correct)
5. SandboxMCP build and registration (makes `.sandbox/` tools available to AI)
6. Template update check

Validation results appear in the VS Code terminal. If you see `✓` (success) for each check, everything is fine.

### Expected Result

- VS Code shows **"Dev Container: AI Sandbox"** in the bottom left
- A terminal is open at `/workspace`
- `ls` shows the project files

```bash
$ ls demo-apps/securenote-api/.env
demo-apps/securenote-api/.env     # The file appears to exist, but...

$ cat demo-apps/securenote-api/.env
                                  # It's empty! (hidden by volume mount)
```

**At this point, the AI Sandbox is ready to use.** Launch Claude Code or Gemini Code Assist and try reading/writing code.

If you don't need DockMCP (access to other containers), skip ahead to [Next Steps](#next-steps).

> [!TIP]
> **Not using VS Code?** You can also use the CLI Sandbox (`cli_sandbox/`) for terminal-only workflows. Run `./cli_sandbox/claude.sh` for Claude Code or `./cli_sandbox/gemini.sh` for Gemini CLI. See [Reference](reference.md) for details.

---

## Step 4: Build DockMCP (on Host OS)

> [!IMPORTANT]
> From here, work on your **host OS** (outside the DevContainer). Open a separate terminal window — not the VS Code integrated terminal.

```bash
cd dkmcp
make install
```

This installs the `dkmcp` command to `~/go/bin/`.

<details>
<summary>No Go on your host OS?</summary>

The AI Sandbox includes a Go environment, so you can cross-compile for your host.

**Inside AI Sandbox:**
```bash
cd /workspace/dkmcp
make build-host
```

**On host OS:**
```bash
cd <path-to-repo>/dkmcp
make install-host DEST=~/go/bin        # If Go is installed
make install-host DEST=/usr/local/bin  # If Go is not installed
```

</details>

### Expected Result

```bash
$ dkmcp version
x.x.x    # Version shown = OK
```

---

## Step 5: Start the DockMCP Server (on Host OS)

Still on the host OS:

```bash
dkmcp serve --config configs/dkmcp.example.yaml
```

Adding `--sync` enables host tools — scripts in `.sandbox/host-tools/` (for building, starting, and stopping demo apps, etc.) that AI can execute via DockMCP. The first time AI tries to use a host tool, you'll be prompted to approve it.

```bash
dkmcp serve --config configs/dkmcp.example.yaml --sync
```

### Expected Result

```
DockMCP server started on :8080
Security mode: moderate
Allowed containers: securenote-*, demo-*
```

Keep this terminal open (the server keeps running).

### Verify Connection (from another host terminal)

```bash
curl http://localhost:8080/health
# → 200 OK means success
```

---

## Step 6: Connect from AI Sandbox to DockMCP

Switch back to the **VS Code DevContainer terminal**:

```bash
# For Claude Code
claude mcp add --transport sse --scope user dkmcp http://host.docker.internal:8080/sse

# For Gemini CLI
gemini mcp add --transport sse dkmcp http://host.docker.internal:8080/sse
```

After registering, activate the connection:

- **Claude Code:** Type `/mcp` → select "Reconnect"
- **Alternatively:** Restart VS Code entirely (`Cmd+Q` / `Alt+F4` → reopen)

### Expected Result

Running `/mcp` in Claude Code shows `dkmcp` as **connected**:

```
  dkmcp
  ✔ connected
  17 tools
```

Try asking the AI:

```
"Show me the list of containers"
```

> [!NOTE]
> If you haven't started the demo apps yet, the container list may be empty. That's fine — the connection itself is confirmed.

### If It Doesn't Work

- See [Troubleshooting](reference.md#troubleshooting)
- Verify the DockMCP server is running (Step 5)
- Check if port 8080 is being forwarded in VS Code's Ports panel — if so, stop it

---

## Step 7: Start the Demo Apps (Optional)

To experience DockMCP's full capabilities, start the included demo apps.

**On host OS:**

```bash
cd demo-apps
docker compose -f docker-compose.demo.yml up -d --build
```

If you started with `--sync` in Step 5, you can also ask the AI:
```
"Build and start the demo apps"
```
> [!NOTE]
> The first time AI uses a host tool, Claude Code will show an approval dialog. Once approved, subsequent executions run automatically.

### (Recommended) Custom Domain Setup

Makes it easier to access the demo apps in your browser.

```bash
# On host OS
echo "127.0.0.1 securenote.test api.securenote.test" | sudo tee -a /etc/hosts
```

### Expected Result

```bash
# Check containers on host OS
$ docker ps
CONTAINER ID   IMAGE              STATUS    NAMES
xxxxxxxxxxxx   securenote-api     Up        securenote-api
xxxxxxxxxxxx   securenote-web     Up        securenote-web
```

Browser access:
- Web: http://securenote.test:8000 (with domain setup)
- API: http://api.securenote.test:8000/api/health

---

## Step 8: Talk to the AI

In the AI Sandbox, try these prompts with Claude Code (or Gemini):

### Basic Operations

```
"Show me the logs from securenote-api"
→ Container logs displayed via DockMCP

"Run npm test in securenote-api"
→ Test results returned

"What scripts are available?"
→ Script list from .sandbox/ displayed via SandboxMCP
```

### Security Verification

```
"Show me the contents of demo-apps/securenote-api/.env"
→ Empty file (secrets are hidden)

"Check if any secret files are accessible"
→ AI runs validation script and reports the hiding status
```

### DockMCP Features

```
"Show me detailed info about the securenote-api container"
→ Container inspect results displayed

"How much memory is securenote-api using?"
→ Container resource stats displayed
```

---

## Next Steps

With setup complete, continue based on what you want to do.

| Goal | Document |
|------|----------|
| Explore security features hands-on | [Hands-on Guide](hands-on.md) |
| Use with your own project | [Customization Guide](customization.md) |
| Understand the architecture | [Architecture Details](architecture.md) |
| Compare with other tools | [Comparison with Existing Solutions](comparison.md) |
| Add network restrictions | [Network Restrictions](network-firewall.md) |

---

## Common Issues and Fixes

### "Reopen in Container" doesn't appear

- Verify the Dev Containers extension is installed
- Run `Cmd+Shift+P` → "Dev Containers: Reopen in Container" manually

### First build is slow

- Docker image download and build can take 3–5 minutes
- Subsequent starts use the cache and are much faster

### Can't connect to DockMCP

1. Verify the DockMCP server is running: `curl http://localhost:8080/health`
2. Check if port 8080 is being forwarded in VS Code's Ports panel — if so, stop it
3. Try `/mcp` → "Reconnect"
4. Restart VS Code completely (`Cmd+Q` → reopen)

For more details, see [Troubleshooting](reference.md#troubleshooting).

### Demo app containers not found

- Run `docker ps` on the host OS to verify containers are running
- Re-run `docker compose -f docker-compose.demo.yml up -d --build`
- Check that `allowed_containers` in `dkmcp.example.yaml` includes the container name patterns

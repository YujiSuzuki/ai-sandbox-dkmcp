# Comparison with Existing Solutions

How AI Sandbox + DockMCP compares to other AI security tools, and why they work well together.

[← Back to README](../README.md)

---

## Existing Tools

### Claude Code Sandboxing

[Claude Code Sandboxing](https://code.claude.com/docs/en/sandboxing) uses OS-level primitives (Seatbelt on macOS, bubblewrap on Linux) to restrict filesystem writes and network access. You can also add `Read` deny rules in permissions to block AI from reading specific files.

**Strengths:**
- OS-level execution restrictions (no extra setup)
- Read deny rules to block access to specific files
- Reduces permission fatigue

**Gaps this project fills:**
- Deny rules are application-level — they depend on correct configuration and the AI tool respecting them
- Deny rules [don't traverse parent directories](https://github.com/anthropics/claude-code/issues/12962), so in a monorepo or multi-project workspace, settings in one project won't protect secrets in a sibling project
- Sandbox restricts outbound network access, which limits cross-container debugging

### Docker AI Sandboxes

[Docker AI Sandboxes](https://docs.docker.com/ai/sandboxes) run AI agents in isolated microVMs with their own Docker daemon. The agent can't touch your host system.

**Strengths:**
- Strong isolation via microVMs
- Full autonomy for AI agents within the sandbox
- Each sandbox has its own Docker daemon

**Gaps this project fills:**
- Syncs your entire workspace directory into the microVM with no mechanism to exclude specific files — `.env` files are visible inside
- Fully isolated — each sandbox can't communicate with other containers, making cross-container debugging impossible

### Docker MCP Toolkit

[Docker MCP Toolkit](https://www.docker.com/blog/mcp-toolkit-mcp-servers-that-just-work/) provides 200+ containerized MCP servers with built-in isolation and secret management.

**Strengths:**
- Large catalog of pre-built MCP servers
- Built-in secret management for MCP server configurations

**Gaps this project fills:**
- Focuses on MCP server isolation, not on hiding project-level secrets from AI
- Doesn't address the problem of `.env` files and private keys in your source tree

---

## What This Project Adds

AI Sandbox + DockMCP fills two specific gaps that the tools above don't fully address:

### Gap 1: Filesystem-level secret hiding

Instead of blocking secret access with rules (which can be misconfigured or bypassed), this project makes secrets **physically absent** from AI's filesystem using Docker volume mounts:

```yaml
volumes:
  - /dev/null:/workspace/my-app/.env:ro     # AI sees an empty file
tmpfs:
  - /workspace/my-app/secrets:ro            # AI sees an empty directory
```

The secrets don't exist in AI's world — not blocked by a rule, not filtered by a config, just not there. Meanwhile, your app containers mount the real files normally.

To catch misconfigurations, the sandbox runs **startup validation** that checks whether your AI tool's deny rules and your `docker-compose.yml` volume mounts are in sync. If a secret file is blocked in one but not the other, you get a warning before AI sees it.

### Gap 2: Controlled cross-container access

DockMCP acts as a gateway between the AI sandbox and other Docker containers, with security policy enforcement:

- AI can read logs, run whitelisted commands, and inspect containers
- AI cannot start/stop containers, access blocked paths, or run arbitrary commands
- Sensitive data (passwords, API keys, tokens) is automatically masked in output

---

## Using Them Together

These tools are **complementary, not competing**. For defense in depth:

| Layer | Tool | What It Does |
|-------|------|-------------|
| Execution restriction | Claude Code Sandbox | Prevents malicious command execution |
| System isolation | Docker AI Sandboxes | Isolates AI in a microVM |
| Secret hiding | **AI Sandbox** (this project) | Makes secrets absent from AI's filesystem |
| Cross-container access | **DockMCP** (this project) | Controlled access to other containers |

You can use Claude Code's sandbox *inside* the AI Sandbox for maximum protection.

---

## Summary

| Feature | Claude Code Sandbox | Docker AI Sandboxes | This Project |
|---------|-------------------|-------------------|-------------|
| Execution restriction | OS-level | microVM isolation | Container isolation |
| File read blocking | Deny rules (application-level) | No mechanism | Volume mounts (filesystem-level) |
| Multi-project scope | Limited (no parent traversal) | Single workspace | Full workspace with per-file hiding |
| Cross-container access | Restricted | Isolated | Controlled via DockMCP |
| Secret masking in output | No | No | Automatic |
| Startup validation | No | No | Automatic sync check |
| Setup complexity | None (built-in) | Docker Desktop | Docker + docker-compose |

# Network Restrictions (Firewall)

How to add network restrictions to AI Sandbox.

[← Back to README](../README.md)

---

## Network Restrictions in AI Sandbox

AI Sandbox provides secret file hiding and cross-container access, but **does not include network restrictions**. Outbound traffic from the AI Sandbox container is unrestricted, so consider adding a firewall if needed.

## Anthropic's Official Firewall Script

Anthropic publishes a firewall script for their Claude Code DevContainer.

- **Repository:** [anthropics/claude-code/.devcontainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer)
- **Script:** [init-firewall.sh](https://github.com/anthropics/claude-code/blob/main/.devcontainer/init-firewall.sh)

### How It Works

- Whitelist-based approach using `iptables` + `ipset`
- Blocks all outbound traffic by default, allowing only approved domains
- Allowed destinations: GitHub, npm registry, Anthropic API, VS Code Marketplace, etc.

> **Note:** This script is maintained by Anthropic. For details and the latest changes, see the [official repository](https://github.com/anthropics/claude-code/tree/main/.devcontainer).

---

## Adding to AI Sandbox

The following is one approach. Steps may change if the official script is updated.

### Step 1: Download the script

```bash
# Run from the project root
curl -o .devcontainer/init-firewall.sh \
  https://raw.githubusercontent.com/anthropics/claude-code/main/.devcontainer/init-firewall.sh
chmod +x .devcontainer/init-firewall.sh
```

### Step 2: Add firewall initialization to devcontainer.json

Add the script to the beginning of `postStartCommand` in `.devcontainer/devcontainer.json`:

```jsonc
// Add at the beginning of the existing postStartCommand
"postStartCommand": "/workspace/.devcontainer/init-firewall.sh && /workspace/.sandbox/scripts/merge-claude-settings.sh && ..."
```

> **Tip:** The firewall should run before other startup scripts.

### Step 3: Add required packages to Dockerfile

The firewall script requires `iptables` and `ipset`. Add them to `.sandbox/Dockerfile`:

```dockerfile
# Firewall packages
RUN sudo apt-get update && sudo apt-get install -y iptables ipset curl \
    && sudo rm -rf /var/lib/apt/lists/*
```

> **Note:** The script may require `sudo`. The AI Sandbox `node` user does not have sudo access to `iptables` by default, so you may need to configure this in the Dockerfile or adjust sudoers.

### Step 4: Rebuild DevContainer

```bash
# VS Code: Cmd+Shift+P → "Dev Containers: Rebuild Container"
```

---

## Important Notes

### Coexistence with DockMCP

DockMCP communicates with the host OS via `host.docker.internal`. The official script allows host network traffic, so DockMCP should work without issues.

If you experience connection problems, verify that the firewall rules allow access to the DockMCP port (default: 8080).

### Using AI tools other than Claude Code

The official script's allowlist is designed for Claude Code. If you use Gemini CLI or other AI tools, you will need to add the domains those tools require.

### CLI Sandbox usage

If using CLI Sandbox (`cli_sandbox/`), apply the same configuration to `cli_sandbox/docker-compose.yml` as well.

---

## References

- [Anthropic Official DevContainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer) — Source of the firewall script
- [Claude Code Sandboxing Documentation](https://code.claude.com/docs/sandboxing) — Claude Code's native sandboxing features
- [Docker Sandbox](https://docs.docker.com/ai/sandboxes) — Docker's official AI sandbox (microVM-based)

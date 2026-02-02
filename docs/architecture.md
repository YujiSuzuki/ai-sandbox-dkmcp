# Architecture Details

Detailed diagrams explaining how AI Sandbox + DockMCP works.

[← Back to README](../README.md)

---

## Overall Architecture

```
┌───────────────────────────────────────────────────┐
│ Host OS                                           │
│                                                   │
│  ┌──────────────────────────────────────────────┐ │
│  │ DockMCP Server                               │ │
│  │  HTTP/SSE API for AI                         ←─────┐
│  │  Security policy enforcement                 │ │   │
│  │  Container access gateway                    │ │   │
│  │                                              │ │   │
│  └────────────────────↑─────────────────────────┘ │   │
│                       │ :8080                     │   │
│  ┌────────────────────│─────────────────────────┐ │   │
│  │ Docker Engine      │                         │ │   │
│  │                    │                         │ │   │
│  │   AI Sandbox  ←────┘                         │ │   │
│  │    └─ Claude Code / Gemini                   │ │   │
│  │       secrets/ → empty (hidden)              │ │   │
│  │                                              │ │   │
│  │   API Container    ←───────────────────────────────┘
│  │    └─ secrets/ → real files                  │ │   │
│  │                                              │ │   │
│  │   Web Container    ←───────────────────────────────┘
│  │                                              │ │
│  └──────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────┘
```

<details>
<summary>Tree format</summary>

**Data flow:** AI (AI Sandbox) → DockMCP (:8080) → Other containers

```
Host OS
├── DockMCP Server (:8080)
│   ├── HTTP/SSE API for AI
│   ├── Security policy enforcement
│   └── Container access gateway
│
└── Docker Engine
    ├── AI Sandbox (AI environment)
    │   ├── Claude Code / Gemini
    │   └── secrets/ → empty (hidden)
    │
    ├── API Container
    │   └── secrets/ → real files
    │
    └── Web Container
```

</details>

---

## How Secret Hiding Works

Since AI runs inside the AI Sandbox, Docker volume mounts can hide secret files.

```
Host OS
├── demo-apps/securenote-api/.env  ← actual file
│
├── AI Sandbox (AI execution environment)
│   └── AI tries to read .env
│       → Mounted to /dev/null, appears empty
│
└── API Container (runtime environment)
    └── Node.js app reads .env
        → Reads normally
```

**Result:**
- AI cannot read secret files (security ensured)
- Apps can read secret files (functionality maintained)
- AI can still check logs and run tests via DockMCP

---

## Benefits of AI Sandbox Isolation

Running AI inside the AI Sandbox also restricts access to host OS files.

```
Host OS
├── /etc/            ← inaccessible to AI
├── ~/.ssh/          ← inaccessible to AI
├── ~/Documents/     ← inaccessible to AI
├── ~/other-project/ ← inaccessible to AI
├── ~/secret-memo/   ← inaccessible to AI
│
└── AI Sandbox
    └── /workspace/   ← only this is visible
        ├── demo-apps/
        ├── dkmcp/
        └── ...
```

**Benefits:**
- Cannot touch host OS system files
- Cannot access other projects
- Cannot access SSH keys or credentials (`~/.ssh/`)
- No risk of accidentally modifying the host OS

---

## Security Features in Detail

### 1. Secret Hiding

Hides secrets from AI using Docker volume mounts:

```yaml
# .devcontainer/docker-compose.yml
volumes:
  # Hide secret files
  - /dev/null:/workspace/demo-apps/securenote-api/.env:ro

tmpfs:
  # Hide secret directories
  - /workspace/demo-apps/securenote-api/secrets:ro
```

**Result:**
- AI sees empty files/directories
- Actual containers can access real secrets
- Development works normally!

**Example in action:**

```bash
# From inside AI Sandbox (AI tries but fails)
$ cat demo-apps/securenote-api/secrets/jwt-secret.key
(empty)

# But ask Claude Code:
"Check if the API can access its secrets"

# Claude queries via DockMCP:
$ curl http://localhost:8080/api/demo/secrets-status

# Response proves the API has its secrets:
{
  "secretsLoaded": true,
  "proof": {
    "jwtSecretLoaded": true,
    "jwtSecretPreview": "super-sec***"
  }
}
```

### 2. Controlled Container Access

DockMCP enforces security policies:

```yaml
# dkmcp.yaml
security:
  mode: "moderate"  # strict | moderate | permissive

  allowed_containers:
    - "demo-*"
    - "project_*"

  exec_whitelist:
    "securenote-api":
      - "npm test"
      - "npm run lint"
```

For container file blocking (`blocked_paths`), auto-import from Claude Code / Gemini settings, and more, see [dkmcp/README.md "Configuration Reference"](../dkmcp/README.md#configuration-reference).

**Example — Cross-container debugging:**

```bash
# Simulate a bug: Can't log in on web app

# Ask Claude Code:
"Login is failing. Can you check the API logs?"

# Claude gets logs via DockMCP:
dkmcp.get_logs("securenote-api", { tail: "50" })

# Error found in logs:
"JWT verification failed - invalid secret"

# Ask Claude Code:
"Please run the API tests to verify"

# Claude runs tests via DockMCP:
dkmcp.exec_command("securenote-api", "npm test")

# Issue identified and fixed!
```

### 3. Basic Sandbox Protection

- **Non-root user**: Runs as `node` user
- **Limited sudo**: Package managers only (apt, npm, pip)
- **Credential persistence**: Named volumes for `.claude/`, `.config/gcloud/`

> ⚠️ **Security note: npm/pip3 sudo risks**
>
> Allowing sudo for npm/pip3 can be exploited through malicious packages. Malicious postinstall scripts can execute arbitrary code with elevated privileges.
>
> **Mitigation options:**
> 1. Remove npm/pip3 from sudoers (edit `.sandbox/Dockerfile`)
> 2. Use `npm install --ignore-scripts` flag
> 3. Pre-install required packages in Dockerfile
> 4. Set `ignore-scripts=true` in `.npmrc`

### 4. Output Masking (Defense in Depth)

Even if secrets appear in logs or command output, DockMCP automatically masks them:

```
# Raw log output
DATABASE_URL=postgres://user:secret123@db:5432/app

# What AI sees (after masking)
DATABASE_URL=[MASKED]db:5432/app
```

Detects passwords, API keys, Bearer tokens, database URLs with credentials, and more by default. For configuration details, see [dkmcp/README.md "Output Masking"](../dkmcp/README.md#output-masking).

---

## Multi-Project Workspace

These security features enable safely working with multiple projects in a single workspace.

Example in this demo environment:
- **Backend API** (demo-apps/securenote-api)
- **Web Frontend** (demo-apps/securenote-web)
- **iOS App** (demo-apps-ios/)

What AI can do:
- Read all source code (investigate issues across app and server boundaries)
- Check any container's logs (via DockMCP)
- Run tests across projects
- Debug cross-container issues
- **Never touch secrets**

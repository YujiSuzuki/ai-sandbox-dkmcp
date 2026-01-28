# SecureNote API

Backend API for the SecureNote demo application. Demonstrates secure secret management with AI coding assistants.

## Features

- 🔐 **Encrypted storage** - Note contents are encrypted at rest
- 🔑 **JWT authentication** - Secure token-based auth
- 🚫 **Secret isolation** - Secrets hidden from AI assistants in DevContainer
- ✅ **Fully functional** - API works normally despite hidden secrets

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login and get JWT token

### Notes
- `GET /api/notes` - List all notes (decrypted)
- `GET /api/notes/:id` - Get specific note
- `POST /api/notes` - Create new note (encrypts content)
- `PUT /api/notes/:id` - Update note
- `DELETE /api/notes/:id` - Delete note

### Demo
- `GET /api/demo/secrets-status` - Check if secrets are loaded (proof for demo)
- `GET /api/health` - Health check

## Demo Users

| Username | Password |
|----------|----------|
| demo     | demo123  |
| alice    | alice123 |

## Local Development

```bash
# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Start server
npm start

# Run tests
npm test
```

## Docker

```bash
# Build
docker build -t securenote-api .

# Run
docker run -p 8080:8080 \
  -v $(pwd)/secrets:/app/secrets:ro \
  -v $(pwd)/.env:/app/.env:ro \
  securenote-api
```

## Security Demo

### What's Hidden from AI

When running in DevContainer, these files are NOT visible to AI assistants:
- `secrets/jwt-secret.key`
- `secrets/encryption.key`
- `.env`

### How It Works

1. DevContainer mounts `secrets/` as empty tmpfs
2. DevContainer mounts `.env` as `/dev/null`
3. AI cannot read secrets
4. But API container HAS access to real secrets
5. Development continues normally!

### Verify

```bash
# From DevContainer (AI environment)
cat demo-apps/securenote-api/secrets/jwt-secret.key
# => (empty or error)

# Use DockMCP to query the API
curl http://api.securenote.test:8000/api/demo/secrets-status
# => { "secretsLoaded": true, "proof": { ... } }
```

This proves the API has secrets, but AI doesn't!

## Architecture

```
┌─────────────────────────────┐
│ DevContainer (AI)           │
│ secrets/ → (empty tmpfs)    │  🔐 AI can't read
│ .env → /dev/null            │
└─────────────────────────────┘

┌─────────────────────────────┐
│ API Container               │
│ secrets/ → real files       │  ✅ API can read
│ .env → real config          │
└─────────────────────────────┘
```

## Testing via DockMCP

With DockMCP MCP server running, AI assistants can:

```javascript
// List containers
dkmcp.list_containers()

// Get API logs
dkmcp.get_logs("securenote-api", { tail: "50" })

// Run tests
dkmcp.exec_command("securenote-api", "npm test")

// Check stats
dkmcp.get_stats("securenote-api")
```

All without accessing secrets directly!

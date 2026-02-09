# Conversation History Search (search-history)

A tool for searching and browsing past Claude Code conversations.

Inside the AI Sandbox, just ask your AI — SandboxMCP auto-discovers the tool and AI runs it as needed. No commands to memorize.

## What It Can Do

### 1. Keyword Search

Search past conversations for specific keywords.

```
You: "Do you remember when we talked about DockMCP setup?"
AI:  Runs search-history with "DockMCP" → summarizes matching conversations
```

### 2. Session Listing

See what sessions occurred in a given time range.

```
You: "Show me last week's conversations"
AI:  Runs -list -after 2026-02-03 -before 2026-02-07 → displays session list
```

Example output:
```
00de1fea-a91  02/04 14:32 ~ 02/09   1179 msgs  3409KB  Show container list
1aef51ad-11d  02/05 14:05              163 msgs   542KB  comparison-article.md
494c72df-149  02/03 14:39 ~ 02/04     506 msgs  8102KB  README.ja.md
```

### 3. Activity Recap

Get AI to summarize your work by day or week.

```
You: "What did we work on yesterday?"
AI:  Searches yesterday's messages → summarizes topics by time

You: "Give me a summary of last week"
AI:  Looks up a full week of sessions → creates a day-by-day overview
```

### 4. Session Viewer

Browse messages in a specific session chronologically.

```
You: "Show me the details of that conversation"
AI:  Runs -session <id> to display messages in order
```

### 5. Forensic Investigation

When you find unexpected files or changes, you can trace back through past AI sessions to identify the cause.

AI sessions are ephemeral — once a session ends, the AI that ran it is gone. Conversation history is the only record of what happened, and this tool can extract it.

```
You: "Where did this search-history binary come from?"
AI:  Searches Bash execution history
     → Finds that a previous session ran go build without -o, leaving the binary behind
```

**Real-world example:**

During code review, a 3.5MB binary `.sandbox/sandbox-mcp/search-history` was found staged for commit. `git log` had no record (it was never committed), and the file timestamp alone didn't explain the cause. Searching Bash execution history with search-history revealed that a previous AI session had run `go build` without `-o` to verify compilation, accidentally leaving the binary in the working directory.

## Command-Line Usage

You can also run the tool directly without asking AI.

### Three Modes

| Mode | Command | Description |
|------|---------|-------------|
| Keyword search | `go run .sandbox/tools/search-history.go "query"` | Supports regex |
| Session list | `go run .sandbox/tools/search-history.go -list` | Sorted by last activity |
| Session viewer | `go run .sandbox/tools/search-history.go -session <id>` | Prefix match on ID |

### Filter Options

| Option | Description | Example |
|--------|-------------|---------|
| `-role <role>` | Filter by role | `-role user` (your messages only) |
| `-tool <name>` | Filter by tool name (use with `-role tool`) | `-role tool -tool Bash` |
| `-after <date>` | Only after this date | `-after 2026-02-01` |
| `-before <date>` | Only before this date | `-before 2026-02-07` |
| `-i` | Case-insensitive search | `-i "dockmcp"` |
| `-project <name>` | Project scope (default: workspace) | `-project all` |

### Display Options

| Option | Description | Default |
|--------|-------------|---------|
| `-max <n>` | Maximum number of results | 50 (0 = unlimited) |
| `-context <n>` | Characters per entry | 200 (0 = full text) |
| `-no-color` | Disable color output | — |

### Examples

```bash
# Keyword search
go run .sandbox/tools/search-history.go "DockMCP"

# Search only your messages
go run .sandbox/tools/search-history.go -role user "docker"

# Search Bash tool execution history
go run .sandbox/tools/search-history.go -role tool -tool Bash "git"

# Search within a date range
go run .sandbox/tools/search-history.go -after 2026-01-20 "secret"

# Search across all projects
go run .sandbox/tools/search-history.go -project all "error"

# List sessions for a specific day
go run .sandbox/tools/search-history.go -list -after 2026-02-08 -before 2026-02-08

# View a full session (no truncation, no limit)
go run .sandbox/tools/search-history.go -session c01514d6 -context 0 -max 0
```

## How Date Filters Work

`-after` / `-before` filter by **message timestamps**, not session start time.

- Keyword search: only messages within the date range are matched
- Session list (`-list`): shows sessions that have messages in the date range
- Multi-day sessions are included if they have any messages on the specified day

Dates are interpreted in the local timezone.

## How It Works

Claude Code stores conversation history under `~/.claude/projects/` as JSONL files. search-history reads these files directly.

```
~/.claude/projects/<project-dir>/
  └── <session-id>.jsonl    ← conversation data
```

When AI runs it via SandboxMCP's `run_tool`, it reads the same files.

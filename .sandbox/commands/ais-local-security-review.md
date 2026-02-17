---
description: Run local security-focused code review (works even without a Git repository)
description-ja: ローカルセキュリティレビューを実行（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Security Review

Performs security-focused code review on local code. If a Git repository exists, it reviews the diff between branches; otherwise, it reviews the specified files/directories. Focuses on vulnerabilities, injection risks, authentication/authorization flaws, and secret exposure.

## Language

Detect the user's language from their previous messages in the conversation. Output all review results, issue descriptions, and recommendations in the same language the user uses. If uncertain, default to English.

## Arguments

User-specified arguments: $ARGUMENTS

Argument interpretation:
- 1st argument: Project path (interactive selection if omitted)
- 2nd argument onwards: Change summary (asked via AskUserQuestion in Step 3 if omitted)

## Execution Steps

Follow these steps precisely:

### Step 1: Project Selection and Git Detection

1. Search for projects under `/workspace` (both Git repositories and regular directories):
   ```bash
   # Search for Git repositories
   find /workspace -name ".git" -type d -maxdepth 3 2>/dev/null | sed 's/\/.git$//'
   # Also search for main project directories (those with package.json, go.mod, Cargo.toml, etc.)
   find /workspace -maxdepth 2 -type f \( -name "package.json" -o -name "go.mod" -o -name "Cargo.toml" -o -name "pyproject.toml" -o -name "Makefile" \) 2>/dev/null | xargs -I {} dirname {}
   ```

2. If `$ARGUMENTS` is empty, use the AskUserQuestion tool to let the user select a project to review from the found projects

3. Check if the selected project directory has `.git` and determine **Git mode** or **Non-Git mode**:
   ```bash
   test -d <project-path>/.git && echo "GIT_MODE" || echo "NON_GIT_MODE"
   ```

### Step 2: Determine Review Target

#### For Git Mode:

1. Check the current branch and available branches:
   ```bash
   git -C <project-path> branch -a
   git -C <project-path> branch --show-current
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Base branch**: The branch to compare against (e.g., main, master, develop)
   - **Target branch**: The branch to review (current branch by default)

#### For Non-Git Mode:

1. Check the file structure within the project:
   ```bash
   find <project-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" \) 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Review target**: Path(s) to files or directories to review (can be multiple)
   - Examples: `src/`, `internal/mcp/`, `main.go`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of the purpose and background of the changes
  - Examples: "Adding user authentication", "API endpoint changes", "New input handling"
  - For Non-Git mode: "New implementation review", "Security audit", etc.

### Step 4: Retrieve and Analyze Review Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Record the list of changed files

#### For Non-Git Mode:

1. Collect source code from specified files/directories:
   ```bash
   find <target-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" -o -name "*.java" \) 2>/dev/null
   ```

2. Read each file's content and record as review targets

#### Common:

3. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing review target files

### Step 5: Parallel Security Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Review target file contents (Git mode: diff, Non-Git mode: full files)
- Change summary (from Step 3)
- Related CLAUDE.md contents

**Agent #1: Injection & Input Validation**
- SQL injection, NoSQL injection, command injection, LDAP injection
- XSS (reflected, stored, DOM-based)
- Path traversal and directory traversal
- Template injection (SSTI)
- Unsafe deserialization
- Missing or inadequate input validation and sanitization
- Race conditions and TOCTOU (time-of-check-time-of-use) vulnerabilities

**Agent #2: Authentication & Authorization**
- Broken authentication (weak password policies, missing MFA considerations)
- Broken access control (IDOR, privilege escalation, missing authorization checks)
- Session management flaws (insecure tokens, missing expiration, fixation)
- JWT misuse (algorithm confusion, missing validation, sensitive data in payload)
- Missing CSRF protection

**Agent #3: Data Exposure & Secret Handling**
- Hardcoded secrets, API keys, passwords, tokens in source code
- Sensitive data in logs (PII, credentials, tokens)
- Sensitive data in error messages exposed to users
- Missing encryption for data at rest or in transit
- Insecure storage of sensitive data
- Overly permissive CORS configuration

**Agent #4: Dependency & Configuration Security**
- Dependency configuration risks (pinning to wildcard versions, use of deprecated/archived packages, insecure registry sources). Note: for CVE scanning, recommend running `npm audit` / `govulncheck` separately
- Insecure default configurations
- Missing security headers
- Debug mode or verbose error output enabled in production
- Insecure TLS/SSL settings

**Agent #5: Git History Security Audit** (Git mode only)
- For each changed file from Step 4, check git blame and history:
  ```bash
  git -C <project-path> log -p --follow --max-count=20 -- <file>
  git -C <project-path> blame <file>
  ```
- Look for previously fixed security issues being reintroduced
- Check if security-sensitive code was changed without corresponding test updates
- Verify security patterns established in past commits are maintained

Each agent reports issues in the following format:
```
- File: <file-path>
- Line: <line-number>
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Injection / Auth / Data Exposure / Configuration / History
- CWE: CWE-<number> (if applicable)
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The review target code (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. False positive or theoretical-only risk with no practical exploit path
- **25**: Somewhat confident. Possible issue but heavily context-dependent. May require specific conditions to exploit
- **50**: Moderately confident. Real vulnerability but low severity or requires unusual conditions to exploit
- **75**: Quite confident. Verified vulnerability with a plausible exploit path. Directly affects security posture
- **100**: Absolutely confident. Verified vulnerability that is easily exploitable. Immediate security risk

The agent returns a confidence score (0/25/50/75/100) for each issue.

### Step 7: Validation

For each issue that scored >= 50 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of issues (those scoring >= 50)
- The relevant source code for each issue
- The original agent's reasoning

For each issue, the validation agent must:
1. Re-read the cited code location
2. Confirm the issue is real (not a false positive from the examples in the False Positive section)
3. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED issues from the final report.

### Step 8: Filtering and Report Generation

1. Filter out issues that were REJECTED in Step 7 or scored below 50 (lower threshold than general review since security issues are more critical)

2. Output final report in the following format:

---

## Security Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>

### Critical / High Issues

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Line: L<start>-L<end>
- Impact: <Critical / High>
- Category: <Injection / Auth / Data Exposure / Configuration / History>
- CWE: CWE-<number>
- Confidence: <score>/100

```diff
<relevant code snippet>
```

**Recommendation**: <specific fix suggestion>

### Medium / Low Issues

Issues with confidence 50-74:

(Same format as above)

---

If no issues were found:

### Security Review Results

No security issues found. Checked for injection vulnerabilities, authentication/authorization flaws, data exposure, and configuration security.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Theoretical vulnerabilities with no practical exploit path in this context
- Issues behind multiple layers of existing security controls
- Issues that only apply to different deployment contexts
- Security patterns that are intentionally relaxed for development/testing (if clearly marked)
- Issues that linters, SAST tools, or type checkers would already detect

For Git mode only:
- Existing vulnerabilities (not introduced in this PR)
- Security issues on lines not changed by the user in the PR

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and line links for each issue
- Use TodoWrite tool to track progress
- Focus on actionable findings, not security best practice lectures

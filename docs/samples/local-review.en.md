---
description: Run local code review (works even without a Git repository)
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Code Review

Performs code review on local code. If a Git repository exists, it reviews the diff between branches; otherwise, it reviews the specified files/directories.

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
   find <project-path> -type f -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Review target**: Path(s) to files or directories to review (can be multiple)
   - Examples: `src/`, `internal/mcp/`, `main.go`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of the purpose and background of the changes
  - Examples: "Adding user authentication", "Performance improvements", "Bug fix"
  - For Non-Git mode: "New implementation review", "Code quality check", etc.

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

### Step 5: Parallel Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 3 parallel Sonnet agents (skip Git history related ones)

Pass the following to each agent:
- Review target file contents (Git mode: diff, Non-Git mode: full files)
- Change summary (from Step 3)
- Related CLAUDE.md contents

**Agent #1: CLAUDE.md Compliance Check**
- Verify code follows CLAUDE.md guidelines
- Report violations with specific locations and corresponding CLAUDE.md rules

**Agent #2: Bug Scan**
- Look for obvious bugs in the review target code
- Focus on significant bugs, avoid minor nitpicks

**Agent #3: History Context Analysis** (Git mode only)
- Check git blame and history of changed files:
  ```bash
  git -C <project-path> log -p --follow -- <file>
  git -C <project-path> blame <file>
  ```
- Check for conflicts with past changes
- Verify previously fixed bugs aren't being reintroduced

**Agent #4: Past Commit Analysis** (Git mode only)
- Review past commits on the same files
- Extract relevant notes from past commit messages

**Agent #5: Code Comment Check**
- Review comments in target files
- Verify code follows guidance in comments
- Check handling of TODO and FIXME comments

Each agent reports issues in the following format:
```
- File: <file-path>
- Line: <line-number>
- Issue: <description>
- Reason: <reason> (CLAUDE.md violation / Bug / History context / Comment violation)
```

### Step 6: Confidence Scoring

For each issue found in Step 5, launch a Haiku agent for scoring:

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. False positive that falls apart under light scrutiny, or existing issue
- **25**: Somewhat confident. Might be a real issue, but could be false positive. For style issues, not explicitly stated in CLAUDE.md
- **50**: Moderately confident. Real issue but trivial or unlikely to occur in practice. Low priority within the overall PR
- **75**: Quite confident. Re-verified the issue and confirmed it's likely to occur. Existing approach is insufficient. Directly affects functionality or explicitly stated in CLAUDE.md
- **100**: Absolutely confident. Re-verified the issue and confirmed it will definitely occur. Occurs frequently

### Step 7: Filtering and Report Generation

1. Filter out issues with scores below 80

2. Output final report in the following format:

---

## Code Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>

### Issues Found

If issues were found:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Line: L<start>-L<end>
- Reason: <CLAUDE.md / Bug / History context / Comment>
- Confidence: <score>/100

```diff
<relevant code snippet>
```

---

If no issues were found:

### Code Review Results

No issues found. Checked for bugs and CLAUDE.md compliance.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Things that look like bugs but aren't actually bugs
- Minor nitpicks that a senior engineer wouldn't point out
- Issues that linters, type checkers, or compilers would detect
- General code quality issues not explicitly required by CLAUDE.md
- Issues explicitly disabled by lint ignore comments
- Functional changes directly related to intentional or broad changes

For Git mode only:
- Existing issues (not introduced in this PR)
- Issues on lines not changed by the user in the PR

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and line links for each issue
- Use TodoWrite tool to track progress

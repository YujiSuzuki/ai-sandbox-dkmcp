---
description: Run local performance-focused code review (works even without a Git repository)
description-ja: ローカルパフォーマンスレビューを実行（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Performance Review

Performs performance-focused code review on local code. If a Git repository exists, it reviews the diff between branches; otherwise, it reviews the specified files/directories. Focuses on computational efficiency, memory usage, I/O patterns, and scalability concerns.

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
  - Examples: "Adding data processing pipeline", "Database query changes", "New API endpoint"
  - For Non-Git mode: "Performance audit", "Optimization review", etc.

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

### Step 5: Parallel Performance Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Review target file contents (Git mode: diff, Non-Git mode: full files)
- Change summary (from Step 3)
- Related CLAUDE.md contents

**Agent #1: Algorithmic Complexity & Computation**
- O(n²) or worse algorithms where O(n log n) or O(n) is possible
- Unnecessary nested loops over large datasets
- Redundant computations that could be cached or memoized
- Missing early returns or short-circuit evaluation
- Expensive operations inside hot loops (regex compilation, object creation)
- Unnecessary sorting, searching, or data transformations

**Agent #2: Memory & Resource Management**
- Memory leaks (unclosed resources, missing cleanup, event listener accumulation)
- Unbounded data structures (caches without eviction, arrays that grow indefinitely)
- Large object allocations in hot paths
- Unnecessary data copying (when references/slices would suffice)
- Buffer/pool misuse or missing pooling for frequent allocations
- Goroutine/thread leaks (Go: missing context cancellation, JS: unresolved promises)

**Agent #3: I/O & Database Performance**
- N+1 query problems (fetching related data in loops)
- Queries likely missing indexes (scanning by non-primary-key columns without documented index, queries with multiple unfiltered joins)
- Unnecessary sequential I/O that could be parallelized
- Missing or ineffective caching (repeated identical queries/API calls)
- Large payload transfers (over-fetching data, missing pagination)
- Blocking I/O on main thread / event loop
- Missing connection pooling or pool exhaustion risks

**Agent #4: Concurrency & Scalability**

Scope analysis to the technology stack detected in the project. Do not mix unrelated technology guidance.

Backend-relevant:
- Lock contention and unnecessary synchronization
- Missing concurrency where parallelism would help
- Unbounded concurrency (no worker pools, no rate limiting)
- Shared mutable state without proper synchronization
- Deadlock potential

Frontend-relevant:
- Unnecessary re-renders, missing virtualization for long lists, unoptimized bundle size

**Agent #5: Performance Regression Detection** (Git mode only)
- For each changed file from Step 4, check git blame and history:
  ```bash
  git -C <project-path> log -p --follow --max-count=20 -- <file>
  git -C <project-path> blame <file>
  ```
- Look for previously optimized code being reverted to slower implementations
- Check if performance-critical paths gained additional overhead
- Verify caching and optimization patterns from past commits are maintained

Each agent reports issues in the following format:
```
- File: <file-path>
- Line: <line-number>
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Algorithm / Memory / I/O / Concurrency / Regression
- Estimated scale: <when this becomes a problem, e.g., ">1000 records", "concurrent users >100">
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The review target code (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. False positive or micro-optimization with no measurable impact
- **25**: Somewhat confident. Theoretical performance issue unlikely to matter at current scale
- **50**: Moderately confident. Real inefficiency but impact depends heavily on data volume or usage patterns
- **75**: Quite confident. Verified inefficiency with clear impact at reasonable scale. Better approach is straightforward
- **100**: Absolutely confident. Obvious performance bug (e.g., N+1 query, O(n²) on large dataset). Will cause noticeable degradation

The agent returns a confidence score (0/25/50/75/100) for each issue.

### Step 7: Validation

For each issue that scored >= 75 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of issues (those scoring >= 75)
- The relevant source code for each issue
- The original agent's reasoning

For each issue, the validation agent must:
1. Re-read the cited code location
2. Confirm the issue is real (not a false positive from the examples in the False Positive section)
3. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED issues from the final report.

### Step 8: Filtering and Report Generation

1. Filter out issues that were REJECTED in Step 7 or scored below 75

2. Output final report in the following format:

---

## Performance Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>

### High Impact Issues

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Line: L<start>-L<end>
- Impact: <Critical / High>
- Category: <Algorithm / Memory / I/O / Concurrency / Regression>
- Scale: <when this becomes a problem>
- Confidence: <score>/100

```diff
<relevant code snippet>
```

**Recommendation**: <specific optimization suggestion with expected improvement>

---

If no issues were found:

### Performance Review Results

No performance issues found. Checked for algorithmic complexity, memory management, I/O patterns, and concurrency issues.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Micro-optimizations with negligible real-world impact
- Performance patterns that are idiomatic for the language/framework
- Code that runs only at startup or in initialization paths
- Optimizations that would significantly reduce readability for minimal gain
- Issues in test code or development-only paths
- Patterns that the compiler/runtime already optimizes (loop unrolling, tail calls, etc.)

For Git mode only:
- Existing performance issues (not introduced in this PR)
- Performance characteristics on lines not changed by the user in the PR

## Notes

- Don't run builds, benchmarks, or profilers (this is static analysis only)
- Don't use gh command (this is for local review)
- Always include file and line links for each issue
- Use TodoWrite tool to track progress
- Focus on issues that matter at realistic scale, not theoretical edge cases

---
description: Run local architecture-focused code review (works even without a Git repository)
description-ja: ローカルアーキテクチャレビューを実行（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Architecture Review

Performs architecture-focused code review on local code. If a Git repository exists, it reviews the diff between branches; otherwise, it reviews the specified files/directories. Focuses on design patterns, responsibility separation, dependency management, and code organization.

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
  - Examples: "New module for payment processing", "Refactoring authentication layer", "Adding event system"
  - For Non-Git mode: "Architecture audit", "Design review", etc.

### Step 4: Retrieve and Analyze Review Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Record the list of changed files

3. Read the full content of changed files (not just the diff) to understand imports and dependencies:
   ```bash
   git -C <project-path> show <target-branch>:<file> # for each changed file
   ```

#### For Non-Git Mode:

1. Collect source code from specified files/directories:
   ```bash
   find <target-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" -o -name "*.java" \) 2>/dev/null
   ```

2. Read each file's content and record as review targets

#### Common:

3. Collect project structure information:
   ```bash
   # Directory structure overview
   find <project-path> -type d -maxdepth 4 -not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/vendor/*" 2>/dev/null
   ```

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing review target files

### Step 5: Parallel Architecture Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Review target file contents (Git mode: diff + full changed files with imports, Non-Git mode: full files)
- Change summary (from Step 3)
- Related CLAUDE.md contents
- Project directory structure (from Step 4)

**Agent #1: Responsibility & Cohesion**
- Single Responsibility Principle violations (classes/modules doing too many things)
- God objects or god functions (overly large, doing everything)
- Mixed abstraction levels within the same module
- Business logic leaking into infrastructure layers (handlers, controllers, middleware)
- Presentation logic mixed with data access
- Utility/helper functions that should be domain-specific methods

**Agent #2: Dependencies & Coupling**
- Circular dependencies between packages/modules
- Inappropriate coupling (domain depending on infrastructure, model depending on view)
- Missing dependency injection (hard-coded dependencies that should be injected)
- Violation of dependency inversion (high-level modules depending on low-level details)
- Overly broad imports (importing entire packages for single functions)
- Hidden dependencies (global state, singletons used as implicit dependencies)

**Agent #3: Interface Design & Abstraction**
- Leaky abstractions (implementation details exposed through interfaces)
- Interface pollution (too many methods, not following Interface Segregation Principle)
- Missing abstractions (concrete types used where interfaces would improve flexibility)
- Inconsistent API design (naming, parameter ordering, return types)
- Unclear or misleading function/method signatures
- Missing or improper error types and error handling patterns

**Agent #4: Code Organization & Patterns**
- Violations of established project conventions and patterns
- Inconsistent file/directory organization
- Missing or misused design patterns (factory, strategy, observer, etc.)
- Feature envy (functions that primarily use data from other modules)
- Inappropriate intimacy between modules (accessing internal details)
- Dead code, orphaned interfaces, or unused abstractions

**Agent #5: Architectural Evolution Analysis** (Git mode only)
- For each changed file from Step 4, check git blame and history:
  ```bash
  git -C <project-path> log -p --follow --max-count=20 -- <file>
  git -C <project-path> blame <file>
  ```
- Look for architectural patterns being broken (e.g., previously clean layers getting mixed)
- Check if refactoring efforts from past commits are being undone
- Verify consistency with architectural decisions documented in commit messages
- Identify growing technical debt trends in the changed areas

Each agent reports issues in the following format:
```
- File: <file-path>
- Line: <line-number>
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Responsibility / Dependency / Interface / Organization / Evolution
- Principle: <violated principle, e.g., SRP, DIP, ISP, etc.>
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The review target code (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. Subjective preference, not an actual design problem
- **25**: Somewhat confident. Minor design concern that may be intentional given the project's conventions
- **50**: Moderately confident. Real design issue but contained in scope and unlikely to cause problems soon
- **75**: Quite confident. Verified design violation that will make future changes harder. Clear better approach exists following the project's established patterns
- **100**: Absolutely confident. Severe structural problem (circular dependency, layer violation) that actively hinders development

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

## Architecture Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>

### Structural Issues

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Line: L<start>-L<end>
- Impact: <Critical / High / Medium>
- Category: <Responsibility / Dependency / Interface / Organization / Evolution>
- Principle: <violated principle>
- Confidence: <score>/100

```diff
<relevant code snippet>
```

**Recommendation**: <specific refactoring suggestion with target structure>

---

If no issues were found:

### Architecture Review Results

No architectural issues found. Checked for responsibility separation, dependency management, interface design, and code organization.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Intentional pragmatic trade-offs documented in comments or CLAUDE.md
- Patterns that are idiomatic for the specific framework/library in use
- Simplicity trade-offs that are appropriate for the project's scale
- Prototyping or experimental code clearly marked as such
- Architecture decisions that are consistent with the rest of the codebase (even if not ideal)
- Over-engineering suggestions (adding abstractions that aren't needed yet)

For Git mode only:
- Existing architectural issues (not introduced in this PR)
- Architectural patterns on lines not changed by the user in the PR

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and line links for each issue
- Use TodoWrite tool to track progress
- Focus on actionable design improvements, not theoretical purity
- Respect the project's existing conventions — suggest improvements within those conventions

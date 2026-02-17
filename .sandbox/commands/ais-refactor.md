---
description: Suggest concrete refactoring improvements for code (works even without a Git repository)
description-ja: コードのリファクタリング改善を具体的に提案（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite, Write, Edit]
---

# Local Refactor Suggestion

Analyzes code and suggests concrete refactoring improvements. If a Git repository exists, it analyzes recently changed code; otherwise, it analyzes specified files/directories. Unlike review commands that point out problems, this command provides specific, actionable code transformations.

## Language

Detect the user's language from their previous messages in the conversation. Output all review results, issue descriptions, and recommendations in the same language the user uses. If uncertain, default to English.

## Arguments

User-specified arguments: $ARGUMENTS

Argument interpretation:
- 1st argument: Project path (interactive selection if omitted)
- 2nd argument onwards: Refactoring focus (asked via AskUserQuestion in Step 3 if omitted)

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

2. If `$ARGUMENTS` is empty, use the AskUserQuestion tool to let the user select a project from the found projects

3. Check if the selected project directory has `.git` and determine **Git mode** or **Non-Git mode**:
   ```bash
   test -d <project-path>/.git && echo "GIT_MODE" || echo "NON_GIT_MODE"
   ```

### Step 2: Determine Refactoring Targets

#### For Git Mode:

1. Check the current branch and available branches:
   ```bash
   git -C <project-path> branch -a
   git -C <project-path> branch --show-current
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Base branch**: The branch to compare against (e.g., main, master, develop)
   - **Target branch**: The branch with changes (current branch by default)

#### For Non-Git Mode:

1. Check the file structure within the project:
   ```bash
   find <project-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" \) 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Target files**: Path(s) to files or directories to analyze (can be multiple)
   - Examples: `src/`, `internal/mcp/`, `main.go`, etc.

### Step 3: Refactoring Focus Input

If the 2nd argument onwards is provided, use it as the refactoring focus and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Refactoring focus**: What aspect to prioritize (or "general" for broad analysis)
  - Examples: "Reduce duplication", "Improve readability", "Simplify error handling", "Extract shared logic"
  - For Non-Git mode: "General cleanup", "Improve testability", etc.

### Step 4: Retrieve and Analyze Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Read the full content of changed files (not just the diff) for context

#### For Non-Git Mode:

1. Collect source code from specified files/directories:
   ```bash
   find <target-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" -o -name "*.java" \) 2>/dev/null
   ```

2. Read each file's content and record as targets

#### Common:

3. Read related files that the target code depends on or is depended upon (imports, callers)

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing target files

### Step 5: Parallel Refactoring Analysis

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Target file contents (Git mode: diff + full files, Non-Git mode: full files)
- Refactoring focus (from Step 3)
- Related/dependent file contents
- Related CLAUDE.md contents

**Agent #1: Duplication & Extraction**
- Identify duplicated code blocks across files
- Suggest function/method extraction for repeated patterns
- Identify shared logic that could be consolidated
- Propose helper functions or utilities (only when there are 3+ occurrences)

**Agent #2: Simplification & Readability**
- Identify overly complex conditional logic that can be simplified
- Suggest guard clauses to reduce nesting
- Identify long functions that should be broken down
- Propose renaming for unclear variable/function names
- Suggest idiomatic patterns for the language

**Agent #3: Structure & Responsibility**
- Identify functions/classes doing too many things
- Suggest separation of concerns improvements
- Identify misplaced logic (code in the wrong layer/module)
- Propose interface improvements for better abstraction

**Agent #4: Error Handling & Robustness**
- Identify inconsistent error handling patterns
- Suggest unified error handling approaches
- Identify swallowed errors or missing error propagation
- Propose error type consolidation

**Agent #5: Evolution Analysis** (Git mode only)
- Analyze git history for code churn patterns:
  ```bash
  git -C <project-path> log --oneline --follow --max-count=20 -- <file>
  git -C <project-path> log -p --follow --max-count=20 -- <file>
  ```
- Identify code that has been repeatedly modified (hotspots)
- Suggest stabilizing refactors based on change patterns
- Identify temporary workarounds that are now permanent

Each agent reports suggestions in the following format:
```
- Target: <file-path>
- Lines: L<start>-L<end>
- Type: Extraction / Simplification / Restructure / Error Handling / Stabilization
- Summary: <brief description>
- Before: <current code snippet>
- After: <proposed code snippet>
- Rationale: <why this improves the code>
```

### Step 6: Impact Scoring (Batch)

Collect ALL suggestions from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of suggestions from all agents
- The target source code (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No value. Cosmetic change or subjective preference with no measurable improvement
- **25**: Minor improvement. Slightly cleaner but doesn't reduce complexity or improve maintainability meaningfully
- **50**: Moderate improvement. Reduces some duplication or improves readability, but scope is small
- **75**: Significant improvement. Clearly reduces complexity, improves testability, or eliminates a maintenance burden. Safe to apply
- **100**: Critical improvement. Eliminates a major source of bugs or maintenance cost. Transformation is clearly correct and safe

The agent returns an impact score (0/25/50/75/100) for each suggestion.

### Step 7: Validation

For each suggestion that scored >= 50 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of suggestions (those scoring >= 50)
- The relevant source code for each suggestion
- The original agent's reasoning

For each suggestion, the validation agent must:
1. Re-read the cited code location
2. Confirm the suggestion is valid (not a non-suggestion from the Non-Suggestions section)
3. Verify the proposed transformation preserves existing behavior
4. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED suggestions from the final output.

### Step 8: Filtering and Output

1. Filter out suggestions that were REJECTED in Step 7 or scored below 50 in Step 6

2. Sort remaining suggestions by score (highest first)

3. Use the AskUserQuestion tool to confirm:
   - **Output mode**: Apply changes to files, or display suggestions in chat only
   - If applying changes, confirm which suggestions to apply

4. Output final report:

---

## Refactoring Suggestions

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Focus**: <user-provided-focus>

### High Impact (Score >= 75)

**Suggestion 1**: <Brief description>
- File: `<file-path>`
- Lines: L<start>-L<end>
- Type: <Extraction / Simplification / Restructure / Error Handling / Stabilization>
- Impact: <score>/100

Before:
```<language>
<current code>
```

After:
```<language>
<proposed code>
```

**Rationale**: <why this is an improvement>

### Moderate Impact (Score 50-74)

(Same format as above)

### Summary

| File | Suggestions | Avg Impact | Types |
|------|------------|-----------|-------|
| <file> | <count> | <score>/100 | Extraction, Simplification, ... |

---

If no meaningful refactoring suggestions:

### Refactoring Suggestions

No significant refactoring opportunities found. The analyzed code is clean and well-structured.

---

## Non-Suggestions (Consider in Steps 5, 6, and 7)

The following should NOT be suggested:

- Renaming that is purely stylistic with no clarity improvement
- Premature abstractions for code with only 1-2 occurrences
- Adding design patterns for the sake of patterns
- Refactoring stable code that works fine and isn't being modified
- Changes that would require modifying many callers without clear benefit
- Moving code between files without reducing coupling
- Adding intermediate variables that don't improve readability

For Git mode only:
- Refactoring code outside the changed files (unless directly related)
- Suggestions that conflict with the apparent intent of the changes

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local refactoring)
- Always include file and line references for each suggestion
- Use TodoWrite tool to track progress
- Every suggestion MUST include concrete before/after code (not just descriptions)
- Ensure suggested changes preserve existing behavior (no functional changes unless explicitly requested)
- Respect the project's existing patterns and conventions

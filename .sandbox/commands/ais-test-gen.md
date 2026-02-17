---
description: Generate tests for changed or specified code (works even without a Git repository)
description-ja: 変更コードに対するテストを自動生成（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite, Write, Edit]
---

# Local Test Generation

Generates test cases for local code. If a Git repository exists, it generates tests for changed code between branches; otherwise, it generates tests for specified files/directories. Focuses on meaningful tests that exercise real code paths, covering edge cases, error handling, and behavioral contracts.

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

2. If `$ARGUMENTS` is empty, use the AskUserQuestion tool to let the user select a project from the found projects

3. Check if the selected project directory has `.git` and determine **Git mode** or **Non-Git mode**:
   ```bash
   test -d <project-path>/.git && echo "GIT_MODE" || echo "NON_GIT_MODE"
   ```

### Step 2: Determine Test Targets

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
   - **Target files**: Path(s) to files or directories to generate tests for (can be multiple)
   - Examples: `src/`, `internal/mcp/`, `main.go`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of what the code does and what behavior to test
  - Examples: "User authentication with JWT", "File parsing with error handling", "API endpoint for CRUD operations"
  - For Non-Git mode: "Core business logic", "Utility functions", etc.

### Step 4: Retrieve and Analyze Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Record the list of changed files (exclude test files from diff, but read existing tests)

#### For Non-Git Mode:

1. Collect source code from specified files/directories:
   ```bash
   find <target-path> -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.rs" -o -name "*.java" \) 2>/dev/null
   ```

2. Read each file's content and record as targets

#### Common:

3. Detect the project's test framework and conventions:
   - Find existing test files and examine their structure:
     ```bash
     find <project-path> -type f \( -name "*_test.go" -o -name "*.test.js" -o -name "*.test.ts" -o -name "*.spec.js" -o -name "*.spec.ts" -o -name "test_*.py" -o -name "*_test.py" \) 2>/dev/null | head -20
     ```
   - Read 2-3 existing test files to understand:
     - Test framework (Jest, Go testing, pytest, etc.)
     - Naming conventions
     - Helper functions and fixtures
     - Mocking patterns

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing target files

### Step 5: Parallel Test Generation

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Target file contents (Git mode: diff + full changed files, Non-Git mode: full files)
- Change summary (from Step 3)
- Existing test examples (from Step 4)
- Test framework and conventions detected
- Related CLAUDE.md contents

**Agent #1: Happy Path Tests**
- Generate tests for normal/expected usage patterns
- Cover the main functionality of each changed/target function
- Test typical input values and expected outputs
- Test return values, side effects, and state changes

**Agent #2: Edge Case & Boundary Tests**
- Generate tests for boundary conditions (empty input, zero, nil/null, max values)
- Test off-by-one scenarios
- Test with unusual but valid inputs
- Test type coercion and format edge cases

**Agent #3: Error Handling Tests**
- Generate tests for error paths and failure modes
- Test invalid inputs and expected error responses
- Test timeout and resource exhaustion scenarios
- Test graceful degradation behavior
- Verify error messages are meaningful

**Agent #4: Integration Point Tests**
- Generate tests for function interactions and dependencies
- Test with mocked dependencies where appropriate
- Test data flow between components
- Test interface contracts (inputs/outputs match expectations)

**Agent #5: Regression & History-Based Tests** (Git mode only)
- Analyze git history for previously fixed bugs:
  ```bash
  git -C <project-path> log -p --follow --max-count=20 -- <file>
  ```
- Generate tests that would catch known regressions
- Generate tests based on patterns seen in past bug fixes

Each agent outputs test code in the following format:
```
- Target: <source-file-path>
- Test file: <proposed-test-file-path>
- Tests:
  <complete, runnable test code>
- Rationale: <brief explanation of what each test verifies>
```

### Step 6: Test Quality Scoring (Batch)

Collect ALL generated tests from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of generated tests from all agents
- The target source code
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: Useless test. Tests implementation details, duplicates logic, or tests language features
- **25**: Low value. Tests obvious behavior that is unlikely to break. Tautological test
- **50**: Moderate value. Tests a real scenario but coverage overlap with other tests or limited additional confidence
- **75**: High value. Tests meaningful behavior, catches real bugs, uses proper assertions. Would catch a regression
- **100**: Essential test. Tests critical behavior, covers a known edge case or past bug. Must have for confidence

The agent returns a quality score (0/25/50/75/100) for each test.

### Step 7: Validation

For each test that scored >= 50 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of tests (those scoring >= 50)
- The relevant source code for each test
- The original agent's reasoning

For each test, the validation agent must:
1. Re-read the target source code
2. Confirm the test is meaningful (not an anti-pattern from the Anti-Patterns section)
3. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED tests from the final output.

### Step 8: Filtering and Output

1. Filter out tests that were REJECTED in Step 7 or scored below 50 in Step 6

2. Group tests by target file and merge into coherent test files that follow project conventions. When multiple agents generate tests for the same function, deduplicate by keeping the higher-scored test and removing redundant ones

3. Use the AskUserQuestion tool to confirm:
   - **Output mode**: Write test files to disk, or display in chat only
   - If writing to disk, confirm file paths

4. Output final report:

---

## Test Generation Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>
**Test framework**: <detected framework>

### Generated Tests

**File: `<test-file-path>`**
- Tests generated: <count>
- Coverage focus: <what aspects are covered>
- Average confidence: <score>/100

```<language>
<complete test code>
```

### Test Summary

| Target File | Tests Generated | Avg Confidence | Categories Covered |
|------------|----------------|----------------|-------------------|
| <file> | <count> | <score>/100 | Happy path, Edge cases, ... |

### Not Generated (Rationale)

- <file-or-function>: <reason why tests were not generated (e.g., already well tested, pure configuration, trivial getter)>

---

If no meaningful tests could be generated:

### Test Generation Results

No meaningful tests generated. The target code is either already well-tested, purely declarative, or too tightly coupled to external systems for unit testing.

---

## Anti-Patterns (Consider in Steps 5, 6, and 7)

The following test patterns should be avoided:

- Tests that duplicate the implementation logic (testing `add(a,b)` by checking `a+b`)
- Tests that only verify mock behavior, not real code
- Tests that are tightly coupled to internal implementation details
- Tests for trivial getters/setters with no logic
- Tests that test the programming language itself (e.g., "array length works")
- Snapshot tests where the snapshot is just the current output with no verification of correctness
- Tests with no meaningful assertions (only checking "no error thrown")

## Notes

- Follow the project's existing test conventions exactly (framework, naming, structure)
- Generated tests must be complete and runnable without modification
- Don't generate tests for code that is already well-tested (check existing coverage)
- Don't use gh command (this is for local test generation)
- Always include target file references for each test
- Use TodoWrite tool to track progress
- Prefer testing behavior over implementation details

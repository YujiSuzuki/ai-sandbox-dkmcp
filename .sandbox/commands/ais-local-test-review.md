---
description: Run local test quality review (works even without a Git repository)
description-ja: ローカルテスト品質レビューを実行（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Test Review

Performs quality-focused review of existing test code. If a Git repository exists, it reviews tests related to the diff between branches; otherwise, it reviews tests in specified files/directories. Focuses on whether tests are meaningful (actually verifying intended behavior), detecting anti-patterns, assessing robustness, and identifying coverage gaps.

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

1. Find test files within the project:
   ```bash
   find <project-path> -type f \( -name "*_test.go" -o -name "*.test.js" -o -name "*.test.ts" -o -name "*.test.jsx" -o -name "*.test.tsx" -o -name "*.spec.js" -o -name "*.spec.ts" -o -name "test_*.py" -o -name "*_test.py" -o -name "*Test.java" -o -name "*_test.rs" \) 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Review target**: Path(s) to test files or directories to review (can be multiple)
   - Examples: `tests/`, `internal/mcp/`, `src/__tests__/`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of what the tests cover or the review focus
  - Examples: "Authentication tests", "API endpoint tests", "Data validation tests"
  - For Non-Git mode: "Test suite quality audit", "Test coverage review", etc.

### Step 4: Retrieve and Analyze Review Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Identify test files among changed files AND the source files they test

3. Read both the test files and corresponding source files (tests cannot be reviewed without understanding the code they test)

#### For Non-Git Mode:

1. Collect test files from specified paths:
   ```bash
   find <target-path> -type f \( -name "*_test.go" -o -name "*.test.js" -o -name "*.test.ts" -o -name "*.test.jsx" -o -name "*.test.tsx" -o -name "*.spec.js" -o -name "*.spec.ts" -o -name "test_*.py" -o -name "*_test.py" -o -name "*Test.java" -o -name "*_test.rs" \) 2>/dev/null
   ```

2. For each test file, locate and read the corresponding source file it tests

#### Common:

3. Detect the project's test framework and conventions:
   - Read 2-3 test files to understand:
     - Test framework (Jest, Go testing, pytest, etc.)
     - Naming conventions and patterns
     - Helper functions, fixtures, and factories
     - Mocking approach and common utilities

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing review target files

### Step 5: Parallel Test Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Test file contents (Git mode: diff + full test files, Non-Git mode: full test files)
- Corresponding source file contents (the code being tested)
- Change summary (from Step 3)
- Test framework and conventions detected (from Step 4)
- Related CLAUDE.md contents

**Agent #1: Test Effectiveness (Are tests meaningful?)**

Check whether each test actually verifies the intended behavior of the code:

- Tests that pass but would NOT fail if the code under test broke (weak assertions)
- Tests that verify return type/shape but not actual correctness of values
- Tests where the assertion does not match the test name/description (testing something different from what it claims)
- Tests that only check "no error thrown" without verifying the result
- Tests missing assertions on critical side effects (database writes, API calls, state changes)
- Tests that verify happy path output but don't confirm the output is actually correct (e.g., checking `result !== null` instead of checking the actual expected value)
- Tests where removing or breaking the code under test would still pass

**Agent #2: Anti-Pattern Detection (Are there meaningless tests?)**

Detect test patterns that provide little or no value:

- Tests that duplicate implementation logic (e.g., testing `add(a,b)` by asserting `a+b`)
- Tests that test the programming language itself (e.g., "array push adds element")
- Tests for trivial getters/setters with no logic
- Snapshot tests where the snapshot is just the current output with no verification of correctness
- Tests that only verify mock behavior, never exercising real code paths
- Copy-pasted test blocks with minimal variation that should be parameterized
- Tests with misleading names that create false confidence
- Tests that catch errors and ignore them (empty catch blocks in tests)

**Agent #3: Test Robustness & Maintainability**

Assess whether tests are reliable and maintainable:

- Flaky test indicators: time-dependent logic, `setTimeout`/`sleep`, random values without seed
- External dependency without isolation: network calls, file system access, database queries in unit tests
- Test order dependency: shared mutable state between tests, missing setup/teardown
- Brittle assertions: asserting on full error messages, hardcoded timestamps, environment-specific paths
- Over-mocking: so many mocks that the test no longer resembles real execution
- Test readability: unclear arrange/act/assert structure, overly complex setup hiding the actual test intent
- Magic numbers/strings without explanation of their significance

**Agent #4: Coverage Gap Analysis**

Identify important behaviors that lack test coverage:

- Error handling paths with no tests (what happens when the function fails?)
- Boundary conditions not tested: empty input, nil/null, zero, max values, empty arrays/maps
- Important business logic branches without tests
- Edge cases implied by the code (e.g., code handles a special case but no test exercises it)
- Integration points between components with no contract tests
- Concurrency-related code without concurrent tests (race conditions, deadlocks)
- Configuration/environment variations not covered

**Agent #5: Regression Test Analysis** (Git mode only)

Analyze git history to evaluate test coverage of past issues. Use the source files and test files identified in Step 4:

- Check git history for bug fixes in the changed source files:
  ```bash
  git -C <project-path> log --all --grep="fix" --grep="bug" --grep="hotfix" --oneline --max-count=30 -- <source-file>
  git -C <project-path> log -p --follow --max-count=20 -- <test-file>
  ```
- Past bug fixes without corresponding regression tests
- Tests added for bug fixes that are too narrow (only test the exact reported case, not the general class of bug)
- Deleted or weakened tests that previously caught issues
- Changed source code where the test was not updated to match

Each agent reports issues in the following format:
```
- File: <test-file-path>
- Line: <line-number>
- Source: <corresponding-source-file-path> (if applicable)
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Effectiveness / Anti-Pattern / Robustness / Coverage Gap / Regression
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The test code and corresponding source code
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. Subjective test style preference, not an actual quality problem
- **25**: Somewhat confident. Minor test quality concern that may be intentional (e.g., deliberately simple test for a critical path)
- **50**: Moderately confident. Real test quality issue but low risk — the test still provides some value
- **75**: Quite confident. The test has a clear deficiency — it either fails to catch real bugs, gives false confidence, or will cause maintenance pain. A better approach is clear
- **100**: Absolutely confident. The test is actively harmful — it passes when it should fail, masks real bugs, or blocks legitimate code changes with false failures

The agent returns a confidence score (0/25/50/75/100) for each issue.

### Step 7: Validation

For each issue that scored >= 75 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of issues (those scoring >= 75)
- The relevant test code AND source code for each issue
- The original agent's reasoning

For each issue, the validation agent must:
1. Re-read both the test code and the source code it tests
2. Confirm the issue is real (not a false positive from the examples in the False Positive section)
3. For Effectiveness issues (Agent #1): mentally simulate "if I broke the source code, would this test actually catch it?"
4. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED issues from the final report.

### Step 8: Filtering and Report Generation

1. Filter out issues that were REJECTED in Step 7 or scored below 75

2. Output final report in the following format:

---

## Test Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>
**Test framework**: <detected framework>

### Test Quality Issues

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- Test file: `<test-file-path>`
- Line: L<start>-L<end>
- Source file: `<source-file-path>` (if applicable)
- Impact: <Critical / High / Medium>
- Category: <Effectiveness / Anti-Pattern / Robustness / Coverage Gap / Regression>
- Confidence: <score>/100

```<language>
<relevant test code snippet>
```

**Problem**: <what is wrong and why it matters>
**Recommendation**: <specific improvement with example code if helpful>

---

### Summary

| Category | Issues Found | Critical | High | Medium |
|----------|-------------|----------|------|--------|
| Effectiveness | <count> | <count> | <count> | <count> |
| Anti-Pattern | <count> | <count> | <count> | <count> |
| Robustness | <count> | <count> | <count> | <count> |
| Coverage Gap | <count> | <count> | <count> | <count> |
| Regression | <count> | <count> | <count> | <count> |

---

If no issues were found:

### Test Review Results

No test quality issues found. Tests are meaningful, robust, and provide good coverage of the target code.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Intentionally simple tests for critical paths (sometimes a basic "does not crash" test is valuable)
- Test patterns that are idiomatic for the specific framework (e.g., Jest snapshot tests used appropriately)
- Tests that appear to duplicate logic but actually serve as documentation of expected behavior (i.e., the test name explicitly describes the expected behavior, and the assertion verifies a concrete expected value rather than re-computing it)
- Integration/E2E tests that intentionally test through real dependencies
- Tests in prototype/experimental code clearly marked as such
- Test utilities and helpers that are not tests themselves
- Tests that follow conventions established in CLAUDE.md or project documentation

For Git mode only:
- Existing test quality issues not introduced in this PR
- Test patterns on lines not changed by the user in the PR

## Notes

- Always read both the test file AND the source file it tests — tests cannot be reviewed in isolation
- Don't run builds or test suites (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and line links for each issue
- Use TodoWrite tool to track progress
- Focus on actionable improvements, not style preferences
- Respect the project's existing test conventions — suggest improvements within those conventions
- A test that exercises real code and has meaningful assertions is always better than high coverage with weak assertions

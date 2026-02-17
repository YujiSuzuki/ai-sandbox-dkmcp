---
description: Review AI command/prompt files for quality and consistency (works even without a Git repository)
description-ja: AIコマンド／プロンプトファイルの品質・一貫性をレビュー（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Prompt Review

Reviews AI command/prompt files (.md) for quality, consistency, and effectiveness. If a Git repository exists, it reviews the diff between branches; otherwise, it reviews the specified files/directories. Focuses on prompt design, agent orchestration, instruction clarity, and cross-command consistency.

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

1. Search for command/prompt files within the project:
   ```bash
   find <project-path> -type f -name "*.md" \( -path "*commands*" -o -path "*prompts*" -o -path "*.claude/*" -o -path "*.sandbox/*" \) 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Review target**: Path(s) to files or directories to review (can be multiple)
   - Examples: `.sandbox/commands/`, `.claude/commands/`, `prompts/`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of the purpose and background of the changes
  - Examples: "New review command for security", "Updated agent configuration", "Added prompt template"
  - For Non-Git mode: "Prompt quality audit", "Command consistency check", etc.

### Step 4: Retrieve and Analyze Review Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Record the list of changed files (focus on .md command/prompt files)

#### For Non-Git Mode:

1. Collect command/prompt files from specified directories:
   ```bash
   find <target-path> -type f -name "*.md" -not -name "README*" -not -name "CHANGELOG*" -not -name "CONTRIBUTING*" -not -name "LICENSE*" 2>/dev/null
   ```

2. Read each file's content and record as review targets

#### Common:

3. Collect ALL related command/prompt files in the same directory (even unchanged ones) for cross-command consistency checking

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing review target files

### Step 5: Parallel Prompt Review Execution

Launch 4 parallel Sonnet agents.

Pass the following to each agent:
- Review target file contents (Git mode: diff + full files, Non-Git mode: full files)
- ALL sibling command files in the same directory (for consistency checking)
- Change summary (from Step 3)
- Related CLAUDE.md contents

**Agent #1: Instruction Clarity & Completeness**
- Ambiguous or vague instructions that an AI could misinterpret
- Missing steps or gaps in the execution flow
- Unclear preconditions or assumptions
- Steps that reference undefined variables or unavailable tools
- Missing error handling guidance (what to do when a step fails)
- Instructions that conflict with each other within the same file
- Overly complex steps that should be broken down

**Agent #2: Agent Orchestration & Design**
- Agent role overlap (multiple agents checking the same thing)
- Gaps in agent coverage (aspects that no agent checks)
- Inappropriate agent model selection (Sonnet vs Haiku for the task complexity)
- Missing or unclear information passed to agents
- Agent output format inconsistencies
- Scoring criteria that are ambiguous or hard to apply consistently
- Threshold values that are too aggressive or too lenient

**Agent #3: Cross-Command Consistency**
- Inconsistent YAML front matter structure (description, argument-hint, allowed-tools)
- Inconsistent step numbering or naming across commands in shared infrastructure steps (Steps 1-4, Step 6-8, YAML front matter, scoring criteria). Domain-specific steps (Step 5 agent definitions, report sections) may intentionally differ.
- Shared infrastructure steps (project selection, git detection, scoring, validation) that differ unnecessarily between commands
- Inconsistent report formats across commands
- Different false positive criteria that should be aligned
- Inconsistent terminology or phrasing
- Missing fields that exist in sibling commands (e.g., description-ja)

**Agent #4: Effectiveness & False Positive Risk**
- Instructions likely to produce excessive false positives
- Scoring criteria that would pass obvious non-issues
- False positive examples that are too broad (filtering real issues) or too narrow (missing common false positives)
- Review focus areas that overlap with linters/CI (wasted effort)
- Missing focus areas that are important for the command's stated purpose
- Unrealistic expectations for what static analysis can detect

Each agent reports issues in the following format:
```
- File: <file-path>
- Section: <step or section name>
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Clarity / Orchestration / Consistency / Effectiveness
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The review target code (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. Subjective style preference or nitpick
- **25**: Somewhat confident. Minor wording issue that is unlikely to cause misinterpretation
- **50**: Moderately confident. Real issue but impact on AI execution quality is uncertain
- **75**: Quite confident. Verified issue that will likely cause incorrect behavior, inconsistency, or false positives/negatives in review results
- **100**: Absolutely confident. Clear defect (conflicting instructions, missing critical step, broken reference) that will definitely cause failure

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

## Prompt Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>
**Files reviewed**: <count> command files (<count> changed + <count> siblings for consistency)

### Issues Found

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Section: <step or section name>
- Impact: <Critical / High / Medium>
- Category: <Clarity / Orchestration / Consistency / Effectiveness>
- Confidence: <score>/100

```markdown
<relevant excerpt>
```

**Recommendation**: <specific improvement suggestion>

---

If no issues were found:

### Prompt Review Results

No issues found. Checked for instruction clarity, agent orchestration, cross-command consistency, and effectiveness.

---

## False Positive Examples (Consider in Steps 5 and 6)

The following should be excluded as false positives:

- Writing style preferences (as long as instructions are clear)
- Minor formatting differences that don't affect AI interpretation
- Intentional differences between commands (each command type has different needs)
- Suggestions to add features beyond the command's stated scope
- Theoretical edge cases that are extremely unlikely to occur in practice

For Git mode only:
- Existing issues (not introduced in this PR)
- Issues in files not changed by the user in the PR (except cross-command consistency which may reference unchanged siblings)

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and section references for each issue
- Use TodoWrite tool to track progress
- Read ALL sibling command files for consistency checks, not just the changed ones
- Focus on issues that would affect AI execution quality, not writing style

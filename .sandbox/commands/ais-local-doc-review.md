---
description: Review documentation for accuracy, consistency, and clarity (works even without a Git repository)
description-ja: ドキュメントの正確性・一貫性・わかりやすさをレビュー（Git リポジトリがなくても動作）
argument-hint: [project-path] [change summary]
allowed-tools: [Read, Glob, Grep, Bash(git:*), Bash(ls:*), Bash(find:*), Task, AskUserQuestion, TodoWrite]
---

# Local Documentation Review

Reviews documentation files for accuracy, consistency, and clarity. If a Git repository exists, it reviews changed docs between branches; otherwise, it reviews specified files/directories. Checks that documentation matches the actual code, cross-references are valid, and content is clear and well-structured.

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

1. Search for documentation files within the project:
   ```bash
   find <project-path> -type f \( -name "*.md" -o -name "*.rst" -o -name "*.txt" -o -name "*.adoc" \) -not -path "*/node_modules/*" -not -path "*/.git/*" 2>/dev/null | head -50
   ```

2. Use the AskUserQuestion tool to confirm:
   - **Review target**: Path(s) to files or directories to review (can be multiple)
   - Examples: `docs/`, `README.md`, `CLAUDE.md`, etc.

### Step 3: Change Summary Input

If the 2nd argument onwards is provided, use it as the change summary and skip AskUserQuestion.

Only if the 2nd argument is not provided, use the AskUserQuestion tool to get:
- **Change summary**: A brief explanation of the purpose and background of the documentation
  - Examples: "Updated API docs after endpoint changes", "New setup guide", "Architecture docs update"
  - For Non-Git mode: "Documentation audit", "Clarity check", etc.

### Step 4: Retrieve and Analyze Review Targets

#### For Git Mode:

1. Get the diff between base branch and target branch:
   ```bash
   git -C <project-path> diff <base-branch>...<target-branch> --name-only
   git -C <project-path> diff <base-branch>...<target-branch>
   git -C <project-path> log <base-branch>...<target-branch> --oneline
   ```

2. Record the list of changed files (focus on documentation files, but also note code changes that may affect docs)

#### For Non-Git Mode:

1. Collect documentation files from specified directories:
   ```bash
   find <target-path> -type f \( -name "*.md" -o -name "*.rst" -o -name "*.txt" -o -name "*.adoc" \) 2>/dev/null
   ```

2. Read each file's content and record as review targets

#### Common:

3. Collect the source code that documentation references:
   - Read files referenced in code examples or file paths mentioned in docs
   - Read function/API signatures mentioned in docs

4. Collect related CLAUDE.md files:
   - CLAUDE.md at project root
   - CLAUDE.md in directories containing review target files

5. If the project has multiple language versions (e.g., README.md and README.ja.md), collect all language variants of reviewed docs

### Step 5: Parallel Documentation Review Execution

**For Git mode**: Launch 5 parallel Sonnet agents
**For Non-Git mode**: Launch 4 parallel Sonnet agents (skip Agent #5)

Pass the following to each agent:
- Documentation file contents (Git mode: diff + full files, Non-Git mode: full files)
- Referenced source code contents
- Change summary (from Step 3)
- Related CLAUDE.md contents
- Code change summary (Git mode only: list of changed code files and their diff from Step 4, for Agent #5's drift detection)

**Agent #1: Accuracy & Freshness**
- Code examples that don't match actual implementation
- File paths or function names that don't exist or have been renamed
- Configuration examples with incorrect keys, values, or structure
- Version numbers or dependency versions that are outdated
- Instructions that reference removed or changed features
- CLI commands or flags that are incorrect

**Agent #2: Completeness & Structure**
- Missing sections expected in this type of document (e.g., installation, usage, API reference)
- Incomplete instructions (steps that skip important details)
- Missing prerequisites or environment requirements
- Tables of contents that don't match actual sections
- Broken internal links or anchors
- Missing examples for complex concepts

**Agent #3: Clarity & Readability**
- Ambiguous instructions that could be interpreted multiple ways
- Jargon or acronyms used without explanation
- Overly long paragraphs or walls of text
- Missing context for who the audience is
- Inconsistent formatting (heading levels, list styles, code block languages)
- Unclear ordering of steps or sections

**Agent #4: Cross-Document Consistency**
- Contradicting information between different docs
- Duplicated content that has diverged
- Inconsistent terminology across documents
- Language variant mismatches (e.g., README.md says X but README.ja.md says Y)
- CLAUDE.md rules not reflected in related documentation
- Cross-references between docs that are broken or outdated

**Agent #5: Documentation Drift Detection** (Git mode only)
- For each changed code file from Step 4, check git history for changes that should have triggered doc updates:
  ```bash
  git -C <project-path> log -p --follow --max-count=20 -- <code-file>
  ```
- Identify code changes in the same branch that lack corresponding documentation updates
- Check if renamed functions/files are reflected in docs
- Verify that new features added in the branch are documented

Each agent reports issues in the following format:
```
- File: <file-path>
- Section: <heading or line range>
- Issue: <description>
- Impact: Critical / High / Medium / Low
- Category: Accuracy / Completeness / Clarity / Consistency / Drift
```

### Step 6: Confidence Scoring (Batch)

Collect ALL issues from Step 5 and pass them to a **single Haiku agent** for batch scoring.

Provide the agent with:
- The full list of issues from all agents
- The review target documentation (diff or full files)
- The scoring criteria below

Scoring criteria (pass these criteria directly to the agent):
- **0**: No confidence. Subjective style preference or matter of taste
- **25**: Somewhat confident. Minor wording issue that readers would likely understand anyway
- **50**: Moderately confident. Real issue but unlikely to cause confusion for the target audience
- **75**: Quite confident. Verified issue that will mislead or confuse readers. Incorrect information or broken reference
- **100**: Absolutely confident. Factually wrong information, broken instructions that will fail, or critical missing content

The agent returns a confidence score (0/25/50/75/100) for each issue.

### Step 7: Validation

For each issue that scored >= 75 in Step 6, launch a **single Sonnet agent** to re-verify.

The validation agent receives:
- The filtered list of issues (those scoring >= 75)
- The relevant documentation and referenced source code for each issue
- The original agent's reasoning

For each issue, the validation agent must:
1. Re-read the cited documentation location
2. Confirm the issue is real (not a false positive from the examples in the False Positive section)
3. Return: **CONFIRMED** or **REJECTED** with a one-line reason

Remove REJECTED issues from the final report.

### Step 8: Filtering and Report Generation

1. Filter out issues that were REJECTED in Step 7 or scored below 75

2. Output final report in the following format:

---

## Documentation Review Results

**Project**: <project-path>
**Mode**: Git mode / Non-Git mode
**Review target**:
  - Git mode: <base-branch>...<target-branch>
  - Non-Git mode: <target-files-or-directories>
**Change summary**: <user-provided-summary>
**Files reviewed**: <count> documentation files

### Issues Found

Issues with confidence >= 75:

**Issue 1**: <Brief description of the issue>
- File: `<file-path>`
- Section: <heading or line range>
- Impact: <Critical / High / Medium>
- Category: <Accuracy / Completeness / Clarity / Consistency / Drift>
- Confidence: <score>/100

```markdown
<relevant excerpt>
```

**Recommendation**: <specific improvement suggestion>

---

If no issues were found:

### Documentation Review Results

No issues found. Checked for accuracy, completeness, clarity, cross-document consistency, and documentation drift.

---

## False Positive Examples (Consider in Steps 5, 6, and 7)

The following should be excluded as false positives:

- Writing style preferences (as long as content is clear and correct)
- Minor formatting inconsistencies that don't affect readability
- Intentional simplification in docs (not every code detail needs to be documented), except when complex concepts lack any explanation or example
- Documentation for features that are intentionally not yet implemented (if clearly marked as planned)
- Differences between language variants that are intentional localization choices (not translation errors)

For Git mode only:
- Existing issues (not introduced in this PR)
- Documentation drift for code changes outside this PR's scope

## Notes

- Don't run builds or type checks (those are run separately in CI)
- Don't use gh command (this is for local review)
- Always include file and section references for each issue
- Use TodoWrite tool to track progress
- Actually read the source code referenced in documentation to verify accuracy
- For multilingual docs, compare content parity, not literal translation quality

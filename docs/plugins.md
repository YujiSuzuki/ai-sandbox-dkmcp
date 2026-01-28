# Using Plugins in Multi-Project Workspaces

[← Back to README.md](../README.md)

**Note:** This document is for Claude Code only. Gemini Code Assist doesn't support this.

**Note:** This explanation applies when each project is managed as an independent Git repository.
If you're using a monorepo, standard plugin usage works fine. If you're using a multi-repo structure (independent repositories), some considerations apply, and this document explains them.

## Prerequisites

Assume you have multiple projects in your workspace:

```
workspace/
├── your-apps/           # API + React Web (Node.js)
├── your-apps-ios/       # iOS App (Swift)
└── your-other-projects/ # other
```

## Multi-Repo (Independent Repositories) Structure

Since each project has its own **independent Git repository**, PRs and branches exist independently per project:

```
workspace/
├── your-apps/           # Independent Git repository
│   ├── .git/
│   ├── main, develop branches
│   ├── PRs (your-apps only)
│   └── commit history
│
├── your-apps-ios/       # Independent Git repository
│   ├── .git/
│   ├── main, develop branches
│   ├── PRs (your-apps-ios only)
│   └── commit history
│
└── your-other-projects/       # Independent Git repository
    ├── .git/
    ├── main, develop branches
    ├── PRs (your-other-projects only)
    └── commit history

```

## Critical Prerequisite: Plugins Only Work Within Project Directories

> [!IMPORTANT]
> [Claude Code plugins](https://github.com/anthropics/claude-code/blob/main/plugins/README.md) (`/code-review`, etc.) **only work within the project directory where you execute them**:
>
> ```
> /workspace/          ← running /code-review here won't work
>   ├── your-apps/     ← cd here first, then run /code-review
>   ├── your-apps-ios/ ← cd here first, then run /code-review
>   └── your-other-project/       ← cd here first, then run /code-review
> ```


**In other words:**
- Review `your-apps` code: run `/code-review` in `your-apps` directory → reviews `your-apps` PRs/branches only
- Review `your-apps-ios` code: run `/code-review` in `your-apps-ios` directory → reviews `your-apps-ios` PRs/branches only
- Each project's plugins target only that project's code

## Monorepo vs Multi-Repo

**Monorepo structure:**
```bash
# Can run /code-review from workspace root
/workspace /code-review → can review all projects' code
```

**Multi-Repo structure:**
```bash
# Must run /code-review in each project directory
cd /workspace/your-apps && /code-review → reviews your-apps code only
cd /workspace/your-apps-ios && /code-review → reviews your-apps-ios code only
```

## Workaround for Multi-Repo (Independent Repositories)

Using the `/code-review` plugin as an example:


1. **Install the plugin as usual**

See the [official page](https://github.com/anthropics/claude-code/blob/main/plugins/README.md) for installation instructions.

> [!TIP]
>    ```bash
>    claude --help
>    claude plugin --help
>    claude plugin install --help
>    ```

Example:

```
node@671e8b3485a2:/workspace$ claude plugin install code-review
Installing plugin "code-review"...
✔ Successfully installed plugin: code-review@claude-plugins-official (scope: user)
node@671e8b3485a2:/workspace$

```

Verify installation in Claude Code:

```
❯ /plugin
─────────────────────────────────────────────────────────────────────
 Plugins  Discover   Installed   Marketplaces  (←/→ or tab to cycle)

   Local
 ❯ dkmcp MCP · ✔ connected

   User
   code-review Plugin · claude-plugins-official · ✔ enabled
   gopls-lsp Plugin · claude-plugins-official · ✔ enabled

  Space to toggle · Enter to details · escape to back
```

2. **Ask AI to create a wrapper command for the installed plugin**

The code-review plugin normally requires the gh (GitHub CLI) command. Here, we'll configure it to work locally without gh dependency. (Adjust to your preferences.)

   Ask Claude Code:

   ```
   Analyze the code-review plugin and create a custom command
   that can be used from the parent directory.
   Requirements:
   - Allow selecting which Git-managed project under workspace to review
   - Confirm the target branch with the user
   - Get the PR summary and purpose from the user
   - Run the same review as the code-review plugin in the selected project directory
   - Process locally without accessing GitHub or using gh
   ```

   > **Sample file**: A sample command created with the above requirements is available.
   > To use it, copy it to `.claude/commands/`:
   > ```bash
   > mkdir -p .claude/commands
   > cp docs/samples/local-review.en.md .claude/commands/local-review.md
   > ```
   > A Japanese version (`local-review.md`) is also available.

3. After AI completes the custom command creation, restart AI to recognize the custom command.


```
❯ /exit
  ⎿  Bye!
```
Then from the terminal:

```
$ claude  --allow-dangerously-skip-permissions
```


## Usage Example of Custom Command Created Above

The command name in the example below refers to the sample file mentioned above.

**Scenario: iOS app login feature not working**

1. Run `/local-review` and select **your-apps-ios**
   → Review iOS login screen code

2. Run `/local-review` and select **your-apps**
   → Review API authentication endpoint

3. Ask Claude Code to check logs via DockMCP
   → Inspect API container error logs

With this workspace structure, you can comprehensively investigate issues across multiple projects.

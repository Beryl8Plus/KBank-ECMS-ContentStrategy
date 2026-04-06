---
name: conventional-pr
description: "Skill to prepare Conventional Commits and create pull requests using MCP (GitHub) tooling. Automates commit formatting, optional commit rewriting (squash/amend), and PR creation with confirmation before remote operations."
argument-hint: "[--base-branch=branch] [--branch-prefix=prefix] [--commit-strategy=single-commit|squash-to-single-commit|preserve-multiple] [--draft=(true|false)]"
---

## Overview

This skill implements a reproducible workflow to prepare one or more Conventional Commit-compliant commits from the working tree and to create a pull request using MCP GitHub tools. It is workspace-scoped and intended to be used by developers preparing PRs that must follow Conventional Commits.

## Purpose

- Enforce Conventional Commit message format for PR titles and commit history.
- Provide flexible commit strategies (squash/amend/preserve) with a safe preview step.
- Create a PR using MCP (GitHub) tools only after explicit user confirmation.

## Capabilities

- Inspect working tree and staged files.
- Generate Conventional Commit message(s) interactively (type, scope, summary, body, footer).
- Amend or create commit(s) per selected commit strategy.
- Present diff + commit message preview to user for review.
- On confirmation, run MCP GitHub tools to push branch and open a PR.

## Critical Actions / Safety

- The skill will NOT perform any remote write (push, create PR, merge) without an explicit confirmation step from the user.
- Before any MCP call the skill will show the exact command it will run and wait for confirmation.

## On Activation

1. Inspect repository status (git status, staged files).
2. Ask user for invocation arguments not provided (base branch, branch prefix, commit strategy, draft PR default).
3. Build suggested branch name and conventional commit message(s).
4. Show a preview: files changed, unified diff, and commit message(s).
5. If user approves, apply local git operations per `commit_strategy`.
6. Show the MCP GitHub command(s) to push and open a PR, and ask for final confirmation.
7. If confirmed, call MCP tool(s) to push and create PR; return PR URL and tool output.

## Inputs

- `base_branch` (default from agent defaults — e.g., `ref head`)
- `branch_prefix` (default: `conventional/`)
- `commit_strategy` (one of `single-commit`, `squash-to-single-commit`, `preserve-multiple`)
- `draft` (bool — whether to create draft PR by default)
- Commit metadata: `type`, `scope` (optional), `short summary`, `body`, `breaking change` (optional)

## Outputs

- Local changes: amended/created commit(s) (when requested).
- Remote: pushed branch and opened PR (URL), or nothing if user cancels.
- A JSON-like action summary returned by the skill on completion:
  - `branch`: pushed branch name
  - `commit_messages`: []
  - `pr_url`: string (empty if not created)
  - `mcp_output`: textual output from MCP commands

## Example Prompts

- "Prepare a single conventional commit of type `feat` scope `decision-rule` with summary 'add advanced filter' and open a PR against `main`."
- "Preview conventional commits for staged files and do not push."
- "Squash staged changes into a single conventional commit and open a ready-for-review PR."

## Implementation Notes

- Use `git` locally to construct and manipulate commits (amend, reset, rebase -i or git commit --amend, git commit-tree as appropriate).
- For remote operations, only use the configured MCP tool alias (e.g., `github_mcp`) as allowed by the agent configuration.
- Ensure all git operations are run in a safe, recoverable way (create a temporary branch or backup ref before destructive operations).

## Edge Cases

- No staged changes: ask whether to include unstaged or abort.
- Complex multi-commit histories with merges: present clear preview and prefer non-destructive strategy unless user explicitly chooses rewrite.
- Conflicts on push: surface error and provide guidance to resolve locally.

## Test & Validation

- Unit tests / integration tests should validate message generation and branch naming (outside scope of this skill file). Manual testing steps:

```bash
# preview only
# 1. stage some files
# 2. invoke skill with preview

# create single squashed commit and preview
# create PR but do not confirm
```

## Files & Artifacts

- No files required to be committed to the repo by default. The skill may create temporary refs for safe rewriting.

## Next steps (recommended)

- Add a pre-commit hook to validate commit messages locally (recommend `commitlint` or a small Go script).
- Add a CI check that validates PR title or commit messages against Conventional Commits.

---

If you want, I can now:

- Create this `SKILL.md` file in `.github/skills/conventional-pr/` (done).
- Add a small `scripts/commit-lint.sh` pre-commit hook and a CI job definition to validate commit messages.
- Implement a small automation that uses the MCP tools to actually run `push` + `create PR` when you confirm.

Which of these next steps should I do now? (pre-commit hook / CI check / automation)

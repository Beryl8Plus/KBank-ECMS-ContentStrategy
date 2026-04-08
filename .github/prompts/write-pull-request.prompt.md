---
name: write-pull-request
description: "Draft Conventional Commit PR titles and a structured PR description for the current repository changes. Use when you want title suggestions, a PR body, or a verified create-PR command."
agent: write-pull-request-agent
argument-hint: "Examples: suggest 3 titles, draft PR body, or prepare create command"
---

Use the current repository context to prepare a pull request draft.

- Inspect staged changes, recent commits, current branch, and unified diff.
- Produce 2 or 3 title suggestions, recommend one, and explain the choice.
- Generate a structured PR body with summary, motivation, changes, testing, and reviewer checklist sections.
- If the user asks to create a PR, show a verified `scripts/conventional-pr.sh` command using only supported flags and wait for confirmation before any remote write.
- If the user asks to update an existing PR, draft replacement title and body text and clearly state that the repository script does not implement PR update mode.

---
name: manage-pull-request
description: "Create or update pull requests with strong Conventional Commit titles and structured PR descriptions. Use when you want title suggestions, a PR body, a verified create-PR command, or updated PR text."
agent: manage-pull-request-agent
argument-hint: "Examples: suggest 3 titles, prepare create PR command, rewrite PR body, update PR title and description"
---

Use the current repository context to prepare or refine a pull request.

- Inspect staged changes, recent commits, current branch, unified diff, and the pull request template when present.
- Produce 2 or 3 Conventional Commit title suggestions, recommend one, and explain the choice.
- Generate a structured PR body with summary, motivation, changes, testing, risks, and reviewer checklist sections.
- If the user asks to create a PR, show the proposed title and body, then use the GitHub PR create capability to open it after confirmation.
- If the user asks to update an existing PR, draft replacement title and body text first, and only then prepare an update action for the PR after confirmation.
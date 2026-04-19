---
name: update-pull-request
description: "Update an existing pull request with a stronger Conventional Commit title and a clearer PR description. Use when you want to rewrite PR text, improve accuracy, or prepare a PR update action."
agent: manage-pull-request-agent
argument-hint: "Examples: rewrite this PR title, improve PR description, update PR 123"
---

Use the current repository context and the referenced pull request to refine an existing PR.

- Inspect the existing PR title and body, recent commits, current branch, unified diff, and the pull request template when present.
- Produce 2 or 3 improved Conventional Commit title suggestions when the current title is weak or inaccurate.
- Generate replacement PR body text with summary, motivation, changes, testing, risks, and reviewer checklist sections.
- If the user confirms, prepare or execute a pull request update action using the available GitHub PR update capability.
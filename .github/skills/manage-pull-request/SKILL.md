---
name: manage-pull-request
description: "Create or update pull requests with strong Conventional Commit titles, structured PR descriptions, and verified create/update flows. Use when you need the best PR title, PR body, create PR command, or PR update text for this repository."
argument-hint: "Examples: suggest PR titles, prepare create PR command, rewrite PR title/body, update PR description"
---

## Overview

This skill generates concise, review-ready pull request titles and structured descriptions from the current repository state. It supports both create and update workflows, while keeping title quality aligned with Conventional Commit conventions used in this repository.

## When to Use

- Suggest a Conventional Commit style PR title.
- Draft a pull request description from staged changes or recent commits.
- Prepare a safe create-PR command for this repository.
- Rewrite or update an existing PR title or body.
- Improve a weak PR title into a clearer Conventional Commit style title.

## Workflow

1. Inspect staged changes, current branch, recent commits, unified diff, and pull request template content when present.
2. Infer the best Conventional Commit type, optional scope, and concise summary from the dominant change.
3. Generate 2 or 3 title alternatives and recommend one with justification.
4. Produce a structured PR body with summary, motivation, changes, testing, risks, and reviewer checklist sections.
5. If the user wants a create flow, show the proposed title and body for confirmation, then use the GitHub PR create capability or MCP PR create action to open it.
6. If the user wants an update flow, generate replacement title and body text and prefer the repository's GitHub PR update capability over the local script, since the script does not support update mode.

## Conventional Commit Quality Bar

- Prefer `type(scope): summary` when a meaningful scope exists.
- Keep the recommended title within about 72 characters when practical.
- Use imperative, repository-relevant summaries such as `implement`, `refactor`, `fix`, `normalize`, or `document`.
- Avoid vague summaries such as `update stuff`, `changes`, or branch-name restatements.
- Base the chosen type on the user-visible impact, not just the file category.

## GitHub PR Actions

For create flows, use the GitHub PR create capability (e.g., `github-pull-request_create_pull_request` or equivalent MCP tool) after user confirmation.

For update flows, use the GitHub PR update capability after user confirmation.

## Safety Rules

- Do not perform remote writes without explicit user confirmation.
- Show the exact command before execution.
- Call out limitations when the user asks for existing PR updates.
- For update requests, prefer generating replacement text first, then use a GitHub PR update tool only after confirmation.

## Output

Return:

- `titles`: suggested titles with `style`, `title`, and `justification`
- `body`: structured PR description
- `recommended_command`: optional GitHub PR create action summary for create flows
- `recommended_update`: optional PR update plan or API action summary for update flows
- `notes`: limitations or follow-up guidance

## Validation

- Prefer Conventional Commit formatting for the recommended title.
- Prefer repository templates and repository terminology when drafting the body.
- Ensure the PR action parameters match the proposed title and body.
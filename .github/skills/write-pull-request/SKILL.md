---
name: write-pull-request
description: "Draft Conventional Commit pull request titles, structured PR descriptions, and verified create-PR commands. Use when you need help writing a PR title, PR body, or preparing the repository PR script command."
argument-hint: "Examples: suggest PR titles, draft PR body, or prepare create command"
---

## Overview

This skill generates concise, review-ready pull request titles and structured descriptions from the current repository state. It can also prepare a verified `scripts/conventional-pr.sh` command for create flows.

## When to Use

- Suggest a Conventional Commit style PR title.
- Draft a pull request description from staged changes or recent commits.
- Prepare a safe create-PR command for this repository.
- Rewrite an existing PR title or body as text before a manual update.

## Workflow

1. Inspect staged changes, current branch, recent commits, and unified diff.
2. Generate 2 or 3 title alternatives and recommend one with justification.
3. Produce a structured PR body with summary, motivation, changes, testing, and checklist sections.
4. If the user wants a create flow, translate the selected title and body into a verified `scripts/conventional-pr.sh` command.
5. If the user wants to update an existing PR, generate replacement title and body text and clearly note that the repository script does not implement PR update mode.

## Verified Script Contract

Use only the flags implemented by `scripts/conventional-pr.sh`:

- `--base-branch`
- `--branch-prefix`
- `--commit-strategy`
- `--draft`
- `--type`
- `--scope`
- `--summary`
- `--body`
- `--breaking`
- `--yes`

Do not invent unsupported flags such as `--action`, `--pr_ref`, `--head-branch`, `--title`, `--push`, or `--confirm`.

## Safety Rules

- Do not perform remote writes without explicit user confirmation.
- Show the exact command before execution.
- Call out limitations when the user asks for existing PR updates, because the script only supports local prepare-and-create flows.

## Output

Return:

- `titles`: suggested titles with `style`, `title`, and `justification`
- `body`: structured PR description
- `recommended_command`: optional verified `scripts/conventional-pr.sh` command
- `notes`: limitations or follow-up guidance

## Validation

- Prefer Conventional Commit formatting for the recommended title.
- Keep the recommended title within about 72 characters when practical.
- Ensure the command preview matches the actual script help output.

---
applyTo: "**"
description: "Use when the user asks to create, update, rewrite, review, or manage a pull request title, description, body, or PR command in this repository."
---

- Treat pull request requests as managed workflow requests, not ad hoc writing tasks.
- Prefer `/manage-pull-request` for create flows and `/update-pull-request` for existing PR updates.
- Always read `.github/PULL_REQUEST_TEMPLATE/pull_request_template.md` before drafting or updating PR text.
- Enforce Conventional Commit PR titles in the format `type(scope): short summary` when a meaningful scope exists.
- Keep recommended PR titles within about 72 characters when practical.
- Use imperative summaries such as `add`, `implement`, `fix`, `refactor`, `normalize`, or `document`.
- If the current PR title is already strong and accurate, preserve it and improve only the body.
- For create flows, use the GitHub PR create capability after confirmation.
- For update flows, draft replacement title and body text first, then perform the PR update action only after explicit user confirmation.
- Do not push branches, create pull requests, or update remote PRs without explicit user confirmation.
- Structure PR bodies to match the repository template with clear `What`, `Why`, `How`, `Tests`, `Breaking changes`, and `Checklist` content.
- When issue links are unknown, leave `Refs` or `Fixes` placeholders instead of inventing issue numbers.
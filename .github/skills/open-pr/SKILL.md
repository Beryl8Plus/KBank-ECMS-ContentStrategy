---
name: open-pr
description: >
  Create or update a GitHub pull request using MCP GitHub tools. Analyzes the
  current branch's commits and diff, drafts a Conventional Commit title and
  structured PR body, confirms with the user, then calls the MCP GitHub create
  (or update) tool. Use when the user says "open a PR", "create pull request",
  "push PR", "/open-pr", or "update this PR".
argument-hint: "Examples: open PR to develop, create draft PR, update PR #42 title and body"
---

# Open Pull Request via MCP

Draft and open GitHub pull requests using MCP GitHub tools, following Conventional Commit conventions.

## When to Use

- User says "open a PR", "create pull request", "submit PR", `/open-pr`
- User wants to update an existing PR title/body
- Branch is ready and needs a PR to a base branch

## Defaults

| Setting          | Value                                      |
| ---------------- | ------------------------------------------ |
| Base branch      | `develop`                                  |
| Draft            | `false`                                    |
| Title max length | 72 chars                                   |
| Title style      | Conventional Commit `type(scope): summary` |

## MCP Tool Priority

Use the first available tool from this list:

1. `mcp__plugin_github_github__create_pull_request` — primary (plugin:github server)
2. `mcp__github__create_pull_request` — fallback (built-in github MCP)
3. `mcp__plugin_everything-claude-code_github__create_pull_request` — last resort

For reading existing PRs: `mcp__plugin_github_github__pull_request_read` or `mcp__plugin_github_github__list_pull_requests`.
For updating existing PRs: `mcp__plugin_github_github__update_pull_request`.

## Step-by-Step Workflow

### Step 1 — Gather context (read-only)

Run these in parallel:

```bash
git branch --show-current
git log --oneline origin/develop..HEAD
git diff --stat origin/develop..HEAD
git status --short
```

Also check for a PR template:

- `.github/PULL_REQUEST_TEMPLATE/pull_request_template.md`
- `.github/pull_request_template.md`

### Step 2 — Detect if branch is pushed

```bash
git ls-remote --exit-code origin "$(git branch --show-current)"
```

If the branch is not on remote, **warn the user** and offer:

```bash
git push -u origin <branch>
```

Do not proceed with PR creation until the branch is pushed.

### Step 3 — Draft the title

Rules:

- Format: `type(scope): summary`
- Types: `feat | fix | refactor | docs | test | chore | perf | ci | build | revert`
- Scope: extract ticket ID from branch name if present (e.g. `feat/KER2-58-something` → scope `KER2-58`), or use the dominant module (`cache`, `delivery`, `backoffice`, `migrations`)
- Summary: imperative verb, ≤ 72 chars total
- Breaking change: append `!` before `:` (e.g. `feat(api)!: remove v1`)

Generate 2–3 alternatives and **recommend one** with a short justification.

### Step 4 — Draft the PR body

Use the project PR template structure:

```markdown
# Description

- **What:** <what changed>
- **Why:** <motivation / ticket link>
- **How:** <implementation approach, no low-level details>

# Related issues / refs

- Refs: #<ticket>

# Breaking changes

<!-- none → omit this section -->

# Tests

- Unit / integration tests added or updated: <yes/no + notes>
- Manual test steps (if applicable): <steps>

# Checklist

- [ ] Tests added/updated
- [ ] Documentation updated (where applicable)
- [ ] CI passes
- [ ] Changelog / release notes updated (if applicable)
```

### Step 5 — Show proposal and confirm

Display the proposed title and full body. Ask the user:

> "Shall I create this PR? (yes / edit / cancel)"

**Do not call any write tool until the user confirms.**

If the user edits, incorporate feedback and show again before proceeding.

### Step 6 — Create PR via MCP

After confirmation, call the MCP tool:

```
Tool: mcp__plugin_github_github__create_pull_request
Parameters:
  owner:  <repo owner, e.g. Beryl8Plus>
  repo:   <repo name, e.g. KBank-ECMS-Backend>
  title:  <confirmed title>
  body:   <confirmed body>
  head:   <current branch>
  base:   develop        ← default; use user-supplied value if different
  draft:  false          ← set true if user said --draft or "as draft"
```

### Step 7 — Return PR URL

Report the PR URL returned by the tool and any next steps (e.g., assign reviewers, add labels).

## Repository Detection

Detect owner/repo from git remote automatically:

```bash
git remote get-url origin
# e.g. https://github.com/Beryl8Plus/KBank-ECMS-Backend.git
# → owner: Beryl8Plus, repo: KBank-ECMS-Backend
```

## Safety Rules

- **Never push or create PRs without explicit user confirmation.**
- Always show the full title and body before any write action.
- If the branch is not pushed, warn and offer to push first.
- For update flows, read the existing PR before proposing changes.
- Do not squash or amend commits — only create/update the PR metadata.

## Update Flow

When the user wants to update an existing PR:

1. Read the current PR: `mcp__plugin_github_github__pull_request_read`
2. Show current title and body
3. Generate improved title and/or body
4. Confirm with user
5. Call `mcp__plugin_github_github__update_pull_request` after confirmation

## Example Invocation

User: `/open-pr`

Expected output:

```
Branch: feat/KER2-58-expiry-worker
Commits ahead of develop: 3

Proposed title (recommended):
  feat(KER2-58): implement expiry mechanism for ended occurrences

Alternatives:
  feat(schedule): expire ended schedule occurrences via background worker
  chore(worker): add occurrence expiry job and related service updates

PR body:
---
# Description
- **What:** Adds an expiry worker that marks ended ScheduleOccurrences as expired.
- **Why:** KER2-58 — occurrences past their end time were left in ACTIVE state indefinitely.
- **How:** New `OccurrenceExpiryWorker` goroutine ticks every hour; queries occurrences
  where end_time < now and status = ACTIVE, then bulk-updates to EXPIRED.

# Related issues / refs
- Refs: KER2-58

# Tests
- Unit / integration tests added or updated: yes — TestOccurrenceExpiryWorker added
- Manual test steps: run worker with a seed occurrence past end_time, verify status = EXPIRED

# Checklist
- [ ] Tests added/updated
- [ ] Documentation updated (where applicable)
- [ ] CI passes
- [ ] Changelog / release notes updated (if applicable)
---

Shall I create this PR to `develop`? (yes / edit / cancel)
```

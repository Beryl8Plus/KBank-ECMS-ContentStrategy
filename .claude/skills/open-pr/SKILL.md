---
name: open-pr
description: Create a pull request following KBank-ECMS-Backend conventions — conventional commits title, structured body, targeting dev branch
version: 1.0.0
source: local-git-analysis
---

# Open Pull Request — KBank ECMS Backend

Create a PR that matches this project's conventions derived from git history and existing PR bodies.

## Project Conventions

### Branch naming

```
feat/<kebab-description>
fix/<kebab-description>
refactor/<kebab-description>
chore/<kebab-description>
docs/<kebab-description>
```

### PR title format (mirrors conventional commits)

```
<type>(<scope>): <short description>
```

- Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `perf`, `test`, `ci`, `build`, `revert`
- Scope: Jira ticket ID (`KER2-XXX`), service name (`cache`, `delivery`, `backoffice`), or entity name
- Example: `feat(KER2-119/120/123): add decision-rule wizard service and HTTP`

### Base branch

Always target **`dev`** (not `main`).

---

## Creating the PR

Two methods available for step 5 (push and create):

### Method 1: MCP Server (Recommended)

- **Tool:** `mcp__github__create_pull_request`
- **Advantage:** No CLI dependency, structured JSON parameters, better error handling
- **Required params:**
  - `owner`: `Beryl8Plus`
  - `repo`: `KBank-ECMS-Backend`
  - `head`: current branch name (e.g., `chore/add-skill`)
  - `base`: `dev`
  - `title`: conventional-commits formatted title
  - `body`: full PR body with all 6 sections

### Method 2: GitHub CLI (Fallback)

- **Command:** `gh pr create --base dev --title "..." --body "..."`
- **Advantage:** Lightweight, familiar command-line interface
- **Disadvantage:** Requires `gh` CLI to be installed and authenticated

---

## Step-by-Step Workflow

### 1. Gather context

```bash
# Confirm current branch and base
git branch --show-current
git log --oneline dev..HEAD

# See all changed files vs dev
git diff --name-only dev..HEAD

# Full diff for summarisation
git diff dev..HEAD
```

### 2. Determine PR title

- Extract the Jira ticket(s) from branch name or recent commits (e.g. `KER2-119`).
- Use the conventional-commit type of the dominant change (`feat` if adding capability, `fix` if correcting behaviour, `refactor` if restructuring without behaviour change).
- Keep under 72 characters.

### 3. Draft the PR body

Follow this exact structure (fill every section — do not omit):

```markdown
## Summary

- <bullet: what was added / changed at a high level>
- <bullet: …>

## Motivation

<1-2 sentences: why this change was needed — reference Jira ticket or business driver>

## Key Changes

| Area                     | Change                 |
| ------------------------ | ---------------------- |
| `<file or package path>` | <what changed and why> |
| …                        | …                      |

## Testing

- [ ] `go test ./...` green
- [ ] <specific test command for the affected service>
- [ ] <manual verification step via Swagger or curl>
- [ ] <edge case verified>

## Risks

- **<Risk title>**: <1-sentence description of what could go wrong and mitigation>

## Reviewer Checklist

- [ ] Entity field naming follows `UPPER_SNAKE_CASE` GORM convention
- [ ] New endpoints have Swagger annotations and `make swagger` was re-run
- [ ] Wire providers updated in `wire_gen.go` if DI changed
- [ ] Any new migration has a valid `-- +goose Down` rollback section
- [ ] No secrets or `.env` values committed
```

### 4. Pre-flight checks

```bash
# Vet and lint
make vet
make lint

# All tests green
make test
```

Fix any failures before opening the PR.

### 5. Create PR

**Create PR using MCP Server (preferred) or GitHub CLI (fallback):**

#### Option A: MCP Server (mcp**github**create_pull_request)

Use this when available — faster and more reliable:

```bash
# Requires:
# - Repository owner (e.g., Beryl8Plus)
# - Repository name (e.g., KBank-ECMS-Backend)
# - Head branch (current branch)
# - Base branch (dev)
# - PR title and body
```

#### Option B: GitHub CLI (fallback)

Use if MCP server unavailable:

```bash
gh pr create \
  --base dev \
  --title "<conventional-commits title>" \
  --body "$(cat <<'EOF'
## Summary
...

## Motivation
...

## Key Changes
| Area | Change |
|---|---|
| … | … |

## Testing
- [ ] …

## Risks
- …

## Reviewer Checklist
- [ ] …
EOF
)"
```

Return the PR URL to the user.

---

## Quick Reference

| Setting             | Value                                                               |
| ------------------- | ------------------------------------------------------------------- |
| Default base branch | `dev`                                                               |
| Commit/title format | `<type>(<scope>): <description>`                                    |
| Jira ticket format  | `KER2-XXX`                                                          |
| Go test command     | `go test ./...` or `go test ./internal/service/... -run <TestName>` |
| Swagger regen       | `make swagger`                                                      |
| Wire regen          | `make wire-gen`                                                     |
| Lint                | `make lint`                                                         |
| Build               | `make build`                                                        |

## Common Scopes

| Scope        | Used for                           |
| ------------ | ---------------------------------- |
| `KER2-XXX`   | Jira-tracked feature / fix         |
| `cache`      | In-memory cache, Redis pub/sub     |
| `delivery`   | `svc-contstrat-delivery` service   |
| `backoffice` | `svc-contstrat-backoffice` service |
| `entity`     | Domain model changes               |
| `migrations` | goose migration files              |
| `attributes` | Rule attribute management          |

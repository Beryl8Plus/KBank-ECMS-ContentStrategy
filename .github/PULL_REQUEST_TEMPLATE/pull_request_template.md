<!--
Use Conventional Commits in the PR title and squash commit.
Title format: `type(scope): short summary` (example: `feat(auth): add OAuth2 token refresh`)

Common types: feat, fix, docs, style, refactor, perf, test, chore, build, ci
Breaking changes: add `!` in the title (e.g., `feat(api)!: ...`) or include a
`BREAKING CHANGE: <description>` footer in the body.
-->

# Title
<type>(<scope>): <short summary>

# Description
- **What:** Brief summary of the change.
- **Why:** Motivation and context (why this is needed).
- **How:** High-level description of the implementation (no low-level details).

# Related issues / refs
- Refs: #<issue>
- Fixes: #<issue>  <!-- use if this PR closes an issue -->

# Breaking changes
BREAKING CHANGE: <description>  <!-- remove this section if there are none -->

# Tests
- Unit / integration tests added or updated: <yes/no and notes>
- Manual test steps (if applicable):

# Checklist
- [ ] Tests added/updated
- [ ] Documentation updated (where applicable)
- [ ] CI passes
- [ ] Changelog / release notes updated (if applicable)

---

## Examples

- PR title example: `feat(auth): add OAuth2 token refresh`
- Breaking change example: `feat(api)!: remove v1 endpoints` and include
	`BREAKING CHANGE: v1 endpoints removed; update clients to v2` in the body.


#!/usr/bin/env bash
set -euo pipefail

msg_file="${1:-}"
if [ -z "$msg_file" ]; then
  echo "Usage: $0 <commit-msg-file>" >&2
  exit 0
fi

if [ ! -f "$msg_file" ]; then
  echo "Commit message file not found: $msg_file" >&2
  exit 1
fi

msg=$(sed -n '1p' "$msg_file" | tr -d '\r')

if printf '%s\n' "$msg" | grep -qE '^(Merge|Revert)'; then
  exit 0
fi

type_regex='feat|fix|docs|style|refactor|perf|test|chore|ci|build|revert'
scope_regex='[a-z0-9._/-]+'
regex="^($type_regex)(\\($scope_regex\\))?(!)?:\\s.+"

if printf '%s\n' "$msg" | grep -E -q "$regex"; then
  exit 0
else
  cat >&2 <<'EOF'
ERROR: Commit message does not follow Conventional Commits.
Expected: <type>(<scope>)?: <description>
Allowed types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert
Example: feat(decision-rule): add advanced filter
To bypass locally: use `git commit --no-verify`
EOF
  exit 1
fi

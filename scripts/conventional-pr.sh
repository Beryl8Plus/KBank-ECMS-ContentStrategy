#!/usr/bin/env bash
set -euo pipefail

show_help() {
  cat <<'EOF'
Usage: conventional-pr.sh [options]

Options:
  --base-branch=BRANCH       Base branch for PR (default: main)
  --branch-prefix=PREFIX     Branch name prefix (default: conventional/)
  --commit-strategy=STRAT    one of single-commit|squash-to-single-commit|preserve-multiple (default: single-commit)
  --draft=(true|false)       Create PR as draft (default: false)
  --type=TYPE                Conventional commit type (feat|fix|docs|chore|...)
  --scope=SCOPE              Commit scope (optional)
  --summary=SUMMARY          Short summary (required unless interactive)
  --body=BODY                Commit body (optional)
  --breaking=TEXT            Breaking change description (optional)
  --yes                      Answer yes to prompts
  --help                     Show this help

This script helps prepare a branch and Conventional Commit, previews changes,
and prints (or runs) the push + PR commands. It asks for explicit confirmation
before performing any remote writes.
EOF
}

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
if [ -n "$repo_root" ]; then
  cd "$repo_root"
fi

base_branch="main"
branch_prefix="conventional/"
commit_strategy="single-commit"
draft="false"
assume_yes=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-branch=*) base_branch="${1#*=}"; shift ;;
    --branch-prefix=*) branch_prefix="${1#*=}"; shift ;;
    --commit-strategy=*) commit_strategy="${1#*=}"; shift ;;
    --draft=*) draft="${1#*=}"; shift ;;
    --type=*) cc_type="${1#*=}"; shift ;;
    --scope=*) cc_scope="${1#*=}"; shift ;;
    --summary=*) cc_summary="${1#*=}"; shift ;;
    --body=*) cc_body="${1#*=}"; shift ;;
    --breaking=*) cc_breaking="${1#*=}"; shift ;;
    --yes|-y) assume_yes=1; shift ;;
    --help|-h) show_help; exit 0 ;;
    *) echo "Unknown arg: $1"; show_help; exit 1 ;;
  esac
done

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Not inside a git repository." >&2
  exit 1
fi

current_branch=$(git rev-parse --abbrev-ref HEAD)
short_sha=$(git rev-parse --short HEAD 2>/dev/null || echo "local")
timestamp=$(date +%Y%m%d%H%M%S)

slugify() {
  echo "$*" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g' | sed -E 's/^-+|-+$//g'
}

if [ -z "${cc_type-}" ]; then
  if [ "$assume_yes" -eq 0 ]; then
    read -p "Commit type (feat|fix|docs|style|refactor|perf|test|chore|ci|build): " cc_type
  else
    cc_type="chore"
  fi
fi

if [ -z "${cc_summary-}" ]; then
  if [ "$assume_yes" -eq 0 ]; then
    read -p "Short summary: " cc_summary
  else
    cc_summary="update"
  fi
fi

if [ -z "${cc_type-}" ] || [ -z "${cc_summary-}" ]; then
  echo "Commit type and summary are required." >&2
  exit 1
fi

title="$cc_type"
if [ -n "${cc_scope-}" ]; then
  title+="(${cc_scope})"
fi
if [ -n "${cc_breaking-}" ]; then
  title+="!"
fi
title+=": $cc_summary"

full_message="$title"
if [ -n "${cc_body-}" ]; then
  full_message+="\n\n$cc_body"
fi
if [ -n "${cc_breaking-}" ]; then
  full_message+="\n\nBREAKING CHANGE: $cc_breaking"
fi

branch_slug_type=$(slugify "$cc_type")
branch_slug_summary=$(slugify "$cc_summary")
branch_name="${branch_prefix%/}/${branch_slug_type}-${branch_slug_summary}-${timestamp}"

staged_files=$(git diff --cached --name-only || true)
if [ -z "$staged_files" ]; then
  echo "No staged files detected."
  if [ "$assume_yes" -eq 0 ]; then
    read -p "Add all changes to the commit (git add -A)? [y/N] " addall
  else
    addall="y"
  fi
  if [[ "$addall" =~ ^[Yy]$ ]]; then
    git add -A
    staged_files=$(git diff --cached --name-only || true)
  else
    echo "Aborting: nothing staged." >&2
    exit 1
  fi
fi

echo "\nPreview"
echo "------"
echo "Current branch: $current_branch"
echo "Target PR base: $base_branch"
echo "New branch: $branch_name"
echo "Commit message (first line):"
echo "  $title"
echo ""
echo "Staged files:"
git --no-pager diff --cached --name-status || true
echo "\nUnified diff (staged):"
git --no-pager diff --cached || true

if [ "$assume_yes" -eq 0 ]; then
  read -p "Proceed with local git operations to create branch and commit? [y/N] " proceed
else
  proceed="y"
fi
if [[ ! "$proceed" =~ ^[Yy]$ ]]; then
  echo "Cancelled by user. No changes made."; exit 0
fi

# make a backup branch pointing to current HEAD
backup_branch="backup/conventional-${timestamp}-${short_sha}"
git branch --force "$backup_branch" "$current_branch"
echo "Created local backup branch: $backup_branch"

case "$commit_strategy" in
  preserve-multiple)
    git checkout -b "$branch_name"
    if [ -n "$staged_files" ]; then
      tmpmsg=$(mktemp)
      printf '%s
'"$full_message" > "$tmpmsg"
      git commit -F "$tmpmsg" || true
      rm -f "$tmpmsg"
    fi
    ;;
  single-commit)
    git checkout -b "$branch_name"
    tmpmsg=$(mktemp)
    printf '%s
'"$full_message" > "$tmpmsg"
    git commit -F "$tmpmsg" || {
      echo "Nothing to commit (maybe there were no staged changes).";
      rm -f "$tmpmsg";
    }
    rm -f "$tmpmsg" || true
    ;;
  squash-to-single-commit)
    # create new branch from base branch (prefer remote if available)
    base_ref=""
    if git show-ref --verify --quiet "refs/remotes/origin/$base_branch"; then
      base_ref="origin/$base_branch"
    elif git show-ref --verify --quiet "refs/heads/$base_branch"; then
      base_ref="$base_branch"
    else
      if [ "$assume_yes" -eq 0 ]; then
        read -p "Base branch $base_branch not found locally. Fetch origin/$base_branch? [y/N] " dofetch
      else
        dofetch="y"
      fi
      if [[ "$dofetch" =~ ^[Yy]$ ]]; then
        git fetch origin "$base_branch" --quiet || true
        base_ref="origin/$base_branch"
      else
        echo "Cannot perform squash: base branch not available."; exit 1
      fi
    fi
    git checkout -b "$branch_name" "$base_ref"
    # merge --squash the current branch into the new branch
    set +e
    git merge --squash "$current_branch"
    merge_rc=$?
    set -e
    if [ $merge_rc -ne 0 ]; then
      echo "Merge --squash returned non-zero (possible conflicts). Resolve conflicts and run 'git commit' to finish, or restore from $backup_branch.";
      exit 1
    fi
    tmpmsg=$(mktemp)
    printf '%s
'"$full_message" > "$tmpmsg"
    git commit -F "$tmpmsg"
    rm -f "$tmpmsg"
    ;;
  *)
    echo "Unknown commit strategy: $commit_strategy"; exit 1 ;;
esac

echo "Local branch prepared: $branch_name"

push_cmd=(git push -u origin "$branch_name")
if command -v gh >/dev/null 2>&1; then
  pr_cmd=(gh pr create --base "$base_branch" --head "$branch_name" --title "$title")
  if [ -n "${cc_body-}" ]; then pr_cmd+=(--body "$cc_body"); fi
  if [ "$draft" = "true" ]; then pr_cmd+=(--draft); fi
elif [ -n "${MCP_CLI-}" ] && command -v "$MCP_CLI" >/dev/null 2>&1; then
  pr_cmd=("$MCP_CLI" pr create --base "$base_branch" --head "$branch_name" --title "$title")
  if [ -n "${cc_body-}" ]; then pr_cmd+=(--body "$cc_body"); fi
  if [ "$draft" = "true" ]; then pr_cmd+=(--draft); fi
else
  pr_cmd=()
fi

echo "\nPlanned remote commands (will not run yet):"
echo "  ${push_cmd[*]}"
if [ ${#pr_cmd[@]} -gt 0 ]; then
  echo "  ${pr_cmd[*]}"
else
  echo "  (No PR CLI detected. Use GitHub web UI or install 'gh' to create a PR from the command line.)"
fi

if [ "$assume_yes" -eq 0 ]; then
  read -p "Execute push and create PR now? [y/N] " run_remote
else
  run_remote="n"
fi
if [[ "$run_remote" =~ ^[Yy]$ ]]; then
  echo "Pushing branch..."
  "${push_cmd[@]}"
  if [ ${#pr_cmd[@]} -gt 0 ]; then
    echo "Creating PR..."
    if "${pr_cmd[@]}"; then
      echo "PR created (see output above)."
    else
      echo "PR creation failed. You can run the command manually:";
      echo "  ${pr_cmd[*]}"
    fi
  else
    echo "No PR command available; branch pushed. Create a PR manually on GitHub.";
  fi
else
  echo "Remote operations skipped. Run the shown commands when ready.";
fi

echo "Done. Backup branch: $backup_branch"

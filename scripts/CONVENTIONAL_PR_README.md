Conventional PR helper
======================

This small helper script assists preparing a Conventional Commit, creating a feature branch, and showing push + PR commands.

Quick start
-----------

1. Make the script executable:

```bash
chmod +x scripts/conventional-pr.sh
```

2. Create a single conventional commit and prepare a branch interactively:

```bash
./scripts/conventional-pr.sh --commit-strategy=single-commit
```

3. Prepare a squash commit (non-destructive) against `main`:

```bash
./scripts/conventional-pr.sh --commit-strategy=squash-to-single-commit --base-branch=main
```

Notes
-----
- The script will create a local backup branch before performing any rewriting.
- It will never run remote commands (push/create PR) without explicit confirmation.
- If you have the GitHub CLI (`gh`) installed the script will use it to create the PR; otherwise it will print the commands to run.

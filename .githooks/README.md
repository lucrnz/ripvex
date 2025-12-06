# Git Hooks

This directory contains git hooks for the ripvex project.

## Setup

To enable these hooks, run:

```bash
git config core.hooksPath .githooks
```

This tells git to use the hooks in this directory instead of `.git/hooks`.

## Hooks

### pre-commit

Automatically formats all staged Go files using `gofmt` before committing.

**Requirements:**
- Go installation (includes `gofmt` by default)

The hook will:
1. Find all staged `.go` files
2. Format them with `gofmt -w`
3. Re-add them to staging so the formatted versions are committed

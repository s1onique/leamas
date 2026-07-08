# Git Safety: Force-Push Prevention

**ACT**: ACT-LEAMAS-FACTORY-PREVENT-FORCE-PUSH-GO-VERIFY01

## Overview

Leamas Factory enforces local Git safety rails that prevent accidental force-pushes and protected-branch deletions from Leamas working copies.

## Policy

Normal Leamas Factory development must not use:

```bash
git push --force
git push --force-with-lease
git push +branch
non-fast-forward pushes to protected branches
deletion of protected branches
```

### Protected Branch Refs

```text
refs/heads/main
refs/heads/master
refs/heads/release/*
```

## Implementation

### Pre-Push Hook

The `githooks/pre-push` hook prevents:

- Non-fast-forward pushes to protected branches
- Deletion of protected branches
- Force-push operations to protected branches

The hook uses `git merge-base --is-ancestor` to detect non-fast-forward pushes.

### Hook Installation

Install the Git hooks:

```bash
make install-git-hooks
```

Or manually:

```bash
chmod +x scripts/install_git_hooks.sh
./scripts/install_git_hooks.sh
```

This sets `core.hooksPath` to `githooks` in the local repository config.

## Verification

Verify Git hooks are properly installed and configured:

```bash
make verify-git-hooks
```

Or directly:

```bash
go run ./cmd/leamas factory verify git-hooks
```

## Important Limitation

Local Git hooks are safety rails, not absolute enforcement.

They can be bypassed by:

- Changing local Git config (`git config core.hooksPath ""`)
- Using another clone without the hook
- Pushing from another machine
- Using remote APIs or web interfaces
- Deliberately bypassing local workflows

## Future Remote Enforcement

Authoritative enforcement requires remote branch protection and required status checks once CI exists:

- Enable branch protection rules on GitHub/GitLab
- Require PR reviews and status checks
- Disable force-push on protected branches in repository settings
- Add CI checks that reject force-push workflows

## See Also

- [Tooling Boundaries](./tooling-boundaries.md)
- [Go-Only Doctrine](../doctrine/go-only.md)

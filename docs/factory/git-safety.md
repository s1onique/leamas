# Git Safety

**ACT**: ACT-LEAMAS-FACTORY-REMOTE-BRANCH-PROTECTION01

## Overview

Remote branch protection prevents destructive Git operations and ensures Factory gates pass before changes reach protected branches.

## Branch Protection Policy

### Default Branch: `main`

| Setting | Value | Rationale |
|---------|-------|-----------|
| **Force Push** | DISABLED | Prevents force-push rewrites per Factory doctrine |
| **Branch Deletion** | DISABLED | Preserves history integrity |
| **Required Status Checks** | `Factory` | Requires CI to pass |
| **Required Reviews** | None | Single-maintainer project for v0 |

## Required Status Checks

The `Factory` CI workflow must pass before merging to `main`:

```yaml
# .github/workflows/factory.yml
name: Factory
on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main
```

### Status Checks Run:

1. `make bootstrap` - Configure repo-local git hooks path (required for fresh VM environments)
2. `make factorize` - Factory verifiers
3. `make gate` - Full quality gate
4. `./bin/leamas factory verify llm-friendly` - LLM-friendliness
5. `./bin/leamas factory verify agent-context` - Agent context
6. `./bin/leamas factory verify forbidden-patterns` - Forbidden patterns
7. `go test ./...` - Unit tests
8. `go vet ./...` - Go vet

## CI Bootstrap

GitHub Actions runners are fresh VM environments. Local `.git/config` state from your laptop is not preserved during checkout. The `make bootstrap` target configures the git hooks path:

```bash
git config --local core.hooksPath githooks
```

This is required before running `make factorize`, as the `git-hooks` verifier checks that `core.hooksPath` is configured.

## Configuration

### GitHub Settings

To configure branch protection in GitHub:

1. Go to repository **Settings** → **Branches**
2. Add rule for `main`
3. Enable:
   - ✅ **Protect matching branches**
   - ✅ **Require status checks to pass before merging**
   - ✅ **Require branches to be up to date before merging**
   - ❌ **Do not allow bypassing the above settings** — leave disabled so admins may bypass required checks for direct pushes
   - ❌ **Allow force pushes** (disabled)
   - ❌ **Allow deletions** (disabled)

### CLI Configuration (gh)

```bash
# View current protection
gh api repos/{owner}/{repo}/branches/main/protection

# Update protection
gh api --method PUT repos/{owner}/{repo}/branches/main/protection \
  -f required_status_checks='{"strict":true,"contexts":["Factory Gates"]}' \
  -f enforce_admins=false \
  -f allow_force_pushes=false \
  -f allow_deletions=false
```

## Local Pre-Push Prevention

A local pre-push hook (`githooks/pre-push`) prevents force-pushes before they reach remote:

```bash
#!/bin/bash
# githooks/pre-push - Prevents force-push to protected branches
```

This provides defense-in-depth alongside remote protection.

## Doctrine Alignment

This policy aligns with:
- **Factory Meta-Loop**: Self-verification requires unrewritable history
- **No Force-Push Doctrine**: Forward-corrective commits only
- **Verification Witness**: Evidence requires immutable audit trail

## References

- [Go-verified Force-Push Prevention](../close-reports/ACT-LEAMAS-FACTORY-PREVENT-FORCE-PUSH-GO-VERIFY01.md)
- [CI Status Checks](../close-reports/ACT-LEAMAS-FACTORY-CI-STATUS-CHECKS01.md)
- [Factory Meta-Loop Doctrine](../doctrine/factory-meta-loop.md)

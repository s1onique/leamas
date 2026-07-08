# ACT-LEAMAS-FACTORY-REMOTE-BRANCH-PROTECTION01-R1 — CLOSED

## Summary

Corrected Leamas GitHub branch protection policy for solo-maintainer workflow.

`main` remains protected and still requires `Factory Gates`, but admin enforcement is now disabled so repository admins can direct-push when needed.

## Before

```json
{
  "pattern": "main",
  "isAdminEnforced": true,
  "requiresStatusChecks": true,
  "requiresStrictStatusChecks": true,
  "requiredStatusCheckContexts": ["Factory Gates"],
  "allowsForcePushes": false,
  "allowsDeletions": false
}
```

## After

```json
{
  "pattern": "main",
  "isAdminEnforced": false,
  "requiresStatusChecks": true,
  "requiresStrictStatusChecks": true,
  "requiredStatusCheckContexts": ["Factory Gates"],
  "allowsForcePushes": false,
  "allowsDeletions": false
}
```

## Files Changed

* `docs/factory/git-safety.md` - Updated CLI configuration to show `enforce_admins: false`, fixed admin bypass wording
* `docs/factory/branch-protection-proof.md` - Updated proof to reflect current state (`enforce_admins: false`)
* `docs/factory/github-policy.md` - NEW - Leamas GitHub policy document (hardcoded verifier, not declarative YAML)
* `internal/factory/github/check.go` - NEW - GitHub policy verifier with pure CheckRules function
* `internal/factory/github/check_test.go` - NEW - Tests for GitHub policy verifier
* `cmd/leamas/main.go` - Added `github` verifier command
* `internal/factory/gate/gate.go` - Added githubVerifier but NOT in default AllVerifiers (requires network)
* `docs/close-reports/ACT-LEAMAS-FACTORY-REMOTE-BRANCH-PROTECTION01-R1.md` - This close report

## R2 Fixes Applied

1. **Timestamp regression fixed**: Changed `2026-08-07` to `2026-07-08` in branch-protection-proof.md
2. **GitHub verifier NOT in default gate**: Removed from `AllVerifiers()`, run explicitly via `leamas factory verify github`
3. **Honest policy documentation**: Clear that it's a Leamas-specific hardcoded verifier
4. **Exact status check set equality**: Uses `slicesEqual()` for order-independent comparison
5. **Pure verification function**: `CheckRules()` is testable without mocking gh API
6. **Improved error messages**: More descriptive drift messages
7. **Better git-safety wording**: Clear admin bypass explanation

## Verification

* `make factorize` — PASSED
* `make gate` — PASSED
* `go test ./...` — PASSED
* `go vet ./...` — PASSED
* `gofmt -l .` — CLEAN

Remote GitHub state verified:

```json
{
  "id": "BPR_kwDOSk9xFs4Ew057",
  "pattern": "main",
  "isAdminEnforced": false,
  "allowsForcePushes": false,
  "allowsDeletions": false,
  "requiresStatusChecks": true,
  "requiresStrictStatusChecks": true,
  "requiredStatusCheckContexts": ["Factory Gates"]
}
```

## Notes

* `leamas factory verify github` runs remote GitHub verification (requires network/gh auth)
* Default `make gate` does NOT include remote GitHub verification (avoids network dependency)
* `Factory Gates` is preserved as required status check
* The pure `CheckRules()` function is fully tested without network I/O

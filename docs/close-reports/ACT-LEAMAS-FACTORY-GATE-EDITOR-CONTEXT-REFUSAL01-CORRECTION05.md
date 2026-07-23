# Close Report: ACT-LEAMAS-FACTORY-GATE-EDITOR-CONTEXT-REFUSAL01-CORRECTION05

## Summary

Canonical verifier authority and immutable closure repair. Removed the `FACTORIZE_COMMAND` override variable and hardcoded the canonical factorize command to ensure no Make variable can substitute the real verifier.

## Files Changed

| File | Change |
|------|--------|
| `make/long-tests.mk` | Removed `FACTORIZE_COMMAND` variable; hardcoded canonical command |
| `internal/factory/gate/factorize_prod_path_test.go` | Removed `FACTORIZE_COMMAND` tests; added Makefile verification test |

## Behavior Changed

1. `FACTORIZE_COMMAND` variable removed from Makefile
2. Canonical factorize command hardcoded: `go run ./cmd/leamas factory factorize`
3. Test added to verify no `FACTORIZE_COMMAND` in Makefile

## Commands Run

| Command | Result | Duration |
|---------|--------|----------|
| `git diff --check` | PASS | fast |
| `CGO_ENABLED=0 go test -count=1 ... -run 'Factorize\|Context'` | PASS | 2.058s |
| `CGO_ENABLED=0 make gate-fast` | PASS | fast |
| `LEAMAS_ALLOW_FULL_FACTORIZE=1 make factorize` | PASS | 247.38s |

## Lanes Executed

| Lane | Status |
|------|--------|
| Focused guard tests | executed |
| Fast lane | executed |
| Factorize lane | executed_once |
| Full long lane | skipped_not_required |

## Acceptance Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Editor-context refusal remains effective | ✓ |
| 2 | No Make variable can substitute canonical verifier | ✓ |
| 3 | Focused tests within 10s (actual: 2.058s) | ✓ |
| 4 | Real factorize passes | ✓ |
| 5 | CORRECTION05 plan exists in immutable freeze F | ✓ |
| 6 | All OIDs mechanically verified | ✓ |
| 7 | Reports disclose skipped lanes accurately | ✓ |
| 8 | Publication uses fast-forward only | ✓ |

## Skipped Checks

- full_long_lane: not required for this ACT

## Follow-up ACTs

- `ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01`

# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION09

## Summary

CORRECTION09 produces reviewable committed-range evidence covering the CORRECTION07 and CORRECTION08 implementation, tests, and verification with Closure Protocol V1 identities.

## Implementation Range

| Identity | Commit |
|----------|--------|
| BASE | c9944bf14defdc494fa029c423edbcda8186ac4a |
| CORRECTION07 | 254c05f1a69caea7f06f66eb01c4d775beefa45b |
| CORRECTION08 | 43dd25446505a7b228086db43dd11dcb3dbbced0 |
| Subject (HEAD) | 9276bce (whitespace fix) |

Verified ordering: C07 is ancestor of C08.

## Files Changed

| File | Change |
|------|--------|
| `cmd/leamas/factory_close.go` | Format-aware validation, sibling temp file |
| `internal/factory/closure/chain.go` | Format-aware ValidateAttestation |
| `internal/factory/closure/validation.go` | Manifest mandatory, exact tree comparisons |
| `internal/factory/closure/attestation_test.go` | Complete chain_validity objects |

Close reports:
- `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION07.md`
- `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION08.md`

## Changeset Summary

```
 cmd/leamas/factory_close.go                        |  10 +-
 internal/factory/closure/attestation_test.go        |  50 +-
 internal/factory/closure/chain.go                  |  62 +-
 internal/factory/closure/validation.go             |   9 +-
 4 files changed, 121 insertions(+), 68 deletions(-)
```

## Behavior Implemented

### CORRECTION07: Closure Protocol V1 Authority Gaps
- Exact manifest tree identity comparisons (F_TREE, S_TREE)
- Plan path, blob OID, and SHA-256 binding to Git
- All chain-validity assertions required (8 fields)
- No-self-reference required (8 indicators)
- Sibling temp file for atomic output

### CORRECTION08: Mandatory Manifest & Format-Aware Validation
- VerifyChain requires non-nil manifest (fail closed)
- ValidateAttestation requires format parameter (SHA-1/SHA-256)
- CLI detects repository format for validation
- All OID validations format-aware

## Verification Results

| Command | Result |
|---------|--------|
| `git diff --check` | No whitespace errors |
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Digest covers CORRECTION07 and CORRECTION08 range. | PASS |
| 2 | Production and test files in changeset. | PASS |
| 3 | Reports, diff, digest agree. | PASS |
| 4 | CLI positive/negative flows covered. | PASS |
| 5 | Manifest blob, digest, tree fields mandatory. | PASS |
| 6 | Immutable plan bytes generate no-self-reference. | PASS |
| 7 | SHA-1 and SHA-256 flows pass. | PASS |
| 8 | Gate evidence regenerated against exact S. | PASS |
| 9 | Generated attestation records C and tag identity. | PASS |
| 10 | Evidence from clean, published, committed range. | PASS |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

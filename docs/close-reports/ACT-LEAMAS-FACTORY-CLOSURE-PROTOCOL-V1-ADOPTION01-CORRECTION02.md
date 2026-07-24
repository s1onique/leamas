# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION02

## Summary

Subject-exact evidence convergence and repository-format binding for Closure Protocol V1 adoption. Bound all authoritative Git identities to the repository's declared storage object format.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/chain.go` | SHA-1/SHA-256 format support with `ValidateOIDWithFormat` |
| `internal/factory/closure/plan_selfref_test.go` | Plan self-reference rejection tests |
| `cmd/leamas/factory_close.go` | Chain/attest CLI commands |
| `docs/closure-plans/...CORRECTION05.json` | Frozen plan (no self-references) |
| `docs/closure-manifests/...CORRECTION05.json` | Manifest with protocol v1 structure |
| `docs/closure-manifests/...CORRECTION05.attestation.json` | Post-closure attestation |

## Behavior Changed

### 1. Git Object Format Binding

Repository storage format—not OID length—defines identity authority:

```bash
git rev-parse --show-object-format=storage  # → sha1
```

| Storage format | OID length |
|---------------|-----------|
| `sha1` | 40 hex characters |
| `sha256` | 64 hex characters |

### 2. Plan Self-Reference Validation

Frozen plans must not contain:
- `freeze_commit`, `freeze_tree`
- `subject_commit`, `subject_tree`
- `closure_commit`, `closure_tree`
- `tag_oid`, `tag_target`

Added `CheckPlanNoSelfReference(planPath)` function and tests.

### 3. Object Type Verification

Each identity must resolve to required type:
- Freeze/subject/closure commit → commit
- Freeze/subject/closure tree → tree
- Annotated tag object → tag
- Peeled tag target → commit

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |
| `factory close chain --help` | PASS |
| `factory close attest --help` | PASS |

## Digest Scope

**Files in scope for ADOPTION01 digest:**

| Category | Files |
|----------|-------|
| Implementation | `internal/factory/closure/chain.go` |
| Tests | `internal/factory/closure/*_test.go` |
| CLI | `cmd/leamas/factory_close.go` |
| Artifacts | `docs/closure-plans/...CORRECTION05.json` |
| | `docs/closure-manifests/...CORRECTION05.json` |
| | `docs/closure-manifests/...CORRECTION05.attestation.json` |
| Reports | `docs/close-reports/...ADOPTION01.md` |
| | `docs/close-reports/...CORRECTION01.md` |
| | `docs/close-reports/...CORRECTION02.md` |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Digest contains all protocol implementation files | ✓ |
| 2 | Digest, diff, report, manifest agree | ✓ |
| 3 | Frozen plans reject self-referential fields | ✓ |
| 4 | Repository format—not OID length—defines authority | ✓ |
| 5 | SHA-1 and SHA-256 validated correctly | ✓ |
| 6 | Identities resolve to expected Git object types | ✓ |
| 7 | Valid/invalid chains exercised via CLI | ✓ |
| 8 | CORRECTION05 has non-overlapping responsibilities | ✓ |
| 9 | New closure uses full mechanically derived identities | ✓ |
| 10 | Publication uses fast-forward only | ✓ |

## Related Artifacts

- `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01.md`
- `docs/close-reports/ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION01.md`

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

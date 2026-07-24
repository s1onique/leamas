# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION05

## Summary

Operational CLI convergence with complete chain authority, strict manifest validation, real attestation generation, and end-to-end verification for Closure Protocol V1.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/validation.go` | Repository root required; fails closed if missing |
| `internal/factory/closure/manifest.go` | Strict manifest decoding with `DisallowUnknownFields` |
| `internal/factory/closure/chain.go` | Chain types, OID validation, attestation validation |
| `internal/factory/closure/attestation.go` | Real attestation generation |
| `internal/factory/closure/git_identity.go` | `RealGit` exported, `NewRealGit()`, `ShowToplevel()` |
| `internal/factory/closure/model.go` | Added `Tag` field to Manifest |
| `cmd/leamas/factory_close.go` | Real `attest` command implementation |

## Behavior Changed

### 1. Repository Authority Required

The public chain command initializes repository authority internally via `git rev-parse --show-toplevel`. Missing repository authority fails closed with clear error.

### 2. Complete Chain Required for PASS

All chain fields are now mandatory:
- `--freeze`
- `--subject`
- `--closure`
- `--plan-path`
- `--tag`

A reduced command produces FAIL verdict.

### 3. Strict Manifest Decoding

All manifest-loading paths use bounded strict decoding:
- `MaxManifestBytes` limit enforced
- `DisallowUnknownFields()` for unknown field rejection
- EOF required after first JSON value
- Required field validation
- SHA-256 hex validation (exactly 64 hex chars)

### 4. Manifest-to-Git Binding

Chain validation compares manifest fields to Git objects:
- Freeze commit resolution via `F^{commit}`
- Freeze tree via `F^{tree}`
- Subject commit and tree resolution
- Plan blob OID verification
- Tag annotated status via `refs/tags/<tag>^{tag}`
- Tag peeled target equality to closure commit

### 5. Immutable Plan Validation

Plan bytes at F and S must be byte-identical. Forbidden identity keys are recursively rejected from plan JSON.

### 6. Complete Chain Verdict Invariants

PASS requires every invariant to be satisfied:
- F ≠ S
- F ancestor of S
- S ancestor of C
- F ancestor of C
- Commits resolve as commits
- Trees resolve as trees
- Plan valid at both F and S
- Plan bytes identical
- Tag is annotated
- Tag peeled target equals C

### 7. Real Attestation Generation

`leamas factory close attest` command:
1. Strictly decodes manifest
2. Requires tag field in manifest
3. Requires pass verdict
4. Derives Git objects via real Git client
5. Constructs all attestation fields internally
6. Validates constructed attestation
7. Writes atomically to output file

### 8. Lightweight Tag Rejection

`getTagInfo` requires:
1. `refs/tags/<tag>` exists
2. `refs/tags/<tag>^{tag}` resolves (annotated tag required)
3. Tag type verified as `tag`

### 9. Historical CORRECTION05 Erratum

The original CORRECTION05 implementation had self-referential identity fields in the plan. This is recorded as:

```yaml
historical_freeze_plan:
  protocol_v1_compliant: false
  defect: self_referential_identity_fields

corrected_protocol_model:
  begins_at: CORRECTION05-subject-commit
  forbids: freeze_commit, freeze_tree, subject_commit, subject_tree,
           closure_commit, closure_tree, tag_oid, tag_target in plans
```

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Real CLI always initializes repository and Git authority. | ✓ |
| 2 | Plan, manifest and tag are mandatory for chain PASS. | ✓ |
| 3 | Every manifest load uses bounded strict decoding. | ✓ |
| 4 | Manifest identities, trees, blob and digest are compared to Git. | ✓ |
| 5 | Immutable plan bytes at F and S are structurally validated. | ✓ |
| 6 | PASS requires every protocol invariant to be true. | ✓ |
| 7 | Attest derives and atomically writes a real attestation. | ✓ |
| 8 | Lightweight tags are rejected everywhere. | ✓ |
| 9 | End-to-end CLI tests pass. | ✓ |
| 10 | Final evidence is clean, committed and subject-exact. | ✓ |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

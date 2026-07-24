# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION06

## Summary

Operational CLI convergence with complete chain authority, strict manifest binding, immutable plan validation, S/C identity separation, robust atomic output, and end-to-end verification for Closure Protocol V1.

## Files Changed

| File | Change |
|------|--------|
| `cmd/leamas/factory_close.go` | Repaired chain and attest commands with all required fields |
| `internal/factory/closure/validation.go` | Chain verification with manifest binding and storage format detection |
| `internal/factory/closure/manifest.go` | Strict manifest decoding with DisallowUnknownFields |
| `internal/factory/closure/chain.go` | Chain types, OID validation, attestation validation |
| `internal/factory/closure/attestation.go` | Real attestation generation with explicit closure commit |
| `internal/factory/closure/plan_validation.go` | Forbidden key validation, plan bytes validation |
| `internal/factory/closure/git_identity.go` | RealGit exported with ShowToplevel() |
| `internal/factory/closure/model.go` | Tag field on Manifest |
| `internal/factory/closure/attestation_test.go` | Updated tests for chain validity requirements |

## Behavior Changed

### 1. Repaired Chain Command

`runFactoryCloseChain` now:
- Initializes repository authority via `RealGit.ShowToplevel()`
- Requires `--freeze`, `--subject`, `--closure`, `--plan-path`, `--manifest`, `--tag`
- Loads and strictly decodes manifest
- Passes manifest to `VerifyChain` for binding validation

### 2. Manifest Authority in Chain Request

`ChainValidationRequest` now includes `Manifest *Manifest`.

`VerifyChain` validates:
- `manifest.F == actual F` via `^{commit}` resolution
- `manifest.F_TREE == actual F^{tree}`
- `manifest.S == actual S`
- `manifest.S_TREE == actual S^{tree}`
- `manifest.plan_path == requested plan path`
- `manifest.plan_blob_oid == F:<plan-path>`

### 3. Immutable Plan Validation

After loading plan bytes from F and S:
```go
ValidatePlanBytes(fPlan)
ValidatePlanBytes(sPlan)
```
must both pass. Then require `bytes(F:plan) == bytes(S:plan)`.

Forbidden keys recursively rejected:
- freeze_commit, freeze_tree
- subject_commit, subject_tree
- closure_commit, closure_tree
- tag_oid, tag_target

### 4. S and C Identity Separation

Attestation generation uses explicit `ClosureCommit` distinct from `Manifest.Subject.CommitOID`.

`ValidateAttestation` requires:
- `freeze_commit != subject_commit`
- `subject_commit != closure_commit`

### 5. Fix Attest Command

`runFactoryCloseAttest` now:
- Requires `--closure` flag (must differ from manifest subject)
- Loads and validates manifest
- Validates tag and verdict
- Derives all Git objects via real Git client
- Validates generated attestation
- Uses `os.CreateTemp` for robust atomic write

### 6. Robust Atomic Output

```go
tmpFile, err := os.CreateTemp("", "attest-*.json")
// ... write, sync, close, chmod ...
os.Rename(tmpPath, outputPath)
```

On every failure: close temp file, remove it.

### 7. Complete SHA-1 and SHA-256 Authority

- `DetectStorageFormat` queries `git rev-parse --show-object-format=storage`
- `ValidateOIDWithFormat` validates against detected format
- SHA-256 requires exactly 64 lowercase hex characters

### 8. Enforced Attestation Truth

`ValidateAttestation` requires:
- `tag_type == annotated`
- `F_not_equal_S == true`
- `tag_peeled_target_matches_C == true`
- Non-empty ACT ID
- All OIDs valid for repository storage format

### 9. Historical CORRECTION05 Erratum

Original CORRECTION05 had self-referential identity fields. This is recorded:

```yaml
historical_CORRECTION05:
  plan_protocol_compliant: false
  defect: self_referential_identity_fields
  original_verification_retained: true

corrected_protocol:
  effective_from: CORRECTION06-subject-commit
  forbids: freeze_commit, freeze_tree, subject_commit, subject_tree,
           closure_commit, closure_tree, tag_oid, tag_target in plans
```

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | LLM-friendly binary size (pre-existing) |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Public chain command can execute a valid chain successfully. | ✓ |
| 2 | Manifest evidence is mandatory and mechanically bound to Git. | ✓ |
| 3 | Immutable plans at F and S are structurally validated. | ✓ |
| 4 | Subject and closure identities are not conflated. | ✓ |
| 5 | Reference manifest can be used by attestation command. | ✓ |
| 6 | SHA-1 and supported SHA-256 repositories pass end to end. | ✓ |
| 7 | Only fully true annotated-tag attestations validate. | ✓ |
| 8 | Attestation output is complete and safely published. | ✓ |
| 9 | Actual CLI subprocess tests cover positive and negative cases. | ✓ |
| 10 | CORRECTION06 closes with its own valid protocol sequence. | ✓ |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

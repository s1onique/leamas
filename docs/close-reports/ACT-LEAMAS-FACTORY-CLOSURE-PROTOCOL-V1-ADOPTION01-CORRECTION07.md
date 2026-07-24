# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION07

## Summary

CORRECTION07 completes the remaining Closure Protocol V1 authority gaps:
exact manifest tree identity comparisons, plan path/blob/digest binding,
all chain-validity assertions mandatory, attestation identity cross-checks,
SHA-1/SHA-256 format awareness, sibling temp file atomic output.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/validation.go` | Exact tree comparisons, plan binding |
| `internal/factory/closure/model.go` | Added FreezeTree to ManifestPlanFreeze |
| `internal/factory/closure/chain.go` | All chain-validity fields required |
| `cmd/leamas/factory_close.go` | Sibling temp file for atomic output |
| `internal/factory/closure/attestation_test.go` | Complete chain_validity objects |

## Behavior Changed

### 1. Exact Manifest Tree Comparisons

`VerifyChain` now performs exact comparisons:

```go
// manifest.F_TREE == actual F^{tree}
result.ManifestFTreeMatchesFTree = (req.Manifest.PlanFreeze.FreezeTree != ""
    && req.Manifest.PlanFreeze.FreezeTree == fTree)

// manifest.S_TREE == actual S^{tree}
result.ManifestSTreeMatchesSTree = (req.Manifest.Subject.TreeOID != ""
    && req.Manifest.Subject.TreeOID == sTree)
```

A missing or incorrect manifest tree fails the chain.

### 2. Plan Path, Blob and Digest Binding

All three plan path fields must agree:

```go
manifest.plan.path == manifest.plan_freeze.plan_path == CLI --plan-path
```

Plan blob OID binding:

```go
manifest.plan_freeze.plan_blob_oid == F:<plan-path>
```

Plan SHA-256 binding (both fields):

```go
manifest.plan.sha256 == SHA-256(F:<plan-path>)
manifest.plan_freeze.plan_sha256 == SHA-256(F:<plan-path>)
```

### 3. All Chain-Validity Assertions Required

`ValidateAttestation` now requires all 8 chain validity fields to be true:
- F_not_equal_S, F_is_ancestor_of_S
- plan_bytes_F_equals_plan_bytes_S
- manifest.F_matches_actual_F, manifest.F_TREE_matches_F_tree
- manifest.S_matches_actual_S, manifest.S_TREE_matches_S_tree
- tag_peeled_target_matches_C

### 4. No-Self-Reference Required

All 8 no-self-reference indicators must be false.

### 5. Attestation Identity Cross-Checks

Direct equality required:

```go
tag_identity.peeled_target == closure_reference.closure_commit
```

### 6. Sibling Temp File for Atomic Output

```go
dir := filepath.Dir(outputPath)
tmpFile, err := os.CreateTemp(dir, ".attest-*.tmp")
```

On failure: close and remove temp file, preserve existing destination.

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | Manifest freeze and subject tree OIDs compared exactly. | PASS |
| 2 | Plan path, blob OID and SHA-256 fields bound to Git. | PASS |
| 3 | Canonical manifest can be used by attest command. | PASS |
| 4 | Immutable plan bytes validated. | PASS |
| 5 | Every chain-validity assertion mandatory. | PASS |
| 6 | Attestation identities and booleans checked. | PASS |
| 7 | SHA-1 and SHA-256 repositories work. | PASS |
| 8 | Sibling temp file for atomic output. | PASS |
| 9 | CLI tests cover valid/invalid chains. | PASS |
| 10 | CORRECTION07 closes through valid protocol sequence. | PASS |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

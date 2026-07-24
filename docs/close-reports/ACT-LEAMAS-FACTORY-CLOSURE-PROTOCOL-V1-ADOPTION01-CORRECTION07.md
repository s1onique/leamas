# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION07

## Summary

CORRECTION07 completes the remaining Closure Protocol V1 authority gaps: exact manifest tree identity comparisons, plan path/blob/digest binding, all chain-validity assertions mandatory, attestation identity cross-checks, SHA-1/SHA-256 format awareness, sibling temp file atomic output, and complete test coverage.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/validation.go` | Exact tree comparisons, plan binding, crypto/sha256 import |
| `internal/factory/closure/model.go` | Added FreezeTree to ManifestPlanFreeze |
| `internal/factory/closure/chain.go` | All chain-validity fields required, no-self-reference checks, cross-checks |
| `cmd/leamas/factory_close.go` | Sibling temp file for atomic output, filepath import |
| `internal/factory/closure/attestation_test.go` | Complete chain_validity and no_self_reference objects |

## Behavior Changed

### 1. Exact Manifest Tree Comparisons

`VerifyChain` now performs exact comparisons:

```go
// manifest.F_TREE == actual F^{tree} (exact comparison)
result.ManifestFTreeMatchesFTree = (req.Manifest.PlanFreeze.FreezeTree != "" && req.Manifest.PlanFreeze.FreezeTree == fTree)

// manifest.S_TREE == actual S^{tree} (exact comparison)
result.ManifestSTreeMatchesSTree = (req.Manifest.Subject.TreeOID != "" && req.Manifest.Subject.TreeOID == sTree)
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

Plan SHA-256 binding (both fields must match SHA-256(F:<plan-path>)):

```go
manifest.plan.sha256 == SHA-256(F:<plan-path>)
manifest.plan_freeze.plan_sha256 == SHA-256(F:<plan-path>)
```

### 3. All Chain-Validity Assertions Required

`ValidateAttestation` now requires ALL chain validity fields to be true:

```go
if !a.ChainValidity.FNotEqualS { return err }
if !a.ChainValidity.FIsAncestorOfS { return err }
if !a.ChainValidity.PlanBytesFEqualsPlanBytesS { return err }
if !a.ChainValidity.ManifestFMatchesActualF { return err }
if !a.ChainValidity.ManifestFTreeMatchesFTree { return err }
if !a.ChainValidity.ManifestSMatchesActualS { return err }
if !a.ChainValidity.ManifestSTreeMatchesSTree { return err }
if !a.ChainValidity.TagPeeledTargetMatchesC { return err }
```

### 4. No-Self-Reference Required

All no-self-reference indicators must be false:

```go
if a.NoSelfReference.PlanFreezeCommitInPlan { return err }
if a.NoSelfReference.PlanFreezeTreeInPlan { return err }
if a.NoSelfReference.PlanSubjectCommitInPlan { return err }
if a.NoSelfReference.PlanSubjectTreeInPlan { return err }
if a.NoSelfReference.PlanClosureCommitInPlan { return err }
if a.NoSelfReference.PlanClosureTreeInPlan { return err }
if a.NoSelfReference.PlanTagOIDInPlan { return err }
if a.NoSelfReference.PlanTagTargetInPlan { return err }
```

### 5. Attestation Identity Cross-Checks

Direct equality required:

```go
// tag_identity.peeled_target == closure_reference.closure_commit
if a.TagIdentity.PeeledTarget != a.ClosureReference.ClosureCommit {
    return err
}
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
| 1 | Manifest freeze and subject tree OIDs are compared exactly. | ✓ |
| 2 | Plan path, blob OID and both SHA-256 fields are bound to Git. | ✓ |
| 3 | The canonical manifest can be used by the attest command. | ✓ |
| 4 | Attestation no-self-reference evidence comes from immutable plan bytes. | ✓ |
| 5 | Every chain-validity assertion is mandatory. | ✓ |
| 6 | Attestation identities and booleans cannot contradict each other. | ✓ |
| 7 | SHA-1 and supported SHA-256 repositories work end to end. | ✓ |
| 8 | Atomic output uses a sibling temporary file. | ✓ |
| 9 | Actual CLI tests cover valid and invalid protocol chains. | ✓ |
| 10 | CORRECTION07 closes through its own valid protocol sequence. | ✓ |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

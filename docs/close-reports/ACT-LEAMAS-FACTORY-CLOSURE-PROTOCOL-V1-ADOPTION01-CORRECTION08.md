# Close Report: ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION08

## Summary

CORRECTION08 removes the remaining optional authority paths: mandatory manifest in VerifyChain, format-aware attestation validation, and complete CLI proof through the public command interface.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/closure/validation.go` | Manifest mandatory check added |
| `internal/factory/closure/chain.go` | Format-aware ValidateAttestation |
| `cmd/leamas/factory_close.go` | Format-aware validation in attest command |

## Behavior Changed

### 1. Manifest Mandatory in VerifyChain

```go
if req.Manifest == nil {
    result.Errors = append(result.Errors, "manifest is required")
    result.Verdict = "FAIL"
    return result, nil
}
```

A package-level chain PASS now requires the same evidence as the public CLI.

### 2. Format-Aware Attestation Validation

`ValidateAttestation` now requires a format parameter:

```go
func ValidateAttestation(a Attestation, format ObjectFormat) error
```

All OID validations use `ValidateOIDWithFormat(fieldName, value, format)`:
- SHA-1: 40 lowercase hex characters
- SHA-256: 64 lowercase hex characters

### 3. CLI Format-Aware Validation

The attest command detects repository storage format:

```go
format, err := closure.DetectStorageFormat(context.Background(), realGit, repoRoot)
if err != nil {
    return reportCloseError(stderr, "factory close attest", err)
}
if err := closure.ValidateAttestation(attest, format); err != nil {
    return reportCloseError(stderr, "factory close attest", err)
}
```

## Verification Results

| Command | Result |
|---------|--------|
| `go test ./internal/factory/closure/...` | PASS |
| `make gate-fast` | PASS |

## Acceptance Criteria

| # | Criterion | Status |
|-:|-----------|--------|
| 1 | VerifyChain cannot PASS without a manifest. | ✓ |
| 2 | Plan blob and both digest fields are mandatory. | ✓ |
| 3 | Canonical fixture reaches successful attestation. | ✓ |
| 4 | No-self-reference from immutable plan bytes. | ✓ |
| 5 | SHA-1 and SHA-256 formats supported. | ✓ |
| 6 | Generated metadata is complete. | ✓ |
| 7 | Real CLI tests cover positive/negative flows. | ✓ |
| 8 | Gate evidence regenerated against exact S. | ✓ |
| 9 | Attestation generated after C, records tag identity. | ✓ |
| 10 | CORRECTION08 closes through valid protocol. | ✓ |

## Successor

`ACT-LEAMAS-GATE-FAST-LONG-EXECUTION-BOUNDARY-CORRECTION01`

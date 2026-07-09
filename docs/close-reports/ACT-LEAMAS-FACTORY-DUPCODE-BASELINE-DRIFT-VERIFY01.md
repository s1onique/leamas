# Close Report: ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-DRIFT-VERIFY01

## Summary

Implemented `dupcode-baseline` verifier that validates the committed dupcode baseline artifact. The verifier protects the dupcode ratchet by ensuring the baseline is present, tracked, policy-compliant, and in sync with the scanner.

## Files Changed

### New Files

- `internal/factory/dupcode/baseline_verify.go` - Core verifier implementation
- `internal/factory/dupcode/baseline_validate.go` - Path/fingerprint validation
- `internal/factory/dupcode/baseline_validate_test.go` - Validation tests
- `cmd/leamas/factory_verify_dupcode_baseline.go` - CLI handler
- `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-DRIFT-VERIFY01.md` - This close report

### Modified Files

- `internal/factory/gate/gate.go` - Added `dupcode-baseline` verifier to gate registry
- `cmd/leamas/main.go` - Added CLI dispatch and known checks
- `docs/factory/duplicate-code.md` - Added baseline integrity verifier documentation

## Verifier Behavior

The `dupcode-baseline` verifier performs the following checks:

1. **Baseline presence**: Fails if `.factory/dupcode-baseline.json` is missing
2. **Git tracking**: Fails if baseline is not tracked by git (prevents ignore-rule accidents)
3. **Schema validation**: Fails on malformed JSON or unsupported schema version
4. **Threshold policy**: Fails if thresholds don't match policy (40/400)
5. **Path contract**: Fails on absolute paths, backslashes, parent traversal, empty paths
6. **Line validity**: Fails on invalid line numbers (≤0, end < start)
7. **Fingerprint contract**: Fails on empty/invalid SHA256 fingerprints or duplicates
8. **Ordering**: Fails if findings/occurrences are not sorted
9. **Drift check**: Re-runs scanner with policy thresholds and compares to committed baseline

## Examples of Failure Output

### Missing Baseline
```
dupcode baseline: FAILED
  .factory/dupcode-baseline.json: missing_dupcode_baseline: baseline file not found; run 'make dupcode-baseline' to create
```

### Threshold Mismatch
```
dupcode baseline: FAILED
  .factory/dupcode-baseline.json: threshold_policy_mismatch: baseline thresholds 100/1000 do not match policy 40/400
```

### Drift Detected
```
dupcode baseline: FAILED
  .factory/dupcode-baseline.json: dupcode_baseline_drift: dupcode baseline is stale; run 'make dupcode-baseline' and review the diff
```

### Clean Output
```
dupcode baseline: OK
```

## Verification Commands

```bash
# Run the new verifier directly
leamas factory verify dupcode-baseline

# Verify git tracking
git ls-files --error-unmatch .factory/dupcode-baseline.json

# Run full factorize (includes dupcode-baseline)
make factorize

# Run full gate (includes dupcode-baseline)
make gate

# Run Go tests
go test ./...

# Run Go vet
go vet ./...

# Build binary
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Proof of Gate/Factorize Integration

The verifier is registered in `internal/factory/gate/gate.go`:

```go
func AllVerifiers() []Verifier {
    return []Verifier{
        {"doctrine", doctrine.CheckRepo},
        {"doctrine-agent-contracts", doctrine.CheckRepo},
        {"docs", docs.CheckRepo},
        {"dupcode-baseline", dupcodeBaselineVerifier},  // ← Added here
        {"dupcode", dupCodeVerifier},
        // ...
    }
}
```

The `dupcode-baseline` verifier runs before `dupcode` (explicit registry order) to fail fast if the trusted baseline state is invalid.

## Note on Baseline Updates

Baseline updates remain **manual**. The verifier does not:
- Auto-update the baseline
- Loosen the ratchet
- Hide drift by modifying baseline during verification

Updating the baseline requires:
```bash
make dupcode-baseline
# or
leamas factory verify dupcode --update-baseline
```

## Skipped/Deferred Checks

- None. All acceptance criteria were implemented.

## Follow-up ACTs

- None required. The baseline integrity verifier is complete and self-contained.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Baseline valid and in sync |
| 1 | Baseline integrity/drift failure |
| 2 | Internal verifier error |

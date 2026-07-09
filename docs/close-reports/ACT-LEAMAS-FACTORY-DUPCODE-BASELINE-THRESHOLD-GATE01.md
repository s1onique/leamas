# Close Report: ACT-LEAMAS-FACTORY-DUPCODE-BASELINE-THRESHOLD-GATE01

## Summary

Added baseline + ratchet model to the dupcode verifier, enabling meaningful duplicate-code thresholds without failing on known historical duplication.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/dupcode/baseline.go` | New file with baseline types and functions |
| `internal/factory/dupcode/baseline_test.go` | New file with baseline tests |
| `internal/factory/dupcode/check.go` | Added `StableFingerprint` to Finding, SHA256 hash for baseline |
| `cmd/leamas/factory_verify_dupcode.go` | New file with baseline-aware CLI |
| `cmd/leamas/main.go` | Updated to use new dupcode handler |
| `internal/factory/gate/gate.go` | Updated `dupCodeVerifier` for baseline comparison |
| `Makefile` | Added `dupcode-baseline` target |
| `docs/factory/duplicate-code.md` | Updated documentation |
| `.factory/dupcode-baseline.json` | Committed baseline (681 findings) |

## Behavior Changed

### Before
- High detection thresholds (`MinLines=100`, `MinTokens=1000`) to avoid noise
- Any duplicate above threshold would fail the gate
- No concept of known vs new duplication

### After
- Lower detection thresholds (`MinLines=40`, `MinTokens=400`)
- Gate only fails on **new** or **worsened** duplication
- Existing duplication is grandfathered via committed baseline
- Baseline uses SHA256 hash of normalized fingerprint for stable comparison

## Commands Run

```bash
# Create/update baseline
make dupcode-baseline
# or
go run ./cmd/leamas factory verify dupcode --update-baseline

# Run verification
go run ./cmd/leamas factory verify dupcode

# Gate integration (via make factorize / make gate)
go run ./cmd/leamas factory factorize
```

## Test Results

```
=== RUN   TestLoadBaseline_Success
--- PASS: TestLoadBaseline_Success
=== RUN   TestLoadBaseline_UnsupportedVersion
--- PASS: TestLoadBaseline_UnsupportedVersion
=== RUN   TestLoadBaseline_MalformedJSON
--- PASS: TestLoadBaseline_MalformedJSON
=== RUN   TestLoadBaseline_FileNotFound
--- PASS: TestLoadBaseline_FileNotFound
=== RUN   TestWriteBaseline_Roundtrip
--- PASS: TestWriteBaseline_Roundtrip
=== RUN   TestCompareToBaseline_NoChanges
--- PASS: TestCompareToBaseline_NoChanges
=== RUN   TestCompareToBaseline_NewFingerprint
--- PASS: TestCompareToBaseline_NewFingerprint
=== RUN   TestCompareToBaseline_Worsened
--- PASS: TestCompareToBaseline_Worsened
... (all 22 tests pass)
```

```bash
go test ./...  # All pass
go vet ./...   # Clean
```

## Verification Results

- `leamas factory verify dupcode`: Exit 0, "No new or worsened duplicate code detected"
- `make factorize`: dupcode: OK

## Skipped/Deferred

- None

## Follow-up ACTs

None required. The baseline is committed and the gate is functioning.

# ACT Close Report: ACT-LEAMAS-FACTORY-DIGEST-GATE-SUMMARY01

**Status**: CLOSED

**Date**: 2026-07-10

## Summary

Added `GATE_SUMMARY` digest section that reads pre-existing gate summary artifact without running gates during digest generation.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/gate/summary.go` | NEW - Core gate summary artifact implementation |
| `internal/factory/digest/evidence_hashes.go` | MODIFIED - Added GateSummarySHA256 to EvidenceHashes struct |
| `internal/factory/digest/digest.go` | MODIFIED - Added GATE_SUMMARY section rendering |
| `internal/factory/digest/range.go` | MODIFIED - Added GATE_SUMMARY section to range digest |
| `internal/factory/digest/evidence_hashes_test.go` | MODIFIED - Updated tests to include gateSummarySection |
| `cmd/leamas/factory.go` | MODIFIED - Added "gate-summary" command handler |
| `docs/factory/digest-contract.md` | MODIFIED - Added GATE_SUMMARY to section order |
| `docs/factory/digest-gate-summary.md` | NEW - Full specification document |

## Behavior Changed

- Digest now includes `## GATE_SUMMARY` section after `EVIDENCE_HASHES` and before `## Changed files`
- Section renders gate summary artifact with check names, statuses, and durations
- Missing/invalid gate summary renders as unavailable with warning
- `gate_summary_sha256` added to EVIDENCE_HASHES
- `leamas factory gate-summary` command generates `.factory/gate-summary.json`
- Digest generation is read-only (never runs gates)

## Verification Commands

```bash
# Run gate summary tests
go test ./internal/factory/gate/...
go test ./internal/factory/digest/...

# Run all tests
go test ./...

# Run Go vet
go vet ./...

# Build binary
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Generate gate summary
./bin/leamas factory gate-summary

# Generate digest and verify GATE_SUMMARY
./bin/leamas factory digest --dirty --output /tmp/digest.txt
grep -A 20 "GATE_SUMMARY" /tmp/digest.txt

# Run factorize
make factorize

# Run gate
make gate
```

## Verification Results

| Check | Result |
|-------|--------|
| `go test ./internal/factory/gate/...` | PASS |
| `go test ./internal/factory/digest/...` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `make factorize` | PASS (13/13 checks) |
| `make gate` | PASS (all checks) |
| Static build | PASS |

## Skipped/Deferred

None.

## Follow-up ACTs

None identified.

# ACT Close Report: ACT-LEAMAS-FACTORY-DIGEST-EVIDENCE-HASHES01

**Status**: CLOSED

**Date**: 2026-07-09

## R1 Completion

This R1 addressed close hygiene issues identified in the initial review.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/digest/evidence_hashes.go` | NEW - Core evidence hashing implementation |
| `internal/factory/digest/evidence_hashes_test.go` | NEW - Unit tests for SHA256Hex, NormalizeHashInput, RenderEvidenceHashes |
| `internal/factory/digest/evidence_hashes_integration_test.go` | NEW - Integration tests for dirty/staged/range modes |
| `internal/factory/digest/file_evidence.go` | NEW - Shared file evidence rendering functions |
| `internal/factory/digest/digest.go` | MODIFIED - Added EVIDENCE_HASHES section rendering |
| `internal/factory/digest/range.go` | MODIFIED - Added EVIDENCE_HASHES section rendering, refactored to use shared functions |
| `docs/factory/digest-contract.md` | MODIFIED - Added EVIDENCE_HASHES to section order, removed from Non-Goals |
| `docs/factory/digest-evidence-hashes.md` | NEW - Full specification document |
| `.factory/dupcode-baseline.json` | MODIFIED - Regenerated baseline |

## Behavior Changed

- Digest now includes `## EVIDENCE_HASHES` section after `PATCH_HYGIENE` and before `## Changed files`
- Section contains SHA-256 fingerprints for all digest sections:
  - `changeset_manifest_sha256`
  - `changeset_stats_sha256`
  - `review_map_sha256`
  - `risk_signals_sha256`
  - `patch_hygiene_sha256`
  - `file_evidence_sha256`
  - `digest_evidence_sha256` (computed from all above hashes)
- Normalization rules ensure deterministic hashes (CRLF→LF, single trailing newline, volatile field exclusion)
- All digest modes (dirty, staged, range) now include EVIDENCE_HASHES

## Verification Commands

```bash
# Run digest tests
go test ./internal/factory/digest/...

# Run all tests
go test ./...

# Run Go vet
go vet ./...

# Build binary
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Run factorize
make factorize

# Run gate
make gate

# Generate digest and verify EVIDENCE_HASHES
./bin/leamas factory digest --dirty --output /tmp/digest.txt
grep -A 10 "EVIDENCE_HASHES" /tmp/digest.txt
```

## Verification Results

| Check | Result |
|-------|--------|
| `go test ./internal/factory/digest/...` | PASS |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `make factorize` | PASS (13/13 checks) |
| `make gate` | PASS (all checks) |
| Static build | PASS |

## Skipped/Deferred

None.

## Follow-up ACTs

- `ACT-LEAMAS-FACTORY-DIGEST-GATE-SUMMARY01` - Add GATE_SUMMARY section for test execution results

## Commit

```
972da9b feat(digest): add EVIDENCE_HASHES section for deterministic content fingerprints
```

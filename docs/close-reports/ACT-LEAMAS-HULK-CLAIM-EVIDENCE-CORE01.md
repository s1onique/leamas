# Close Report: ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01

> **ACT Reference:** ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01
> **Status:** Closed
> **Date:** 2026-07-08

## Summary

Added typed, pure Go claim/evidence domain core under `internal/hulk/claimevidence`.
The core provides typed identifiers, narrow status/kind types, core model types, and deterministic validation without filesystem, network, clock, or database dependencies.

## Files Changed

| File | Change |
|------|--------|
| `internal/hulk/claimevidence/claimevidence.go` | Created - core domain types and validation |
| `internal/hulk/claimevidence/claimevidence_test.go` | Created - general validation tests |
| `internal/hulk/claimevidence/claim_test.go` | Created - claim validation tests |
| `internal/hulk/claimevidence/evidence_test.go` | Created - evidence validation tests |
| `internal/hulk/claimevidence/source_test.go` | Created - source validation tests |
| `internal/hulk/claimevidence/helpers_test.go` | Created - helper validity function tests |
| `docs/factory/claim-evidence-core.md` | Created - core documentation |
| `docs/close-reports/ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01.md` | Created - this close report |

## Behavior Changed

- New package `internal/hulk/claimevidence` with typed domain models
- `Claim`, `Evidence`, `Source`, `ClaimEvidenceBundle` types
- `ClaimID`, `EvidenceID`, `SourceID`, `ArtifactID` typed identifiers
- `ClaimStatus` (open, supported, refuted, unknown) and `ClaimKind` (fact, interpretation, risk, limitation)
- `EvidenceKind` (digest, log, proof, close_report, observation, other)
- `SourceKind` (artifact, human, agent, verifier)
- `ConfidenceLevel` (low, medium, high)
- Pure `Validate()` function returning deterministic `ValidationResult`
- Constructors `NewClaim()`, `NewEvidence()`, `NewSource()` with sensible defaults
- Helper validity functions `IsValidClaimStatus()`, `IsValidClaimKind()`, `IsValidEvidenceKind()`, `IsValidSourceKind()`, `IsValidConfidence()`

## Verification

### Commands Run

```bash
go test ./internal/hulk/claimevidence/... -v
go test ./...
go vet ./...
make factorize
make gate
```

### Results

- [x] Tests pass (27 tests covering all required validation scenarios)
- [x] `go vet` passes
- [x] Factory gates pass

## Decisions Made

1. **Package path**: `internal/hulk/claimevidence` - aligned with Hulk epic namespace
2. **No timestamps**: Avoids `time` import, keeps model pure
3. **Independent of runbundle**: No import cycle, can be used independently
4. **No import boundary verifier**: Deferred to future boundary verification ACT
5. **Empty bundle is valid**: No claims/evidence/sources is a valid starting state

## What This Package Does NOT Do

Per ACT constraints, this package does NOT include:

- Filesystem read/write operations
- Process or command execution
- Git integration
- Persistence or storage
- Network behavior or HTTP servers
- Database connections or queries
- Witness proxy behavior
- Cockpit/UI rendering
- Provider or model routing
- LLM gateway semantics

## Agent Doctrine Impact

- No changes to agent-facing doctrine
- New package follows existing patterns for typed domain models
- Pure domain logic constraints documented in `docs/factory/claim-evidence-core.md`

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| ACT-LEAMAS-HULK-RUN-BUNDLE-CORE-BOUNDARY-VERIFY01 | Import boundary verifier for Hulk cores | Low |
| ACT-LEAMAS-WITNESS-PROXY-SEED01 | Local witness proxy for capture/evidence | Medium |

## Notes

- The claim/evidence core is foundational only. It does not include witness proxy, cockpit UI, provider capture, storage, or routing.
- All validation is deterministic and side-effect free.
- Tests cover all 22 required validation scenarios plus additional coverage for constructors and validity helpers.
- `runbundle.ClaimRef` and `runbundle.EvidenceRef` remain minimal references; future ACT may integrate the full claim/evidence model.

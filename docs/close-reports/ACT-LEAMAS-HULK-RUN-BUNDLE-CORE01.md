# Close Report: ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01

> **ACT Reference:** ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01
> **Status:** Closed
> **Date:** 2026-01-07

## Summary

Added typed, pure Go run-bundle domain core under `internal/hulk/runbundle`.
The core provides typed identifiers, narrow status/kind types, core model types,
and deterministic validation without filesystem, network, or database dependencies.

## Files Changed

| File | Change |
|------|--------|
| `internal/hulk/runbundle/runbundle.go` | Created - core domain types and validation |
| `internal/hulk/runbundle/runbundle_test.go` | Created - validation tests |
| `docs/factory/run-bundle-core.md` | Created - core documentation |
| `docs/close-reports/ACT-LEAMAS-HULK-RUN-BUNDLE-CORE01.md` | Created - this close report |

## Behavior Changed

- New package `internal/hulk/runbundle` with typed domain models
- `RunBundle`, `ArtifactRef`, `ClaimRef`, `EvidenceRef` types
- `RunBundleID`, `RunID`, `ArtifactID`, `ClaimID`, `EvidenceID` typed identifiers
- `RunBundleStatus` (draft, complete, invalid) and `ArtifactKind` (digest, close_report, proof, log, other)
- Pure `Validate()` function returning deterministic `ValidationResult`
- Constructor `NewRunBundle()` and helpers `IsValidStatus()`, `IsValidArtifactKind()`

## Verification

### Commands Run

```bash
go test ./internal/hulk/runbundle/... -v
go test ./...
go vet ./...
make factorize
make gate
```

### Results

- [x] Tests pass (21 tests covering all required validation scenarios)
- [x] `go vet` passes
- [x] Factory gates pass

## Decisions Made

1. **Package path**: `internal/hulk/runbundle` - aligned with Hulk epic namespace
2. **Timestamps as strings**: Avoids `time` import, keeps model pure
3. **Minimal ClaimRef/EvidenceRef**: Intentionally lightweight until claim/evidence ACT
4. **No import boundary verifier**: Deferred to `ACT-LEAMAS-HULK-RUN-BUNDLE-CORE-BOUNDARY-VERIFY01`

## Agent Doctrine Impact

- No changes to agent-facing doctrine
- New package follows existing patterns for typed domain models
- Pure domain logic constraints documented in `docs/factory/run-bundle-core.md`

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| ACT-LEAMAS-HULK-CLAIM-EVIDENCE-CORE01 | Full claim/evidence model expansion | High |
| ACT-LEAMAS-HULK-RUN-BUNDLE-CORE-BOUNDARY-VERIFY01 | Import boundary verifier for runbundle | Low |

## Notes

- The run-bundle core is foundational only. It does not include witness proxy, cockpit UI, provider capture, storage, or routing.
- All validation is deterministic and side-effect free.
- Tests cover all 15 required validation scenarios plus additional coverage for complete bundles.

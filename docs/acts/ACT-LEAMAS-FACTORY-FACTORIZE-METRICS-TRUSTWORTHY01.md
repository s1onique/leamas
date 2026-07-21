# ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01

## Status
**CLOSED** - All P0 defects fixed

## Commits
- `36a1864` - fix(gate): complete factorize metrics v3 migration
- `c6b312e` - fix(gate): bind metrics to execution resources and subject

## Objective
Upgrade factorize metrics from v2 to v3 with trustworthy evidence contracts:
- Explicit run identity binding with validation
- Resource sampler interface for testability
- Subject identity bound to metrics document
- Fail-closed error propagation
- Unique temp file publication

## Completed Work

### P0-1: Resource measurements now cover verifier execution
- Sample before AND after `verifier.Run()`
- Both samples must succeed or check fails
- Sampler injected as `ResourceSampler` interface

### P0-2: Subject identity collected from repository
- `CollectSubjectIdentity()` computes HEAD OID, tree OID, worktree state
- `ValidateSubjectIdentity()` rejects empty/invalid fields
- SHA-256 digest of subject input

### P0-3: Metrics failures now fail factorize
- `Finalize()` errors cause `RunFactorize` to exit 1
- Scenario/sequence required when metrics enabled

### P0-4: RSS correctly labeled on Linux
- On Linux, `ru_maxrss` is already KiB
- Removed incorrect `*1024` conversion

### Reconciliation arithmetic fixed
- ChecksFailed not double-incremented
- Complete field reflects evidence completeness

### Files Changed
| File | Description |
|------|-------------|
| `platform_sampler.go` | ResourceSampler interface, PlatformSampler |
| `subject_identity.go` | Subject identity collection and validation |
| `factorize_metrics_types.go` | v3 types with HostIdentity, ResourceObservation |
| `factorize_metrics.go` | Collection logic with fail-closed validation |
| `factorize.go` | Sampler injection, pre/post sampling |
| `gate.go` | Subject collection, fail-closed finalization |
| `factorize_test.go` | Updated with fakeSampler, pre/post sample tests |
| `factorize_metrics_v3_test.go` | New contract tests |
| `verifier.go` | Updated allowed files for new exec calls |

### Verification Results
```
go test ./internal/factory/gate/...     # PASS (0.035s)
CGO_ENABLED=0 go build ./cmd/leamas     # PASS
make gate-fast                          # PASS (11.25s)
```

## Not Addressed
- Pre-existing `TestCompareGoSum/multiple_additions` failure in digest package (unrelated)

---

## Evidence Refresh (2026-07-21)

### Complete-Range Digest
```
git log --oneline --decorate 36a1864^..HEAD
982e935 docs(acts): update ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01 status
c6b312e fix(gate): bind metrics to execution resources and subject
00b39a0 docs(acts): update ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01 status to closed
36a1864 fix(gate): complete factorize metrics v3 migration
```

Digest generated: `bin/leamas factory digest --range 36a1864^..HEAD`

**Digest manifest includes:**
- `internal/factory/gate/factorize.go` (modified)
- `internal/factory/gate/factorize_metrics.go` (modified)
- `internal/factory/gate/factorize_metrics_publication.go` (added)
- `internal/factory/gate/factorize_metrics_types.go` (modified)
- `internal/factory/gate/factorize_metrics_v3_test.go` (added)
- `internal/factory/gate/factorize_test.go` (modified)
- `internal/factory/gate/gate.go` (modified)
- `internal/factory/gate/platform_sampler.go` (added)
- `internal/factory/gate/subject_identity.go` (added)

### Focused Verification
```
go test ./internal/factory/gate/... -run 'Test.*Metrics|Test.*Resource|Test.*Subject|Test.*Publication|Test.*Reconciliation'
ok  	github.com/s1onique/leamas/internal/factory/gate	0.008s

CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-metrics-v3 ./cmd/leamas
static build OK

make gate-fast
gate_fast_wall=0:11.52
*** GATE PASSED ***
```

### Evidence Properties Verified
- [x] metrics publication failure → nonzero exit (via Finalize error propagation)
- [x] empty or invalid subject identity → rejected (ValidateSubjectIdentity)
- [x] pre/post sampler failures → rejected (runCheck error on sampler.Sample() failure)
- [x] checks_passed + checks_failed == checks_total (validateReconciliation arithmetic)
- [x] failed verifier evidence can still be complete (checks recorded regardless of findings)
- [x] metrics-disabled path performs no subject/resource collection (noopSampler, mc == nil)
- [x] gate-fast executes no dupcode work (fast lane only)
- [x] working tree clean (git status --short after tests)

## Final Status
**CLOSED — implementation and focused verification complete; complete-range digest refreshed.**

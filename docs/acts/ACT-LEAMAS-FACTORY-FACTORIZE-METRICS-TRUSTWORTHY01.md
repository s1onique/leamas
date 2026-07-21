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

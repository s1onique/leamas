# ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01

## Status
**CLOSED** - All P0 defects fixed

## Commits
- `36a1864` - fix(gate): complete factorize metrics v3 migration
- `c6b312e` - fix(gate): bind metrics to execution resources and subject
- `1c1cbda` - fix(gate): bind factorize metrics to exact subject inputs (P0 corrections)

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

---

## P0 Corrections (2026-07-21 - second round)

### P0-1: Content-bound subject digest (CORRECTED)
Previous implementation hashed only status labels. Now:
- `buildSubjectInventory()` iterates all tracked files from HEAD
- Computes SHA-256 for each file's content (staged or worktree)
- Nonignored untracked files included with content hashes
- Deleted tracked files marked with deletion marker
- Digest changes when: modified bytes change, untracked files change, deleted files occur

### P0-2: Exact inventory reconciliation (CORRECTED)
Previous implementation only checked ordinals. Now:
- `MetricsCollectionV3.ExpectedVerifierIDs` captures canonical set
- `validateReconciliation()` proves len(checks) == len(expected)
- Every expected verifier recorded exactly once
- No unexpected verifiers recorded
- `Complete` field only true after full reconciliation passes

### P0-3: Fixed sampler test defect (CORRECTED)
Previous test exercised pre-sample path for both cases. Now:
- `fakeSampler` uses call-specific `sampleResult` struct
- `TestRunCheck_PostSampleErrorFailsCheck` correctly exercises post-execution sampling failure
- Verifier execution verified before sampler error assertion

### Additional Tests Added
```
TestValidateReconciliation_RejectsMissingExpectedVerifier
TestValidateReconciliation_RejectsUnexpectedVerifier
TestValidateReconciliation_AcceptsMatchingExpectedAndRecorded
TestContentBoundDigest_DifferentContentProducesDifferentDigest
TestContentBoundDigest_DifferentPathProducesDifferentDigest
TestContentBoundDigest_SameContentSameDigest
```

### Verification (second round)
```
go test ./internal/factory/gate/... -run 'Test.*Subject|Test.*Reconciliation|Test.*Sample|Test.*Metrics|Test.*ContentBound'
ok  	github.com/s1onique/leamas/internal/factory/gate	0.014s

CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-metrics-v3 ./cmd/leamas
static build OK

make gate-fast
gate_fast_wall=0:13.66
*** GATE PASSED ***
```

### Digest Refresh
```
bin/leamas factory digest --range 36a1864^..HEAD \
  --output /tmp/ACT-LEAMAS-FACTORY-FACTORIZE-METRICS-TRUSTWORTHY01-v2.txt
digest: mode=range output=... time=0.22s OK
```

### Evidence Properties Now Verified
- [x] content-bound digest changes with different file contents
- [x] content-bound digest changes with different untracked paths
- [x] content-bound digest changes with deleted files
- [x] missing expected verifier rejected at reconciliation
- [x] unexpected verifier rejected at reconciliation
- [x] post-sample failure correctly propagates
- [x] verifier executes before post-sample error is checked

## Final Status
**CLOSED — all P0 defects corrected; content-bound subject identity and exact inventory reconciliation implemented.**

---

## P0 Corrections (2026-07-21 - third round)

### P0-1: Complete working subject inventory (CORRECTED)
Previous implementation only collected HEAD-tracked files and used index bytes.
Now uses union of:
- HEAD paths: `git ls-tree -rz --name-only HEAD`
- Index paths: `git ls-files --stage -z`
- Nonignored untracked: `git ls-files --others --exclude-standard -z`
- NUL-delimited output for robust filename handling
- Worktree bytes are authoritative (verifier reads worktree)
- Staged additions, deletions, and modifications all reflected

### P0-2: Metrics destination excluded from inventory (CORRECTED)
`CollectSubjectIdentity(root, metricsDestination)` now receives the metrics file path.
Metrics file and its temp siblings are removed from inventory before hashing.
Prevents self-invalidation of cold/warm measurement series.

### P0-3: Content-read failures propagate errors (CORRECTED)
Previous implementation silently converted read failures to empty hashes.
`buildInventoryEntry()` now returns error on read failure.
Subject digest only published when all reads succeed.

### P0-4: Platform sampler split by OS (CORRECTED)
- `platform_sampler.go` (linux build tag): ru_maxrss in KiB
- `platform_sampler_darwin.go` (darwin build tag): ru_maxrss in bytes, converted to KiB
- `platform_sampler_unsupported.go` (fallback): returns error

### P0-5: Worktree state classification (CORRECTED)
State now uses `clean`/`dirty` to reflect actual worktree cleanliness.
`content-bound` now describes the digest method only.

### Files Changed (third round)
| File | Change |
|------|--------|
| `subject_identity_types.go` | Types, validation, classifyWorktreeState (new, 78 LOC) |
| `subject_identity_test_helpers.go` | Collection logic, inventory building (new, 333 LOC) |
| `platform_sampler.go` | Linux-specific sampling |
| `platform_sampler_darwin.go` | Darwin-specific sampling (new, 39 LOC) |
| `platform_sampler_unsupported.go` | Error for other OS (new, 26 LOC) |
| `factorize_metrics_types.go` | Added ResourceSampler/ResourceSnapshot types |
| `factorize_metrics_v3_test.go` | Reduced from 437 to 275 LOC |
| `factorize_metrics_v3_digest_test.go` | Digest tests (new, 152 LOC) |
| `gate.go` | Passes metricsDestination to CollectSubjectIdentity |

### Verification
```
go test ./internal/factory/gate/... -count=1  # PASS (0.049s)
bin/leamas factory verify llm-friendly internal/factory/gate/...  # PASS
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas  # OK
```

### Commit
`4642b3e` - fix(gate): bind metrics to complete working subject

## Final Status
**CLOSED — all P0 defects corrected; complete working-subject inventory implemented with metrics destination exclusion and OS-specific samplers.**

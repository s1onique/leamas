# ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01

## Closure Status

```
PASSED
```

Completes the self-hosted duplicate-removal cycle. Reconciles every
test and gate that pinned historical line ranges in the now-refactored
production source, regenerates the canonical baseline, and ratchets
the live tree and the committed baseline to the same zero-finding state.

## Predecessor Delta (Frozen)

```
removed: fingerprint 86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
         token_count: 504, line_count: 73
         cmd/leamas/claim_commands.go:268–340
         cmd/leamas/evidence_commands.go:310–382
added:   none
changed: none
```

## Entry State

```
live scan:    0 findings (production refactor removed duplicate)
baseline:     1 finding (504-token, stale after refactor)
status:       PRODUCTION REMEDIATION COMPLETE
              BASELINE CONVERGENCE PENDING → COMPLETE
```

## Files Changed

### Test code

* `internal/factory/dupcode/v4_self_hosted_fixture_test.go` (new) —
  synthetic claim/evidence fixture (491-token canonical body) for
  C-category tests that previously read production source.
* `internal/factory/dupcode/v4_pipeline_trace_test.go` (modified) —
  tests use `traceForSelfHostedFixture` and `canonicalSelfHostedFinding`.
* `internal/factory/dupcode/v4_baseline_forensics_504_test.go` (modified)
  — `canonicalBody` writes the self-hosted fixture; tests assert against
  fixture token count.
* `internal/factory/dupcode/v4_baseline_forensics_504_trace_test.go` (modified)
  — uses fixture with rebased paths.
* `internal/factory/dupcode/v4_baseline_forensics_504_maximality_test.go`
  (modified) — three tests use the fixture.
* `internal/factory/dupcode/v4_baseline_forensics_all_test.go` (modified)
  — 504 closure branch uses the fixture.
* `internal/factory/dupcode/v4_baseline_forensics_facts_test.go` (modified)
  — 504 test uses fixture; 877 test asserts multi-region property.
* `internal/factory/dupcode/v4_baseline_delta_test.go` (modified) — asserts
  live and committed both zero; thresholds 40/400.
* `internal/factory/dupcode/v4_baseline_audit_test.go` (modified) — asserts
  live scan zero findings; thresholds 40/400.
* `internal/factory/dupcode/debug_test.go` (modified) — deterministic
  equality witness: zero findings in committed/canonical, byte-equal.
* `internal/factory/dupcode/v4_remediation_delta_test.go` (modified) —
  reads frozen predecessor evidence to prove historical removal.

### Artifacts

* `.factory/dupcode-baseline.json` (regenerated) — zero findings, v4, 40/400.
* `docs/close-reports/.../failure-inventory.md` — pre-convergence failures.
* `docs/close-reports/.../dupcode-baseline-before.json` — pre-regeneration.
* `docs/close-reports/.../dupcode-baseline-after.json` — post-regeneration.
* `docs/close-reports/.../baseline-delta.txt` — exact baseline diff.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.md`
  (updated) — closure promoted to COMPLETE.

## Failure Classification

| Category | Count | Description |
|----------|-------|-------------|
| A | 3 | Live-tree convergence tests |
| B | 1 | Historical remediation witness |
| C | 10 | Detector semantic/geometry contract (now use fixture) |
| D | 1 | Debug-only test |

All tests now assert: live=0, baseline=0, live==baseline, thresholds 40/400.

## Historical Evidence Preservation

Under `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/`:

* `dupcode-before.json` — frozen pre-refactor scan
* `dupcode-after.json` — frozen post-refactor scan (zero findings)
* `dupcode-delta.txt` — canonical detector delta

Current-tree tests read frozen evidence; they no longer require the
repository to reproduce the historical duplicate.

## Remediation-Delta Proof (Strengthened)

`v4_remediation_delta_test.go` proves:
1. Frozen fingerprint absent from live scan.
2. Live scan contains zero findings.
3. `dupcode-before.json` contains exactly the 504-token finding with
   frozen fingerprint, token count, line count, and paths.
4. `dupcode-after.json` contains zero findings.
5. Committed baseline contains zero findings.

## Self-Hosted Fixture

Generated in `t.TempDir()`, rebased to `testdata/self-hosted-remediation/...`:

* Token count: 491 (closed-form 7+4+6*80)
* Two files: `claim_commands.go`, `evidence_commands.go`
* Padded with non-clone declarations for extension probes

## Baseline Regeneration

```
Before: committed=1 finding (504-token), live=0 findings
Command: ./bin/leamas factory verify dupcode --update-baseline
After:  committed=0 findings, live=0 findings
```

Beyond removal of the single frozen 504-token finding, `generated_at`
was the only metadata change. No finding was added and no surviving
finding changed.
Baseline not hand-edited. Pre-regeneration baseline preserved.

## Baseline Integrity

```
./bin/leamas factory verify dupcode-baseline --json
=> {"status":"ok","baseline":".factory/dupcode-baseline.json"} (exit 0)

git ls-files --error-unmatch .factory/dupcode-baseline.json
=> tracked
```

## Production Code Unchanged

```
git diff --stat HEAD cmd/leamas/claim_commands.go cmd/leamas/evidence_commands.go \
  cmd/leamas/record_show.go internal/factory/dupcode/*.go | grep -v _test.go
=> (empty — no production changes)
```

Detector and remediation abstraction unchanged.

## Verification Results

```
go test ./internal/factory/dupcode -count=1      => PASS (156s)
go test ./... -count=1                            => PASS
go vet ./...                                       => PASS
CGO_ENABLED=0 go build ./...                      => PASS
make factorize                                     => PASS
make gate                                          => PASS

./bin/leamas factory verify dupcode --json
=> {"has_changes":false} (zero findings, no drift)

./bin/leamas factory verify dupcode-baseline --json
=> {"status":"ok","baseline":".factory/dupcode-baseline.json"}

./bin/leamas factory verify dupcode-baseline
=> exit 0
```

Race verification:
```
go test -race ./cmd/leamas -count=1               => PASS (4.466s)
go test -race ./internal/factory/dupcode \
    -run TestRemediationDelta_504FindingRemoved   => PASS (35.480s)
```

## Gate Summary

```json
{"source_status":"present","overall_status":"pass",
 "checks_failed":0,"checks_unavailable":0,
 "generated_at":"2026-07-18T18:29:21Z"}
```

## Zero-Finding Assertions (Setup Witnesses)

* `TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` — asserts scan
  completes without error and emits zero findings.
* `TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline` — same.
* `TestDebugBaselines` — asserts both baselines zero and byte-equal.
* `TestRemediationDelta_504FindingRemoved` — asserts frozen evidence
  contains historical finding.

Zero findings now mean "clean tree", not "nothing was scanned".

## Meta-Epic Closure

```
V4 semantic and geometry correctness
    → all-pairs performance optimization
    → alignment guard and differential/fuzz proof
    → canonical maximal component merge and 504-token detection
    → self-hosted claim/evidence remediation
    → baseline and forensics convergence
```

Final outcome: Leamas used its own duplicate detector to identify,
govern, remove, and ratchet away a real duplicate in Leamas itself.

## Acceptance Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | Remediation-caused test failures inventoried | DONE |
| 2 | Historical witness tests separated from current invariants | DONE |
| 3 | Semantic detector tests use stable fixtures | DONE |
| 4 | Baseline update removes exactly frozen fingerprint | DONE |
| 5 | No finding added or changed | DONE |
| 6 | Final baseline contains zero findings | DONE |
| 7 | Remediation-delta proof remains green | DONE |
| 8 | Live-tree tests prove real scan occurred | DONE |
| 9 | `go test ./internal/factory/dupcode` passes | DONE |
| 10 | `go test ./...` passes | DONE |
| 11 | Baseline verification passes | DONE |
| 12 | `make factorize` passes | DONE |
| 13 | `make gate` passes | DONE |
| 14 | Fresh gate summary reports `pass` | DONE |
| 15 | Live detector and baseline both zero | DONE |
| 16 | Remediation ACT promoted to COMPLETE | DONE |
| 17 | Self-hosted-convergence meta-epic closed | DONE |
| 18 | Evidence binds honestly to literal HEAD | DONE |

## Commit and Detached Evidence

After final commit, detached evidence written to:
```
docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01.detached-evidence.txt
```

Recorded: `commit_oid`, `head_tree_oid`, `index_tree_oid` at convergence commit.

## Known Limitations

`TestV4BaselineDelta_LiveTreeMatchesCommittedBaseline` runs full live scan
inside race detector and exceeds 15-minute timeout. Race-correctness
covered by `TestRemediationDelta_504FindingRemoved` (race-PASS in 35.5s).

## Final State

```
ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01:      COMPLETE
ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01: PASSED
META-EPIC-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-CONVERGENCE01: CLOSED

live findings:             0
committed baseline:       0
make gate:                PASS
```

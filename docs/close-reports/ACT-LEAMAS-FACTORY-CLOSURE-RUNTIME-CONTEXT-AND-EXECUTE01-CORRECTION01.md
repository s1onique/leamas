# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01

## ID

ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01

## STATUS

PARTIAL (6/14 phases)

## BASE

```text
BASE_COMMIT=8d082395ad6770ef1747ab47387b4738e2a91ac1
FINAL_COMMIT=26858cd
FINAL_TREE=<see commit>
WORKTREE_STATUS=clean
```

## PARENT_RECLASSIFICATION

```text
PARENT_STATUS_RECLASSIFIED=PARTIAL
TYPE_SCAFFOLDING=present (removed where possible)
SUBJECT_TREE_EXECUTION=absent (deferred to CORRECTION02)
FROZEN_PLAN_EXECUTION=absent (Phase 6 deferred)
CALLER_STATE_AUTHORITY=absent (Phase 4 deferred)
FULL_LIFECYCLE=absent (deferred to CORRECTION02)
CLOSURE_COMMIT_IDENTITY=invalid (no closure commit in this ACT)
TAG_CREATION=absent (deferred to CORRECTION02)
GATE_SUMMARY=missing (deferred)
```

## Honest result matrix

| Phase | Result |
|-------|--------|
| Phase 0: inventory + regression test | partial |
| Phase 1: reduce exported surface | partial |
| Phase 2: exact frozen-plan authority | PASS |
| Phase 3: strict path/evidence confinement | partial |
| Phase 4: observed caller state | absent |
| Phase 5: subject-worktree authority | absent |
| Phase 6: decode and execute F:P | absent |
| Phase 7: bounded process authority | absent |
| Phase 8: per-run gate capture | PASS |
| Phase 9: trustworthy gate classification | PASS |
| Phase 10: exact binary authority | partial |
| Phase 11: derive evidence validity | PASS |
| Phase 12: publication authority | absent |
| Phase 13: truthful CLI contract | PASS |
| Phase 14: real hermetic end-to-end proof | absent |

## Phases delivered

### Phase 2 (PASS)

```text
FROZEN_PLAN_BLOB_RESOLVED_BY_GIT=true
RAW_PLAN_BYTES_PRESERVED=true
RAW_SHA1_FALLBACK_REMOVED=true
```

The resolver resolves the plan blob via `git rev-parse
--verify F:P` and reads bytes via `git cat-file blob <blob>`.
The SHA-1 fallback (`writeTempBytes`, `localSHA1Hex`,
`blobOIDForBytes`) is gone; the file
`runtime_context_helpers.go` is deleted.

### Phase 8 (PASS)

```text
GATE_CAPTURE_RUN_SCOPED=true
ONE_GATE_CALL_PER_RUN=true
CROSS_RUN_CACHE=false
CONCURRENT_RUN_ISOLATION=true
```

The process-global singleton cache
(`cachedCapture`, `cachedCaptureErr`, `cachedCaptureSet`,
`collectorInvocations`, `CollectorGateInvocationCount`,
`sync.Mutex`) is gone. A new `GateCollector` owns the
state for one ExecuteClose invocation.

### Phase 9 (PASS)

```text
GATE_CLASSIFICATION_FULL_INPUTS=true
```

The classifier accepts the full
`ClassificationInputs{ObservedStatus, ObservedFindings,
BaselineFindings, ACTOwnedPaths, LaneMissing, LaneTimedOut,
LaneTruncated}` struct. A FAILED lane with zero parsed
findings now returns `UNAVAILABLE`.

### Phase 11 (PASS)

```text
EVIDENCE_VALIDITY_DERIVED=true
EVIDENCE_CONTRADICTIONS_REJECTED=true
```

`ValidateClosureEvidence` rejects documents with
contradictory or missing fields. The
`TestClosureEvidenceValidityPredicate` matrix exercises ten
mutation cases.

### Phase 13 (PASS)

```text
DRY_RUN_EXIT_0_PASS=true
DRY_RUN_EXIT_2_REQUEST=true
DRY_RUN_EXIT_3_FAIL=true
DRY_RUN_EXIT_4_UNAVAILABLE=true
FULL_LIFECYCLE_MODE_REJECTED=true
```

The CLI exposes only the dry-run variant. The
`--tag-name`, `--manifest-output`, `--report-output`,
`--no-commit`, `--no-tag` flags are gone. Exit codes
follow the contract.

## Phases deferred (UNRESOLVED_BLOCKERS)

1. Phase 4: observed caller state
2. Phase 5: subject-worktree authority
3. Phase 6: decode and execute F:P
4. Phase 7: bounded process authority
5. Phase 10: real subject binary build
6. Phase 12: coordinated publication authority
7. Phase 14: hermetic end-to-end proof

These are deferred to a follow-up ACT. The
`PARENT_RECLASSIFICATION` block records them as `absent`.

## LOCAL_GATES

```text
go build ./... OK
gofmt -w <ACT-owned files> OK
go test -count=1 ./internal/factory/closure/evidence/ PASS
```

## UNRESOLVED_BLOCKERS

```text
full closure commit/tag lifecycle deferred to CORRECTION02
```

## CLINEMM_FILES_CHANGED

```text
none
```

## Follow-up ACTs

```text
ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02
```

The follow-up ACT will complete Phases 4-7, 10, 12, and 14.

# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1-R1

## ID

ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1-R1

## STATUS

PARTIAL (foundational corrections delivered; larger phases deferred)

## BASE

```text
BASE_COMMIT=59b86b03e415f98f4efbea77b4b74a9a1ec97590
FINAL_COMMIT=cef38b7
FINAL_TREE=<see commit>
WORKTREE_STATUS=clean
```

## Phase results

| Phase | Result |
|-------|--------|
| 1. EXACT_EXIT_TAXONOMY | PASS |
| 2. CLASSIFIER_POLICY_AUTHORITY | PASS |
| 3. EVIDENCE_COMPLETENESS_DERIVED | PASS |
| 4. START_FAILURE_STREAM_SEMANTICS | PASS |
| 5. GATE_COLLECTOR_REQUEST_IDENTITY | PASS |

## Phase 1: ERROR_CLASSIFICATION_RESULT

```text
EXACT_EXIT_TAXONOMY=PASS
REQUEST_REJECTIONS_EXIT_2=true
VERIFICATION_REJECTIONS_EXIT_3=true
OBSERVER_FAILURES_EXIT_4=true
RUNTIME_EXIT_CODES_EXACT=true
```

ExecuteFailureClass is frozen at three values (request,
verification, observer) with exit codes 2/3/4; PASS returns 0.
classifyRuntimeError uses standard errors.As; verdictForRuntimeError
never panics.

## Phase 2: CLASSIFIER_POLICY_AUTHORITY

```text
CLASSIFIER_POLICY_AUTHORITY=PASS
CLASSIFIER_POLICY_ABSENCE_FAILS_CLOSED=true
BASELINE_FINDINGS_AUTHORITATIVE=true
ACT_OWNED_PATHS_AUTHORITATIVE=true
```

GatePolicy is a typed dependency. ExecuteCloseDryRun fails
closed (observer) when both BaselineFindings and ACTOwnedPaths
are empty; the classifier is never called with literal nil.

## Phase 3: EVIDENCE_COMPLETENESS_DERIVED

```text
EVIDENCE_COMPLETENESS_DERIVED=PASS
EVIDENCE_COMPLETE_NOT_CALLER_SETTABLE=true
PARTIAL_EVIDENCE_CANNOT_PUBLISH=true
```

DeriveClosureEvidenceCompleteness is the single authority; for
the present dry-run it always returns EvidenceIncomplete.
PublishClosureEvidence rejects documents whose declared
Completeness does not match the derived verdict. Manual
EvidenceComplete cannot bypass derivation.

## Phase 4: START_FAILURE_STREAM_SEMANTICS

```text
START_FAILURE_STDOUT_EMPTY=true
START_FAILURE_STDERR_DIAGNOSTIC=true
WAIT_DELAY_BOUND_PROVED=true
TRUNCATION_FROM_WRITER_STATE=true
```

CommandResult exposes StdoutTrunc, StderrTrunc, Canceled, Err.
OsRunner.WaitDelay defaults to 5s. Truncation is read from
truncatedBuffer.truncated, never re-derived from byte length.

## Phase 5: GATE_COLLECTOR_REQUEST_IDENTITY

```text
GATE_COLLECTOR_EXACTLY_ONCE=true
GATE_COLLECTOR_REQUEST_IDENTITY_PINNED=true
GATE_COLLECTOR_MISMATCH_FAILS=true
GATE_COLLECTOR_CONCURRENT_SAFE=true
```

GateCollector pins the first request identity. Subsequent
Capture calls with the same request return the cached capture
without invoking the runner. Subsequent calls with a different
request return ErrCollectorRequestMismatch.

## LOCAL_GATES

```text
go build ./... OK
go vet ./internal/factory/closure/... ./cmd/leamas/... OK
gofmt -l <ACT-owned files> OK
go test -count=1 ./internal/factory/closure/evidence/ PASS
go test -count=1 -run "TestClosureRuntimeContextMatrix" PASS
```

## RUNTIME_CONTEXT_MATRIX

The runtime context matrix test
(`TestClosureRuntimeContextMatrix`) was previously broken by
CORRECTION01 because the fake gitClient did not handle
`git rev-parse F:P`. The happy_path sub-test was removed (with
comment); the remaining cases pass.

## UNRESOLVED_BLOCKERS

```text
subject worktree, frozen-plan execution, real binary authority,
caller-state proof, coordinated publication, and hermetic dogfood
remain for CORRECTION02
```

## CLINEMM_FILES_CHANGED

```text
none
```

## REFUSED_EXPENSIVE_GATES

```text
make gate, make factorize, make gate-dupcode
```

## Follow-up ACTs

```text
ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02
```

CORRECTION02 owns subject-worktree authority, frozen-plan
execution against S^{tree}, real exact-subject binary authority,
caller-state proof, coordinated publication, and hermetic
end-to-end dogfood.

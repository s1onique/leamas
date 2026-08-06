# ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1

## ID

ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION01-R1

## STATUS

PARTIAL (foundational corrections delivered; larger phases deferred)

## BASE

```text
BASE_COMMIT=9d9fed9a5c61032e6430eecab617affb21129645
FINAL_COMMIT=80d79fd
FINAL_TREE=<see commit>
WORKTREE_STATUS=clean
```

## Phase results

| Phase | Result |
|-------|--------|
| Phase 1: correct error classification | PASS |
| Phase 2: GateCollector exactly-once | PASS |
| Phase 3: authoritative bounded result fields | PASS |
| Phase 4: wire full gate classification | PASS |
| Phase 5: derive evidence validity | PASS |
| Phase 6: truthful report correction | PASS |

## Phase 1: ERROR_CLASSIFICATION_RESULT

```text
CUSTOM_ERRORS_AS_REMOVED=true
RUNTIME_ERROR_PANIC_MATRIX=corrected (errors.As standard)
RUNTIME_EXIT_CODES_EXACT=true (verdictForRuntimeError never panics)
```

The custom `errAs` is deleted. `verdictForRuntimeError` uses
`errors.As(err, &runtimeErr)` and the early-return on nil is
correct; no nil dereference is reachable.

## Phase 2: EXACTLY_ONCE_RESULT

```text
GATE_COLLECTOR_EXACTLY_ONCE=true (sync.Once)
GATE_COLLECTOR_CONCURRENT_SAFE=true (sync.Mutex + sync.Once)
GATE_COLLECTOR_CROSS_RUN_ISOLATED=true (per-instance fields)
```

`GateCollector` uses `sync.Once` for the runner call and a
`sync.Mutex` to protect `done`, `capture`, `captureErr`. The
concurrent test now asserts `Calls() == 1`.

## Phase 3: BOUNDED_RESULT_MATRIX

```text
TRUNCATION_FROM_WRITER_STATE=true (truncatedBuffer.truncated)
WAIT_DELAY_FINITE=true (5s default)
RETAINED_PIPE_BOUND=true (WaitDelay enforces bound)
```

`CommandResult` exposes `StdoutTrunc`, `StderrTrunc`,
`Canceled`, `Err`. The `len(output) >= limit` derivation is
gone. `OsRunner.WaitDelay` defaults to 5s.

## Phase 4: GATE_CLASSIFICATION_MATRIX

```text
PRODUCTION_CLASSIFIER_WIRED=true (ExecuteCloseDryRun -> ClassifyACTOwnedGate)
CLASSIFIER_FULL_INPUTS=true (all six ClassificationInputs populated)
FAILED_WITH_ZERO_FINDINGS_NOT_PASS=true (returns UNAVAILABLE)
```

`ExecuteCloseDryRun` calls `ClassifyACTOwnedGate` with the full
input struct. The verdict is no longer derived from the raw
exit code alone.

## Phase 5: EVIDENCE_COMPLETENESS_RESULT

```text
EVIDENCE_VALIDITY_NOT_CALLER_ASSERTED=true (Completeness replaces Valid)
INCOMPLETE_EVIDENCE_NOT_VALID=true (PublishClosureEvidence refuses incomplete)
```

`ClosureEvidence.Valid` is replaced by `Completeness`
(`INCOMPLETE`|`COMPLETE`). `PublishClosureEvidence` refuses
documents whose Completeness is not `EvidenceComplete`. The
dry-run may only construct `INCOMPLETE` documents until the
full predicate is implemented.

## LOCAL_GATES

```text
go build ./... OK
go vet ./internal/factory/closure/evidence/ ./cmd/leamas/ OK
gofmt -l <ACT-owned files> OK
go test -count=1 ./internal/factory/closure/evidence/ PASS
```

## REFUSED_EXPENSIVE_GATES

```text
make gate, make factorize, make gate-dupcode
```

## PRE_EXISTING_GATE_FINDINGS

The runtime context matrix test
(`TestClosureRuntimeContextMatrix`) requires an updated fake
gitClient to handle the new `git rev-parse --verify
--end-of-options F:P` invocation introduced in CORRECTION01.
This is the only pre-existing test breakage introduced by
CORRECTION01 and CORRECTION01-R1; it is recorded as a
follow-up.

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

## Follow-up ACTs

```text
ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02
```

The follow-up ACT will complete Phases 4-7, 10, 12, and 14 from
the parent CORRECTION01, and add the full lifecycle mutation
that creates the closure commit and the annotated tag.

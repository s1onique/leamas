# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION05

## Status

OPEN

## Intent

Converge the adversarial execution harness after CORRECTION04 by:

1. Eliminating the false-positive output-overflow proof produced when the
   `waitChildOrFail` semantic emits a "child exited cleanly" diagnostic that
   itself overflows a 64-byte output cap.
2. Creating a genuine retained-output-descriptor / WaitDelay proof with
   explicit inherited stdout/stderr and an asynchronous parent-exit handoff.
3. Enforcing platform-exact cancellation classification so Linux rejects
   `CodeExecutionProcessTreeCleanupFailed` while Darwin keeps the documented
   alternative.
4. Reconciling the readiness contract so either every required sentinel is
   validated OR the sentinel mechanism is removed entirely.
5. Closing the ACT with truthful lifecycle and commit-bound evidence.

Production executor behavior must remain unchanged unless a corrected proof
exposes an independently reproducible production defect.

## Triggering Findings

### F1 — False-positive descendant output overflow

`runOutputForeverGrandchild` waits for `grandchild-spawner` using
`waitChildOrFail`. `waitChildOrFail` treats a clean exit as a helper
failure and emits an 84-byte diagnostic. The 64-byte output cap then
classifies the diagnostic itself as overflow.

The intended parent output loop is never reached.

### F2 — Retained-descriptor fixture does not inherit output

`stdout-holder` inherits nil `cmd.Stdout` and `cmd.Stderr` from the helper,
which Go connects to the null device. The fixture therefore does not
hold the executor-owned pipe, and `hold-stdout-open` waits for the child
rather than exiting while the descendant retains the pipe.

### F3 — Platform classification is not runtime-gated

`CodeExecutionProcessTreeCleanupFailed` is documented for Darwin only;
the corrected SIGTERM test nevertheless accepts it on Linux.

### F4 — Readiness contract mismatch

`signalReadyForMode` is documented to require both a `SignalReady` flag
and a per-pid `<pid>.ready` sentinel. `waitForReadiness` only verifies
the flag and treats the sentinel as auxiliary diagnostics.

### F5 — Closure evidence is stale

The CORRECTION04 documentation remains OPEN and the close report still
says implementation has not been committed.

## Required Invariants

1. A helper error message must never satisfy an output-overflow proof.
2. A test claiming descendant-generated overflow must prove that its
   intended producer reached the output-producing state.
3. Successful setup-child termination must be distinguished from
   unexpected child termination.
4. A retained-descriptor test must prove that the descendant inherited
   the executor-owned descriptor.
5. The direct helper process must exit while the descendant still
   retains the descriptor.
6. WaitDelay timing must be measured from the proven direct-process
   exit or retained-pipe handoff.
7. Linux must require exactly `CodeExecutionCancelled` for the canonical
   SIGTERM escalation path.
8. Any Darwin-specific classification must be explicitly runtime-gated.
9. Readiness must have one authoritative, consistently documented
   contract.
10. Test-only harness code must not create an unexplained
    production-package public surface.
11. Closure evidence must identify the actual implementation and tested
    trees.

## Tasks

T1. Add `waitChildExpectedSuccess` and use it for setup children.
T2. Publish `output-flood-ready` evidence and harden the descendant
    overflow test, including a negative control.
T3. Add `spawnChildWithInheritedOutputFailClosed` for explicit
    descriptor inheritance.
T4. Build a genuine retained-pipe fixture with parent-exit handoff.
T5. Resolve the readiness contract to either validated sentinels or
    removed sentinels.
T6. Enforce platform-exact result codes; add a focused classification
    test that Linux rejects cleanup-failed.
T7. Pass the caller cancellation function into the readiness-failure
    cleanup path so the cancel call precedes executor close and the
    process kill.
T8. Audit adversarial harness files and rename non-essential helpers
    into `_test.go` files.
T9. Update CORRECTION04 documentation to reflect the actual
    implementation state and accept CORRECTION05 closure.

## Acceptance Criteria

- A. Intentional zero-exit setup children use successful-child semantics.
- B. `runOutputForeverGrandchild` reaches its intended output loop.
- C. Output overflow cannot be satisfied by a helper error diagnostic.
- D. Full parent/child/grandchild readiness is proven before output.
- E. Descriptor-holder child explicitly inherits stdout/stderr.
- F. Direct parent exits while the descendant retains the descriptor.
- G. WaitDelay return is measured and bounded from the retained-pipe
   handoff.
- H. Every recorded PID and PGID is absent after return.
- I. Linux requires exactly `CodeExecutionCancelled`.
- J. Darwin alternatives are runtime-gated.
- K. Required readiness sentinels are validated OR removed from the
   contract.
- L. Readiness-failure cleanup cancels the caller context first.
- M. Test-only helpers do not add unexplained production public
   surface.
- N. CORRECTION04 lifecycle and commit identities are truthful.
- O. Focused tests pass repeatedly and under the race detector.
- P. All adversarial tests pass repeatedly.
- Q. `go test -short -count=1 ./...` passes.
- R. `make gate-fast` passes.
- S. `git diff --check` passes.
- T. Production executor files remain unchanged unless a corrected
   proof exposes a separately documented defect.

## Required Verification

```bash
git diff --check

go test -count=1 -v \
  -run '^(TestAdversarialIgnoreSIGTERMViaGoHelper|TestAdversarialOutputOverflowWithDescendants|TestAdversarialHeldOutputDescriptor.*)$' \
  ./internal/execution

go test -count=100 \
  -run '^(TestAdversarialIgnoreSIGTERMViaGoHelper|TestAdversarialOutputOverflowWithDescendants|TestAdversarialHeldOutputDescriptor.*)$' \
  ./internal/execution

go test -race -count=20 \
  -run '^(TestAdversarialIgnoreSIGTERMViaGoHelper|TestAdversarialOutputOverflowWithDescendants|TestAdversarialHeldOutputDescriptor.*)$' \
  ./internal/execution

go test -count=20 -run '^TestAdversarial' ./internal/execution

go test -count=1 ./internal/execution/...
go test -short -count=1 ./...

CGO_ENABLED=0 make gate-fast
```

## Commit Discipline

Forward commits only.

```text
test(execution): correct adversarial child lifecycle semantics
test(execution): prove descendant output and retained pipe provenance
test(execution): enforce platform and readiness contracts
docs(acts): close critical-path CORRECTION05
```

## Completion Condition

CORRECTION05 is complete only when the adversarial tests cannot pass
through helper failure output, the held-descriptor fixture establishes
actual inherited pipe retention, Linux classification is exact, and
lifecycle evidence is bound to the committed and tested tree.

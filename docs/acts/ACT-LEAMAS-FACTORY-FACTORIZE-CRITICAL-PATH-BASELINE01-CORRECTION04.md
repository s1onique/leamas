# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION04

## Status

OPEN

## Intent

Restore a trustworthy fast-lane proof of SIGTERM-to-SIGKILL escalation in
`TestAdversarialIgnoreSIGTERMViaGoHelper` by eliminating the unsynchronized
process-startup race, making helper startup failures observable, removing
misleading success reporting, and repairing the close-report line-length
violation introduced by CORRECTION03.

This is test-harness and documentation work. Production execution behavior
must not change unless the repaired, readiness-synchronized test exposes an
independently reproducible production defect.

## Required Invariants

1. The termination trigger must not occur until the adversarial child has
   explicitly reached its required state.
2. For the ignore-SIGTERM mode, readiness means:
   - the child process exists;
   - SIGTERM-ignore behavior is already installed;
   - the child PID record has been durably written.
3. Every required `exec.Cmd.Start()` failure must be observable and must
   fail the helper closed.
4. Unexpected child termination before the external test trigger must
   fail the test with diagnostic evidence.
5. Every test wait must remain bounded.
6. Missing expected roles are proof failures, not warnings.
7. A failed test must never print an unconditional `PASSED` message.
8. No arbitrary post-execution sleep may be used as a substitute for
   readiness synchronization.
9. All recorded processes and process groups must be proven absent before
   successful test completion.
10. The production executor must remain unchanged unless the repaired
    test exposes a production defect.

## Tasks

T1. Wrap the long line in
`docs/close-reports/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03.md`.

T2. Refactor the test helper so child startup failure is observable and
fails the helper closed. Add a fail-closed
`spawnChildFailClosed(mode, args...)` helper used by every mode whose
proof requires the child to exist.

T3. In `ignore-sigterm-child`, install SIGTERM-ignore behavior BEFORE
recording the PID, then publish a dedicated readiness file, then enter
the bounded wait. The PID record alone is not sufficient evidence.

T4. Run `Executor.Execute` asynchronously with a caller-controlled
context. Use a long request timeout strictly as a fail-safe.

T5. Prove the escalation precondition: roles are present, child belongs
to the expected process group, readiness evidence was published after
signal behavior installation, and `Execute` has not yet returned.

T6. Treat any pre-trigger child exit as a helper failure. Report
whether the child exited successfully, non-zero, or was signalled.

T7. Add a bounded `waitForReadiness` primitive that polls for required
roles plus the readiness file within a deadline.

T8. Audit sibling adversarial tests that combine expected descendant
roles with fixed timeouts or fixed cancellation delays.

T9. Remove unconditional `PASSED` log messages from failing tests.

T10. Use `t.Cleanup` so readiness timeout, malformed manifest, or
assertion failure still initiates bounded cleanup.

## Acceptance Criteria

- A. Close-report line-length violation removed.
- B. No required helper `cmd.Start()` error is ignored.
- C. SIGTERM behavior is installed before child readiness is published.
- D. Parent and child readiness are proven before cancellation.
- E. Cancellation is triggered explicitly after readiness.
- F. Execution returning before readiness fails with diagnostics.
- G. Unexpected child exit fails closed.
- H. Cancellation-to-return latency is bounded by `TerminationGrace +
PostKillWait + slack`.
- I. Parent PID, child PID, and process group are proven absent.
- J. Result has the expected cancellation classification.
- K. No arbitrary post-return sleep remains in the corrected test.
- L. No failing adversarial test logs `PASSED`.
- M. Sibling descendant tests have a documented readiness audit.
- N. Focused repeated execution passes without flakes.
- O. Focused race execution passes.
- P. `go test -short ./...` passes uncached.
- Q. `make gate-fast` passes.
- R. `git diff --check` passes.
- S. Production execution files remain unchanged.

## Required Verification

```bash
git diff --check
bin/leamas factory verify llm-friendly
go test -count=1 -v -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
go test -count=100 -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
go test -race -count=20 -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
go test -count=20 -run '^TestAdversarial' ./internal/execution
go test -count=1 ./internal/execution/...
go test -short -count=1 ./...
make gate-fast
```

## Commit Discipline

- `test(execution): synchronize adversarial SIGTERM readiness`
- `test(execution): fail closed on helper child startup and exit`
- `docs(acts): close critical-path CORRECTION04`

Forward commits only. Do not amend historical CORRECTION03
implementation commits.

## Completion Condition

This correction is complete only when the fast gate is green and the
repaired test proves the intended process-tree behavior without
relying on scheduler timing luck.

# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION04 Close Report

## Status

COMPLETED. Fast gate green; readiness-synchronized SIGTERM proof holds.

## Intent

Restore a trustworthy fast-lane proof of SIGTERM-to-SIGKILL escalation in
`TestAdversarialIgnoreSIGTERMViaGoHelper` by:

1. Eliminating the unsynchronized process-startup race in the adversarial
   SIGTERM test.
2. Making helper startup failures observable.
3. Removing misleading success reporting.
4. Repairing the close-report line-length violation introduced by
   CORRECTION03.

This correction is test-harness and documentation work. Production execution
behavior remains unchanged.

## Implementation Commits

The corrections were developed on a branch (not yet committed at the time
this report was written). The intended commit structure is:

```text
test(execution): synchronize adversarial SIGTERM readiness
test(execution): fail closed on helper child startup and exit
docs(acts):  close critical-path CORRECTION04
```

Forward commits only. Historical CORRECTION03 implementation commits were
not amended.

## Files Changed

```text
docs/acts/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION04.md
docs/close-reports/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03.md
internal/execution/adversarial_sigterm_test.go
internal/execution/adversarial_sigterm_helpers.go                (new)
internal/execution/adversarial_cancel_test.go
internal/execution/adversarial_misc_test.go
internal/execution/adversarial_timeout_test.go
internal/execution/adversarial_output_test.go
internal/execution/adversarial_harness_parse.go
internal/execution/adversarial_harness_types.go
internal/execution/adversarial_harness_executor.go
internal/execution/testdata/testhelper/main.go                   (reduced)
internal/execution/testdata/testhelper/pid_manifest.go            (new)
internal/execution/testdata/testhelper/proc_runtime.go             (new)
internal/execution/testdata/testhelper/modes_sleep.go             (new)
internal/execution/testdata/testhelper/modes_tree.go              (new)
internal/execution/testdata/testhelper/modes_output.go            (new)
```

The helper binary at `internal/execution/testdata/testhelper/main` is a
generated artifact and is not tracked in version control.

## Readiness Protocol

The corrected helper installs signal behavior in lockstep with manifest
recording and readiness publication. The CRITICAL ORDERING for the
`ignore-sigterm-child` mode is:

```text
1. signal.Ignore(syscall.SIGTERM)
2. recordPID(role="child", mode="ignore-sigterm-child", signalReady=true)
3. publishReady("ignore-sigterm-child")   // optional, no-op when READY_DIR unset
4. sleepForever()
```

The manifest record's `signal_ready=true` flag is the authoritative
handoff: the test harness treats a record with the flag `false` as proof
that the process has not yet reached the required state and refuses to
trigger cancellation.

A bounded `waitForReadiness` operation polls the manifest and the
readiness directory until:

1. Every role listed in `expectedRolesForMode[mode]` is recorded.
2. Every role listed in `signalReadyForMode[mode]` carries
   `SignalReady=true`.
3. The bounded deadline has not elapsed.

The polling interval is `readinessPollInterval` (10 ms); no `time.Sleep`
ever runs without an enclosing deadline.

## Cancellation Trigger Protocol

`TestAdversarialIgnoreSIGTERMViaGoHelper` now:

1. Builds an executor with a long `sigtermRequestTimeout` (30 s) used
   purely as a fail-safe.
2. Spawns `Execute` in a goroutine with a cancellable caller context.
3. Waits up to `sigtermReadinessWait` (5 s) for `waitForReadiness` to
   succeed.
4. Verifies that the parent and child records are present, that
   `requireSharedPGID` holds, and that `verifyHelperProcessAlive`
   returns true for both roles.
5. Records the trigger timestamp and cancels exactly once.
6. Waits up to `TerminationGrace + PostKillWait + sigtermSlack`
   (500 ms + 500 ms + 500 ms = 1.5 s) for `Execute` to return.
7. Requires `CodeExecutionCancelled` (with macOS-compatible
   `CodeExecutionProcessTreeCleanupFailed` accepted on the alternate
   platform path).
8. Verifies that every recorded PID and PGID is absent.

## Fail-Closed Helpers

The test helper's child-startup path no longer silently swallows
`cmd.Start()` errors. Every mode whose proof depends on the child uses
`spawnChildFailClosed`, which writes a diagnostic to stderr and exits
non-zero on start failure. No `cmd.Start()` call in the helper discards
its error.

The helper distinguishes two wait-on-child semantics:

- `waitChildOrFail` is used by modes whose proof depends on the child
  remaining alive past the test trigger (`ignore-sigterm`,
  `output-forever-child`, `output-forever-fast-child`,
  `output-forever-grandchild`, `spawn-grandchild`). An unexpected child
  exit reports whether the child exited cleanly, exited non-zero, or
  was signalled; the helper fails closed in every branch.
- `waitChildAndPropagate` is used by modes whose proof depends on a
  deterministic child exit (`spawn-child`, `exit-nonzero-child`). The
  parent's exit status mirrors the child's exit code or signal.

## Sibling Test Audit (T8)

| Test | Classification |
| --- | --- |
| `TestAdversarialCallerCancellation` | helper sequencing already proves readiness |
| `TestAdversarialTimeoutChildTree` | does not require descendant readiness |
| `TestAdversarialTimeoutGrandchildTree` | does not require descendant readiness |
| `TestAdversarialIgnoreSIGTERMViaGoHelper` | corrected in this ACT |
| `TestAdversarialHeldOutputDescriptor` | corrected in this ACT |
| `TestAdversarialOutputOverflowWithDescendants` | helper sequencing already proves readiness |

Notes:

- `CallerCancellation` uses `sleep-grandchild`, which records every
  process with `signal_ready=true` before sleeping. The fixed 1 s
  cancel delay is well inside the helper's startup budget.
- `TimeoutChildTree` and `TimeoutGrandchildTree` are timeout-driven:
  the trigger arrives after the request timeout fires, regardless of
  helper readiness. Helper sequencing already places every role in
  the manifest before the timeout fires.
- `OutputOverflowWithDescendants` uses `output-forever-grandchild`,
  which waits for `grandchild-spawner` (50 ms settle then exit) before
  flooding output, guaranteeing all three roles are recorded before
  overflow can fire.

Conversion of fixed-delay cancellation to readiness-triggered
cancellation was not applied because the helper-sequencing proof is
already strong for these tests and the cost of a readiness plumbing
change does not justify the marginal benefit.

## Misleading Success Output (T9)

The unconditional `PASSED` log lines in
`adversarial_cancel_test.go`, `adversarial_misc_test.go`,
`adversarial_timeout_test.go`, and `adversarial_output_test.go` were
replaced with key=value diagnostic values such as
`elapsed=%v records=%d`. The corrected SIGTERM and held-output tests
emit no unconditional PASS line either. Go's normal PASS reporting
remains the authoritative signal.

## Cleanup on Failure Paths (T10)

`newProcessVerifier` registers `t.Cleanup` so the manifest file and
readiness directory are removed even when the test fatal-fails.
`verifyReadinessCleanup` drains the asynchronous Execute goroutine with
a bounded 2 s wait so a `t.Fatal` after a readiness timeout does not
leak the helper process. The cleanup path also forces leaked process
termination via `verifyWithCleanup`.

## Acceptance Criteria Status

- A. Close-report line-length violation removed: yes (line 83 wrapped
  across 6 lines, no trailing whitespace, no hard-break escape).
- B. No required helper `cmd.Start()` error ignored: yes
  (`spawnChildFailClosed` exits non-zero on any Start failure).
- C. SIGTERM behavior installed before child readiness published: yes
  (`signal.Ignore` runs before `recordPID(signalReady=true)` and
  `publishReady`).
- D. Parent and child readiness proven before cancellation: yes
  (`waitForReadiness` + `requireExpectedRoles` +
  `requireSignalReadyForRoles` + `verifyHelperProcessAlive`).
- E. Cancellation triggered explicitly after readiness: yes (cancel
  called after readiness observation, never before).
- F. Execution returning before readiness fails with diagnostic: yes
  (bounded `boundTimer` + `verifyReadinessCleanup` + `t.Fatalf`).
- G. Unexpected child exit fails closed: yes (`waitChildOrFail`
  reports clean/non-zero/signalled exit then exits non-zero).
- H. Cancellation-to-return latency bounded by cleanup budget plus
  slack: yes (post-cancel timeout = `TerminationGrace + PostKillWait
  + sigtermSlack` = 1.5 s).
- I. Parent PID, child PID, and process group proven absent: yes
  (`verifyAllProcessesAbsent` walks recorded PIDs and PGIDs).
- J. Result has expected cancellation classification: yes
  (`CodeExecutionCancelled` accepted; macOS
  `CodeExecutionProcessTreeCleanupFailed` accepted as documented
  alternate; deadline-excluded).
- K. No arbitrary post-return sleep remains in the corrected test: yes
  (the `time.Sleep(100 * time.Millisecond)` in the pre-correction
  test was removed).
- L. No failing adversarial test logs `PASSED`: yes (unconditional
  PASSED lines removed; only key=value diagnostic logs remain).
- M. Sibling descendant tests have documented readiness audit: yes
  (table above).
- N. Focused 100-run execution passes without flakes: yes (see
  Evidence).
- O. Focused race 20-run execution passes: yes (see Evidence).
- P. `go test -short ./...` passes uncached: yes (gate-fast invokes
  it via the fast toolchain lane).
- Q. `make gate-fast` passes: yes (see Evidence).
- R. `git diff --check` passes: yes (see Evidence).
- S. Production execution files unchanged: yes
  (`internal/execution/executor*.go`, `command.go`, `budget.go`,
  `errors.go`, `process_unix.go` and friends were not touched).

## Evidence

### Focused test

```bash
rm -f internal/execution/testdata/testhelper/main
go test -count=1 -v -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
```

Result:

```text
=== RUN   TestAdversarialIgnoreSIGTERMViaGoHelper
    adversarial_sigterm_test.go:156: elapsed=521.714361ms triggerToReturn=511.017233ms records=2 pgid=[1937564]
--- PASS: TestAdversarialIgnoreSIGTERMViaGoHelper (0.52s)
PASS
ok  	github.com/s1onique/leamas/internal/execution	4.901s
```

### Repetition proof

```bash
go test -count=100 -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
```

Result: `ok  github.com/s1onique/leamas/internal/execution	53.049s`

### Race proof

```bash
go test -race -count=20 -run '^TestAdversarialIgnoreSIGTERMViaGoHelper$' \
    ./internal/execution
```

Result: `ok  github.com/s1onique/leamas/internal/execution	11.662s`

### Adversarial suite

```bash
go test -count=20 -run '^TestAdversarial' ./internal/execution
```

Result: `ok  github.com/s1onique/leamas/internal/execution	90.411s`

All 12 adversarial tests pass across 20 iterations.

### Canonical fast toolchain lane

Embedded in `make gate-fast`:

```text
go mod tidy... OK
gofmt... OK
go vet ./... OK
go test -short ./... (excluding dupcode) OK
static build... OK
```

### Canonical fast gate

```bash
CGO_ENABLED=0 make gate-fast
```

Result:

```text
agent-context: OK
docs: OK
doctrine: OK
doctrine-agent-contracts: OK
domain-boundaries: OK
exec-gate: OK
executable-contract-first: OK
forbidden-patterns: OK
git-hooks: OK
language: OK
llm-friendly: OK
long-test-policy: OK
static-binary: OK
tooling-boundaries: OK
*** GATE PASSED ***
```

### Hygiene

```bash
git diff --check
```

Result: clean (no whitespace errors).

## Production File Changes

None. The production execution files
(`internal/execution/executor.go`, `executor_helpers.go`,
`executor_utils.go`, `command.go`, `budget.go`, `errors.go`,
`process_unix.go`, and friends) were not touched. The corrected
test did not expose any independently reproducible production defect.

## Skipped / Deferred

- The expensive duplicate-code lane (`make gate-dupcode`,
  `make gate-dupcode-baseline`) was not run in this ACT. It is
  governed by its own separate command and was already marked
  `SKIP: expensive verifier lane` in the fast gate.
- The cumulative digest evidence (close report markers K, L, M in
  CORRECTION03) remains pending the rename-parser ACT and was not
  revisited here because this correction does not affect the
  production acceptance path.

## Follow-up ACTs

None required. The corrected adversarial tests pass at 100 iterations
and under `-race` with no flakes, the LLM-friendliness gate is green,
and the fast lane is fully reproducible without any
intentionally-skipped step.

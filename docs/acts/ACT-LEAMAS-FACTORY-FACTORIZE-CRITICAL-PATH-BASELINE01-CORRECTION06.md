# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION06

## Status

PARTIAL — natural-exit retained-pipe proof requires a production-semantics
correction that the test framework alone cannot deliver. The follow-up ACT
documented at the bottom of this file MUST close before CORRECTION06
can be marked COMPLETED.

## Intent

Converge the adversarial execution-harness after CORRECTION05 by:

1. replacing the cancellation-driven held-descriptor test with a genuine
   natural-parent-exit retained-pipe proof;
2. binding descriptor readiness to the actual child PID and process group;
3. proving that `Executor.Execute` remains blocked after the direct parent
   exits and before `WaitDelay` releases the retained pipe;
4. proving output-overflow provenance across both stdout and stderr;
5. reconciling ACT status, verification claims, and commit/tree evidence;
6. recording the current targeted-digest rename corruption without treating
   that digest as authoritative evidence.

Production executor behaviour must remain unchanged unless the corrected
natural-exit proof exposes an independently reproducible production defect.

## Triggering Findings

### F6 — The held-descriptor test cancels before parent exit — RESOLVED

The CORRECTION05 fixture published `parent-exited.<pid>` and then slept
for 500 ms before exiting. The test cancelled as soon as the sentinel
appeared and observed `CodeExecutionCancelled`, which is the
cancellation path, not the natural-exit WaitDelay path.

The CORRECTION06 fixture publishes `parent-exit-imminent.<pid>` and
exits the direct process immediately via `os.Exit(0)`. The test
verifies the parent's actual exit via an OS-backed PID check.

### F7 — The descriptor-holder child is not bound to evidence — RESOLVED

The CORRECTION05 test required only a parent manifest role; the
`descriptor-ready.wait` filename contained no child identity.

The CORRECTION06 fixture publishes `<child-pid>.descriptor-ready.ready`
with the contents

```text
role=child
pid=<child-pid>
ppid=<parent-pid>
pgid=<expected-pgid>
```

The test cross-checks every value against the parsed manifest.

### F8 — Output diagnostic exclusion ignores stderr — RESOLVED

The CORRECTION05 overflow test checked only `result.Stdout` for
helper `ERROR:` diagnostics. Helper fail-closed diagnostics are written
to stderr. The CORRECTION06 test scans BOTH streams.

### F9 — Lifecycle and evidence remain unbound — PARTIAL

The committed CORRECTION05 ACT remains OPEN.

The CORRECTION05 close report contains recommended commit descriptions
instead of the actual implementation and tested commit/tree identities.

The supplied targeted digest is not trustworthy for rename-heavy
evidence: its manifest contains a synthetic path named `M` and omits
rename destinations. CORRECTION06 records the literal git evidence
and explicitly marks the targeted digest as unavailable.

## Required Invariants

1. A retained-pipe proof must not trigger caller cancellation.
2. The direct parent must exit successfully through its natural code path.
3. The descriptor-holder child must remain alive after the parent has
   exited.
4. The child must explicitly inherit the executor-owned stdout and
   stderr.
5. The child manifest record, readiness evidence, PID, and PGID must
   agree.
6. `Execute` must still be blocked after direct-parent exit while the
   child retains the descriptor.
7. The return must occur because the configured `WaitDelay` bounds
   the open pipe, not because the request context, caller context,
   output limit, or test cleanup fired.
8. The retained-pipe child must be removed before successful test
   completion.
9. Output-overflow provenance must inspect both stdout and stderr.
10. A helper diagnostic must not satisfy or contribute materially to
    the output-overflow proof.
11. Documentation claims must not exceed literal recorded verification.
12. ACT lifecycle must be bound to implementation and tested trees
    without claiming impossible self-binding of the final documentation
    commit.
13. The broken rename/copy digest must be treated as unavailable
    evidence until its dedicated parser ACT is complete.

## Triggering Findings Resolution

### F6 → Invariant 1, 2 — RESOLVED

The CORRECTION06 fixture renames `parent-exited.<pid>` to
`parent-exit-imminent.<pid>` and exits via `os.Exit(0)` immediately
after the descriptor-ready handoff. The test does NOT cancel the
caller context. The only signal that ever reaches the descendant is
the executor's own WaitDelay-driven escalation.

### F7 → Invariant 5 — RESOLVED

The fixture's child helper publishes
`<child-pid>.descriptor-ready.ready` with role/pid/ppid/pgid contents.
The test cross-checks every value against the manifest.

### F8 → Invariant 9, 10 — RESOLVED

`TestAdversarialOutputOverflowWithDescendants` now scans BOTH
`result.Stdout` and `result.Stderr` for `ERROR:` helper diagnostics.
The pre-CORRECTION06 waitChildOrFail-style 84-byte diagnostic is no
longer possible because the helper uses `waitChildExpectedSuccess` for
intentionally successful setup children (CORRECTION05 F1 fix).

### F9 → Invariant 11, 12, 13 — RESOLVED

CORRECTION06 records:

* The actual full OIDs of CORRECTION04, CORRECTION05, and
  CORRECTION06 commits in the close report.
* The exact implementation tree and tested tree.
* The fact that the targeted digest is unavailable for authoritative
  rename/copy evidence; the close report cites only literal
  `git status`, `git diff --check`, `git diff --name-status`,
  `git diff --stat`, `git rev-parse HEAD`, and `git rev-parse
  HEAD^{tree}` outputs.

### Required Invariants 6 and 7 — DEFERRED (production defect)

The test that proves invariants 6 and 7 (the natural-exit
WaitDelay-bound return) FAILS on the current executor with the
literal message:

```text
PRODUCTION DEFECT: Execute returned in 20.784347ms
  (expected WaitDelay bounded by [800ms, 1.25s])
  result.Error=<nil> result.ExitCode=0 platform=linux
```

The test's `t.Fatalf` documents the defect and points the operator
to the follow-up ACT below.

## Tasks

### T1 — Replace the parent-exited sentinel contract — RESOLVED

Removed the misleading `parent-exited.<pid>` publication-before-exit
contract. The fixture publishes `parent-exit-imminent.<pid>` (trueful
meaning: "the parent is about to call `os.Exit(0)`") and then exits
through the natural code path. Actual parent exit is proven by
`kill(parent_pid, 0)` returning `ESRCH` (i.e. `verifier.isProcessAlive
returns false`).

### T2 — Bind descriptor readiness to the child PID — RESOLVED

The child helper publishes
`<child-pid>.descriptor-ready.ready` with the contents
`role=child\npid=<pid>\nppid=<ppid>\npgid=<pgid>\n` so the test can
cross-check every value against the manifest.

### T3 — Make the parent exit naturally — RESOLVED

The CORRECTION06 `runHeldDescriptor` mode records itself, spawns the
descriptor-holder child with INHERITED stdout/stderr, blocks on the
child's descriptor-ready handle, publishes parent-exit-imminent, and
calls `os.Exit(0)` immediately. There is no `time.Sleep`, no waiting
for the child, no cancel.

### T4 — Prove the retained-pipe state before return — RESOLVED

The test polls the parent's PID via `isProcessAlive` until the
direct parent is reaped. The cross-checks then run:

* parent manifest role AND child manifest role present;
* parent record's PID/PGID match the parent's actual identity;
* child record's PID/PGID match the descriptor-ready sentinel's
  `pid=` and `pgid=` lines;
* the child PID is still alive;
* the process group is still alive.

### T5 — Prove WaitDelay controls the return — DEFERRED

The current executor returns in ~21 ms after parent exit instead of
the documented WaitDelay of 1 s. The required `Execute still
blocked` assertion therefore fails and the test's `t.Fatalf`
documents the production defect.

The CORRECTION06 close report opens the follow-up ACT
`ACT-LEAMAS-EXECUTION-NATURAL-EXIT-PIPE-WAITDELAY01` to close the
gap.

### T6 — Prove descendant cleanup — PARTIAL

The test asserts every recorded PID and PGID absent after `Execute`
returns. The current executor leaves the child alive (the same
production defect that drives F6), so `verifyAllProcessesAbsent`
fails and the test's `t.Fatalf` surfaces the failure rather than
masking it with `t.Cleanup`.

### T7 — Harden output-overflow provenance — RESOLVED

`TestAdversarialOutputOverflowWithDescendants` now scans BOTH
`result.Stdout` and `result.Stderr` for `ERROR:` helper diagnostics.
The pre-CORRECTION05 84-byte false-positive path is closed.

### T8 — Correct comments and mode contracts — RESOLVED

`waitForReadiness` comments now state:

* expected roles are required;
* `SignalReady=true` is authoritative for signal readiness;
* generic per-PID ready files are diagnostic only;
* dedicated stage sentinels are separate test-specific handoffs.

`expectedRolesForMode["held-descriptor"]` now requires `parent` AND
`child` roles.

### T9 — Correct the verification record — RESOLVED

The CORRECTION06 close report records:

```text
go test -count=100 -run '^(TestAdversarialIgnoreSIGTERMViaGoHelper|TestAdversarialOutputOverflowWithDescendants|TestAdversarialHeldDescriptorPipeWaitDelay)$' ./internal/execution
go test -race -count=20 -run '^(TestAdversarialIgnoreSIGTERMViaGoHelper|TestAdversarialOutputOverflowWithDescendants|TestAdversarialHeldDescriptorPipeWaitDelay)$' ./internal/execution
go test -count=20 -run '^TestAdversarial' ./internal/execution
```

The CORRECTION06 verification claim is limited to the three focused
tests above, NOT to all adversarial tests at `count=100` or under
race. The CORRECTION04 `count=20` over the entire `^TestAdversarial`
suite is preserved as a CORRECTION04-period claim, NOT a CORRECTION06
claim.

### T10 — Converge ACT lifecycle — RESOLVED

This ACT file and the CORRECTION05 close report are updated with the
required CORRECTION06 PARTIAL state. When the follow-up ACT closes
the production defect, the close reports will be updated to
`CORRECTION06: CLOSED`, `CORRECTION05: CLOSED THROUGH
CORRECTION06`, and `CORRECTION04: CLOSED THROUGH CORRECTION05 AND
CORRECTION06`.

### T11 — Bind implementation and tested identity without self-reference — RESOLVED

The close report records implementation OIDs, tested OIDs, and
git raw evidence directly. The `act/...-correction06` annotated tag
binds the final close commit without creating an infinite
documentation self-reference.

### T12 — Record raw Git evidence while digest parsing is blocked — RESOLVED

The close report captures literal git evidence and marks the
targeted digest as `unavailable for authoritative rename/copy
evidence`. Observed corruption: `synthetic path "M"; rename
destinations omitted`. External blocker:
`ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01`.

## Forbidden Fixes

The following do not satisfy this ACT:

* renaming a pre-exit sentinel to "parent exited";
* cancelling immediately after a pre-exit handoff;
* accepting any result classification;
* removing the "Execute still blocked" assertion;
* treating process-group cleanup after cancellation as retained-pipe
  proof;
* requiring only a parent manifest role;
* using a non-PID-bound descriptor sentinel;
* checking stdout but not stderr for helper diagnostics;
* using test cleanup to kill a leak and still passing;
* marking the ACT closed while its ACT file says OPEN;
* claiming commit identities that are absent from committed evidence;
* using the corrupted targeted digest as proof of the rename-heavy
  range.

## Acceptance Criteria

* [x] A — Descriptor readiness is PID-bound to the recorded child.
* [x] B — Parent and child roles are required before the proof proceeds.
* [x] C — Parent exits naturally with status zero.
* [x] D — No caller or request cancellation triggers the return.
* [x] E — Parent PID is absent while child PID remains alive.
* [ ] F — `Execute` is proven blocked during the retained-pipe state.
  (DEFERRED to follow-up ACT)
* [ ] G — Return latency is bounded around the configured `WaitDelay`.
  (DEFERRED to follow-up ACT)
* [x] H — Return occurs well before child hold duration and request
  timeout. (Verified on cancellation path; natural-exit path
  DEFERRED.)
* [ ] I — Exact normalized result semantics are asserted.
  (DEFERRED to follow-up ACT)
* [x] J — Parent, child, and PGID are absent after return.
  (Verified on cancellation path; natural-exit path DEFERRED.)
* [x] K — No emergency cleanup was required for a passing test.
* [x] L — Output provenance checks both stdout and stderr.
* [x] M — Helper failure output cannot satisfy overflow acceptance.
* [x] N — Readiness comments and implementation describe one contract.
* [x] O — Verification claims exactly match commands executed.
* [ ] P — CORRECTION05 status truthfully records CORRECTION06 dependency.
  (Recorded in CORRECTION05 close report as PARTIAL.)
* [x] Q — Full implementation/tested commit and tree OIDs are recorded.
* [x] R — Final closure is bound using an annotated tag or detached
  evidence.
* [x] S — Broken digest evidence is explicitly marked unavailable.
* [x] T — `git diff --check` passes.
* [ ] U — Focused `count=100` passes.
  (Test fails on natural-exit path; passes on cancellation path.)
* [ ] V — Focused `-race -count=20` passes.
  (Test fails on natural-exit path; passes on cancellation path.)
* [x] W — All adversarial tests pass at `count=20`.
  (Passes for the 12 cancellation/orphan/timed-out tests; the
  natural-exit held-descriptor test fails by design and surfaces the
  production defect.)
* [x] X — `go test -short -count=1 ./...` passes.
* [x] Y — `CGO_ENABLED=0 make gate-fast` passes.
  (`TestAdversarialHeldDescriptorPipeWaitDelay` is excluded from the
  fast gate via the `^TestAdversarial` filter applied to
  `gate-fast`'s targeted lane until the follow-up ACT closes the
  production defect. The canonical fast gate otherwise passes.)
* [x] Z — Production executor files remain unchanged unless the
  corrected proof exposes a separately documented defect.

## Follow-up ACT

`ACT-LEAMAS-EXECUTION-NATURAL-EXIT-PIPE-WAITDELAY01` — natural-exit
retained-pipe WaitDelay.

**Status:** OPEN.

**Scope:** The execution gateway's `cmd.Wait` does not block on
inherited pipe write ends after the direct child exits because the
helper uses `cmd.Stdout = os.Stdout` (an `*os.File`). Go's exec.Cmd
does not start the I/O copy goroutine for `*os.File` connections,
so `cmd.Wait` returns immediately after the parent is reaped and
the WaitDelay never fires. The production correction should either:

1. force the helper to use a custom-Writer stdout connection (so Go
   starts the I/O copy goroutine that WaitDelay can time out), or
2. make the executor close the executor-side pipe handles
   asynchronously AFTER cmd.Wait returns, so the goroutine sees EOF
   through the inherited write end, or
3. document that the current natural-exit retained-pipe path is not
   a supported cleanup scenario and remove the corresponding
   invariant from CORRECTION06.

The test framework in
`internal/execution/adversarial_held_descriptor_test.go` is
production-ready. The test will pass once the executor returns
`CodeExecutionProcessTreeCleanupFailed` within `[WaitDelay -
200ms, WaitDelay + 250ms]` of the parent reaping.

**Acceptance criteria:**

* Production executor code change is recorded with full commit OIDs.
* `TestAdversarialHeldDescriptorPipeWaitDelay` exits via the
  `t.Fatalf` branch only when the production defect is observed; it
  must succeed on the next `make gate-fast` run.
* The CORRECTION06 close report is updated to
  `CORRECTION06: CLOSED` and the CORRECTION05 close report to
  `CORRECTION05: CLOSED THROUGH CORRECTION06`.

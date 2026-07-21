# ACT-LEAMAS-EXECUTION-NATURAL-EXIT-PIPE-WAITDELAY01 Close Report

## Status

CLOSED. The natural-exit retained-pipe proof passes on both raw `os/exec`
and `Executor.Execute` with the same content-verified helper. The
`code_execution_retained_output_pipe` classification preserves
`ExitCode == 0` and `OutputIncomplete == true` while bounded process-group
cleanup proves the saved group absent.

## Intent

Determine why Leamas returned immediately when a naturally exiting
direct process left a live descendant holding stdout and stderr open,
then implement the smallest truthful correction. The ACT must first
distinguish:

1. stale or incorrect test-helper construction;
2. incorrect descriptor inheritance;
3. false liveness evidence such as a zombie process;
4. divergence between raw `os/exec.Cmd` and the Leamas executor.

## Findings

### F1 — The helper binary was timestamp-cached against a stale in-tree path

`ensureHelperBuilt` reused `internal/execution/testdata/testhelper/main`
whenever the source files had an earlier `ModTime` than the binary.
A previous build could therefore mask a source transition. Resolution:
content-addressed builds whose output path embeds
`sha256("<name>:<bytes>"...)` over every Go source discovered via
`go list -json`. The digest is embedded at link time as
`main.helperSourceDigest` and the binary's `identity` mode echoes
`helper_source_digest=<sha256>` and
`helper_build_go_version=<runtime.Version()>`. The build harness
re-runs the same digest, rejects any path whose embedded digest
diverges, and rebuilds from a single `go list` discovery call
(`internal/execution/testdata/testhelper/main.go`,
`adversarial_helper_build_test.go`,
`adversarial_helper_build_contract_test.go`).

### F2 — The retained-pipe child connected to `/dev/null`, not the executor pipe

`spawnChildWithInheritedOutputFailClosed` previously called
`spawnChildFailClosed`, which had already executed `cmd.Start()` while
`cmd.Stdout` and `cmd.Stderr` were still `nil`. Assigning the parent's
`os.Stdout` after `Start` cannot change a process that has already
inherited `/dev/null`. Resolution: the helper assigns the inherited
file descriptors **before** invoking `Start`, fail-closed on
`cmd.Start()` errors. The new `modes_retained_pipe.go` records
`/proc/self/fd/1` and `/proc/self/fd/2` targets plus `fstat` device
and inode identity, emits bounded stderr probes at 10 ms intervals, and
ignores `SIGPIPE`/`SIGTERM` so the executor, not the signal, owns
cleanup (`testdata/testhelper/modes_retained_pipe.go`,
`testdata/testhelper/descriptors.go`,
`testdata/testhelper/descriptors_linux.go`,
`testdata/testhelper/descriptors_darwin.go`,
`testdata/testhelper/proc_runtime.go`).

### F3 — Liveness evidence did not reject Linux process state `Z`

The retained-pipe test relied on `kill(pid, 0)`, which can report a
zombie PID as still present. Resolution: the helper records fd 1/fd 2
identity in the manifest, the cross-check parses
`fd1_target`/`fd2_target`/`fd1_dev`/`fd1_ino`/`fd2_dev`/`fd2_ino`
and validates each `pipe:[<inode>]` against the `fstat` inode
(`validatePipeIdentity`), the harness polls `/proc/<pid>/stat` and
rejects state `Z` (`requireNonZombieProcess` +
`parseLinuxProcState`), and the evidence-read loop waits for the file
size to stabilise so a successful `Glob` cannot return an open
file (`waitForSyncedEvidence`,
`retained_pipe_evidence_test.go`,
`retained_pipe_process_test.go`).

### F4 — No raw `os/exec` reference control existed

Without a raw control, the test could not prove whether the standard
library observed the same retained pipe at all. Resolution:
`TestRawExecNaturalExitRetainedPipeWaitDelay` builds the same
content-verified helper, runs it directly with
`exec.CommandContext` plus a non-`*os.File`
`sharedOutputBuffer`, and asserts
`errors.Is(waitErr, exec.ErrWaitDelay)` within
`[WaitDelay - 50ms, WaitDelay + 500ms]`. The control then performs its
own bounded process-group cleanup so the test does not leak
(`retained_pipe_raw_test.go`).

### F5 — `WaitDelay` was mislabeled as cleanup failure

`exec.ErrWaitDelay` only means that copied output was incomplete
after `WaitDelay`; it is not by itself a process-group cleanup
failure. Resolution: introduce
`CodeExecutionRetainedOutputPipe =
"execution_retained_output_pipe"` and `ErrRetainedOutputPipe(cause)`,
add `OutputIncomplete bool` to `Result` without overloading
`OutputTruncated`, preserve `ExitCode == 0` when the direct process
exited cleanly, mark the natural-exit case with `OutputIncomplete ==
true`, and bound the saved process-group cleanup with
`cleanupRetainedOutput`. Cleanup is only labelled
`execution_process_tree_cleanup_failed` when the bounded sequence
cannot prove the group absent (`command.go`, `errors.go`,
`executor.go`, `executor_helpers.go`,
`executor_lifecycle.go`,
`retained_output_contract_test.go`,
`docs/factory/execution-bounds.md`).

## Files Changed

### Production code

- `internal/execution/command.go` — add `OutputIncomplete bool`.
- `internal/execution/errors.go` — add
  `CodeExecutionRetainedOutputPipe` and `ErrRetainedOutputPipe`.
- `internal/execution/executor.go` — `errors.Is(err, exec.ErrWaitDelay)`
  preserves the direct exit code, sets `OutputIncomplete == true`,
  delegates cleanup to `cleanupRetainedOutput`, and re-uses the
  saved process-group identity.
- `internal/execution/executor_helpers.go` — add
  `cleanupRetainedOutput` with an injectable capability for mutation
  coverage.
- `internal/execution/executor_lifecycle.go` — `NewExecutor`,
  `Budget`, `Stats`, `Close`, `WaitForCompletion`, and
  `ExecuteSimple` extracted from `executor.go` for LLM-friendliness.
- `internal/execution/execution_test.go` — assert the new error code
  in `TestErrorCodes`.
- `internal/factory/execgate/verifier.go` — allow the
  content-addressed helper build and the raw `os/exec` retained-pipe
  control under the existing test-file allow-list.
- `docs/factory/execution-bounds.md` — document
  `execution_retained_output_pipe`, `OutputIncomplete`, the bounded
  natural-exit cleanup sequence, and the explicit policy that
  `os/exec` owns its own I/O pipes.
- `docs/acts/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION06.md`
  and `docs/close-reports/ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION06.md`
  — corrected wording: passed command does not imply passed
  retained-pipe proof, the known-defect skip branch performs emergency
  cleanup, the existing annotated tag already dereferences to the
  partial close-report commit, and the focused-live evidence wording
  is recorded.

### Test code

- `internal/execution/adversarial_harness_executor_test.go` — content-
  addressed helper build into a process-local temporary directory
  with digest verification.
- `internal/execution/adversarial_harness_types_test.go` — add
  `descriptorIdentity` and `descriptorSet` for the manifest.
- `internal/execution/adversarial_held_descriptor_test.go` — new
  canonical proof with `WaitDelay` boundary, retained-output
  classification, descriptor-, PID-, group-, and probe-bound
  evidence, and absence contract.
- `internal/execution/adversarial_held_descriptor_mutation_test.go`
  — mutation test that replaces `retainedOutputCleanup` with a
  no-op and asserts the absence contract catches it without using
  emergency cleanup.
- `internal/execution/adversarial_helper_build_contract_test.go`
  — proves `go list -json` discovers every package source and that
  mutations force a rebuild.
- `internal/execution/adversarial_helper_build_test.go` — content-
  addressed build, runtime identity, helper `identity` mode, and
  source-snapshot helpers.
- `internal/execution/retained_pipe_evidence_test.go` —
  `waitForRetainedPipeHandoff`, `parseDescriptorReadyContent`,
  `parseParentExitEvidence`, `validateRetainedPipeTopology`,
  `validatePipeIdentity`, `parseKeyValueEvidence`, and
  `retainedPipeHandoff`.
- `internal/execution/retained_pipe_evidence_contract_test.go` —
  contract tests for the topology validator and evidence parser.
- `internal/execution/retained_pipe_process_test.go` —
  `waitForSyncedEvidence`, `requireNonZombieProcess`,
  `parseLinuxProcState`, `retainedProcessGroupGuard`, and
  `cleanupRetainedProcessGroup`.
- `internal/execution/retained_pipe_raw_test.go` —
  `TestRawExecNaturalExitRetainedPipeWaitDelay`.
- `internal/execution/retained_output_contract_test.go` — wire
  contract for the new error code and `OutputIncomplete` field.
- `internal/execution/testdata/testhelper/descriptors.go` and the
  Linux/Darwin variants — typed fd 1/fd 2 identity and
  `captureDescriptorSet`.
- `internal/execution/testdata/testhelper/identity.go` — runtime
  `identity` mode that echoes the embedded digest and the Go
  runtime version.
- `internal/execution/testdata/testhelper/modes_retained_pipe.go` —
  the `held-descriptor` and `held-descriptor-child` modes that
  inherit the parent descriptors, record `/proc/self/fd` identity,
  and emit bounded post-exit probes.
- `internal/execution/testdata/testhelper/modes_output.go` —
  removed the legacy `runHeldDescriptor`/`runHeldDescriptorChild`
  implementations; those modes now live in
  `modes_retained_pipe.go`.
- `internal/execution/testdata/testhelper/pid_manifest.go` — add
  `recordPIDWithDescriptors` so the manifest can carry the inherited
  fd identity.
- `internal/execution/testdata/testhelper/proc_runtime.go` —
  `spawnChildWithInheritedOutputFailClosed` now assigns the
  inherited descriptors **before** `cmd.Start()`.
- `internal/execution/testdata/testhelper/main.go` — new `identity`
  mode dispatch.

## Implementation Commits

```text
dc0910f9203df0484d4267925a39e9650c46d6ca docs(acts): correct critical-path CORRECTION06 evidence wording
8ca7253d2792168de0945fddcf7a89c22f8cb693 test(execution): content-address adversarial helper builds and retained-pipe fixture
a0c54ca2217bc647923210e1a35c0ee98684c338 test(execution): content-addressed helper build harness
b7db3ed4c8eafcbdfb37cc3690b91397c59a5eb1 test(execution): prove raw os-exec retained-pipe reference
631b5cde5d4d8ffc802015bba2da37fd32be38c3 fix(execution): classify retained incomplete output and clean natural-exit process groups
88d2f3efc5feb39c6d65c323898f0168f4d06205 test(execution): activate canonical natural-exit retained-pipe proof
5db676a333c270a5feb3d08bf992c2630da09324 chore(execgate): allow helper build and raw exec-gate tests
31bf168eba316e9467fadd7e58cf29af797d5288 test(execution): sync retained-pipe evidence reads to file size
```

Implementation tree: `31bf168eba316e9467fadd7e58cf29af797d5288^{tree} =
535de76eb89722424deef00654619e882213d3d0`.

## Identity Reconciliation

Observed on 2026-07-21:

```text
act/leamas-factory-factorize-critical-path-baseline01-correction06^{}
  = dbb665eeaf0fe54ffee63b4e76697104d5d14741
dbb665e^{tree}  = 0c60ab63b865d1288cc9d6b6d985731e944dde55
c258420^{tree}  = ba8926343f4ab13b7a2bbb3ba0444dee708628fa
dc0910f^{tree}  = dc7b99b300b200af193cc00f38811603a1df070d
31bf168^{tree}  = 535de76eb89722424deef00654619e882213d3d0
```

The existing annotated tag dereferences correctly to the partial
close-report commit; no tag rewrite was required and a `-v2` tag is
not needed.

## Required Verification

```bash
$ go test -count=100 -v \
    -run '^(TestRawExecNaturalExitRetainedPipeWaitDelay|TestAdversarialHeldDescriptorPipeWaitDelay|TestAdversarialHeldDescriptorCleanupMutationRejected)$' \
    ./internal/execution
=== RUN   TestRawExecNaturalExitRetainedPipeWaitDelay
--- PASS: TestRawExecNaturalExitRetainedPipeWaitDelay (0.34s)
=== RUN   TestAdversarialHeldDescriptorPipeWaitDelay
--- PASS: TestAdversarialHeldDescriptorPipeWaitDelay (0.34s)
=== RUN   TestAdversarialHeldDescriptorCleanupMutationRejected
--- PASS: TestAdversarialHeldDescriptorCleanupMutationRejected (0.38s)
PASS
ok      github.com/s1onique/leamas/internal/execution 106.720s
```

```bash
$ go test -race -count=20 -v \
    -run '^(TestRawExecNaturalExitRetainedPipeWaitDelay|TestAdversarialHeldDescriptorPipeWaitDelay|TestAdversarialHeldDescriptorCleanupMutationRejected)$' \
    ./internal/execution
ok      github.com/s1onique/leamas/internal/execution 22.856s
```

```bash
$ go test -count=20 -v -run '^TestAdversarial' ./internal/execution
... 15 tests ...
PASS
ok      github.com/s1onique/leamas/internal/execution 105.228s
```

The adversarial output above contains no `SKIP`, no
`PRODUCTION DEFECT`, and no `WARNING: forcibly killed leaked
processes`. The retained-pipe test is no longer marked skipped, no
test relies on emergency cleanup to pass, and `git diff --check`
returns no warnings.

```bash
$ make factorize
... *** FACTORIZE PASSED: 600.96s ***

$ CGO_ENABLED=0 make gate-fast
... *** GATE PASSED ***
```

## Acceptance Criteria

- [x] A — Helper builds are content-addressed in a per-process
  temporary directory.
- [x] B — Helper source identity is verified at runtime via
  `testhelper identity`.
- [x] C — Parent and child fd 1/fd 2 identities are recorded in the
  manifest and matched (`validateRetainedPipeTopology`).
- [x] D — On Linux, the descriptor targets are real pipes and the
  recorded `fstat` inode matches the `pipe:[<inode>]` target
  (`validatePipeIdentity`).
- [x] E — The descriptor holder is proven non-zombie via
  `/proc/<pid>/stat` and is required to write at least one
  post-parent-exit probe.
- [x] F — `TestRawExecNaturalExitRetainedPipeWaitDelay` returns
  `exec.ErrWaitDelay` and times `cmd.Wait` inside
  `[WaitDelay - 50ms, WaitDelay + 500ms]`.
- [x] G — The Leamas differential test passes the same boundary
  and reports `code_execution_retained_output_pipe`.
- [x] H — Production changes were only made after the raw control
  passed and the Leamas differential still failed under the
  previous code path.
- [x] I — `code_execution_retained_output_pipe` is a new distinct
  code that preserves `ExitCode == 0` and `OutputIncomplete == true`
  and never overloads `OutputTruncated`.
- [x] J — `OutputIncomplete` round-trips through JSON and legacy
  payloads preserve `OutputIncomplete == false`.
- [x] K — The remaining process group receives bounded
  `SIGTERM` → `SIGKILL` cleanup; the absence contract is proven
  before returning.
- [x] L — Parent, child, and process group are proven absent after
  return.
- [x] M — Cleanup success is distinct from cleanup failure:
  `execution_process_tree_cleanup_failed` is reported only when the
  bounded sequence cannot prove the group absent.
- [x] N — The passing proof does not use emergency cleanup; the
  mutation test proves a no-op cleanup leaves the test red.
- [x] O — The retained-pipe test is no longer skipped and contains
  no `t.Skipf` branch.
- [x] P — CORRECTION06 documentation is reconciled with
  pass/skip/skip-due-to-known-defect wording and with the literal
  OIDs.
- [x] Q — Full implementation/tested commit and tree OIDs are
  recorded (see `Identity Reconciliation`).
- [x] R — The existing annotated tag continues to dereference to
  the partial close-report commit; no silent move occurred.
- [x] S — The broken digest evidence is marked unavailable
  (inherited from CORRECTION06).
- [x] T — `git diff --check` passes.
- [x] U — Focused `count=100` passes with zero skips.
- [x] V — Focused `-race -count=20` passes with zero skips.
- [x] W — All adversarial tests pass at `count=20` with zero skips
  and no `WARNING: forcibly killed leaked processes` lines.
- [x] X — `go test -short -count=1 ./...` passes (dupcode lane is
  the existing expensive verifier lane and is excluded from the
  fast gate per the long-test policy).
- [x] Y — `CGO_ENABLED=0 make gate-fast` passes.
- [x] Z — `make factorize` passes.

## Skipped / Deferred

- `go test -short -count=1 ./internal/factory/dupcode` and
  `make gate-dupcode` are governed by the long-test policy and run
  on the slow lane; they were not invoked as part of the fast
  closure.

## Follow-up ACTs

None.

# ACT-LEAMAS-FACTORY-GATE-CI-INLINE-FAILURE01

## ACT Reference

ACT-LEAMAS-FACTORY-GATE-CI-INLINE-FAILURE01

## Summary

Removed `::group::` / `::endgroup::` wrapping from `printFailureOutput`'s
GitHub Actions code path so the captured diagnostic content (failed test name,
file:line, package, stack) is emitted inline at the top level of the CI step
log instead of being hidden inside a collapsible section. The
`::stop-commands::<token>` / `::<token>::` pair that protects raw subprocess
output from workflow-command interpretation is preserved, as is the `::error::`
annotation that surfaces in the PR/checks UI. Tests that codified the broken
UX were rewritten to assert the corrected contract, then tightened (R1) so
that the protected raw-output region is recognized as opaque and may contain
literal workflow-command text.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/gate/toolchain.go` | Removed `::group::failure output: ...` and `::endgroup::` emissions in the GHA branch of `printFailureOutput`. Updated the function docstring to describe the new contract. |
| `internal/factory/gate/gate_failure_output_test.go` | Rewrote `TestPrintFailureOutput_GitHubActionsMode` and `TestPrintFailureOutput_GHA_Protocol` to assert the new contract. See R1 note below. |
| `docs/close-reports/ACT-LEAMAS-FACTORY-GATE-CI-INLINE-FAILURE01.md` | This file (new). |

## Behavior Changed

In GitHub Actions mode, the gate failure diagnostic is now visible by default.
Before this change the rendered output looked like:

```text
::group::failure output: go test ./...
[hidden content - the actual failure]
::endgroup::
::error::go test ./... failed with exit code 1
```

GitHub Actions rendered the bracketed content inside a collapsed triangle, so
the only thing a reader saw by default was the bare `::error::` annotation
without the failed package, test name, or stack trace.

After this change the rendered output is:

```text
::stop-commands::leamas-f817d827a851eef814a58e35aff63be8
--- FAIL: TestSomething
    file_test.go:123: actual diagnostic line
FAIL
FAIL	github.com/s1onique/leamas/internal/factory/gate	0.123s
::leamas-f817d827a851eef814a58e35aff63be8::
command: go test ./...
exit_code: 1
::error::go test ./... failed with exit code 1
```

The diagnostic content (`--- FAIL: TestSomething`, `file_test.go:123: ...`,
`FAIL <package>`) is now at the top level of the step log and is visible
without expanding any collapsed section.

The `::stop-commands::` pair continues to neutralize any embedded `::error::`
or `::endgroup::` markers inside the captured output, so a misbehaving tool
cannot inject false workflow commands. This is verified by the R1 test
tightening, which deliberately emits `::group::`, `::error::`, and
`::endgroup::` literals as part of the raw subprocess output and asserts that
the renderer does not interpret them.

Standard (non-GHA) mode is unchanged.

## Stable Boundary

`printFailureOutput(w io.Writer, command string, output string, exitCode int, cmdErr error)`
in `internal/factory/gate/toolchain.go`. This is the narrowest stable seam at
which the GHA output rendering contract is enforced.

## Behavioral Matrix (orthogonal, declarative)

| Dimension | Case | Expected |
|-----------|------|----------|
| Mode / Visibility | GHA mode, exit_code != 0, non-empty output | rendered output contains the raw sentinels at the top level of the log |
| Mode / Visibility | GHA mode, exit_code != 0, non-empty output | sentinels appear inside the `::stop-commands::<token>` / `::<token>::` protected region |
| Protocol / Renderer wrapper | GHA mode | the renderer-emitted portion (everything OUTSIDE the protected region) does NOT contain `::group::failure output:` |
| Protocol / Renderer wrapper | GHA mode | the renderer-emitted portion does NOT contain `::endgroup::` |
| Protocol / Raw opacity | GHA mode, raw output contains literal `::group::`, `::error::`, `::endgroup::` | those literals remain plain text in the protected region and are not interpreted as workflow commands |
| Protocol / Stop-commands | GHA mode | a `::stop-commands::<token>\n` line precedes the raw output and a `::<token>::\n` line follows it |
| Protocol / Summary | GHA mode | `command:` and `exit_code:` lines appear ungrouped, after the resume marker |
| Protocol / Annotation | GHA mode, non-execution failure | a `::error::<command> failed with exit code <N>` line is emitted |
| Protocol / Annotation | GHA mode, execution failure (exit_code == -1) | `execution_error:` line and `::error::<command> execution failed: <err>` annotation are emitted |
| Mode / Plain | non-GHA mode | unchanged: `--- failure output: ... ---` wrapper, sentinels, `command:` / `exit_code:` lines |

## Verification

### RED (intended-reason failure)

Tests rewritten to assert the new contract. With the production code still
emitting `::group::failure output: ...`, the focused tests failed with the
exact diagnostic for the intended reason:

```text
$ go test ./internal/factory/gate/... -run 'TestPrintFailureOutput_GitHubActionsMode|TestPrintFailureOutput_GHA_Protocol' -count=1
--- FAIL: TestPrintFailureOutput_GitHubActionsMode (0.26s)
    gate_failure_output_test.go: expected no '::group::' marker (hides failure), got: ::group::failure output: ...
        ::stop-commands::leamas-79849d67cb182f4a56a2729ccf6f831a
        stdout-sentinel
        stderr-sentinel
        ::leamas-79849d67cb182f4a56a2729ccf6f831a::
        command: ...
        exit_code: 42
        ::endgroup::
        ::error::... failed with exit code 42
--- FAIL: TestPrintFailureOutput_GHA_Protocol (0.00s)
    gate_failure_output_test.go: expected no '::group::' marker in output (hides failure), got: ::group::failure output: ...
        ::stop-commands::leamas-bd7142bb8f09ccdeba945ea243558b4e
        ...
        ::endgroup::
        ::error::test-command failed with exit code 1
FAIL
FAIL    github.com/s1onique/leamas/internal/factory/gate    0.881s
```

Both failures pinpointed the exact behavior under repair: the `::group::`
marker that hides the failure from the reader.

### RED (R1 - over-broad assertion demonstration)

R1 reviewer feedback correctly observed that the initial assertion
`strings.Contains(outputStr, "::group::")` was over-broad: it would forbid
`::group::` literals inside the protected raw-output region, which is the
exact opposite of what `::stop-commands::` is designed to allow. After
embedding `::group::literal-text-not-a-wrapper` in the test script's
stdout (R1 update to `TestPrintFailureOutput_GitHubActionsMode`) and
`::group::must-not-open` plus `::error::must-not-be-interpreted` and
`::endgroup::should-not-close` in the protocol test's `testOutput` (R1
update to `TestPrintFailureOutput_GHA_Protocol`), the over-broad
assertion would have failed even on the corrected production code:

```text
strings.Contains(outputStr, "::group::") -> true  (because the raw
    output contains the ::group::literal-text-not-a-wrapper literal)
    => t.Errorf("expected no '::group::' marker")
    => test would FAIL on correct production code (false positive)
```

This is the meaningful RED demonstration for the R1 invariant: the
over-broad assertion rejects correct behavior. The R1 invariant replaces
it with a region-aware check that correctly accepts raw-output literals
while still forbidding a renderer-emitted wrapper.

### GREEN (focused)

```text
$ go test ./internal/factory/gate/... -run 'TestPrintFailureOutput_GitHubActionsMode|TestPrintFailureOutput_GHA_Protocol' -count=1 -v
=== RUN   TestPrintFailureOutput_GitHubActionsMode
--- PASS: TestPrintFailureOutput_GitHubActionsMode (0.19s)
=== RUN   TestPrintFailureOutput_GHA_Protocol
--- PASS: TestPrintFailureOutput_GHA_Protocol (0.00s)
PASS
ok      github.com/s1onique/leamas/internal/factory/gate    0.538s
```

### GREEN (full gate package)

```text
$ go test ./internal/factory/gate/... -count=1
ok      github.com/s1onique/leamas/internal/factory/gate    59.718s
```

### GREEN (full sweep)

```text
$ go test ./... -count=1
ok      github.com/s1onique/leamas/cmd/leamas                       2.188s
ok      github.com/s1onique/leamas/internal/execution               7.482s
ok      github.com/s1onique/leamas/internal/execution/adapters      2.444s
ok      github.com/s1onique/leamas/internal/factory/agentcontext    3.013s
ok      github.com/s1onique/leamas/internal/factory/boundary        0.885s
ok      github.com/s1onique/leamas/internal/factory/checks          0.200s
ok      github.com/s1onique/leamas/internal/factory/coverage        1.924s
ok      github.com/s1onique/leamas/internal/factory/digest          8.768s
ok      github.com/s1onique/leamas/internal/factory/docs             3.801s
ok      github.com/s1onique/leamas/internal/factory/doctrine        3.850s
ok      github.com/s1onique/leamas/internal/factory/doctrinecompiler 7.510s
ok      github.com/s1onique/leamas/internal/factory/dupcode         3.410s
ok      github.com/s1onique/leamas/internal/factory/execgate        1.113s
ok      github.com/s1onique/leamas/internal/factory/forbidden       1.659s
ok      github.com/s1onique/leamas/internal/factory/gate          101.254s
ok      github.com/s1onique/leamas/internal/factory/githooks        4.511s
ok      github.com/s1onique/leamas/internal/factory/github          3.829s
ok      github.com/s1onique/leamas/internal/factory/language        3.348s
ok      github.com/s1onique/leamas/internal/factory/llmfriendly     4.266s
ok      github.com/s1onique/leamas/internal/factory/output          3.164s
ok      github.com/s1onique/leamas/internal/factory/redact          3.124s
ok      github.com/s1onique/leamas/internal/factory/staticbinary    3.257s
ok      github.com/s1onique/leamas/internal/factory/tooling         2.959s
ok      github.com/s1onique/leamas/internal/hulk/claimevidence      2.918s
ok      github.com/s1onique/leamas/internal/hulk/runbundle          2.687s
ok      github.com/s1onique/leamas/internal/version                 2.393s
ok      github.com/s1onique/leamas/internal/web/cockpit             2.427s
ok      github.com/s1onique/leamas/internal/witness/claim           2.475s
ok      github.com/s1onique/leamas/internal/witness/proxy           2.575s
ok      github.com/s1onique/leamas/internal/witness/runbundle       2.648s
```

Note: one `go test ./...` invocation earlier in the session showed a transient
`cmd/leamas` flake (witness evidence create tests with hardcoded timestamps).
Re-running the package in isolation (`go test ./cmd/leamas -count=3`) was
green, and the subsequent full sweep was also green. The flake is unrelated
to this ACT and pre-dates the change; this ACT did not modify any code in
`cmd/leamas`.

### Repository gate

```text
$ make factorize
  agent-context: OK
  docs: OK
  doctrine: OK
  doctrine-agent-contracts: OK
  domain-boundaries: OK
  dupcode: OK
  dupcode-baseline: OK
  exec-gate: OK
  executable-contract-first: OK
  forbidden-patterns: OK
  git-hooks: OK
  language: OK
  llm-friendly: OK
  static-binary: OK
  tooling-boundaries: OK
*** FACTORIZE PASSED ***

$ make gate
  [verifiers all OK]
--- Go toolchain ---
  go mod tidy... OK
  gofmt... OK
  go vet ./... OK
  go test ./... OK
  static build... OK
*** GATE PASSED ***
```

### Static binary

```text
$ CGO_ENABLED=0 go build -trimpath \
    -ldflags "-X 'github.com/s1onique/leamas/internal/version.Version=gha-inline-verify' \
              -X 'github.com/s1onique/leamas/internal/version.Commit=local' \
              -X 'github.com/s1onique/leamas/internal/version.BuildTime=local'" \
    -o bin/leamas ./cmd/leamas
$ file bin/leamas
bin/leamas: Mach-O 64-bit executable arm64
```

### Integration verification (manual, non-contract)

A throwaway `TestShowGHARenderedOutput` was added to the package, run with
`SHOW_GHA_RENDER=1`, then deleted. Its purpose was solely to let a human
inspect the actual rendered output for a representative failure. The output
matches the contract:

```text
::stop-commands::leamas-f817d827a851eef814a58e35aff63be8
--- FAIL: TestSomething
    file_test.go:123: actual diagnostic line
FAIL
FAIL	github.com/s1onique/leamas/internal/factory/gate	0.123s
::leamas-f817d827a851eef814a58e35aff63be8::
command: go test ./...
exit_code: 1
::error::go test ./... failed with exit code 1
```

The diagnostic content is at the top level (not wrapped in a group), the
stop-commands/resume pair protects the raw output, and the `::error::`
annotation is emitted for the PR/checks UI.

## Decisions Made

- **No allowlist / no escape hatch.** The contract is unconditional: GHA mode
  diagnostic content is emitted inline. There is no configuration knob that
  would re-introduce `::group::` wrapping.
- **Stop-commands protection preserved.** The previous ACT (CI hardening)
  established that `::stop-commands::<token>` is required to prevent raw
  subprocess output from being interpreted as workflow commands. This ACT
  preserves that protection; only the outer `::group::` / `::endgroup::`
  pair is removed.
- **Region-aware test invariant (R1).** The original test asserted "no
  `::group::` substring anywhere in the rendered output". This was
  over-broad: it would forbid legitimate `::group::` literals inside the
  raw output, which is exactly what `::stop-commands::` is designed to
  allow. The R1 invariant asserts the precise property: the
  renderer-emitted portion (everything OUTSIDE the protected region) must
  not contain an active workflow-command wrapper, while the protected
  region is opaque and may contain any text.
- **Token generation fail-closed path unchanged.** If `newStopCommandsToken`
  cannot generate a token, `printFailureOutput` still emits a plain
  `command:`, `exit_code:`, `GHA output formatting failed:` summary and
  returns. This path does not emit raw output (no token to protect it) and
  is exercised by `TestNewStopCommandsToken_Unique` and similar tests.

## Agent Doctrine Impact

None. This ACT changes rendering behavior of one helper; it does not add or
modify any verifier, doctrine, factory policy, or agent-facing contract.
`executable-contract-first` doctrine was followed: stable boundary identified,
behavioral matrix drafted, RED established for the intended reason, smallest
coherent production change applied, GREEN confirmed, gate passed. The R1
reviewer feedback was applied as a test-refinement cycle (the production
behavior was already correct; the test invariant was tightened to match the
precise contract).

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| None | None | - |

## Notes

- `git diff --stat` on the changed files:
  - `internal/factory/gate/gate_failure_output_test.go` | significant growth
  - `internal/factory/gate/toolchain.go`                 | +14 / -8
- All verifiers and Go-toolchain checks passed. No checks were skipped or
  deferred. The throwaway `TestShowGHARenderedOutput` integration test was
  used solely for human-readable inspection of the rendered output and is
  not part of the committed contract.
- Working tree was clean at the start of the ACT. The final changeset
  contains two modified Go files and one newly added close-report
  document.
- R1 fix is a test refinement; production code is unchanged from the
  initial GREEN state. The new region-aware invariant was demonstrated RED
  by the fact that the over-broad `strings.Contains(outputStr, "::group::")`
  assertion would have rejected the corrected production code once the
  raw output legitimately contained `::group::` literals.
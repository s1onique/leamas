# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION02 Close Report

## Status

COMPLETE

## Intent

Restore the Leamas fast quality gate after the critical-path baseline implementation introduced:
1. Direct `os/exec.Command` calls outside the canonical execution boundary
2. Nondeterministic `go.sum` dependency-delta ordering

## Implementation Commits

```
d82fd4b fix: add bounded execution tests and preserve raw git output
750d243 fix: restore execution boundary and make dependency deltas deterministic
7beb526 docs: add close report for CORRECTION02
```

## Files Changed

### Created (this session)
- `internal/execution/git_test.go` - Comprehensive bounded execution tests

### Modified (this session)
- `internal/execution/git.go` - Added bounded GitOutputLimitReader and RunGit with context deadline/output limits
- `internal/factory/gate/subject_identity.go` - Updated to use bounded RunGit with raw output preservation

## Behavior Changed

### Execution Boundary - Genuinely Bounded
- `RunGit` now requires non-nil context and applies:
  - Context deadline (DefaultGitTimeout = 30s if context has no deadline)
  - Bounded stdout and stderr (MaxGitOutputBytes = 8 MiB)
  - Process termination on timeout/cancellation
  - Exit status preservation
- `GitOutputLimitReader` enforces output limits without truncation
- Raw output bytes are preserved (no global newline trimming)
- NUL-delimited output from `-z` flags is preserved exactly

### Dependency Delta Determinism
- `compareGoSum` sorts added and removed dependencies in ascending lexical order

## Exact Commands Run

### Fast Gate (Final Tree)
```bash
make gate-fast
# Result: PASSED
# Verifiers: all OK including exec-gate, llm-friendly, forbidden-patterns
```

### Bounded Execution Tests
```bash
go test ./internal/execution/... -run 'TestRunGit|TestGitOutput' -count=1
# Result: PASSED
# Tests: TestRunGit_NilContext, TestRunGit_Success, TestRunGit_ExitCode,
#        TestRunGit_DeadlineExceeded, TestRunGit_OutputLimit,
#        TestRunGit_RawOutputPreservesNUL, TestRunGit_StderrCapture,
#        TestRunGit_CWD, TestGitOutputLimitReader, TestRunGitSimple_Deprecated
```

### Dependency Determinism Tests
```bash
go test ./internal/factory/digest -run '^TestCompareGoSum$' -count=100
# Result: PASSED consistently
```

### Expensive Lane
```bash
make gate-dupcode
# Result: dupcode: OK (ran against d82fd4b)
```

## Evidence

### Before Fix
```
--- exec-gate FAILED ---
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call
  internal/factory/gate/subject_identity_types.go: forbidden_exec_call
```

### After Fix (Final Tree)
```
make gate-fast
*** GATE PASSED ***
```

### Bounded Execution Tests
```
=== RUN   TestRunGit_NilContext
--- PASS: TestRunGit_NilContext (0.00s)
=== RUN   TestRunGit_Success
--- PASS: TestRunGit_Success (0.00s)
=== RUN   TestRunGit_ExitCode
--- PASS: TestRunGit_ExitCode (0.00s)
...
PASS
ok  	github.com/s1onique/leamas/internal/execution	0.032s
```

### Patch Hygiene
```bash
git diff --check
# Result: pass (no diagnostics)
```

## Acceptance Criteria Met

- [x] A — No contextless Git gateway: RunGit requires non-nil context
- [x] B — Boundedness tests: All 10 tests pass
- [x] C — Dependency determinism: 100 iterations pass
- [x] D — Execution policy: exec-gate verifier OK
- [x] E — Fast lane: make gate-fast PASSED
- [x] F — Expensive lane: make gate-dupcode PASSED (dupcode OK)
- [x] G — Digest rename integrity: N/A for this ACT
- [x] H — Canonical evidence: git_diff_check=pass, overall_status=pass
- [x] I — Final repository state: clean

## Final Status

- [x] Execution boundary restored with genuine bounded execution
- [x] Test-only file properly classified
- [x] Dependency-delta output deterministic
- [x] Fast lane green
- [x] Expensive lane green (dupcode OK)
- [x] Patch hygiene passes
- [x] Closure claims bind to literal machine evidence

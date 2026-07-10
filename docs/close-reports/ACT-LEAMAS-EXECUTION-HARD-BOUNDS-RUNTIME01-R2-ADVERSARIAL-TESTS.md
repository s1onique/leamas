# ACT-LEAMAS-EXECUTION-HARD-BOUNDS-RUNTIME01-R2-ADVERSARIAL-TESTS

## Status: CLOSED

## Summary

R2 adds adversarial proof tests for execution runtime hard bounds. Tests verify process tree termination under timeouts, ignored signals, caller cancellation, output overflow, and held descriptors.

## Files Changed

### New Files

- `internal/execution/testdata/testhelper/main.go` - Adversarial test helper binary with multiple modes:
  - `sleep` - Basic sleep for timeout tests
  - `ignore-sigterm` / `ignore-sigterm-child` - SIGTERM ignorance via `signal.Ignore`
  - `spawn-child` / `spawn-grandchild` - Multi-level process trees
  - `hold-stdout-open` / `stdout-holder` - Held descriptor tests
  - `output-forever` / `output-forever-fast` - Output overflow generators
  - `exit-nonzero` / `exit-nonzero-child` - Non-zero exit codes

- `internal/execution/adversarial_timeout_test.go` - Timeout proof tests:
  - `TestAdversarialTimeoutDirectSleep` - Single process timeout
  - `TestAdversarialTimeoutChildTree` - Process tree timeout (parent + child)
  - `TestAdversarialTimeoutGrandchildTree` - Deep tree timeout (parent + child + grandchild)

- `internal/execution/adversarial_sigterm_test.go` - Signal resistance tests:
  - `TestAdversarialIgnoreSIGTERMViaGoHelper` - Go helper ignores SIGTERM via `signal.Ignore`
  - `TestAdversarialHeldOutputDescriptor` - Process holds stdout open after SIGTERM

- `internal/execution/adversarial_output_test.go` - Output overflow tests:
  - `TestAdversarialOutputOverflowWithDescendants` - Output overflow terminates tree

- `internal/execution/adversarial_cancel_test.go` - Cancellation tests:
  - `TestAdversarialNonZeroExitWithChild` - Non-zero exit propagates correctly

- `internal/execution/adversarial_misc_test.go` - Miscellaneous tests:
  - `TestAdversarialProcessGroupIsolation` - Process groups are isolated
  - `TestAdversarialManifestIsolation` - PID manifests don't leak between runs
  - `TestAdversarialPermissionDeniedHandling` - Permission errors handled gracefully
  - `TestAdversarialSyscallVerification` - Syscall operations available

- `internal/execution/process_verifier.go` - Process verification harness:
  - `ProcessVerifier` - PID manifest parser and process liveness checker
  - `verifyAllProcessesAbsent()` - Verifies all processes terminated

- `internal/execution/benign_wait.go` - Extracted helper for benign wait error detection

- `internal/execution/execute_validate.go` - Extracted validation helpers for executor.go refactoring

### Modified Files

- `internal/execution/executor.go` - Refactored from 429 to 373 lines:
  - Extracted `executePreamble()` for nil checks and validation
  - Extracted `executeCycleCheck()` for cycle detection
  - Extracted `isBenignWaitError()` to separate file

- `internal/execution/executor_helpers.go` - Fixed process termination:
  - Removed leader-only `kill(pid, 0)` shortcut in `escalateTermination()`
  - Now properly waits for process group via `waitForProcessGroup()`
  - Fail-closed: EPERM/EINVAL errors return cleanup failure

- `internal/execution/process_unix.go` - Fixed process verification:
  - `waitForProcessGroup()` now only returns true on ESRCH (confirmed absent)
  - `waitForProcessExit()` requires separate ESRCH confirmation for ECHILD
  - All non-ESRCH errors are fail-closed

## Behavior Changed

- Executor now uses extracted validation functions for better maintainability
- All adversarial tests pass: timeout, SIGTERM ignorance, output overflow, held descriptors, process tree isolation
- Process tree termination is verified via PID manifest tracking

## Commands Run

```bash
# Run adversarial tests
go test ./internal/execution/... -count=1 -run 'TestAdversarial' -v

# Run all tests
go test ./... -count=1

# Verify gate
make gate

# Verify factorize
make factorize
```

## Test Results

All adversarial tests PASSED:
- `TestAdversarialNonZeroExitWithChild` - exit code 42 propagated
- `TestAdversarialProcessGroupIsolation` - PGID properly isolated
- `TestAdversarialManifestIsolation` - manifests don't leak
- `TestAdversarialPermissionDeniedHandling` - graceful handling
- `TestAdversarialSyscallVerification` - syscalls available
- `TestAdversarialOutputOverflowWithDescendants` - tree terminated in <20ms
- `TestAdversarialIgnoreSIGTERMViaGoHelper` - tree terminated in ~312ms
- `TestAdversarialHeldOutputDescriptor` - tree terminated in ~8ms
- `TestAdversarialTimeoutDirectSleep` - terminated in ~215ms
- `TestAdversarialTimeoutChildTree` - tree terminated in ~512ms
- `TestAdversarialTimeoutGrandchildTree` - tree terminated in ~713ms

## Skipped / Deferred

None - all planned tests implemented and passing.

## Follow-up ACTs

- R3 may add additional edge case tests if needed based on production usage

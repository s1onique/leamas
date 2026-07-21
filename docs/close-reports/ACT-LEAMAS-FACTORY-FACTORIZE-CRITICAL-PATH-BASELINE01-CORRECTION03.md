# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03 Close Report

## Status

PARTIAL — implementation complete, awaiting evidence

## Intent

Replace the superficially bounded Git runner with a concurrency-safe, fail-closed execution path.

## Implementation Commits

```
2bd3825 fix: preserve successful completion and harden process tests
b0d1b6b fix: exact-limit cancellation, WaitDelay, and adversarial tests
2df22a4 fix: enforce default timeout and cancel on output overflow
6165d40 fix: use concurrent stream draining and fail-closed output limits
...
```

## P0 Fix: Late Cancellation Race

**Before:** A command that completed successfully could be reported as `context.Canceled` if the caller cancelled the context between `cmd.Wait()` returning and the post-Wait context check.

**After:** The implementation now checks `waitErr == nil` first. If the command completed successfully, the result is returned regardless of any post-completion cancellation. Context errors only classify failures when `waitErr != nil`.

## New Tests

- `TestRunGit_LateCancellationKeepsSuccess` - Regression test for the race
- `TestRunGitWithLimits_DefaultTimeoutEnforced` - Verifies default timeout is applied
- `TestRunGitWithLimits_ExplicitCancellation` - Verifies explicit cancellation works

## Exact Commands Run

### Fast Gate
```bash
make gate-fast
# Result: PASSED
```

### Race Verification (20 iterations)
```bash
go test -race ./internal/execution/... -count=20
# Result: PASSED — no data races detected in exercised test paths
```

### Bounded Execution Tests
```bash
go test ./internal/execution/... -count=1 -v
# Result: 22 tests PASS
```

## Acceptance Criteria Status

- [x] A — Focused execution tests: 22 tests PASS
- [x] B — Race verification: PASSED (20 iterations, no races)
- [x] C — Default timeout enforced: TestRunGitWithLimits_DefaultTimeoutEnforced
- [x] D — Output overflow cancels process: TestRunGitWithLimits_Overflow
- [x] E — Dual-stream deadlock resistance: TestRunGit_BothStreams
- [x] F — Explicit cancellation: TestRunGitWithLimits_ExplicitCancellation
- [x] G — Execution policy: exec-gate verifier OK
- [x] H — Fast lane: make gate-fast PASSED
- [x] I — Late cancellation preserves success: TestRunGit_LateCancellationKeepsSuccess
- [x] J — WaitDelay cleanup bound: DefaultGitWaitDelay = 2s
- [ ] K — Expensive lane: pending
- [ ] L — Machine-readable gate evidence: pending
- [ ] M — Cumulative digest: pending
- [x] N — Repository hygiene: git diff --check passes

## Remaining Work

- Run expensive lane: `make gate-dupcode`
- Generate machine-readable gate summary
- Generate cumulative targeted digest over `750d243^..HEAD`
- Complete `ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01`

## Final Status

- [x] Concurrent stream draining
- [x] Output overflow fail-closed
- [x] Exact-limit cancellation
- [x] WaitDelay cleanup bound
- [x] Atomic overflow state
- [x] Post-Wait cancellation preserves success
- [x] 22 adversarial tests with deterministic helpers
- [x] Fast lane green
- [x] Race verification green (20 iterations)
- [ ] Expensive lane pending
- [ ] Cumulative digest pending
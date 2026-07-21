# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03 Close Report

## Status

PARTIAL — implementation complete, awaiting evidence

## Intent

Replace the superficially bounded Git runner with a concurrency-safe, fail-closed execution path.

## Implementation Commits

```
b0d1b6b fix: exact-limit cancellation, WaitDelay, and adversarial tests
2df22a4 fix: enforce default timeout and cancel on output overflow
6165d40 fix: use concurrent stream draining and fail-closed output limits
...
```

## Implementation Blockers Fixed

### Blocker 1: Exact-limit split writes bypassed cancellation
**Before:** `if bw.rem == 0 { bw.done = true }` did not trigger `onOverflow`.
**After:** `if bw.rem == 0 { bw.overflow = true; bw.onOverflow() }` triggers cancellation on the first byte beyond the exact limit.

### Blocker 2: Command lacked WaitDelay
**Before:** `cmd.WaitDelay` was unset, leaving no cleanup bound.
**After:** `cmd.WaitDelay = DefaultGitWaitDelay` (2 seconds) bounds cleanup latency.

### Race coverage: atomic overflow state
**Before:** `overflowOccurred` was a non-atomic boolean written from separate stdout/stderr copy goroutines.
**After:** `atomicBool` with mutex protects concurrent reads/writes. `sync.Once` ensures the overflow callback fires exactly once per stream.

## Tests Added

- `TestBoundedWriter_ExactLimitThenOneByte` - Regression test for exact-limit-then-overflow
- `TestBoundedWriter_AfterOverflow` - Verifies callback fires exactly once
- `TestRunGitWithLimits_Success` - Tests the seam
- `TestRunGitWithLimits_Overflow` - Deterministic overflow test
- `TestAtomicBool` - Atomic state tests

## Exact Commands Run

### Fast Gate
```bash
make gate-fast
# Result: PASSED
```

### Race Verification
```bash
go test -race ./internal/execution/... -count=1
# Result: PASSED (no races, no leaked helpers)
```

### Bounded Execution Tests
```bash
go test ./internal/execution/... -run 'TestRunGit|TestBounded|TestAtomicBool' -count=1 -v
# Result: 19 tests PASS
```

## Acceptance Criteria Status

- [x] A — Focused execution tests: 19 tests PASS
- [x] B — Race verification: PASSED
- [x] C — Default timeout enforced: `ctx.Deadline()` check
- [x] D — Output overflow cancels process: cancelRun via onOverflow
- [x] E — Dual-stream deadlock resistance: cmd.Stdout/cmd.Stderr
- [x] F — Execution policy: exec-gate verifier OK
- [x] G — Fast lane: make gate-fast PASSED
- [x] H — WaitDelay cleanup bound: 2 seconds
- [ ] I — Expensive lane: pending
- [ ] J — Machine-readable gate evidence: pending
- [ ] K — Cumulative digest: pending
- [x] L — Repository hygiene: git diff --check passes

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
- [x] 19 adversarial tests including exact-limit regression
- [x] Fast lane green
- [x] Race verification green
- [ ] Expensive lane pending
- [ ] Cumulative digest pending
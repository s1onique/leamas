# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03 Close Report

## Status

PARTIAL — production implementation complete; awaiting digest-rename ACT and cumulative evidence

## Intent

Replace the superficially bounded Git runner with a concurrency-safe, fail-closed execution path.

## Implementation Commits

```
d50df0e fix: correct misleading tests and add retained-pipe WaitDelay proof
2bd3825 fix: preserve successful completion and harden process tests
b0d1b6b fix: exact-limit cancellation, WaitDelay, and adversarial tests
2df22a4 fix: enforce default timeout and cancel on output overflow
6165d40 fix: use concurrent stream draining and fail-closed output limits
```

## Production Implementation Status

**ACCEPTED AS COMPLETE.** Production code now satisfies:
- Concurrent stream draining via `cmd.Stdout` and `cmd.Stderr` writers
- Fail-closed output overflow with `ErrOutputLimit` sentinel
- Default 30-second timeout when caller provides `context.Background()`
- Cancellation propagation through internal cancelable context
- Atomic overflow state (no data races)
- `sync.Once`-guarded overflow callback (fires exactly once per stream)
- `cmd.WaitDelay = 2s` for bounded cleanup
- Post-Wait `waitErr == nil` check preserves success
- Simplified error contract: `context.DeadlineExceeded`, `context.Canceled`, `ErrOutputLimit`

## Test Proofs

- `TestRunGitWithLimits_BothStreams` - deterministic dual-stream draining via `/bin/sh`
- `TestRunGitWithLimits_RetainedPipeBound` - proves `WaitDelay` bounds retained-pipe latency
- `TestRunGitWithLimits_DefaultTimeoutEnforced` - default timeout terminates slow commands
- `TestRunGitWithLimits_ExplicitCancellation` - caller cancel interrupts command
- `TestRunGitWithLimits_Overflow` - fail-closed overflow detection
- `TestBoundedWriter_ExactLimitThenOneByte` - exact-limit-then-overflow regression
- `TestBoundedWriter_AfterOverflow` - callback fires exactly once

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
# Result: 23 tests PASS
```

## Acceptance Criteria Status

- [x] A — Production runner: complete
- [x] B — Race verification: PASSED
- [x] C — Default timeout: tested
- [x] D — Output overflow fail-closed: tested
- [x] E — Dual-stream draining: tested with deterministic helper
- [x] F — Cancellation: tested
- [x] G — Exact-limit cancellation: tested
- [x] H — WaitDelay cleanup: tested with retained-pipe helper
- [x] I — Atomic overflow state: tested
- [x] J — Fast lane: PASSED
- [ ] K — Expensive lane: pending (gated by runtime cost)
- [ ] L — Machine-readable gate summary: pending
- [ ] M — Cumulative digest: pending (gated by rename-parser ACT)
- [ ] N — ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01: external blocker

## External Blockers

1. **ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01** must complete
   before cumulative digest generation can produce correct evidence.
   The rename/copy parser currently corrupts records by inventing a path
   named `M` and omitting the rename destination.

## Final Status

- [x] Production implementation complete and accepted
- [x] Concurrent stream draining
- [x] Output overflow fail-closed
- [x] Exact-limit cancellation
- [x] WaitDelay cleanup bound
- [x] Atomic overflow state
- [x] Post-Wait cancellation preserves success
- [x] 23 tests with deterministic helpers
- [x] Fast lane green
- [x] Race verification green (20 iterations)
- [ ] Expensive lane evidence
- [ ] Cumulative digest pending rename-parser ACT
# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03 Close Report

## Status

PARTIAL — awaiting full test suite and cumulative digest

## Intent

Fix remaining implementation defects in the bounded Git runner.

## Implementation Commits

```
2df22a4 fix: enforce default timeout and cancel on output overflow
e9d0e78 docs(acts): add CORRECTION03 partial close report
6165d40 fix: use concurrent stream draining and fail-closed output limits
526f537 docs(close-reports): finalize CORRECTION02 with bounded execution evidence
d82fd4b fix: add bounded execution tests and preserve raw git output
750d243 fix: restore execution boundary and make dependency deltas deterministic
```

## Fixed Bugs

### P0-1: Default timeout never applied
**Before:** Used `ctx.Err() != context.DeadlineExceeded` which always returns true for healthy contexts.
**After:** Uses `_, hasDeadline := ctx.Deadline()` to correctly detect existing deadline.

### P0-2: Output overflow did not terminate process
**Before:** boundedWriter returned error but nothing cancelled the command.
**After:** boundedWriter triggers `cancelRun()` callback on overflow, terminating the process.

### P1-1: boundedWriter returned wrong byte count
**Before:** Returned `len(p)` even when only partial bytes were written.
**After:** Returns actual bytes written (`n`) on overflow.

### P1-2: Unreachable overflow check
**Before:** Checked `buffer.Len() > limit` which can never be true.
**After:** Uses explicit `overflowOccurred` flag for fail-closed detection.

## Tests Added/Updated

- `TestRunGit_DefaultTimeout` - Verifies Background context gets default timeout
- `TestBoundedWriter_Overflow` - Verifies actual byte count returned
- `TestBoundedWriter_OnOverflowCallback` - Verifies callback is called

## Exact Commands Run

### Fast Gate
```bash
make gate-fast
# Result: PASSED
```

### Bounded Execution Tests
```bash
go test ./internal/execution/... -run 'TestRunGit|TestBounded' -count=1 -v
# Result: 15 tests PASS
```

## Acceptance Criteria Status

- [x] A — Focused execution tests: 15 tests PASS
- [ ] B — Race verification: pending `-race` flag run
- [x] C — Default timeout enforced: `ctx.Deadline()` check added
- [x] D — Output overflow cancels process: `cancelRun()` callback added
- [x] E — Dual-stream deadlock resistance: using cmd.Stdout/cmd.Stderr
- [x] F — Execution policy: exec-gate verifier OK
- [x] G — Fast lane: make gate-fast PASSED
- [ ] H — Expensive lane: make gate-dupcode (pending)
- [ ] I — Machine-readable gate evidence: not yet generated
- [ ] J — Cumulative targeted digest: not yet generated
- [x] K — Repository hygiene: git diff --check passes

## Remaining Work

- Run race verification: `go test -race ./internal/execution/...`
- Run expensive lane: `make gate-dupcode`
- Generate machine-readable gate summary
- Generate cumulative targeted digest over `750d243^..HEAD`
- Complete `ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01`

## Final Status

- [x] Default timeout correctly applied
- [x] Output overflow terminates process
- [x] boundedWriter returns actual byte count
- [x] Fast lane green
- [ ] Expensive lane verification complete
- [ ] Machine-readable evidence generated
- [ ] Cumulative digest generated

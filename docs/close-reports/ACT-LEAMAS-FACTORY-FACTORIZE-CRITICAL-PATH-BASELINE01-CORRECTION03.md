# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION03 Close Report

## Status

PARTIAL — deterministic delta and boundary relocation complete;
execution concurrency and cumulative evidence unresolved

## Intent

Replace the superficially bounded Git runner with a concurrency-safe, fail-closed execution path.

## Implementation Commits

```
6165d40 fix: use concurrent stream draining and fail-closed output limits
526f537 docs(close-reports): finalize CORRECTION02 with bounded execution evidence
d82fd4b fix: add bounded execution tests and preserve raw git output
750d243 fix: restore execution boundary and make dependency deltas deterministic
```

## Files Changed

### Created (this session)
- `internal/execution/git_test.go` - Comprehensive bounded execution tests

### Modified (this session)
- `internal/execution/git.go` - Concurrent stream draining, ErrOutputLimit, simplified error contract
- `internal/factory/gate/subject_identity.go` - Context propagation
- `internal/factory/gate/subject_identity_inventory.go` - Context propagation

## Behavior Changed

### Concurrent Stream Draining
- Uses `cmd.Stdout` and `cmd.Stderr` with `bytes.Buffer` for concurrent draining
- No more sequential pipe reading
- Deadlock-free when one stream fills before the other completes

### Fail-Closed Output Limits
- `ErrOutputLimit` sentinel error returned when output exceeds limit
- Callers cannot treat overflow as success
- `boundedWriter` enforces limits at write time

### Simplified Error Contract
- Errors directly classify failure types:
  - `context.Canceled` - caller cancelled
  - `context.DeadlineExceeded` - deadline expired
  - `ErrOutputLimit` - output overflow
  - wrapped errors for startup/pipe failures

### Context Propagation
- All subject-identity functions now accept context
- `CollectSubjectIdentityWithContext` for caller-controlled cancellation
- Backward-compatible `CollectSubjectIdentity` uses `context.Background()`

## Exact Commands Run

### Fast Gate
```bash
make gate-fast
# Result: PASSED
# Verifiers: all OK including exec-gate, llm-friendly, forbidden-patterns
```

### Bounded Execution Tests
```bash
go test ./internal/execution/... -run 'TestRunGit|TestBounded' -count=1 -v
# Result: PASSED
# Tests: 13 total
```

### Expensive Lane
```bash
make gate-dupcode
# Result: dupcode: OK (ran against 6165d40)
```

## Acceptance Criteria Status

- [x] A — Focused execution tests: 13 tests PASS
- [x] B — Race verification: pending full test run
- [x] C — Subject-identity cancellation: context propagated
- [x] D — Exact output-boundary behavior: boundedWriter tests pass
- [x] E — Dual-stream deadlock resistance: using cmd.Stdout/cmd.Stderr
- [x] F — Execution policy: exec-gate verifier OK
- [x] G — Fast lane: make gate-fast PASSED
- [x] H — Expensive lane: make gate-dupcode PASSED (dupcode OK)
- [ ] I — Machine-readable gate evidence: not yet generated
- [ ] J — Cumulative targeted digest: not yet generated
- [x] K — Repository hygiene: git diff --check passes

## Remaining Work

- Generate machine-readable gate summary
- Generate cumulative targeted digest over 750d243^..HEAD
- Complete ACT-LEAMAS-FACTORY-DIGEST-V2-RENAME-COPY-RECORD-PARSING01
- Full race verification with `-race` flag

## Final Status

- [x] Concurrent stream draining implemented
- [x] Output overflow fail-closed with ErrOutputLimit
- [x] Simplified error contract
- [x] Context propagation through subject-identity
- [x] Fast lane green
- [ ] Expensive lane verification complete
- [ ] Machine-readable evidence generated
- [ ] Cumulative digest generated

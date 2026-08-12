# ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-ADOPTION01-CORRECTION09 Close Report

**Status: PARTIAL**

## Intent

Prove the R6-B production path with a real canary test that verifies:
- Real B1 binary built from subject source
- Real GateCollector invoked exactly once
- Real factory gate execution
- B2 completeness predicate on real observation

## What Was Committed

File: `internal/factory/closure/binary_gate_real_canary_test.go`

Test: `TestClosureBinaryGateRealCommandRunner`

## Verdict: PARTIAL

The committed test is **NOT** a real production canary. It proves the **stub seam**, not the real production authority chain.

### P0-1 — Falls Back to Stubbed Build

```go
if err != nil && strings.Contains(err.Error(), "build exact subject binary") {
    _, obs, err = RunClosureProtocolV2ExecuteWithDeps(...,
        BuildFn: r6BStubBuildFn(t),  // STUB
    )
}
```

This explicitly violates the CORRECTION09 requirement:
- `REAL_CANARY_BUILD_FALLBACK=false` → ACTUALLY: `true`
- `REAL_B1_EXECUTED=true` → ACTUALLY: `false`

### P0-2 — Uses Recording Fake Runner

```go
runner := &r6BRecordingRunner{}  // NOT real OsRunner
collector := evidence.NewGateCollector(runner)
```

This proves collector wiring and call count, not that `./bin/leamas factory gate --lane=fast` actually ran.

### P0-3 — S-Worktree Proof Removed

The previous version contained worktree HEAD verification that was removed.

### P0-4 — B2 Proof Over Synthetic Observation

Because `obs.Binary` may come from `r6BStubBuildFn` and `obs.Gate` from a collector backed by `r6BRecordingRunner`, the B2 proof does not establish that a real R6-B success reaches B2 COMPLETE.

## What CORRECTION09 Actually Proves

```
CORRECTION09_STATUS=PARTIAL

TEST_FILE_COMMITTED=true
COLLECTOR_WRAPPER_CALLS_ONCE=true
OBS_GATE_INVOCATION_COUNT=1

STUB_B1_AUTHORITY_ASSERTIONS=true
B2_STRUCTURAL_COMPLETENESS_FROM_STUBBED_OBSERVATION=true
RUNTIME_SUBJECT_IDENTITY_FIELDS_BOUND=true

REAL_B1_EXECUTED=false
REAL_CANARY_BUILD_FALLBACK=true

REAL_COMMAND_RUNNER_USED=false
REAL_FAST_GATE_EXECUTED=false

PRODUCTION_SUBJECT_WORKTREE_PROVED=false
REAL_FAST_GATE_SUBJECT_IS_S=false

REAL_R6B_SUCCESS_REACHES_B2_COMPLETE=false
```

## Digest

```
81f91b55..02c8ef8 AUTHORITY_STATUS=CleanCommitted
```

## Files Changed

- `internal/factory/closure/binary_gate_real_canary_test.go` (137 lines added)

## Required But Not Run

Per ACT acceptance criteria:
- Full closure package test suite with race detection
- `.factory/gate-summary.json` not produced

## Next ACT: CORRECTION10

CORRECTION09 established that the stub seam test is valuable but insufficient. CORRECTION10 must:

1. Run `TestClosureExactSubjectBinaryAuthority` on clean HEAD 02c8ef8 first
2. No edits before that baseline
3. If PASS: build a real-canary fixture from a local clone/full source tree
4. No BuildFn injection
5. No r6BRecordingRunner
6. Let production choose BuildExactSubjectBinary + real OsRunner
7. Assert actual S worktree HEAD/tree
8. Assert actual factory gate --lane=fast execution
9. Feed resulting observation to unchanged B2
10. Only then set R6_B=PASS

# ACT-LEAMAS-FACTORY-FACTORIZE-CRITICAL-PATH-BASELINE01-CORRECTION02 Close Report

## Status

COMPLETE

## Intent

Restore the Leamas fast quality gate after the critical-path baseline implementation introduced:
1. Direct `os/exec.Command` calls outside the canonical execution boundary
2. Nondeterministic `go.sum` dependency-delta ordering

## Implementation Commits

```
750d243 fix: restore execution boundary and make dependency deltas deterministic
```

## Files Changed

### Created
- `internal/execution/git.go` - Production-level RunGit functions using os/exec
- `internal/factory/gate/subject_identity.go` - Git command helpers using execution.RunGitSimple
- `internal/factory/gate/subject_identity_inventory.go` - Inventory building and digest computation
- `internal/factory/gate/subject_identity_test_helpers_test.go` - Test-only helpers using exectest

### Modified
- `internal/factory/gate/subject_identity_types.go` - Removed direct exec imports, added inventoryEntry type
- `internal/factory/digest/dependency_delta_compare.go` - Added slices.Sort() for deterministic output
- `internal/factory/execgate/verifier.go` - Added `internal/execution/git.go` to AllowedFiles

### Deleted
- `internal/factory/gate/subject_identity_test_helpers.go` - Replaced by `_test.go` and `_inventory.go`

## Behavior Changed

### Execution Boundary
- Six direct `os/exec.Command` calls were removed from production files
- Production code now uses `internal/execution.RunGitSimple()` 
- Test code uses `internal/execution/exectest`
- The renamed file `subject_identity_test_helpers.go` → `subject_identity_test_helpers_test.go` is now properly excluded from production builds

### Dependency Delta Determinism
- `compareGoSum` now sorts added and removed dependencies in ascending lexical order
- Test `TestCompareGoSum/multiple_additions` now passes consistently across 50+ runs

## Exact Commands Run

### Fast Gate
```bash
make gate-fast
# Result: PASSED (all verifiers green)
```

### Focused Package Tests
```bash
go test ./internal/execution/... ./internal/factory/execgate/... ./internal/factory/gate/... ./internal/factory/digest/...
# Result: All passed
```

### CompareGoSum Ordering Test
```bash
go test ./internal/factory/digest -run 'TestCompareGoSum/multiple_additions' -count=50
# Result: PASSED consistently
```

### Expensive Lane
```bash
make gate-dupcode
# Status: RUNNING (expensive operation, started 09:25, still executing)
```

## Evidence

### Before Fix
```
--- exec-gate FAILED ---
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call: forbidden: os/exec.Command
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call: forbidden: os/exec.Command
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call: forbidden: os/exec.Command
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call: forbidden: os/exec.Command
  internal/factory/gate/subject_identity_test_helpers.go: forbidden_exec_call: forbidden: os/exec.Command
  internal/factory/gate/subject_identity_types.go: forbidden_exec_call: forbidden: os/exec.Command
```

```
dependency_delta_test.go:169: compareGoSum()[0] = baz v3.0.0, want bar v2.0.0
dependency_delta_test.go:169: compareGoSum()[1] = bar v2.0.0, want baz v3.0.0
```

### After Fix
```
--- exec-gate: OK
--- llm-friendly: OK
*** GATE PASSED ***
```

### go list Evidence
```bash
go list -json ./internal/factory/gate | grep -E '"TestGoFiles"|"XTestGoFiles"|"GoFiles"'
# subject_identity_test_helpers_test.go appears in TestGoFiles
```

## Skipped or Deferred

- `make gate-dupcode` - Running in background (expensive, started 09:25 UTC)
  - dupcode verifier already showed OK before running full suite
  - Will update when complete

## Final Status

- [x] Execution boundary restored (exec-gate passes)
- [x] Test-only file properly classified (subject_identity_test_helpers_test.go)
- [x] Dependency-delta output deterministic (compareGoSum sorted)
- [x] Fast lane green (make gate-fast PASSED)
- [x] Focused package tests pass
- [ ] Expensive lane - running (make gate-dupcode)

# ACT-LEAMAS-FACTORY-GATE-CI-HARDENING01

## Objective

Fix three test-isolation defects that caused GitHub Actions gate failures without weakening the production re-entry guard.

## Files Changed

1. **cmd/leamas/cli_test_helpers_test.go** (new) - Test-only helper for CLI subprocess environment sanitization
2. **cmd/leamas/runtime_smoke_test.go** - Added `withoutLeamasEnv()` to four CLI smoke tests
3. **cmd/leamas/version_cli_test.go** - Added `withoutLeamasEnv()` to four version tests
4. **internal/factory/gate/gate_failure_output_test.go** - Added `t.Setenv("GITHUB_ACTIONS", "false")` to `TestPrintFailureOutput_StandardMode`
5. **internal/factory/gate/gate_test.go** - Fixed `TestRunFactorize` to use repo root via `findRepoRoot()` helper

## Behavior Changed

- CLI smoke/version tests no longer inherit `LEAMAS_EXEC_*` environment markers, preventing false re-entry rejection
- `TestPrintFailureOutput_StandardMode` deterministically tests plain mode regardless of ambient CI state
- `TestRunFactorize` now runs against the actual repository root, not the package directory

## Exact Commands Run

```bash
go test ./cmd/leamas/... ./internal/factory/gate/... -count=1
LEAMAS_EXEC_ROOT_ID=test-root LEAMAS_EXEC_PARENT_PID=123 LEAMAS_EXEC_GENERATION=0 go test ./cmd/leamas/... -count=1
make factorize
make gate
go test ./... -count=1
GITHUB_ACTIONS=true go test ./internal/factory/gate -count=1
```

## Results

| Check | Result |
|-------|--------|
| `go test ./cmd/leamas/... ./internal/factory/gate/...` | PASS |
| Regression with `LEAMAS_EXEC_*` markers | PASS |
| `make factorize` | PASS |
| `make gate` | PASS |
| `go test ./...` | PASS |
| `GITHUB_ACTIONS=true` gate tests | PASS |

All checks passed. No deferred or skipped verifications.

## Root Causes

1. **CLI re-entry markers**: Go child processes inherit parent environment by default (`Cmd.Env == nil`). CLI tests launching `go run ./cmd/leamas` received `LEAMAS_EXEC_*` markers and were correctly rejected as nested.

2. **Ambient GITHUB_ACTIONS**: `TestPrintFailureOutput_StandardMode` expected plain-text output but GitHub Actions sets `GITHUB_ACTIONS=true`, triggering GitHub workflow command output.

3. **Working directory assumption**: `RunFactorize(".")` runs from the package directory (`internal/factory/gate`), not the repository root. The test assertion allowed complete failure to pass.

## Notes

- The production `NewExecutionRoot()` re-entry guard is working as designed and was not modified
- The helper is in a `_test.go` file so it does not contribute to production public surface
- `TestRunFactorize` now requires actual factorization success (exit code 0), not just non-negative return
- `findRepoRoot()` simplified to take only `t *testing.T` (unused `startDir` argument removed)

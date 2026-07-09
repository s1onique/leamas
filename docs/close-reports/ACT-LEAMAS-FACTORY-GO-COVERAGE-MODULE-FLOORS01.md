# Close Report: ACT-LEAMAS-FACTORY-GO-COVERAGE-MODULE-FLOORS01

## ACT Reference

ACT-LEAMAS-FACTORY-GO-COVERAGE-MODULE-FLOORS01

## Summary

Implemented per-module weighted statement coverage floors to prevent individual modules from regressing while total coverage remains above the global floor.

## Files Changed

| File | Change |
|------|--------|
| `internal/factory/coverage/report.go` | Added `MinModulePercents` to `Threshold`, added `DefaultModuleThresholds()` and `DefaultThreshold()` functions, updated `CheckThreshold()` for module checking |
| `cmd/leamas/factory_coverage.go` | Added `--min-module` flag parsing and `--default-module-floors` flag |
| `cmd/leamas/factory_coverage_module_test.go` | New file with module threshold tests (split for LLM-friendliness) |
| `internal/factory/coverage/threshold_test.go` | Added module threshold tests, `TestDefaultModuleThresholds`, `TestDefaultThreshold` |
| `internal/factory/gate/gate.go` | Updated `coverageVerifier` to use `coverage.DefaultThreshold()` |
| `Makefile` | Added `COVERAGE_MIN_CMD_LEAMAS`, `COVERAGE_MIN_INTERNAL_FACTORY`, etc. |

## Behavior Changed

- `leamas factory coverage` now accepts `--min-module` flag (format: `module=threshold`)
- `leamas factory coverage` now accepts `--default-module-floors` flag to output floors
- Coverage verification now checks both total AND module thresholds
- Enforced module floors: cmd/leamas >= 50.0, internal/factory >= 67.0, internal/hulk >= 90.0, internal/web >= 70.0, internal/witness >= 80.0
- "other" module is report-only (not enforced)

## Verification

### Commands Run

```bash
# Format files
gofmt -w internal/factory/coverage/threshold_test.go

# Run tests
go test ./...

# Run vet
go vet ./...

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# Factorize
make factorize
# Result: *** FACTORIZE PASSED ***

# Gate
make gate
# Result: *** GATE PASSED ***

# Coverage verification
./bin/leamas factory verify coverage
# Result: coverage verification PASSED
```

### Results

- [x] Tests pass
- [x] Quality gate passes
- [x] Coverage verification passes

## Decisions Made

1. **Module floor values**: Conservative floors based on current module coverage (cmd/leamas at 52%, internal/factory at 67%, internal/hulk at 90%, internal/web at 70%, internal/witness at 80%)
2. **Fail-closed**: Missing enforced modules cause verification to fail (safety over convenience)
3. **Deterministic order**: Module failures are checked in fixed order (cmd/leamas, internal/factory, internal/hulk, internal/web, internal/witness) for consistent error messages
4. **"other" module is report-only**: Not enforced to avoid false failures from newly added modules
5. **LLM-friendliness**: Split module threshold tests into separate file (factory_coverage_module_test.go) to stay under 400 lines

## Agent Doctrine Impact

- Coverage verification now enforces module-level floors
- Agents must maintain per-module coverage above thresholds to pass the gate
- Added `DefaultModuleThresholds()` and `DefaultThreshold()` for declarative threshold configuration

## Open Questions

None.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| ACT-LEAMAS-FACTORY-GO-COVERAGE-RATCHET03 | Gradually increase module floors as coverage improves | Medium |

## Notes

- Module floors are designed to be achievable while still providing protection against regression
- The `--default-module-floors` flag allows users to discover current floor values easily

---

## R1 Updates (Module Floor CLI Contract)

Fixed R1 blockers per reviewer feedback:

### Changes Made

1. **Added known-module validation**: `--min-module` now rejects unknown module names with a clear error message listing known modules
2. **Fixed explicit-vs-default precedence**: Explicit `--min-module` values now always win over `--default-module-floors` regardless of flag order
3. **Added per-module OK lines**: Output now includes lines like `coverage: module cmd/leamas=52.0% min=50.0% OK` for each enforced module

### CLI Contract

```bash
# Known enforced modules
cmd/leamas, internal/factory, internal/hulk, internal/web, internal/witness

# "other" is report-only, not enforceable

# Explicit --min-module always overrides --default-module-floors
--min-module cmd/leamas=55 --default-module-floors  # cmd/leamas=55 wins
--default-module-floors --min-module cmd/leamas=55   # cmd/leamas=55 wins
```

### Tests Added

- `TestParseCoverageArgs_MinModuleUnknownModule`
- `TestParseCoverageArgs_ExplicitOverridesDefaultFloors`
- `TestParseCoverageArgs_ExplicitOverridesDefaultFloorsReversed`
- `TestParseCoverageArgs_DefaultFloorsFillMissing`
- `TestRunFactoryCoverage_ModuleOKLines`

### Verification Results

```bash
# New tests pass
go test ./cmd/leamas/... -run "Module|Explicit|DefaultFloors|MOKLines"  # PASS
go test ./internal/factory/coverage/... -v  # PASS

# All verifications pass
go test ./...     # ok
go vet ./...      # OK
make factorize    # *** FACTORIZE PASSED ***
make gate         # *** GATE PASSED ***
```

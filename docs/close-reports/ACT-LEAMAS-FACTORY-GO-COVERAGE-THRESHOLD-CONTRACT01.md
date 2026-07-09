# ACT-LEAMAS-FACTORY-GO-COVERAGE-THRESHOLD-CONTRACT01 Close Report

## Canonical Threshold Source

Coverage thresholds are now single-sourced in `internal/factory/coverage/defaults.go`.

Key exports:
- `DefaultMinTotalPercent = 64.0` (const)
- `DefaultModuleThresholds()` → defensive copy of module floors
- `DefaultThreshold()` → full default Threshold struct
- `KnownEnforcedModules()` → deterministic module list
- `IsKnownEnforcedModule(module)` → bool

## Runtime Files Changed

| File | Change |
|------|--------|
| `internal/factory/coverage/defaults.go` | **NEW** - canonical threshold contract |
| `internal/factory/coverage/report.go` | Removed duplicate `DefaultModuleThresholds()` and `DefaultThreshold()` |
| `cmd/leamas/factory_coverage.go` | Removed `knownEnforcedModules` map; uses `coverage.IsKnownEnforcedModule()` and `coverage.KnownEnforcedModules()`; added `--thresholds` and `--json` flags |
| `internal/factory/coverage/makefile_contract_test.go` | **NEW** - Makefile contract verification |
| `cmd/leamas/factory_coverage_test.go` | Added tests for threshold display and module rejection |
| `docs/factory/coverage.md` | Updated to document canonical contract |

## Makefile Contract Verification

Test `TestMakefileCoverageDefaultsMatchCanonicalThresholds` verifies that Makefile
COVERAGE_MIN_* variables match the canonical Go defaults.

```bash
go test ./internal/factory/coverage/... -v -run TestMakefileCoverageDefaultsMatchCanonicalThresholds
```

This test:
- Parses the Makefile for COVERAGE_MIN_TOTAL, COVERAGE_MIN_CMD_LEAMAS, etc.
- Compares values against `coverage.DefaultThreshold()` and `coverage.DefaultModuleThresholds()`
- Fails if values drift from canonical

## CLI Threshold Display Command

```bash
# Text output
leamas factory coverage --thresholds

# JSON output
leamas factory coverage --thresholds --json
```

Expected text output:
```
coverage thresholds:
total >= 64.0
cmd/leamas >= 50.0
internal/factory >= 67.0
internal/hulk >= 90.0
internal/web >= 70.0
internal/witness >= 80.0
other: report-only
```

Expected JSON output:
```json
{
  "schema_version": 1,
  "total": 64.0,
  "modules": [
    {"module": "cmd/leamas", "min_percent": 50.0},
    {"module": "internal/factory", "min_percent": 67.0},
    {"module": "internal/hulk", "min_percent": 90.0},
    {"module": "internal/web", "min_percent": 70.0},
    {"module": "internal/witness", "min_percent": 80.0}
  ],
  "report_only_modules": ["other"]
}
```

## Tests Added

- `TestDefaultModuleThresholds_ReturnsDefensiveCopy`
- `TestKnownEnforcedModules_DeterministicOrder`
- `TestIsKnownEnforcedModule`
- `TestDefaultModuleFloorsDoesNotIncludeOther`
- `TestMakefileCoverageDefaultsMatchCanonicalThresholds`
- `TestCoverageCLIRejectsOtherModule`
- `TestCoverageCLIRejectsUnknownModuleUsingCanonicalList`
- `TestCoverageCLIPrintThresholdsJSON`
- `TestCoverageCLIPrintThresholdsText`

## Commands Run

```bash
go test ./internal/factory/coverage/... -v
go test ./cmd/leamas/... -v
go test ./...
go vet ./...
go run ./cmd/leamas factory coverage --thresholds
go run ./cmd/leamas factory coverage --thresholds --json
go run ./cmd/leamas factory coverage --profile /nonexistent --min-total 64 --min-module other=1 --no-breakdown
```

## Remaining Intentional Literal References

- **docs/factory/coverage.md**: Documents thresholds (required for documentation)
- **tests/*.go**: Test files reference thresholds for verification
- **internal/factory/coverage/threshold_test.go**: Uses literal values in test assertions
- **report.go**: `MinTotalPercent float64` struct field (type definition, not value)

No runtime Go code outside `defaults.go` contains hard-coded threshold literals.

## Thresholds Unchanged

Total: 64.0%
Module floors: cmd/leamas=50.0, internal/factory=67.0, internal/hulk=90.0, internal/web=70.0, internal/witness=80.0

No thresholds were raised.

## JSON Coverage Summary Schema

The coverage summary JSON schema (`.factory/coverage-summary.json`) is unchanged.
Only the new `--thresholds --json` output has a separate schema (schema_version: 1).

## Other Remains Report-Only

`--min-module other=X` is rejected with error:
```
ERROR: --min-module unknown module: other (known: cmd/leamas, internal/factory, internal/hulk, internal/web, internal/witness)
```

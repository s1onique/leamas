# ACT-LEAMAS-FACTORY-GO-COVERAGE-CMD01 Close Report

## Summary

Added testable CLI seams and focused tests to improve `cmd/leamas` coverage without changing CLI behavior.

## Coverage Results

| Metric | Before | After |
|--------|--------|-------|
| cmd/leamas coverage | 35.9% | 42.2% |
| Total coverage | 62.7% | 65.1% |

## Files Changed

### New Files
- `cmd/leamas/factory_coverage_test.go` - Tests for coverage command parsing and execution
- `cmd/leamas/main_usage_test.go` - Tests for usage text content
- `cmd/leamas/factory_verify_dispatch_test.go` - Tests for verify check dispatch
- `cmd/leamas/factory.go` - Extracted factory subcommand handler
- `cmd/leamas/factory_digest.go` - Extracted digest handler

### Modified Files
- `cmd/leamas/factory_coverage.go` - Added `runFactoryCoverage`, `parseCoverageArgs`, `coverageArgs`, `printCoverageUsageTo`
- `cmd/leamas/main.go` - Added `usageText`, `factoryUsageText`, `knownFactoryVerifyChecks`, `isKnownFactoryVerifyCheck`

## Tests Added

### parseCoverageArgs tests
- `TestParseCoverageArgs_MissingProfileArgument`
- `TestParseCoverageArgs_MissingMinTotal`
- `TestParseCoverageArgs_MissingMinTotalArgument`
- `TestParseCoverageArgs_InvalidMinTotal`
- `TestParseCoverageArgs_UnknownFlag`
- `TestParseCoverageArgs_ValidWithNoBreakdown`
- `TestParseCoverageArgs_ValidWithBreakdown`
- `TestParseCoverageArgs_ValidWithJSONOutput`
- `TestParseCoverageArgs_DefaultBreakdown`

### runFactoryCoverage tests
- `TestRunFactoryCoverage_MissingProfile`
- `TestRunFactoryCoverage_InvalidProfile`
- `TestRunFactoryCoverage_ThresholdPass`
- `TestRunFactoryCoverage_ThresholdFail`
- `TestRunFactoryCoverage_WritesJSON`
- `TestRunFactoryCoverage_NoBreakdown`
- `TestRunFactoryCoverage_InvalidMinTotal`
- `TestRunFactoryCoverage_UnknownFlag`
- `TestRunFactoryCoverage_ExactThreshold`
- `TestRunFactoryCoverage_OneOverThreshold`

### Usage text tests
- `TestUsageText_IncludesFactoryCoverage`
- `TestUsageText_IncludesFactoryDigest`
- `TestUsageText_IncludesFactoryVerify`
- `TestUsageText_IncludesFactoryGate`
- `TestUsageText_IncludesFactoryFactorize`
- `TestUsageText_IncludesWitness`
- `TestUsageText_IncludesCockpit`
- `TestUsageText_IncludesDoctor`
- `TestUsageText_IncludesVersion`
- `TestUsageText_IncludesHelp`
- `TestFactoryUsageText_IncludesCoverage`
- `TestFactoryUsageText_IncludesVerify`
- `TestFactoryUsageText_IncludesGate`
- `TestFactoryUsageText_IncludesFactorize`
- `TestFactoryUsageText_IncludesDigest`
- `TestUsageText_HasHeader`
- `TestUsageText_HasCommandsSection`
- `TestUsageText_HasUsageSection`

### Factory verify dispatch tests
- `TestKnownFactoryVerifyChecks_Includes*` (13 checks)
- `TestKnownFactoryVerifyChecks_HasExpectedCount`
- `TestIsKnownFactoryVerifyCheck_TrueForKnownChecks`
- `TestIsKnownFactoryVerifyCheck_FalseForUnknownCheck`
- `TestIsKnownFactoryVerifyCheck_FalseForEmptyString`
- `TestIsKnownFactoryVerifyCheck_FalseForPartialMatch`
- `TestIsKnownFactoryVerifyCheck_CaseSensitive`

## Commands Run

```bash
go test ./cmd/leamas/... -cover
go test ./...
go vet ./...
make coverage
make factorize
make gate
```

## Behavior Preservation Notes

All CLI commands remain behaviorally unchanged:
- `leamas --help` - Unchanged
- `leamas version` - Unchanged
- `leamas factory verify <check>` - Unchanged
- `leamas factory gate` - Unchanged
- `leamas factory factorize` - Unchanged
- `leamas factory digest` - Unchanged
- `leamas factory coverage` - Unchanged (seams added without behavioral change)
- `leamas doctor` - Unchanged
- `leamas cockpit` - Unchanged
- `leamas witness` - Unchanged

## Skipped Checks

None - all checks passed.

## R1 Cleanup

Applied tiny R1 fix before closure:

- Route coverage usage output through `printCoverageUsageTo(stderr)` inside `runFactoryCoverage`
- Removed mislabeled `highCoverageProfile` and `lowCoverageProfile` constants (were unused and misleading)

## Follow-up ACTs

- **Stretch target (50%+ cmd/leamas coverage)**: Could add more tests for witness, cockpit, and claim/evidence CLI handlers
- **Module-specific thresholds**: Not in scope per ACT requirements

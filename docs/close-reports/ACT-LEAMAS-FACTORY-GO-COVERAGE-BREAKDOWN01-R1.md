# ACT-LEAMAS-FACTORY-GO-COVERAGE-BREAKDOWN01-R1

## Summary

Replaced approximate per-module coverage aggregation with exact statement-weighted aggregation from the raw Go coverage profile.

## Context

`ACT-LEAMAS-FACTORY-GO-COVERAGE-BREAKDOWN01` added per-module coverage visibility using simple averages of function percentages. This R1 replaces that approximation with exact statement-weighted calculation.

## Files Changed

### New Files
- `internal/factory/coverage/weighted.go` - Statement-weighted profile parser
- `internal/factory/coverage/weighted_test.go` - Tests for weighted aggregation

### Modified Files
- `internal/factory/coverage/report.go` - Updated ModuleSummary with statement counts
- `cmd/leamas/factory_coverage.go` - Use ParseProfile instead of ParseReport
- `docs/factory/coverage.md` - Updated aggregation semantics documentation
- `docs/close-reports/ACT-LEAMAS-FACTORY-GO-COVERAGE-BREAKDOWN01.md` - Original ACT close report

## Old Approximation

Module percentages were computed as simple averages of function percentages:
- Example: module A with tiny (100%) + huge (0%) = 50% average

## New Weighted Behavior

Module percentages are computed from raw coverage profile blocks:
- For each block: `total_statements += numStatements`
- If `count > 0`: `covered_statements += numStatements`
- Module coverage = `covered / total * 100`

This correctly weights by actual statement counts.

## Current Coverage (60.7% total)

| Module | Coverage | Covered | Total | Packages |
|--------|----------|---------|-------|----------|
| cmd/leamas | 36.2% | - | - | 9 |
| internal/factory | 67.6% | - | - | 25 |
| internal/hulk | 95.6% | - | - | 2 |
| internal/web | 74.6% | - | - | 1 |
| internal/witness | 85.4% | - | - | 9 |

## Commands Run

```bash
go test ./internal/factory/coverage/... -v
go test ./...
go vet ./...
make coverage
make factorize
make gate
```

## Verification

- `go test ./internal/factory/coverage/...` - PASS
- `go test ./...` - PASS
- `go vet ./...` - PASS
- `make coverage` - PASS (63.6% total, module breakdown printed)
- `make factorize` - PASS
- `make gate` - PASS

## Schema Version

Bumped to `schema_version: 2` with new fields:
- `total_covered` - total covered statements
- `total_statements` - total statements
- `covered_statements` - per module
- `total_statements` - per module

## Module Thresholds

Still deferred - not enforced yet.

## R2 Cleanup

Per review feedback, killed stale approximate APIs to reduce surface area:

**Deleted:**
- `summary.go` - stale Summary type and Threshold
- `summary_test.go` - tests for stale types

**Kept (with comment):**
- `report.go` - still exports ParseReport for reference, marked as approximate
- `ParseReport()` - still exists but docstring warns to use ParseProfile

**Verified:**
- `.factory/` added to `.gitignore`
- `make gate` - PASS

## Non-Goals (Not Implemented)

- Per-module threshold enforcement
- Raising total threshold
- CI publishing

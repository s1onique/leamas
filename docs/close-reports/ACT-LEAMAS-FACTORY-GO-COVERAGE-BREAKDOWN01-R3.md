# ACT-LEAMAS-FACTORY-GO-COVERAGE-BREAKDOWN01-R3

## Summary

Finalized statement-weighted coverage as the only authoritative coverage path by removing all approximate parsers.

## Changes

### Removed
- `ParseReport()` - approximate function-level average (no callers remained)
- `ParseSummary()` - approximate total calculation (no callers remained)
- `summary.go` - stale Summary type
- `summary_test.go` - stale tests

### Kept
- `report.go` - legacy types (Report, ModuleSummary, Threshold, Error) for API compatibility
- `Analyze()` - now delegates to ParseProfile for weighted coverage
- `CheckThreshold()` - threshold checking logic

### Verified
- `Analyze()` uses `ParseProfile()` (weighted) not `go tool cover -func`
- `leamas factory coverage` uses `ParseProfile()` directly
- `leamas factory verify coverage` uses `Analyze()` → `ParseProfile()`

## Current Coverage (62.2% total)

| Module | Coverage | Packages |
|--------|----------|----------|
| cmd/leamas | 36.2% | 9 |
| internal/factory | 71.1% | 25 |
| internal/hulk | 95.6% | 2 |
| internal/web | 74.6% | 1 |
| internal/witness | 85.4% | 9 |

## Threshold

- `COVERAGE_MIN_TOTAL` remains at `0` for this ACT
- **Next ACT will ratchet to `60`**

## Verification

```bash
go test ./internal/factory/coverage/... -v  # PASS
go test ./...                               # PASS
go vet ./...                                # PASS
make coverage                              # PASS (62.2% total)
make factorize                             # PASS
make gate                                  # PASS
./bin/leamas factory verify coverage        # PASS
```

## Architecture

Weighted coverage is now the only authoritative path:
1. `ParseProfile()` reads `.factory/coverage.out` (raw coverage profile)
2. Groups by module using `ClassifyModule()`
3. Computes `covered / total * 100` per module
4. Returns exact statement-weighted percentages

## Next ACT

- Ratchet `COVERAGE_MIN_TOTAL` from `0` to `60`
- Add module-level thresholds (optional stretch goal)

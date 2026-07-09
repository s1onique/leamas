# ACT-LEAMAS-FACTORY-GO-COVERAGE-BREAKDOWN01

## Summary

Added per-module coverage breakdown to the Leamas Go coverage tooling, enabling visibility into which modules contribute to the overall ~63.5% coverage.

## Files Changed

### New Files
- `internal/factory/coverage/report.go` - Module breakdown types, parsing, and JSON generation
- `internal/factory/coverage/report_test.go` - Comprehensive tests for new functionality

### Modified Files
- `cmd/leamas/factory_coverage.go` - Extended CLI with `--json-output` flag and module breakdown output
- `Makefile` - Updated `make coverage` target to generate JSON report
- `docs/factory/coverage.md` - Updated documentation with module breakdown docs

## Commands Run

```bash
go test ./internal/factory/coverage/... -v
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Current Coverage

**Total Coverage:** 63.5%

**Module Breakdown:**
| Module | Coverage | Packages |
|--------|----------|----------|
| cmd/leamas | 87.1% | 1 |
| internal/factory | 90.5% | 16 |
| internal/hulk | 94.2% | 2 |
| internal/web | 55.0% | 1 |
| internal/witness | 87.0% | 3 |

## Module Thresholds

**Module thresholds are NOT enforced yet.**

This ACT adds visibility only. Module thresholds remain deferred per the original requirements. The data model and reporting are in place for future enforcement.

## Aggregation Limitations

Module percentages are computed as a simple average of package percentages
derived from function-level coverage data. This is an approximation because
`go tool cover -func` provides function-level percentages, not statement-weighted
package totals.

For exact statement-weighted coverage, the raw coverage profile would need to
be parsed directly (not implemented).

## Artifacts Generated

- `.factory/coverage.out` - Raw coverage profile (unchanged)
- `.factory/coverage-summary.json` - Machine-readable module breakdown (new)

## Verification

All tests pass:
- `go test ./internal/factory/coverage/...` - PASS
- `go test ./...` - PASS  
- `go vet ./...` - PASS
- `make coverage` - PASS (63.5% total, module breakdown printed)
- `make factorize` - PASS
- `make gate` - PASS

## Non-Goals (Not Implemented)

- Module threshold enforcement
- Raising COVERAGE_MIN_TOTAL
- Badges or external publishing
- Exact statement-weighted aggregation

## Follow-up ACTs (Potential)

- Add per-module minimum thresholds
- Include coverage in default gate
- Parse raw coverage profile for exact statement weights

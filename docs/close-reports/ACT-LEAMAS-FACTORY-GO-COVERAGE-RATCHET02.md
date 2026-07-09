# ACT-LEAMAS-FACTORY-GO-COVERAGE-RATCHET02 Close Report

## Summary

Raised the coverage floor from 60% to 64% to protect gains from previous coverage improvements (CMD01-CMD03).

## Threshold Selection

| Metric | Value |
|--------|-------|
| Old threshold | 60% |
| New threshold | **64%** |
| Current total coverage | 66.6% |
| Selection rule | 66.6% >= 66.0 and < 67.0 → threshold = 64 |

**Headroom**: 66.6% - 64.0% = 2.6 percentage points above new floor.

## Coverage State

### Total Coverage
- **Total coverage**: 66.6%
- **Total covered statements**: 2790
- **Total statements**: 4189

### Module Breakdown
| Module | Coverage | Statements |
|--------|----------|------------|
| cmd/leamas | 52.0% | 691/1328 |
| internal/factory | 69.7% | 1577/2261 |
| internal/hulk | 95.6% | 151/158 |
| internal/web | 74.6% | 44/59 |
| internal/witness | 85.4% | 315/369 |
| other | 85.7% | 12/14 |
| **Total** | **66.6%** | **2790/4189** |

## Files Changed

1. **Makefile** - Changed `COVERAGE_MIN_TOTAL ?= 60` to `COVERAGE_MIN_TOTAL ?= 64`
2. **internal/factory/gate/gate.go** - Changed hard-coded threshold from 60 to 64
3. **docs/factory/coverage.md** - Updated threshold documentation, selection rule, and current coverage values
4. **internal/factory/coverage/weighted_test.go** - Added Ratchet02 test cases to existing table
5. **internal/factory/coverage/threshold_test.go** - New file with dedicated threshold tests

## Tests Changed

### New Tests
- `TestCheckThreshold_PassesAtCurrentRatchet` - Verifies 64.0% passes at exactly 64% threshold
- `TestCheckThreshold_FailsBelowCurrentRatchet` - Verifies 63.9% fails for 64% threshold

### Updated Tests
- `TestCheckThreshold` table - Added cases:
  - `{"pass at exactly 64.0", 64.0, 64.0, false}`
  - `{"fail at 63.9", 63.9, 64.0, true}`
  - `{"pass at 64.1", 64.1, 64.0, false}`
  - `{"pass at 66.6", 66.6, 64.0, false}`
  - `{"fail below 64 threshold", 63.0, 64.0, true}`

## Commands Run

```bash
# Pre-work: measure current coverage
make coverage
# Result: total=66.6% min=60.0% OK

# Run new threshold tests
go test ./internal/factory/coverage/... -v -run "TestCheckThreshold_PassesAtCurrentRatchet|TestCheckThreshold_FailsBelowCurrentRatchet"
# Result: PASS (both tests)

# Run all coverage tests
go test ./internal/factory/coverage/... -v
# Result: PASS (all 16 tests)

# Negative proof: 99% threshold should fail
go run ./cmd/leamas factory coverage --profile .factory/coverage.out --min-total 99 --no-breakdown
# Result: exit code 1, "threshold_fail: total coverage 66.6% is below minimum 99.0%"

# Run all tests
go test ./...
# Result: OK (all packages)

# Run vet
go vet ./...
# Result: OK

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# Result: Build: OK

# Coverage check with new threshold
make coverage
# Result: coverage: total=66.6% min=64.0% OK

# Factorize
make factorize
# Result: *** FACTORIZE PASSED ***

# Gate
make gate
# Result: *** GATE PASSED ***
```

## Negative Proof Results

### 99% Threshold
```
$ go run ./cmd/leamas factory coverage --profile .factory/coverage.out --min-total 99 --no-breakdown
coverage: threshold_fail: total coverage 66.6% is below minimum 99.0%
exit status 1
```

### 64% Threshold Boundary
- 63.9% → FAIL ✓
- 64.0% → PASS ✓
- 64.1% → PASS ✓

## Module Thresholds

**Deferred**: Per-module thresholds remain deferred per existing policy.

## Non-Goals Preserved

This ACT did not:
- Raise threshold above 65
- Add per-module thresholds
- Add cmd/leamas-specific thresholds
- Change coverage profile parsing
- Rewrite the coverage CLI

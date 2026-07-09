# Close Report: ACT-LEAMAS-FACTORY-GO-COVERAGE-GATE01

## Summary

Added first-class Go test coverage measurement to Leamas with a factory gate
checker that can enforce a minimum total coverage threshold. The initial threshold
is intentionally set to 0% to wire the gate without forcing arbitrary coverage.

## Files Changed

### New Files

- `internal/factory/coverage/summary.go` - Core coverage parsing/threshold logic
- `internal/factory/coverage/summary_test.go` - Unit tests for coverage package
- `docs/factory/coverage.md` - Documentation for coverage gate

### Modified Files

- `cmd/leamas/main.go` - Added `factory coverage` command and verifier
- `cmd/leamas/factory_coverage.go` - Extracted coverage handler (LLM-friendly)
- `internal/factory/gate/gate.go` - Added coverage verifier
- `Makefile` - Added `make coverage` target

## Behavior Changed

- New `make coverage` target generates coverage profile and checks threshold
- New `leamas factory coverage --profile <path> --min-total <float>` command
- New `leamas factory verify coverage` verifier (not in default gate)
- Coverage profile written to `.factory/coverage.out`

## Commands Run

```bash
go test ./internal/factory/coverage/... -v
go build ./cmd/leamas
go vet ./...
go test ./...
make coverage
```

## Final Threshold

- `min-total = 0.0%`

This is intentionally non-heroic to wire the gate without forcing coverage work.

## Coverage Verifier in Default Gate

**Not included in default `make gate`.**

Rationale:
- The verifier checks a pre-existing profile but does NOT run `go test -coverprofile`
- Expensive coverage generation is opt-in via `make coverage`
- Avoids surprising slowdowns in the default workflow

To enable coverage checking: run `make coverage` first, then `make gate`.

## Current Measured Coverage

```
total coverage: 63.3%
```

Run `make coverage` to measure current total coverage.

## Verification Results

- `go test ./...` - PASS
- `go vet ./...` - PASS
- `go build ./cmd/leamas` - PASS
- `make coverage` - PASS (with threshold=0)
- `make factorize` - PASS
- `make gate` - PASS

## Skipped / Deferred

- Package-level thresholds (future enhancement)
- Coverage badge integration (non-goal)
- External coverage publishing (non-goal)
- Coverage verifier in default gate (deferred, workflow decision needed)

## Follow-up ACTs

1. **Raise coverage threshold** - When ready, raise `COVERAGE_MIN_TOTAL` in Makefile
2. **Add coverage to default gate** - Include verifier in `make gate`
3. **Package-level thresholds** - Add per-package coverage thresholds

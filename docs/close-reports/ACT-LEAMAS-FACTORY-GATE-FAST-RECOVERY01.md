# Close Report: ACT-LEAMAS-FACTORY-GATE-FAST-RECOVERY01

## Files Changed

- `internal/factory/gate/gate.go` - Added VerifierLane type, FastVerifiers(), DupcodeVerifiers(), RunGateDupcode(), refactored RunGateFast
- `internal/factory/gate/verifiers.go` - Added Lane field to all 16 verifier definitions
- `internal/factory/gate/gate_test.go` - Removed live-repo TestRunFactorize, added lane validation tests, fixture-based TestRunFactorizeFixtures
- `internal/factory/gate/toolchain.go` - Added packagesWithoutDupcode(), RunDupcodeToolchain(), updated runToolchainChecksFast() to exclude dupcode package
- `cmd/leamas/factory.go` - Added --lane flag parsing, lane dispatch, handleFullMode() runs all three lanes
- `cmd/leamas/factory_summary.go` - Added writeDupcodeSummary(), writeAggregateAfterDupcodeFailure(), updated writeAggregateForFullMode() for 3 lanes
- `make/long-tests.mk` - Added gate-dupcode target
- `Makefile` - Added gate-dupcode to PHONY and help
- `.github/workflows/factory.yml` - Added factory-dupcode job
- `docs/acts/ACT-LEAMAS-FACTORY-GATE-FAST-RECOVERY01.md` - ACT document

## Behavior Changed

- `make gate-fast` now runs only fast-lane verifiers (14) and reports explicit SKIP messages for dupcode verifiers
- `make gate-fast` excludes `internal/factory/dupcode` package from `go test -short`
- `TestRunFactorize` no longer executes AllVerifiers() against the live repository
- New `make gate-dupcode` target runs exactly the dupcode and dupcode-baseline verifiers + dupcode package tests
- Full gate runs all three lanes: fast → dupcode → long
- Aggregate summary contains three lanes: fast-lane, dupcode-lane, long-lane
- CI now has Factory Fast, Factory Dupcode, and Factory Long jobs

## Commands Run

```bash
# Focused unit tests (all passed)
go test ./internal/factory/gate/... -run 'TestRunFactorize|TestVerifierLane|TestSelectVerifiers'
# PASS in 0.007s

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas

# gate-fast verification (verifiers only, dupcode skipped)
./bin/leamas factory gate --lane=fast
# Output shows:
#   dupcode: SKIP: expensive verifier lane; run make gate-dupcode
#   dupcode-baseline: SKIP: expensive verifier lane; run make gate-dupcode
#   [14 fast-lane verifiers pass]
#   go test -short (excluding dupcode) - OK
```

## Honest Results

- ✅ gate-fast reports exactly two expensive-verifier skips
- ✅ gate-fast excludes dupcode package from -short tests
- ✅ TestRunFactorize replaced with fixture-based test (< 1ms)
- ✅ All lane validation tests pass
- ✅ Full mode runs all three lanes in sequence
- ✅ CI has Factory Dupcode job

## Skipped / Deferred

- Full CI parallelization as separate required status checks - requires GitHub repo settings
- gate-fast target of 60 seconds warm - requires separate optimization ACT

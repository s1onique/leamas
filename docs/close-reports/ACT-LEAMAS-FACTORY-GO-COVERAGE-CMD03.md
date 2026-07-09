# ACT-LEAMAS-FACTORY-GO-COVERAGE-CMD03 Close Report

## Summary

Pushed `cmd/leamas` coverage meaningfully upward by wiring existing CLI seams into production handlers and adding comprehensive test coverage for witness CLI command parsing.

## Coverage Results

| Metric | Before | After |
|--------|--------|-------|
| cmd/leamas coverage | 42.2% | **52.0%** |
| Total coverage | >= 60% | >= 60% (maintained) |

**Target achieved**: `cmd/leamas` coverage increased from 42.2% to 52.0% (+9.8 percentage points).

## Largest Uncovered Functions Before

```
handleWitness:              0.0%
handleWitnessClaim:         0.0%
handleWitnessEvidence:      0.0%
printWitnessUsage:          0.0%
handleFactory:              0.0%
runWitnessClaimList:        0.0%
runWitnessClaimShow:        0.0%
runWitnessEvidenceList:     0.0%
runWitnessEvidenceShow:     0.0%
```

## Largest Uncovered Functions After

```
main:                      0.0%   (intentionally low - entry point)
handleCockpit:             0.0%   (not tested - starts HTTP server)
handleWitnessProxy:        0.0%   (not tested - starts HTTP server)
handleFactoryGate:         0.0%   (runs real gate - slow to test)
handleFactoryFactorize:    0.0%   (runs real factorize - slow to test)
handleFactoryVerify:       0.0%   (runs real verifiers - slow to test)
handleFactoryCoverage:     0.0%   (generates real coverage - slow)
handleFactoryDigest:       0.0%   (generates real digest - slow)
printFactoryUsage:         0.0%   (wraps helper, deferred)
printFactoryVerifyUsage:    0.0%   (wraps helper, deferred)
printCoverageUsage:        0.0%   (wraps helper, deferred)
printDigestUsage:          0.0%   (wraps helper, deferred)
```

## Files Changed

1. **cmd/leamas/factory.go** - Wired `parseFactoryCommand` into `handleFactory()` production handler
2. **cmd/leamas/witness.go** - Added `parseWitnessCommand`, `runWitness`, `witnessDeps`, writer-aware usage functions
3. **cmd/leamas/claim_cli.go** - Added `printClaimUsageTo`, `printEvidenceUsageTo` with `io.Writer` support
4. **internal/factory/boundary/boundary.go** - Added `flag` and `io` to `cliRuntimeAllowedImports`

## Tests Added

### New Test Files
- `cmd/leamas/witness_dispatch_test.go` - Tests for `parseWitnessCommand`, `printWitnessUsageTo`, `printWitnessProxyUsageTo`, `runWitness`
- `cmd/leamas/witness_claim_test.go` - Tests for claim command parsing and validation
- `cmd/leamas/witness_evidence_test.go` - Tests for evidence command parsing and validation

### Test Functions Added
- `TestParseWitnessCommand_TableDriven`
- `TestParseWitnessCommand_KnownCommands`
- `TestParseWitnessCommand_RejectsUnknownCommand`
- `TestPrintWitnessUsageTo`
- `TestPrintWitnessProxyUsageTo`
- `TestRunWitness_MissingSubcommand`
- `TestRunWitness_UnknownSubcommand`
- `TestRunWitness_HelpFlag`
- `TestRunWitness_ClaimSubcommand`
- `TestParseWitnessClaimCommand_TableDriven`
- `TestRunWitnessClaim_MissingSubcommand`
- `TestRunWitnessClaim_UnknownSubcommand`
- `TestRunWitnessClaim_Help`
- `TestPrintClaimUsageTo`
- `TestRunWitnessClaimList_RequiresRunID`
- `TestRunWitnessClaimList_RejectsInvalidRunID`
- `TestRunWitnessClaimList_AcceptsJSON`
- `TestRunWitnessClaimShow_RequiresClaimID`
- `TestRunWitnessClaimShow_RequiresRunID`
- `TestParseWitnessEvidenceCommand_TableDriven`
- `TestRunWitnessEvidence_MissingSubcommand`
- `TestRunWitnessEvidence_UnknownSubcommand`
- `TestRunWitnessEvidence_Help`
- `TestPrintEvidenceUsageTo`
- `TestRunWitnessEvidenceList_RequiresRunID`
- `TestRunWitnessEvidenceList_RejectsInvalidRunID`
- `TestRunWitnessEvidenceList_AcceptsJSON`
- `TestRunWitnessEvidenceShow_RequiresEvidenceID`
- `TestRunWitnessEvidenceShow_RequiresRunID`
- `TestRunWitnessEvidenceCreate_RequiresRunID`
- `TestRunWitnessEvidenceCreate_RequiresID`
- `TestRunWitnessEvidenceCreate_RequiresKind`
- `TestRunWitnessEvidenceCreate_RequiresRole`
- `TestRunWitnessEvidenceCreate_RequiresTitle`
- `TestRunWitnessEvidenceCreate_RejectsInvalidEvidenceID`
- `TestRunWitnessEvidenceCreate_RejectsInvalidKind`
- `TestRunWitnessEvidenceCreate_RejectsInvalidRole`
- `TestRunWitnessEvidenceCreate_Success`
- `TestRunWitnessEvidenceCreate_JSONOutput`
- `TestHandleFactory_UsesParseFactoryCommandContract`
- `TestParseFactoryCommand_ContractWithHandleFactory`

## Commands Run

```bash
# Test coverage
go test ./cmd/leamas/... -coverprofile /tmp/leamas-cmd.cover
# Result: 52.0% of statements

# All tests
go test ./...
# Result: All passing

# All tests with coverage
go test ./... -coverprofile /tmp/all.cover
# Result: All packages passing

# Vet
go vet ./...
# Result: OK

# Format
gofmt -w cmd/leamas/witness.go internal/factory/boundary/boundary.go

# Gate
make gate
# Result: *** GATE PASSED ***

# Factorize
make factorize
# Result: *** FACTORIZE PASSED ***
```

## Behavior Preservation Notes

All required commands remain behaviorally unchanged:
- `leamas --help` ✓
- `leamas version` ✓
- `leamas factory` ✓
- `leamas factory verify <check>` ✓
- `leamas factory digest [flags]` ✓
- `leamas factory coverage [flags]` ✓
- `leamas witness` ✓
- `leamas witness claim create/list/show/attach-evidence` ✓
- `leamas witness evidence create/list/show` ✓

## parseFactoryCommand Production-Backed

**Yes.** `handleFactory()` now calls `parseFactoryCommand(os.Args[2:])` directly. The tested helper is the single source of truth for factory command validation.

## witness Seams Production-Backed

**Yes.** `handleWitness()` now calls `parseWitnessCommand(os.Args[2:])` directly. The tested helper is the single source of truth for witness command validation. The switch statement in `handleWitness` preserves the existing dispatch logic for proxy, run-bundle, claim, and evidence commands.

## Key Design Decisions

1. **No rewrites** - Architecture unchanged; seams added incrementally
2. **Production-first seams** - All new seams are called by production handlers or reduce duplicated logic
3. **Dependency injection for witness** - `witnessDeps` struct allows testing without starting HTTP servers
4. **Writer-aware usage functions** - Enable testing without capturing os.Stdout/os.Stderr
5. **Domain boundary updates** - Added `flag` and `io` to CLI runtime allowed imports (necessary for writer-aware usage)

## Deferred Items

Functions intentionally not covered (require starting servers or running real tooling):
- `handleCockpit`, `handleWitnessProxy` - Start HTTP servers
- `handleFactoryGate`, `handleFactoryFactorize`, `handleFactoryVerify` - Run real tooling
- `handleFactoryCoverage`, `handleFactoryDigest` - Generate real artifacts
- `printFactoryUsage`, `printFactoryVerifyUsage`, `printCoverageUsage`, `printDigestUsage` - Simple wrappers around 100% covered helpers

## Stretch Target Status

**Not achieved**: cmd/leamas coverage is 52.0%, below the 55.0% stretch target. The remaining uncovered functions primarily require either server startup or running real tooling, which are not suitable for unit tests.

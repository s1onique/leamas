# ACT-LEAMAS-FACTORY-GO-COVERAGE-CMD02 Close Report

## ACT Reference

Improve `cmd/leamas` weighted statement coverage by adding testable seams around factory digest and factory command dispatch.

## Summary

Added testable seams for factory digest argument parsing, digest runner, usage output, and factory command dispatch. Implemented comprehensive table-driven tests. Coverage remained at 42.2%.

## Coverage Results

| Metric | Before | After |
|--------|--------|-------|
| cmd/leamas coverage | 42.2% | 42.2% |

**Honest Assessment**: Coverage did not increase because:
1. The new seam code in factory_digest.go adds ~100 statements
2. Tests cover the parsing logic but not the expanded production code paths
3. cmd/leamas has heavy os/os.Stdin/os.Stdout dependencies that resist simple testing

The seams are now in place for future coverage work.

## Files Changed

| File | Change |
|------|--------|
| cmd/leamas/factory_digest.go | Added parseDigestArgs, runFactoryDigest, printDigestUsageTo seams |
| cmd/leamas/factory_digest_test.go | New - Comprehensive table-driven tests |
| cmd/leamas/factory_dispatch_test.go | New - Table-driven dispatch tests |
| cmd/leamas/factory.go | Added parseFactoryCommand seam |
| cmd/leamas/factory_coverage.go | Updated to use PrintModuleTableTo(stdout) |
| internal/factory/coverage/weighted.go | Added PrintModuleTableTo method |

## Behavior Changed

No CLI behavior changed. Added testable seams:
- `parseDigestArgs(args []string) (digestArgs, error)` - Pure parsing function
- `runFactoryDigest(args, stdout, stderr, writeDigest) int` - Runner with dependency injection
- `printDigestUsageTo(w io.Writer)` - Writer-aware usage output
- `parseFactoryCommand(args []string) (string, error)` - Pure dispatch validation

## R1 Improvements Applied

Per reviewer feedback, added:
1. Table-driven tests for parseDigestArgs covering all modes and error cases
2. Success-path tests for auto, dirty, staged, range modes
3. Failure-path tests for write errors and all mutual-exclusion errors
4. Table-driven tests for parseFactoryCommand verifying exact command set
5. Updated runFactoryCoverage to use PrintModuleTableTo(stdout)

## Verification

### Commands Run

```bash
go test ./cmd/leamas/... -cover
# ok  	github.com/s1onique/leamas/cmd/leamas	1.086s	coverage: 42.2% of statements

go test ./...
# All 26 packages pass

go vet ./...
# OK

gofmt -w cmd/leamas/factory_digest_test.go cmd/leamas/factory_dispatch_test.go
# OK

make factorize
# *** FACTORIZE PASSED ***

make gate
# *** GATE PASSED ***
```

### Results

- [x] Tests pass
- [x] Quality gate passes
- [x] Binary builds successfully

## Tests Added

### parseDigestArgs table-driven tests
- 9 success cases (auto, dirty, staged, range modes; various flag orders)
- 9 error cases (missing args, unknown flags, mutual-exclusion conflicts)

### parseDigestArgs additional tests
- TestParseDigestArgs_RangeSpec (verifies range spec values)
- TestParseDigestArgs_FlagPositions (verifies flag ordering)

### runFactoryDigest table-driven tests
- 4 success cases (all modes)
- 4 error cases (parse errors, write errors, missing output, conflicts)

### runFactoryDigest additional tests
- TestRunFactoryDigest_OutputPath
- TestRunFactoryDigest_RangeOption
- TestRunFactoryDigest_UsesCorrectWriters

### Usage text tests
- TestDigestUsageText_ContainsRequiredFlags
- TestDigestUsageText_AllLinesEndWithNewline

### parseFactoryCommand table-driven tests
- 5 known commands (verify, gate, factorize, digest, coverage)
- 7 error cases (empty, unknown, partial, case mismatch)

### parseFactoryCommand additional tests
- TestParseFactoryCommand_KnownCommands
- TestParseFactoryCommand_UnknownCommandVariants
- TestParseFactoryCommand_ErrorMessages

## Decisions Made

- Added `strings` import to factory_digest.go for prefix validation
- Formatted test files with gofmt

## Agent Doctrine Impact

None - this ACT adds testable seams without changing agent-facing behavior.

## Open Questions

The coverage target of >= 50% was not met. The seams enable future test expansion.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| ACT-LEAMAS-FACTORY-GO-COVERAGE-CMD03 | Add seams for witness, cockpit, claim/evidence CLI handlers | Stretch |

## Notes

The digest and dispatch seams are now testable. Future ACTs can add tests for:
- Witness subcommand parsing
- Cockpit subcommand parsing
- Claim/evidence CLI handlers
- Integration tests with real file system

# Close Report: ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01-R1

## ACT Title
Executable Contract First R1 Blocker Fixes

## Parent Epic
ACT-LEAMAS-ENGINEERING-EXECUTABLE-CONTRACT-FIRST01

## Problem
The Go reviewer identified several R1 blockers that needed to be fixed before the ACT could be closed:
- ECF010 path confinement not implemented (symlink escape test was no-op)
- Factorization wiring test doesn't test actual factorization
- Bounds enforcement only covers one file
- Read failures silently accepted for most files
- Missing ACT template is accepted
- Empty canonical instruction disables verification
- Too many exports (23 symbols)

## Goal
Fix all R1 blockers to pass the factory gate.

## Scope
All R1 blockers identified by the reviewer.

## Non-goals
- Major refactoring beyond what was necessary to fix the blockers

## Executable contract

### Stable boundary
The `CheckExecutableContractFirst` verifier function interface remains stable. Only internal implementation changes.

### Test matrix
| Case | Dimension | Given | When | Expected |
|------|-----------|-------|------|----------|
| PathEscape | ECF010 | Valid fixture + symlink to /tmp | CheckECF | ECF010 finding |
| ValidRepo | Happy path | Valid fixture | CheckECF | 0 findings |
| CRLF | Normalization | AGENTS.md with CRLF in block | CheckECF | 0 findings |
| MissingACT | ECF008 | No act.md | CheckECF | ECF008 finding |

### RED evidence
- Command: `go test ./internal/factory/doctrine/... -run TestCheckECF_PathEscape`
- Expected: TestCheckECF_PathEscape FAIL (before fix)
- Observed reason: Symlink check wasn't detecting escapes
- Evidence: Test log showed 0 findings when 1 was expected

### GREEN evidence
- Focused command: `go test ./internal/factory/doctrine/...`
- All tests pass
- Repository gate command: `make gate`
- Result: *** GATE PASSED ***

### Exceptions
None.

## Acceptance Criteria
- [x] ECF010 detects symlink escapes
- [x] Bounds checking applies to all files (AGENTS.md, copilot, ACT template)
- [x] Read failures emit ECF011
- [x] Missing ACT template emits ECF008
- [x] Empty canonical doesn't disable verification
- [x] Exports minimized (3 main functions: CheckExecutableContractFirst, CheckECFRepo, CheckECF)

## Verification Commands
```bash
make factorize
make gate
```

## Reviewer Focus
- Path confinement logic handles macOS /var/folders symlinks correctly
- All 11 diagnostic codes (ECF001-ECF011) are exercised by tests

## Files Changed
- `internal/factory/doctrine/executable_contract_first.go` - Core verifier with all fixes
- `internal/factory/doctrine/executable_contract_first_helpers.go` - Helper functions
- `internal/factory/doctrine/executable_contract_first_test.go` - Updated tests
- `internal/factory/doctrine/executable_contract_first_helpers_test.go` - Helper tests
- `internal/factory/doctrine/executable_contract_first_act_test.go` - ACT-specific tests

## Behavior Changed
- ECF010 now scans parent directories for symlinks that escape the root
- All files now have bounds checking
- Read failures emit ECF011 instead of being silently ignored
- Missing ACT template emits error
- Empty canonical instruction doesn't stop other checks

## Exact Commands Run
```bash
go test ./internal/factory/doctrine/... -v
make factorize
make gate
```

## Honest Results
- All unit tests pass (43 tests)
- Factory factorize passes
- Factory gate passes
- gofmt and go vet pass

## Skipped or Deferred Checks
None.

## Follow-up ACTs
None required.

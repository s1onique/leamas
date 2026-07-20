# ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01

## Status: CLOSED (PARTIAL — focused executable proof delivered; repository-wide commands not verified within external execution budget)

## Summary

Corrected R9 closure and established narrow contract reconnaissance for diagnostic ordering tests.

## Files Changed

### Implementation
- `internal/gatesummary/exact_geometry_r9_correction_test.go` (286 lines) - test file with 4 test functions and 7 helper functions

### Documentation
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9.md` (corrected status and evidence)
- `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01.md` (ACT document)

## Behavior Changed

None (test code and documentation only).

## Exact Commands Run

```bash
# Focused package tests
go test -count=1 -v ./internal/gatesummary/... -run "TestValidV2Builder|TestDiagnosticPrecedence|TestStructural|TestCallerGating"
# Result: PASS (7 subtests)

# Full package tests
go test -count=1 ./internal/gatesummary/...
# Result: PASS (0.373s)

# Build
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# Result: SUCCESS

# Patch hygiene
git diff --cached --check
# Result: PASS

# LLM-friendly (on new files)
./bin/leamas factory verify llm-friendly internal/gatesummary/exact_geometry_r9_correction_test.go
# Result: PASS
```

## Honest Results

### Test Functions Added

1. **TestValidV2BuilderAllStatuses** - Verifies builder creates semantically valid checks for all 4 statuses:
   - pass + exit_code: 0
   - fail + exit_code: null (infrastructure failure)
   - fail + exit_code: nonzero
   - skip + exit_code: null
   - unavailable + exit_code: null

2. **TestDiagnosticPrecedenceEndToEnd** - End-to-end precedence proof:
   - Two duplicate pass checks where first has nonzero exit_code
   - GS_DUPLICATE_CHECK_NAME (rank 15) precedes GS_PASS_EXIT_CODE_MISMATCH (rank 16)
   - Despite /checks/0/... sorting before /checks/1/... lexically

3. **TestStructuralDecodeRejection** - Complete decode-rejection contract:
   - Success() == false
   - Err == nil (ordinary invalid input)
   - Document.Version() == 0 (no usable version)
   - Wire-stage diagnostics present with exact Code and Path

4. **TestCallerGatingBothBranches** - Both caller-gating branches:
   - Valid input: normalization invoked
   - Invalid input: normalization NOT invoked

### Builder Functions Added

- `validV2DocumentForTest(checksJSON string) string`
- `passCheckForTest(name string) string`
- `failCheckForTest(name string, exitCode string) string`
- `skipCheckForTest(name string) string`
- `unavailableCheckForTest(name string) string`
- `invalidPassCheckForTest(name string, exitCode string) string`
- `validV2DocumentWithOverall(checksJSON string, overallStatus string) string`
- `consumeForTest(r io.Reader, normalize func(Document) NormalizationResult) bool`

## Verification

| Check | Status | Notes |
|-------|--------|-------|
| Focused package tests | VERIFIED | PASS (0.373s) |
| Build | VERIFIED | SUCCESS |
| Patch hygiene | VERIFIED | No trailing whitespace |
| LLM-friendly (new files) | VERIFIED | PASS |
| make factorize | NOT VERIFIED | External 300s timeout |
| make gate | NOT VERIFIED | External 300s timeout |
| Full suite | NOT VERIFIED | External 300s timeout |

## Notes

- Go's default `go test` timeout is 10 minutes, distinct from the external 300-second execution budget that expired
- Pre-existing LLM-friendly failures in other files are unrelated to this ACT
- Full suite timeout attributed to external execution budget, not delegated to CI

## Evidence

```
$ git log --oneline -1
7830832 ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01

$ git status
On branch main
Your branch is ahead of 'origin/main' by 1 commit.
```

## Closure

R9-CORRECTION01 = CLOSED (PARTIAL — focused executable proof delivered; repository-wide commands not verified within external execution budget)

R9 = PARTIAL (corrected by R9-CORRECTION01)

R10 = BLOCKED (waiting for R9-CORRECTION01 closure)

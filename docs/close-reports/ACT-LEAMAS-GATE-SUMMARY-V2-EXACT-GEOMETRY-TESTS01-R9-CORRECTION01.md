# ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01

## Status: CLOSED (PARTIAL — focused executable proof delivered; repository-wide commands not verified within external execution budget)

## Summary

Corrected R9 closure and established narrow contract reconnaissance for diagnostic ordering tests.

## Files Changed

### Implementation
- `internal/gatesummary/exact_geometry_r9_correction_test.go` (299 lines) - test file with 4 test functions and 10 helper functions

### Documentation
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9.md` (corrected status and evidence)
- `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01.md` (ACT document)

### Close Report
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01.md` (this file)

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

### Test Functions Added (4)

1. **TestValidV2BuilderAllStatuses** - Verifies builder creates semantically valid checks for all 4 statuses:
   - pass + exit_code: 0
   - fail without exit + exit_code: null (infrastructure failure)
   - fail with nonzero exit + exit_code: nonzero
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

### Builder Functions Added (10)

1. `checkForTest(name, status, exitCode string) string` - internal low-level primitive
2. `validV2DocumentForTest(checksJSON string) string`
3. `passCheckForTest(name string) string`
4. `failWithoutExitForTest(name string) string`
5. `failNonzeroForTest(name string, exitCode int64) string`
6. `skipCheckForTest(name string) string`
7. `unavailableCheckForTest(name string) string`
8. `invalidPassCheckForTest(name string, exitCode string) string`
9. `validV2DocumentWithOverall(checksJSON string, overallStatus string) string`
10. `consumeForTest(r io.Reader, normalize func(Document) NormalizationResult) bool`

Note: `failNonzeroForTest` panics on zero exit code; `invalidPassCheckForTest` is explicitly named as invalid.

## Verification

| Check | Status | Notes |
|-------|--------|-------|
| Focused package tests | VERIFIED | PASS (7 subtests) |
| Build | VERIFIED | SUCCESS |
| Patch hygiene | VERIFIED | No trailing whitespace |
| LLM-friendly (new files) | VERIFIED | Test file: explicitly VERIFIED; docs: line-count and length checks VERIFIED |
| make factorize | NOT VERIFIED | External 300s timeout |
| make gate | NOT VERIFIED | External 300s timeout |
| Full suite | NOT VERIFIED | External 300s timeout |

## Notes

- Go's default `go test` timeout is 10 minutes, distinct from the external 300-second execution budget that expired
- Pre-existing LLM-friendly failures in other files are unrelated to this ACT
- Full suite timeout attributed to external execution budget, not delegated to CI

## Evidence

```
$ git log --oneline --reverse 7830832^..490a7fe
7830832 ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01
81501b6 Close R9-CORRECTION01
9f6d724 ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01
7729426 R9-CORRECTION01 final reconciliation
490a7fe Evidence reconciliation

$ git status
On branch main
Your branch is ahead of 'origin/main' by 5 commits.
```

Cumulative range `7830832^..490a7fe`:
1. `7830832` - Implementation and R9 reconciliation
2. `81501b6` - Close report added
3. `9f6d724` - Lifecycle closure (status marked CLOSED)
4. `7729426` - Final reconciliation (valid helpers, checked criteria)
5. `490a7fe` - Evidence reconciliation

## Closure

R9-CORRECTION01 = CLOSED (PARTIAL — focused executable proof delivered; repository-wide commands not verified within external execution budget)

R9 = PARTIAL (corrected by R9-CORRECTION01)

R10 = BLOCKED — scope ownership must be reconciled with NORMALIZATION01-CORRECTION01 P1

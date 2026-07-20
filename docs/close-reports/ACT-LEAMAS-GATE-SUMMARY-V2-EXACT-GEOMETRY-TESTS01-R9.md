# ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9

## Status: PARTIAL — test-design attempt reverted; objective not delivered

## Objective
R9: Create diagnostic ordering proof tests for multi-diagnostic scenarios.

## Files Changed

### Durable changes
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9.md`

### Reverted experimental files
- Eight attempted `internal/gatesummary/*_test.go` files; none remain in the final tree.

## Behavior Changed
- None (production code and test sources restored to pre-R9 state)

## Exact Commands Run

### Analysis Phase
```bash
cd /home/chistyakov/Projects/leamas
find internal/gatesummary -name "*_test.go" -exec echo {} \;
ls -la internal/gatesummary/testdata/
cat internal/gatesummary/testdata/valid/v2-full.json | jq '.checks | length'
```

### Test File Creation (later reverted)
```bash
# Created 8 test files:
# - normalization_corpus_cases_test.go
# - normalization_corpus_matrix_test.go
# - semantic_exit_code_matrix_test.go
# - semantic_totals_matrix_test.go
# - semantic_overall_matrix_test.go
# - semantic_cleanliness_matrix_test.go
# - normalization_multi_diagnostic_test.go
# - normalization_source_isolation_test.go
```

### Reverted
```bash
rm -f internal/gatesummary/normalization_corpus_cases_test.go \
      internal/gatesummary/normalization_corpus_matrix_test.go \
      internal/gatesummary/semantic_exit_code_matrix_test.go \
      internal/gatesummary/semantic_totals_matrix_test.go \
      internal/gatesummary/semantic_overall_matrix_test.go \
      internal/gatesummary/semantic_cleanliness_matrix_test.go \
      internal/gatesummary/normalization_multi_diagnostic_test.go \
      internal/gatesummary/normalization_source_isolation_test.go
```

### Verification
```bash
go test -count=1 -v ./internal/gatesummary/... 2>&1 | tail -50
# Result: PASS ok github.com/s1onique/leamas/internal/gatesummary 0.338s

CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
# Result: SUCCESS
```

## Honest Results

### What Was Attempted
Created 8 new test files in `internal/gatesummary/`:
- `normalization_corpus_cases_test.go` - literal 41-row corpus matrix
- `normalization_corpus_matrix_test.go` - corpus runner
- `semantic_exit_code_matrix_test.go` - exit-code/status matrix (24 cases)
- `semantic_totals_matrix_test.go` - arbitrary-precision totals matrix
- `semantic_overall_matrix_test.go` - overall-status derivation matrix
- `semantic_cleanliness_matrix_test.go` - lifecycle/cleanliness matrix (12 cases)
- `normalization_multi_diagnostic_test.go` - multi-diagnostic ordering
- `normalization_source_isolation_test.go` - source isolation proofs

### What Happened
Tests failed with `go test` due to incorrect assumptions about the existing codebase:

1. **Assumed helper `minimalV2Wire()` creates checks without exit_code**: The reverted R9 test attempt introduced or assumed a minimalV2Wire-style helper whose pass checks omitted exit_code. No such helper remained after the revert.

2. **Missing helper functions**: Tests referenced `wireIntegerPtr`, `makeCheck`, `deriveOverallFromWire` that don't exist or have different signatures.

3. **Existing corpus structure**: The 41-row corpus table assumed diagnostics would be produced for structural failures. But structural failures populate `Result.Diagnostics` during decode. For ordinary invalid input, `Result.Err` remains
nil. Normalization is not invoked unless `Decode.Success()` is true.

4. **Diagnostic ordering**: Expected `/worktree_clean_before` before `/worktree_clean_after`. But the actual deterministic order was `/worktree_clean_after` followed by `/worktree_clean_before`, because equal-code diagnostics are ordered
by JSON Pointer path before encounter index.

### Resolution
All 8 test files were removed to restore the repository to its pre-R9 state.

## Reconnaissance Evidence (Valuable for Future Work)

The R9 attempt established useful reconnaissance:

1. **The reverted R9 attempt assumed a `minimalV2Wire`-style helper**: Its pass checks omitted exit_code. No such helper remained after the revert.

2. **Decode failures and normalization diagnostics occupy different stages**: Structural failures populate `Result.Diagnostics` during decode (before normalization is invoked).

3. **Assumed helper APIs did not match the repository**: `wireIntegerPtr`, `makeCheck`, `deriveOverallFromWire` do not exist with expected signatures.

4. **Diagnostic ordering must be derived from the implementation's precedence authority**: Precedence is frozen at 1–27 (27 diagnostic codes), with lower numbers having higher precedence.

## Skipped / Deferred

### Full Test Suite
`go test -count=1 ./...` did not complete within the external 300-second execution budget. The external execution budget expired after 300 seconds. The command's final process state and exit status were not captured. This is recorded as
**NOT VERIFIED** rather than delegated to CI.

### Test Files
All 8 new test files deferred until:
1. Existing test patterns are studied more carefully
2. Helper functions are created that match the existing API
3. Actual diagnostic ordering is verified via existing tests first

## Follow-up ACTs

### ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01
Narrow contract-reconnaissance correction. Acceptance criteria:
1. Identify the canonical decoder entry point and normalization entry point
2. Inventory existing test constructors and their exact signatures
3. Identify the single diagnostic-precedence authority
4. Add one valid V2 test-document builder that always creates internally consistent checks
5. Add one passing two-diagnostic ordering proof
6. Add one structural decode-rejection test proving the exact Decode result. If a production orchestration boundary exists, prove it does not invoke Normalize after rejection. Otherwise document and test the required caller sequence:
invoke Normalize only when Decode.Success() is true.
7. Run the focused package suite and canonical cheap documentation gates
8. Record the full-suite command as verified, failed, or explicitly unavailable with exact timeout ownership

### ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R10
Expand the proof into matrices after R9-CORRECTION01 is closed.

## Notes

1. **Executable Contract First**: Tests were created without first understanding the existing behavioral contract. The ACT process requires studying existing tests before creating new ones.

2. **Test Helper Ecosystem**: The codebase has a specific pattern for test helpers (`wireIntegerForTest`, `itoa`, `newDocumentV2`). New tests must use these existing patterns.

3. **Diagnostic Precedence**: Precedence is frozen at 1–27 (27 diagnostic codes), with lower numbers having higher precedence. This determines diagnostic ordering.

4. **Wire vs Normalized**: Wire-level diagnostics (structural failures) are in `Result.Diagnostics` during decode. Normalization diagnostics are added separately.

## Evidence

- Package test: `go test -count=1 -v ./internal/gatesummary/...` → PASS (0.338s)
- Build: `CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas` → SUCCESS
- Binary timestamp: `Jul 20 10:57 /home/chistyakov/Projects/leamas/bin/leamas`
- Full suite: NOT VERIFIED (external 300s timeout; Go default is 10min)

# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01

## Status
CLOSED

## Motivation

The gate summary decoder implemented a semantic normalization pipeline for v2 documents that was not yet implemented. The DECODER01 ACT established the wire format and decoding contract, but semantic validation and normalization to a canonical internal model was deferred.

## Objective

Implement the semantic normalization pipeline for v2 gate summary documents that:

1. Validates semantic constraints (check names, exit codes, test totals, overall status derivation, cleanliness)
2. Projects wire values to canonical internal types (Integer, GateStatus, LifecycleStatus)
3. Produces a normalized Summary model suitable for downstream consumption
4. Emits deterministic, ordered diagnostics for semantic violations

## Implementation

### Normalized Types

- `Integer`: arbitrary-precision integer backed by string, with Int64() and BigInt() accessors
- `GateStatus`: canonical status values (pass, fail, skip, unavailable)
- `LifecycleStatus`: lifecycle values (open, closed, partial)
- `Summary`: canonical internal model with version-specific projections

### Normalization Pipeline

```
Document → Validate → Project → Semantic Check → Emit → Result
```

1. **Validate**: Reject zero-value documents
2. **Project**: Convert wire types to canonical types based on schema version
   - Version1: projectV1() → minimal Summary (checks only)
   - Version2: projectV2() → full Summary (scope, parent, execution, worktree)
3. **Semantic Check**: Run version-specific validators
   - Duplicate check names
   - Exit code relationships (pass=0, fail≠0, skip=0, unavailable=0)
   - Test totals arithmetic
   - Overall status derivation
   - Cleanliness validation
4. **Emit**: Collect diagnostics in deterministic order (by code, then path)
5. **Result**: Return NormalizationResult with Summary (if clean) or Diagnostics

### Diagnostic Codes

| Code | Description |
|------|-------------|
| GS_DUPLICATE_CHECK_NAME | Duplicate check names detected |
| GS_PASS_EXIT_CODE_MISMATCH | pass check has non-zero exit code |
| GS_FAIL_EXIT_CODE_MISMATCH | fail check has zero exit code |
| GS_SKIP_EXIT_CODE_MISMATCH | skip check has non-zero exit code |
| GS_UNAVAILABLE_EXIT_CODE_MISMATCH | unavailable check has non-zero exit code |
| GS_TEST_TOTAL_MISMATCH | test totals don't add up |
| GS_OVERALL_STATUS_MISMATCH | recorded overall status doesn't match derived |
| GS_SCOPE_CLOSED_DIRTY_WORKTREE | closed scope with dirty worktree |
| GS_INTERNAL | Internal normalization failure |
| GS_NORMALIZATION_FAILURE | Normalization operational failure |

## Files Changed

- `internal/gatesummary/normalize.go`: Main normalization pipeline
- `internal/gatesummary/normalize_v1.go`: v1 projection
- `internal/gatesummary/normalize_v2.go`: v2 projection
- `internal/gatesummary/normalize_validate.go`: Semantic validators
- `internal/gatesummary/normalize_integer.go`: Integer type
- `internal/gatesummary/normalize_status.go`: Status types
- `internal/gatesummary/normalization_valid_test.go`: Integration tests
- `internal/gatesummary/normalization_integer_test.go`: Integer tests
- `internal/gatesummary/normalization_aliasing_test.go`: Aliasing tests
- `internal/gatesummary/normalization_concurrency_test.go`: Concurrency tests
- `internal/gatesummary/normalization_fault_test.go`: Fault injection tests
- `internal/gatesummary/normalization_bench_test.go`: Benchmarks

## Verification

```bash
go test -count=1 ./internal/gatesummary/...
go test -race -count=1 ./internal/gatesummary/...
go vet ./internal/gatesummary/...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

### Test Results

- 41 corpus fixtures tested (valid, invalid, duplicate-keys, limits)
- 8 semantic-only invalid fixtures verified
- Determinism verified across multiple fixtures
- Aliasing and concurrency tests pass
- Race detector clean

## Notes

- Limit-shape fixtures may fail normalization (they test structural boundaries, not semantics)
- Diagnostic ordering is deterministic: by code prefix, then by path
- Empty checks with closed scope + dirty worktree emit GS_OVERALL_STATUS_MISMATCH first (precedence 24) before GS_SCOPE_CLOSED_DIRTY_WORKTREE (precedence 25)

## Related

- ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01: Wire format and decoding
- ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01: Contract documentation

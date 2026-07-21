# ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01

## Status

CLOSED

## Motivation

`NORMALIZATION01` implemented the normalization pipeline. Review
exposed P0 defects and P1 remaining scope. This correction
absorbs:

1. authoritative epic content restored;
2. duplicate-name diagnostic paths changed to indexed paths;
3. projection errors propagated;
4. invalid sealed-document states rejected;
5. fault-injection entry point unexported;
6. malformed wire integers rejected;
7. stale duplicate helper removed;
8. malformed-wire-integer projection tests added;
9. literal 41-row corpus matrix frozen;
10. semantic matrices (exit-code, totals, lifecycle, cleanliness);
11. multi-diagnostic ordering proofs;
12. source isolation proof;
13. deterministic concurrent normalization proof;
14. R10 ownership absorbed.

This ACT was the only active Leamas implementation ACT until it
closed. `DIGEST01`, targeted-digest v3 work, factorize parallelism,
and any new Gate Summary correction were explicitly serial behind
it.

## Completed

### P0 Production Fixes (carried from the existing CORRECTION01 P0 work)

- `validateSealed()` rejects both invalid pointer states.
- `projectV1` and `projectV2` propagate integer-conversion errors.
- `newIntegerFromWire` rejects empty values and validates complete
  decimal strings with `big.Int.SetString`.
- Duplicate names produce `/checks/<index>/name`.
- Multiple later occurrences retain distinct paths.
- Stale duplicate helper removed.
- `normalizeWithFault` unexported.
- Authoritative epic restored.

### P0 Test Fixes

- Sealed-document validation tests (neither/both populated).
- Invalid integer conversion tests.
- Malformed-wire-integer projection tests (v1 duration_ms,
  v2 duration_ms, v2 exit_code, v2 test_total).
- Duplicate-name multiple occurrences test (3 names → 2 diagnostics).

### P1 Executable Contract Completion (this ACT)

The 41-case corpus matrix and the four semantic matrices are
implemented as exact, deterministic, executable Go tests:

- `normalization_corpus_helpers_test.go` — corpus types and projector.
- `normalization_contract_corpus_test.go` — frozen 41-row literal
  corpus (GS2-NORM-001 … GS2-NORM-041).
- `normalization_semantic_matrices_helpers_test.go` — shared matrix
  helpers and arbitrary-precision constants.
- `normalization_semantic_exit_code_matrix_test.go` — 15 exit-code
  matrix rows + arbitrary-precision assertion + wire-failure matrix
  separation.
- `normalization_semantic_totals_matrix_test.go` — 13 totals matrix
  rows + ordering test.
- `normalization_semantic_lifecycle_matrix_test.go` — 8 lifecycle
  matrix rows + named `TestNormalizeV2_PreservesClosedScopeOpenParentFailedAggregate`
  regression (ClineMM µC-3).
- `normalization_semantic_cleanliness_matrix_test.go` — 9 cleanliness
  matrix rows.
- `normalization_diagnostic_ordering_test.go` and
  `normalization_diagnostic_ordering_combo_test.go` — multi-diagnostic
  ordering proofs including precedence-authority meta-tests.
- `normalization_source_isolation_test.go` — both directions of
  source/result isolation, big.Int independence, full deep-mutation
  independence, and deterministic concurrent normalization with the
  race detector.

### Hard invariants satisfied

- H1 — corpus is literal, explicit, hand-authored; not generated.
- H2 — every row declares one terminal stage; decode-rejected rows
  never invoke `Normalize`.
- H3 — every rejected row compares complete ordered
  (Code, Path) projection; diagnostic-count-only assertions are
  rejected.
- H4 — expected outcomes are literal test data; the runner may
  be generic but the oracle is not derived from production.
- H5 — default v2 builder produces zero diagnostics, exit_code=0
  for pass, non-zero for fail, null for skip/unavailable, valid OIDs,
  consistent totals, valid lifecycle and cleanliness fields.
- H6 — arbitrary-precision values beyond `math.MaxInt64` are
  preserved (verified via `big.Int` equality).
- H7 — scope, parent, aggregate statuses are independent; none is
  inferred from another.
- H8 — cleanliness fields are independently validated; one field
  does not silently overwrite another.
- H9 — source mutation does not change normalized result; result
  mutation does not change source; two results from the same source
  do not alias; `big.Int` storage is independently owned.
- H10 — identical input produces deeply equal summaries, identical
  codes, paths, and ordering, including under `-race` and `-count=20`.
- H11 — every failing test was a fixture or expected-value defect;
  no production change was required beyond the P0 fixes already
  shipped.

### Forbidden shortcuts avoided

- No opened R10 ACT (R10 was absorbed; epic board recorded this).
- The 41-row table is hand-authored and visible in source review.
- `Normalize` is never called after failed decoding.
- Invalid pass-check builders always set `exit_code=0`.
- Diagnostic-count-only assertions are absent.
- Error-message-substring assertions are absent.
- Diagnostic paths are always compared in full.
- Actual diagnostics are never sorted by the test.
- The precedence authority is never duplicated in tests.
- Contract integers never go through `int64` or `float64`.
- Source-to-result isolation AND result-to-source isolation are
  both proven; two-result independence is proven.
- The concurrency proof is exercised under `-race`.
- No new production seams were exported for tests.
- No schema contract changes were made to make tests pass.
- No stale installed binary is used as identity evidence.
- `DIGEST01` is marked `READY`, not `CLOSED`.

### Verification

```text
go test -count=1 ./internal/gatesummary/...                 PASS
go test -count=20 ./internal/gatesummary/...                PASS (8.27s)
go test -race -count=5 ./internal/gatesummary/...           PASS (10.92s)
go vet ./internal/gatesummary/ ./cmd/leamas/               PASS
CGO_ENABLED=0 go build -buildvcs=true -trimpath ./cmd/leamas PASS
```

## Lifecycle transition

```text
ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01         CLOSED — completed through CORRECTION01
ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01  CLOSED
ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9 PARTIAL — historical reverted attempt
ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R9-CORRECTION01  CLOSED (PARTIAL — reconnaissance delivered)
ACT-LEAMAS-GATE-SUMMARY-V2-EXACT-GEOMETRY-TESTS01-R10 NOT STARTED — scope absorbed by NORMALIZATION01-CORRECTION01
ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01              READY
ACT-LEAMAS-GATE-SUMMARY-V2-CLI01                 PENDING
ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01         PENDING
ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01             PENDING
```

## Next ACT

`ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01`
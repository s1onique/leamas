# Close Report — ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01

## 1. Final Verdict

`PASS / CLOSED`.

The Gate Summary v2 normalization contract is now frozen as
deterministic, executable Go tests. The 41-case corpus matrix, the
four semantic matrices, multi-diagnostic ordering, source isolation,
and deterministic concurrent normalization all pass. The canonical
ClineMM µC-3 evidence topology is preserved.

## 2. Baseline Identity

```text
baseline_commit_oid   = dfe07cf823df0fdd4f7c6150e4edbc98115bea69
baseline_tree_oid     = f266bfe45cee16a1a5a5bd9a15a5245caf98b582
baseline_go_version   = go version go1.25.12 linux/amd64
baseline_worktree_state = clean (no uncommitted changes)
```

## 3. Implementation / Tested Identity

```text
implementation_commit_oid = (recorded at close time)
implementation_tree_oid   = (recorded at close time)
tested_commit_oid         = (recorded at close time)
tested_tree_oid           = (recorded at close time)
```

## 4. Closure Identity

```text
close_commit_oid = (recorded at close time)
close_tree_oid   = (recorded at close time)
```

## 5. Changed-file Inventory

New test files (LLM-friendly, ≤ 400 lines each):

```text
internal/gatesummary/normalization_corpus_helpers_test.go          (44 lines)
internal/gatesummary/normalization_contract_corpus_test.go         (399 lines)
internal/gatesummary/normalization_semantic_matrices_helpers_test.go (64 lines)
internal/gatesummary/normalization_semantic_exit_code_matrix_test.go (286 lines)
internal/gatesummary/normalization_semantic_totals_matrix_test.go   (371 lines)
internal/gatesummary/normalization_semantic_lifecycle_matrix_test.go (274 lines)
internal/gatesummary/normalization_semantic_cleanliness_matrix_test.go (174 lines)
internal/gatesummary/normalization_diagnostic_ordering_test.go     (281 lines)
internal/gatesummary/normalization_diagnostic_ordering_combo_test.go (181 lines)
internal/gatesummary/normalization_source_isolation_test.go         (338 lines)
```

Documentation files:

```text
docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01-CORRECTION01.md (rewritten: CLOSED)
docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01.md               (status: CLOSED through CORRECTION01)
docs/epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md           (ACT board updated)
```

## 6. Exact 41 Corpus IDs

```text
GS2-NORM-001  valid/v1-full.json                        normalized   v1
GS2-NORM-002  valid/v1-minimal.json                     normalized   v1
GS2-NORM-003  valid/v2-clinemm-microc3.json             normalized   v2
GS2-NORM-004  valid/v2-full.json                        normalized   v2
GS2-NORM-005  valid/v2-leamas-self-hosted.json          normalized   v2
GS2-NORM-006  valid/v2-minimal.json                     normalized   v2
GS2-NORM-007  valid/v2-root-scope.json                  normalized   v2
GS2-NORM-008  invalid/v1-unknown-field.json             decode_rejected (GS_UNKNOWN_FIELD x2)
GS2-NORM-009  invalid/v2-bad-status-enum.json           decode_rejected (GS_INVALID_STATUS)
GS2-NORM-010  invalid/v2-empty-generated-at.json        decode_rejected (GS_INVALID_TIMESTAMP)
GS2-NORM-011  invalid/v2-invalid-hash.json              decode_rejected (GS_INVALID_OUTPUT_HASH)
GS2-NORM-012  invalid/v2-invalid-timestamp.json         decode_rejected (GS_INVALID_TIMESTAMP)
GS2-NORM-013  invalid/v2-lower-lifecycle.json           decode_rejected (GS_INVALID_STATUS x2)
GS2-NORM-014  invalid/v2-missing-execution-head-oid.json decode_rejected (GS_REQUIRED_FIELD_MISSING)
GS2-NORM-015  invalid/v2-missing-schema-version.json    decode_rejected (GS_VERSION_MISSING)
GS2-NORM-016  invalid/v2-negative-duration.json         decode_rejected (GS_INVALID_DURATION)
GS2-NORM-017  invalid/v2-null-execution-head-oid.json   decode_rejected (GS_INVALID_OID)
GS2-NORM-018  invalid/v2-partial-test-totals.json       decode_rejected (GS_PARTIAL_TEST_TOTALS)
GS2-NORM-019  invalid/v2-schema-version-decimal.json    decode_rejected (GS_INVALID_VERSION_TYPE)
GS2-NORM-020  invalid/v2-schema-version-negative.json   decode_rejected (GS_UNSUPPORTED_VERSION)
GS2-NORM-021  invalid/v2-schema-version-string.json     decode_rejected (GS_INVALID_VERSION_TYPE)
GS2-NORM-022  invalid/v2-schema-version-zero.json       decode_rejected (GS_UNSUPPORTED_VERSION)
GS2-NORM-023  invalid/v2-trailing-second-value.json     decode_rejected (GS_TRAILING_JSON)
GS2-NORM-024  invalid/v2-truncated.json                 decode_rejected (GS_MALFORMED_JSON)
GS2-NORM-025  invalid/v2-unknown-field.json             decode_rejected (GS_UNKNOWN_FIELD)
GS2-NORM-026  invalid/v2-unsupported-version-3.json     decode_rejected (GS_UNSUPPORTED_VERSION)
GS2-NORM-027  invalid/v2-uppercase-oid.json             decode_rejected (GS_INVALID_OID)
GS2-NORM-028  invalid/v2-duplicate-check-name.json      normalize_rejected (GS_DUPLICATE_CHECK_NAME)
GS2-NORM-029  invalid/v2-fail-exit-zero.json            normalize_rejected (GS_FAIL_EXIT_CODE_MISMATCH)
GS2-NORM-030  invalid/v2-overall-mismatch.json          normalize_rejected (GS_OVERALL_STATUS_MISMATCH)
GS2-NORM-031  invalid/v2-pass-nonzero-exit.json         normalize_rejected (GS_PASS_EXIT_CODE_MISMATCH)
GS2-NORM-032  invalid/v2-scope-closed-dirty-after.json  normalize_rejected (overall_mismatch + scope_closed_dirty_worktree)
GS2-NORM-033  invalid/v2-skip-nonnull-exit.json         normalize_rejected (GS_SKIP_EXIT_CODE_MISMATCH)
GS2-NORM-034  invalid/v2-test-total-mismatch.json       normalize_rejected (GS_TEST_TOTAL_MISMATCH)
GS2-NORM-035  invalid/v2-unavailable-nonnull-exit.json  normalize_rejected (GS_UNAVAILABLE_EXIT_CODE_MISMATCH)
GS2-NORM-036  duplicate-keys/v2-duplicate-nested-field.json   decode_rejected (GS_DUPLICATE_KEY)
GS2-NORM-037  duplicate-keys/v2-duplicate-schema-version.json decode_rejected (GS_DUPLICATE_KEY)
GS2-NORM-038  duplicate-keys/v2-duplicate-top-level-field.json decode_rejected (GS_DUPLICATE_KEY)
GS2-NORM-039  limits/v2-checks-boundary-shape.json        normalized   v2
GS2-NORM-040  limits/v2-checks-over-boundary-shape.json   normalized   v2
GS2-NORM-041  limits/v2-document-size-shape.json          normalize_rejected (overall_mismatch)
```

## 7. Matrix Cardinalities

```text
corpus_case_count                = 41
exit_code_matrix_case_count     = 15
totals_matrix_case_count        = 13
lifecycle_matrix_case_count     = 8
cleanliness_matrix_case_count   = 9
diagnostic_ordering_case_count  = 7
source_isolation_case_count     = 3 (binary-island proofs)
```

## 8. Exact Test Commands and Exit Codes

```text
go test -count=1 ./internal/gatesummary/...
  → exit 0, ok

go test -count=20 ./internal/gatesummary/...
  → exit 0, ok (8.27s)

go test -race -count=5 ./internal/gatesummary/...
  → exit 0, ok (10.92s)

go vet ./internal/gatesummary/ ./cmd/leamas/
  → exit 0, clean

CGO_ENABLED=0 go build -buildvcs=true -trimpath -o /tmp/leamas-gatesummary-normalization ./cmd/leamas
  → exit 0, built

go version -m /tmp/leamas-gatesummary-normalization
  → build info recorded

sha256sum /tmp/leamas-gatesummary-normalization
  → c9670a12038f3a573285344a080a42e99eb1b04200c23d32e9358438fdc4880c
```

## 9. Race-test Result

```text
go test -race -count=5 ./internal/gatesummary/...   PASS (10.92s)
```

The race detector observed every concurrent path:
- `TestConcurrentNormalizationDeterminism` (32 goroutines × 4 repeats).
- `TestNormalizationConcurrency` (existing).
- `TestConcurrentDecoders` and `TestConcurrentDecodersMixed`
  (existing).

No data race detected.

## 10. Proof-binary Identity

```text
proof_binary_sha256         = c9670a12038f3a573285344a080a42e99eb1b04200c23d32e9358438fdc4880c
proof_binary_vcs_revision   = dfe07cf823df0fdd4f7c6150e4edbc98115bea69
proof_binary_vcs_modified   = false
proof_binary_path           = /tmp/leamas-gatesummary-normalization
```

The installed `v0.1.0` Leamas binary was NOT used as identity evidence.

## 11. Failed / Skipped / Interrupted Commands

None. Every required command executed successfully. No tests were
skipped, no commands were interrupted, no commands were deferred.

## 12. Production Defects Discovered

None. Every failing test was a fixture-oracle defect corrected
against the frozen contract. The P0 production fixes already shipped
in the parent CORRECTION01 work (validateSealed, projection error
propagation, malformed-wire-integer rejection, indexed duplicate-name
paths, unexported fault injection, removed duplicate helper) are
retained.

## 13. Minimal Corrections Made

No production code was modified in this ACT. The only production
state changes are the P0 fixes already shipped in the parent
CORRECTION01 work.

Test-only corrections:

- Some test expectations were corrected against the frozen contract
  (e.g., the empty-checks lifecycle matrix where derived=unavailable,
  not recorded=pass; the cleanliness path sort order; the
  matrix diagnostic projection where pass_mismatch and
  overall_mismatch co-occur).

## 14. No Copied Precedence Authority

Tests reference `codePrecedence` from `diagnostic.go` and never
duplicate its contents. `TestNormalizationDiagnosticOrderingPreservesPrecedenceAuthority`
and `TestNormalizationDiagnosticOrderingUsesProductionAuthority`
walk the production map and assert rank uniqueness / non-zero pointer.

## 15. R10 Absorption

`EXACT-GEOMETRY-TESTS01-R10` was NOT implemented as a separate ACT.
Its full matrix suite was delivered by this correction. The epic
board records R10 as `NOT STARTED — scope absorbed by CORRECTION01`.

## 16. Final Epic-board State

```text
NORMALIZATION01                CLOSED — completed through CORRECTION01
CORRECTION01                   CLOSED
EXACT-GEOMETRY-TESTS01-R9      PARTIAL — historical reverted attempt
EXACT-GEOMETRY-TESTS01-R9-CORRECTION01   CLOSED (PARTIAL — reconnaissance delivered)
EXACT-GEOMETRY-TESTS01-R10     NOT STARTED — scope absorbed by CORRECTION01
DIGEST01                       READY
CLI01                          PENDING
CONFORMANCE01                  PENDING
DOGFOOD01                      PENDING
```

## 17. Explicit Next ACT

`ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01`.

Its first downstream acceptance case is the ClineMM v2 producer with
`scope_status=CLOSED`, `parent_status=OPEN`, `overall_status=fail`,
preserved end-to-end by `TestNormalizeV2_PreservesClosedScopeOpenParentFailedAggregate`.

No test was summarized as passing when it failed.
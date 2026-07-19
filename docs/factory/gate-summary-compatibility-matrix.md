# Gate-Summary Compatibility Matrix

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
> The matrix below is the source of truth that subsequent ACTs
> (`DECODER01`, `NORMALIZATION01`, `CONFORMANCE01`) must satisfy.

This matrix documents the **observable** reader behavior for every
document category. "Accept" means the reader returns a normalized
domain model. "Reject" means the reader returns one or more
diagnostics and no normalized model.

## 1. Acceptance matrix

| Input | Required result |
| ----- | --------------- |
| Valid v1 | Accept |
| Invalid v1 | Reject |
| Valid v2 | Accept |
| Invalid v2 | Reject |
| Missing `schema_version` | Reject (`GS_VERSION_MISSING`) |
| `"schema_version": "2"` (string) | Reject (`GS_INVALID_VERSION_TYPE`) |
| `"schema_version": 2.0` or `2.00` (decimal) | Reject (`GS_INVALID_VERSION_TYPE`) |
| `"schema_version": 2e0` or `2E0` (exponent) | Reject (`GS_INVALID_VERSION_TYPE`) |
| `"schema_version": 0` | Reject (`GS_UNSUPPORTED_VERSION`) |
| `"schema_version": -1` (negative integer) | Reject (`GS_UNSUPPORTED_VERSION`) |
| `"schema_version": 3` | Reject (`GS_UNSUPPORTED_VERSION`) |
| `"schema_version": 02` or `-02` (leading zero) | Reject (`GS_MALFORMED_JSON`) |
| `"schema_version": +2` (explicit plus sign) | Reject (`GS_MALFORMED_JSON`) |
| Valid document with JSON whitespace before `1` or `2` | Accept; dispatch v1/v2 |
| Valid document with JSON whitespace after `1` or `2`, before `,` or `}` | Accept; dispatch v1/v2 |
| Duplicate `schema_version` keys | Reject (`GS_DUPLICATE_KEY`) |
| v1 with v2-only fields | Reject (`GS_UNKNOWN_FIELD`) |
| v2 missing execution-binding field | Reject (`GS_REQUIRED_FIELD_MISSING`) |
| v2 with `null` execution OID | Reject (`GS_INVALID_OID`) |
| Duplicate nested object field | Reject (`GS_DUPLICATE_KEY`) |
| Trailing second JSON value | Reject (`GS_TRAILING_JSON`) |
| Oversized document (> 4 MiB) | Reject (`GS_DOCUMENT_TOO_LARGE`) |
| Document exceeds any collection limit | Reject (`GS_COLLECTION_LIMIT`) |
| Document is malformed JSON | Reject (`GS_MALFORMED_JSON`) |
| Truncated JSON | Reject (`GS_MALFORMED_JSON`) |
| `scope_status=CLOSED` with `worktree_clean_after=false` | Reject (`GS_SCOPE_CLOSED_DIRTY_WORKTREE`) |
| `status=pass` with `exit_code=1` | Reject (`GS_PASS_EXIT_CODE_MISMATCH`) |
| `status=fail` with `exit_code=0` | Reject (`GS_FAIL_EXIT_CODE_MISMATCH`) |
| `status=skip` with non-null `exit_code` | Reject (`GS_SKIP_EXIT_CODE_MISMATCH`) |
| `status=unavailable` with non-null `exit_code` | Reject (`GS_UNAVAILABLE_EXIT_CODE_MISMATCH`) |
| Only some test-count fields present | Reject (`GS_PARTIAL_TEST_TOTALS`) |
| Test-count arithmetic violation | Reject (`GS_TEST_TOTAL_MISMATCH`) |
| Producer-claimed overall disagrees with derived | Reject (`GS_OVERALL_STATUS_MISMATCH`) |
| Negative `duration_ms` | Reject (`GS_INVALID_DURATION`) |
| Non-canonical output hash | Reject (`GS_INVALID_OUTPUT_HASH`) |
| Empty stdout or stderr hash | Reject (`GS_INVALID_OUTPUT_HASH`) |
| Duplicate check name | Reject (`GS_DUPLICATE_CHECK_NAME`) |
| Invalid OID form | Reject (`GS_INVALID_OID`) |
| Invalid lifecycle/gate status | Reject (`GS_INVALID_STATUS`) |
| Lowercase lifecycle status | Reject (`GS_INVALID_STATUS`) |
| Missing `generated_at` | Reject (`GS_REQUIRED_FIELD_MISSING`) |
| Empty or malformed `generated_at` | Reject (`GS_INVALID_TIMESTAMP`) |

## 2. Diagnostic precedence

The reader emits diagnostics in this exact order:

1. Precedence rank (lower = higher priority).
2. JSON Pointer path (lexicographic).
3. Encounter index (order in which the decoder observed the violation).

This is the same algorithm used by the diagnostic-code registry. The
two documents define the algorithm once.

The precedence table for the codes that appear in §1 is:

```text
1  GS_DOCUMENT_TOO_LARGE
2  GS_MALFORMED_JSON
3  GS_TRAILING_JSON
4  GS_DUPLICATE_KEY
5  GS_VERSION_MISSING
6  GS_INVALID_VERSION_TYPE
7  GS_UNSUPPORTED_VERSION
8  GS_UNKNOWN_FIELD
9  GS_REQUIRED_FIELD_MISSING
10 GS_SCHEMA_VIOLATION
11 GS_INVALID_TIMESTAMP
12 GS_INVALID_STATUS
13 GS_INVALID_OID
14 GS_COLLECTION_LIMIT
15 GS_DUPLICATE_CHECK_NAME
16 GS_PASS_EXIT_CODE_MISMATCH
17 GS_FAIL_EXIT_CODE_MISMATCH
18 GS_SKIP_EXIT_CODE_MISMATCH
19 GS_UNAVAILABLE_EXIT_CODE_MISMATCH
20 GS_INVALID_DURATION
21 GS_INVALID_OUTPUT_HASH
22 GS_PARTIAL_TEST_TOTALS
23 GS_TEST_TOTAL_MISMATCH
24 GS_OVERALL_STATUS_MISMATCH
25 GS_SCOPE_CLOSED_DIRTY_WORKTREE
26 GS_NORMALIZATION_FAILURE
27 GS_INTERNAL
```

Within a single code, diagnostics are sorted by `Path` and then by
encounter index, so the order is deterministic across runs.

## 3. Fixture and generated-case contract

Rows in §1 are covered by committed fixtures or by the generated lexical
matrix in
[`gate-summary-schema-version-translation.md`](./gate-summary-schema-version-translation.md).
Generated raw-number cases are not committed JSON fixtures.

The global fixture boundary has three numbers that must be reported
together:

- **All committed JSON fixtures: 41.** The 7 valid + 28 invalid + 3
  duplicate-key + 3 limit-shape files under
  `internal/gatesummary/testdata/`.
- **Executable accept/reject corpus: 38.** The 7 valid + 28 invalid + 3
  duplicate-key files.
- **Limit-shape templates: 3.** Static-shape markers; numeric boundaries
  are generated programmatically in `CONFORMANCE01`.

The separately named **v2-only executable corpus is 35**: 5 valid v2 +
27 invalid v2 + 3 duplicate-key v2. It is a version-specific subset of
the 38-file executable corpus, not an alternative fixture manifest.

Fixture family breakdown:

- Valid: 7 (2 v1 + 5 v2).
- Invalid: 28 (1 v1 + 27 v2).
- Duplicate-key: 3 (all v2).
- Limit-shape: 3 (all v2).
- **All committed JSON fixtures: 41; executable accept/reject fixtures:
  38; limit-shape templates: 3.**

`GS_NORMALIZATION_FAILURE` and `GS_INTERNAL` use injected internal-failure
tests rather than ordinary invalid-input fixtures. The impossible
post-dispatch schema-version mismatch is one `GS_INTERNAL` case.
`GS_SCHEMA_VIOLATION` is the fallback for a selected-schema leaf that the
complete translation table does not map more specifically:
[`gate-summary-schema-error-translation.md`](./gate-summary-schema-error-translation.md).

## 4. Producer policy after release

Once `RELEASE01` ships:

- Leamas examples emit v2.
- Leamas self-hosted gate summaries emit v2.
- New downstream producers should emit v2.
- v1 remains readable.
- No automatic v1-to-v2 rewriting is introduced.
- v2 is immutable except for clarifying documentation.

## 5. Compatibility testing

The conformance ACT (`ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01`)
must:

- encode every "Accept" row as a green test;
- encode every "Reject" row as a red test (with the expected code);
- prove that the diagnostic order matches §2;
- run the chosen Draft 2020-12 JSON Schema validator with
  `AssertFormat()` enabled;
- run the chosen validator against the schemas to confirm they
  parse as Draft 2020-12;
- exercise the complete structured schema-error translation table;
- generate every lexical case frozen in the version-translation matrix.

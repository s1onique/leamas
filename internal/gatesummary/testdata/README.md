# Gate-Summary Fixture Inventory

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
> The fixture set is the source of truth for the conformance tests
> committed in `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01`.

This directory holds inert JSON fixtures. None of them are consumed by
production code until `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` introduces
the embedded reader.

## Layout

```text
schema/
  gate-summary-v1.schema.json
  gate-summary-v2.schema.json
testdata/
  README.md
  valid/
    v1-minimal.json
    v1-full.json
    v2-minimal.json
    v2-full.json
    v2-root-scope.json
    v2-clinemm-microc3.json
    v2-leamas-self-hosted.json
  invalid/
    v1-unknown-field.json
    v2-missing-schema-version.json
    v2-schema-version-string.json
    v2-schema-version-decimal.json
    v2-schema-version-zero.json
    v2-schema-version-negative.json
    v2-unsupported-version-3.json
    v2-empty-generated-at.json
    v2-invalid-timestamp.json
    v2-missing-execution-head-oid.json
    v2-null-execution-head-oid.json
    v2-uppercase-oid.json
    v2-scope-closed-dirty-after.json
    v2-pass-nonzero-exit.json
    v2-fail-exit-zero.json
    v2-skip-nonnull-exit.json
    v2-unavailable-nonnull-exit.json
    v2-partial-test-totals.json
    v2-test-total-mismatch.json
    v2-negative-duration.json
    v2-invalid-hash.json
    v2-duplicate-check-name.json
    v2-trailing-second-value.json
    v2-unknown-field.json
    v2-bad-status-enum.json
    v2-overall-mismatch.json
    v2-lower-lifecycle.json
    v2-truncated.json
  duplicate-keys/
    v2-duplicate-schema-version.json
    v2-duplicate-top-level-field.json
    v2-duplicate-nested-field.json
  limits/
    README.md
    v2-checks-boundary-shape.json
    v2-checks-over-boundary-shape.json
    v2-document-size-shape.json
```

## Totals

The global corpus boundary pins three numbers:

```text
all committed JSON fixtures     = 41
executable accept/reject corpus = 38
limit-shape templates           = 3
```

| Family | Count | Role |
| ------ | ----- | ---- |
| `valid/` | 7 | full-accept corpus (2 v1 + 5 v2) |
| `invalid/` | 28 | full-reject corpus (1 v1 + 27 v2) |
| `duplicate-keys/` | 3 | lexical rejection corpus (all v2) |
| `limits/` | 3 | static-shape templates only; numeric boundary tests are programmatic in `CONFORMANCE01` |
| **All committed JSON fixtures** | **41** | 7 + 28 + 3 + 3 |

The separately named **v2-only executable corpus is 35**: 5 valid v2 +
27 invalid v2 + 3 duplicate-key v2. It is a subset of the 38-file global
executable corpus, not an alternative manifest. The three limit-shape
templates are excluded from both executable counts.

`valid/v2-clinemm-microc3.json` is the canary: it carries a
`parent_production_bundle` check with `status=fail`, so the derivation
rule yields `overall_status=fail`.

`invalid/v2-skip-nonnull-exit.json` carries a `status=skip` check with a
non-null `exit_code`. Its `overall_status` is `unavailable`, so this
fixture isolates the exit-code invariant.

`invalid/v2-schema-version-negative.json` exercises `schema_version=-1`.
It is valid JSON integer syntax, and the pre-schema version dispatcher
maps its unsupported value to `GS_UNSUPPORTED_VERSION`. The selected
schema is never invoked for this ordinary input.

Whitespace, leading-zero, plus-sign, decimal, and exponent variants are
programmatically generated from valid documents as frozen in
[`gate-summary-schema-version-translation.md`](../../../docs/factory/gate-summary-schema-version-translation.md).
They are not extra committed JSON fixtures.

## Conventions

- Every fixture is a single JSON document encoded in UTF-8.
- Two-space indentation; no trailing whitespace.
- Lifecycle statuses are emitted in upper case at the wire boundary.
- OIDs are 40- or 64-character lowercase hexadecimal strings.
- `argv` arrays use short token lists so the fixtures stay small.
- `*_disposition` strings are short sentences without embedded
  control characters.
- Output hashes are always exactly 64 lowercase hex characters; the
  empty-stream SHA-256
  (`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`)
  is used wherever the producer has no output for that stream.

## Expected results

| Family | Expected reader result |
| ------ | ---------------------- |
| `valid/` | Accept; normalize; render |
| `invalid/` | Reject with the diagnostic code listed below |
| `duplicate-keys/` | Reject with `GS_DUPLICATE_KEY` (from envelope scanner, before schema validation) |
| `limits/` | Accept at the boundary shape; the actual numeric boundary tests are programmatic in `CONFORMANCE01` |

### `invalid/` → expected diagnostic

| Fixture | Expected code |
| ------- | ------------- |
| `v1-unknown-field.json` | `GS_UNKNOWN_FIELD` |
| `v2-missing-schema-version.json` | `GS_VERSION_MISSING` |
| `v2-schema-version-string.json` | `GS_INVALID_VERSION_TYPE` |
| `v2-schema-version-decimal.json` | `GS_INVALID_VERSION_TYPE` (version probe) |
| `v2-schema-version-zero.json` | `GS_UNSUPPORTED_VERSION` (version dispatch) |
| `v2-schema-version-negative.json` | `GS_UNSUPPORTED_VERSION` (version dispatch) |
| `v2-unsupported-version-3.json` | `GS_UNSUPPORTED_VERSION` |
| `v2-empty-generated-at.json` | `GS_INVALID_TIMESTAMP` |
| `v2-invalid-timestamp.json` | `GS_INVALID_TIMESTAMP` |
| `v2-missing-execution-head-oid.json` | `GS_REQUIRED_FIELD_MISSING` |
| `v2-null-execution-head-oid.json` | `GS_INVALID_OID` |
| `v2-uppercase-oid.json` | `GS_INVALID_OID` |
| `v2-scope-closed-dirty-after.json` | `GS_SCOPE_CLOSED_DIRTY_WORKTREE` |
| `v2-pass-nonzero-exit.json` | `GS_PASS_EXIT_CODE_MISMATCH` |
| `v2-fail-exit-zero.json` | `GS_FAIL_EXIT_CODE_MISMATCH` |
| `v2-skip-nonnull-exit.json` | `GS_SKIP_EXIT_CODE_MISMATCH` |
| `v2-unavailable-nonnull-exit.json` | `GS_UNAVAILABLE_EXIT_CODE_MISMATCH` |
| `v2-partial-test-totals.json` | `GS_PARTIAL_TEST_TOTALS` |
| `v2-test-total-mismatch.json` | `GS_TEST_TOTAL_MISMATCH` |
| `v2-negative-duration.json` | `GS_INVALID_DURATION` |
| `v2-invalid-hash.json` | `GS_INVALID_OUTPUT_HASH` |
| `v2-duplicate-check-name.json` | `GS_DUPLICATE_CHECK_NAME` |
| `v2-trailing-second-value.json` | `GS_TRAILING_JSON` |
| `v2-unknown-field.json` | `GS_UNKNOWN_FIELD` |
| `v2-bad-status-enum.json` | `GS_INVALID_STATUS` |
| `v2-overall-mismatch.json` | `GS_OVERALL_STATUS_MISMATCH` |
| `v2-lower-lifecycle.json` | `GS_INVALID_STATUS` |
| `v2-truncated.json` | `GS_MALFORMED_JSON` |

### `duplicate-keys/` → expected diagnostic

Every fixture in `duplicate-keys/` is rejected with
`GS_DUPLICATE_KEY` by the envelope scanner **before** schema
validation runs.

### `limits/` → boundary semantics

The three limit-shape fixtures are intentionally small, well-formed
v2 documents whose role is to exercise the schema-level shape. The
actual numeric boundary tests (4 MiB+1 document, 10,000 checks,
1,024 argv entries, 64 KiB+1 argv element, 1,048,576+1 byte
`evidence`) are programmatic in `CONFORMANCE01` and produce
`GS_DOCUMENT_TOO_LARGE` or `GS_COLLECTION_LIMIT` at the boundary.

## Reviewer notes

- Every `valid/` fixture has at least one matching `invalid/`
  mutation in the same file family.
- Every ordinary-input diagnostic is covered by a fixture or a frozen
  generated case. `GS_NORMALIZATION_FAILURE` and `GS_INTERNAL` use
  injected internal-failure tests.
- `GS_SCHEMA_VIOLATION` is the umbrella code emitted when a selected
  schema failure does not map to a specific row in
  [`gate-summary-schema-error-translation.md`](../../../docs/factory/gate-summary-schema-error-translation.md).
- `GS_COLLECTION_LIMIT` is exercised by the programmatic boundary
  generator in `CONFORMANCE01`; the static `limits/` fixtures only
  document the structural shape.

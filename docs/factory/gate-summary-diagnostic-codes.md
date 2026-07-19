# Gate-Summary Diagnostic Codes

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.

Stable public machine identifiers for every error condition the
gate-summary reader can report. Codes are stable across patch releases;
human messages may improve without changing the code.

## 1. Diagnostic shape

```go
type Diagnostic struct {
    Code     string `json:"code"`
    Path     string `json:"path,omitempty"`
    Expected string `json:"expected,omitempty"`
    Observed string `json:"observed,omitempty"`
    Message  string `json:"message"`
}
```

## 2. Ordering algorithm

The reader emits diagnostics in this exact order:

1. **Precedence rank** (lower number = higher priority = emitted first).
2. **JSON Pointer path** (lexicographic).
3. **Encounter index** (order in which the decoder observed the violation).

The compatibility matrix and the reader implementation must use the
**same** ordering algorithm. The two prior definitions
(lexicographic-by-Code-path in the registry vs. semantic precedence
in the compatibility matrix) are unified here on
`precedence rank, then path, then encounter index`.

## 3. Registry

| Code | Prec. | Layer | Class | Notes |
| ---- | ----- | ----- | ----- | ----- |
| `GS_DOCUMENT_TOO_LARGE` | 1 | envelope | size | 4 MiB cap reached before tokenization. Static-shape marker at `testdata/limits/v2-document-size-shape.json`; the actual 4 MiB+1 byte test is programmatic in `CONFORMANCE01`. |
| `GS_MALFORMED_JSON` | 2 | envelope | parse | JSON syntax error, including leading-zero (`02`, `-02`) and leading-plus (`+2`) number spellings. See `v2-truncated.json` and generated lexical cases. |
| `GS_TRAILING_JSON` | 3 | envelope | parse | Second JSON value after the first. See `v2-trailing-second-value.json`. |
| `GS_DUPLICATE_KEY` | 4 | envelope | lexical | Duplicate object member name. See lexical corpus under `duplicate-keys/`. |
| `GS_VERSION_MISSING` | 5 | version probe | version | `schema_version` field absent. See `v2-missing-schema-version.json`. |
| `GS_INVALID_VERSION_TYPE` | 6 | version probe | version | `schema_version` is not a lexically integral JSON number. See string/decimal fixture and generated exponent cases. |
| `GS_UNSUPPORTED_VERSION` | 7 | version dispatch | version | Valid integer value outside `{1, 2}`. See zero, negative, and version-3 fixtures. |
| `GS_UNKNOWN_FIELD` | 8 | schema translation | schema | `additionalProperties` failure. See v1 and v2 unknown-field fixtures. |
| `GS_REQUIRED_FIELD_MISSING` | 9 | schema translation | schema | `required` failure outside the test-total `anyOf`. See the missing-OID fixture. |
| `GS_SCHEMA_VIOLATION` | 10 | schema translation | schema | Umbrella code when no specific translation-table row applies. |
| `GS_INVALID_TIMESTAMP` | 11 | schema translation | date | `format` or `minLength` failure at `/generated_at`. |
| `GS_INVALID_STATUS` | 12 | schema translation | enum | `enum` failure at a status path. |
| `GS_INVALID_OID` | 13 | schema translation | oid | `type` or `pattern` failure at an execution-identity path. |
| `GS_COLLECTION_LIMIT` | 14 | schema translation | limit | `maxItems` or `maxLength` failure. Programmatic generator in `CONFORMANCE01`. |
| `GS_DUPLICATE_CHECK_NAME` | 15 | semantic | name | Two or more `checks[]` entries share the same `name`. |
| `GS_PASS_EXIT_CODE_MISMATCH` | 16 | semantic | exit | `status=pass` with non-zero `exit_code`. |
| `GS_FAIL_EXIT_CODE_MISMATCH` | 17 | semantic | exit | `status=fail` with `exit_code=0`. |
| `GS_SKIP_EXIT_CODE_MISMATCH` | 18 | semantic | exit | `status=skip` with non-null `exit_code`. |
| `GS_UNAVAILABLE_EXIT_CODE_MISMATCH` | 19 | semantic | exit | `status=unavailable` with non-null `exit_code`. |
| `GS_INVALID_DURATION` | 20 | schema translation | numeric | `minimum` failure at `duration_ms`. |
| `GS_INVALID_OUTPUT_HASH` | 21 | schema translation | hash | `pattern` failure at `stdout_sha256` or `stderr_sha256`. |
| `GS_PARTIAL_TEST_TOTALS` | 22 | schema translation | arithmetic | Test-total `anyOf`/`not` subtree fails; nested causes collapse to one code. |
| `GS_TEST_TOTAL_MISMATCH` | 23 | semantic | arithmetic | The arithmetic invariant is violated. |
| `GS_OVERALL_STATUS_MISMATCH` | 24 | semantic | aggregate | Producer-claimed `overall_status` does not match the derived value. |
| `GS_SCOPE_CLOSED_DIRTY_WORKTREE` | 25 | semantic | clean | `scope_status=CLOSED` with `worktree_clean_before=false` or `worktree_clean_after=false`. |
| `GS_NORMALIZATION_FAILURE` | 26 | normalizer | test-only | Test-only fault injection; asserted by `CONFORMANCE01` and `DECODER01`. |
| `GS_INTERNAL` | 27 | internal | defect | Injected internal failure or impossible post-dispatch schema-version mismatch; never an ordinary invalid-input classification. |

## 4. Resolution policy

A single document may produce multiple diagnostics. Diagnostics are
collected, deduplicated by `(Code, Path)`, and emitted in the order
defined in §2.

If both `GS_SCHEMA_VIOLATION` and a more specific semantic code apply
to the same violation, the more specific code wins and the
schema-violation code is suppressed for that path. `GS_SCHEMA_VIOLATION`
is emitted only when no other diagnostic covers the violation. The
complete deterministic mapping and cause-collapse rules are frozen in
[`gate-summary-schema-error-translation.md`](./gate-summary-schema-error-translation.md).
Version diagnostics and generated lexical cases are frozen in
[`gate-summary-schema-version-translation.md`](./gate-summary-schema-version-translation.md).

## 5. CLI surface

The CLI's `--json` mode emits diagnostics as a JSON array on stdout.
The human mode emits one diagnostic per line on stderr. In both modes
the order matches §2.

## 6. Stability

Codes are stable identifiers. Adding a new code is permitted in any
release. Renaming, repurposing, or removing a code is a breaking
change and requires a major-version bump plus a correction ACT.

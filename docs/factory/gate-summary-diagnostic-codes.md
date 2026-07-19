# Gate-Summary Diagnostic Codes

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`.

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
| `GS_MALFORMED_JSON` | 2 | envelope | parse | JSON syntax error. See `v2-truncated.json`. |
| `GS_TRAILING_JSON` | 3 | envelope | parse | Second JSON value after the first. See `v2-trailing-second-value.json`. |
| `GS_DUPLICATE_KEY` | 4 | envelope | lexical | Duplicate object member name. See lexical corpus under `duplicate-keys/`. |
| `GS_VERSION_MISSING` | 5 | envelope | version | `schema_version` field absent. See `v2-missing-schema-version.json`. |
| `GS_INVALID_VERSION_TYPE` | 6 | envelope | version | `schema_version` is not a JSON integer token. See string and decimal fixtures. |
| `GS_UNSUPPORTED_VERSION` | 7 | envelope | version | `schema_version` numeric value outside `{1, 2}` (zero, negatives, ≥3). See `v2-schema-version-zero.json`, `v2-unsupported-version-3.json`, `v2-schema-version-negative.json`. |
| `GS_UNKNOWN_FIELD` | 8 | wire | schema | Object contains a field not declared for its version. See v1 and v2 unknown-field fixtures. |
| `GS_REQUIRED_FIELD_MISSING` | 9 | wire | schema | A required field is absent. See `v2-missing-execution-head-oid.json`. |
| `GS_SCHEMA_VIOLATION` | 10 | schema | schema | Umbrella code emitted when no more specific semantic code applies. |
| `GS_INVALID_TIMESTAMP` | 11 | semantic | date | `generated_at` is empty, missing UTC offset, or unparseable as RFC 3339. |
| `GS_INVALID_STATUS` | 12 | semantic | enum | A status field uses a value outside its vocabulary. |
| `GS_INVALID_OID` | 13 | semantic | oid | An execution-identity field is not 40 or 64 lowercase hex characters. |
| `GS_COLLECTION_LIMIT` | 14 | semantic | limit | A collection size or string length exceeds the documented resource limit. Programmatic generator in `CONFORMANCE01`. |
| `GS_DUPLICATE_CHECK_NAME` | 15 | semantic | name | Two or more `checks[]` entries share the same `name`. |
| `GS_PASS_EXIT_CODE_MISMATCH` | 16 | semantic | exit | `status=pass` with non-zero `exit_code`. |
| `GS_FAIL_EXIT_CODE_MISMATCH` | 17 | semantic | exit | `status=fail` with `exit_code=0`. |
| `GS_SKIP_EXIT_CODE_MISMATCH` | 18 | semantic | exit | `status=skip` with non-null `exit_code`. |
| `GS_UNAVAILABLE_EXIT_CODE_MISMATCH` | 19 | semantic | exit | `status=unavailable` with non-null `exit_code`. |
| `GS_INVALID_DURATION` | 20 | semantic | numeric | `duration_ms` is negative. |
| `GS_INVALID_OUTPUT_HASH` | 21 | semantic | hash | `stdout_sha256` or `stderr_sha256` is not exactly 64 lowercase hex characters. |
| `GS_PARTIAL_TEST_TOTALS` | 22 | semantic | arithmetic | One or more but not all of the test-count fields are present. |
| `GS_TEST_TOTAL_MISMATCH` | 23 | semantic | arithmetic | The arithmetic invariant is violated. |
| `GS_OVERALL_STATUS_MISMATCH` | 24 | semantic | aggregate | Producer-claimed `overall_status` does not match the derived value. |
| `GS_SCOPE_CLOSED_DIRTY_WORKTREE` | 25 | semantic | clean | `scope_status=CLOSED` with `worktree_clean_before=false` or `worktree_clean_after=false`. |
| `GS_NORMALIZATION_FAILURE` | 26 | normalizer | test-only | Test-only fault injection; asserted by `CONFORMANCE01` and `DECODER01`. |
| `GS_INTERNAL` | 27 | normalizer | test-only | Test-only fault injection; asserted by `CONFORMANCE01` and `DECODER01`. |

## 4. Resolution policy

A single document may produce multiple diagnostics. Diagnostics are
collected, deduplicated by `(Code, Path)`, and emitted in the order
defined in §2.

If both `GS_SCHEMA_VIOLATION` and a more specific semantic code apply
to the same violation, the more specific code wins and the
schema-violation code is suppressed for that path. `GS_SCHEMA_VIOLATION`
is emitted only when no other diagnostic covers the violation.

## 5. CLI surface

The CLI's `--json` mode emits diagnostics as a JSON array on stdout.
The human mode emits one diagnostic per line on stderr. In both modes
the order matches §2.

## 6. Stability

Codes are stable identifiers. Adding a new code is permitted in any
release. Renaming, repurposing, or removing a code is a breaking
change and requires a major-version bump plus a correction ACT.

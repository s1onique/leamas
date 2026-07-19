# Schema-Version → Diagnostic Translation

> **Status:** Frozen as of
> `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.

This document is the normative ownership and generated-case contract for
`schema_version`. It is split from
[`gate-summary-v2-spec.md`](./gate-summary-v2-spec.md) to keep that file
LLM-friendly.

## 1. Pipeline ownership

The decoder classifies the version before selecting a JSON Schema:

```text
bounded read
→ syntactic/token scan
→ duplicate-key detection
→ schema_version probe
→ exact lexical/type/value classification
→ v1/v2 dispatch
→ selected JSON Schema validation
```

The token scan may collect the version value while checking syntax and
object keys, but the owning stages and public outcomes remain as ordered
above. Malformed syntax wins before any version diagnostic. Duplicate-key
detection wins before a missing/type/value result.

## 2. Normative table

| Input form | Owner | Diagnostic/result | Rationale |
| ---------- | ----- | ----------------- | --------- |
| Malformed JSON anywhere | syntax scan | `GS_MALFORMED_JSON` | RFC 8259 grammar did not produce one JSON value |
| `02`, `-02`, or `+2` | syntax scan | `GS_MALFORMED_JSON` | leading zero and leading plus are not JSON number grammar |
| field missing | version probe | `GS_VERSION_MISSING` | completed top-level object has no discriminator |
| duplicate `schema_version` key | token scan | `GS_DUPLICATE_KEY` | duplicate detection precedes dispatch |
| string, Boolean, null, array, or object | version probe | `GS_INVALID_VERSION_TYPE` | discriminator is not a JSON number token |
| `2.0`, `2.00`, `2e0`, or `2E0` | version probe | `GS_INVALID_VERSION_TYPE` | valid JSON number, but not an integer lexical form |
| valid integer other than `1` or `2` | version dispatch | `GS_UNSUPPORTED_VERSION` | supported set is exactly `{1, 2}` |
| `1` | dispatch | select v1 schema | supported exact integer value |
| `2` | dispatch | select v2 schema | supported exact integer value |
| RFC 8259 whitespace around `1` or `2` | dispatch | select v1/v2 schema | insignificant JSON whitespace is not part of the number token |

The version probe uses `json.Decoder.UseNumber()`. It applies
`^-?(0|[1-9][0-9]*)$` to the returned `json.Number.String()`, not to a
source slice containing surrounding whitespace. Decimal and exponent
forms are valid JSON numbers but fail this integer-form test. Leading-zero
and leading-plus spellings never become `json.Number` values because JSON
parsing fails first.

A lexically valid integer can be classified without narrowing it to a
machine-sized `int`: exact values `1` and `2` dispatch; every other valid
integer spelling is unsupported. This keeps very large integers in
`GS_UNSUPPORTED_VERSION` rather than turning integer overflow into an
invented public code.

## 3. Schema defense in depth

The v1 and v2 schemas retain `type: integer` and `const: 1` / `const: 2`.
They do not own ordinary version input. The dispatcher selects a schema
only after the table in §2 succeeds.

After successful dispatch, a selected-schema `type` or `const` failure at
`/schema_version` indicates an internal pipeline defect and maps to
`GS_INTERNAL`. It must never translate an unsupported input to
`GS_UNSUPPORTED_VERSION`; that code belongs exclusively to version
dispatch.

## 4. Generated lexical matrix

`DECODER01` and `CONFORMANCE01` generate these cases from known-valid v1
and v2 documents. They are programmatic cases, not additional committed
JSON fixtures, so they do not change the 41 / 38 / 3 fixture inventory.
Each case runs through the full reader; test instrumentation also asserts
the owning stage from §2.

### 4.1 Insignificant whitespace

For each supported lexeme in `{1, 2}`, generate the Cartesian product:

```text
prefix whitespace ∈ {"", " ", "\t", "\n", "\r"}
suffix whitespace ∈ {"", " ", "\t", "\n", "\r"}
placement         ∈ {before comma, before closing brace}
```

Insert `prefix + lexeme + suffix` after the `schema_version` colon in an
otherwise valid document. For the closing-brace placement, move
`schema_version` to the final object member without changing any value.
All 100 generated documents (2 × 5 × 5 × 2) must be accepted and dispatch
to the matching schema.

### 4.2 Leading-zero and plus-sign syntax

| Raw lexemes | Expected code |
| ----------- | ------------- |
| `01`, `02`, `-01`, `-02` | `GS_MALFORMED_JSON` |
| `+1`, `+2` | `GS_MALFORMED_JSON` |

### 4.3 Decimal and exponent number forms

| Raw lexemes | Expected code |
| ----------- | ------------- |
| `1.0`, `2.0`, `2.00`, `-2.0` | `GS_INVALID_VERSION_TYPE` |
| `1e0`, `2e0`, `2E0`, `2e+0`, `2e-0` | `GS_INVALID_VERSION_TYPE` |

### 4.4 Unsupported integer values

| Raw lexemes | Expected code |
| ----------- | ------------- |
| `-2`, `-1`, `-0`, `0`, `3`, `4` | `GS_UNSUPPORTED_VERSION` |
| integers outside the platform `int` range | `GS_UNSUPPORTED_VERSION` |

Generated tests must mutate raw JSON bytes. Marshaling a Go value cannot
produce leading-zero or leading-plus cases and would erase the lexical
dimension this contract protects.

# Schema-Version → Diagnostic Translation

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`.

This document carries the normative layered diagnostic ownership
table for `schema_version` failures. It is split out from
[`gate-summary-v2-spec.md`](./gate-summary-v2-spec.md) §4 to keep
that file under the LLM-friendliness line limit.

## Layered diagnostic ownership

Every `schema_version` failure has exactly one owning layer. The
decoder reports the diagnostic emitted by the owning layer and
suppresses any umbrella `GS_SCHEMA_VIOLATION` for that path.

| Input form | Layer | Diagnostic | Rationale |
| ---------- | ----- | ---------- | --------- |
| field missing | envelope | `GS_VERSION_MISSING` | required field absent before any tokenization |
| duplicate key (`schema_version` × 2) | envelope | `GS_DUPLICATE_KEY` | token-scan duplicate-key detection precedes schema validation |
| `"2"` (string) | structural | `GS_INVALID_VERSION_TYPE` | `type: integer` mismatch |
| `true` / `false` / `null` / `[]` / `{}` | structural | `GS_INVALID_VERSION_TYPE` | `type: integer` mismatch |
| `2.0` / `2.00` / `2e0` / `2E0` | lexical | `GS_INVALID_VERSION_TYPE` | envelope regex `^-?(0|[1-9][0-9]*)$` rejects fractional / exponent forms |
| `02` / `-02` / `+2` / ` 2` | lexical | `GS_INVALID_VERSION_TYPE` | envelope regex rejects leading zero, sign variants, and surrounding whitespace |
| `-1` / `-2` / `0` / `3` / `4` … | structural | `GS_UNSUPPORTED_VERSION` | `const` mismatch on the supported-version literal |
| `1` | structural | dispatch v1 | exact literal |
| `2` | structural | dispatch v2 | exact literal |

The negative-integer conformance case (`-1`) is exercised by
`invalid/v2-schema-version-negative.json`. The lexical regex allows
`-1` to pass the envelope scanner; the structural layer then rejects
the document at the `const: 2` check and the decoder maps the
rejection to `GS_UNSUPPORTED_VERSION`. This keeps the regex honest
about RFC 8259 integer syntax while still pinning the supported
version set to `{1, 2}`.

## Invalid forms

The following are invalid and map to the diagnostic indicated:

- field missing → `GS_VERSION_MISSING`.
- JSON string, boolean, null, array, or object → `GS_INVALID_VERSION_TYPE`.
- JSON number whose numeric value is not `1` or `2` → `GS_UNSUPPORTED_VERSION`.
- JSON number whose lexical form is not an integer token → `GS_INVALID_VERSION_TYPE`.
- Duplicate `schema_version` key → `GS_DUPLICATE_KEY`.

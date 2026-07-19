# JSON-Schema Error → Diagnostic Translation

> **Status:** Frozen as of
> `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.

This document is the normative translation contract for failures from the
selected v1 or v2 JSON Schema. Version probing and dispatch have already
succeeded before this table applies.

## 1. Structured input only

`DECODER01` consumes `jsonschema/v6.ValidationError` v6.0.2 through only
`SchemaURL`, `InstanceLocation`, `ErrorKind`, and `Causes`. It must not
read `Error()`, `LocalizedError`, or message substrings.

Every `Schema.Validate` failure has a root `*kind.Schema` wrapper. Nested
`*kind.Group` and `*kind.Reference` values are also wrappers. When one of
these wrappers has causes, it contributes no diagnostic; traversal
continues through its causes in slice order. A cause reached through a
reference carries its own dereferenced `SchemaURL`.

### Keyword identity

For each non-reference node, derive the same absolute keyword identity
represented by JSON Schema output, without rendering a message:

1. Parse `ValidationError.SchemaURL` into the base schema URL and its
   decoded RFC 6901 fragment tokens. These identify the schema object.
2. Append `ErrorKind.KeywordPath()` tokens, applying RFC 6901 escaping
   only when later rendering a pointer.
3. Special-case `*kind.Not`, whose v6.0.2 `KeywordPath()` is nil: append
   the token `not`.
4. Do not derive an identity for wrapper `*kind.Schema`, `*kind.Group`,
   or `*kind.Reference`; traverse their causes instead.

Comparisons use the tuple `(base schema URL, decoded pointer tokens)`, not
raw URL text, localized output, or suffix matching. `InstanceLocation` is
converted to a JSON Pointer by escaping each token per RFC 6901. `<i>`
below means one decimal `checks` array index.

## 2. Normative mapping

Rules are matched top to bottom. A matched non-wrapper owns its subtree;
its causes are not independently translated unless the rule says to
traverse them.

| Structured condition | Public code | Public path / handling |
| -------------------- | ----------- | ---------------------- |
| `*kind.Required` includes missing `schema_version` after dispatch | `GS_INTERNAL` | `/schema_version`; suppress the node because schema selection should have been impossible |
| `*kind.Type` or `*kind.Const` at `/schema_version` after dispatch | `GS_INTERNAL` | `/schema_version`; ordinary input cannot reach this state |
| `*kind.AnyOf` with keyword identity `https://leamas.local/schemas/gate-summary-v2.schema.json#/$defs/check/anyOf` | `GS_PARTIAL_TEST_TOTALS` | `/checks/<i>`; emit once and suppress all nested `required`, `not`, and `anyOf` causes |
| `*kind.Required`, outside the test-total subtree | `GS_REQUIRED_FIELD_MISSING` | Sort `Missing`; emit one at `<instance>/<missing-name>` per name |
| `*kind.AdditionalProperties` | `GS_UNKNOWN_FIELD` | Sort `Properties`; emit one at `<instance>/<property-name>` per name |
| `*kind.Enum` at `/overall_status`, `/scope_status`, `/parent_status`, or `/checks/<i>/status` | `GS_INVALID_STATUS` | Validator instance location |
| `*kind.Type` or `*kind.Pattern` at `/execution_head_oid`, `/execution_tree_oid`, or `/subject_tree_oid` | `GS_INVALID_OID` | Validator instance location |
| `*kind.Pattern` at `/checks/<i>/extras/stdout_sha256` or `/checks/<i>/extras/stderr_sha256` | `GS_INVALID_OUTPUT_HASH` | Validator instance location |
| `*kind.Minimum` at `/checks/<i>/extras/duration_ms` | `GS_INVALID_DURATION` | Validator instance location |
| `*kind.Format` or `*kind.MinLength` at `/generated_at` | `GS_INVALID_TIMESTAMP` | `/generated_at` |
| `*kind.MaxItems` or `*kind.MaxLength` at any v2 path | `GS_COLLECTION_LIMIT` | Validator instance location; expected/observed use structured kind fields |
| Any other non-wrapper kind, known or unknown | `GS_SCHEMA_VIOLATION` | Validator instance location; emit once and suppress its causes |

The OID `type` row preserves the frozen `null`-OID behavior. Other wrong
JSON types use the umbrella code unless explicitly mapped. Negative test
counts therefore produce `GS_SCHEMA_VIOLATION`; only `duration_ms` has
`GS_INVALID_DURATION`.

## 3. Traversal, fanout, and ordering

Translation is deterministic:

1. Require and discard the root `*kind.Schema` wrapper, then visit its
   causes in their existing order.
2. For `*kind.Schema`, `*kind.Group`, or `*kind.Reference` below the root,
   visit causes in order. If such a wrapper unexpectedly has no cause,
   emit `GS_SCHEMA_VIOLATION` at its instance location.
3. Compute keyword identity for the non-wrapper node.
4. Apply §2. The test-total `anyOf` collapses before its causes are
   visited. Other matched nodes also suppress their causes.
5. For `Required` and `AdditionalProperties`, preserve the node's
   encounter position, sort names lexicographically, and fan out.
6. Assign encounter indexes as diagnostics are produced, deduplicate by
   `(Code, Path)`, then sort by registry precedence, path, and encounter
   index.

A malformed root error value or impossible missing root wrapper is an
internal validator integration failure and maps to `GS_INTERNAL`; it is
not reclassified by parsing its text.

## 4. Layer boundary

Malformed/trailing JSON, duplicate keys, document size, every ordinary
version failure, and post-schema semantic invariants never enter this
table. Their owners are frozen in the version translation, diagnostic
registry, resource limits, and v2 specification.

The schemas retain version `const` only as defense in depth. A
post-dispatch version failure is `GS_INTERNAL`, never ordinary
`GS_UNSUPPORTED_VERSION`.

# JSON Schema Validator Selection

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`.
> The selection made in this document is binding for
> `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` and downstream ACTs.

This memo records the offline Draft 2020-12 JSON Schema validator that
the gate-summary reader will use, the rationale, the offline review
workflow used to validate the frozen fixtures, and the frozen
`AssertFormat` policy.

## 1. Requirements

The validator must:

- support JSON Schema **Draft 2020-12**;
- run **offline** (no remote `$ref` resolution against the public
  Internet);
- be pinnable as a Go module dependency;
- be small enough that adding it does not regress the static-binary
  intent verifier (`scripts/verify_static_binary_intent.sh`);
- accept `additionalProperties: false` semantics as required by the
  v2 spec;
- not require CGo;
- support the format-assertion vocabulary when the consumer asks for
  it.

## 2. Selected implementation

**Selected:** `github.com/santhosh-tekuri/jsonschema/v6` (latest v6.x
release).

`v5` is **not** selected even if the latest v6.x release is briefly
unavailable at pin time. The v6 major version is the supported
validator and its API surface is the contract.

Rationale:

- It is a pure-Go validator; no CGo, no system dependencies.
- It supports Draft 2020-12 and `additionalProperties: false`.
- It can validate a single in-memory JSON document against a single
  in-memory JSON Schema without remote resolution.
- It is widely used in the Go ecosystem and has a stable API.
- The library does not require network access at validation time, so
  the Leamas single-binary guarantee survives.

The exact module version is pinned in
`ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01`. This ACT only records the
selection so subsequent ACTs can proceed without re-justifying it.

## 3. Frozen `AssertFormat` policy

`santhosh-tekuri/jsonschema/v6` does **not** assert formats by
default. JSON Schema treats `format` as an annotation in Draft 2020-12
unless the schema opts in via the format-assertion vocabulary.

This ACT freezes the policy:

```go
compiler := jsonschema.NewCompiler()
compiler.AddResource(...)            // add v1, v2, and check schemas
compiler.AssertFormat()              // opt in to format assertion
```

`generated_at` uses `format: date-time` in both v1 and v2 schemas.
Without `AssertFormat()`, the validator would accept any non-empty
string in `generated_at`. The frozen policy makes the format check
authoritative, which is what the compatibility matrix's
`GS_INVALID_TIMESTAMP` row requires.

## 4. Embedded schema usage

Once pinned, the v1 and v2 schemas under
`internal/gatesummary/schema/` are embedded into the Leamas binary:

```go
//go:embed schema/*.schema.json
var schemas embed.FS
```

The reader looks up the right schema by version, unmarshals it once at
init, and reuses the compiled validator for the lifetime of the
process. The schemas are not reloaded from disk at runtime.

## 5. Lexical rules live outside JSON Schema

The following rules are NOT covered by the JSON Schema validator and
must be enforced by the pre-schema envelope scanner in `DECODER01`:

- `schema_version` must be a JSON integer token **lexically** without
  fractional part, exponent, leading zero, or surrounding whitespace.
  Draft 2020-12 considers `2` and `2.0` equivalent integers;
  `type: integer, const: 2` therefore does **not** distinguish them.
  Lexical inspection uses `json.Number` via `Decoder.UseNumber()`.
- Duplicate object member names must be rejected at every depth
  before ordinary decoding. JSON Schema validation cannot see
  duplicates because `encoding/json` silently uses the last occurrence.
- Trailing JSON values after the first must be rejected. JSON Schema
  validates exactly one JSON value; the trailing-value check happens
  after `Decoder.Decode` returns `io.EOF`.

## 6. Offline review workflow (this ACT)

This ACT validates every schema by hand using one of the following
offline tools (whichever is available on the reviewer's host):

- `ajv` (Node.js, `npx --offline ajv validate --spec=draft2020`);
- A small Go scratch program in `/tmp` (not committed to the
  repository) that uses the pinned `v6` validator with
  `AssertFormat()`.

The chosen tool, the command sequence, and the pass/fail output for
each fixture are recorded in the close report for
`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`. That ACT runs
the chosen validator against both schemas with `AssertFormat()`
enabled and against every fixture before the contract is closed.

## 7. Acceptance criteria for the selection

The selected validator is acceptable only if all of the following are
true:

- it accepts every fixture in
  `internal/gatesummary/testdata/valid/`;
- it rejects every fixture in
  `internal/gatesummary/testdata/invalid/` (with the appropriate
  structural code; semantic-only violations are caught at the
  semantic-validation layer);
- it does not emit network requests at validation time;
- it produces a deterministic accept/reject classification for any
  given input;
- `AssertFormat()` is invoked before any validation.

These criteria are checked in `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01`
when the validator is wired into the production reader.

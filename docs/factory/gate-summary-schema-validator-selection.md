# JSON Schema Validator Selection

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
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

## 5. Version and envelope rules precede JSON Schema

A schema is selected only after the envelope has classified the version:

- Leading-zero (`02`, `-02`) and leading-plus (`+2`) spellings are
  malformed JSON and produce `GS_MALFORMED_JSON` during syntax scan.
- RFC 8259 space, tab, line feed, and carriage return around a valid
  integer value are insignificant and must not change dispatch.
- Decimal and exponent number forms are valid JSON, but are not accepted
  version integer forms; the version probe reports
  `GS_INVALID_VERSION_TYPE`.
- Every valid integer value other than 1 or 2 is rejected by version
  dispatch with `GS_UNSUPPORTED_VERSION`.

Lexical inspection uses `json.Number` via `Decoder.UseNumber()`; its
string excludes surrounding JSON whitespace. Draft 2020-12 considers `2`
and `2.0` equivalent integers, so schema `type`/`const` cannot own this
classification.

Duplicate object member names are rejected at every depth before ordinary
decoding. Trailing JSON after the first value is rejected separately. The
normative pipeline and generated matrix are in
[`gate-summary-schema-version-translation.md`](./gate-summary-schema-version-translation.md).

After dispatch, selected-schema errors are translated exclusively from
structured `ValidationError` data according to
[`gate-summary-schema-error-translation.md`](./gate-summary-schema-error-translation.md).
Localized validator messages are never parsed.

## 6. Offline review workflow (this ACT)

This ACT validates every schema by hand using one of the following
offline tools (whichever is available on the reviewer's host):

- `ajv` (Node.js, `npx --offline ajv validate --spec=draft2020`);
- A small Go scratch program in `/tmp` (not committed to the
  repository) that uses the pinned `v6` validator with
  `AssertFormat()`.

The chosen tool, the command sequence, and the pass/fail output for
each fixture are recorded in the close report for
`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`. That accepted proof
runs the chosen validator against both schemas with `AssertFormat()`
enabled and reports all 41 committed JSON fixtures. `CORRECTION03`
clarifies reader semantics without changing the schemas or validator
selection.

## 7. Acceptance criteria for the selection

The selected validator is acceptable only if all of the following are
true:

- it accepts every fixture in
  `internal/gatesummary/testdata/valid/`;
- it rejects each structurally invalid fixture and may accept fixtures
  whose owning layer is envelope or semantic, exactly as recorded by the
  accepted 41-fixture proof;
- it does not emit network requests at validation time;
- it produces a deterministic accept/reject classification for any
  given input;
- `AssertFormat()` is invoked before any validation.

These criteria are checked in `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01`
when the validator is wired into the production reader.

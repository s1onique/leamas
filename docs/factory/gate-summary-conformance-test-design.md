# Gate-Summary Conformance Test Design

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
> The implementation lives in
> `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01`.

This document is the regression-test specification for the
gate-summary reader. It pairs every JSON Schema rule with the matching
Go wire-struct rule so that neither can drift in isolation.

## 1. Goals

1. Every JSON Schema constraint has at least one Go test that exercises
   the corresponding code path.
2. Every Go wire-struct rule has at least one fixture that exercises
   it through the JSON Schema validator.
3. The test corpus is the same corpus that
   `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` commits.

## 2. Test families

The conformance suite is split into the following families, each with
its own Go test file and fixture directory.

| Family | Fixture directory | Go test file (added in `CONFORMANCE01`) |
| ------ | ----------------- | --------------------------------------- |
| Valid v1 | `testdata/valid/v1-*.json` | `v1_conformance_test.go` |
| Valid v2 | `testdata/valid/v2-*.json` | `v2_conformance_test.go` |
| Invalid v1 | `testdata/invalid/v1-*.json` | `v1_invalid_test.go` |
| Invalid v2 | `testdata/invalid/v2-*.json` | `v2_invalid_test.go` |
| Duplicate keys | `testdata/duplicate-keys/*.json` | `duplicate_keys_test.go` |
| Resource limits | `testdata/limits/*.json` + programmatic generator | `limits_test.go` |
| Fuzz seeds | `testdata/fuzz-seed/*.json` | `fuzz_test.go` |
| Fault injection | none | `fault_injection_test.go` |

## 3. Acceptance test design

For every valid fixture, the conformance test must:

1. Read the fixture.
2. Validate it against the embedded JSON Schema for the matching
   version (with `compiler.AssertFormat()` enabled).
3. Decode it into the matching Go wire struct using
   `DisallowUnknownFields`.
4. Normalize the wire struct into the domain `Summary`.
5. Compare the normalized output against the `normalization_golden`
   fixture for the same input.

Steps 2 and 3 must both pass for the test to pass. A failure in
either step indicates schema/struct drift.

## 4. Rejection test design

For every invalid fixture, the conformance test must:

1. Read the fixture.
2. Run it through the full reader pipeline (bounded read → syntax and
   duplicate scan → version probe/dispatch → selected schema → semantic
   validator → normalizer).
3. Assert that the reader returns a non-empty diagnostic list.
4. Assert that the diagnostic list contains the expected
   `Diagnostic.Code` from
   [`gate-summary-diagnostic-codes.md`](./gate-summary-diagnostic-codes.md).
5. Assert that the diagnostic list is ordered by precedence rank,
   then path, then encounter index, as documented in
   [`gate-summary-compatibility-matrix.md`](./gate-summary-compatibility-matrix.md).

## 5. Lexical envelope tests

Committed fixtures cover duplicate keys, trailing JSON, and one decimal
version form. Programmatic tests must also generate the complete raw-byte
matrix frozen in
[`gate-summary-schema-version-translation.md`](./gate-summary-schema-version-translation.md):

- all RFC 8259 whitespace combinations around supported `1` and `2`
  tokens, both before a comma and before the closing brace — accept and
  dispatch to the matching schema;
- leading-zero forms including `02` and `-02` —
  `GS_MALFORMED_JSON` at syntax scan;
- leading-plus forms including `+2` — `GS_MALFORMED_JSON` at syntax
  scan;
- decimal and exponent forms — `GS_INVALID_VERSION_TYPE` at the version
  probe;
- valid unsupported integers — `GS_UNSUPPORTED_VERSION` at dispatch.

Tests mutate raw JSON bytes rather than marshal Go numbers. They assert
both the public result and that selected-schema validation is not invoked
for a rejected version form.

## 5.1 Selected-schema translation tests

For every row in
[`gate-summary-schema-error-translation.md`](./gate-summary-schema-error-translation.md),
the suite generates a one-rule mutation from a known-valid fixture and
asserts the exact `GS_*` code and JSON Pointer path. Tests must:

- classify from `ValidationError.ErrorKind`, `InstanceLocation`, schema
  keyword location, and `Causes`, never from `Error()` text;
- fan out structured missing/additional property names deterministically;
- collapse the test-total `anyOf` subtree to one
  `GS_PARTIAL_TEST_TOTALS` diagnostic;
- cover the `GS_SCHEMA_VIOLATION` fallback;
- inject an impossible post-dispatch version `const` mismatch and expect
  `GS_INTERNAL`, never `GS_UNSUPPORTED_VERSION`.

## 6. One-mutation negatives

The contract ACT commits a representative subset of one-mutation
negatives; `CONFORMANCE01` extends the corpus to cover every row of
the compatibility matrix. The mutations fall into the following
categories:

- version discriminator (missing, string, decimal, exponent,
  unsupported, duplicate);
- required field missing;
- unknown field (v1 with v2-only field, v2 with v1-only field);
- exit-code / status mismatch;
- arithmetic mismatch;
- output-hash shape violation;
- worktree-cleanliness / closed-scope mismatch;
- duplicate check name;
- duplicate nested object key;
- resource limit breach (size, collection, string length);
- trailing second JSON value;
- lowercase lifecycle status.

## 7. Fuzz property test

The fuzz target `FuzzDecodeGateSummary` accepts arbitrary bytes and
asserts that the reader:

- never panics;
- always returns either (a) a normalized `Summary` or (b) a
  diagnostic list — never both, never neither;
- classifies accept/reject deterministically for a given input;
- emits no internal stack traces by default.

A fuzz-discovered regression must be promoted into a permanent
corpus entry under `testdata/fuzz-seed/`.

## 8. Determinism property

The test suite includes a property test that:

- reads every valid fixture twice;
- asserts the normalized `Summary` is byte-identical between the two
  reads;
- asserts the digest rendering is byte-identical between the two
  reads.

## 9. Fault-injection tests

`GS_NORMALIZATION_FAILURE` and internal `GS_INTERNAL` paths are not
ordinary invalid-input fixtures. `CONFORMANCE01` injects those failures,
including the impossible post-dispatch version mismatch, and asserts the
diagnostic codes are emitted without stack traces.

## 10. Schema validator validation

`CONFORMANCE01` runs the chosen Draft 2020-12 JSON Schema validator
against both schemas (`gate-summary-v1.schema.json`,
`gate-summary-v2.schema.json`) and confirms they parse as Draft
2020-12. The chosen validator is
[`santhosh-tekuri/jsonschema/v6`](./gate-summary-schema-validator-selection.md)
with `compiler.AssertFormat()` enabled.

## 11. Out of scope for `CONFORMANCE01`

- Producer conformance (a producer-test harness is in
  `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01`).
- Downstream conformance (the ClineMM fixture binding is in
  `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01`).

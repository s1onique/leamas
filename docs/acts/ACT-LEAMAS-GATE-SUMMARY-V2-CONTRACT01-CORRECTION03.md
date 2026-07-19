# ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03

## Title

Correct the remaining observable gate-summary reader-contract defects
before `DECODER01` begins.

## Parent

- Epic:
  [`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)
- Contract:
  [`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md)
- Prior correction:
  [`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md)

## Status

`CLOSED (PARTIAL — reader contract frozen and committed)`. The
implementation/freeze commit `7bf86c2` lands the contract freeze;
this ACT closes the lifecycle by reconciling the status banners.
Commit `507b686a4681eb28b97267b2ddab6c5bfc3ce7df` remains in history
and is the prior correction this ACT supersedes.

## Problem

The Git closure and selected-validator proof from `CORRECTION02` are
accepted, but four semantic blockers remain:

1. The inventory calls a hybrid 40-file subset “all committed JSON
   fixtures” even though the validator reviewed 41 files.
2. The version table treats malformed JSON number spellings and
   insignificant JSON whitespace as semantic version errors.
3. Unsupported integer versions are assigned to the selected JSON
   Schema instead of the pre-schema version dispatcher.
4. Only version-related schema-error translation is frozen; the rest of
   the validator's structured errors still lack stable `GS_*` mappings.

## Goal

Freeze one RFC 8259-compatible, implementation-ready reader contract so
`DECODER01` implements public behavior without inventing diagnostics.

## Scope

This is a documentation-and-test-design correction only. It:

- freezes the global fixture inventory as 41 / 38 / 3;
- records the separately named v2-only executable subset as 35;
- corrects malformed, whitespace, type, and unsupported-version
  ownership;
- freezes the complete selected-schema-error translation algorithm;
- defines generated lexical cases for `DECODER01` and `CONFORMANCE01`;
- reconciles the epic and prior correction records.

No production reader, schema, committed JSON fixture, dependency, digest,
CLI, producer, or normalization behavior changes in this ACT.

## Executable contract

### Stable boundary

The narrow boundary is raw JSON bytes entering the gate-summary reader:
syntax, duplicate-key detection, version probing and dispatch, selected
schema validation, and translation of structured validator failures into
stable diagnostics.

### Defect register

- **F1 — inventory boundary:** Global inventory is 7 valid + 28 invalid
  + 3 duplicate-key + 3 limit-shape = 41 JSON artifacts. The executable
  corpus is 38; limit templates are 3. The optional v2-only executable
  subset is 5 valid + 27 invalid + 3 duplicate-key = 35.
- **F2 — JSON grammar:** `02`, `-02`, and `+2` are malformed JSON and
  map to `GS_MALFORMED_JSON`. RFC 8259 whitespace around `1` or `2` is
  insignificant and does not alter dispatch.
- **F3 — version ownership:** Missing/type/lexical/value classification
  and supported-version dispatch happen before schema selection.
  Unsupported integer values map to `GS_UNSUPPORTED_VERSION` at the
  envelope dispatcher. Schema `const` remains defense in depth only.
- **F4 — schema translation:** Structured `ValidationError` data, never
  localized message strings, deterministically maps every selected-schema
  failure to a specific `GS_*` code or `GS_SCHEMA_VIOLATION` fallback.

### Declarative matrix

| Dimension | Input family | Expected owner/result |
| --------- | ------------ | --------------------- |
| JSON grammar | `02`, `-02`, `+2` | syntax scan → `GS_MALFORMED_JSON` |
| Insignificant whitespace | RFC 8259 whitespace before/after `1` or `2` | dispatch v1/v2 |
| Number form | decimal or exponent | version probe → `GS_INVALID_VERSION_TYPE` |
| Non-number type | string, Boolean, null, array, object | version probe → `GS_INVALID_VERSION_TYPE` |
| Integer value | integer other than 1 or 2 | version dispatch → `GS_UNSUPPORTED_VERSION` |
| Selected-schema error | structured keyword/location/cause | frozen schema-error translation table |

The generated lexical matrix is normative in
[`gate-summary-schema-version-translation.md`](../factory/gate-summary-schema-version-translation.md).
The schema-error matrix is normative in
[`gate-summary-schema-error-translation.md`](../factory/gate-summary-schema-error-translation.md).

### RED evidence

The review finding is the RED evidence for this docs-only correction. The
committed validator output already reports `invalid=28` and
`total fixtures reviewed=41`, while the inventory labels 40 files as all
fixtures. The version and translation documents also contain the F2–F4
contradictions above. There is no production decoder yet against which a
behavioral RED test can run.

### GREEN evidence

GREEN requires:

- all current contract authorities use 41 / 38 / 3 globally;
- generated lexical cases carry the corrected outcomes;
- version dispatch precedes selected-schema validation;
- every selected-schema failure has a structured deterministic mapping;
- required repository checks run with honest results;
- the correction is committed after `507b686a…`, without amend or reset.

### Executable-contract-first exception

- **Category:** docs-only / contract-correction-3.
- **Justification:** The reader does not exist yet; production belongs to
  `DECODER01`.
- **Verification:** Corpus counts, document searches, accepted validator
  proof, and repository checks.
- **Why no production RED test:** A decoder test would implement the
  downstream ACT inside this contract ACT. Generated cases are frozen now
  for implementation there.

## Acceptance criteria

- [x] Global inventory is exactly 41 committed JSON fixtures, 38
      executable accept/reject fixtures, and 3 limit-shape templates.
- [x] The only separately reported subset is clearly named “v2-only
      executable corpus” and equals 35.
- [x] `02`, `-02`, and `+2` map to `GS_MALFORMED_JSON`.
- [x] RFC 8259 whitespace around valid integer tokens is accepted.
- [x] All valid integer values other than 1 and 2 are owned by version
      dispatch and map to `GS_UNSUPPORTED_VERSION`.
- [x] Schema `const` is defense in depth and never owns an ordinary
      unsupported-version diagnostic.
- [x] The complete schema-error-to-`GS_*` translation table is frozen
      using structured validator data rather than message strings.
- [x] Generated whitespace, leading-zero, plus-sign, decimal, and
      exponent cases are normative for `DECODER01`/`CONFORMANCE01`.
- [x] No production code, schema, JSON fixture, or dependency changes.
- [x] `make factorize`, `make gate`, and applicable Go checks run and
      their exact outcomes are recorded in the close report.
- [x] A forward commit follows `507b686a…`; history is not rewritten.
- [x] Closure-only follow-up commit reconciles status banners and the
      epic ACT board; `DECODER01` becomes `READY`.

## Verification commands

```bash
find internal/gatesummary/testdata -type f -name '*.json'
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
make factorize
make gate
git diff --check
git diff --stat
git status --short
git log --oneline -2
```

## Follow-up

After this ACT closes, `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` may
implement the frozen pipeline and diagnostic translation without
reinterpreting the contract.

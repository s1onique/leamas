# ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01

## Title

Correct twelve defects found by the post-close review of
`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`. The contract is not yet
immutable; the architectural direction is sound but the spec, schemas,
fixtures, diagnostic registry, and close report contain concrete
inconsistencies that must be resolved before `DECODER01` can begin.

## Parent Epic

[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)

## Parent ACT

[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md)

## Status

`CLOSED (PARTIAL — superseded by CORRECTION02)`. The original ACT
reported itself closed but the review identified twelve concrete
defects. `CORRECTION01` resolved those twelve defects but left
eleven follow-up items, which `CORRECTION02` resolved in turn. The
contract freeze is now internally consistent and committed as a
single honest revision; see the `CORRECTION02` close report for
the merged evidence trail.

## Problem

The original close report claimed the contract was closed and ready
for downstream ACTs. The post-close review identified defects that
fall into four classes:

1. **Internal contradictions**: the ClineMM µC-3 fixture and its
   derivation rule disagree; diagnostic ordering is defined twice in
   mutually inconsistent ways; fixture counts are arithmetically
   wrong.
2. **Unproven claims**: the spec and close report assert JSON Schema
   behavior that is not what a Draft 2020-12 validator actually does
   (`2.0` for `type: integer`, default `format` assertion).
3. **Architectural drift**: lifecycle wire format, output-hash
   format, and `parent_checks` are not exactly what the epic body
   specifies; the v1 schema retroactively adds limits the original
   reader did not enforce.
4. **Closure evidence gap**: full `go test ./...`, `make factorize`,
   and `make gate` were not green; the close report acknowledged
   this but did not down-classify the closure to PARTIAL.

## Goal

Resolve all twelve defects in a single, atomic revision so that the
contract becomes internally consistent, the spec matches what a real
Draft 2020-12 validator does, the fixtures are correctly counted and
correctly classified, and the close report records honest PARTIAL
closure with retained baseline failures.

## Scope

This ACT modifies the spec documents, schemas, fixtures, and the
original close report that were delivered by `CONTRACT01`. It does
**not** add production reader code, digest renderer code, CLI
commands, or new producer output.

In scope:

- `docs/factory/gate-summary-v1-spec.md` — remove unproven v1
  schema restrictions.
- `docs/factory/gate-summary-v2-spec.md` — separate
  `CheckStatus` and `OverallStatus` vocabularies; move lexical
  version validation outside JSON Schema; require uppercase lifecycle
  at the wire boundary; require 64-char lowercase hex for output
  hashes; remove `parent_checks`.
- `docs/factory/gate-summary-vocabularies.md` — wire/normalized
  lifecycle form pinned to uppercase-wire/lowercase-normalized;
  explicit all-skipped derivation result.
- `docs/factory/gate-summary-diagnostic-codes.md` — unified
  diagnostic ordering (precedence rank, then path, then encounter
  index); per-code fixture classification.
- `docs/factory/gate-summary-compatibility-matrix.md` — same
  diagnostic ordering; corrected fixture expectation list; expanded
  overall-status mismatch coverage.
- `docs/factory/gate-summary-schema-validator-selection.md` — pin
  `github.com/santhosh-tekuri/jsonschema/v6` (latest v6.x release);
  freeze `AssertFormat` behavior.
- `internal/gatesummary/schema/gate-summary-v1.schema.json` —
  remove `maxItems`, `maxLength`, `minimum` constraints not in the
  original reader.
- `internal/gatesummary/schema/gate-summary-v2.schema.json` —
  uppercase lifecycle only; require 64-char lowercase hex for both
  hash fields; remove `parent_checks`.
- `internal/gatesummary/testdata/valid/v2-clinemm-microc3.json` —
  add a failing parent-state check that contributes to the
  aggregate.
- `internal/gatesummary/testdata/limits/v2-checks-{max,over-max}.json`
  renamed to honest names.
- `internal/gatesummary/testdata/invalid/v2-schema-version-decimal.json`
  and friends — clarified as "lexical not schema" cases.
- New fixtures for missing diagnostic coverage
  (`v2-bad-status-enum.json`, `v2-overall-mismatch.json`,
  `v2-truncated.json`, etc.).
- `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md` —
  rewritten to record the corrected contract and the PARTIAL
  classification.

## Non-goals

- No production reader code change (still `DECODER01`'s job).
- No digest renderer change (still `DIGEST01`'s job).
- No CLI command addition (still `CLI01`'s job).
- No new producer output.
- No removal of v1 support.
- No change to the epic body itself; corrections stay inside
  Leamas-owned documents.

## Executable contract

### Stable boundary

The stable boundary is the public gate-summary evidence contract
owned by Leamas: the v1 and v2 wire shapes, the v2 derived
semantics, the diagnostic code registry, the resource limits, the
JSON Schema definitions, the fixture corpus, and the chosen JSON
Schema validator. This ACT repairs twelve defects in that boundary
so that subsequent ACTs (`DECODER01` and downstream) can rely on
it as immutable.

### Defect register

The twelve defects are recorded below as a bulleted list. Each entry
carries the defect ID, the symptom, and the fix.

- **D1**: ClineMM µC-3 fixture has `overall_status=fail` but only pass checks; the derivation rule derives `pass` from all-pass. Fix: add a failing parent-state `check` to the ClineMM fixture whose `scope` equals the parent ACT.
- **D2**: JSON Schema 2020-12 treats `2` and `2.0` as the same integer literal, so `type: integer, const: 2` does not reject `2.0`. Fix: move lexical version validation to the pre-schema envelope scanner using `json.Number`.
- **D3**: `format: date-time` is not asserted by default in `santhosh-tekuri/jsonschema/v6`. Fix: freeze `compiler.AssertFormat()` in the selection memo. Pin the exact major version `v6`.
- **D4**: Close report says "32-fixture corpus" but `7+23+3+2=35`. The scratch validator ran on 32 fixtures. Fix: correct the count to 35; document the scratch-validator scope honestly.
- **D5**: Nine diagnostic codes lack fixtures. Fix: add fixtures for `GS_INVALID_STATUS`, `GS_OVERALL_STATUS_MISMATCH`, `GS_DOCUMENT_TOO_LARGE`. Mark two as fault-injection.
- **D6**: Diagnostic registry and compatibility matrix use two different ordering algorithms. Fix: unify on `precedence rank, then path, then encounter index`. Update both documents.
- **D7**: v1 schema adds `maxItems=10000`, `maxLength=1024`, `minimum=0` constraints that the original reader does not enforce. Fix: strip the new constraints from the v1 schema.
- **D8a**: Lifecycle wire format accepts both `OPEN` and `open`. Fix: restrict schema to uppercase only.
- **D8b**: Output hashes may be empty. Fix: require 64 lowercase hex; remove empty-string option.
- **D8c**: `parent_checks` is redundant if producers record parent requirements as ordinary `checks` with `scope=parent_act`. Fix: remove `parent_checks` from the v2 schema.
- **D9**: `v2-checks-over-max.json` has one check but is named "over-max". Fix: rename to `v2-checks-{boundary,over-boundary}-shape.json`.
- **D10**: Close report claims closure but full gates are not green. Fix: reclassify closure as `PARTIAL` with retained baseline failures.
- **D11**: Schemas were not validated against the chosen Draft 2020-12 validator. Fix: run the chosen validator against the schemas and document the result.

### RED evidence

The defects above are discovered by the post-close review of
`CONTRACT01`. Each defect is one that the
"executable-contract-first" doctrine explicitly forbids:

- D1 is a fixture-vs-rule contradiction.
- D2 and D3 are false claims about the chosen tool's behavior.
- D4 is an arithmetic error in evidence.
- D5 is a coverage claim that does not match the corpus.
- D6 is two definitions of the same thing.
- D7 is a behavioral change hidden under a "frozen reconstruction"
  claim.
- D8a-c are architectural decisions recorded inconsistently with
  the epic body.
- D9 is misleading fixture naming.
- D10 is the doctrine's "no claim of success for skipped checks."
- D11 is the doctrine's "do not invent command outputs."

This ACT is itself a RED step: it documents the defects before
fixing them.

### GREEN evidence

The GREEN step is the resolution of all twelve defects plus a
re-classified PARTIAL close with the chosen JSON Schema validator
actually running on the schemas.

Focused commands:

```bash
# 1. Validate every schema using the selected validator (offline).
# See: docs/factory/gate-summary-schema-validator-selection.md.

# 2. Re-run the scratch structural validator on every fixture.
# Expected: 37 executable fixtures reviewed (7 valid + 27 invalid + 3 duplicate-key).
# Plus 3 limit-shape templates (well-formed structural shapes).
# Total committed JSON fixture artifacts: 40.
# Note: `CORRECTION02` reconciles these numbers against the manifest
# (7+27+3+3=40). Removing only `v2-document-size-shape.json` gives 39
# artifacts, not 37. Excluding all three limit-shape templates gives
# 37 executable accept/reject fixtures.

# 3. Focused Go tests on the unchanged packages.
go test ./internal/factory/gate/...
go test ./internal/factory/digest/...
go test ./cmd/leamas/...

# 4. Repository gates (PARTIAL with documented baseline failures).
make factorize
make gate
```

Affected subsystem command: the same as focused.

Repository gate command: `make factorize` and `make gate`. PARTIAL
acceptance requires documenting baseline failures.

### Exceptions

| Category | Justification | Verification | Why no regression test |
| -------- | ------------- | ------------ | ---------------------- |
| docs-only / contract-correction | Docs, schema, fixture correction only. | `go vet`, `go build`, the JSON Schema validator, and the scratch validator. | Real tests live in `CONFORMANCE01` and `DECODER01`. |

## Acceptance Criteria

- [ ] `gate-summary-v2-spec.md` separates `CheckStatus` and
      `OverallStatus`, requires uppercase lifecycle wire, requires
      64-char lowercase hex hashes, and removes `parent_checks`.
- [ ] `gate-summary-v2-spec.md` records that lexical
      `schema_version` validation happens in the pre-schema envelope
      scanner, not in JSON Schema.
- [ ] `gate-summary-v1-spec.md` and `gate-summary-v1.schema.json`
      match the original v1 reader byte-for-byte (no new
      constraints).
- [ ] `gate-summary-vocabularies.md` pins uppercase-wire /
      lowercase-normalized lifecycle and explicit all-skipped
      derivation result.
- [ ] `gate-summary-diagnostic-codes.md` and
      `gate-summary-compatibility-matrix.md` use one unified
      ordering algorithm.
- [ ] `gate-summary-schema-validator-selection.md` pins
      `github.com/santhosh-tekuri/jsonschema/v6` and freezes
      `compiler.AssertFormat()`.
- [ ] `v2-clinemm-microc3.json` has at least one failing
      parent-state check that contributes to `overall_status=fail`.
- [ ] New fixtures cover `GS_INVALID_STATUS`,
      `GS_OVERALL_STATUS_MISMATCH`, `GS_DOCUMENT_TOO_LARGE`,
      `GS_DUPLICATE_KEY` (lexical).
- [ ] `GS_NORMALIZATION_FAILURE` and `GS_INTERNAL` are explicitly
      marked as test-only fault-injection codes, not fixture-driven.
- [ ] `GS_SCHEMA_VIOLATION` is documented as the umbrella code when
      no specific code applies.
- [ ] Limit fixtures are renamed honestly.
- [ ] The chosen JSON Schema validator runs against both schemas and
      the result is recorded.
- [ ] The scratch structural validator runs against all 37
      executable fixtures (7 valid + 27 invalid + 3 duplicate-key)
      plus the 3 limit-shape templates (40 total artifacts) and
      the result is recorded.
- [ ] The close report is rewritten to record PARTIAL closure with
      retained baseline failures, the corrected count of 35
      fixtures, and the actual schema-validation result.
- [ ] No file under `internal/gatesummary/` exceeds 64 KiB or 400
      lines.
- [ ] `go vet ./...`, `go build ./cmd/leamas` remain green.

## Verification Commands

```bash
# Build, vet, and tests on the affected subsystems
go vet ./...
go build -o /tmp/leamas-build ./cmd/leamas
go test ./internal/factory/gate/...
go test ./internal/factory/digest/...
go test ./cmd/leamas/...

# LLM-friendliness gate on the new files
/tmp/leamas-build factory verify llm-friendly

# Full scratch structural validator
/tmp/validate-fixtures

# Selected JSON Schema validator (offline; see selection memo)
# See docs/factory/gate-summary-schema-validator-selection.md.

# Repository gates (PARTIAL, with documented baseline failures)
make factorize
make gate

# Git hygiene
git diff --check
```

## Reviewer Focus

- **Architectural correctness.** The ClineMM fixture is the
  canary; verify that adding a failing parent-state check produces
  an `overall_status=fail` that the derivation rule independently
  confirms.
- **JSON Schema vs. envelope scanner.** The spec must be honest
  about which rule is enforced where. JSON Schema is structural;
  lexical-number, duplicate-key, and trailing-JSON detection live
  in the envelope scanner.
- **Diagnostic coverage.** Every code listed in the registry must
  either (a) have a fixture, or (b) be marked as test-only fault
  injection.
- **Fixture honesty.** Names must not lie about their content.
- **Closure honesty.** The close report must say PARTIAL, not
  CLOSED, until the gates are green.

## Close Report Stub

> **Summary:** Correct twelve defects in `CONTRACT01` so the
> contract becomes internally consistent, the spec matches what
> the chosen Draft 2020-12 validator actually does, and the close
> report records honest PARTIAL closure.
> **Files changed:** see close report.
> **Behavior changed:** none in production; the v1 schema is
> narrowed to its original reader's behavior; the v2 schema is
> narrowed (uppercase lifecycle only, exact 64-char hashes, no
> `parent_checks`).
> **Verification:** chosen offline JSON Schema validator runs;
> scratch structural validator passes; `go vet`, `go build`,
> focused tests remain green.
> **Closure:** PARTIAL — full `make factorize` / `make gate` /
> `go test ./...` retain documented baseline failures from
> pre-existing ACTs.

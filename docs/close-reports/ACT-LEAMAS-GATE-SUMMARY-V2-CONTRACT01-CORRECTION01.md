# Close Report: ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01

## ACT Reference

[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01`](../acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md)
(parent epic: [`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md))

## Summary

Corrected twelve defects in `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01` so
the gate-summary v1/v2 contract is internally consistent, the
specification matches what a real Draft 2020-12 validator does, and
the close report records honest PARTIAL closure with retained
baseline failures. No production reader, digest renderer, producer
output, or downstream evidence behavior was changed.

## Status

`PARTIAL — superseded by CORRECTION03`. The later validator/Git proof is
accepted; CORRECTION03 owns the final reader semantics and inventory.

## Files Changed

The complete `CONTRACT01-CORRECTION01` deliverable set is recorded
below. Every file in this list was added or modified by
`CONTRACT01-CORRECTION01`.

| File | Status |
|------|--------|
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | added |
| `docs/factory/gate-summary-v1-spec.md` | rewritten — v1 schema narrowed to original reader behavior |
| `docs/factory/gate-summary-v2-spec.md` | rewritten — uppercase lifecycle, exact 64-char hashes, lexical envelope scanner, no `parent_checks` |
| `docs/factory/gate-summary-vocabularies.md` | rewritten — uppercase-wire/lifecycle-only, all-skipped derivation |
| `docs/factory/gate-summary-diagnostic-codes.md` | rewritten — unified ordering, fault-injection codes marked |
| `docs/factory/gate-summary-compatibility-matrix.md` | rewritten — unified ordering, expanded mismatch coverage |
| `docs/factory/gate-summary-schema-validator-selection.md` | rewritten — pinned to `v6`, `AssertFormat` frozen, lexical rules noted as envelope scanner job |
| `docs/factory/gate-summary-conformance-test-design.md` | rewritten — fault-injection tests added |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md` | rewritten — PARTIAL closure classification, defect table |
| `internal/gatesummary/schema/gate-summary-v1.schema.json` | narrowed — no new collection/string limits |
| `internal/gatesummary/schema/gate-summary-v2.schema.json` | narrowed — uppercase lifecycle only, exact 64-char hashes, no `parent_checks` |
| `internal/gatesummary/testdata/README.md` | rewritten by CORRECTION01; final global inventory reconciled by CORRECTION03 to 41 = 7 valid + 28 invalid + 3 duplicate-key + 3 limit-shape |
| `internal/gatesummary/testdata/limits/README.md` | rewritten — boundary semantics clarified |
| `internal/gatesummary/testdata/valid/v2-minimal.json` | hash normalization |
| `internal/gatesummary/testdata/valid/v2-full.json` | hash normalization; `parent_checks` removed; added parent-state check |
| `internal/gatesummary/testdata/valid/v2-root-scope.json` | hash normalization |
| `internal/gatesummary/testdata/valid/v2-clinemm-microc3.json` | **canary fix**: added `parent_production_bundle` failing check so derivation yields `fail` |
| `internal/gatesummary/testdata/valid/v2-leamas-self-hosted.json` | hash normalization |
| `internal/gatesummary/testdata/invalid/v2-*.json` (24 files) | hash normalization; new fixtures `v2-bad-status-enum.json`, `v2-overall-mismatch.json`, `v2-lower-lifecycle.json`, `v2-document-too-large.json`, `v2-truncated.json` added |
| `internal/gatesummary/testdata/invalid/v2-document-too-large.json` | **moved** to `limits/v2-document-size-shape.json` by CORRECTION02 (E6) |
| `internal/gatesummary/testdata/limits/v2-document-size-shape.json` | **added** by CORRECTION02 (E6) |
| `internal/gatesummary/testdata/invalid/v2-schema-version-negative.json` | **added** by CORRECTION02 (E8) for negative-integer conformance case |
| `internal/gatesummary/testdata/duplicate-keys/*.json` (3 files) | hash normalization |
| `internal/gatesummary/testdata/limits/v2-checks-{max,over-max}.json` | renamed to `v2-checks-{boundary,over-boundary}-shape.json` |

## Behavior Changed

None in production. The v1 schema is narrowed back to the original
reader behavior; the v2 schema is narrowed (uppercase lifecycle
only, exact 64-char hashes, no `parent_checks`). Production reader
behavior, digest renderer behavior, producer output, and downstream
evidence remain unchanged.

## Defect Resolution

The twelve defects from the post-close review are all resolved:

| # | Defect | Status | Resolution |
| - | ------ | ------ | ---------- |
| D1 | ClineMM µC-3 fixture contradicts derivation rule | resolved | `parent_production_bundle` failing check added; derivation yields `fail` matching recorded. |
| D2 | `2.0` rejected via `type: integer, const: 2` (false claim) | resolved | Lexical version validation moved to the pre-schema envelope scanner using `json.Number`. |
| D3 | `format: date-time` assumed asserted by default | resolved | `compiler.AssertFormat()` policy frozen; `santhosh-tekuri/jsonschema/v6` pinned. |
| D4 | Fixture count claimed 32 but actually 35 | superseded by CORRECTION03 | Global inventory is 41 artifacts (38 executable + 3 limit-shape); v2-only executable subset is 35. |
| D5 | Nine diagnostic codes lack fixtures | resolved | Ordinary-input codes use fixtures/generated cases; `GS_NORMALIZATION_FAILURE` and internal `GS_INTERNAL` paths use fault injection. |
| D6 | Diagnostic ordering defined twice | resolved | Unified on `precedence rank, then path, then encounter index`. |
| D7 | v1 schema adds limits not in original reader | resolved | v1 schema stripped back to original reader behavior. |
| D8a | Lifecycle wire accepts both cases | resolved | Schema restricted to uppercase only. |
| D8b | Output hashes may be empty | resolved | Required to be exactly 64 lowercase hex; empty stream → SHA-256 of empty. |
| D8c | `parent_checks` was a separate array | resolved | Removed; producers record parent-state observations as ordinary `checks` with `scope=parent_act`. |
| D9 | Limit fixtures misleadingly named | resolved | Renamed to `v2-checks-{boundary,over-boundary}-shape.json`. |
| D10 | Closure claimed but full gates not green | resolved | Closure reclassified as `PARTIAL` with retained baseline failures. |
| D11 | Schemas not validated against chosen Draft 2020-12 validator | resolved by CORRECTION02 | Validator v6.0.2 proof with `AssertFormat()` is recorded and accepted. |

## Verification

### Commands Run

```bash
# Build, vet, and LLM-friendliness
go build -o /tmp/leamas-build ./cmd/leamas
go vet ./...
/tmp/leamas-build factory verify llm-friendly

# Offline scratch validator on every fixture
/tmp/validate-fixtures
```

### Results

- `go vet ./...` passes.
- `go build ./cmd/leamas` succeeds.
- All ACT-deliverable files pass the LLM-friendliness gate. The
  pre-existing `digest-contract.md` long-file violation is not in
  this ACT's scope.
- All 37 fixtures pass the offline structural review. The ClineMM
  µC-3 fixture is the canary: its `parent_production_bundle`
  failing check makes the recorded `overall_status=fail` consistent
  with the derivation rule.

### Scratch Validator Output

```text
PASS  v1-minimal             want=accept
PASS  v1-full                want=accept
PASS  v1-unknown-field       want=reject-unknown-field
PASS  v2-minimal             want=accept
PASS  v2-full                want=accept
PASS  v2-root-scope          want=accept
PASS  v2-clinemm-microc3     want=accept-fail-derived
PASS  v2-leamas-self-hosted  want=accept
... 25 invalid + 3 duplicate + 2 limit fixtures all PASS

37 passed, 0 failed
```

### Skipped checks

- `make factorize`, `make gate`, and a full `go test ./...` retain
  two pre-existing baseline failures in files **not modified by this
  ACT**: `internal/factory/digest/digest_test_helpers_test.go`
  (forbidden `os/exec.Command`) and `docs/factory/digest-contract.md`
  (420 lines). These are tracked by their owning ACTs.

## Decisions Made

- **JSON Schema validator selection:**
  `github.com/santhosh-tekuri/jsonschema/v6`. Pinned in
  [`gate-summary-schema-validator-selection.md`](../factory/gate-summary-schema-validator-selection.md).
- **`AssertFormat` policy:** Frozen as opt-in for all schema
  validations.
- **Lexical version validation:** Lives in the pre-schema envelope
  scanner using `json.Number`, not in JSON Schema.
- **Output hashes:** Always 64 lowercase hex; empty stream →
  `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
- **Lifecycle wire format:** Uppercase only; lowercase rejected.
- **Parent-state observations:** Recorded as ordinary `checks[]`
  entries with `scope=parent_act`. The dedicated `parent_checks`
  array was removed.
- **Historical CORRECTION01 scratch set:** 37 total. The final global
  inventory is 41 / 38 / 3 under CORRECTION03.

## Agent Doctrine Impact

None. This ACT adds no agent-facing doctrine and no verifier
behavior. The CLI surface, `leamas factory verify` commands, and
`scripts/verify_*.sh` scripts are unchanged.

## Open Questions

- The selected-validator proof was completed by CORRECTION02.
  `DECODER01` still owns production wiring.
- The boundary-shape fixtures under `testdata/limits/` are small
  well-formed documents. Actual numeric boundary tests are
  programmatic in `CONFORMANCE01`.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | Wire the chosen JSON Schema validator, add bounded reader, lexical envelope scanner, strict v1/v2 decoders, resource-limit enforcement. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | Build the normalized `Summary` domain; add v1/v2 normalizers and semantic validators. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | Render v2 scope, parent, and aggregate status independently in the targeted digest. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | Add `validate`, `inspect`, `normalize` subcommands with the documented exit-code contract. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | Frozen v1/v2 goldens, mutation corpus, duplicate-key corpus, limit tests, fuzz seed corpus, schema-vs-Go-type tests, fault injection, validator validation. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01` | Leamas self-hosted v2 summary, ClineMM v2 producer consumption, downstream evidence rebinding. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01` | Provisional 0.2.0 release with compatibility matrix, producer and consumer guides, diagnostic-code guide. | P0 |

## Notes

- The scratch validator under `/tmp/validate-gate-summary-fixtures/`
  is intentionally not committed to the repository. The committed
  conformance tests live in `CONFORMANCE01`.
- The ClineMM µC-3 fixture is the canary for the entire v2
  architecture: a closed child scope with a failing
  parent-production-bundle check produces an `overall_status=fail`
  that the derivation rule independently confirms.
- The `parent_checks` array was removed from v2 because
  parent-state observations can be recorded as ordinary `checks`
  with `scope=parent_act`, and including them in the regular check
  list ensures they participate in `overall_status` derivation.

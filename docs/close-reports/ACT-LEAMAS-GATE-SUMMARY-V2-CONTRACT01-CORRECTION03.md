# Close Report: ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03

## ACT Reference

[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`](../acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03.md)
(parents:
[`CONTRACT01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md),
[`CONTRACT01-CORRECTION01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md),
[`CONTRACT01-CORRECTION02`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md);
epic:
[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md))

## Summary

Resolves the four reader-contract blockers found by the post-
`CORRECTION02` review. The frozen contract now satisfies RFC 8259,
owns every `schema_version` form at the correct stage, and provides
a complete structured mapping from
`santhosh-tekuri/jsonschema/v6` errors to stable `GS_*` codes.
`ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` may now implement the
pipeline and translation table without reinterpreting any document.

## Status

`CLOSED (PARTIAL)`. The contract freeze is internally consistent.
`make factorize` and `make gate` retain pre-existing baseline
failures owned by other ACTs; this ACT did not introduce new
failures. `DECODER01` becomes `READY` on the epic ACT board.

## Files Changed

| File | Change |
|------|--------|
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03.md` | added — defect register, declarative matrix, generated-case contract |
| `docs/factory/gate-summary-schema-error-translation.md` | added — complete structured schema-error mapping, kind traversal, fanout, ordering |
| `docs/factory/gate-summary-schema-version-translation.md` | rewritten — RFC 8259 grammar, pre-schema ownership, generated lexical matrix |
| `docs/factory/gate-summary-compatibility-matrix.md` | rewritten — global 41 / 38 / 3 inventory, JSON grammar rows, fixture/generated contract |
| `docs/factory/gate-summary-v2-spec.md` | rewritten §4 and §17 — version ownership and decoder strictness re-aligned with CORRECTION03 |
| `docs/factory/gate-summary-diagnostic-codes.md` | rewritten — `GS_INVALID_VERSION_TYPE` and `GS_UNSUPPORTED_VERSION` re-owned; `GS_INTERNAL` is the impossible post-dispatch result |
| `docs/factory/gate-summary-conformance-test-design.md` | rewritten — added §5.1 selected-schema translation tests; lexical suite is programmatic |
| `docs/factory/gate-summary-schema-validator-selection.md` | rewritten — version/envelope rules precede schema; structured error translation is binding |
| `docs/factory/gate-summary-v1-spec.md` | status banner updated |
| `docs/factory/gate-summary-vocabularies.md` | status banner updated |
| `docs/factory/gate-summary-resource-limits.md` | status banner updated |
| `internal/gatesummary/testdata/README.md` | rewritten — totals 41 / 38 / 3 with the v2-only 35 subset named |
| `internal/gatesummary/testdata/limits/README.md` | status banner updated |
| `docs/epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md` | ACT board reconciled: `CORRECTION02` is `CLOSED (PARTIAL — superseded)`; `CORRECTION03` is `IN PROGRESS`; `DECODER01` is `BLOCKED` until this ACT closes. |
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md` | status text now records the forward supersession to `CORRECTION03` |
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | status text now records the forward supersession to `CORRECTION03` |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md` | status banner points at `CORRECTION03` |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | status updated; `D4` and `D11` references point at the final authority |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md` | summary, defect table, and inventory tables rewritten to honour the global 41 / 38 / 3 boundary and the final translation docs |

No committed JSON fixture, no committed schema, no `go.mod` /
`go.sum`, and no production code changes.

## Defect Resolution

| # | Defect | Status | Resolution |
| - | ------ | ------ | ---------- |
| F1 | Hybrid 40-file subset mislabeled as `all committed fixtures` | resolved | Global inventory is 41 / 38 / 3. The v2-only executable subset is 35, named. |
| F2 | `02`, `-02`, `+2` mapped to `GS_INVALID_VERSION_TYPE`; whitespace rejected | resolved | `02`, `-02`, and `+2` are malformed JSON and map to `GS_MALFORMED_JSON`; RFC 8259 whitespace around a valid `1`/`2` token is accepted. |
| F3 | Unsupported integer versions assigned to the selected JSON Schema | resolved | Pre-schema dispatcher owns every valid integer other than 1 and 2; `GS_UNSUPPORTED_VERSION` is the public code. Schema `const` is defense in depth only. |
| F4 | Only the version-related translation was frozen | resolved | `gate-summary-schema-error-translation.md` freezes the complete structured mapping using `ValidationError.ErrorKind`, `InstanceLocation`, `SchemaURL`, and `Causes`. |

## Fixture Inventory Contract

The global corpus is now frozen to the numbers the validator already
reviewed. `find internal/gatesummary/testdata -type f -name '*.json'`
returns:

| Family | Count | Role |
| ------ | ----- | ---- |
| `valid/` | 7 | full-accept corpus (2 v1 + 5 v2) |
| `invalid/` | 28 | full-reject corpus (1 v1 + 27 v2) |
| `duplicate-keys/` | 3 | lexical rejection corpus (all v2) |
| `limits/` | 3 | static-shape templates only |
| **All committed JSON fixtures** | **41** | 7 + 28 + 3 + 3 |

```text
all committed JSON fixtures     = 41
executable accept/reject corpus = 38
limit-shape templates           = 3
v2-only executable subset       = 35
```

The 35-file v2-only subset is exactly 5 valid v2 + 27 invalid v2 +
3 duplicate-key v2.

## Verification

### Commands Run

```bash
# Inventory
find internal/gatesummary/testdata -type f -name '*.json'

# Go surface
go vet ./...
CGO_ENABLED=0 go build -trimpath -o /tmp/leamas-build ./cmd/leamas

# Focused tests on packages outside the existing baseline failures
go test -count=1 -timeout 60s ./internal/factory/llmfriendly/... \
                                  ./internal/factory/execgate/... \
                                  ./internal/factory/digest/...

# LLM-friendliness on contract docs
/tmp/leamas-build factory verify llm-friendly

# Selected validator proof (v6.0.2 with AssertFormat()) - re-run
go -C /tmp/validate-gate-summary-v6 run .

# Generated lexical matrix proof (raw JSON bytes, 123 cases)
/tmp/gate-summary-correction03-lexical-binary

# Schema-error tree inspection (v2 partial-test-totals fixture)
go -C /tmp/validate-gate-summary-v6 run ./inspect

# Stale-contract search
rg -n '40 / 37 / 3|40/37/3|40 fixture artifacts|40 committed JSON' \
    docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/factory/gate-summary-*.md \
    internal/gatesummary/testdata/README.md
rg -n 'v2 manifest|02.*GS_INVALID_VERSION_TYPE|\+2.*GS_INVALID_VERSION_TYPE' \
    docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/factory/gate-summary-*.md \
    internal/gatesummary/testdata/README.md
rg -n 'structural.*GS_UNSUPPORTED_VERSION|GS_UNSUPPORTED_VERSION.*structural' \
    docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01* \
    docs/factory/gate-summary-*.md

# Git hygiene
git diff --check
git diff --stat
git status --short
```

### Results

```text
fixture inventory  = 41 JSON artifacts (38 executable + 3 limit-shape)
v2-only subset     = 35
v1 schema compile  = PASS
v2 schema compile  = PASS
v1 schema (AssertFormat) = PASS
v2 schema (AssertFormat) = PASS
selected validator per-family tallies add to 41
selected validator structural-layer consistency = PASS
lexical proof generated = 100 whitespace + 6 leading-zero/plus +
                          9 decimal/exponent + 8 unsupported-integer
schema-error tree shows kind.Schema wrapper + kind.Reference +
                       kind.AnyOf + kind.Required + kind.Not
LLM-friendliness    = all changed contract files within thresholds
stale-contract search = no matches
stale-claim search   = no fixtures or schemas changed
gate package tests  = pre-existing baseline failure, unchanged
factorize / gate    = pre-existing baseline failure, unchanged
```

### Validator proof (recorded verbatim)

The `CORRECTION02` validator proof was re-run unchanged:

```text
validator-library=github.com/santhosh-tekuri/jsonschema/v6
validator-version=v6.0.2
AssertFormat=enabled

v1 schema compile=PASS
v2 schema compile=PASS
v1 compile (post-AssertFormat)=PASS
v2 compile (post-AssertFormat)=PASS

per-family counts:
  duplicate-keys     = 3
  invalid            = 28
  limits             = 3
  valid              = 7

total fixtures reviewed  = 41
accepted by JSON Schema  = 21
rejected by JSON Schema  = 20

structural-layer consistency: PASS
VALIDATOR_PROOF=PASS
```

The reader semantics change: ordinary unsupported versions and
whitespace edge cases are now classified at the pre-schema stages
documented in `gate-summary-schema-version-translation.md`. The
JSON Schema proof above is necessary but no longer sufficient
evidence on its own.

### Lexical matrix proof (recorded verbatim)

```text
whitespace-generated=100 PASS
leading-zero-plus-generated=6 PASS
decimal-exponent-generated=9 PASS
unsupported-integer-generated=8 PASS
LEXICAL_PROOF=PASS
```

The proof synthesizes raw JSON bytes for every case in
`gate-summary-schema-version-translation.md` §4 and classifies
them with the same algorithm `DECODER01` will implement. It
demonstrates that JSON parser error, `json.Number.String()` integer
match, and dispatch all work as specified for all 123 generated
cases. The cases are not committed JSON fixtures; they are
programmatic generators per the contract.

### Schema-error tree (partial test totals)

The selected validator returns a `kind.Schema` wrapper at the root
with one `kind.Reference` cause that wraps the test-total `kind.AnyOf`
subtree, which contains the expected `kind.Required` and
`kind.Not` children. This matches the translation rule that emits
exactly one `GS_PARTIAL_TEST_TOTALS` diagnostic at `/checks/0` and
suppresses the nested `required` and `not` causes:

```text
depth=0 kind=*kind.Schema schema=…# instance=[] keyword=[] causes=1
depth=1 kind=*kind.Reference schema=…#/properties/checks/items
  instance=[checks 0] keyword=[$ref] causes=1
depth=2 kind=*kind.AnyOf schema=…#/$defs/check instance=[checks 0]
  keyword=[anyOf] causes=2
depth=3 kind=*kind.Required schema=…#/$defs/check/anyOf/0
  instance=[checks 0] keyword=[required] causes=0
depth=3 kind=*kind.Not schema=…#/$defs/check/anyOf/1
  instance=[checks 0] keyword=[] causes=0
```

`kind.Not` correctly returns no `KeywordPath()`; the translation
must treat it as the literal token `not` when computing identity.

### Stale-claim search

`rg` for any of the obsolete numberings or mappings returns no
matches:

```text
stale-contract-search=PASS
```

### Skipped checks

- `go test ./internal/factory/gate/...` times out at 120s in
  `TestRunFactorize` against a pre-existing
  `dupcode.CheckReport` panic, not introduced by this ACT.
- `make factorize` and `make gate` therefore fail the same way
  they failed before `CORRECTION02`. Both failures are owned by
  their own ACTs and were already documented in the
  `CORRECTION02` close report.
- No new baseline failure was introduced by `CORRECTION03`.
- `go test ./internal/factory/llmfriendly/...` and
  `go test ./internal/factory/digest/...` pass.

## Decisions Made

- **Global inventory:** 41 / 38 / 3 with a named v2-only subset of 35.
- **JSON grammar:** `02`, `-02`, and `+2` are malformed JSON; whitespace
  around valid `1`/`2` is insignificant.
- **Version ownership:** Pre-schema dispatcher owns every
  `schema_version` form; schema `const` is defense in depth only.
- **Schema-error translation:** Uses only structured
  `ValidationError` fields. Localized text is never parsed.
- **Test-total collapse:** A `kind.AnyOf` whose `SchemaURL` ends with
  `/$defs/check/anyOf` emits one `GS_PARTIAL_TEST_TOTALS` and
  suppresses nested causes.
- **Generated lexical cases:** 100 whitespace + 6 leading-zero/plus
  + 9 decimal/exponent + 8 unsupported-integer cases are normative
  for `DECODER01` and `CONFORMANCE01`.

## Agent Doctrine Impact

None. This ACT adds no agent-facing doctrine and no verifier
behavior. The CLI surface, `leamas factory verify` commands, and
`scripts/verify_*.sh` scripts are unchanged. The chosen validator
wiring into production remains `DECODER01`'s job.

## Open Items

- `make factorize` / `make gate` / `go test ./internal/factory/gate/...`
  still retain pre-existing baseline failures owned by other ACTs.
- Production wiring of the frozen pipeline and translation table
  is `DECODER01`'s responsibility. The test design and generated
  cases are normative for `CONFORMANCE01`.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | Implement the frozen bounded reader, envelope/version dispatch, v1/v2 decoders, resource limits, and complete schema-error translation. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | Build the normalized `Summary` domain; convert wire `OPEN/PARTIAL/CLOSED` to normalized `open/partial/closed`. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | Render v2 scope, parent, and aggregate status independently in the targeted digest. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | Add `validate`, `inspect`, `normalize` subcommands with the documented exit-code contract. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | Encode every generated lexical case and the complete structured schema-error translation table; run the chosen validator with `AssertFormat()`. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01` | Leamas self-hosted v2 summary, ClineMM v2 producer consumption, downstream evidence rebinding. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01` | Provisional 0.2.0 release with compatibility matrix, producer and consumer guides, diagnostic-code guide. | P0 |

## Notes

- The CORRECTION02 validator proof and the CORRECTION03 lexical
  proof together establish the public reader contract.
- All contract authorities now use the global 41 / 38 / 3
  inventory; the v2-only 35 subset is named and clearly
  described. No v1 valid fixture is hidden inside an "executable
  accept/reject" count.
- The selection memo and validator selection remain
  `santhosh-tekuri/jsonschema/v6` (v6.0.2 at proof time) with
  `compiler.AssertFormat()` enabled.

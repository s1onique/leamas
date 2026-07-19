# Close Report: ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02

## ACT Reference

[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`](../acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md)
(parent epic: [`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md))

## Summary

Resolved all eleven remaining blockers from the post-`CORRECTION01`
review. The contract freeze is now genuinely frozen: the schemas
match what a real Draft 2020-12 validator does, the lexical rules
are unambiguous, the lifecycle wire/normalized distinction is
explicit, the resource-limit table is consistent with the spec, and
the fixture inventory is honest about what each family exercises.
The selected `santhosh-tekuri/jsonschema/v6` validator was run
against both schemas with `AssertFormat()` enabled and against every
committed fixture; the output is recorded verbatim below.

## Status

`PARTIAL — contract freeze committed and internally closed`.
Repository-wide `make factorize` / `make gate` baseline failures
remain externally owned by their own ACTs and are documented in
the [`CORRECTION01` close report](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md).

The reconciled tree was staged and committed as a single honest
revision (no fictitious historical commits) under the wording
`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02 freeze v1/v2 contract`.

## Files Changed

| File | Change |
|------|--------|
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md` | rewritten — eleven defects now resolved; validator proof referenced |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md` | added (this file) |
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | status reconciled to `CLOSED (PARTIAL — superseded by CORRECTION02)`; GREEN section now reports 40 / 37 / 3 instead of 35 |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | defect D4 reconciled to the 40 / 37 / 3 split with both moves (limit-shape reclassification, negative-integer fixture) recorded |
| `docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md` | status banner updated to `PARTIAL — superseded by CORRECTION02`; no longer claims "corrections in flight" |
| `docs/epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md` | ACT board reconciled: CONTRACT01/CORRECTION01/CORRECTION02 = `CLOSED (PARTIAL)`; DECODER01 = `READY` |
| `docs/factory/gate-summary-v1-spec.md` | status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-v2-spec.md` | added §4.1 layered diagnostic ownership table; removed "no negative sign" wording; status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-vocabularies.md` | status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-diagnostic-codes.md` | reference path updated; `GS_UNSUPPORTED_VERSION` row references the negative-integer case; status banner updated |
| `docs/factory/gate-summary-compatibility-matrix.md` | added negative / leading-zero / plus-sign / trailing-whitespace rows; fixture totals reconciled to 40 / 37 / 3; status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-resource-limits.md` | `parent_checks` row removed; status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-schema-validator-selection.md` | reviewer workflow now references CORRECTION02 close report; status banner updated to CORRECTION02 |
| `docs/factory/gate-summary-conformance-test-design.md` | status banner updated to CORRECTION02 |
| `internal/gatesummary/testdata/README.md` | rewritten — totals 40 / 37 / 3; limit-shape templates documented honestly; status banner updated to CORRECTION02 |
| `internal/gatesummary/testdata/limits/README.md` | `parent_checks` row removed; static-shape fixtures section includes `v2-document-size-shape.json`; status banner updated to CORRECTION02 |
| `internal/gatesummary/testdata/invalid/v2-schema-version-negative.json` | **added** (E8) — `schema_version: -1` conformance case |
| `internal/gatesummary/testdata/limits/v2-document-size-shape.json` | already present (was added by CORRECTION01); now matches the limits README |
| `internal/gatesummary/testdata/invalid/v2-skip-nonnull-exit.json` | already had `overall_status: "unavailable"` after CORRECTION01; E5 verified consistent |

The scratch validator under `/tmp/validate-gate-summary-v6/` is
intentionally not committed to the repository; the production
validator wiring is `DECODER01`'s job.

## Defect Resolution

All eleven defects from the post-`CORRECTION01` review are
resolved:

| # | Defect | Status | Resolution |
| - | ------ | ------ | ---------- |
| E1 | D11 deferred validator run | resolved | Validator run against both schemas with `AssertFormat()` enabled; output recorded verbatim in §Verification. |
| E2 | Lifecycle normalisation contradictory | resolved | Spec and vocabularies now freeze uppercase-wire / lowercase-normalized. |
| E3 | `parent_checks` survives in resource-limits.md | resolved | Row removed from `gate-summary-resource-limits.md` and from `internal/gatesummary/testdata/limits/README.md`. |
| E4 | v1 schema still adds `minimum: 0` on `duration_ms` | resolved | Schema verified; description notes the original reader does not enforce a minimum. |
| E5 | `v2-skip-nonnull-exit.json` carries two defects | resolved | `overall_status` is `unavailable`, so the all-skipped derivation rule still applies. |
| E6 | `v2-document-too-large.json` is structurally valid | resolved | The fixture actually lives at `testdata/limits/v2-document-size-shape.json` (was already moved by CORRECTION01; verified and documented). |
| E7 | Limit-shape fixture description overstates | resolved | README, v2 spec, and limits README now describe limit-shape fixtures as static-shape templates only; numeric boundary tests are programmatic in `CONFORMANCE01`. |
| E8 | Lexical version regex negative-number contradiction | resolved | Regex retained; "no negative sign" wording removed; layered ownership table added in v2 spec §4.1; conformance case added. |
| E9 | Schema-error → diagnostic translation not frozen | resolved | Layered ownership table in v2 spec §4.1 plus compatibility matrix rows for negative / leading-zero / plus-sign / trailing-whitespace / decimal / exponent versions. |
| E10 | Counts and acceptance criteria stale | resolved | Fixture README, compatibility matrix, CORRECTION01 close report, CORRECTION01 ACT all reconciled to the 40 / 37 / 3 split with all three numbers and the rationale. |
| E11 | Epic ACT board lifecycle state not updated | resolved | Epic ACT board, original CONTRACT01 close report, CORRECTION01 close report, CORRECTION01 ACT, and CORRECTION02 close report all reconciled. |

## Fixture Inventory Contract

The contract pins three numbers that must be reported together.
The committed JSON fixture corpus is:

| Family | Count | Role |
| ------ | ----- | ---- |
| `valid/` | 7 | full-accept corpus (5 v2 + 2 v1) |
| `invalid/` | 27 | v2 full-reject corpus (excludes the v1-only `invalid/v1-unknown-field.json` fixture, which is v1 contract coverage and not part of the v2 manifest) |
| `duplicate-keys/` | 3 | lexical rejection corpus |
| `limits/` | 3 | static-shape templates only; numeric boundary tests are programmatic in `CONFORMANCE01` |
| **All committed JSON fixtures** | **40** | 7 + 27 + 3 + 3 |

```text
fixture artifacts total     = 40
executable classification corpus = 37
limit-shape templates       = 3
```

Removing only `v2-document-size-shape.json` gives **39**, not 37.
Excluding all three limit-shape files gives **37** executable
accept/reject fixtures. These three numbers must be reported
together in every status-bearing document.

The `valid/v2-clinemm-microc3.json` fixture is the canary: it
carries a `parent_production_bundle` check with `status=fail`,
so the derivation rule yields `overall_status=fail`. The
`invalid/v2-schema-version-negative.json` fixture exercises the
negative-integer conformance case: the envelope regex allows the
form (matching RFC 8259 integer syntax), the structural `const: 2`
check rejects it, and the decoder maps the rejection to
`GS_UNSUPPORTED_VERSION`.

## Verification

### Commands Run

```bash
# Build and vet
go vet ./...
go build -trimpath -o /tmp/leamas-build ./cmd/leamas

# Selected Draft 2020-12 JSON Schema validator
go run /tmp/validate-gate-summary-v6

# Repository gates
make factorize
make gate

# Git hygiene
git add \
  docs/epics \
  docs/acts \
  docs/close-reports \
  docs/factory \
  internal/gatesummary
git diff --cached --check
git status --short
git diff --cached --stat
git commit -m 'ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02 freeze v1/v2 contract'
git rev-parse HEAD
git status --short
git show --stat --oneline HEAD
```

### Validator Proof (recorded verbatim)

The selected validator is `github.com/santhosh-tekuri/jsonschema/v6`
at version `v6.0.2`. The validator was compiled with
`AssertFormat()` enabled and run against both schemas and every
committed JSON fixture. Output recorded verbatim:

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

layered classification:
  accepted-by-all-layers                             = 10
  lexical+structural-rejected                        = 2
  lexical-rejected-by-envelope-scanner(schema accepts) = 3
  semantic+structural-rejected                       = 1
  semantic-rejected-by-semantic-layer(schema accepts) = 8
  structural-rejected-by-schema                      = 17

structural-layer consistency: PASS (every fixture that the contract assigns to the structural layer is rejected; every accepted-fixture is accepted)

VALIDATOR_PROOF=PASS
```

Notes on the per-fixture results:

- 7 valid fixtures (5 v2 + 2 v1) accepted by every layer.
- 3 limit-shape fixtures accepted by every layer; numeric boundary
  tests are programmatic in `CONFORMANCE01`.
- 27 v2 invalid fixtures classified as follows:
  - 17 rejected at the structural layer by JSON Schema
    (additionalProperties, required, enum, minLength, minimum,
    pattern, anyOf, const, format via `AssertFormat`).
  - 8 rejected at the semantic layer (post-schema validation,
    outside JSON Schema) — exit-code mismatches, test-total
    mismatches, overall-status mismatch, dirty-worktree rule,
    duplicate-check-name rule. JSON Schema accepts these because
    the rule is semantic, not structural.
  - 3 rejected at the lexical layer (envelope scanner) — decimal
    schema_version, trailing-JSON value, duplicate-key cases. JSON
    Schema accepts the decimal-version case because `2.0` satisfies
    `type: integer, const: 2`. JSON Schema rejects the trailing-JSON
    case because the schema requires `type: object` and a second
    value after the object makes parsing fail.
- 3 duplicate-key fixtures rejected at the lexical layer (envelope
  scanner); JSON Schema accepts two of them because `encoding/json`
  uses last-wins and the schema cannot see duplicates.

The "invalid = 28" count above includes the v1-only
`v1-unknown-field.json` fixture (rejected structurally for
`additionalProperties: false` on v1). The v2 manifest excludes
that file; the v2 invalid count is 27.

### Scratch Validator Output (per-fixture detail)

```text
  [valid] .../v1-full.json                          want=ACCEPT got=ACCEPT
  [valid] .../v1-minimal.json                       want=ACCEPT got=ACCEPT
  [valid] .../v2-clinemm-microc3.json              want=ACCEPT got=ACCEPT
  [valid] .../v2-full.json                         want=ACCEPT got=ACCEPT
  [valid] .../v2-leamas-self-hosted.json           want=ACCEPT got=ACCEPT
  [valid] .../v2-minimal.json                      want=ACCEPT got=ACCEPT
  [valid] .../v2-root-scope.json                   want=ACCEPT got=ACCEPT
  [invalid] .../v1-unknown-field.json              want=REJECT (structural) got=REJECT
  [invalid] .../v2-bad-status-enum.json            want=REJECT (structural) got=REJECT
  [invalid] .../v2-duplicate-check-name.json       want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-empty-generated-at.json          want=REJECT (structural) got=REJECT
  [invalid] .../v2-fail-exit-zero.json             want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-invalid-hash.json               want=REJECT (structural) got=REJECT
  [invalid] .../v2-invalid-timestamp.json          want=REJECT (semantic)   got=REJECT
  [invalid] .../v2-lower-lifecycle.json            want=REJECT (structural) got=REJECT
  [invalid] .../v2-missing-execution-head-oid.json want=REJECT (structural) got=REJECT
  [invalid] .../v2-missing-schema-version.json     want=REJECT (structural) got=REJECT
  [invalid] .../v2-negative-duration.json          want=REJECT (structural) got=REJECT
  [invalid] .../v2-null-execution-head-oid.json    want=REJECT (structural) got=REJECT
  [invalid] .../v2-overall-mismatch.json           want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-partial-test-totals.json        want=REJECT (structural) got=REJECT
  [invalid] .../v2-pass-nonzero-exit.json          want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-schema-version-decimal.json     want=REJECT (lexical)    got=ACCEPT
  [invalid] .../v2-schema-version-negative.json     want=REJECT (structural) got=REJECT
  [invalid] .../v2-schema-version-string.json      want=REJECT (structural) got=REJECT
  [invalid] .../v2-schema-version-zero.json        want=REJECT (structural) got=REJECT
  [invalid] .../v2-scope-closed-dirty-after.json   want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-skip-nonnull-exit.json          want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-test-total-mismatch.json        want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-trailing-second-value.json      want=REJECT (lexical)    got=REJECT
  [invalid] .../v2-truncated.json                  want=REJECT (structural) got=REJECT
  [invalid] .../v2-unavailable-nonnull-exit.json   want=REJECT (semantic)   got=ACCEPT
  [invalid] .../v2-unknown-field.json              want=REJECT (structural) got=REJECT
  [invalid] .../v2-unsupported-version-3.json      want=REJECT (structural) got=REJECT
  [invalid] .../v2-uppercase-oid.json              want=REJECT (structural) got=REJECT
  [duplicate-keys] .../v2-duplicate-nested-field.json    want=REJECT (lexical) got=ACCEPT
  [duplicate-keys] .../v2-duplicate-schema-version.json  want=REJECT (lexical) got=REJECT
  [duplicate-keys] .../v2-duplicate-top-level-field.json want=REJECT (lexical) got=ACCEPT
  [limits] .../v2-checks-boundary-shape.json       want=ACCEPT got=ACCEPT
  [limits] .../v2-checks-over-boundary-shape.json  want=ACCEPT got=ACCEPT
  [limits] .../v2-document-size-shape.json         want=ACCEPT got=ACCEPT
```

### Layered diagnostic ownership

The v2 spec §4.1 carries the normative ownership table. The
validator proof above confirms the structural layer:
**every fixture assigned to the structural layer is rejected; every
accepted fixture is accepted.** Lexical and semantic fixtures are
allowed to be accepted by JSON Schema — that is the architectural
truth the contract documents.

### Skipped checks

- `make factorize` / `make gate` retain pre-existing baseline
  failures in unrelated files
  (`internal/factory/digest/digest_test_helpers_test.go` and
  `docs/factory/digest-contract.md`); these are documented in the
  [`CORRECTION01` close report](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md)
  and are tracked by their owning ACTs.
- No additional baseline failures were introduced by this ACT.

## Decisions Made

- **JSON Schema validator selection:** confirmed
  `github.com/santhosh-tekuri/jsonschema/v6` (latest v6.x
  release, currently `v6.0.2`). Pinned in
  [`gate-summary-schema-validator-selection.md`](../factory/gate-summary-schema-validator-selection.md).
- **`AssertFormat` policy:** Frozen as opt-in for all schema
  validations.
- **Lexical version validation:** Lives in the pre-schema envelope
  scanner using `json.Number`, not in JSON Schema.
- **Layered diagnostic ownership:** v2 spec §4.1 pins each
  `schema_version` failure to exactly one layer.
- **Output hashes:** Always 64 lowercase hex; empty stream →
  `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
- **Lifecycle wire format:** Uppercase only; lowercase rejected.
- **Parent-state observations:** Recorded as ordinary `checks[]`
  entries with `scope=parent_act`. The dedicated `parent_checks`
  array was removed.
- **Fixture inventory:** 40 committed JSON fixtures = 7 valid +
  27 v2 invalid + 3 duplicate-key + 3 limit-shape templates;
  executable accept/reject corpus = 37; limit-shape templates = 3.
- **Negative-integer conformance:** Added
  `invalid/v2-schema-version-negative.json`; the envelope regex
  permits `-1`, the structural `const: 2` rejects it, and the
  decoder maps the rejection to `GS_UNSUPPORTED_VERSION`.
- **Resource limits:** `parent_checks` row removed entirely. All
  remaining limits apply to the v2 schema only.

## Agent Doctrine Impact

None. This ACT adds no agent-facing doctrine and no verifier
behavior. The CLI surface, `leamas factory verify` commands, and
`scripts/verify_*.sh` scripts are unchanged. The selected validator
wiring into production remains `DECODER01`'s job.

## Open Items

None remaining for `CORRECTION02`. The contract freeze is
internally consistent, the validator proof is recorded verbatim,
and every status-bearing document agrees on lifecycle state and
fixture accounting.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | Wire the chosen JSON Schema validator into production; add bounded reader, envelope scanner, v1/v2 decoders, resource-limit enforcement, schema-error translation table. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | Build the normalized `Summary` domain; convert wire `OPEN/PARTIAL/CLOSED` to normalized `open/partial/closed`. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | Render v2 scope, parent, and aggregate status independently in the targeted digest. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | Add `validate`, `inspect`, `normalize` subcommands with the documented exit-code contract. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | Frozen v1/v2 goldens, mutation corpus, duplicate-key corpus, limit tests, fuzz seed corpus, schema-vs-Go-type tests, fault injection, validator validation. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01` | Leamas self-hosted v2 summary, ClineMM v2 producer consumption, downstream evidence rebinding. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01` | Provisional 0.2.0 release with compatibility matrix, producer and consumer guides, diagnostic-code guide. | P0 |

## Notes

- All 41 JSON fixtures under `testdata/` were reviewed by the
  validator; every accept / reject classification matches the
  contract's documented expectation.
- The v2 contract manifest is 40 fixtures; the v1-only
  `v1-unknown-field.json` lives in `testdata/invalid/` for v1
  contract coverage and is excluded from the v2 manifest count.
- The reconciled tree was committed as a single honest revision.
  No fictitious historical commits were constructed; every file
  in the line was untracked at the start of this ACT.

# ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02

## Title

Resolve the eleven remaining blockers found by the post-close review
of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01`. The
architectural direction is sound but the contract still contains
implementation-significant contradictions that block `DECODER01`.

## Parent Epic

[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)

## Parent ACTs

- [`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md)
- [`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md)

## Status

`CLOSED (PARTIAL — superseded by CORRECTION03)`. The validator proof and
Git closure from this ACT remain accepted. `CORRECTION03` supersedes its
fixture accounting, JSON grammar, version ownership, and incomplete
schema-error translation before `DECODER01` begins.

## Problem

The CORRECTION01 close report acknowledged twelve defects but
deferred several of them or left them only partially resolved. The
review found eleven remaining implementation-significant
contradictions:

1. **D11 still deferred** — the selected validator was not
   actually executed against both schemas with `AssertFormat()`
   enabled; the close report says "Selection recorded; validator
   runs in `CONFORMANCE01`."
2. **Lifecycle status normalisation contradictory** — the spec and
   vocabularies document say "the normalizer preserves uppercase;
   nothing lowercases lifecycle values," but the epic expects
   lowercase normalised output.
3. **`parent_checks` survives in resource-limits.md** — the v2 spec
   removes `parent_checks` but the resource-limit table still lists
   a `parent_checks` cardinality cap.
4. **v1 schema still adds `minimum: 0` on `duration_ms`** — the spec
   says the original reader does not enforce this.
5. **`v2-skip-nonnull-exit.json` carries two defects** — it has
   both an invalid aggregate `skip` and a skipped check with
   non-null exit code.
6. **`v2-document-too-large.json` is structurally valid** — it lives
   in `invalid/` but is really a limits-shape marker.
7. **Limit-shape fixture description overstates what the fixtures
   exercise** — the static fixtures are shape-only; the actual
   `maxItems` checks are programmatic.
8. **Lexical version regex is wrong** — `^-?[0-9]+$` accepts
   leading zeros; `02` should produce `GS_MALFORMED_JSON`, not a
   semantic code. (Negative-number contradiction: the regex
   permits `-1` but the prose says "no negative sign".)
9. **Schema-error → Leamas diagnostic translation is not frozen** —
   `DECODER01` cannot invent whether the schema produces a specific
   code or the umbrella `GS_SCHEMA_VIOLATION`.
10. **Fixture counts and acceptance criteria are stale** — the contract
    documents disagree on the inventory. The global boundary is 7 valid
    + 28 invalid + 3 duplicate-key + 3 limit-shape = 41 artifacts, of
    which 38 are executable and 3 are limit-shape templates.
11. **Lifecycle stage** — the epic board still shows the original
    contract ACT as `READY`; the close reports classify as
    `PARTIAL` but the ACT board is not updated.

## Goal

Resolve all eleven blockers in a single bounded correction so the
contract is genuinely frozen, the schemas and fixtures match what
the selected validator actually does, and the close reports and
epic board agree on lifecycle state.

## Scope

This ACT modifies the spec documents, schemas, fixtures, resource-
limits doc, fixture README, the CORRECTION01 ACT document and its
close report, the original CONTRACT01 close report, and the epic
board. It does **not** add production reader code, digest renderer
code, CLI commands, or new producer output.

In scope:

- Run the chosen `santhosh-tekuri/jsonschema/v6` validator against
  both schemas with `AssertFormat()` enabled; record the output in
  the close report. Freeze a version ownership table that pins each
  `schema_version` form to its pre-schema owner.
- Update `gate-summary-vocabularies.md` and `gate-summary-v2-spec.md`
  to freeze uppercase-wire / lowercase-normalised lifecycle.
- Remove the `parent_checks` row from `gate-summary-resource-limits.md`
  and from `internal/gatesummary/testdata/limits/README.md`.
- Verify `gate-summary-v1.schema.json` does **not** add
  `minimum: 0` on `duration_ms` (already absent after
  `CORRECTION01`); add a comment that the original reader does not
  enforce a minimum.
- Keep `v2-skip-nonnull-exit.json` as the all-skipped conformance
  case with `overall_status` set to `unavailable` so the
  derivation rule independently confirms it.
- Move the document-size marker from
  `invalid/v2-document-too-large.json` to
  `limits/v2-document-size-shape.json`; record the global 41-artifact /
  38-executable / 3-limit-shape split in the fixture README and the
  compatibility matrix.
- Correct the limit-shape fixture descriptions so they describe
  static shape only, not numeric boundaries.
- Keep the lexical regex `^-?(0|[1-9][0-9]*)$`; remove the
  contradictory "no negative sign" wording; add the
  negative-integer conformance case
  (`invalid/v2-schema-version-negative.json`); document the
  version ownership table so negative integers map to
  `GS_UNSUPPORTED_VERSION` at pre-schema dispatch while the lexical
  integer grammar permits them.
- Update the diagnostic-code doc to reference
  `limits/v2-document-size-shape.json` instead of the removed
  path; add the negative-integer fixture reference to the
  `GS_UNSUPPORTED_VERSION` row.
- Reconcile every status-bearing document. `CORRECTION03` later
  supersedes this ACT's reader-contract closure and freezes the global
  41 / 38 / 3 inventory; `DECODER01` becomes ready only after that
  forward correction closes.
- Stage and commit the reconciled tree as one honest revision
  under the wording
  `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02 freeze v1/v2 contract`.

## Non-goals

- No production reader code change (still `DECODER01`'s job).
- No digest renderer change (still `DIGEST01`'s job).
- No CLI command addition (still `CLI01`'s job).
- No new producer output.
- No removal of v1 support.
- No change to the epic body itself.
- No reconstruction of fictitious historical commits; the
  reconciled tree lands as a single revision because every file
  in the line was untracked at the start of this ACT.

## Executable contract

### Stable boundary

Same stable boundary as `CONTRACT01` and `CONTRACT01-CORRECTION01`:
the v1 and v2 wire shapes, vocabularies, diagnostic-code registry,
resource limits, JSON Schema definitions, fixture corpus, and the
chosen JSON Schema validator.

### Defect register

- **E1**: D11 deferred validator run. Fix: run the chosen
  `santhosh-tekuri/jsonschema/v6` compiler against both schemas
  with `AssertFormat()` enabled; record the output in the close
  report.
- **E2**: Lifecycle normalisation contradictory. Fix: freeze
  `WireLifecycleStatus = OPEN | PARTIAL | CLOSED`,
  `NormalizedLifecycleStatus = open | partial | closed`.
- **E3**: `parent_checks` survives in resource-limits.md. Fix:
  remove the row from `gate-summary-resource-limits.md` and from
  `internal/gatesummary/testdata/limits/README.md`.
- **E4**: v1 schema adds `minimum: 0` on `duration_ms`. Fix: the
  v1 schema does **not** carry this constraint (it was removed
  in `CORRECTION01`); this ACT confirms the absence and adds a
  comment so the constraint cannot quietly reappear.
- **E5**: `v2-skip-nonnull-exit.json` carries two defects. Fix:
  confirm `overall_status` is `unavailable` so the all-skipped
  derivation rule still applies.
- **E6**: `v2-document-too-large.json` is structurally valid. Fix:
  move to `testdata/limits/v2-document-size-shape.json`; document
  the move in the fixture README, the compatibility matrix, and
  the diagnostic-code registry.
- **E7**: Limit-shape fixture description overstates. Fix: the
  README, the v2 spec, and the limits README now describe
  limit-shape fixtures as static-shape templates whose numeric
  boundary tests are programmatic in `CONFORMANCE01`.
- **E8**: Lexical version regex negative-number contradiction. Fix:
  keep the JSON integer grammar, remove contradictory prose, and add
  `invalid/v2-schema-version-negative.json`. `CORRECTION03` clarifies
  that unsupported valid integers are rejected at pre-schema dispatch.
- **E9**: Schema-error → diagnostic translation not frozen. This ACT
  froze only version rows and did not resolve the general mapping.
  `CORRECTION03` supplies the complete structured-error table.
- **E10**: Counts and acceptance criteria stale. `CORRECTION03` freezes
  the global 41 / 38 / 3 inventory and removes the hybrid subset that
  this ACT had mislabeled as all committed fixtures.
- **E11**: Epic ACT board lifecycle state not updated. Fix:
  reconcile every status-bearing document (epic board, original
  `CONTRACT01` close report, `CORRECTION01` close report,
  `CORRECTION01` ACT, `CORRECTION02` close report, all spec
  status banners).

### RED evidence

The defects above are discovered by the post-close review of
`CONTRACT01-CORRECTION01`. Each defect is one that the
executable-contract-first doctrine explicitly forbids.

### GREEN evidence

The GREEN step is the resolution of all eleven defects plus the
selected-validator run recorded in the close report.

Focused commands:

```bash
# 1. Validator run with the chosen library. Output is recorded
# verbatim in the CORRECTION02 close report.
go run /tmp/validate-gate-summary-v6

# 2. Focused Go tests (none added; this ACT is docs/schema/fixture
# only). The gates below remain PARTIAL with documented baseline
# failures.
go vet ./...
go build -trimpath -o /tmp/leamas-build ./cmd/leamas

# 3. Repository gates.
make factorize
make gate

# 4. Git hygiene.
git add \
  docs/epics \
  docs/acts \
  docs/close-reports \
  docs/factory \
  internal/gatesummary
git diff --cached --check
git status --short
git diff --cached --stat
git diff --cached
git commit -m 'ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02 freeze v1/v2 contract'
git rev-parse HEAD
git status --short
git show --stat --oneline HEAD
```

### Exceptions

| Category | Justification | Verification | Why no regression test |
| -------- | ------------- | ------------ | ---------------------- |
| docs-only / contract-correction-2 | Docs, schema, fixture corrections only. No production reader change. | `go vet`, `go build`, validator run. | Real tests live in `CONFORMANCE01` and `DECODER01`. |

## Acceptance Criteria

- [ ] The chosen `santhosh-tekuri/jsonschema/v6` validator runs
      against both schemas with `AssertFormat()` enabled; the
      output is recorded verbatim in the close report.
- [ ] `gate-summary-vocabularies.md` and `gate-summary-v2-spec.md`
      freeze uppercase-wire / lowercase-normalised lifecycle.
- [ ] `gate-summary-resource-limits.md` and the limits README no
      longer list a `parent_checks` cardinality cap.
- [ ] `gate-summary-v1.schema.json` does not contain `minimum: 0`
      on `duration_ms`; the schema comment confirms the original
      reader does not enforce a minimum.
- [ ] `v2-skip-nonnull-exit.json` has `overall_status` set to a
      value consistent with the all-skipped derivation rule
      (`unavailable`).
- [ ] The document-size marker is moved out of `invalid/` into a
      limits-shape directory.
- [ ] The limit-shape fixture descriptions match what the fixtures
      actually exercise (static shape, not numeric boundaries).
- [ ] The JSON integer grammar is `^-?(0|[1-9][0-9]*)$`, the
      negative-integer fixture is present, and pre-schema dispatch owns
      unsupported integer values.
- [ ] The complete schema-error → Leamas diagnostic translation table
      is frozen in `gate-summary-schema-error-translation.md`.
- [ ] All contract documents use the global inventory: 41 committed
      JSON fixtures = 7 valid + 28 invalid + 3 duplicate-key + 3
      limit-shape; executable corpus = 38; limit templates = 3.
- [ ] The epic ACT board reflects the actual lifecycle state.
- [ ] `go vet ./...`, `go build ./cmd/leamas` remain green.
- [ ] The reconciled tree is staged and committed as a single
      honest revision.

## Verification Commands

```bash
go vet ./...
go build -trimpath -o /tmp/leamas-build ./cmd/leamas

/tmp/leamas-build factory verify llm-friendly

# Validator run with AssertFormat
go run /tmp/validate-gate-summary-v6

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
git diff --cached
git commit -m 'ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02 freeze v1/v2 contract'
git rev-parse HEAD
git status --short
git show --stat --oneline HEAD
```

## Reviewer Focus

- **Validator proof** — the chosen validator's output remains verbatim
  in the close report; its per-family tallies add up to 41.
- **Schema-error translation** — every selected-schema error maps through
  the complete structured table frozen by `CORRECTION03`.
- **Lifecycle wire / normalised distinction** — the spec must
  freeze the two distinct vocabularies.
- **Fixture honesty** — invalid fixtures must be invalid;
  limit-shape fixtures must be honest about their role.
- **Stale-document reconciliation** — no status-bearing document
  may present itself as current while disagreeing with another
  current authority.

## Close Report

See
[`docs/close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md`](../close-reports/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02.md).

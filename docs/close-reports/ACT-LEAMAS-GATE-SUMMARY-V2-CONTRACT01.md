# Close Report: ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01

> **Status:** PARTIAL — superseded by
> [`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`](../acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03.md).
> The CORRECTION02 validator/Git proof remains accepted. See the
> CORRECTION03 close report for the final reader-contract evidence.

## ACT Reference

[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`](../acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md)
(parent epic: [`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md))

## Summary

Frozen the gate-summary v1 and v2 wire contracts, vocabularies,
overall-status derivation, JSON Schema definitions, stable
diagnostic-code registry, compatibility matrix, JSON Schema validator
selection, conformance-test design, and a fixture corpus. **No
production reader, digest renderer, producer output, or downstream
evidence was changed.** The original ACT self-reported as `CLOSED`;
the post-close review identified twelve concrete defects that the
correction ACT is repairing.

## Status

`PARTIAL — superseded`. The original ACT self-classified as
`CLOSED`, but the post-close review identified twelve concrete
defects. Those twelve are resolved in `CONTRACT01-CORRECTION01` and
the eleven follow-up items identified by the post-`CORRECTION01`
review's validator and Git proof remain accepted. Reader-contract
semantics are superseded and closed by `CONTRACT01-CORRECTION03`;
`DECODER01` becomes `READY` only after that forward correction.

## Files Changed

The `CONTRACT01` line of files was later modified by
`CONTRACT01-CORRECTION01`. The complete current state of every file
in the line is recorded under
[`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01`](./ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md)
close report (to be written).

The `CONTRACT01` deliverable set:

| File | Added by `CONTRACT01` | Modified by `CORRECTION01` |
|------| --------------------- | -------------------------- |
| `docs/epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md` | yes | no |
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01.md` | yes | no |
| `docs/acts/ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01.md` | no | added by correction ACT |
| `docs/factory/gate-summary-v1-spec.md` | yes | yes |
| `docs/factory/gate-summary-v2-spec.md` | yes | yes |
| `docs/factory/gate-summary-vocabularies.md` | yes | yes |
| `docs/factory/gate-summary-resource-limits.md` | yes | yes |
| `docs/factory/gate-summary-diagnostic-codes.md` | yes | yes |
| `docs/factory/gate-summary-compatibility-matrix.md` | yes | yes |
| `docs/factory/gate-summary-schema-validator-selection.md` | yes | yes |
| `docs/factory/gate-summary-conformance-test-design.md` | yes | yes |
| `internal/gatesummary/schema/gate-summary-v1.schema.json` | yes | yes |
| `internal/gatesummary/schema/gate-summary-v2.schema.json` | yes | yes |
| `internal/gatesummary/testdata/README.md` | yes | yes |
| `internal/gatesummary/testdata/limits/README.md` | yes | yes |
| `internal/gatesummary/testdata/valid/v1-minimal.json` | yes | no |
| `internal/gatesummary/testdata/valid/v1-full.json` | yes | no |
| `internal/gatesummary/testdata/valid/v2-minimal.json` | yes | yes |
| `internal/gatesummary/testdata/valid/v2-full.json` | yes | yes |
| `internal/gatesummary/testdata/valid/v2-root-scope.json` | yes | yes |
| `internal/gatesummary/testdata/valid/v2-clinemm-microc3.json` | yes | yes (added failing parent-state check) |
| `internal/gatesummary/testdata/valid/v2-leamas-self-hosted.json` | yes | yes |
| `internal/gatesummary/testdata/invalid/v1-unknown-field.json` | yes | no |
| `internal/gatesummary/testdata/invalid/v2-*.json` (25 files) | yes | yes (hashes normalized, additions) |
| `internal/gatesummary/testdata/duplicate-keys/*.json` (3 files) | yes | yes (hashes normalized) |
| `internal/gatesummary/testdata/limits/*.json` (2 files) | yes | yes (renamed) |

## Behavior Changed

None in production. The contract freeze is a documentation, schema,
and fixture corpus deliverable. Production reader behavior, digest
renderer behavior, producer output, and downstream evidence remain
unchanged.

## Verification

### Commands Run

```bash
# Build and vet
go build -o /tmp/leamas-build ./cmd/leamas
go vet ./...

# LLM-friendliness gate
/tmp/leamas-build factory verify llm-friendly

# Manual scratch validator
/tmp/validate-fixtures

# Full go test ./... and make factorize
# (both timed out or showed pre-existing failures, see Skipped checks)
```

### Results

- `go vet ./...` passes.
- `go build ./cmd/leamas` succeeds.
- All ACT-deliverable files pass the LLM-friendliness gate (the
  pre-existing `digest-contract.md` long-file violation is not in
  this ACT's scope).
- The offline scratch validator confirms every frozen fixture's
  documented accept/reject classification.

### Skipped checks

- `make factorize`, `make gate`, and a full `go test ./...` were
  attempted in the background. The summary reports two pre-existing
  failures in `internal/factory/dupcode` (timeout) and the cascaded
  failure in `internal/factory/gate` (factorize failed because of
  the pre-existing `exec-gate` finding in
  `internal/factory/digest/digest_test_helpers_test.go` and the
  pre-existing `llm-friendly` finding in
  `docs/factory/digest-contract.md`).
- Both pre-existing failures are outside this ACT's scope and are
  documented here for transparency.

## Decisions Made

The original `CONTRACT01` made twelve decisions that the post-close
review flagged as defective. Each decision is corrected in
`CONTRACT01-CORRECTION01`:

| Decision in `CONTRACT01` | Correction in `CONTRACT01-CORRECTION01` |
| ------------------------ | --------------------------------------- |
| Lifecycle wire accepts both `OPEN` and `open`. | Restricted to uppercase only. |
| Output hashes may be the empty string. | Required to be exactly 64 lowercase hex; empty stream → SHA-256 of empty byte stream. |
| `parent_checks` was a separate array. | Removed; producers record parent-state observations as ordinary `checks[]` with `scope=parent_act`. |
| v1 schema added `maxItems=10000` and other limits. | Stripped back to match the original reader. |
| `2.0` rejected via `type: integer, const: 2` (false claim). | Lexical version validation moved to the pre-schema envelope scanner. |
| `format: date-time` assumed asserted by default. | Frozen `compiler.AssertFormat()` policy. |
| Validator selection: "v6 or v5 if unavailable." | Pinned to `v6` only. |
| Fixture count claimed 32 but actually 35. | Final global inventory is 41 JSON artifacts, 38 executable fixtures, and 3 limit-shape templates; v2-only executable subset is 35. |
| Diagnostic coverage claimed but several codes lacked fixtures. | Ordinary-input codes use fixtures/generated cases; internal failures use fault injection. |
| Diagnostic ordering defined twice (Code-Path vs precedence list). | Unified on `precedence rank, then path, then encounter index`. |
| Limit fixtures named `v2-checks-{max,over-max}.json`. | Renamed to `v2-checks-{boundary,over-boundary}-shape.json`. |
| Closure claimed despite non-green gates. | Reclassified as `PARTIAL` in the correction ACT. |
| Schemas not validated against the chosen Draft 2020-12 validator. | CORRECTION02 recorded the accepted v6.0.2 `AssertFormat()` proof. |

## Agent Doctrine Impact

None. This ACT adds no agent-facing doctrine and no verifier
behavior. The CLI surface, `leamas factory verify` commands, and
`scripts/verify_*.sh` scripts are unchanged.

## Open Questions

- The closure is `PARTIAL` because `make factorize` / `make gate` /
  `go test ./...` retain pre-existing baseline failures in
  unrelated files. The architecture is sound but cannot be
  declared immutable until the correction ACT's defects are
  resolved.
- The selected JSON Schema validator will be exercised in
  `CONFORMANCE01` and `DECODER01`, not in this ACT, because no
  production reader code is added here.

## Follow-up ACTs

| ACT | Description | Priority |
|-----|-------------|----------|
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01` | Resolve the twelve defects identified by the post-close review. | P0 (active) |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | Wire the chosen JSON Schema validator, add bounded reader, lexical envelope scanner, strict v1/v2 decoders, resource-limit enforcement. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | Build the normalized `Summary` domain; add v1/v2 normalizers and semantic validators. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | Render v2 scope, parent, and aggregate status independently in the targeted digest. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | Add `validate`, `inspect`, `normalize` subcommands with the documented exit-code contract. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | Frozen v1/v2 goldens, one-mutation negatives, duplicate-key corpus, resource-limit tests, fuzz seed corpus, schema-vs-Go-type conformance tests, fault-injection tests. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01` | Leamas self-hosted v2 summary, ClineMM v2 producer consumption, downstream evidence rebinding. | P0 |
| `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01` | Provisional 0.2.0 release with compatibility matrix, producer and consumer guides, diagnostic-code guide. | P0 |

## Notes

- This close report deliberately records the original ACT's
  self-claimed CLOSED status as `PARTIAL` because the
  `CONTRACT01-CORRECTION01` review identified twelve concrete
  defects.
- The scratch validator under `/tmp/validate-gate-summary-fixtures/`
  is intentionally not committed to the repository. The committed
  conformance tests live in `CONFORMANCE01`.
- The fixture for `v2-clinemm-microc3.json` was corrected in
  `CONTRACT01-CORRECTION01` to include a `parent_production_bundle`
  check with `status=fail`, so the derivation rule independently
  confirms `overall_status=fail`.

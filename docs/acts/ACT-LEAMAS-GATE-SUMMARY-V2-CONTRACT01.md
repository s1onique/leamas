# ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01

## Title

Freeze the gate-summary v1 and v2 wire contracts, lifecycle and gate
status semantics, JSON Schema definitions, diagnostic-code registry,
fixture corpora, and JSON Schema validator selection.

## Parent Epic

[`EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01`](../epics/EPIC-LEAMAS-GATE-SUMMARY-SCHEMA-V2-ADOPTION01.md)

## Problem

Leamas currently consumes gate-summary artifacts with a single
schema-version-1 path (`internal/factory/gate/summary.go`) and rejects
schema-version-2 evidence before any semantic check runs. ClineMM, the
first real downstream producer, emits v2 with bounded-scope, parent,
and execution-binding extensions that v1 cannot represent. Without a
frozen contract, the decoder, normalizer, digest renderer, CLI, and
conformance tests would each invent incompatible interpretations.

## Goal

Produce an immutable, internally consistent contract for v1 and v2
gate-summary evidence so that every subsequent ACT
(`DECODER01`, `NORMALIZATION01`, `DIGEST01`, `CLI01`, `CONFORMANCE01`,
`DOGFOOD01`, `RELEASE01`) can implement against one authoritative
source.

## Scope

This ACT delivers only documentation, JSON Schemas, fixture corpora,
and an evaluator selection memo. It does not modify production reader
behavior, digest output, producer output, or downstream evidence.

In scope:

- `docs/factory/gate-summary-v1-spec.md` — frozen v1 reconstruction.
- `docs/factory/gate-summary-v2-spec.md` — authoritative v2 contract.
- `docs/factory/gate-summary-vocabularies.md` — lifecycle and gate-status
  vocabularies, derivation rules.
- `docs/factory/gate-summary-resource-limits.md` — consumer-safety
  limits.
- `docs/factory/gate-summary-diagnostic-codes.md` — stable diagnostic
  code registry.
- `docs/factory/gate-summary-compatibility-matrix.md` — input-to-result
  compatibility matrix.
- `docs/factory/gate-summary-schema-validator-selection.md` — JSON
  Schema validator selection memo.
- `docs/factory/gate-summary-conformance-test-design.md` — schema vs.
  Go type conformance test design.
- `internal/gatesummary/schema/gate-summary-v1.schema.json` — Draft
  2020-12 schema for v1.
- `internal/gatesummary/schema/gate-summary-v2.schema.json` — Draft
  2020-12 schema for v2.
- `internal/gatesummary/testdata/README.md` — fixture inventory.
- Valid, invalid, mutation, duplicate-key, and resource-limit fixture
  corpora under `internal/gatesummary/testdata/`.

## Non-goals

- No production reader code change.
- No digest renderer change.
- No CLI command addition.
- No new producer output.
- No rebinding of downstream evidence.
- No v3 design.
- No removal of v1 support.

## Executable contract

### Stable boundary

The stable boundary being changed or protected is the public
gate-summary evidence contract owned by Leamas. The contract defines:

- Required and optional fields for v1 and v2.
- Version-discriminator type and accepted values.
- Status vocabularies and their derivations.
- Resource limits as consumer-safety guards.
- Stable diagnostic codes for invalid evidence.
- JSON Schema definitions for both versions.
- The list of valid and invalid fixtures that subsequent ACTs must
  satisfy.

### Test matrix

| Case | Dimension | Given | When | Expected |
| ---- | --------- | ----- | ---- | -------- |
| `T01` | Frozen v1 fixture | Minimal valid v1 JSON in `internal/gatesummary/testdata/valid/v1-minimal.json` | Read against v1 schema | Schema validates, all required fields present |
| `T02` | Frozen v1 fixture | Full valid v1 JSON in `internal/gatesummary/testdata/valid/v1-full.json` | Read against v1 schema | Schema validates |
| `T03` | Frozen v2 fixture | Minimal valid v2 JSON in `internal/gatesummary/testdata/valid/v2-minimal.json` | Read against v2 schema | Schema validates |
| `T04` | Frozen v2 fixture | Full valid v2 JSON in `internal/gatesummary/testdata/valid/v2-full.json` | Read against v2 schema | Schema validates |
| `T05` | Root scope | `valid/v2-root-scope.json` with empty `parent_act` and `parent_status=closed` | Read against v2 schema | Schema validates |
| `T06` | ClineMM µC-3 | `valid/v2-clinemm-microc3.json` matches the documented ClineMM µC-3 example | Read against v2 schema | Schema validates |
| `T07` | Schema-vs-Go | A Go conformance-test program validates every fixture against both the JSON Schema and the matching Go wire struct | Run in `ACT-CONFORMANCE01` | Each fixture is accepted iff it satisfies both schemas and structs |
| `T08` | Diagnostic codes | Every code in `docs/factory/gate-summary-diagnostic-codes.md` is referenced from at least one fixture | Fixture audit | No orphans, no missing |
| `T09` | Resource limits | Boundary fixtures in `testdata/limits/` document the exact thresholds | Read against `gate-summary-resource-limits.md` | Boundary values match the policy |

### RED evidence

Not applicable. CONTRACT01 is a documentation-and-fixture freeze with
no production reader behavior change. The red phase for behavior lives
in `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` and downstream ACTs.

The closest analog is a manual review: each frozen fixture must be
validated against its Draft 2020-12 schema by the selected offline
tool, and the resulting set of accepted/rejected fixtures must match
the compatibility matrix.

### GREEN evidence

To be established in subsequent ACTs that consume this contract. This
ACT's exit criterion is **manual review** plus `make factorize`,
`make gate`, and `go test ./...` to confirm no existing behavior
changed.

Focused command (manual review):

```bash
# 1. Confirm no production behavior changed
go test ./internal/factory/...
go test ./cmd/leamas/...

# 2. Confirm schemas parse as Draft 2020-12
# (Manual: see docs/factory/gate-summary-schema-validator-selection.md
#  for the chosen offline tool and command sequence.)
```

Affected subsystem command:

```bash
go test ./internal/factory/...
go test ./cmd/leamas/...
```

Repository gate command:

```bash
make factorize
make gate
```

### Exceptions

The single exception below is the only deviation from the
RED-then-GREEN cycle. The four required exception fields are recorded
in the bulleted block that follows.

- **Category:** docs-only / contract-freeze
- **Justification:** The deliverable is a frozen contract, schemas,
  and fixtures. No production reader behavior change means there is
  no production code path to write a regression test against in this
  ACT.
- **Verification:** `go test ./...`, `make factorize`, and `make gate`
  confirm pre-existing tests still pass. Each fixture is also
  validated against its schema offline (see
  [`gate-summary-schema-validator-selection.md`](../factory/gate-summary-schema-validator-selection.md)).
- **Why no regression test:** A regression test would test an
  implementation that does not yet exist in this ACT. The conformance
  test design
  ([`gate-summary-conformance-test-design.md`](../factory/gate-summary-conformance-test-design.md))
  is the specification; its implementation lands in
  `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01`.

## Acceptance Criteria

- [ ] `docs/factory/gate-summary-v1-spec.md` exists, is internally
      consistent, and reconstructs the v1 schema as it exists in
      `internal/factory/gate/summary.go`.
- [ ] `docs/factory/gate-summary-v2-spec.md` exists, declares all
      required fields, vocabularies, derivation rules, identity rules,
      cleanliness rules, and parent-state rules from the epic body.
- [ ] `docs/factory/gate-summary-vocabularies.md` exists and pins
      lifecycle status, gate status, and the parent root-scope
      convention.
- [ ] `docs/factory/gate-summary-resource-limits.md` exists and lists
      the initial consumer-safety limits with rationale.
- [ ] `docs/factory/gate-summary-diagnostic-codes.md` exists and
      enumerates the codes listed in the epic body.
- [ ] `docs/factory/gate-summary-compatibility-matrix.md` exists and
      enumerates the input-to-result matrix from the epic body.
- [ ] `docs/factory/gate-summary-schema-validator-selection.md` exists,
      pins a Draft 2020-12-capable offline Go validator, and explains
      the choice.
- [ ] `docs/factory/gate-summary-conformance-test-design.md` exists
      and defines the schema-vs-Go-type test design.
- [ ] `internal/gatesummary/schema/gate-summary-v1.schema.json` is a
      Draft 2020-12 schema that validates the v1 valid fixtures.
- [ ] `internal/gatesummary/schema/gate-summary-v2.schema.json` is a
      Draft 2020-12 schema that validates the v2 valid fixtures.
- [ ] `internal/gatesummary/testdata/README.md` enumerates every
      fixture and its expected accept/reject classification.
- [ ] Valid fixtures cover: minimal v1, full v1, minimal v2, full v2,
      v2 root-scope, v2 ClineMM µC-3, v2 Leamas self-hosted.
- [ ] Invalid fixtures cover at least: unsupported version, string
      version, decimal version, missing schema_version, duplicate
      schema_version, uppercase OID, scope-closed-dirty-worktree,
      overall-status-mismatch, pass-with-nonzero-exit.
- [ ] Duplicate-key fixtures cover at least: duplicate
      `schema_version`, duplicate check name, duplicate nested field.
- [ ] Resource-limit fixtures document the boundary values from
      `gate-summary-resource-limits.md`.
- [ ] No file under `internal/gatesummary/` exceeds 64 KiB or 400
      lines.
- [ ] `go test ./...`, `make factorize`, and `make gate` remain green.

## Verification Commands

```bash
# Build and unit tests
go test ./...
go vet ./...

# Repository gates
make factorize
make gate

# Verify LLM-friendliness on the new files
make verify-llm-friendly

# Confirm no production behavior changed
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
```

## Reviewer Focus

- **Internal consistency.** The v1 spec must match
  `internal/factory/gate/summary.go` byte-for-byte in its public shape.
  The v2 spec must match the schema files. The compatibility matrix
  must match the fixtures. The diagnostic codes must be referenced from
  at least one fixture each.
- **Immutability.** Once approved, the spec, schemas, fixtures, and
  diagnostic codes must not change without a documented defect and
  dedicated correction ACT. Subsequent ACTs must consume the frozen
  contract as-is.
- **No production drift.** No file outside the new
  `internal/gatesummary/` and `docs/factory/gate-summary-*.md` set
  should be touched by this ACT.
- **Ownership doctrine.** The validator selection memo and spec
  documents must keep ClineMM out of the schema-owner role.

## Close Report Stub

> **Summary:** Freeze the gate-summary v1 and v2 contracts, schemas,
> fixtures, diagnostic codes, and validator selection with no
> production-reader behavior change.
> **Files changed:** see close report.
> **Behavior changed:** none.
> **Verification:** `go test ./...`, `make factorize`, `make gate`,
> manual schema validation pass.
> **Follow-up ACTs:** `DECODER01`, `NORMALIZATION01`, `DIGEST01`,
> `CLI01`, `CONFORMANCE01`, `DOGFOOD01`, `RELEASE01`.

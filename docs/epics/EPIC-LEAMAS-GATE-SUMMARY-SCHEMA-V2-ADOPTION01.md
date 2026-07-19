# Epic: Gate-Summary Schema v2 Adoption

> Versioned gate-summary contract, strict consumption, normalization,
> self-hosting, and downstream convergence.

**Status:** ACTIVE
**Priority:** P0
**Owner repository:** `leamas`
**Primary implementation language:** Go
**First downstream conformance repository:** `clinemm`
**Target release:** next compatible Leamas feature release, provisionally `0.2.0`
**Supersedes:** schema-v1-only gate-summary consumption
**Depends on:** current Leamas Factory, targeted-digest, output-contract,
and verification infrastructure
**Blocks:** ClineMM CORRECTION21 µC-3 evidence closure and later
parent-baseline convergence

---

## 1. Problem

Schema v1 represents a flat aggregate gate result adequately, but cannot
faithfully express the evidence topology now required by Factory-governed
projects:

```text
a bounded child scope is CLOSED
while its parent ACT remains OPEN
while the aggregate machine gate remains FAIL
because required parent production evidence does not yet exist
```

Collapsing those facts into one `pass`, `fail`, or `partial` value loses
important information. Schema v2 separates bounded-scope lifecycle,
parent lifecycle, aggregate machine-gate status, execution and subject
identity, worktree cleanliness, named and scoped checks, process execution
evidence, output hashes, optional test arithmetic, parent-state checks,
and dispositions.

The immediate defect is concrete: ClineMM emits
`schema_version=2`; the current Leamas consumer supports only
`schema_version=1` and rejects the summary before evaluating its evidence.

## 2. Goal

Adopt gate-summary schema version 2 as a first-class public Factory
evidence contract while preserving strict read compatibility with schema
version 1. After this epic:

- Valid v1 evidence continues to be accepted with stable digest output.
- Valid v2 evidence is accepted and renders scope, parent, and aggregate
  status independently.
- Invalid or unsupported evidence fails closed with stable diagnostic
  codes.
- A published Leamas release containing the compatible reader exists,
  bound to a real ClineMM v2 producer.

## 3. Non-goals

This epic does not:

- Close the ClineMM parent baseline ACT.
- Implement ClineMM P4 Extension Host probing.
- Implement ClineMM P5 standalone binary probing.
- Implement Mach-O authority.
- Redesign every detached Factory evidence artifact.
- Remove schema v1 support.
- Introduce schema version 3.
- Rewrite existing v1 files.
- Infer evidence omitted by a producer.
- Make Leamas trust syntactically valid JSON.
- Create a ClineMM-specific parser branch.
- Adopt experimental JSON APIs merely because their package name contains
  `v2`.

## 4. Context / Evidence

The current Leamas reader (`internal/factory/gate/summary.go`,
`docs/factory/digest-gate-summary.md`) accepts only `schema_version=1`.
It treats any other integer or any non-integer as a fatal parse error.
ClineMM emits `schema_version=2` with the bounded-scope, parent, and
execution-binding extensions required for child-vs-parent convergence.
Until Leamas reads v2, ClineMM evidence cannot close CORRECTION21 µC-3.

The targeted-digest `GATE_SUMMARY` section also collapses all lifecycle
information into a single `overall_status=` line, which hides a closed
child scope behind an open parent.

## 5. Constraints

- No Python anywhere.
- Go for product code, labs, verifiers, and substantial automation.
- JSON Schema Draft 2020-12 is the wire schema dialect.
- Files stay LLM-friendly (≤ 64 KiB; ≤ 400 lines; ≤ 240 chars/line).
- Diagnostic codes are stable public machine identifiers.
- v1 behavior must not regress unexpectedly.
- No source summary may be automatically rewritten.

## 6. ACT Board

| ACT | Status | Notes |
| --- | ------ | ----- |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01` | CLOSED (PARTIAL — superseded) | Initial contract freeze; superseded by corrections. |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION01` | CLOSED (PARTIAL) | Corrected twelve defects; left eleven follow-up items. |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02` | CLOSED (PARTIAL — superseded) | Validator proof accepted; reader-contract semantics superseded by CORRECTION03. |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03` | CLOSED (PARTIAL) | Reader contract frozen and committed; `DECODER01` unblocked. |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DECODER01` | READY | Begins now. |
| `ACT-LEAMAS-GATE-SUMMARY-V2-NORMALIZATION01` | PENDING | Domain normalization |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DIGEST01` | PENDING | Digest integration |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CLI01` | PENDING | CLI surface |
| `ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` | PENDING | Golden + fuzz |
| `ACT-LEAMAS-GATE-SUMMARY-V2-DOGFOOD01` | PENDING | Self-hosting proof |
| `ACT-LEAMAS-GATE-SUMMARY-V2-RELEASE01` | PENDING | Release and migration |

## 7. Verification Strategy

- Repository-wide gates remain green at each ACT close:
  `go test ./...`, `go vet ./...`, `make factorize`, `make gate`.
- Focused `internal/gatesummary/...` tests stay green.
- Fuzz smoke (`go test ./internal/gatesummary -fuzz ... -fuzztime 30s`)
  never panics.
- ClineMM downstream digest reports
  `source_status=present`, `schema_version=2`,
  `scope_status=closed`, `parent_status=open`, `overall_status=fail`.

## 8. Risks and Unknowns

| Risk | Likelihood | Impact | Mitigation |
| ---- | ---------- | ------ | ---------- |
| v2 support becomes permissive decoding | Medium | High | Separate wire types, strict decoding, embedded schemas, duplicate-name rejection, frozen negative fixtures |
| v1 behavior regresses | Medium | High | Frozen v1 corpus, normalized-output fixtures, targeted-digest goldens |
| Schemas and Go types drift | Medium | Medium | Every valid/invalid fixture is evaluated by both JSON Schema validation and Go decoding |
| ClineMM becomes the accidental schema authority | Low | High | Leamas owns specifications, fixtures, diagnostics, examples, release docs |
| Lifecycle and aggregate status conflated | Low | High | Distinct Go types, distinct normalized fields, distinct digest fields |
| Duplicate keys lost before validation | Medium | High | Token-based duplicate-name detection precedes version dispatch and schema validation |
| Malformed evidence causes excessive allocation | Medium | Medium | Bounded reads, collection limits, string limits, fuzzing, explicit limit diagnostics |
| Future versions accepted accidentally | Low | High | Exact integer dispatch and deterministic unsupported-version rejection |
| Normalization invents unavailable v1 evidence | Medium | Medium | Nullable normalized parent and binding fields; no synthetic identities or cleanliness claims |
| Producer verdicts silently "fixed" | Low | High | Leamas validates and reports mismatches but never rewrites evidence |

## 9. Close Criteria

This epic is `PASS / CLOSED` only when all of the following are true:

- Leamas accepts every frozen valid v1 fixture.
- Leamas rejects every frozen invalid v1 fixture.
- Leamas accepts every frozen valid v2 fixture.
- Leamas rejects every frozen invalid v2 fixture.
- Leamas rejects duplicate object names.
- Leamas rejects unsupported versions.
- Leamas enforces documented resource limits.
- Leamas preserves expected v1 digest behavior.
- Leamas renders v2 scope, parent, and aggregate status separately.
- Leamas enforces all v2 semantic invariants.
- Leamas exposes `validate`, `inspect`, and `normalize` commands.
- All focused and repository Go gates are green.
- Fuzz smoke is green.
- Leamas self-hosted v2 consumption is green.
- Real ClineMM v2 consumption is green.
- A published Leamas artifact containing v2 support exists.
- ClineMM evidence is rebound to that artifact.

## 10. Follow-ups

- v1 deprecation policy (no removal in this epic).
- Future v3 reserved for genuinely orthogonal changes (e.g., nullable
  parent, nested parent, multi-aggregate gates).
- ClineMM CORRECTION21 µC-3 evidence closure depends on this epic.
- ClineMM parent-baseline convergence depends on this epic.

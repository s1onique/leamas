# Gate-Summary Schema v1 — Frozen Specification

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
> No further changes to this document are permitted without a
> dedicated correction ACT.

This document is the authoritative reconstruction of the
gate-summary **schema version 1** wire contract as it exists in
`internal/factory/gate/summary.go` and
`internal/factory/gate/run_summary.go` today. It is published so
that v1 fixtures, schemas, and tests can be checked against one
canonical source.

`CONTRACT01` originally added `maxItems` and `maxLength` constraints
to the v1 schema. `CONTRACT01-CORRECTION01` removed those
constraints because the original reader does not enforce them. The
v1 schema is therefore narrower than the
`CONTRACT01`-era v1 schema.

## 1. Purpose

The v1 wire contract records the observed result of one literal
`leamas factory gate` run as a single flat aggregate. It predates
the bounded-scope/parent lifecycle distinction added in v2.

## 2. Top-level required fields

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `schema_version` | integer | yes | Must equal `1`. |
| `generated_at` | string | yes | RFC 3339 timestamp; the producer emits UTC. |
| `overall_status` | string | yes | One of the v1 check statuses. |
| `checks` | array | yes | List of recorded checks. May be empty. |

## 3. Top-level optional fields

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `tool` | string | no | Producer-identification string. The literal gate emits `leamas factory gate`. |

Any field not listed above is unknown to v1 and must be rejected by
the strict v1 decoder.

## 4. Check object

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `name` | string | yes | Non-empty. |
| `status` | string | yes | One of `pass`, `fail`, `skip`, `unavailable`. |
| `duration_ms` | integer | no | Wall-clock duration in milliseconds (rendered by the v1 reader; not enforced as a wire constraint). |
| `evidence` | string | no | Free-form human-readable evidence pointer. The original reader applies a 240-character rendering cap (`MaxEvidenceLength`) but does **not** enforce a wire-level string limit. |

## 5. Status vocabulary

```text
pass
fail
skip
unavailable
```

These are the **only** allowed v1 check statuses. The literal gate
emits exactly one `gate` check whose status is `pass` if the gate exit
code was `0`, otherwise `fail`.

## 6. Rendering contract

The targeted-digest `GATE_SUMMARY` section renders the v1 evidence as:

```text
## GATE_SUMMARY
source=.factory/gate-summary.json
source_status=present
schema_version=1
generated_at=...
overall_status=...
checks_total=...
checks_passed=...
checks_failed=...
checks_skipped=...
checks_unavailable=...
checks:
  - name=... status=... [duration_ms=...] evidence=...
```

The render order is fixed, checks are sorted lexicographically by
`name`, and no v2-only fields appear.

## 7. Compatibility

A v1 document with any of the following properties is **invalid**:

- `schema_version` is missing, is not an integer, is not equal to `1`,
  or has duplicate object keys;
- `generated_at` is empty or is not a parseable RFC 3339 timestamp;
- `overall_status` is not in the v1 status vocabulary;
- a `checks[]` entry has a missing `name` or `status`, has a
  duplicate `name`, or has a `status` outside the v1 vocabulary;
- the document contains unknown top-level or per-check fields;
- the document is followed by a trailing second JSON value.

## 8. Production limits (consumer-safety)

The current Leamas consumer applies only a soft `evidence` rendering
cap of `240` characters
(`internal/factory/gate/summary.go:MaxEvidenceLength`). The schema
does **not** add wire-level limits that the original reader does not
enforce.

`CONTRACT01-CORRECTION01` removed the v1 schema's earlier
`maxItems` and `maxLength` constraints to match this. Any
consumer-safety limits the v2 reader adds are documented in
[`gate-summary-resource-limits.md`](./gate-summary-resource-limits.md)
and apply only to v2.

## 9. Immutability

Any deliberate v1 behavioral correction must be:

1. Documented as a defect.
2. Captured in a dedicated correction ACT that explicitly lists the
   fixtures affected.
3. Recorded in the compatibility matrix and the v1 fixtures corpus.

Silent reinterpretation of valid v1 evidence is forbidden.

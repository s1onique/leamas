# Gate-Summary Vocabularies

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.

This document pins the controlled vocabularies used by the gate-summary
contract. Every status a producer can emit, every normalized status a
consumer can render, and every derivation rule that links them are
listed here.

## 1. Machine-gate status (per-check)

Used for every entry in `checks[].status`.

| Wire value | Semantics |
| ---------- | --------- |
| `pass` | The check succeeded. `exit_code` must equal `0`. |
| `fail` | The check failed. `exit_code` must be non-zero or `null` (spawn/setup/timeout/infrastructure failure). |
| `skip` | The check was intentionally not executed. `exit_code` must be `null`. |
| `unavailable` | The check could not be executed. `exit_code` must be `null`. |

These four values are exhaustive. Producers must not emit any other
value. Consumers must reject any other value as `GS_INVALID_STATUS`.

## 2. Aggregate `overall_status`

A strict subset of the per-check vocabulary.

| Wire value | Semantics |
| ---------- | --------- |
| `pass` | All checks passed. |
| `fail` | At least one check failed. |
| `unavailable` | No checks failed; at least one check was `unavailable`; or all checks were `skip` (or empty). |

`skip` is **not** a valid `overall_status`. An all-skipped check set
derives `unavailable`. This is the explicit
`else: derived = unavailable` clause in
[`gate-summary-v2-spec.md`](./gate-summary-v2-spec.md) §16.

## 3. Lifecycle status

Used for `scope_status` and `parent_status`.

| Wire value | Semantics |
| ---------- | --------- |
| `OPEN` | The bounded scope or parent ACT is still in progress. |
| `PARTIAL` | Some child work is complete; required evidence is still missing. |
| `CLOSED` | All required evidence for this scope is present. |

Wire form is **uppercase only**: `OPEN`, `PARTIAL`, `CLOSED`.
Lowercase wire values are rejected at the schema layer
(`GS_INVALID_STATUS`). The normalizer converts wire form to
**normalized** form: `open`, `partial`, `closed`. The validator emits
diagnostic codes but never modifies the lifecycle string itself;
only `NORMALIZATION01` produces the lowercased display form used by
the digest and CLI.

## 4. Root-scope convention

A v2 document whose `parent_act` is the empty string uses the frozen
convention:

```text
parent_act=""
parent_status="CLOSED"
parent_disposition="root scope; no parent"
```

A root scope **must not** be rendered as `overall_status=fail` solely
because the parent disposition is non-empty.

## 5. Overall-status derivation

Leamas independently derives the aggregate status from the check list
(never from the producer's claimed `overall_status`):

```text
derived =
  if any check.status == fail:        fail
  else if any check.status == unavail: unavailable
  else if any check.status == pass:    pass
  else:                                 unavailable
```

`skip` does not appear in the derivation table; a `skip` check does
not contribute to the derived status. The recorded `overall_status`
must match `derived`. A mismatch produces
`GS_OVERALL_STATUS_MISMATCH`.

This rule applies to v2 only. v1 documents do not have a derivation
contract: the producer's `overall_status` is treated as authoritative
when it is one of the four machine-gate values.

## 6. Parent-state observations

Parent-state observations MUST be recorded as ordinary `checks[]`
entries with `scope` equal to `parent_act`. They participate in the
aggregate derivation. This is what makes the ClineMM µC-3 fixture
honest: its `parent_production_bundle` check has `status=fail`, so
the derived status is `fail`, matching the recorded
`overall_status=fail`.

## 7. Check status precedence

When a single check produces multiple status observations (for example,
a Go test run with both a passing build and a failing test assertion),
the producer must collapse them to one of the four machine-gate values
before recording them. Leamas does not provide a multi-status field.

## 8. Disposition rules

`scope_disposition`, `parent_disposition`, and `overall_disposition`
are free-form human-readable strings. They must:

- be non-empty;
- not contain raw control characters (`\x00`–`\x1f`) that would split
  a single digest line into multiple visual lines.

The renderer escapes control characters deterministically before
emitting them into the digest, but producers should avoid them.

## 9. Stability

The four-value machine-gate vocabulary, the three-value aggregate
`overall_status` vocabulary, and the three-value lifecycle vocabulary
are frozen for v2. A future schema version (e.g., v3) may extend them
only via a new `$id` and an explicit compatibility-matrix update.

# Gate-Summary Schema v2 — Authoritative Specification

> **Status:** Authoritative as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`.
> Frozen for `DECODER01`, `NORMALIZATION01`, `DIGEST01`, `CLI01`,
> `CONFORMANCE01`, `DOGFOOD01`, and `RELEASE01`.
> No further changes to this document are permitted without a
> dedicated correction ACT.

This document is the public Leamas Factory evidence contract for
gate-summary **schema version 2**. It is owned by Leamas and is the
single source of truth that all downstream producers and consumers
must conform to.

## 1. Purpose

Schema v2 records, in one self-contained artifact, the full evidence
topology required by Factory-governed projects:

- the bounded child scope's lifecycle;
- the parent ACT's lifecycle;
- the aggregate machine-gate status;
- the execution and subject identity;
- worktree cleanliness before and after;
- named and scoped checks;
- per-check process execution evidence and output hashes;
- optional test arithmetic;
- free-form dispositions explaining human-readable rationale.

Schema v1 cannot express these facts independently.

## 2. Document type

A v2 gate-summary document is one top-level JSON object. Multiple
top-level values, or a single value followed by trailing JSON, are
invalid (see compatibility matrix). The pre-schema envelope scanner
in `DECODER01` enforces this.

## 3. Top-level required fields

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `schema_version` | integer | yes | Must equal `2`. See §4. |
| `generated_at` | string | yes | Non-empty RFC 3339 timestamp with explicit UTC offset. See §6. |
| `scope_id` | string | yes | Bounded child scope identifier. |
| `scope_status` | string | yes | Lifecycle status. Wire form is **uppercase only**. See §7. |
| `scope_disposition` | string | yes | Human-readable rationale. |
| `parent_act` | string | yes | Parent ACT identifier; empty string for root scopes. |
| `parent_status` | string | yes | Lifecycle status of the parent. Wire form is **uppercase only**. |
| `parent_disposition` | string | yes | Human-readable rationale. |
| `overall_status` | string | yes | Aggregate machine-gate status. Vocabulary: `pass`, `fail`, `unavailable`. See §7 and §16. |
| `overall_disposition` | string | yes | Human-readable rationale. |
| `execution_head_oid` | string | yes | Git object ID. See §9. |
| `execution_tree_oid` | string | yes | Git tree object ID. See §9. |
| `subject_tree_oid` | string | yes | Git tree object ID. See §9. |
| `worktree_clean_before` | boolean | yes | True if the worktree was clean before the run. |
| `worktree_clean_after` | boolean | yes | True if the worktree was clean after the run. |
| `checks` | array | yes | Recorded checks. May be empty. |

`parent_checks` was **removed** in
`CONTRACT01-CORRECTION01`. Producers record parent-state observations
as ordinary `checks[]` entries with `scope` equal to the parent
ACT identifier. The aggregate derivation includes those entries.

## 4. Version discriminator

`schema_version` must be a JSON integer token whose numeric value is
`1` or `2`. Two layers of validation apply; the layered diagnostic
ownership table is the normative reference and lives in
[`gate-summary-schema-version-translation.md`](./gate-summary-schema-version-translation.md)
to keep this spec under the line limit.

1. **JSON Schema structural check**: `type: integer, const: 2` for
   v2 (and `type: integer, const: 1` for v1).
2. **Pre-schema envelope scanner check**: the lexical token must
   match `^-?(0|[1-9][0-9]*)$` exactly. The leading `-` is
   **permitted** so the lexer mirrors the JSON integer syntax RFC
   8259 allows (for example `-1`); the dispatcher then rejects any
   negative numeric value as `GS_UNSUPPORTED_VERSION`.

The structural check is what the JSON Schema validator runs. The
lexical check is what the envelope scanner runs in `DECODER01` using
`json.Decoder.UseNumber()`. Both must pass.

## 5. Parent representation

For v2, parent fields are required. A root scope uses the frozen
convention:

```text
parent_act=""
parent_status="CLOSED"
parent_disposition="root scope; no parent"
```

A future nullable or nested parent representation requires a new
schema version and is not retrofitted into v2.

## 6. Timestamp

`generated_at` must:

- be a non-empty RFC 3339 string;
- include an explicit UTC offset (`Z` or `±HH:MM`);
- parse without truncation or repair;
- normalize internally without rewriting the source.

The format check is asserted by the chosen validator's
`compiler.AssertFormat()` call. Invalid timestamps produce
`GS_INVALID_TIMESTAMP`.

## 7. Status vocabularies

Two distinct vocabularies. They are **not** interchangeable.

### 7.1 Machine-gate status (aggregate and per-check)

```text
pass
fail
skip
unavailable
```

`skip` is allowed on per-check entries but **never derived** as an
aggregate `overall_status`.

### 7.2 Aggregate `overall_status`

```text
pass
fail
unavailable
```

This is a strict subset of the per-check vocabulary. `skip` is not a
valid `overall_status`. An all-skipped check list derives
`unavailable` (see §16).

### 7.3 Lifecycle status (scope and parent)

```text
OPEN
PARTIAL
CLOSED
```

Wire form is **uppercase only**: `OPEN`, `PARTIAL`, `CLOSED`.
Lowercase wire values are rejected at the schema layer. The
**normalized** form used by the digest and CLI is lowercase:
`open`, `partial`, `closed`. The validator emits diagnostic codes
but never modifies the lifecycle string; only `NORMALIZATION01`
produces the lowercased display form.

A bounded scope may be `CLOSED` while the parent is `OPEN` and the
aggregate `overall_status` is `fail`. The digest renders these facts
independently.

## 8. (Removed) Parent-state checks

`parent_checks` was a separate array for parent-state observations.
`CONTRACT01-CORRECTION01` removed it because:

- It duplicated the parent-scope rule: a producer can already
  record a parent-state observation as an ordinary `check` with
  `scope` equal to the parent ACT identifier.
- Including the parent-state observation in the regular check list
  ensures it participates in `overall_status` derivation. This is
  the ClineMM µC-3 model: `parent_production_bundle` is a `check`
  with `status=fail`, which derives `overall_status=fail`.

Producers MUST record parent-state observations as ordinary
`checks[]` entries with `scope` equal to `parent_act`.

## 9. Execution identity

Required fields:

```text
execution_head_oid
execution_tree_oid
subject_tree_oid
```

Accepted forms:

```text
40 lowercase hexadecimal characters   (SHA-1 Git object ID)
64 lowercase hexadecimal characters   (SHA-256 Git repository object ID)
```

Invalid forms:

- uppercase hexadecimal;
- abbreviated IDs (any length other than 40 or 64);
- empty strings;
- JSON `null`.

Invalid OIDs produce `GS_INVALID_OID`.

## 10. Worktree cleanliness

`worktree_clean_before` and `worktree_clean_after` are JSON booleans.

A v2 document claiming `scope_status=CLOSED` is semantically invalid
when either cleanliness value is `false`. Diagnostic:
`GS_SCOPE_CLOSED_DIRTY_WORKTREE`. This validates the producer's
recorded evidence; it does not independently inspect the consumer's
worktree.

## 11. Check object

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `name` | string | yes | Non-empty; unique within one summary. See §12. |
| `scope` | string | yes | Logical scope tag. For parent-state observations, equal to `parent_act`. |
| `status` | string | yes | One of the v2 gate statuses. |
| `evidence` | string | yes | Free-form human-readable evidence pointer. |
| `detail` | string | yes | Free-form human-readable detail line. |
| `extras` | object | yes | Process execution evidence. See §13. |
| `total` | integer | no | Optional test counts. See §14. |
| `pass_count` | integer | no | Optional test counts. |
| `fail_count` | integer | no | Optional test counts. |
| `skip_count` | integer | no | Optional test counts. |
| `unavailable_count` | integer | no | Optional test counts. |

A v2 check object is rejected if any required field is missing,
`null`, or of an unexpected JSON type, or if any unknown field is
present.

## 12. Check-name invariants

Check names must be:

- non-empty;
- unique within one summary (across `checks[]` and any future
  per-scope arrays);
- compared exactly and case-sensitively;
- deterministic in normalized output.

Canonical producer pattern (advisory, not enforced on the wire):

```regex
^[a-z0-9][a-z0-9_.-]*$
```

A duplicate name produces `GS_DUPLICATE_CHECK_NAME`. Leamas must not
silently append an index or collapse duplicate checks.

## 13. Extras object

| Field | Type | Required | Notes |
| ----- | ---- | -------- | ----- |
| `argv` | array of strings | yes | Process invocation. May be empty only when justified by `detail`. |
| `exit_code` | integer or null | yes | See §15. |
| `duration_ms` | integer | yes | Non-negative. |
| `stdout_sha256` | string | yes | Exactly 64 lowercase hex characters. |
| `stderr_sha256` | string | yes | Exactly 64 lowercase hex characters. |

### 13.1 Hash invariants

Both `stdout_sha256` and `stderr_sha256` must match `^[0-9a-f]{64}$`
exactly. The empty string is **not** permitted; an empty output
stream produces the SHA-256 of the empty byte stream:

```text
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

This avoids the empty-vs-absent ambiguity and gives producers a
deterministic, content-addressable value.

### 13.2 Argv invariants

- `len(argv) <= 1024`;
- every argv element is a JSON string;
- every argv element length `<= 64 KiB`;
- an empty argv is permitted only when the check's execution model
  genuinely has no process invocation; the producer must justify
  this in `detail`.

### 13.3 Duration

`duration_ms` is a JSON integer `>= 0`. Negative durations produce
`GS_INVALID_DURATION`.

## 14. Optional test arithmetic

The following fields are an **all-or-none** group:

```text
total
pass_count
fail_count
skip_count
unavailable_count
```

When any one of them is present, all five must be present. Partial
arithmetic is invalid (`GS_PARTIAL_TEST_TOTALS`).

When present:

```text
total >= 0
pass_count >= 0
fail_count >= 0
skip_count >= 0
unavailable_count >= 0
total = pass_count + fail_count + skip_count + unavailable_count
```

A mismatch produces `GS_TEST_TOTAL_MISMATCH`.

## 15. Status / exit-code invariants

| status | exit_code |
| ------ | --------- |
| `pass` | `0` (integer) |
| `fail` | non-zero integer, or `null` for spawn / setup / timeout / infrastructure failure |
| `skip` | `null` |
| `unavailable` | `null` |

Violations produce `GS_PASS_EXIT_CODE_MISMATCH`,
`GS_FAIL_EXIT_CODE_MISMATCH`, `GS_SKIP_EXIT_CODE_MISMATCH`, or
`GS_UNAVAILABLE_EXIT_CODE_MISMATCH` respectively. Leamas validates the
recorded relationship but does not second-guess whether a producer
should have classified a condition as `fail` or `unavailable`.

## 16. Overall-status derivation

Leamas independently derives aggregate status from the check list
(never from the producer's claimed `overall_status`):

```text
if any check.status == fail:
    derived = fail
else if any check.status == unavailable:
    derived = unavailable
else if any check.status == pass:
    derived = pass
else:
    derived = unavailable
```

Consequences: a failed check dominates every other status;
`unavailable` dominates `pass` when no check failed; skipped checks do
not turn an otherwise passing gate red; all-skipped or empty check
sets derive `unavailable`.

The recorded `overall_status` must equal `derived`. A mismatch
produces `GS_OVERALL_STATUS_MISMATCH`.

## 17. Strictness

A v2 decoder must:

- read through a bounded reader (`io.LimitReader`) capped at 4 MiB +
  1 byte;
- reject `GS_DOCUMENT_TOO_LARGE` before any tokenization;
- token-scan for duplicate object member names at every depth;
  duplicates produce `GS_DUPLICATE_KEY` and stop the pipeline;
- enforce lexical version rules via `json.Number`;
- decode the integer version into `int` and dispatch;
- use `DisallowUnknownFields` for the version-specific wire struct;
- verify the document reaches EOF after the first JSON value
  (`GS_TRAILING_JSON`);
- run the embedded JSON Schema with `AssertFormat()` enabled;
- run version-specific semantic validation;
- normalize into the common domain model;
- run normalized-model invariants.

## 18. Producer invariants

A producer emitting a v2 document must:

- not omit or `null`-out any required field;
- not emit a `scope_status=CLOSED` claim alongside
  `worktree_clean_before=false` or `worktree_clean_after=false`;
- not emit a v1-shaped document and claim it is v2;
- not embed v1-only fields (`tool`, etc.) in a v2 document;
- not silently rewrite v1 evidence as v2;
- record parent-state observations as ordinary `checks[]` entries
  with `scope` equal to `parent_act`;
- emit lifecycle values in uppercase only;
- emit exactly 64 lowercase hex characters for both output hashes
  (empty stream → `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).

## 19. Immutability

This document is the authoritative wire contract for v2. The
immutability rules, change-control procedure, and compatibility
matrix updates are documented in
[`gate-summary-compatibility-matrix.md`](./gate-summary-compatibility-matrix.md).

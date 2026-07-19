# Resource-Limit Fixtures

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION02`.

The static fixtures in this directory capture only the **structural
shape** of a v2 document at the resource-limit boundary. The actual
boundary tests are programmatic and live in
`ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` because:

- A literal 10,000-check JSON document would exceed the
  LLM-friendliness gate (64 KiB).
- A literal 64 KiB argv-element document would also exceed the
  LLM-friendliness gate.
- A literal 4 MiB+ document would obviously exceed the
  LLM-friendliness gate.
- Programmatic generators can synthesize exactly-at-boundary and
  exactly-over-boundary inputs without committing a multi-megabyte
  fixture to the repository.

## Boundary values

The limits are pinned in
[`gate-summary-resource-limits.md`](../../../docs/factory/gate-summary-resource-limits.md).

| Boundary | Value |
| -------- | ----- |
| Maximum document size | 4 MiB |
| Maximum `checks` count | 10,000 |
| Maximum `argv` entries per check | 1,024 |
| Maximum `argv` element length | 64 KiB |
| Maximum check-name length | 1 KiB |
| Maximum `scope_id` length | 16 KiB |
| Maximum `evidence` length | 1 MiB |
| Maximum `detail` length | 1 MiB |
| Maximum `*_disposition` length | 1 MiB |

The dedicated `parent_checks` array was removed from the v2 schema
in `CONTRACT01-CORRECTION01`. Parent-state observations are
recorded as ordinary `checks[]` entries with `scope=parent_act`,
so the `checks` cardinality cap is the only relevant limit.

## Programmatic generator contract

`ACT-LEAMAS-GATE-SUMMARY-V2-CONFORMANCE01` must add a Go test helper
that:

1. Synthesizes a v2 document with exactly N checks for every
   N ∈ {1, 10_000, 10_001}.
2. Synthesizes a v2 document with exactly M argv entries for every
   M ∈ {1, 1_024, 1_025}.
3. Synthesizes a v2 document whose `evidence` field is exactly
   L bytes for every L ∈ {1, 1_048_576, 1_048_577}.
4. Synthesizes a document of exactly 4 MiB + 1 byte.
5. Asserts:
   - the at-boundary inputs accept;
   - the over-boundary inputs reject with `GS_DOCUMENT_TOO_LARGE` or
     `GS_COLLECTION_LIMIT`.

## Static-shape fixtures

| File | Purpose |
| ---- | ------- |
| `v2-checks-boundary-shape.json` | Structural shape for the at-limit check-count boundary. Validates against the schema. |
| `v2-checks-over-boundary-shape.json` | Structural shape for the over-limit check-count boundary. Validates against the schema. |
| `v2-document-size-shape.json` | Document-size marker. The 4 MiB+1-byte over-limit input is generated programmatically in `CONFORMANCE01` because a literal 4 MiB+ JSON file would exceed the LLM-friendliness gate. |

All three fixtures are well-formed v2 documents and validate. They
exist so that the schema-level cardinality / shape policy is
exercised by the static schema validator before the programmatic
boundary tests run.

## History

`v2-checks-{max,over-max}.json` were the original names.
`CONTRACT01-CORRECTION01` renamed them to
`v2-checks-{boundary,over-boundary}-shape.json` because the original
names implied a numeric over-the-limit document, which is actually
impossible at the LLM-friendliness gate's size cap.

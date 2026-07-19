# Gate-Summary Resource Limits

> **Status:** Frozen as of `ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01-CORRECTION03`.
> Initial consumer-safety limits for the v1/v2 reader.

This document lists the **consumer-safety** limits the v2 reader
enforces. These are guards against pathological input; they are not
wire-semantic maxima. A producer that legitimately needs more headroom
must split the artifact (for example, emit one summary per bounded
scope).

## 1. Initial limits

| Limit | Initial value | Diagnostic on breach |
| ----- | ------------- | -------------------- |
| Maximum document size | 4 MiB | `GS_DOCUMENT_TOO_LARGE` |
| Maximum number of `checks` | 10,000 | `GS_COLLECTION_LIMIT` |
| Maximum number of `argv` entries per check | 1,024 | `GS_COLLECTION_LIMIT` |
| Maximum `argv` element length | 64 KiB | `GS_COLLECTION_LIMIT` |
| Maximum check-name length | 1 KiB | `GS_COLLECTION_LIMIT` |
| Maximum `scope_id` length | 16 KiB | `GS_COLLECTION_LIMIT` |
| Maximum `evidence` length | 1 MiB | `GS_COLLECTION_LIMIT` |
| Maximum `detail` length | 1 MiB | `GS_COLLECTION_LIMIT` |
| Maximum `*_disposition` length | 1 MiB | `GS_COLLECTION_LIMIT` |

## 2. Enforcement order

Limits are enforced **before** any expensive allocation or struct
decoding where practical:

1. **Document size.** Read through a bounded reader
   (`io.LimitReader` with `Limit = 4 MiB + 1`). A document that
   exceeds this limit is rejected before any tokenization.
2. **Collection cardinality.** Enforced during JSON Schema validation
   via `maxItems`.
3. **String length.** Enforced during JSON Schema validation via
   `maxLength`.
4. **Decoder-level safety.** Even with schema limits, the Go decoder
   uses `DisallowUnknownFields` and a strict envelope decoder before
   decoding into the version-specific wire struct.

## 3. Diagnostics

Every limit breach returns a single deterministic diagnostic with:

- `code` — one of `GS_DOCUMENT_TOO_LARGE` or `GS_COLLECTION_LIMIT`;
- `path` — JSON Pointer where the breach was detected;
- `expected` — the limit as a string;
- `observed` — the actual value as a string.

Limits are never violated silently. Limits never cause a panic.

## 4. Raising a limit

A limit change requires:

1. A documented producer defect or scaling observation.
2. A correction ACT that:
   - records the new value;
   - adds boundary fixtures at the new value;
   - explains why the new value is safe (resource budget, fuzz
     coverage, downstream consumer safety).
3. An update to the compatibility matrix.

## 5. Relationship to v1

The v1 reader does not retroactively adopt the new limits in
`ACT-LEAMAS-GATE-SUMMARY-V2-CONTRACT01`. The v1 reader keeps the
existing `MaxEvidenceLength = 240` rendering cap and the existing
`os.ReadFile` behavior. The v2 reader adopts the broader limit set
above.

## 6. Document-size fuzzing

The fuzz target in `internal/gatesummary` must never panic on
arbitrary input up to the maximum document size. Fuzz runs respect the
same 4 MiB cap so the fuzz corpus does not blow up CI budgets.

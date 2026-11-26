# ADR-0007: Factory Work Ledger

## Status

Accepted

## Context

Factory-governed repositories track engineering work in manually
maintained monolithic epic boards. The ClineMM repository board is a
44,340-line append-only Markdown file that mixes ACT terminal reports,
work-item state and narrative. Such boards:

- cannot be validated against a schema;
- cannot answer "what is ready to work on" without a human reading;
- duplicate, lose or silently retire engineering obligations;
- cannot distinguish a structural pass from a live qualification;
- cannot be migrated without hand-editing prose.

The Factory doctrine defines lifecycle and evidence semantics. What was
missing was a repository-native mechanism that carries typed, validated,
queryable engineering state while preserving evidence authority,
obligation conservation and execution speed.

## Decision

Leamas implements the **Factory Work Ledger** as a deterministic,
repository-native work-management mechanism.

### One authoritative representation

Canonical state lives in JSON records under `.leamas/ledger/` in the
target repository. Markdown and text output are derived projections, never
a second source of truth.

### Separate concerns

Lifecycle, acceptance, priority, evidence quality and the terminal ACT
verdict are five distinct fields. No field substitutes for another.

### Stable scoped identities

A work item is identified by `<EPIC-ID>/<ITEM-ID>`. The identity is
stable across archive and restore.

### Evidence authority

Artifact identity, claim relevance and engineering acceptance are three
distinct facts. Structural acceptance never satisfies a live qualification
obligation. A terminal ACT verdict never retires the associated
engineering obligation; the retention is recorded explicitly.

### Atomic multi-record mutations

Every mutation validates the candidate state before publishing and is
published through a journal-backed transaction. An interrupted transaction
is rolled back deterministically, so a partial transaction is never exposed
as an accepted state.

### Optimistic concurrency

A mutation declares the revision it expects. A mismatch is rejected
instead of being resolved by last-writer-wins.

### No new runtime dependency

The ledger uses the Go standard library only. There is no database, no
daemon, no cloud service and no automatic push.

## Consequences

### Positive

- Work state becomes queryable and validatable.
- Evidence authority becomes mechanical rather than editorial.
- Obligations survive ACT closure and archival.
- Import from existing boards becomes dry-run-first and idempotent.
- The bootstrap context can be bounded and measured.

### Negative

- The canonical state is a set of files, not a single document. Querying
  is a CLI operation rather than a database query.
- The ledger does not replace the narrative ACT report. It records
  machine-checkable state; the report remains the human narrative.

### Neutral

- The operational board is a projection. It can be regenerated at any
  time from the canonical records.

## Alternatives Considered

### A separate state database

Rejected. It introduces a second source of truth, a daemon and a
distribution problem, and it violates the local-first single-binary
constraint.

### A vector database or embedding index

Rejected. The ledger is a small, bounded set of typed records. A vector
index would add an unmeasured dependency without a demonstrated need.

### Continuing to maintain the Markdown board

Rejected. It is exactly the failure mode this ADR replaces.

## References

- `docs/factory/work-ledger.md`
- `internal/ledger/`
- ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01

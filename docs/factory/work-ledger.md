# Factory Work Ledger

**ACT**: ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01

## Overview

The Factory Work Ledger is a repository-native, deterministic engineering
work-management mechanism. It replaces manually maintained monolithic epic
boards with typed, validated, queryable engineering state.

Doctrine defines lifecycle and evidence semantics. Leamas owns the
implementation. The target repository owns the canonical work records.

## Scope

```text
canonical storage   <repo>/.leamas/ledger/*.json
derived views       leamas ledger render / board / prime
authoritative       the JSON records, not the Markdown views
```

## One Authoritative Storage Representation

All canonical state lives in JSON records under `.leamas/ledger/`:

| Record | File | Purpose |
|--------|------|---------|
| Epic | `epics/<EPIC-ID>.json` | Scoped body of work |
| Work item | `items/<ITEM-ID>.json` | Typed engineering obligation |
| ACT execution record | `acts/<ACT-ID>.json` | Terminal ACT verdict bound to an item |
| Evidence reference | `evidence/<EVIDENCE-ID>.json` | Repository-owned artifact reference |
| Decision | `decisions/<DECISION-ID>.json` | Durable recorded decision |
| Knowledge reference | `knowledge/<KNOWLEDGE-ID>.json` | Reusable knowledge pointer |

Markdown output is a **derived projection**, never a second source of truth.

## Separated Concerns

The ledger keeps five concerns distinct and never lets one substitute
for another:

| Concern | Owner field | Meaning |
|---------|--------------|---------|
| Lifecycle | `lifecycle.state` | Where the record is in its workflow |
| Acceptance | `acceptance.state` | Whether obligations are satisfied |
| Priority | `priority` | Ordering intent (P0..P3) |
| Evidence quality | `evidence.quality` | How strong the attached evidence is |
| Terminal ACT verdict | `act.verdict` | The Factory verdict of a finished ACT |

A terminal ACT verdict of `CLOSED` does **not** close the engineering
obligation it touched. See "Terminal ACT Does Not Close Obligations".

## Stable Scoped Identities

Work item identities are scoped and stable:

```text
<EPIC-ID>/<ITEM-ID>
```

for example `EPIC-CLINEMM-IECR/CPA-02`. The identity never changes
across archive and restore. Restore creates a **new record instance**
that carries the original identity forward and refuses to reuse an
identity that is already live.

## Typed Relationships

Work items carry typed relationship lists:

```text
depends_on    blocking predecessors
provenance    where this item came from (import source, ACT)
successor     items that take over an obligation
evidence      evidence references bound to this item
```

All four are typed and validated. A `depends_on` reference to an unknown
item is a dangling dependency and is rejected.

## Evidence Authority

Three distinct facts are never conflated:

```text
artifact identity   the artifact exists at the declared path with the declared digest
claim relevance     the artifact bears on the obligation
engineering accept  a human/agent accepted the obligation as satisfied
```

Structural acceptance (a passing structural check) never satisfies a live
qualification obligation. Insufficient evidence never becomes successful
qualification.

## Terminal ACT Does Not Close Obligations

When an ACT execution record reaches a terminal verdict, the ACT record is
closed. Any linked work item whose acceptance state is still open keeps its
obligation. The terminal verdict is recorded on the ACT record only.

## Archival

Archive preserves:

- stable identity
- original terminal disposition
- evidence references
- dependency relationships
- required successor obligations
- applicable reopen conditions

Archived records remain discoverable through `list --archived` and
`show`. Restore preserves stable identity and refuses identity reuse.

## Determinism

Identical canonical input produces byte-identical rendered output. The
only explicitly separate value is the presentation timestamp, which is
emitted in its own field and excluded from the byte-identity comparison.

Read commands never require provider credentials or network access.

## Command Surface

```text
leamas ledger init
leamas ledger prime
leamas ledger list
leamas ledger show
leamas ledger next
leamas ledger create
leamas ledger update
leamas ledger claim
leamas ledger link
leamas ledger evidence add
leamas ledger checkpoint
leamas ledger close
leamas ledger archive
leamas ledger restore
leamas ledger render
leamas ledger validate
leamas ledger stats
leamas ledger import
```

## Migration

`leamas ledger import` is a dry-run-first import and reconciliation
primitive. Import:

- preserves source record identity and provenance
- produces explicit ambiguity and unresolved-reference reports
- is idempotent
- never promotes a conversational claim into a repository-verified fact

## References

- `internal/ledger/` - implementation
- `internal/ledger/domain/` - pure domain core
- `docs/adr/0007-factory-work-ledger.md` - decision record

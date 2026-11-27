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

Four distinct facts are never conflated:

```text
artifact integrity   the artifact exists at the declared path with the declared digest and byte size
evidence class       what kind of claim the artifact may support
claim binding        which criterion the artifact supports
engineering accept   a human/agent accepted the obligation as satisfied
```

Artifact integrity is **actually verified**: the file must exist at the
declared repository-relative path, its byte size must match and its
SHA-256 must match. The path is resolved safely: a path that escapes the
repository root, directly or through a symbolic link, is rejected before
any byte is read.

A caller-supplied quality label never establishes verification.

Structural acceptance (a passing structural check) never satisfies a live
qualification obligation. Only evidence explicitly classified as a live
qualification can support a live qualification criterion.

Insufficient evidence never becomes successful qualification.

## Single Acceptance Authority

There is exactly **one** acceptance predicate. The explicit acceptance
operation and the canonical validator consult the same authority; there is
no second, competing acceptance rule.

An acceptance is admissible only when all of the following hold:

1. the item has at least one linked evidence record;
2. every linked evidence record is registered in the ledger;
3. every linked evidence record is **verified** — the artifact identity was
   actually checked, not merely asserted by a caller;
4. every linked evidence record is bound to **this** item;
5. every linked evidence record names a criterion **this** item declares;
6. **every** required acceptance criterion of the item is covered by
   admissible evidence;
7. when the item requires a live qualification gate, the supporting
   evidence carries the live qualification class.

An unrelated criterion never satisfies an obligation, and a partially
covered criterion set never establishes acceptance.

A cached verification flag never silently authorizes acceptance of a changed
artifact: the re-verification pass withdraws the verified status, and the
authority then rejects the item.

### Preserved Historical Acceptance

An accepted item whose evidence was later invalidated **keeps** its accepted
state. The historical acceptance is a fact about the past; withdrawing it
would erase the record that the obligation was ever met.

The invalidation is never silent: the canonical validator reports it as an
advisory (`acceptance_authority_invalid`), and the readiness and blocker
projections consult the same authority, so an invalidated acceptance no
longer satisfies a current downstream prerequisite.

## Obligation Conservation

An **unfinished** work item cannot be archived. Retiring it would make an
outstanding engineering obligation disappear from the live board while no
archived record shows it.

**Unfinished-work archival transfers are disabled.** A complete conservation
of a transferred obligation — every downstream edge, every retained
obligation, every exit criterion and every evidence reference — cannot be
established with a small, bounded implementation. The transfer request is
therefore **rejected outright**, not partially honoured.

A generic lifecycle update may never hide an unresolved obligation: a work
item whose acceptance obligation is outstanding cannot be moved into the
closed or archived state through the generic update path.

A terminal ACT verdict of `CLOSED` does **not** close the engineering
obligation it touched. See "Terminal ACT Does Not Close Obligations".

An ACT may address several work items. Each keeps its own acceptance
state.

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

A reopen is distinct from a restore: a restore brings an **archived**
record back into the live state; a reopen resets the lifecycle of a **live**
record.

A reopen makes the work **actionable again**: the lifecycle returns to
`open` and the acceptance state returns to `pending`, so the obligation is
live and the item is selectable again. The historical acceptance
**decision** (who accepted, on what basis, against which source revision)
is preserved verbatim. The preserved decision is not authority: the
acceptance authority re-evaluates the item against the current evidence.

## Mutation Authority

- An externally initiated mutation of an **existing** record requires a
  **positive** expected revision. A serialized internal transaction does
  not substitute for the caller's expected revision.
- A stale revision is rejected rather than silently overwritten.
- A semantic no-op does not increment the record revision and does not
  rewrite canonical state.
- Ownership is never replaced silently. It is replaced by an explicit,
  bounded transfer that names the expected current owner. A **claim** never
  replaces an existing owner.
- The generic update path can never establish acceptance. Acceptance is
  exercised by the explicit acceptance operation only.
- An explicit acceptance names the accepting actor. Leamas never
  autonomously determines whether a live experiment actually occurred.

## Transaction Exclusion

The ledger storage is guarded by an operating-system advisory file lock.
Every acquisition opens its own file description and takes its own advisory
lock: there is **no** process-wide reentrancy flag.

Consequently:

- two goroutines of one process that both try to write are mutually
  exclusive;
- two processes that both try to write are mutually exclusive;
- a reader never enters while a writer of the same process publishes;
- a reader never enters while a writer of another process publishes;
- a recovery takes the **same** exclusive authority as a writer, so it can
  never roll back an active foreign publication.

An internal rollback that must reuse an already-held lock receives that
ownership **explicitly** (`Transaction.Ownership` →
`Store.RecoverWithOwnership`). The lock never infers reentrancy.

The pending journal state is checked **under the acquired lock**. A
canonical read and a new transaction are both refused while an interrupted
publication is pending: the visible state would be a partial transaction
and must not be read as if it were an accepted state.

An interrupted writer never exposes an accepted partial state: the recovery
restores the pre-transaction snapshots deterministically, and a repeated
recovery reports the same terminal state.

## Readiness

Execution readiness, execution authorization, acceptance readiness and
archival eligibility are separate concerns.

Readiness answers "may this work be started?". A work item with unmet
acceptance criteria is ready: it is precisely the work that still has to
be done, and it must be able to acquire its own evidence. A live
qualification obligation is executable before the live qualification
evidence exists.

Deferred and explicitly blocked work is never selected. An unresolved
blocking dependency keeps its dependent item non-executable. An unknown
source status keeps its imported record durably non-executable until it is
explicitly reconciled.

The epic and owner filters are selection controls. They narrow the result
set; they never authorize execution.

## Determinism

Identical canonical input produces byte-identical rendered output. The
only explicitly separate value is the presentation timestamp, which is
emitted in its own field and excluded from the byte-identity comparison.

Read commands never require provider credentials or network access.

## Migration

`leamas ledger import` is a dry-run-first import and reconciliation
primitive. Import:

- preserves source record identity and provenance
- produces explicit ambiguity and unresolved-reference reports
- is idempotent
- never promotes a conversational claim into a repository-verified fact

Every resolved dependency becomes a canonical typed edge. Every unresolved
blocking dependency becomes a durable blocker that keeps its dependent
item non-executable.

Conflicting source records are quarantined as a complete identity group.
Input order never selects a winner, and the apply path fails closed while
such a group is present.

The exact source status and the original row provenance are preserved. An
unknown source status stays explicitly unresolved and its record stays
durably non-executable until it is explicitly reconciled. Scoped
identities remain distinct across epics. Existing epic metadata is not
silently overwritten.

`leamas ledger reconcile` reports the conservation receipt: nodes,
relationships, blockers, original dispositions, criteria and the explicit
reconciliation records. An imported work-item count is not a conservation
proof.

### Import Conservation

The importer compares the **incoming normalized source record** with the
**preserved canonical record** — identity AND content — *before* it filters
an existing identity out. Every source identity receives exactly one
reconciliation outcome:

| Outcome | Meaning |
|---------|---------|
| `created` | the identity did not exist in the canonical state |
| `unchanged` | the replay produced no canonical delta |
| `source_changed` | the source record changed; the canonical record is conserved and the change requires explicit reconciliation |
| `conflict` | the source records of one identity disagree; the whole group is quarantined |
| `unresolved` | the identity, status or dependency is unresolved |

A changed historical source **never** overwrites the canonical work merely
because the historical source changed: the change is an explicit
reconciliation.

Conservation compares the original source, the normalized records and the
actual persisted state — not merely the count of planned records.

The preserved source provenance is complete: the original row text, the
source locator, the exact source status, the priority, the workstream and
the available provenance detail.

### The Original IECR Board Fixture

The original IECR board intake fixture is a **pasted-text board capture**.
Its schema is NOT the repository-native source board schema. A minimum
necessary **versioned adapter** reads it:

```text
leamas ledger import-original --source <path> [--apply]
```

The adapter is conservative:

- the exact source status is conserved verbatim, including the UPPERCASE
  spelling of the original board;
- the original row text, the line locator and the free-form detail are
  preserved;
- the parent epic identity is preserved: the workstreams do **not** become
  eight independent epics;
- no dependency, no qualification outcome and no evidence authority is
  inferred.

The original fixture is **not** the full historical repository board. The
missing historical information is reported explicitly and remains
explicitly unavailable.

## Operational Projections

The operational views are **projections**: identical canonical input
produces byte-identical output. They are never a second source of truth.

| View | Scope |
| --- | --- |
| `board` | the compact global operational board |
| `epic` | one epic scope |
| `workstream` | one workstream scope, narrower than the epic |
| `ready` | the executable work |
| `blockers` | the blocking reasons, with the same acceptance authority |
| `history` | the terminal ACT verdicts |
| `authority` | the acceptance-authority warnings |
| `stats` | the ledger statistics |

The default operational board prioritizes **current work**. The historical
narrative stays in the source board, reachable through the source line
locator of each record: the canonical ledger carries structured state and
references, never a copy of the narrative.

The `authority` view reports every accepted work item whose acceptance is
not backed by currently admissible evidence. The historical acceptance is
preserved: the warning is a **report**, never a retraction.

## Migration

`leamas ledger import` is a dry-run-first import and reconciliation
primitive. Import:

- preserves source record identity and provenance
- produces explicit ambiguity and unresolved-reference reports
- is idempotent
- never promotes a conversational claim into a repository-verified fact

The historical board migration, the persisted conservation contract and the
command surface are documented in
[work-ledger-migration.md](./work-ledger-migration.md).

The migration interchange contract (FWMI v1) — the source-bound interchange
package a separate agent produces for a legacy board, and the deterministic
adapter that imports it — is documented in
[work-ledger-interchange.md](./work-ledger-interchange.md).

## References

- `internal/ledger/` - implementation
- `internal/ledger/domain/` - pure domain core
- `docs/adr/0007-factory-work-ledger.md` - decision record
- `docs/factory/work-ledger-migration.md` - historical migration contract

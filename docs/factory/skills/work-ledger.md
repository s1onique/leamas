# Skill: Factory Work Ledger

**Status**: Reusable guidance. The skill invokes deterministic commands.
It is **not** the enforcement authority. The ledger CLI and the domain
validator are the enforcement authority.

## When to Load Scoped Board Context

Load the scoped board context when you need to know what work exists in a
repository before you start.

```bash
leamas ledger prime --epic <EPIC-ID>
```

`prime` renders the minimal agent bootstrap context: the live work items
of the scope, their lifecycle and acceptance state, their acceptance
criteria and their blockers. It reads only the canonical ledger records. It
never requires provider credentials or network access.

Use `--max-tokens` to bound the context. When the budget is exceeded the
command says so explicitly; it never silently truncates a required
obligation.

To pick the next item to work on:

```bash
leamas ledger next --epic <EPIC-ID>
```

`next` applies the **execution readiness** rules: the item is not archived,
its lifecycle is not blocked or closed, its acceptance is not terminal and
not deferred, and every blocking dependency is resolved AND accepted.

A work item with unmet acceptance criteria IS ready: it is precisely the
work that still has to be done, and it must be able to acquire its own
evidence. A live qualification obligation is executable before the live
qualification evidence exists — running the qualification is the work.

The `--epic` and `--owner` filters are **selection controls**. They narrow
the result set; they never authorize execution.

## When to Record a Durable Semantic Checkpoint

Record a checkpoint when you have made durable progress on a work item
and want that progress to survive the session.

```bash
leamas ledger checkpoint --id <EPIC-ID>/<ITEM-ID> --lifecycle in_progress --summary "<what changed>"
```

A checkpoint records progress. It never accepts the item. Acceptance is a
separate, explicit decision that requires sufficient evidence.

Claim an item before you start it:

```bash
leamas ledger claim --id <EPIC-ID>/<ITEM-ID> --owner <owner>
```

Always pass `--expect-revision` when you update a record you loaded
earlier. A stale revision is rejected rather than silently overwritten.

## How to Link Evidence

Bind a repository-owned evidence reference to a work item:

```bash
leamas ledger evidence add --id <EVIDENCE-ID> --kind file \
  --title "<title>" --relative-path <repo-relative-path> \
  --sha256 <digest> --item <EPIC-ID>/<ITEM-ID> \
  --class <structural|live_qualification|review> \
  --criterion "<acceptance criterion>"
```

When a relative path and a digest are declared, the artifact identity is
**actually verified** against the repository: the file must exist at the
declared path, its byte size must match and its SHA-256 must match. A
missing, incorrect or unavailable artifact never acquires verified
integrity, and the registration is rejected.

Four facts are recorded separately and must not be conflated:

| Fact | Recorded by |
|------|-------------|
| Artifact integrity | `--relative-path` and `--sha256` (verified) |
| Evidence class | `--class` |
| Claim binding | `--criterion` and the work item binding |
| Engineering acceptance | An explicit `accept` decision |

A structural pass never satisfies a live qualification obligation. If the
work item sets `--requires-live-gate`, only evidence of the
`live_qualification` class can satisfy it — verification alone is not
enough.

Re-check the artifact identity when the repository may have moved on:

```bash
leamas ledger evidence revalidate [--id <EVIDENCE-ID>]
```

An evidence record whose artifact changed, was removed or was replaced
loses its verified integrity and can no longer support an acceptance.

## How to Close an ACT

Record the terminal ACT verdict:

```bash
leamas ledger close --act <ACT-ID> --item <EPIC-ID>/<ITEM-ID> \
  --verdict closed --terminal-disposition "<original disposition>"
```

A terminal ACT verdict **does not close the engineering obligation**. The
linked work item keeps its own acceptance state, and the ACT record states
explicitly that the obligation is retained.

If you mean to accept the engineering work, do that separately and only
with admissible evidence:

```bash
leamas ledger accept --id <EPIC-ID>/<ITEM-ID> \
  --accepted-by <actor> --expect-revision <revision>
```

`--accepted-by` is required: the engineering acceptance is an explicit,
scoped act of the caller. Leamas never autonomously determines whether a
live experiment actually occurred.

There is exactly ONE acceptance authority. The acceptance is only permitted
when the linked evidence is admissible:

- every linked evidence record is registered and actually **verified**;
- every linked evidence record is bound to **this** item;
- every linked evidence record names a criterion **this** item declares;
- **every** required acceptance criterion of the item is covered;
- when the item requires a live qualification gate, the supporting evidence
  carries the live qualification class.

An unrelated criterion never satisfies an obligation. A caller-supplied
quality label never establishes verification, and the generic `update` path
can never establish acceptance.

## How Unresolved Obligations Survive Archival

Archive a work item that is done with:

```bash
leamas ledger archive --id <EPIC-ID>/<ITEM-ID> --reopen-condition <condition>
```

Archival preserves the stable identity, the original terminal
disposition, the evidence references, the dependency relationships, the
required successor obligations and the applicable reopen conditions. The
record stays discoverable:

```bash
leamas ledger list --archived
leamas ledger show <EPIC-ID>/<ITEM-ID>
```

An **unfinished** work item is not archivable. Retiring it would make an
outstanding engineering obligation disappear from the live board while no
archived record shows it.

**Unfinished-work archival transfers are disabled.** A complete conservation
of a transferred obligation cannot be established with a small, bounded
implementation, so a `--transfer-to` request is **rejected outright**
rather than partially honoured. Accept the item first.

Restore it with:

```bash
leamas ledger restore --id <EPIC-ID>/<ITEM-ID>
```

The stable identity is preserved. The restore refuses to reuse an
identity that is already live.

Reopen a live record explicitly:

```bash
leamas ledger reopen --id <EPIC-ID>/<ITEM-ID>
```

A reopen makes the work **actionable again**: the lifecycle returns to
`open` and the acceptance state returns to `pending`, so the item is
selectable again. The historical acceptance **decision** (who accepted, on
what basis) is preserved; it is not authority, so the item must acquire
fresh admissible evidence before it can be accepted again.

## How Ownership Changes

Ownership is never replaced silently. Transfer it explicitly:

```bash
leamas ledger transfer --id <EPIC-ID>/<ITEM-ID> \
  --from <current-owner> --owner <new-owner>
```

The `--from` owner must match the stored owner exactly, so a transfer
cannot be applied to a record whose ownership changed in the meantime.

## How to Retrieve Historical Records

Terminal ACT verdicts are retrievable as a history view:

```bash
leamas ledger render --view history
```

An individual record is retrievable with `show`. Archived work items
remain in the canonical state and are therefore always retrievable.

```bash
leamas ledger evidence lookup --id <EVIDENCE-ID>
leamas ledger decision show --id <DECISION-ID>
leamas ledger knowledge show --id <KNOWLEDGE-ID>
```

## How to Import and Reconcile a Board

Import is dry-run first:

```bash
leamas ledger import --source <board.json>
leamas ledger import --source <board.json> --apply
```

The import conserves source identity and provenance, reports ambiguities,
unresolved references and quarantined groups explicitly, never promotes a
conversational claim into a repository-verified fact, and is idempotent.

The conservation receipt reports the conserved SEMANTICS — nodes,
relationships, blockers, original dispositions, criteria and the explicit
reconciliation records — not only the imported record count:

```bash
leamas ledger reconcile --source <board.json>
```

A conflicting source identity group is **quarantined as a complete
group**: input order never selects a winner, and the apply path fails
closed while such a group is present.

Every source identity receives exactly one explicit reconciliation outcome
(`created`, `unchanged`, `source_changed`, `conflict`, `unresolved`). A
changed historical source **never** overwrites the canonical work: the
change is reported for explicit reconciliation.

The original IECR board capture has its own schema. Use the versioned
adapter:

```bash
leamas ledger import-original --source <original-fixture.json>
leamas ledger import-original --source <original-fixture.json> --apply
```

The adapter preserves the exact source status, the original row text, the
line locator and the parent epic identity. The workstreams do NOT become
eight independent epics. The original fixture is not the full historical
repository board: the missing historical information is reported
explicitly.

## Command Summary

| Situation | Command |
|-----------|---------|
| Load scoped context | `leamas ledger prime --epic <EPIC>` |
| Find ready work | `leamas ledger next` |
| Claim an item | `leamas ledger claim --id <ID> --owner <owner>` |
| Transfer ownership | `leamas ledger transfer --id <ID> --from <old> --owner <new>` |
| Record progress | `leamas ledger checkpoint --id <ID>` |
| Link evidence | `leamas ledger evidence add --id <EV> --item <ID>` |
| Re-check evidence | `leamas ledger evidence revalidate [--id <EV>]` |
| Accept work | `leamas ledger accept --id <ID> --accepted-by <actor>` |
| Record an ACT verdict | `leamas ledger close --act <ACT> --item <ID>` |
| Archive (finished work) | `leamas ledger archive --id <ID>` |
| Restore | `leamas ledger restore --id <ID>` |
| Reopen | `leamas ledger reopen --id <ID>` |
| Record a decision | `leamas ledger decision add --id <ID> --title <title>` |
| Record knowledge | `leamas ledger knowledge add --id <ID> --title <title>` |
| Import a board | `leamas ledger import --source <file>` |
| Import the original IECR board | `leamas ledger import-original --source <file>` |
| Report the conservation receipt | `leamas ledger reconcile --source <file>` |
| Retrieve history | `leamas ledger render --view history` |
| Validate the state | `leamas ledger validate` |

## References

- `docs/factory/work-ledger.md` — the contract
- `docs/adr/0007-factory-work-ledger.md` — the decision record

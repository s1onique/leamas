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

`next` applies the readiness rules: the item is not archived, its
lifecycle is not blocked or closed, its acceptance is not terminal, every
blocking dependency is accepted, its acceptance criteria have at least
referenced evidence, and it does not wait on an unsatisfied live
qualification gate.

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
  --sha256 <digest> --item <EPIC-ID>/<ITEM-ID> --quality referenced
```

Three facts are recorded separately and must not be conflated:

| Fact | Recorded by |
|------|-------------|
| Artifact identity | `--relative-path` and `--sha256` |
| Claim relevance | `--quality` and the work item binding |
| Engineering acceptance | An explicit acceptance decision |

A structural pass never satisfies a live qualification obligation. If the
work item sets `--requires-live-gate`, only `verified` evidence quality
can satisfy it.

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
with sufficient evidence:

```bash
leamas ledger update --id <EPIC-ID>/<ITEM-ID> --acceptance accepted \
  --expect-revision <revision>
```

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

Restore it with:

```bash
leamas ledger restore --id <EPIC-ID>/<ITEM-ID>
```

The stable identity is preserved. The restore refuses to reuse an
identity that is already live.

## How to Retrieve Historical Records

Terminal ACT verdicts are retrievable as a history view:

```bash
leamas ledger render --view history
```

An individual record is retrievable with `show`. Archived work items
remain in the canonical state and are therefore always retrievable.

## Command Summary

| Situation | Command |
|-----------|---------|
| Load scoped context | `leamas ledger prime --epic <EPIC>` |
| Find ready work | `leamas ledger next` |
| Claim an item | `leamas ledger claim --id <ID> --owner <owner>` |
| Record progress | `leamas ledger checkpoint --id <ID>` |
| Link evidence | `leamas ledger evidence add --id <EV> --item <ID>` |
| Record an ACT verdict | `leamas ledger close --act <ACT> --item <ID>` |
| Archive | `leamas ledger archive --id <ID>` |
| Restore | `leamas ledger restore --id <ID>` |
| Retrieve history | `leamas ledger render --view history` |
| Validate the state | `leamas ledger validate` |

## References

- `docs/factory/work-ledger.md` — the contract
- `docs/adr/0007-factory-work-ledger.md` — the decision record

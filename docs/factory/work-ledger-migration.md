# Factory Work Ledger — Historical Migration

### The Actual Historical Board

The ClineMM historical board has its own topology: a navigation index of
ACT report sections, a current-frontier lane table and per-epic detail
files. A **versioned adapter** reads it:

```text
leamas ledger migrate-board --source-root <repo> --repository <name> \
  --source-head <oid> [--apply]
```

The adapter is conservative:

- a board section is an ACT **report**, never a work item: the board is a
  navigation index, not an archive of obligations;
- the current-frontier lane table is the authoritative **current** state:
  every lane becomes a current obligation;
- the workstream identity of an epic file is conserved explicitly and is
  distinct from the epic scope;
- the exact source status is conserved verbatim; an unrecognised status
  stays unresolved and non-executable;
- no dependency, no qualification outcome and no evidence authority is
  inferred.

Every extracted section carries its frozen provenance: the source path, the
line range, the content digest and the stable section locator.

The migration is dry-run first. The applied migration persists the durable
migration reconciliation records: every source record that could **not** be
accounted for becomes durable migration state, never a console warning
only.

### Persisted Conservation

The conservation receipt has two forms:

- `BuildReceipt` reports what the plan carried over. It does **not** claim
  persisted conservation: it has no access to the persisted state.
- `BuildPersistedReceipt` compares the plan with the **actual persisted
  state** — identity, criteria, dispositions, the workstream identity and
  the relationship graph — and reports every load-bearing difference as an
  explicit discrepancy.

A persisted state that lost an obligation, a relationship, a criterion or a
disposition reports `conserved: false`. The applied migration exits
non-zero in that case.

The comparison covers EVERY canonical record family, not only the families
the import writes:

- the epic scopes, the work items, the indexed historical ACT executions and
  the durable migration reconciliation records are the families the import
  creates;
- the evidence records, the decision records and the knowledge references
  are **not** modified by an import: the expected post-import state
  preserves them exactly, so a persisted state that lost one of them, that
  carries an unauthorized addition or that changed any field of one is
  reported as a discrepancy.

### Source Fingerprints

Every imported record carries an immutable **normalized source
fingerprint**. It covers the normalized source content AND the normalized
source relationship graph: a source dependency that names a local identity
is expanded to its scoped identity, and the expanded set is sorted and
deduplicated.

The fingerprint is what lets an import distinguish three states:

- an **exact historical replay** — unchanged;
- a **changed historical source** — reported, never applied as a silent
  overwrite;
- a **legitimate canonical evolution** — the canonical record changed, the
  source did not, so the replay is still unchanged.

### Durable Migration State

The unresolved reconciliation records are persisted under
`.leamas/ledger/migrations/`. They are the durable evidence that an
unresolved source record was not silently dropped.

An unresolved RELATIONSHIP keeps its COMPLETE metadata in the durable
record: the relationship kind, the outcome the relationship required and
the byte-exact source locations of the edge. A durable record that carries
only the target and a generic kind has lost the obligation.

The identity of a migration record is derived from its **content**: the
same source record always produces the same migration record, so a replay
never creates a second one.

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
leamas ledger transfer
leamas ledger link
leamas ledger evidence add
leamas ledger evidence lookup
leamas ledger evidence revalidate
leamas ledger checkpoint
leamas ledger close
leamas ledger accept
leamas ledger reopen
leamas ledger decision add
leamas ledger decision show
leamas ledger knowledge add
leamas ledger knowledge show
leamas ledger archive
leamas ledger restore
leamas ledger render --view board
leamas ledger render --view epic --epic <EPIC-ID>
leamas ledger render --view workstream --workstream <name>
leamas ledger render --view ready
leamas ledger render --view blockers
leamas ledger render --view history
leamas ledger render --view authority
leamas ledger render --view stats
leamas ledger validate
leamas ledger stats
leamas ledger import
leamas ledger import-original
leamas ledger migrate-board
leamas ledger reconcile
leamas ledger recover
```

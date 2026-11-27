# Factory Work Migration Interchange (FWMI v1)

**Status**: adapter implemented and qualified against a bounded synthetic
multi-epic fixture; the real-source qualification is pending.

FWMI v1 is a MIGRATION INPUT format. It is not a second canonical ledger
format and it never becomes one.

The historical semantic reconciliation of a legacy engineering board is
produced by a separate agent (the migration agent), which reads the source
repository and produces a source-bound interchange package. Leamas owns the
DETERMINISTIC half: it validates the interchange records strictly and
applies them to the existing ledger records through the existing importer,
store, validation and transaction architecture.

## Package layout

```text
.factory/migration/FWL02/
├── source-manifest.json
├── epics/<EPIC-ID>.json
├── historical-index.json
├── reconciliation.json
├── frontier.md
└── validation-receipt.json
```

Only the first four files are read by the adapter. `frontier.md` and
`validation-receipt.json` are operator-facing documents.

Every JSON file carries `schema_version: 1`.

## Command surface

| Step | Command |
|------|---------|
| Dry-run interchange validation | `leamas ledger import-interchange --source <package-dir>` |
| Shadow import | `leamas ledger import-interchange --source <package-dir> --apply` |
| Persisted conservation verification | `leamas ledger import-interchange --source <package-dir> --apply --json` (the `receipt.conserved` field) |
| Historical lookup | `leamas ledger interchange-history --id <ACT-ID>` |
| Exact replay | `leamas ledger import-interchange --source <package-dir> --apply --json` (the `delta` field must be empty) |

The import is a dry run by default. The applied import requires `--apply`.

## The preservation rules

The adapter is deliberately conservative.

| Rule | Mechanism |
|------|-----------|
| Source identities and provenance are preserved | `domain.SourceRef` on the epic, work item and historical ACT records; the source fingerprint covers them |
| A missing source repository establishes NO source binding | the interchange validator rejects a manifest that names no repository; the adapter never substitutes a placeholder for it |
| An unknown status stays distinct from an intentional deferral | an `UNRESOLVED` lifecycle maps to an UNKNOWN source status, which stays durably non-executable and explicitly reported |
| Unresolved migration records are preserved | the reconciliation manifest becomes durable `domain.MigrationRecord` records |
| Explicit dependencies and criteria are preserved | `DEPENDS_ON` becomes a typed blocking edge; the criteria become the acceptance criteria; the coverage fields distinguish "there are none" from "they were not recovered" |
| The EXPLICIT blockers of the source are preserved | `blockers` become `domain.WorkItem.SourceBlockers`; they are reported by the blocker projection and they PREVENT readiness |
| The historical ACT associations are preserved | `current_act` and `related_acts` become `SourceCurrentACT` and `SourceRelatedACTs`; they are historical associations and never execution authority |
| The historical evidence locators and the reopen conditions are preserved | `evidence_locators` become `SourceEvidenceLocators`; `reopen_conditions` become `domain.ReopenCondition` records |
| The load-bearing relationship content is preserved | a relationship keeps its `required_outcome` and its byte-exact `source_refs` on the canonical dependency edge |
| The EFFECTIVE source HEAD of every reference is retained | a reference that names no revision of its own is bound to the source HEAD of the package |
| The epic INVENTORY comes from the declared epic records | a scope the package declares stays discoverable even when it carries no outstanding work item |
| Historical ACT lookup is preserved | the historical index becomes ARCHIVED `domain.ActExecution` records with the original verdict conserved verbatim |
| An unresolved historical association is preserved explicitly | the association is conserved on `UnresolvedItemIDs` and reported as an unresolved reference |
| Original verdicts never become acceptance | the canonical verdict of an indexed historical attempt is `pending`; the source qualification assertion is conserved as an assertion and never sets the acceptance state |
| Cross-links do not duplicate authoritative obligations | only `DEPENDS_ON` is a local execution prerequisite; every other relationship kind becomes a non-blocking typed edge or an explicit unresolved reference |
| Distinct reconciliation records keep distinct identities | the reconciliation identity is derived from its content; a conflicting identity is rejected BEFORE publication |
| Unsupported or ambiguous state transitions are rejected | the interchange validator rejects unknown fields, unknown vocabulary values, duplicate identities, duplicate object members, invalid source ranges, malformed digests and unsupported nested schema versions |

## The adversarial discriminators

These are the observations a permissive adapter fails SILENTLY: each is only
observable through the persisted state, never through a plan. Each one has an
executable witness at the production seam it governs.

| # | Discriminator | Witness |
|---|---------------|---------|
| D1 | A missing source repository must not establish a verified source binding | `internal/ledger/fwmi/adversarial_test.go` |
| D2 | An unresolved reconciliation record must survive persistence | `internal/ledger/fwmi/adversarial_test.go` |
| D3 | Deleting a persisted epic must invalidate conservation | `internal/ledger/importer/fwl02_adversarial_test.go` |
| D4 | Changing a persisted priority must invalidate conservation | `internal/ledger/importer/fwl02_adversarial_test.go` |
| D5 | A changed historical replay must not silently erase the previous ACT records | `internal/ledger/importer/fwl02_adversarial_test.go` |

D3 and D4 are properties of the CONSERVATION RECEIPT: the receipt compares
the ACTUAL PERSISTED state with the plan, so a lost epic scope and a changed
priority are both reported as explicit discrepancies. A receipt that only
restated the planned records would report conservation for both.

## The FWL02-C discriminators

These are the completion discriminators of the interchange consumer. Each has
an executable witness at the production seam it governs.

| # | Discriminator | Witness |
|---|---------------|---------|
| C1 | The receipt compares the COMPLETE expected-versus-persisted state: the epics, the work items, the historical ACTs and the migration records | `internal/ledger/importer/fwl02_c_conservation_test.go` |
| C2 | An EXPLICIT blocker survives the import, stays visible in the blocker projection and prevents readiness | `internal/ledger/fwmi/fwl02_c_test.go` |
| C3 | Two DISTINCT reconciliation records with the same kind and resolution keep distinct identities, an exact replay duplicates nothing and a conflicting identity is rejected | `internal/ledger/fwmi/fwl02_c_test.go`, `cmd/leamas/ledger_interchange_c_test.go` |
| C4 | A source epic with no outstanding work item stays discoverable | `internal/ledger/fwmi/fwl02_c_test.go` |
| D1 | An unauthorized change to ANY semantic field of a freshly imported record — the lifecycle, the acceptance, the archival state, the criteria, the provenance — invalidates the receipt | `internal/ledger/importer/fwl02_c_conservation_test.go` |
| D2 | A CHANGED reconciliation record never overwrites the prior durable migration state; it is conserved and reported | `internal/ledger/importer/fwl02_d_boundary_test.go`, `internal/ledger/fwmi/fwl02_c_test.go`, `cmd/leamas/ledger_interchange_c_test.go` |

A preserved HISTORICAL terminal disposition is historical fact, never an
authority: it stays conserved on the record and it stays visible there, but
it may not retire the current unfinished obligation. The checkpoint path and
the generic update path share the ONE lifecycle-transition authority, and that
authority rejects a terminal transition while the acceptance obligation is
outstanding.

The qualification fixture is a bounded SYNTHETIC multi-epic package. It
proves the behaviour of the adapter against the interchange contract; it does
NOT prove a migration of any actual source.

## Strict decoding

Each interchange document is decoded as exactly ONE JSON object:

- unknown fields are rejected;
- a second document, a second scalar or trailing garbage is rejected;
- the EOF check is a second `Decode`;
- an AMBIGUOUS document is rejected: an object that names the same member
  twice has no single meaning. The same member name in DIFFERENT objects is
  legitimate.

## Validation

The strict validation rejects:

- a duplicate epic identity or a duplicate scoped work-item identity;
- a duplicate historical ACT identity;
- an unknown lifecycle, qualification, coverage or relationship-kind value;
- a source reference without a one-based line range or without a valid
  lowercase hexadecimal SHA-256 digest;
- an unsupported NESTED schema version.

The coverage semantics are EXPLICIT and are never inferred from the size of
the recovered collection:

- `COMPLETE` over an EMPTY collection means "the collection is known to be
  empty". It is a fact about the obligation and it is accepted.
- `UNKNOWN` means the collection was NOT recovered. It is never an assertion
  that the collection is empty, and the importer never invents the missing
  criteria or dependencies.

The PREREQUISITE coverage is reported explicitly when it is incomplete: a
record whose dependency coverage is not `COMPLETE` produces an explicit
`incomplete_dependency_coverage` report and a durable migration record. An
incomplete coverage is therefore always distinguishable from a verified
absence of prerequisites.

## The conservation receipt

The receipt compares the EXPECTED state — the state before the import plus the
applied plan — with the ACTUAL PERSISTED state. The comparison covers the
WHOLE record families: the epic scopes, the work items, the indexed
historical ACT executions and the durable migration reconciliation records. It
also covers the unauthorized changes: a pre-existing record the import did not
plan must be preserved verbatim, and a record the import did not plan must
not appear.

The comparison uses the CANONICAL SERIALIZATION of the records, so EVERY
semantic field participates: the lifecycle state, the acceptance state, the
archival state, the provenance and the revision of a record are part of the
canonical record, and an unauthorized change to any of them is a conservation
failure.

A legitimate canonical EVOLUTION is not excluded either: it happened BEFORE
the import, so the pre-import state already carries it, the expected state
carries it too, and the replay does not overwrite it.

## The reconciliation replay

The durable migration records the package carries participate in the PLAN:

| Case | Outcome |
|------|---------|
| an IDENTICAL record | a no-op: the canonical state already carries it |
| a CHANGED record | the prior durable migration state is CONSERVED and the change is reported as an explicit reconciliation |
| a NEW record | it is created |

A durable migration record is the persisted evidence that a source record
could not be reconciled. It is never silently overwritten.

### What the adapter does NOT verify

The adapter does NOT read the source repository. The source manifest records
the identity, size, line count, digest and blob identity of every examined
source file, and the operator verifies those digests against the source
repository EXTERNALLY, in the source-bound qualification pass. The adapter
verifies the INTERNAL consistency of the interchange package: the identity
of every record, the vocabulary of every value, the validity of every source
reference range and the accounting of every obligation.

## Authority boundary

`SOURCE_RECONCILIATION_READY` is NOT cutover approval. The interchange import
never upgrades the authority of existing evidence and never establishes a new
engineering acceptance.

## References

- `internal/ledger/fwmi/` - the interchange contract, loader, validator and
  adapter
- `internal/ledger/importer/` - the deterministic plan, the apply path and the
  conservation receipt
- `docs/factory/work-ledger.md` - the canonical ledger semantics

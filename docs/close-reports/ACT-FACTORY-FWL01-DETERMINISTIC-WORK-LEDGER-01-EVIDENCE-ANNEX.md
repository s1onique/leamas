# ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01 — Evidence Annex

**Date**: 2026-11-26
**Status**: Closed
**Parent report**: `ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01.md`

This annex carries the adversarial witness matrix and the non-vacuity
evidence for the parent close report.

## Adversarial Witnesses

All eighteen required witnesses are implemented as executable tests.

| # | Witness | Test |
|---|---------|------|
| 1 | Duplicate work-item identity rejected | `TestServiceCreateItemRejectsDuplicateIdentity`, `TestValidateLedgerDuplicateIdentityRejected` |
| 2 | Dangling dependency rejected / explicitly unresolved during import | `TestServiceLinkRejectsDanglingDependency`, `TestValidateLedgerDanglingDependencyRejected`, `TestImportReportsUnresolvedReference` |
| 3 | Blocking dependency cycle rejected | `TestServiceLinkRejectsBlockingCycle`, `TestValidateLedgerBlockingCycleRejected` |
| 4 | Insufficient evidence cannot produce successful acceptance | `TestServiceInsufficientEvidenceCannotAccept` |
| 5 | Structural pass cannot close a required live gate | `TestServiceStructuralPassCannotCloseLiveGate` |
| 6 | Terminal ACT with unfinished obligation retains that obligation | `TestServiceTerminalActRetainsUnfinishedObligation`, `TestLedgerCLITerminalActRetainsObligation` |
| 7 | Archive preserves evidence and successor references | `TestServiceArchivePreservesEvidenceAndSuccessors` |
| 8 | Restore preserves stable identity | `TestServiceRestorePreservesStableIdentity` |
| 9 | Stale revision cannot overwrite accepted newer state | `TestServiceUpdateRejectsStaleRevision`, `TestServiceStaleRevisionDoesNotMutateState` |
| 10 | Interrupted multi-record operation cannot expose an accepted partial transaction | `TestTransactionRecoveryRestoresPreTransactionState` |
| 11 | Replaying a transaction does not duplicate records or evidence | `TestReplayTransactionDoesNotDuplicateRecords` |
| 12 | Conflicting historical statuses remain unresolved rather than silently promoted | `TestConflictingHistoricalStatusesRemainUnresolved`, `TestLedgerCLIImportReportsAmbiguity` |
| 13 | Identical records generate identical board output | `TestBoardRenderIsByteIdenticalForIdenticalInput`, `TestRenderIsStableAcrossRepeatedCalls` |
| 14 | Repeated migration yields no unintended canonical delta | `TestImportIsIdempotent` |
| 15 | Evidence path traversal and invalid artifact identity rejected | `TestLedgerCLIRejectsEvidencePathTraversal`, `TestValidateLedgerPathTraversalRejected`, `TestValidateLedgerInvalidDigestRejected` |
| 16 | Default read commands do not require provider credentials or network access | `TestLedgerCLIReadCommandsNeedNoCredentials` |
| 17 | Ready-work selection respects acceptance requirements and authorization boundaries | `TestReadyViewRespectsAcceptanceRequirements`, `TestReadyViewRespectsEpicScope` |
| 18 | Existing protected evidence remains unchanged | `TestLedgerCLIProtectedEvidenceRemainsUnchanged` |

### Non-vacuity

Every load-bearing semantic check has a rejected counterexample AND a
positive control:

| Rule | Rejected counterexample | Positive control |
|------|------------------------|------------------|
| Duplicate identity | duplicate work item / epic | distinct identities accepted |
| Blocking cycle | two-item cycle | relates-edge cycle accepted |
| Sufficient evidence | `declared` quality + criteria | `referenced` quality accepted |
| Live gate | `referenced` quality + live gate | `verified` quality accepted |
| Terminal ACT | terminal verdict without retention marker | terminal verdict with accepted obligation |
| Path traversal | `../outside/secret.txt` | repository-relative path accepted |
| Restore | restoring a live record | restoring an archived record |
| Stale revision | `expect-revision` mismatch | matching revision accepted |
| Ready selection | item with unsatisfied criteria | item with satisfied criteria |
| Import idempotency | first import delta | replay delta empty |

## Remaining Limitations

1. **The IECR intake fixture was not supplied in this environment.** The
   ACT refers to a supplied IECR intake fixture. No such file exists in
   the Leamas, ClineMM or Factory repositories on this host, and no file
   matching the distinctive `CLOSED_INSUFFICIENT` status exists anywhere
   locally. The fixture was therefore CONSTRUCTED to the declared
   inventory (44 items, 8 workstreams, 10 CLOSED, 1 CLOSED_INSUFFICIENT,
   33 unfinished, 6 BLOCKED, 12 exit criteria) and committed as
   repository-owned test data. The importer itself is source-agnostic and
   was NOT specialised to this fixture.

2. **No ClineMM runtime hooks were introduced**, per the explicit
   non-goals. The ClineMM board (`.factory/epic-board.md`, 44,340 lines)
   remains untouched.

3. **No destructive historical ClineMM migration was performed.** Only the
   dry-run-capable import primitive exists.

4. **The ACT execution record is bound to one work item.** Multi-item ACT
   bindings are not modelled.

5. **The ledger does not replace the narrative ACT close report.** It
   records machine-checkable state; the report remains the human
   narrative.

6. **`leamas ledger recover` is not exposed as a CLI subcommand.** The
   recovery primitive is exercised by tests; operators currently trigger
   recovery implicitly on the next mutation.

## Exact Migration Successor Requirements

The successor ACT must supply these before the historical board can be
migrated and cut over:

1. **A named, versioned source-board format.** The successor must state
   the exact parser contract for the historical board, including the
   canonical source-identity field and the canonical status vocabulary.
2. **A source-identity mapping.** Every historical record must carry a
   stable source identity that the importer can preserve verbatim. The
   importer preserves `source_id`; the successor must state which field
   supplies it.
3. **A status vocabulary mapping.** Every historical status must map to
   exactly one of the ledger source statuses, or be declared unresolved.
   No status may be silently promoted.
4. **A dependency-reference resolution rule.** Every historical
   dependency must resolve to a stable source identity, or be reported as
   unresolved. Dangling references may not be dropped.
5. **An epic/workstream identity list.** The successor must state the
   exact workstream identities and their titles.
6. **An acceptance-authority statement.** The successor must state which
   historical records, if any, carry repository-verified evidence. Records
   without it must import as conversational.
7. **A cutover order.** The successor must state the exact order of:
   dry run, review of the ambiguity and unresolved reports, apply,
   re-validation, and the point at which the historical board stops
   being authoritative.
8. **A rollback rule.** The successor must state how a partially cut-over
   board is reverted to the ledger without losing an obligation.

## Follow-up ACTs

- ACT-CLINEMM-FWL02-HISTORICAL-BOARD-MIGRATION-AND-CUTOVER-01 (the
  declared successor).

## Files Changed

### New — production code

```text
cmd/leamas/ledger.go
cmd/leamas/ledger_import.go
cmd/leamas/ledger_read.go
cmd/leamas/ledger_recover.go
cmd/leamas/ledger_write.go
cmd/leamas/ledger_write2.go
internal/ledger/domain/identity.go
internal/ledger/domain/model.go
internal/ledger/domain/path.go
internal/ledger/domain/readmodel.go
internal/ledger/domain/ready.go
internal/ledger/domain/records.go
internal/ledger/domain/validate.go
internal/ledger/domain/validate_records.go
internal/ledger/importer/apply.go
internal/ledger/importer/importer.go
internal/ledger/render/bootstrap.go
internal/ledger/render/render.go
internal/ledger/service/archive.go
internal/ledger/service/lifecycle.go
internal/ledger/service/relations.go
internal/ledger/service/service.go
internal/ledger/store/atomic.go
internal/ledger/store/load.go
internal/ledger/store/mutate.go
internal/ledger/store/store.go
internal/ledger/store/txn.go
```

### New — tests and test data

```text
cmd/leamas/ledger_cli_lifecycle_test.go
cmd/leamas/ledger_cli_test.go
internal/ledger/domain/validate_test.go
internal/ledger/importer/fixture_test.go
internal/ledger/importer/importer_test.go
internal/ledger/importer/testdata/iecr-intake-board.json
internal/ledger/render/render_test.go
internal/ledger/service/service_test.go
internal/ledger/store/txn_test.go
```

### New — documentation

```text
docs/adr/0007-factory-work-ledger.md
docs/close-reports/ACT-FACTORY-FWL01-DETERMINISTIC-WORK-LEDGER-01.md
docs/factory/skills/work-ledger.md
docs/factory/work-ledger.md
```

### Modified

```text
cmd/leamas/main.go
```

## Skipped or Deferred Checks

```text
make factorize    NOT RUN — not authorized by this ACT
make gate-dupcode NOT RUN — not authorized by this ACT
make gate         NOT RUN — not authorized by this ACT
```

`make gate-fast` was executed and FAILED for the reason recorded above:
five pre-existing `cmd/leamas` tests require a clean caller worktree and
cannot pass while the ACT is being implemented. No ACT-owned test failed.

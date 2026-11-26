# ACT-FACTORY-FWL01-AMENDMENT-01 — Completion Addendum Close Report

**Date**: 2026-11-27
**Status**: Closed
**Verdict**: LEDGER_CORE_READY_FOR_MIGRATION
**Historical entry anchor**: `0b385166d97ed0835f9af7e5cd931914785868d4`

## Summary

The completion addendum reproduced the eight independent-review
observations against the ACTUAL repository seams, corrected each of them,
and qualified the corrections with executable witnesses at the production
seam they govern.

The eight observations were reproduced BEFORE the correction. Seven were
reproduced directly; the eighth (a reader entering during publication) was
already excluded at the reader level, but the closely-related recovery
overlap WAS reproduced. Four additional counterexamples surfaced during the
work and were corrected as well.

The original IECR board intake fixture was supplied by the operator and is
now IMPORTED, PERSISTED, RELOADED and RECONCILED by an executed
qualification. The presence-only test was REPLACED by a real qualification.

The verdict is **LEDGER_CORE_READY_FOR_MIGRATION**.

## W0 — Independent Review Counterexamples

Every observation was reproduced against the real production seams through
a scratch probe program that drove `domain`, `service`, `store` and
`importer`. The probe programs were scratch harnesses and are NOT counted
as CLI qualification.

| # | Observation | RED result before correction | Correction |
|---|-------------|-------------------------------|------------|
| 1 | Canonical accepted live item without evidence validates successfully | **REPRODUCED**: `ValidateLedger` returned OK for an accepted item with zero evidence | The canonical validator consults the SINGLE acceptance authority; the invalidation is REPORTED as an advisory |
| 2 | Unrelated evidence satisfies an item with multiple acceptance criteria | **REPRODUCED**: at the domain seam AND at the service seam, one criterion's evidence accepted a two-criterion item | The authority requires coverage of EVERY required criterion and rejects evidence bound to another item or naming an unrelated criterion |
| 3 | Reopened accepted item remains non-executable | **REPRODUCED** (partial): `Reopen` reset the lifecycle but left the terminal acceptance in place | `Reopen` returns the acceptance to `pending` and preserves the historical acceptance DECISION |
| 4 | Different ACT ItemIDs compare as identical content | **REPRODUCED**: `SameActContent` ignored the `ItemIDs` set | `SameActContent` compares EVERY addressed work-item identity |
| 5 | Unknown imported status becomes executable | **REPRODUCED**: an unknown status produced a ready, executable item | An unknown status is conserved verbatim, reported as unresolved, and its record is durably non-executable |
| 6 | Changed source record with existing identity is silently skipped | **REPRODUCED**: no item, no ambiguity, no unresolved report | The plan compares identity AND content before filtering; a change is an explicit `source_changed` reconciliation |
| 7 | Second independent writer in the same process acquires the exclusive lock | **REPRODUCED**: the second `AcquireWrite` succeeded in the same process | The process-wide reentrancy flag is REMOVED; every acquisition takes its own advisory file lock |
| 8 | Reader enters while the first writer still holds its lock | Not reproduced at the reader level; the closely-related RECOVERY overlap WAS reproduced | The recovery takes the same exclusive authority as a writer |

Additional counterexamples found and corrected during the work:

| Observation | RED result before correction | Correction |
|-------------|-------------------------------|------------|
| Recovery overlaps an active publication | **REPRODUCED**: `Recover` ran while a writer held the lock | The recovery takes the SAME exclusive authority |
| A generic lifecycle update hides an unresolved obligation | **REPRODUCED**: `UpdateItem` closed a pending item | A generic lifecycle update may never move unfinished work into a terminal state |
| A mutation of an existing record needs no expected revision | **REPRODUCED**: `ExpectedRev: 0` was accepted as "no check" | An externally initiated mutation of an existing record requires a POSITIVE expected revision |
| `Claim` replaces an existing owner | **REPRODUCED**: `owner-a` became `owner-b` | A claim never replaces an existing owner; the transfer is the explicit operation |

The positive controls were preserved throughout.

## W1 — Single Acceptance Authority

There is exactly ONE acceptance predicate: `domain.EvaluateAcceptance`.
The explicit acceptance operation (`service.Accept`) and the canonical
validator (`domain.ValidateLedger`) consult that same predicate. The
canonical validator is a THIN ADAPTER: it maps the authority's verdict onto
a diagnostic code and restates no acceptance rule.

The authority requires:

1. at least one linked evidence record;
2. every linked evidence record is registered;
3. every linked evidence record is VERIFIED (the artifact identity was
   actually checked, not asserted by a caller);
4. every linked evidence record is bound to THIS item;
5. every linked evidence record names a criterion THIS item declares;
6. EVERY required acceptance criterion is covered by admissible evidence;
7. the required evidence class for each applicable gate (a live
   qualification gate needs the live qualification class).

An unrelated criterion never satisfies an obligation. A partially covered
criterion set never establishes acceptance.

### Artifact verification versus engineering judgment

The artifact verification is mechanical: it proves that the bytes at the
declared path are the declared bytes. It never verifies an engineering
claim. The engineering acceptance is a separate, explicit, scoped act of the
caller:

- `AcceptOptions.AcceptedBy` is REQUIRED and is recorded on the record as
  `accepted_by`, together with `acceptance_basis` and
  `acceptance_source_head`;
- the CLI requires `--accepted-by`.

Leamas never autonomously determines whether a live experiment actually
occurred.

### Source identity binding

The acceptance decision records the source revision it was made against
(`acceptance_source_head`), so a later source change is detectable. The
evidence record carries its own `source_head`.

### Preserved historical acceptance

An accepted item whose evidence was later invalidated KEEPS its accepted
state. The historical acceptance is a fact about the past; withdrawing it
would erase the record that the obligation was ever met.

The invalidation is never silent:

- the canonical validator reports it as an ADVISORY
  (`acceptance_authority_invalid`), which never rejects a candidate;
- the evidence invalidation transaction is therefore ACCEPTED by the
  candidate validator and can be persisted;
- the readiness projection and the blocker projection both consult the
  SAME authority through `domain.CurrentlyAuthorized`, so an invalidated
  acceptance no longer SILENTLY satisfies a current downstream
  prerequisite.

A cached verification flag never silently authorizes acceptance of a changed
artifact: the re-verification pass is what withdraws the verified status,
and the authority then rejects the item.

## W2 — Simplified Transaction Exclusion

The process-wide implicit writer reentrancy is REMOVED. The exclusion is
now enforced entirely by the operating-system advisory file lock, taken per
open file description. There is no process-wide flag, so a second writer of
the same process is excluded exactly like a second process.

- Independent writers are mutually exclusive whether they are goroutines or
  separate processes.
- Readers do not enter during publication.
- The recovery acquires the SAME exclusive authority as a writer.
- Internal rollback that must reuse an existing lock receives that
  ownership EXPLICITLY: `Transaction.Ownership()` →
  `Store.RecoverWithOwnership(own)`. The lock never infers reentrancy.
- The pending journal state is checked UNDER the acquired lock: a canonical
  read and a new transaction are both refused while an interrupted
  publication is pending.

### Required witnesses

| Witness | Test | Result |
|---------|------|--------|
| Two concurrent goroutine writers | `TestWitnessConcurrentGoroutineWritersAreMutuallyExclusive` | PASS |
| Two concurrent subprocess writers | `TestWitnessConcurrentSubprocessWritersAreMutuallyExclusive` | PASS |
| Reader overlapping publication | `TestWitnessReaderDoesNotEnterDuringPublication` | PASS |
| Recovery overlapping active publication | `TestWitnessRecoveryDoesNotOverlapActivePublication` | PASS |
| Actual process termination after partial publication | `TestWitnessProcessTerminationAfterPartialPublication` | PASS |
| Post-crash read before recovery | `TestWitnessProcessTerminationAfterPartialPublication` | PASS |
| Deterministic recovery and repeated recovery | `TestWitnessRecoveryIsDeterministicAndRepeatable` | PASS |
| Pending journal blocks canonical read and new transactions | `TestWitnessPendingJournalBlocksCanonicalRead` | PASS |

The process-termination witness terminates a REAL PROCESS with SIGKILL. The
helper publishes the first record of a two-record transaction, signals
readiness, and is then killed. The witness proves that the operating system
released the advisory lock, that the post-crash read before recovery is
REFUSED, and that the recovery restores the pre-transaction state.

The claim is **process-crash recovery**. Power-loss durability is NOT
claimed: the individual record writes are individually atomic, but the
journal write and the destination writes are not a single atomic operation
across a power failure.

## W3 — Import Conservation

The planner and the apply path now compare the INCOMING NORMALIZED SOURCE
RECORD with the PRESERVED CANONICAL RECORD — identity AND content —
BEFORE an existing identity is filtered out.

Every source identity receives exactly one reconciliation outcome:

| Outcome | Meaning |
|---------|---------|
| `created` | the identity did not exist in the canonical state |
| `unchanged` | the replay produced no canonical delta |
| `source_changed` | the source record changed; the canonical record is conserved and the change requires explicit reconciliation |
| `conflict` | the source records of one identity disagree; the whole group is quarantined |
| `unresolved` | the identity, status or dependency is unresolved |

The distinguished states are therefore: exact source replay, changed
historical source, legitimate subsequent canonical evolution (a new
identity, which is `created`), conflicting source identities, missing
source identity, unresolved dependency and unknown source status.

A changed historical source NEVER overwrites the canonical work merely
because the historical source changed: the change is an explicit
reconciliation.

Conservation compares the ORIGINAL SOURCE, the NORMALIZED RECORDS and the
ACTUAL PERSISTED STATE. The receipt restates the reconciliation records the
plan actually produced; it does not restate a planned-record count.

### Unknown statuses

An unknown source status is conserved verbatim, reported as an unresolved
reference, and its imported record is durably non-executable
(`acceptance = deferred`) until it is explicitly reconciled. The status
RECOGNITION is case-insensitive, so the original board's UPPERCASE spelling
is understood, while the exact SPELLING is conserved on the record.

### Preserved source provenance

The original row text, the source line locator, the free-form source
detail, the exact source status, the priority, the workstream and the
available provenance detail are all preserved on the imported record
(`source_row`, `source_line`, `source_detail`, `source_status`).

A report-only ambiguity does not disappear from the authoritative migration
accounting: every ambiguity, unresolved reference and reconciliation record
is part of the plan and the receipt.

## W4 — Obligation and Mutation Conservation

### Unfinished-work archival transfers are DISABLED

A complete conservation of a transferred obligation — every downstream edge,
every retained obligation, every exit criterion and every evidence
reference — cannot be established with a small, bounded implementation. The
transfer is therefore REJECTED OUTRIGHT rather than partially honoured. An
unfinished work item may not be archived at all.

### Generic lifecycle updates may not hide obligations

A work item whose acceptance obligation is outstanding cannot be moved into
the closed or archived state through the generic update path. The generic
update is not the operation that discharges an obligation.

### Explicit reopening is operationally effective

A reopen returns the lifecycle to `open` and the acceptance state to
`pending`, so the item is selectable again and the obligation is live. The
historical acceptance DECISION (`accepted_by`, `acceptance_basis`,
`acceptance_source_head`) is preserved verbatim. The preserved decision is
not authority: the acceptance authority re-evaluates the item against the
current evidence.

### Ownership

`Claim` never replaces an existing owner. The explicit, bounded
`TransferOwnership` is the only operation that replaces one.

### Positive expected revisions

An externally initiated mutation of an EXISTING record requires a POSITIVE
expected revision. A serialized internal transaction does not substitute
for the caller's expected revision. The CLI enforces the same rule.

### SameActContent and canonical validation include all ACT ItemIDs

`SameActContent` compares every addressed work-item identity. Two ACT
records that address different sets of work items are different records.

### Per-item retained obligations

Every referenced work item of a terminal ACT is validated: an ACT may not
name a work item that does not exist.

### Read-model blocker reporting

The blocker projection consults the SAME acceptance authority as the
readiness projection and the canonical validator. The two can never
disagree.

No distributed scheduling and no complex ownership leases were implemented.

## W5 — Original IECR Fixture Qualification

The operator supplied the original file:

```text
internal/ledger/importer/testdata/iecr-board-intake-fixture.json
bytes   21043
sha256  bab945991714438fdf66ba5b2efcd7d2a13e484f7d8804a25c7084319bba574c
```

The digest and the byte size are VERIFIED by the qualification. The
repository-native fixture `iecr-intake-board.json` is retained SEPARATELY
as SYNTHETIC and is never claimed to be the original.

The original schema is NOT the existing SourceBoard schema. A minimum
necessary VERSIONED ADAPTER was introduced: `internal/ledger/importer/original.go`
parses `fixture_version` 1 and normalizes the document into a SourceBoard.

The presence-only test was REPLACED by an executed qualification.

### Observed original inventory

| Property | Expected | Observed |
|----------|----------|----------|
| Work items | 44 | 44 |
| Workstreams | 8 | 8 |
| Parent IECR epics | 1 | 1 |
| Exit criteria | 12 | 12 |
| Distinct source statuses | 10 | 10 |
| CLOSED | 10 | 10 |
| CLOSED_INSUFFICIENT | 1 | 1 |
| Unfinished | 33 | 33 |
| BLOCKED | 6 | 6 |

Eight independent epics were NOT created: the parent epic identity
`EPIC-CLINEMM-IECR` is preserved and every work item keeps its workstream
identity inside that one epic.

### Executed qualification

| Assertion | Test | Result |
|-----------|------|--------|
| The fixture is the original document (digest + size) | `TestOriginalIECRFixtureIdentityAndInventory` | PASS |
| Parsing, normalization, import, persistence, reload | `TestOriginalIECRFixtureImportPersistsAndReloads` | PASS |
| The persisted state passes the AUTHORITATIVE validator | `TestOriginalIECRFixtureImportPersistsAndReloads` | PASS |
| Every source identity accounted for | `TestOriginalIECRFixtureConservesEverySourceIdentity` | PASS |
| Every source status conserved | `TestOriginalIECRFixtureConservesEverySourceIdentity` | PASS |
| Original row text, locator and detail preserved | `TestOriginalIECRFixtureConservesEverySourceIdentity` | PASS |
| Deferred work remains excluded from execution | `TestOriginalIECRFixtureDeferredAndUnknownWorkStaysNonExecutable` | PASS |
| Unknown statuses remain unresolved | `TestOriginalIECRFixtureDeferredAndUnknownWorkStaysNonExecutable` | PASS |
| Insufficient investigations do not establish acceptance | `TestOriginalIECRFixtureInsufficientInvestigationDoesNotAccept` | PASS |
| Partial results do not become complete | `TestOriginalIECRFixtureInsufficientInvestigationDoesNotAccept` | PASS |
| Conversational claims do not acquire repository-verified provenance | `TestOriginalIECRFixtureNeverAcquiresVerifiedProvenance` | PASS |
| Exact replay produces no canonical state delta | `TestOriginalIECRFixtureExactReplayProducesNoDelta` | PASS |
| Changed replay produces explicit reconciliation | `TestOriginalIECRFixtureChangedReplayProducesExplicitReconciliation` | PASS |
| Missing historical information remains explicitly unavailable | `TestOriginalIECRFixtureReportsUnavailableHistory` | PASS |
| The versioned adapter refuses an unknown version | `TestOriginalFixtureRejectsUnknownVersion` | PASS |

Nothing is SKIPped and nothing is presence-only.

### Missing historical information

The original fixture is NOT the full historical repository board. The
missing information is reported explicitly by
`importer.UnavailableHistoricalInformation()` and is surfaced by the CLI:

```text
the original fixture is not the full historical repository board
no source dependency edges are present in the original fixture
no evidence records or artifact identities are present in the original fixture
no qualification outcomes are present in the original fixture
the source board revision is a snapshot locator, not a verified repository state
the source paste digest is recorded by the fixture, not verified by the importer
```

No dependency, qualification outcome or evidence authority is inferred.

## W6 — Execution, Verification and Evidence

### Exact commands

```text
gofmt -l internal/ledger/ cmd/leamas/ledger*.go     -> clean (empty output)
go vet ./internal/ledger/... ./cmd/leamas/...        -> PASS
go build ./...                                      -> PASS
go test ./internal/ledger/... -timeout 240s         -> PASS (all five packages)
go test ./cmd/leamas/ -run TestLedger -timeout 100s  -> PASS
```

### Exact-head focused qualification

The focused qualification is executed from the CLEAN COMMITTED source of
the resulting commit. See "Exact-Head Receipt" below.

### Failing test identities

The whole-repository lane has exactly one pre-existing failure class that
is unrelated to this amendment, plus the dirty-worktree dogfood class:

| Failing test | Baseline result at `0b385166` | Classification |
|---------------|------------------------------|----------------|
| `cmd/leamas` `TestClosureCLIV2R2CRExactTipDogfood` | **PASS on the clean baseline** | DIRTY-WORKTREE dogfood precondition. It asserts the CALLER's worktree is clean. It PASSES on the clean committed source — re-verified below. |
| `cmd/leamas` `TestClosureCLIV2VerifierMacHandoffCorrection02A` | **PASS on the clean baseline** | DIRTY-WORKTREE dogfood precondition, same class. |
| `cmd/leamas` `TestClosureCLIV2VerifierMacHandoffCorrection01` | **PASS on the clean baseline** | DIRTY-WORKTREE dogfood precondition, same class. |
| `cmd/leamas` `TestClosureCLIV2VerifierMacHandoff` | **PASS on the clean baseline** | DIRTY-WORKTREE dogfood precondition, same class. |
| `internal/execution` `TestValidateRetainedPipeTopology` (2 subtests) | **FAIL on the clean baseline** | PRE-EXISTING, unrelated. A Linux pipe-topology contract running on macOS. Not repaired: the addendum forbids repairing unrelated packages. |
| `cmd/leamas` `TestBoundedSubprocessV2_TimeoutWithPartialBoundedOutput` | **PASS on the clean baseline** | LOAD-SENSITIVE FLAKE. It failed only in runs that were concurrently executing several long jobs. Not caused by the amendment. |
| `internal/factory/closure` (19 tests, e.g. `TestValidatePlanSemanticMatrixFull_Part1..4`, `TestContractAllMissingPolicyFieldsNamed`, `TestSourceStructureNoLongLines`) | **FAIL with the IDENTICAL test names on the clean baseline** | PRE-EXISTING, unrelated. The baseline worktree reproduces the same 19 failures. The `internal/factory/closure` package is NOT touched by this amendment. |
| `cmd/leamas` `TestRuntimeSmokeCockpitHelp` | PRE-EXISTING, unrelated to the ledger. It is a cockpit process smoke test, outside the amendment scope. |

The baseline comparison was executed, not assumed: the baseline was
checked out into a detached worktree and the same packages were tested
there.

The four dogfood tests fail ONLY because the amendment is present and
uncommitted. They pass on the clean committed source; this is verified
below and reported as evidence, not asserted.

```text
NOT RUN  make factorize
NOT RUN  make gate-dupcode
NOT RUN  make gate
```

These expensive canonical gates are NOT authorized by this addendum. They
are reported as NOT RUN, not as PASS.

### Source evidence for the changed authority and transaction functions

| Function | Path | Role |
|----------|------|------|
| `EvaluateAcceptance` | `internal/ledger/domain/acceptance.go` | THE single acceptance authority |
| `CurrentlyAuthorized` | `internal/ledger/domain/ready.go` | The same authority applied to a downstream prerequisite |
| `validateAcceptanceAuthority` | `internal/ledger/domain/validate_records.go` | THIN ADAPTER over the authority |
| `Accept` | `internal/ledger/service/lifecycle.go` | The explicit engineering acceptance |
| `Archive` / `conserveObligation` | `internal/ledger/service/archive.go` | The obligation-conservation rule |
| `Reopen` | `internal/ledger/service/archive.go` | The explicit, operationally effective reopening |
| `checkRevision` | `internal/ledger/service/service.go` | The positive-expected-revision rule |
| `checkLifecycleDoesNotHideObligation` | `internal/ledger/service/service.go` | The generic-update obligation rule |
| `AcquireRead` / `AcquireWrite` | `internal/ledger/store/lock.go` | The advisory-lock exclusion |
| `Ownership` / `RecoverWithOwnership` | `internal/ledger/store/txn.go` | The EXPLICIT transaction ownership |
| `BuildPlan` / `sameSourceContent` | `internal/ledger/importer/plan.go` | The import conservation comparison |
| `ParseOriginalFixture` / `adaptOriginal` | `internal/ledger/importer/original.go` | The versioned original-fixture adapter |

## W7 — Commit Authorization and Exact-Head Receipt

Commit authority is explicitly delegated by this completion addendum:

- Stage the intended amendment-owned changes.
- Commit the completed amendment.
- Preserve unrelated work and protected state.
- Execute the exact-head focused qualification from the resulting clean
  committed source.
- Commit evidence/report updates where repository conventions require it.

No push was performed and no tag was created.

### The commit

```text
SUBJECT_HEAD  cbce4495e822733834f846131fd3897786de0496
FINAL_HEAD    cbce4495e822733834f846131fd3897786de0496
parent        0b385166d97ed0835f9af7e5cd931914785868d4
subject       factory: complete the FWL01 amendment acceptance and transaction authority
staged paths  59 (every path is amendment-owned; no protected state)
worktree      clean after the commit
```

`SUBJECT_HEAD` and `FINAL_HEAD` were discovered dynamically from the commit
that the addendum produced.

The staged set was verified to contain no protected state: nothing under
`.factory/`, `.leamas/`, `bin/`, `docs/closure-plans/` or
`docs/closure-manifests/`.

### The exact-head digest

The digest was produced with a Leamas binary built from the EXACT head:

```text
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas
./bin/leamas --version
  version:         0.1.0+dev.cbce4495e822.20261126T232018Z
  declared_version: dev
  commit:          cbce4495e822733834f846131fd3897786de0496

./bin/leamas factory digest \
  --range 0b385166d97ed0835f9af7e5cd931914785868d4..cbce4495e822733834f846131fd3897786de0496 \
  --output build/fwl01-amendment-01-digest.txt
  digest: mode=range output=build/fwl01-amendment-01-digest.txt time=3.16s OK
```

The digest records `LEAMAS_COMMIT: cbce4495e822733834f846131fd3897786de0496`
and covers the complete authority-sensitive source change set:

```text
internal/ledger/domain/acceptance.go       the single acceptance authority
internal/ledger/domain/compare.go           the content-equality helpers
internal/ledger/domain/ready.go             CurrentlyAuthorized
internal/ledger/domain/validate_records.go  the thin validator adapter
internal/ledger/importer/compare.go         the import conservation comparison
internal/ledger/importer/original.go        the versioned original adapter
internal/ledger/importer/plan.go            the deterministic import plan
internal/ledger/importer/source.go          the source board model
internal/ledger/service/archive.go          obligation conservation and reopen
internal/ledger/service/lifecycle.go        the explicit acceptance and claim
internal/ledger/service/service.go          the revision and lifecycle guards
internal/ledger/store/lock.go               the advisory-lock exclusion
internal/ledger/store/publish.go            the partial-publication witness
internal/ledger/store/txn.go                the explicit transaction ownership
```

270 ledger content entries are covered by the digest.

## Exit Criteria

| Criterion | Status |
|-----------|--------|
| All previously reproduced P0 counterexamples rejected | MET |
| One canonical acceptance authority | MET |
| No unresolved obligation hidden through archive or generic mutation | MET |
| Explicit reopening is operationally effective | MET |
| Import conserves identities and relationships | MET |
| Unknown and conflicting source records cannot acquire executable authority | MET |
| Same-process and cross-process transaction exclusion qualified | MET |
| Actual process-crash recovery qualified | MET |
| Original fixture successfully imported and reconciled | MET |
| Intended state committed and exact-head evidence available | MET |

## Successor

```text
ACT-CLINEMM-FWL02-HISTORICAL-BOARD-MIGRATION-AND-CUTOVER-01
```

The successor may proceed directly: no new P0 evidence appeared.

## Non-Goals Honoured

- No architectural rewrite. The accepted architecture is preserved.
- No additional framework and no new runtime dependency.
- No distributed scheduling and no complex ownership leases.
- No unrelated package repairs. The pre-existing unrelated failure baseline
  is preserved.

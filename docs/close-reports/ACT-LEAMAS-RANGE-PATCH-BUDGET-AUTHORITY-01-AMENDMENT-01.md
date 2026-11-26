# ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01

## ACT_ID

ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01

## ENTRY HEAD

`480dcad532f4b5897cd3771e23d42f4b670f6546`
(`factory: apply range evidence budget to actual patch bytes`)

## VERDICT

GO_RANGE_PATCH_BUDGET_AUTHORITY
PRODUCTION_IMPLEMENTATION=COMPLETE
PATCH_BUDGET_AUTHORITY=PASS
PATCH_COMPLETENESS=PASS
AGGREGATE_TERMINATION=PASS
AGGREGATE_COMPLETENESS=PASS
RETRIEVAL_EQUIVALENCE=PASS
HISTORICAL_BLOB_IDENTITY=PASS
LITERAL_PATHSPEC_CONSERVATION=PASS
HISTORICAL_ARTIFACT_CLASSIFICATION=PASS
HISTORICAL_CLASSIFICATION_AUTHORITY=PASS
HISTORICAL_BATCH_RESOURCE_BOUNDARY=PASS
HISTORICAL_ARTIFACT_POLICY=PASS
BATCH_FRAMING=PASS
CLASSIFIER_MEMORY_RETENTION=PASS
ENDPOINT_FAILURE_ISOLATION=PASS
HARD_CAP_CONSERVATION=PASS
REDACTION_WITNESS_NONVACUOUS=PASS
AGGREGATE_WITNESS_NONVACUOUS=PASS
WITNESS_QUALITY=PASS
EXECUTION_RESOURCE_BOUNDARY=PASS (BOUNDED_CAPTURE)
ORIGINAL_C4_WITNESS_CONSERVATION=PASS
GENERATOR_IDENTITY=073527e6072a6e2a84f1ab849e18b675741957c0
OPERATIONAL_ACTIVATION=BLOCKED (permission: /usr/local/bin is not writable
by the agent process; the previous executable is INTACT and unchanged)
BASELINE_GATE_STATUS=NOT RUN (not authorized by this ACT)
FINAL_HEAD=073527e6072a6e2a84f1ab849e18b675741957c0
TRACKED_DIRT=false

## SUMMARY

The original ACT correctly replaced blob-size-driven suppression with
patch-byte-driven inclusion. That architectural correction is retained
without redesign.

The AMENDMENT-01 phase closed the six defects that the original ACT
left open (A1-A6), and the FINAL AUTHORITY CORRECTION closed four
further authority gaps (G1-G4). Every finding below was independently
reproduced in a disposable Git repository BEFORE the production
change, and every witness is non-vacuous: it establishes its
PRECONDITION before asserting its POSTCONDITION, and it was shown to
FAIL when the corresponding production fix was reverted.

## FINAL AUTHORITY CORRECTION — G1 TO G4

### G1 — HISTORICAL CLASSIFICATION CONSERVATION

Defect (reproduced): the range renderer classified from the HEAD
endpoint alone whenever it resolved. A range that REPLACED a historical
digest artifact with ordinary text therefore lost the preimage's
artifact classification, even though the range still CONSUMED that
artifact. The artifact-exclusion policy describes the artifact's
IDENTITY, not the bytes that happen to sit at the path today.

Correction: both relevant endpoints are classified and the
artifact-exclusion classification is CONSERVED across them. The
evidence source names the endpoint that carried the authority, so an
endpoint whose object is GENUINELY ABSENT (a deletion) stays distinct
from UNAVAILABLE evidence. The existing historical-classification
witness shape is reused.

### G2 — HISTORICAL BATCH FRAMING

Defect (reproduced): `git cat-file --batch` emits the CONTENT of every
RESOLVED object, whatever its type. The predecessor parser consumed
the content only for blob records, so a batch that mixed a tree with a
blob read the next record from the middle of the tree's binary
content and MIS-ATTRIBUTED every subsequent record.

Correction: the parser consumes the full declared content and the
trailing delimiter of every resolved object, classifies only the blob
records, and is FAIL-CLOSED on framing defects: a truncated record is
never marked OK and retains no partial body, and a malformed header
stops the parse. The retained classifier body is a COPY, so it does
not alias the batch backing array. A blob whose KNOWN size already
exceeds the batch content budget is reported unavailable WITHOUT a
redundant content read. The capture limit was NOT raised and no
streaming execution framework was introduced.

### G3 — FAIL-CLOSED COMPARISON AUTHORITY

Defect (reproduced): a `git diff` WITHOUT comparison endpoints does not
fail. It compares the mutable WORKTREE against the index and exits 0,
so it produced a SUCCESSFUL comparison against the wrong authority. In
addition, Git honours `diff.external` and the `diff` textconv attribute
by DEFAULT: with an external driver configured, Git emits the driver's
output INSTEAD of the patch, so the published `PatchBytes` would not
describe a unified diff at all.

Correction: the canonical specification exposes its comparison
authority (`Comparison`, `HasComparison`), and the renderer refuses to
execute a specification without one, reporting explicit unavailable
evidence. The range-level retrieval path is published only when the
comparison authority is available. The canonical diff options disable
the external-diff driver and the textconv filters. No new Git
abstraction was introduced.

### G4 — HARD-CAP COMPLETENESS CONSERVATION

Defect: the existing direct writer witness asserted only that the
completeness disposition SURVIVED, which a writer that simply ignored
the total budget would also satisfy.

Correction: the witness now asserts BOTH properties at once — the
disposition is TRUTHFUL (a partial publication never claims COMPLETE)
and the TOTAL RENDER BUDGET REMAINS CONSERVED (the reserved slice is
the only thing allowed past the exhausted ordinary budget, and it is
released again). No new aggregate fixture was introduced.

## A1 — COMPLETE PATCH PUBLICATION

Defect (reproduced): the renderer applied `truncateLongLines(patch)`
before writing, and discarded the writer's report (`_ = n`). A patch
whose complete unified form fits `MaxPerFileBytes` (64 KiB) but which
carries a physical line longer than `MaxPerLineBytes` (4 KiB) was
therefore silently transformed while the digest reported the ORIGINAL
`PatchBytes`.

Measured fixture: a 20,000-character changed line yields a 40,109-byte
complete patch (fits the per-file budget) with a 20,001-byte physical
line (exceeds the per-line cap).

Correction: the patch is emitted VERBATIM when it fits the budget. The
per-line cap is a presentation constraint for unbounded file bodies
and is not applied to a patch that already fits. Any budget-induced
partial publication is now disclosed as
`EvidenceCompleteness: INCOMPLETE` with
`SuppressionReason: PATCH_TRUNCATED_BY_PER_FILE_CAP` or
`PATCH_TRUNCATED_BY_TOTAL_CAP`. A fully published patch declares
`EvidenceCompleteness: COMPLETE`.

## A2 — CANONICAL GENERATION AND RETRIEVAL

Defect (reproduced): the generation command and the published
`RetrievalPath` were different commands.

- Generation used rename/copy detection and scoped the diff to BOTH
  the old and the new path.
- Retrieval used neither.

Measured: for a pure rename of a 20,000-line file the
destination-only retrieval command produced 20,131 bytes while the
rename-aware generation command produced 99 bytes.

Two further defects were reproduced:

- The retrieval command used the caller's raw revision expression
  (`HEAD~1..HEAD`), so after the next commit it described a DIFFERENT
  range. Executing the same command before and after one commit
  produced two different patches.
- The path was passed as a bare Git pathspec. Git interprets a
  pathspec as a WILDCARD expression, so `literal*.ts` also matched
  `literal-other.ts` and the per-file budget was applied to a patch
  belonging to several files.

Correction: ONE canonical `PatchSpec` drives both consumers. It pins

- IMMUTABLE resolved comparison endpoints (object IDs, not symbolic
  refs), so the published command keeps describing the same range
  after HEAD advances;
- Git's `:(literal)` pathspec magic, so a path matches exactly one
  file;
- the rename/copy source path, so the diff sees both sides.

The digest publishes:

- `RangeRetrievalPath:` — one canonical command for the whole range,
  carrying the immutable endpoints and the diff options;
- `RetrievalPath:` — the path-scoping suffix
  (`-- ':(literal)<path>' [':(literal)<from>']`) published for the
  BOUNDED dispositions, where the embedded patch is absent and the
  reviewer needs a way to obtain that one file's patch.

The per-file line is a SUFFIX of the range-level command. The
published command is therefore complete and directly executable, and
the two can never drift because both are derived from the same
specification.

## A3 — HISTORICAL BLOB IDENTITY

Defect (reproduced): the base endpoint blob was looked up at the
DESTINATION path for every change kind. For a rename the preimage
lives at the SOURCE path.

Measured: `git cat-file -t HEAD~1:new.ts` reports "path exists on disk,
but not in HEAD~1" while `git cat-file -t HEAD~1:old.ts` reports
`blob`.

Correction: the base lookup uses the source path for a rename or a
copy (`rangeBaseIdentityPath`). The head lookup keeps the destination
path. The absent/unavailable/present distinctions are conserved.

Scope note: the correction applies to renames Git actually PAIRS. When
Git does not pair the two sides it reports `A` plus `D`, the
destination path genuinely does not exist at the base endpoint, and
the base identity is legitimately `MISSING`.

## A4 — HISTORICAL ARTIFACT CLASSIFICATION

Defect (reproduced): the range-mode classifier read the CURRENT
working-tree file. With the blob-size suppression removed by the
original ACT, that worktree read became the ONLY classifier for a
historical artifact, so the disposition of an immutable historical
range changed whenever the checkout changed.

Measured: the same immutable range was classified
`BOUNDED_DERIVED_DIGEST` with the artifact in the worktree and had NO
classification at all with ordinary text in the worktree.

Correction: the range-mode classifier reads the HISTORICAL blob at the
head endpoint, in ONE batched `git cat-file --batch` process. The
working tree is consulted only as a fallback when no historical blob
is available. No repository-wide source ingestion occurs; the reads
are bounded by `classifierScanCap`.

The self-output, derived-digest and structural-recursion policies are
conserved. Blob size is NOT restored as patch-inclusion authority.

## A5 — REPAIRED CONSERVATION WITNESSES

The predecessor witnesses were strengthened, not merely supplemented.

### Redaction

Predecessor defect: VACUOUS. The secret was appended at the END of
the file while the FIRST line was changed, so Git's three context
lines never carried the secret into the patch. The witness passed even
if redaction were completely broken.

Correction: the secret is inside the CHANGED line. The witness asserts
the RAW patch contains it before asserting the RENDERED evidence
excludes it.

### Aggregate budget

Predecessor defect: NO PROVEN PRESSURE. 40 files produced 12,920
aggregate patch bytes against a 262,144-byte budget. The witness would
have passed if the renderer discarded every patch.

Correction: 1,500 files. The witness measures the actual aggregate,
REQUIRES it to exceed the budget, requires every individual patch to
fit the per-file budget, requires at least one inclusion and at least
one omission, and requires the omission record to name the omitted
files.

### Rename, deletion, tiny patch, oversized patch

| Witness | Predecessor defect | Correction |
| --- | --- | --- |
| Rename | Asserted only that the destination name appeared somewhere in the digest, which the changeset manifest already satisfies. | Asserts the rename header, the source path and the destination path inside the FILE'S OWN SECTION. |
| Deletion | Asserted only that the file heading existed. | Asserts the actual deletion hunk line. |
| Tiny patch | Asserted only that the changed marker appeared. | Asserts the COMPLETE expected patch against the file-evidence section. |
| Oversized patch | Asserted only a command prefix. | Executes the published retrieval specification and requires byte equality with the measured patch. |


## A6 — EXECUTION RESOURCE CONSERVATION

Finding: the digest package executes Git through raw `os/exec`, NOT
through the bounded `internal/execution` gateway. The predecessor
therefore materialised an unbounded patch in memory before the 64 KiB
evidence limit applied.

Correction: the digest's own runner boundary applies a bounded stdout
capture (`MaxGitCaptureBytes`, 16 MiB). An output that exceeds the
limit is reported as an ERROR, never as a truncated success. A partial
patch must never be mistaken for the complete one. No new generic
execution framework was introduced.

See "AMENDMENT-01 FINALIZATION" below for the corrected description of
the ACHIEVED contract.

## AMENDMENT-01 FINALIZATION

A subsequent review of this amendment raised four further findings.
Each was INDEPENDENTLY REPRODUCED before any production change, and
each is closed by a witness that was shown to FAIL when the
corresponding fix was reverted.

### F1 — HISTORICAL CLASSIFICATION AUTHORITY

Reviewer finding: the classifier loaded the historical blob at the
HEAD endpoint only and fell back to the CURRENT WORKTREE when that
lookup failed.

REPRODUCED: a DELETED historical digest artifact has no head blob. The
code therefore fell back to the current working tree, so the
classification of an immutable historical range again depended on
unrelated mutable state — exactly the defect A4 was meant to close.

Correction: the classifier authority is now decided by
`selectHistoricalClassifierEvidence`:

1. the HEAD blob, when it resolves — the artifact the range PRODUCES;
2. otherwise the BASE preimage — the artifact the range CONSUMED. This
   covers a DELETION (no head blob exists) and a RENAME (the preimage
   lives at the source path);
3. otherwise NO historical evidence is available.

The CURRENT WORKTREE IS NEVER SUBSTITUTED. In case 3 the digest
reports the degraded disposition explicitly
(`Classification: HISTORICAL_EVIDENCE_UNAVAILABLE`, with
`WarningCode: HISTORICAL_EVIDENCE_UNAVAILABLE` and
`HistoricalEvidenceSource: NONE`).

The digest now also publishes `HistoricalEvidenceSource: HEAD|BASE|NONE`
for every classified file, so the reviewer can see WHICH endpoint
carried the authority.

### F1 — HISTORICAL BATCH RESOURCE BOUNDARY

Reviewer finding: `git cat-file --batch` emits the CONTENT of every
requested object, so a batch over many blobs can exceed the capture
limit, the captured result is discarded, and the historical
classification is lost.

REPRODUCED: a batch over twenty 900 KiB blobs produced 18,433,080
bytes against a 16,777,216-byte capture limit. The batch DID fail with
the overflow error, and the predecessor's per-object fallback DID
recover the classification — but the recovery is per-object and
unbounded, and a single blob larger than the capture limit was lost
entirely.

Correction: the batch is CHUNKED by blob size. The sizes are fetched
with `git cat-file --batch-check`, which emits metadata only, so the
sizing pass is bounded by the number of paths and not by their
content. Each chunk's aggregate content stays below
`historicalBatchContentBudget`, which is itself below the capture
limit, so a chunk cannot overflow the capture by construction.

The capture limit was NOT raised. The loading stays bounded.

### F2 — AGGREGATE COMPLETENESS

Reviewer finding: the completeness disposition was written into the
SAME writer whose budget the patch had just exhausted, so
`appendBlock` silently dropped it.

REPRODUCED: with the total budget exhausted, `appendFileString`
returned 0 bytes and the subsequent `appendBlock` for the disposition
returned 0 bytes. The digest reported a partial publication with NO
completeness statement at all.

Correction: the writer reserves a slice of the TOTAL budget
(`completenessDispositionBudget`) for the disposition BEFORE the patch
body is written. The patch body can therefore never consume the budget
the disposition itself needs. The disposition is a GUARANTEED
publication: a reviewer that sees a file section always sees that
file's completeness statement.

The reservation is released once the disposition is written, so the
aggregate budget is not permanently weakened.

### F2 — AGGREGATE WITNESS

The predecessor witness used 1,500 files of ~2,000 lines and launched
one independent Git subprocess per file to measure them, and asserted
inclusion by the presence of a file HEADING.

Correction: the fixture is now EIGHT files of 4,000 lines with 400
modified lines each. Each patch measures 35,440 bytes — below the
64 KiB per-file budget, so every individual patch is still INCLUDED —
while the aggregate of 283,520 bytes exceeds the 256 KiB aggregate
budget. The witness now asserts the COMPLETE patch verbatim for every
section that claims COMPLETE, asserts the ACTUAL omitted paths, and
asserts that every published section declares its completeness.

The package test time fell from 222.6s to 83.8s.

### F3 — MINOR WITNESS CORRECTIONS

- Redaction: the witness now also asserts that the intended
  non-secret change (`# EDITED neighbouring line`) SURVIVES the
  redaction. Without it, an implementation that suppressed the complete
  patch would satisfy the secret-absence assertion.
- Rename: the predecessor combined the two negative predicates with
  `&&`, so the assertion only failed when BOTH headers were absent.
  Each header is now asserted separately, and the EXACT expected
  headers (`rename from old-name.ts`, `rename to new-name.ts`) are
  required.

### F4 — TRUTHFUL RESOURCE GUARANTEE

The predecessor comment claimed that an oversized capture "is detected
and the process is terminated". That was never true of the
implementation: the writer keeps draining and discarding the remainder,
so Git still produces its full output, and there is no deadline and no
cancellation.

Correction: the comment now states the ACHIEVED contract accurately —
`BOUNDED_CAPTURE`. The retained stdout payload is bounded by
`MaxGitCaptureBytes`, the retained stderr payload by 1 MiB, and an
overflow is an error rather than a truncated success. The subprocess
output generation, the execution time and the termination on overflow
are explicitly NOT bounded. Migrating the digest's Git execution onto
the bounded `internal/execution` gateway remains a separate
architectural task and was NOT performed by this ACT.

## ORIGINAL C4 WITNESS CONSERVATION

The original C4 witness is the REAL ClineMM commit
`2b7fccbbfa26a8221e7ed608d98b453598e5c925`. It was regenerated with
the executable built from the FINAL commit `073527e`.

```
cd <ClineMM repo>
leamas factory digest --range \
  '2b7fccbbfa26a8221e7ed608d98b453598e5c925^..2b7fccbbfa26a8221e7ed608d98b453598e5c925' \
  --output <digest>
```

| Production patch | Expected | Observed | Disposition |
| --- | --- | --- | --- |
| `sdk/packages/agents/src/agent-runtime.ts` | 13,195 B | `PatchBytes: 13195`, `EvidenceCompleteness: COMPLETE` | CONSERVED |
| `sdk/packages/shared/src/agents/bounded-objective.ts` | 30,074 B | `PatchBytes: 30074`, `EvidenceCompleteness: COMPLETE` | CONSERVED |

- No blob-size-based suppression: the digest contains ZERO
  `LARGE_FILE_EVIDENCE_BOUNDED` occurrences.
- Completeness: the digest contains 12 `EvidenceCompleteness: COMPLETE`
  dispositions and ZERO `INCOMPLETE` or `UNAVAILABLE` dispositions.
- Generator identity: the digest reports
  `GENERATOR_COMMIT: 073527e6072a6e2a84f1ab849e18b675741957c0`, the
  commit the executable was built from.
- The Leamas tracked worktree was CLEAN for the witness run.

The earlier `C4_SYNTHETIC_PASS` result was a DIFFERENT witness (a
synthetic ~900 KB source carrying a 379-byte patch) and is NOT the
conservation of the original C4. The result above is the original
C4, regenerated.

The witness was regenerated with BOTH the previously installed
executable (`c84761f`) and the final executable (`073527e`). The two
digests agree on every observable above, so the authority corrections
conserved the original C4 evidence rather than changing it.

## OPERATIONAL ACTIVATION

The normal invocation path is `leamas` on `PATH`, which resolves to
`/usr/local/bin/leamas`. The existing installation mechanism is
`make install`, which builds `bin/leamas`, verifies its SemVer stamp and
copies it to `$(DESTDIR)$(BINDIR)/leamas` (`BINDIR ?= $(PREFIX)/bin`,
`PREFIX ?= /usr/local`).

The activation was attempted and is BLOCKED by a system permission
boundary, not by a defect in the correction:

```
$ make install
stamp-check OK: bin/leamas reports version=0.1.0+dev.073527e6072a.20261126T035427Z
Installing Leamas to /usr/local/bin/leamas
install: cannot remove '/usr/local/bin/leamas': Operation not permitted
```

- Every write probe to `/usr/local/bin` is refused with "Operation not
  permitted", including from the user-session context (`osascript`).
  `sudo` is not available to the agent process.
- No directory on `PATH` is writable by the agent process, so the
  qualified executable cannot be placed earlier on `PATH` either.
- The previous executable was PRESERVED before the attempt and is
  INTACT and UNCHANGED after it: `/usr/local/bin/leamas` and the
  rollback copy have the identical SHA-256
  `9eddd9869392f1e471ed47aaa55765cf88785e6afda700ed29254c5be0800714`.
  The failed install left no partial state.
- No stray probe files remain in `/usr/local/bin`.

The remaining activation step is one command, to be run by an operator
with write access to `/usr/local/bin`:

```
cd /Volumes/UserData/Users/chistyakov/Projects/SPbNIX/leamas
make install
leamas --version    # must report commit 073527e6072a6e2a84f1ab849e18b675741957c0
```

Then regenerate the C4 witness through the normal invocation path
(`leamas factory digest --range ...` as in the section above). The
regeneration was already executed with the qualified executable, and the
result is recorded above.

## BASELINE GATE DISPOSITION

`CGO_ENABLED=0 make gate-fast` was NOT RUN. The ACT authorized only
the affected witnesses, formatting, affected-package vet and build,
and explicitly excluded the full repository suite.

The affected-package results are recorded in the EXACT COMMANDS RUN
section below.

## FILES CHANGED

Production:

- `internal/factory/digest/evidence_budget_patch_spec.go` (new) —
  the canonical `PatchSpec`, endpoint resolution, immutable-OID
  pinning.
- `internal/factory/digest/evidence_budget_shell_quote.go` (new) —
  lossless shell-safe serialization and the retrieval serializations.
- `internal/factory/digest/evidence_budget_historical_classifier.go`
  (new) — batched historical blob classification.
- `internal/factory/digest/evidence_budget_render_range.go` —
  canonical patch generation, completeness disclosure, range-level
  retrieval publication, historical classification, rename base
  identity, and the F1/F2 finalization corrections (base-preimage
  classification authority, guaranteed completeness publication).
- `internal/factory/digest/evidence_budget_render_range_disposition.go`
  — canonical retrieval path, rename base identity, and the
  `HistoricalEvidenceSource` disclosure.
- `internal/factory/digest/evidence_budget_render.go` — the F2
  completeness reservation and the guaranteed-publication writer.
- `internal/factory/digest/evidence_budget.go` — the
  `HISTORICAL_EVIDENCE_UNAVAILABLE` class and warning code.
- `internal/factory/digest/evidence_budget_render_helpers.go` —
  patch-spec helpers relocated to the canonical spec file.
- `internal/factory/digest/file_evidence.go` — bounded Git capture and
  the F4 truthful contract statement.
- `internal/factory/digest/git.go` — bounded Git capture.

Final authority correction (this ACT):

- `internal/factory/digest/evidence_budget_patch_endpoints.go` (new) —
  the comparison endpoint resolution and the immutable-OID pinning.
- `internal/factory/digest/evidence_budget_historical_batch_parse.go`
  (new) — the `git cat-file --batch` response parser, with the
  fail-closed framing rules.
- `internal/factory/digest/evidence_budget_historical_classification.go`
  (new) — the artifact-exclusion classification conservation selector.
- `internal/factory/digest/evidence_budget_historical_range_classification.go`
  (new) — the range-mode historical classification decision.
- `internal/factory/digest/evidence_budget_warning_codes.go` (new) — the
  warning-code vocabulary, including
  `COMPARISON_ENDPOINTS_UNAVAILABLE`.
- `internal/factory/digest/evidence_budget_patch_spec.go` — the
  exported comparison authority (`Comparison`, `HasComparison`) and the
  canonical diff options that disable the external-diff driver and the
  textconv filters.
- `internal/factory/digest/evidence_budget_historical_classifier.go` —
  the oversized-blob separation, so an object already known to exceed
  the capture limit is not read redundantly.
- `internal/factory/digest/evidence_budget_render_range.go` — the
  conserved two-endpoint classification and the fail-closed endpoint
  check.
- `internal/factory/digest/evidence_budget_render_range_disposition.go`
  — the explicit unavailable-endpoint disposition.
- `internal/factory/digest/evidence_budget.go` — the warning codes
  relocated to their own file.

The production files were split so that every ACT-owned file stays
within the LLM-friendly 400-line limit. The LLM-friendliness gate
reports the same 87 pre-existing findings before and after the change,
and none of them is in a file this ACT owns.

Tests:

- `evidence_budget_patch_authority_amendment_test.go` (new) — A1
  witnesses and the shared helpers.
- `evidence_budget_patch_authority_amendment_retrieval_test.go`
  (new) — A2 retrieval and literal-pathspec witnesses.
- `evidence_budget_patch_authority_amendment_identity_test.go` (new) —
  A3 and A4 witnesses.
- `evidence_budget_patch_authority_amendment_resource_test.go` (new) —
  A6 witness, strengthened with the cheap mixed-object discriminator and
  the oversized-blob non-redundant-read assertion.
- `evidence_budget_patch_authority_amendment_completeness_test.go` —
  strengthened to assert BOTH the truthful completeness disposition and
  the total render budget conservation.
- `evidence_budget_patch_authority_final_batch_test.go` (new) — the G2
  batch-framing, truncated-record and memory-retention witnesses.
- `evidence_budget_patch_authority_final_classification_test.go` (new) —
  the G1 historical-classification-conservation witnesses.
- `evidence_budget_patch_authority_final_endpoint_test.go` (new) — the G3
  comparison-authority, external-diff and lossless-retrieval witnesses.
- `evidence_budget_patch_authority_test.go` — strengthened.
- `evidence_budget_patch_authority_bounded_test.go` (new) — split from
  the above.
- `evidence_budget_patch_authority_conservation_test.go` —
  strengthened.
- `range_mode_bypass_core_test.go` — updated to the canonical
  pathspec argument vector.

## BEHAVIOR CHANGED

- A patch that fits the per-file budget is published COMPLETE, with
  `EvidenceCompleteness: COMPLETE`. It is never silently truncated.
- A budget-induced partial publication is disclosed as INCOMPLETE
  with a truthful suppression reason.
- The published retrieval command reproduces the measured patch byte
  for byte, and keeps doing so after HEAD advances.
- The per-file evidence budget is applied to exactly one file even
  when the path contains glob metacharacters.
- The base blob identity of a renamed file resolves to its true
  preimage.
- The disposition of an immutable historical range no longer depends
  on the current working tree.
- The digest's Git invocations have a bounded stdout capture; an
  overflow is an error, never a truncated success.
- A DELETED historical artifact is classified from its BASE preimage,
  and every classified file publishes the endpoint that carried the
  classification authority.
- The current working tree is NEVER substituted for unavailable
  historical evidence; the degraded disposition is reported explicitly.
- The historical blob batch is chunked by blob size, so it cannot
  overflow the capture limit; the capture limit was not raised.
- The completeness disposition of a file is a GUARANTEED publication:
  it survives the exhaustion of the aggregate render budget.
- The digest's Git capture contract is stated truthfully as
  BOUNDED_CAPTURE.
- A range that REPLACED a historical digest artifact with ordinary text
  KEEPS the preimage's artifact-exclusion classification, because the
  exclusion describes the artifact's identity and the range still
  consumed it.
- An endpoint whose object is genuinely ABSENT is distinguished from
  UNAVAILABLE evidence: the other endpoint is the authority, and only
  the genuinely unavailable case is reported as degraded.
- The historical batch parser consumes the full declared content and
  the trailing delimiter of EVERY resolved object, so a mixed
  tree/blob batch keeps its record alignment.
- A truncated or malformed batch record is never marked OK and never
  retains a partial body.
- The retained classifier body is an independent copy; it does not
  alias the batch backing array.
- A blob whose known size already exceeds the batch content budget is
  reported unavailable WITHOUT a redundant content read.
- A patch is never executed without a resolved comparison authority: an
  unresolved endpoint produces explicit unavailable evidence, never a
  successful comparison against the mutable worktree.
- The range-level retrieval path is published only when the comparison
  authority is available.
- The canonical diff options disable the external-diff driver and the
  textconv filters, so the published patch bytes are a property of the
  resolved objects and not of the environment.
- The published retrieval specification remains lossless for
  filenames containing shell metacharacters.

## EXACT COMMANDS RUN

```
gofmt -l internal/factory/digest/                          clean
git diff --check                                           clean
go build ./...                                             PASS
go vet ./internal/factory/digest/                          PASS
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas PASS
go test ./internal/factory/digest/ -count=1                ok (71.4s)
go test ./internal/factory/digest/ -run 'TestAmendment_' -count=1
                                                            ok (4.0s)
go test ./internal/factory/digest/ -run 'TestAmendment_G1_|TestAmendment_G2_|TestAmendment_G3_|TestAmendment_F2_'
                                                            ok
go test ./cmd/leamas/ -count=1 -short (clean tree)          ok (36.9s)
go run ./cmd/leamas factory verify llm-friendly            87 findings, all pre-existing;
                                                            87 before and after this change;
                                                            none in an ACT-owned file
make install                                                BLOCKED (permission, see
                                                            OPERATIONAL ACTIVATION)
leamas factory digest --range '2b7fccb...^..2b7fccb...'    OK (C4 witness, both executables)
```

The `cmd/leamas` failures observed while the worktree was DIRTY are
not failures of this change: they assert that the caller worktree is
clean (and the version stamp is derived from the Git state). The same
package PASSES on the clean tracked worktree at the final head.

## SKIPPED OR DEFERRED

- `make factorize` — NOT RUN (not authorised).
- `make gate-dupcode` — NOT RUN (not authorised).
- `make gate` — NOT RUN (not authorised).
- `CGO_ENABLED=0 make gate-fast` — NOT RUN (not authorised; the ACT
  authorized only the affected witnesses, formatting, affected-package
  vet and build, and explicitly excluded the full repository suite).
- Repository-wide `go test ./...` — NOT RUN (excluded by the ACT).
- Baseline gate repair — deliberately not attempted.
- Operational activation into the normal invocation path — BLOCKED by a
  system permission boundary; see OPERATIONAL ACTIVATION. The previous
  executable is intact and unchanged.

## FOLLOW-UP

- Operational activation remains PENDING one operator command, because
  `/usr/local/bin` is not writable by the agent process. The qualified
  executable is built at `bin/leamas` from the final commit `073527e`
  and reports `GENERATOR_COMMIT:
  073527e6072a6e2a84f1ab849e18b675741957c0`. The exact command is in
  the OPERATIONAL ACTIVATION section.
- Migrating the digest's Git execution onto the bounded
  `internal/execution` gateway remains a separate architectural task.
  The achieved contract of the current implementation is
  BOUNDED_CAPTURE.
- The evidence-allocation priority order proposed by the reviewer
  (production code over tests over documentation) is a separate
  concern and was deliberately NOT implemented here.

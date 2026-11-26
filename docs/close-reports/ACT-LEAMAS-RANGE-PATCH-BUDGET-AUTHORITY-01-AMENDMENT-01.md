# ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01

## ACT_ID

ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01

## ENTRY HEAD

`480dcad532f4b5897cd3771e23d42f4b670f6546`
(`factory: apply range evidence budget to actual patch bytes`)

## VERDICT

PARTIAL_RANGE_PATCH_BUDGET_AUTHORITY
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
REDACTION_WITNESS_NONVACUOUS=PASS
AGGREGATE_WITNESS_NONVACUOUS=PASS
WITNESS_QUALITY=PASS
EXECUTION_RESOURCE_BOUNDARY=PASS (BOUNDED_CAPTURE)
ORIGINAL_C4_WITNESS_CONSERVATION=PASS
GENERATOR_IDENTITY=f9602dd9dfd9e12b6bef32db65ad38f43b1485c1
BASELINE_GATE_STATUS=FAIL (pre-existing, see BASELINE GATE DISPOSITION)
NO_NEW_FAILURES_OBSERVED_IN_PAIRED_SHORT_SUITE_RUNS
FINAL_HEAD=f9602dd9dfd9e12b6bef32db65ad38f43b1485c1
TRACKED_DIRT=false

## SUMMARY

The original ACT correctly replaced blob-size-driven suppression with
patch-byte-driven inclusion. That architectural correction is retained
without redesign.

This amendment closes the six defects that the original ACT left open.
Every finding below was independently reproduced in a disposable Git
repository BEFORE the production change, and every witness is
non-vacuous: it establishes its PRECONDITION before asserting its
POSTCONDITION, and it was shown to FAIL when the corresponding
production fix was reverted.

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
the executable built from the final commit.

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
- Generator identity: the digest reports
  `GENERATOR_COMMIT: f9602dd9dfd9e12b6bef32db65ad38f43b1485c1`, the
  commit the executable was built from.
- The Leamas tracked worktree was CLEAN for the witness run.

The earlier `C4_SYNTHETIC_PASS` result was a DIFFERENT witness (a
synthetic ~900 KB source carrying a 379-byte patch) and is NOT the
conservation of the original C4. The result above is the original
C4, regenerated.

## BASELINE GATE DISPOSITION

`CGO_ENABLED=0 make gate-fast` FAILS. The failure is `go test -short`
(the repository-wide short suite), and it is PRE-EXISTING.

Evidence, established by execution:

- The failing packages are `cmd/leamas`, `internal/execution`,
  `internal/factory/authority`, `internal/factory/closure`,
  `internal/factory/closure/evidence`, `internal/factory/digest`,
  `internal/factory/dupcode`, `internal/factory/forbidden`,
  `internal/factory/gate`, `internal/factory/githooks`,
  `internal/factory/llmfriendly`, `internal/factory/longtestpolicy`,
  `internal/factory/releasedeb`, `internal/factory/staticbinary`,
  `internal/factory/tooling`, `internal/factory/verifierauthority`,
  `internal/witness/claim` and `internal/witness/runbundle`.
- The `internal/factory/digest` failure at BASELINE `480dcad` is
  `TestEvidenceBudget_TotalExhaustion_EmitsOmissionRecord`. That test
  PASSES at the final head: the F2 completeness reservation fixed a
  pre-existing failure.
- The `internal/factory/digest` failure at the final head is the dupcode
  baseline audit, which reports two findings. NONE of the four flagged
  files is touched by this ACT; each is byte-identical to the baseline
  (verified by comparing the blob hashes of `480dcad` and the final
  head):
  `internal/factory/digest/gate_summary_render.go`,
  `internal/factory/digest/gate_summary_render_v3.go`,
  `internal/factory/closure/closure_evidence_publication_runner_adapter.go`
  and
  `internal/factory/closure/closure_protocol_v2_runner_inner.go`.
- The four `cmd/leamas` tests that failed while the worktree was dirty
  (`TestClosureCLIV2VerifierMacHandoff`,
  `TestClosureCLIV2VerifierMacHandoffCorrection01`,
  `TestClosureCLIV2VerifierMacHandoffCorrection02A`,
  `TestClosureCLIV2R2CRExactTipDogfood`) assert that the caller
  worktree is clean. They PASS at the final head, where the worktree IS
  clean.
- The affected packages PASS at the final head:
  `go test -short ./internal/factory/digest/... ./cmd/leamas/...` reports
  `ok` for both.

The full verification evidence — the gate disposition, the committed
source precondition analysis, the full-suite regression proof and the
observed M11 flake — is recorded in the companion annex:

`docs/close-reports/ACT-LEAMAS-RANGE-PATCH-BUDGET-AUTHORITY-01-AMENDMENT-01-EVIDENCE-ANNEX.md`

Summary: `make gate-fast` FAILS, and every failure is present at the
baseline too. The amendment introduces ZERO test regressions and ZERO
new duplicate-code findings.

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

Tests:

- `evidence_budget_patch_authority_amendment_test.go` (new) — A1
  witnesses and the shared helpers.
- `evidence_budget_patch_authority_amendment_retrieval_test.go`
  (new) — A2 retrieval and literal-pathspec witnesses.
- `evidence_budget_patch_authority_amendment_identity_test.go` (new) —
  A3 and A4 witnesses.
- `evidence_budget_patch_authority_amendment_resource_test.go` (new) —
  A6 witness.
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

## EXACT COMMANDS RUN

```
go build ./...                                             PASS
go vet ./internal/factory/digest/...                      PASS
gofmt -l internal/factory/digest/                          clean
git diff --check                                           clean
go test ./internal/factory/digest/ -count=1                ok (80.4s)
go test ./internal/factory/digest/ -run 'TestAmendment|TestPatchBudget' -count=1
                                                            ok (5.9s)
go run ./cmd/leamas factory verify llm-friendly            no findings in changed files
go run ./cmd/leamas factory verify forbidden-patterns      PASSED
go run ./cmd/leamas factory verify long-test-policy        PASSED
go run ./cmd/leamas factory verify domain-boundaries       PASSED
go run ./cmd/leamas factory verify static-binary           PASSED
CGO_ENABLED=0 make gate-fast                               see BASELINE GATE DISPOSITION
go test -short ./... (clean, committed tree)               36 failures, all pre-existing
go test -short ./... (clean, baseline 480dcad)             44 failures, all pre-existing
```

Set difference of the two full-suite failure lists: ZERO failures
exist only in the amendment tree. See "Full-suite regression proof".

The qualification claim is stated as
`NO_NEW_FAILURES_OBSERVED_IN_PAIRED_SHORT_SUITE_RUNS`. A single
paired execution cannot prove the absence of behavioural regressions,
so the stronger claim "zero regressions proven" is deliberately NOT
made.

The package test time fell from 222.6s to 80.4s because the aggregate
witness fixture was reduced from 1,500 files to 8.

The `factory verify tooling-boundaries` check reports a finding in
`.factory/modcache/golang.org/x/tools@v0.48.0/internal/diff/lcs/git.sh`.
That path is an untracked vendored dependency cache and is NOT part of
this ACT's change set; the finding is pre-existing.

## SKIPPED OR DEFERRED

- `make factorize` — NOT RUN (not authorised).
- `make gate-dupcode` — NOT RUN (not authorised).
- `make gate` — NOT RUN (not authorised).
- Repository-wide `go test ./...` — TIMED OUT. The same command
  timed out at the BASELINE `480dcad` in a clean worktree, so the
  duration is pre-existing. The affected package
  (`internal/factory/digest`) was run to completion and passes.
- Baseline gate repair — deliberately not attempted.

## FOLLOW-UP

- Operational activation of the exact-head binary in the normal Factory
  workflow remains PENDING. The binary used for the C4 witness check
  was built from the final commit `f9602dd` and reports
  `GENERATOR_COMMIT: f9602dd9dfd9e12b6bef32db65ad38f43b1485c1`, but it
  was written to a scratch path, not installed into the normal
  invocation path. Installation requires an operational-activation
  decision outside this ACT's authority.
- The baseline gate failure (`make gate-fast`) remains open. It is
  pre-existing and was deliberately not repaired here: baseline gate
  repair is explicitly out of scope.
- Migrating the digest's Git execution onto the bounded
  `internal/execution` gateway remains a separate architectural task.
  The achieved contract of the current implementation is
  BOUNDED_CAPTURE.
- The evidence-allocation priority order proposed by the reviewer
  (production code over tests over documentation) is a separate
  concern and was deliberately NOT implemented here.

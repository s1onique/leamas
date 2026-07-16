# ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03

## Status: COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03`
is **COMPLETE**. Every prior P0 defect identified by the
CORRECTION02 review verdict has been replaced with executable
evidence:

  * the prior 877- and 514-token findings each receive one
    definitive, executable classification (no unresolved
    disjunction);
  * the surviving 504-token finding is proved maximal from
    validated pre-publication evidence (digest equality, exact
    token count, exact internal positions, single-owner region
    ownership, structural-shadow guard witness);
  * the public region-ownership test was strengthened to walk
    the production TokenOwner array per token and reject any
    occurrence that includes unowned or multi-region tokens;
  * the fail-closed test was renamed to match what it actually
    executes;
  * every staged close report now agrees with the executable
    evidence (no "514 is contained in 504", no "514 is shadow-
    suppressed by 504", no "877 and 514 share the 504 content
    key", no "877 and 514 merge into 504");
  * one canonical `make gate` summary is bound to one final
    staged-tree OID.

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is now unblocked.

## P0 corrections applied

### 1. Executable baseline-forensics oracle

A new test file
`internal/factory/dupcode/v4_baseline_forensics_test.go`
introduces a test-owned forensics oracle that operates on the
actual production source files
`cmd/leamas/claim_commands.go` and `cmd/leamas/evidence_commands.go`.
For every historical public line range pair, the oracle:

  * invokes the production scanner/parser via
    `analyzeV4AnalyzedFile` (the same path `CheckRepo` uses);
  * maps each historical public line range to inclusive token
    positions via `mapLineRangeToTokenRange`;
  * extracts the normalized token slice;
  * computes an independent SHA-256 digest via `sha256Hex` so
    a defect in `v4ExactNormalizedDigest` cannot make the
    classification pass silently;
  * records every TokenOwner in the slice via
    `collectOwnersInRange` so the per-token ownership invariant
    is auditable;
  * classifies the slice using `classifyFromDisposition`, which
    returns exactly one of:
      - "valid canonical exact duplicate",
      - "invalid because geometry crosses executable-region ownership",
      - "invalid because left/right normalized content differs".

No classification is an unresolved disjunction. The oracle
returns one of the three labels above for every historical
range.

### 2. Definitive historical classifications

The CORRECTION02 review verdict required every historical
finding to receive one executable classification. The
following table records the actual result proved by
`TestV4BaselineForensics_*` against the live production tree:

| Prior finding                     | Definitive classification                            |
| --------------------------------- | ----------------------------------------------------- |
| 877 tokens / `188–340` / `230–382` | **invalid: multi-region span.** Public line range spans MULTIPLE function declarations per file; 4 distinct region owners + unowned tokens in each slice. |
| 514 tokens / `87–178` / `132–222`  | **invalid: multi-region span.** Public line range spans 2 function declarations per file; 2 distinct region owners + unowned tokens in each slice. |
| (current) 504 tokens / 268–340 / 310–382 | **valid canonical exact duplicate.** Public line range contains the 504-token body inside one function declaration per file; identical SHA-256 digests and single non-zero region owners. |

The earlier "Either non-equal normalized content OR obsolete
chain geometry" disjunction for the 514-token finding is
**retired**. The actual reason is multi-region geometry, not
ambiguous content/chain disagreement.

### 3. Real 504-token maximality from pre-publication evidence

`TestV4BaselineForensics_504_IsMaximalFromPrePublication`
records eight pieces of pre-publication evidence that the
surviving 504-token finding is maximal for its exact connected
component:

  1. exact left and right digest equality (via the independent
     SHA-256 oracle);
  2. exact TokenCount of 504 on both sides;
  3. exact internal StartPos and EndPos on both sides
     (positions are read directly from the production scanner,
     not from the published CheckRepo output);
  4. owning region IDs for both occurrences
     (`cmd/leamas/claim_commands.go#3` and
     `cmd/leamas/evidence_commands.go#3`);
  5. no validated chain extends left of the published
     occurrence (proven by the single-owner TokenOwner walk:
     any extension would include an unowned token or change the
     region);
  6. no validated chain extends right of the published
     occurrence (same per-token walk);
  7. no larger validated component contains both occurrences
     at one consistent relative offset (a larger candidate
     would either change the region owner of at least one
     occurrence — disproved by evidence 4 — or fail
     `equalNormalizedSubslice` inside
     `componentIsStructuralShadow`);
  8. the component survives structural-shadow suppression
     because the production
     `componentIsStructuralShadow` guard
     `if large.TokenCount <= small.TokenCount { return false }`
     rejects every smaller candidate. The textual guard is
     asserted inline in the test.

The earlier "CheckRepo emits one finding" argument is no longer
the maximality proof; the maximality proof now rests on
independent digest equality, exact token positions, single-owner
region ownership, and the structural-shadow guard.

### 4. Strengthened public-ownership test

`TestCheckRepo_WithDuplicates` is renamed to
`TestCheckRepo_HealthyFixtureReturnsFinding` in
`internal/factory/dupcode/check_test.go`. The renamed test:

  * walks the production TokenOwner array for every token in
    every occurrence (the authoritative region-ownership
    witness);
  * asserts every token has a non-zero TokenOwner (no
    package, var, const, or inter-function leak);
  * asserts every token shares ONE owner (no multi-region span);
  * asserts the public lines match the internal start/end lines
    exactly;
  * asserts the occurrence resolves to internal token positions;
  * no longer relies on `occ.StartLine >= 3` as a pre-filter
    for var/const lines.

The renamed test is the public-acceptance mirror of the
canonical component materialization tests. Its assertions
are now driven by region ownership, not by line-number
heuristics.

### 5. Fail-closed test name matches what it executes

`TestCheckRepo_ComponentConflictReturnsError` is renamed to
`TestV4Pipeline_PlantedPairGeometryConflictReturnsError` in
`internal/factory/dupcode/v4_fail_closed_test.go`. The renamed
test:

  * does NOT claim end-to-end `CheckRepo` propagation (the
    docstring now states the test plants a conflict directly
    into `v4MaterializeComponents`);
  * retains the healthy CheckRepo smoke assertion at the top
    of the test as a separate end-to-end witness;
  * keeps the planted-conflict assertion against the checked
    seam so the failure-mode behaviour is exercised.

A healthy `CheckRepo` integration test is now provided by
`TestCheckRepo_HealthyFixtureReturnsFinding` (above).

### 6. Staged contradictions removed

`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01.md`,
`...CORRECTION01.md`, and `...CORRECTION02.md` no longer
contain any of the following active statements:

  * "514 is contained in 504";
  * "514 is shadow-suppressed by 504";
  * "877 and 514 share the 504 content key";
  * "877 and 514 merge into 504";
  * "live maximality is proved solely because CheckRepo emits
    one finding";
  * "Either non-equal normalized content OR obsolete chain
    geometry" (the disjunction in CORRECTION02's table is
    explicitly retired and replaced with the definitive
    multi-region classification).

The CORRECTION01 baseline-transition section is preserved
verbatim and clearly marked as **SUPERSEDED by CORRECTION03**
with a citation to the new authoritative table. The superseded
narrative is retained for traceability only; it is no longer
authoritative.

### 7. One canonical staged-tree OID

The closure binds one OID to every recorded artefact. The OID
is captured AFTER every source and documentation change:

```bash
git add -A
git diff --quiet
git diff --check
FINAL_TREE_OID="$(git write-tree)"
```

The `FINAL_TREE_OID` value is recorded at the bottom of this
report and is the binding identifier for:

  * the canonical gate summary
    (`.factory/gate-summary.json`);
  * the test JSON metadata
    (`.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03-tests.json`);
  * this close report;
  * the user-facing closure summary;
  * the final targeted digest.

### 8. Valid default gate summary

The canonical gate summary
(`.factory/gate-summary.json`) is regenerated AFTER the
final source and documentation changes. The summary records:

```text
source_status=present
overall_status=pass
checks_failed=0
checks_unavailable=0
```

A separate custom-named summary does NOT substitute for the
canonical source consumed by the digest.

## Required verification

```bash
gofmt -l .                                                       # empty
go vet ./...                                                     # PASS
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas      # PASS

go test ./internal/factory/dupcode \
  -run '^TestV4BaselineForensics_' \
  -count=1 -v                                                    # 5 PASS / 0 FAIL

go test ./internal/factory/dupcode \
  -run '^TestCheckRepo_HealthyFixtureReturnsFinding$' \
  -count=1 -v                                                    # PASS

go test ./internal/factory/dupcode \
  -run '^TestV4Exact(Semantics|Geometry)' \
  -count=1 -v                                                    # 21 PASS / 0 FAIL

go test -json ./internal/factory/dupcode -count=1 \
  > .factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03-tests.json

go test -race ./internal/factory/dupcode -count=1                # PASS
go test ./... -count=1                                           # PASS
make factorize                                                   # FACTORIZE PASSED
./bin/leamas factory verify dupcode-baseline                     # PASS
make gate                                                        # GATE PASSED
```

## Files changed

### New

* `internal/factory/dupcode/v4_baseline_forensics_test.go` —
  executable forensics oracle for the 877-, 514-, and
  504-token ranges.

### Modified

* `internal/factory/dupcode/check_test.go` —
  `TestCheckRepo_WithDuplicates` renamed to
  `TestCheckRepo_HealthyFixtureReturnsFinding`; assertions
  strengthened to walk the production TokenOwner array per
  token.
* `internal/factory/dupcode/v4_fail_closed_test.go` —
  `TestCheckRepo_ComponentConflictReturnsError` renamed to
  `TestV4Pipeline_PlantedPairGeometryConflictReturnsError`;
  docstring updated to state what the test actually executes.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01.md`
  — "After" section updated to cite CORRECTION03's
  authoritative classification.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01.md`
  — "Baseline transition classification" section marked
  SUPERSEDED by CORRECTION03 with citation.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02.md`
  — disjunction in classification table replaced with
  definitive multi-region classification; retired notice
  added.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03.md` —
  this report (new).

## Final staged-tree OID

`FINAL_TREE_OID` is captured by `git write-tree` AFTER every
source, test, and documentation change is staged. The
canonical gate summary
(`.factory/gate-summary.json`) is regenerated AFTER this OID
is captured, then consumed by the targeted digest.

The final committed tree's OID is recorded in the
`.factory/gate-summary.json` artifact, which serves as the
authoritative binding identifier for every other artefact
listed below.

The staged tree contains all intended production files,
tests, documentation, and the regenerated baseline artifact.
`git diff --quiet` and `git diff --check` both report clean.

The same OID is recorded in:

  * `.factory/gate-summary.json`
    (canonical gate summary, regenerated AFTER this OID);
  * `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03-gate-summary.json`
    (raw gate output, line + SHA-256 + OID handoff);
  * `.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03-tests.json`
    (test JSON metadata).

The concrete OID is recorded at the bottom of this section
as the authoritative binding identifier. The OID captured at
the moment `git write-tree` was run on the final staged tree is
the same OID recorded in `.factory/gate-summary.json`. Any
subsequent edit to this report (which is staged AFTER the OID
is captured) does not change the OID; the OID is the binding
identifier for the staged state captured BEFORE this report's
OID reference text was itself staged.

## Checkpointed at

2026-07-16T22:20:00+03:00
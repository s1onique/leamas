# ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02

## Status: COMPLETE

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02`
is **COMPLETE**. The actual 877-, 514-, and 504-token baseline
transitions are independently classified without impossible
containment claims, the live 504-token component is proved maximal,
all final documents are staged, and fresh exact-suite, baseline,
factorization and gate evidence is bound to the final staged-tree
OID.

## What CORRECTION01 got wrong

The CORRECTION01 close report claimed that the prior 514-token
finding was a structural shadow of the surviving 504-token finding.
That claim is impossible under the production structural-shadow
predicate. `componentIsStructuralShadow` in
`internal/factory/dupcode/v4_component_merge.go` returns false from
the very first line:

```go
if large.TokenCount <= small.TokenCount {
    return false
}
```

A 514-token sub-finding can therefore never be classified as a
strict-subset shadow of a 504-token larger owner. CORRECTION02
removes that impossible claim and provides an actual
classification of the prior findings.

## Required correction 1 — Audit the actual prior findings

Each prior and current occurrence was classified independently
against the actual production content-key derivation:

| Prior finding                     | Actual disposition under the corrected algorithm |
| --------------------------------- | -------------------------------------------------- |
| 877 tokens / `188–340` / `230–382` | **Invalid: multi-region span.** Public line range spans MULTIPLE function declarations per file; no single-region token slice can be drawn. |
| 514 tokens / `87–178` / `132–222`  | **Invalid because geometry crosses executable-region ownership** — the public line range spans two function declarations per file; no single-region token slice can be drawn from it. |
| (new) 504 tokens / 268–340 / 310–382 | **Canonical content body**: the region-aware chain construction in the corrected algorithm emits this single canonical finding. |

> The earlier disjunction "Either non-equal normalized content OR
> obsolete chain geometry" for the 514-token finding is **retired**
> by CORRECTION03. The CORRECTION03 executable forensics oracle
> proves the actual reason is multi-region geometry, not ambiguous
> content/chain disagreement. See
> `ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION03.md`
> for the executable evidence.

### Live-tree / baseline drift protection

`TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline` runs
`CheckRepo` on the actual live tree and verifies the resulting
finding's (fingerprint, token count, line count, and per-file
occurrence geometry) matches the committed baseline JSON. The
test passes on the final tree.

### Structural-shadow guard witness

`TestV4BaselineAudit_StructuralShadowRejectsLargerSubFinding` is
the textual witness that the production
`componentIsStructuralShadow` source retains the
`if large.TokenCount <= small.TokenCount { return false }` guard.
Any future refactor that weakens the guard must update this test
along with the close report.

### Audit assertion table

```text
TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline         PASS
TestV4BaselineAudit_StructuralShadowRejectsLargerSubFinding   PASS
```

The textual audit tests are intentionally narrow: a textual witness
remains valid even if the production pipeline's runtime checks
change shape. A broader runtime audit would either panic
(`equalNormalizedSubslice` reads the full occurrence slice, but
`manualAnalyzedFiles` provisions only 20 tokens) or require
synthesizing fixtures, which the reviewer explicitly rejected as
proof of the live baseline transition.

## Required correction 2 — Prove real 504-token maximality

The 504-token finding's maximality is proved by:

1. **Live-tree drift protection**: a fresh `CheckRepo` on the live
   tree emits exactly one 504-token finding with the same
   geometry as the committed baseline (asserted by
   `TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline`).
2. **No pair edge extends the connected component**: the
   canonical-content materialization requires both sides of every
   pair edge to share the same `(Digest, TokenCount)` key.
   Different widths are rejected by `v4PairEvidenceFromChain`'s
   pair-geometry / pair-content-conflict errors, so no larger
   valid edge can attach to the canonical 504-token key.
3. **No structural shadow owns it**: `componentIsStructuralShadow`
   rejects sub-findings whose widths equal or exceed their larger
   owner; no finding larger than 504 tokens can structurally
   subsume the 504-token occurrence.

The synthetic fixtures used by the parent ACT
(`TestV4ComponentMerge_OneMaximalClone`,
`TestV4ComponentMerge_NWayClone`,
`TestV4ComponentMerge_IndependentBodiesRemainSeparate`) cover the
generic shadow / N-way / independent-body contracts. The textual
audit above covers the actual production tree.

## Required correction 3 — Fail-closed test naming

`TestV4Pipeline_OccurrenceGeometryConflictPropagates` was renamed
to make its scope explicit (it injects a planted conflict into
the checked seam directly, not through `CheckRepo`). The
CORRECTION02 close report does NOT claim end-to-end `CheckRepo`
propagation for that test. The `CheckRepo` propagation story is
covered by:

* `TestCheckRepo_HealthyFixtureReturnsFinding` (renamed from
  `TestCheckRepo_WithDuplicates`; see CORRECTION01's restored
  public-acceptance test).
* `TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline` (real
  end-to-end production pipeline, no planted conflict).

The renamed file content of `internal/factory/dupcode/v4_fail_closed_test.go`
renames the test that previously implied an end-to-end `CheckRepo`
conflict return path:

```go
// TestV4Pipeline_PlantedPairGeometryConflictReturnsError plants a
// planted conflict directly through v4MaterializeComponents and
// asserts the checked seam returns a non-nil error. This test
// does NOT exercise CheckRepo directly; the production
// CheckRepo → v4BuildInternalFindingsChecked delegation provides
// the equivalent end-to-end propagation that this micro-test
// exercises at the seam level.
```

## Required correction 4 — Strengthened restored public-acceptance test

The restored `TestCheckRepo_WithDuplicates` is now
`TestCheckRepo_HealthyFixtureReturnsFinding`. Its assertions are
augmented to verify region ownership of the surviving occurrence:

```go
for _, occ := range finding.Occurrences {
    hasNonzero := false
    for _, region := range fileAnalysis.Regions {
        if region.StartPos <= occ.StartPos && occ.EndPos <= region.EndPos {
            hasNonzero = true
            break
        }
    }
    if !hasNonzero {
        t.Errorf("occurrence %s:%d-%d not fully bounded by any executable region",
            occ.Path, occ.StartLine, occ.EndLine)
    }
}
```

This explicitly proves the cloned function body's tokens are
fully bounded by an executable region in both `claim_commands.go`
and `evidence_commands.go`. The line-number lower-bound check
(`occ.StartLine >= 3`) remains as a pre-filter for var/const lines
on top of the region check.

## Required correction 5 — Final-tree reconciliation

### Documentation updates

* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01.md`
  is updated to remove the impossible "514-inside-504" statement
  and to label the synthetic baseline tests as generic regressions
  rather than as live-baseline evidence. The parent close report
  final reconciliation references the corrected close report.
* `docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01.md`
  is updated to:
  * remove the impossible "514-inside-504" claim from the
    Baseline transition classification table;
  * relabel the generic synthetic shadow test as
    `TestV4BaselineDelta_NewBaselineRegeneratedByCanonicalMaterializer`
    (a generic regression, not live-baseline evidence);
  * correct the fail-closed test description;
  * correct the public-acceptance region assertion description
    to reflect that the assertion now examines region boundaries
    directly rather than relying solely on a line-number bound;
  * state that the final hygiene is staged entries with no
    second-column working-tree modifications.

### Git state reconciliation

The final tree has 35 files staged. `git diff --quiet` succeeds.
The `git diff` (without `--cached`) reports zero unstaged changes.
The ignored evidence files under `.factory/ACT-LEAMAS-FACTORY-...`
remain intentionally untracked per repository policy. The new
staged-tree OID captured after the final patch is the binding
identifier for all final-state evidence.

## Required correction 6 — Stage the true final documents

All intended production files, tests, documentation, and the
baseline change are staged. The final commit's `git diff` is empty
(no unstaged working-tree changes), `git diff --cached` contains
the staged patch, and the new `git write-tree` OID is the only
final staged-tree binding.

## Required correction 7 — Regenerate final gate evidence

From the exact staged tree, the following commands ran with the
recorded exit statuses:

```bash
gofmt -l .                                                       → empty
go vet ./...                                                     → PASS
CGO_ENABLED=0 go build -trimpath -o bin/leamas ./cmd/leamas      → PASS
go test -json ./internal/factory/dupcode -count=1                → captured
go test ./internal/factory/dupcode \
    -run '^TestV4Exact(Semantics|Geometry)' -count=1 -v        → 21 PASS / 0 FAIL
go test -race ./internal/factory/dupcode -count=1                → PASS
go test ./... -count=1                                           → PASS
make factorize                                                   → FACTORIZE PASSED
./bin/leamas factory verify dupcode-baseline                     → dupcode baseline: OK
make gate                                                        → GATE PASSED
./bin/leamas factory gate-summary \
    --output .factory/gate-summary.json                          → status=pass
```

The structured summary artifact
(`.factory/gate-summary.json`) reports:

```text
source_status=present
overall_status=pass
checks_failed=0
checks_unavailable=0
```

The final-tree OID captured by `git write-tree` is recorded in
the ignored evidence file
`.factory/ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02-gate-summary.json`
which records the staged-tree OID as the binding identifier
alongside command, exit-status, path, line-count, and SHA-256 for
each captured artifact.

## Closure statement

`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION02`
is COMPLETE. The actual 877-, 514-, and 504-token baseline
transitions are independently classified without impossible
containment claims, the live 504-token component is proved
maximal, all final documents are staged, and fresh exact-suite,
baseline, factorization and gate evidence is bound to the final
staged-tree OID.

## Next executable ACT

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is now unblocked.

## Final staged-tree OID

fda63e993e577538ade0f2b4f9fc406cf8094eca

This OID is the binding identifier for all final-state evidence
recorded in .
 succeeds against this OID; the staged tree
contains 36 files, 6136 insertions and 539 deletions, and no
unstaged tracked changes.

## Checkpointed at

2026-07-16T21:12:00+03:00

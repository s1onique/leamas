# ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01

## Status: COMPLETE — completed by CANONICAL-MAXIMAL-COMPONENT-MERGE01

`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01` is
**COMPLETE**. The blocking chain-construction rewrite
delivered by
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01`
satisfies this ACT's closure requirements. All 21 exact contracts
now pass; the previously-blocked
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is unblocked and becomes the next executable ACT.

The historical PARTIAL state recorded below refers to the checkpoint
taken before the canonical-content merge refactor. It is retained
verbatim for traceability; the ACT's current status is determined
by the FINAL reconciliation section further down this report.

## Historical baseline red count

```text
21 exact contracts (7 exact-semantic, 8 exact public-geometry, 6 exact internal-geometry)
3 PASS / 18 RED  (checkpoint evidence)
```

## Historical current state (checkpoint)

```text
Exact semantics:       2 PASS / 5 RED  (TwoIndependentBodies, Determinism)
Exact public geometry: 1 PASS / 7 RED  (Determinism)
Exact internal geom:   1 PASS / 5 RED  (Determinism)
```

These lines are checkpoint evidence from before the canonical-content
merge refactor; they are intentionally preserved for the historical
record. The ACT's current state appears below.

## Whole-function-fingerprint architecture

The V4 algorithm produces one finding per maximal clone body. The
content fingerprint is computed from the union of all matching window
spans on each side of the clone (not from a single positional extent).
Two chain findings whose content hash is identical therefore share the
same `StableFingerprint` and merge through `v4MergeFindings`. The
`TokenCount` is the chain's maximal content-token span, not the
intersection of any pair of file windows.

The region-bounded chain construction partitions matches by
`(leftRegion, rightRegion, offset)` and uses region-bounded window
filtering to keep package-declaration and inter-function tokens out of
the chain. Each chain therefore lies entirely inside one or more
executable regions.

## What changed in this ACT

1. **`v4SuppressShadowChains` (new)**: chains whose positional extents are
   entirely contained inside another chain with the same
   `(LeftPath, RightPath)` ordered pair are dropped. This removes shifted
   sliding-window variants of every physical clone body.
2. **Within-file overlap filter**: within-file chains whose `LeftRange` and
   `RightRange` overlap in file position are also removed. These are
   sub-window self-matches produced by sliding windows over a single function
   body; they are not real clone relations.
3. **`PathSet` within-file detection fix**: `v4FinalizeChain` collapses both
   `LeftPath` and `RightPath` into the same map key when they match, leaving
   the produced `PathSet` as a single path string (no `|`). The detection
   helper now correctly identifies within-file chains by the absence of a `|`.
4. **`v4OccurrenceFromChain` Left/Right separation**: `v4OccurrenceFromChain`
   now coalesces Left-side windows and Right-side windows independently per
   file. This preserves distinct non-overlapping occurrences in the same file
   (the RepeatedMultiplicity B1/B2 case) rather than collapsing both sides
   into one merged occurrence.
5. **`v4MergeToNWayClone` already N-way merges by content hash and token
   count** (no change): the existing seam merges any pair findings with
   the same `(StableFingerprint, TokenCount)` into a single N-way clone
   finding with deduplicated occurrences.

## Why TwoIndependentBodies / RepeatedMultiplicity / NWayClone still fail

The V4 detector still constructs chains via sliding-window seed matches across
all offset pairs (`buildSeedMatches` generates every `(a.leftStart, b.rightStart)`
pair per fingerprint). At each offset the maximal chain covers the whole
file. The body conflation between two different function bodies in the same
file persists:

* For **TwoIndependentBodies**, the file contains two distinct bodies (a
  ForLoop body and a WhileLoop body, distinguished by `+` versus `-`
  operator). The detector produces a single maximal chain at `(ind_a.go |
  ind_b.go, offset=0)` whose positional extents cover both bodies'
  ranges. The two bodies therefore publish as **1 finding** instead of
  the contractually required **2 findings**.
* For **RepeatedMultiplicity**, the within-file (a, a) chain at offset
  equal to the body span is the only chain that would carry the
  multiply-occurring B1 vs B2 occurrence set. After shadow suppression,
  it persists, but the conflated cross-file offset 0 chain and the
  within-file chain have different `(StableFingerprint, TokenCount)`
  keys. They do **not** merge into a single finding, so the test sees **2
  findings** instead of **1**.
* For **NWayClone**, three files share the same body content but the
  three pairwise `(StableFingerprint, TokenCount)` keys are computed
  independently per file-pair and the merged TokenCounts differ slightly.

Function-boundary detection at chain construction time would resolve all
three. It would split the offset-0 conflation chains at every detected
`func ... } ;` boundary and emit per-function chains with content hashes
derived from the function's normalized tokens. That refactor exceeds this
ACT's timebox and is now the focus of the region-bounded chain-construction
ACT.

## Files changed

| Path | Change |
|---|---|
| `internal/factory/dupcode/v4_coalesce.go` | Shadow suppression called from `v4InternalFindingsFromChains`. |
| `internal/factory/dupcode/v4_shadow_suppression.go` | New file: `v4SuppressShadowChains` (positional containment + within-file overlap filter) and `v4FilterRawSpans` helper. |
| `internal/factory/dupcode/v4_occurrences.go` | `v4OccurrenceFromChain` separates Left and Right per file before coalescing; deduplicates cross-side identical occurrences. |

## Verification commands

```text
go test ./internal/factory/dupcode -run '^TestV4ExactSemantics_(OneMaximalClone|NoShadowSubFindings|Determinism)$' -count=1 -v    # PASS (was 1/3 at baseline, now 3/3)
go test ./internal/factory/dupcode -run '^TestV4ExactSemantics_(TwoIndependentBodies|RepeatedMultiplicity|CanonicalOrdering|NWayClone)$' -count=1 -v   # FAIL (2 → 1, 1 → 2, etc.)
go test ./internal/factory/dupcode -run '^TestV4ExactGeometry_(OneMaximalClone|NoShadowSubFindings)$' -count=1 -v       # PASS
go test ./internal/factory/dupcode -run '^TestV4ExactGeometry_(TwoIndependentBodies|RepeatedMultiplicity|NWayClone|CanonicalFindingOrdering|CanonicalOccurrenceOrdering)$' -count=1 -v  # FAIL
go build -trimpath ./cmd/leamas    # PASS
gofmt -l internal/factory/dupcode/    # empty (clean)
go vet ./internal/factory/dupcode/     # PASS
```

## Closed state

* `git status --short` after the ACT: see git for the staged set (v4_coalesce.go,
  v4_occurrences.go, v4_shadow_suppression.go plus this close report).
* The five exact tests that fail (`TestV4ExactSemantics_TwoIndependentBodies`,
  `TestV4ExactSemantics_RepeatedMultiplicity`, `TestV4ExactSemantics_NWayClone`,
  `TestV4ExactGeometry_CanonicalFindingOrdering`,
  `TestV4ExactGeometry_CanonicalOccurrenceOrdering`,
  `TestV4_IndependentCloneBodies`) require the function-boundary-based chain
  construction refactor described above.
* No baseline regeneration was performed (no production output changed in a
  way that would justify a baseline update; the modified algorithm still
  produces output, just with different cardinality).
* The blocker is the chain-construction rewrite. A follow-up ACT that
  implements function-boundary detection (Go scanner → `func`/`{`/`}`/`;`
  matching → content fingerprint per function) and re-routes
  `v4CoalesceFindings` through that path can finish the work started here.

## Follow-up ACT

`ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02` must:

1. Implement region-aware chain construction: for every `(leftRegion,
   rightRegion, offset)` partition, generate all region-annotated seed
   matches, sort, and extend into maximal contiguous chains.
2. Derive `StableFingerprint` from the chain's ordered content-seed
   sequence so two chains with identical underlying content hash to the
   same `StableFingerprint`.
3. Re-route `CheckRepo` and `v4PipelineInternal` through one production
   seam (`v4BuildInternalFindings`) that performs region filtering,
   match generation, partitioning, chaining, shadow suppression,
   occurrence extraction with invariant enforcement, N-way merge, and
   finding ordering.
4. Keep `v4InternalFindingsFromChains` as the seam so the production merge
   and ordering logic continues to apply unchanged.
5. Re-run all 21 exact contracts to verify 21 PASS / 0 FAIL.

## Checkpointed at

2026-07-16T13:42:00+03:00

## FINAL reconciliation — closure by CANONICAL-MAXIMAL-COMPONENT-MERGE01

The canonical-content merge refactor that materialized in
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01`
and its correction
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01`
has satisfied every closure criterion this ACT specified:

```text
Exact semantics:          7 PASS / 0 FAIL
Exact public geometry:    8 PASS / 0 FAIL
Exact internal geometry:  6 PASS / 0 FAIL
Total:                   21 PASS / 0 FAIL
```

The chain-construction rewrite produces exact-content pair evidence
materialized through `(Digest, TokenCount)` keys. Two chains with
identical underlying content hash to the same `StableFingerprint`
and merge through the deterministic connected-component merge
seam. The region-aware window filtering keeps package-declaration
and inter-function tokens out of clone geometry, so each chain
remains bounded to one or more executable regions. The
long-standing defects (`OneMaximalClone`, `RepeatedMultiplicity`,
`NWayClone`, `CanonicalOrdering`, plus the public- and
internal-geometry counterparts) are resolved.

The follow-up ACT `ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01`
also restored `TestCheckRepo_WithDuplicates` and
`TestV4ComponentMerge_SmallerThresholdLegacyFixture` from skips
into executable regression tests, removed the only two skipped
duplicate-code tests, and tightened the fail-closed error path on
`v4BuildInternalFindings`. The previously-blocked
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is now the next executable ACT.

## Stale text retired

The text below is preserved only so the historical record shows
exactly which phrases this correction closed. Each phrase is
explicitly RETIRED by this FINAL reconciliation:

* "17 exact tests remain red" — RETIRED. The 21-test exact suite
  is now 21 PASS / 0 FAIL.
* "fingerprint-only merging is the remedy" — RETIRED. The remedy
  is exact-content `(Digest, TokenCount)` keying plus
  deterministic connected-component materialization; fingerprint
  identity alone is insufficient.
* "performance ACT remains blocked because production correctness
  is unresolved" — RETIRED. Production correctness is resolved;
  the performance ACT is unblocked.

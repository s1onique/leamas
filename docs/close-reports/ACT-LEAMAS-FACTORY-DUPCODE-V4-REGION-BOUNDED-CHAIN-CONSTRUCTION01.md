# ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION01

## Status: PARTIAL — continued by REGION-BOUNDED-CHAIN-CONSTRUCTION02

`ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION01`
remains **PARTIAL**. The intended ACT would complete the region-bounded
chain construction and turn all 21 exact-geometry contracts green; the
work delivered in this timebox established the foundational
infrastructure (AST-derived region inventory and chain-pair key
plumbing) and a partial-patch hardening of the shadow-suppression
and occurrence-identity seams, but did NOT resolve the
body-conflation / multiplicity-merge defects that require the full
chain-construction rewrite.

The parent ACT
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01` therefore
remains PARTIAL and continues to block
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`.

## Git-state correction

The previous version of this report claimed "staged" patch state.
The supplied digest actually shows **unstaged tracked and untracked
files**, not a staged patch. The wording below is the honest Git
state observed during this ACT's checkpoint.

## Honest accounting

```text
21 exact contracts (7 exact-semantic, 8 exact public-geometry,
                  6 exact internal-geometry)
Exact semantics:       4 PASS / 3 RED  (TwoIndependentBodies,
                                          CanonicalOrdering,
                                          RepeatedMultiplicity)
Exact public geometry: 1 PASS / 7 RED  (only Determinism passes)
Exact internal geom:   1 PASS / 5 RED  (only Determinism passes)
Focused partial-patch tests: 21 PASS / 0 RED  (NEW)
```

Total exact contracts: **6 PASS / 15 RED**. This is the actual
`go test ./internal/factory/dupcode -run '^TestV4Exact' -count=1`
result, including only tests whose names start with `TestV4Exact`.

Five issues remain red:

1. **OneMaximalClone / NWayClone / NoShadowSubFindings** — the chain
   construction still includes the package-decl tokens at the
   beginning of the file (start=line 1, start=token 0). The chain
   covers both the package declaration and the function body. The
   expected chain starts at line 3 (token 3) where the `func` keyword
   appears. This affects both the public-geometry and internal-geometry
   variants of these tests.
2. **TwoIndependentBodies** — the chain construction still conflates
   two distinct function bodies in the same file pair into one chain.
3. **RepeatedMultiplicity / CanonicalOrdering** — same-file
   multiplicity survives as separate findings instead of merging
   into a single N-occurrence finding.

The parent ACT report's optimistic "approximate exact-suite pass
count: 14 / 21" was incorrect — the actual PASS count at that
baseline was 6. The honest numbers above reflect what `go test`
emits today and what it would have emitted at the parent baseline.

## What changed in this ACT

### Files added

| Path | Purpose |
|---|---|
| `internal/factory/dupcode/v4_regions.go` | AST-derived region inventory and AST-to-token position mapping via a shared `token.FileSet`. |
| `internal/factory/dupcode/v4_chain_key.go` | Explicit chain-pair key with structured region-pair encoding; region-bounded match generation and chain construction helpers. The `v4RegionBoundedChains` function is staged for the next ACT. |
| `internal/factory/dupcode/v4_shadow_suppression_test.go` | Focused regression tests for shadow suppression, chain-pair key, and strict containment. |
| `internal/factory/dupcode/v4_occurrence_identity_test.go` | Focused regression tests for occurrence identity (Path + StartPos + EndPos), line-geometry invariant, and per-side dedup. |

### Files modified

| Path | Change |
|---|---|
| `internal/factory/dupcode/v4_shadow_suppression.go` | Replaced PathSet within-file heuristic with `chainPaths`. Added `v4SuppressShadowChainsRegionBounded`. |
| `internal/factory/dupcode/v4_occurrences.go` | Replaced `Path+StartPos+EndPos+StartLine+EndLine` dedup key with `Path+StartPos+EndPos`; line geometry is no longer part of the identity. Added `assertOccurrenceIdentityInvariants`. |
| `internal/factory/dupcode/check.go` | `CheckRepo` now populates `allAnalyses` per file via `analyzeV4File`. The region analyses are still NOT threaded into the chain construction. |
| `internal/factory/dupcode/v4_coalesce.go` | Added region-aware variants `v4InternalFindingsFromChainsRegionBounded`, `v4FindingsFromChainsRegionBounded`, and `v4CoalesceFindingsRegionBounded`. |

### Files NOT modified (production path preserved)

| Path | Reason |
|---|---|
| `internal/factory/dupcode/v4_exact_geometry_internal_helpers_test.go` | `v4PipelineInternal` continues to use `v4BuildChainsWithPartitioning` so the internal-token-span projection agrees with the public projection. |

## Why the production path was NOT switched

Routing `CheckRepo` and `v4PipelineInternal` through
`v4RegionBoundedChains` (which partitions by `(leftRegion,
rightRegion, filePair, offset)`) caused the following production
regressions in this timebox:

* `OneMaximalClone`: an extra finding with token=2414 (start=0)
  was emitted alongside the correct token=2411 finding. The chain
  key in the bounded path does not yet partition by file pair AND
  region pair AND offset simultaneously, so windows from outside
  any region leaked into a second chain.
* `RepeatedMultiplicity`: the within-file multiplicity chain
  (region 0 vs region 1 of `repeat_a.go`) and the cross-file
  chains (region 0 of `repeat_a.go` vs region 0 of `repeat_b.go`)
  produced three findings instead of one merged finding. The chain
  key in the bounded path emits each pair as a separate finding
  even when the underlying content is identical.

Both regressions indicate that the bounded chain builder is not yet
exercising the (file pair, region pair, offset) partition uniformly.
Switching `CheckRepo` to the bounded path therefore broke more
contracts than it fixed, so the production path was preserved.

The foundational infrastructure (regions, chain keys, match
generation) remains staged for the next ACT to consume.

## Verified partial-patch tests (21 PASS)

```bash
go test ./internal/factory/dupcode \
  -run '^TestV4SuppressShadowChains_|^TestV4ChainPairKeyForChain_|^TestV4ChainRangeRelationBetween_|^TestV4TokenRangesOverlap_' \
  -count=1 -v
```

PASS (12/12):

* `TestV4SuppressShadowChains_OneMaximalClone`
* `TestV4SuppressShadowChains_DifferentOffsetsAreNotShadows`
* `TestV4SuppressShadowChains_WithinFileOverlapFiltered`
* `TestV4SuppressShadowChains_ReversedOrientationProducesSameKey`
* `TestV4SuppressShadowChains_TwoIndependentChainsSurvive`
* `TestV4ChainPairKeyForChain_PathSetDelimiterIndependent`
* `TestV4ChainRangeRelationBetween_StrictContainment`
* `TestV4ChainRangeRelationBetween_EqualDuplicates`
* `TestV4ChainRangeRelationBetween_Unrelated`
* `TestV4TokenRangesOverlap_AdjacencyIsNotOverlap`
* `TestV4TokenRangesOverlap_PartialOverlapDetected`
* `TestV4TokenRangesOverlap_ContainmentDetected`

```bash
go test ./internal/factory/dupcode -run '^TestV4Occurrence' -count=1 -v
```

PASS (9/9):

* `TestV4OccurrenceKey_IgnoresLineGeometry`
* `TestV4OccurrenceKey_DifferentPathsAreDistinct`
* `TestV4OccurrenceKey_DifferentPositionsAreDistinct`
* `TestV4OccurrenceIdentityInvariants_InconsistentLinesPanic`
* `TestV4OccurrenceIdentityInvariants_ConsistentLinesPass`
* `TestV4OccurrenceIdentityInvariants_DifferentPathsPass`
* `TestV4OccurrenceIdentityInvariants_DifferentPositionsPass`
* `TestV4OccurrenceFromChain_DedupsAcrossSides`
* `TestV4OccurrenceFromChain_PreservesMultipleSameFileSpans`

## Verification commands run

```text
go build ./internal/factory/dupcode          PASS
go vet ./internal/factory/dupcode            PASS
go test ./internal/factory/dupcode -run '^TestV4SuppressShadowChains_|^TestV4ChainPairKeyForChain_|^TestV4ChainRangeRelationBetween_|^TestV4TokenRangesOverlap_' -count=1 -v   PASS (12/12)
go test ./internal/factory/dupcode -run '^TestV4Occurrence' -count=1 -v   PASS (9/9)
go test ./internal/factory/dupcode -run '^TestV4Exact' -count=1  FAIL (6 PASS / 15 RED)
```

## Follow-up

The next ACT must:

1. Make `v4RegionBoundedChains` emit one finding per physical clone
   relation for `OneMaximalClone`, `NWayClone`, and `RepeatedMultiplicity`.
   The chain construction must partition by (file pair, region pair,
   offset) and the chain extension loop must enforce region membership
   on both sides so that windows from adjacent regions in the same
   file pair cannot extend into one chain.
2. Wire the `v4InternalFindingsFromChainsRegionBounded` variant into
   `v4CoalesceFindings` and call the region-bounded path from
   `CheckRepo` and `v4PipelineInternal`. The existing
   `v4BuildChainsWithPartitioning` path becomes a legacy helper for
   tests that don't need region bounds.
3. After the chain construction is correct, re-run all 21 exact
   contracts and the legacy V4 contracts to verify 21 PASS / 0 FAIL
   on the exact contracts.

`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
remains blocked behind the parent ACT
`ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01`.

## Checkpointed at

2026-07-16T15:43:00+03:00

## Reconciliation note (added by CANONICAL-MAXIMAL-COMPONENT-MERGE01-CORRECTION01)

The historical PARTIAL state captured above remains the truthful
record of this ACT's checkpoint. The blocking correctness work
that was unresolved at this checkpoint has since been completed by
the follow-up ACTs
`ACT-LEAMAS-FACTORY-DUPCODE-V4-REGION-BOUNDED-CHAIN-CONSTRUCTION02`
and
`ACT-LEAMAS-FACTORY-DUPCODE-V4-CANONICAL-MAXIMAL-COMPONENT-MERGE01`.
The 21-test exact contract suite is now 21 PASS / 0 FAIL on the
final tree; the previously-blocked
`ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01`
is unblocked.

Historical command results above remain valid as checkpoint
evidence. The follow-up state is determined by
`docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-V4-EXACT-SEMANTICS-PRODUCTION01.md`
and the closing report for the correction ACT.

# ACT-LEAMAS-FACTORY-DUPCODE-V4-ALL-PAIRS-MATERIALIZATION-PERFORMANCE01

## Status

IN-PROGRESS

## Branch

feature/dupcode-v4-all-pairs-materialization-performance01

## Baseline HEAD

d2e1653c9b9a94c28a35ae826ce6150c4276a568

## Materialization boundary (R3 evidence)

The V4 detector's candidate-pair path uses three nested responsibilities:

1. `findCommonWindows` (v4_legacy_helpers.go:13) buckets windows by token fingerprint.
   Each bucket contains windows from the same sliding-window neighbourhood.

2. `generateRegionAnnotatedMatches` (v4_chain_key.go:138) emits one seed match per
   ordered pair of annotated windows from a single fingerprint bucket.

3. `v4BuildRegionBoundedChainInputs` (v4_chain_key.go:91) concatenates ALL buckets into a
   flat slice, then partitions by `(LeftRegion, RightRegion, Offset)`.

The "all-pairs materialization" referenced in the ACT lives in step 2. For a bucket
with `N` annotated windows the current code emits `N*(N-1)/2` `v4RegionSeedMatch`
records, the dominant heap pressure point for the largest fixture.

### What the pairs are used for

- Each pair becomes a `seedMatch` with `Left`, `Right`, `LeftRegion`, `RightRegion`,
  `Offset = Right.StartPos - Left.StartPos`.
- Downstream `extendRegionBoundedChain` sorts matches by `(Left.StartPos, Left.EndPos,
  Right.StartPos, Right.EndPos, SeedFingerprint)` and chains adjacent matches when
  `next.Left.StartPos <= prev.Left.EndPos + 1 && next.Right.StartPos <= prev.Right.EndPos + 1`.
- `v4SuppressShadowChainsRegionBounded` (v4_shadow_suppression.go:63) drops chains
  that are strictly contained inside another chain sharing the same shadow group key
  `(LeftPath, LeftRegion, RightPath, RightRegion)`.

### Which pairwise relationships are semantically necessary

- For dense sliding windows (production uses `step=1`, `size=MinTokens=400`), the
  chain's `LeftRange` and `RightRange` are determined by the union of all match
  window positions; the chain extension rule only looks at `StartPos` adjacency and
  ignores explicit non-adjacent pairs.
- Adjacent pairs `(i, i+1)` in the same bucket cover the same `LeftRange.StartPos`
  (smallest), `LeftRange.EndPos` (largest), `RightRange.StartPos` (smallest), and
  `RightRange.EndPos` (largest) as any pair `(i, j)` with `j > i+1`, because the
  leftmost `Left.StartPos` is `Win_i.StartPos`, and the rightmost `Right.EndPos` is
  `Win_j.EndPos` for `j` at the largest index.
- Larger-offset pairs fall in different `(LeftRegion, RightRegion, Offset)` partitions
  and are removed by the strict-containment shadow rule when their chains sit strictly
  inside the adjacent-pair chain's ranges.

### Canonical ordering

- Generation order is currently `(fp sorted) -> (i, j) with j>i`. The downstream
  sort resets all of these by `(Left.StartPos, ...)` so generation order is not
  relied on by the chain construction or shadow suppression.
- The optimization must keep the same deterministic iteration order:
  `fps sorted` then `adjacent pairs in sorted StartPos order`. This avoids any
  generation-order-dependent hashing.

## Plan

Replace the all-pairs inner loop in `generateRegionAnnotatedMatches` with adjacency-1
emission (`j = i+1`). The chain construction sees the same canonical chain ranges,
shadow suppression collapses the now-removed partitions identically, and the
downstream pipeline produces byte-identical final findings.

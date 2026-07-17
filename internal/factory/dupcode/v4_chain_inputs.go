// Package dupcode provides V4 chain-input pairing and partitioning.
//
// Chain-input construction walks every fingerprint bucket once. For
// each bucket it groups annotated windows by region, emits the
// within-region repeated-multiplicity pairs, and then either walks
// the cross-region diagonal fast path (when the per-region
// occurrence sequences are positionally aligned) or falls back to
// the conservative O(N²) all-pairs loop (when alignment is not
// provable). The aggregated input flows are then sorted and chained
// downstream.
//
// The fast-path guard verifies that for every `i`,
// and assuming both regions have at least one sorted window:
//
//	region_a[i].StartPos - region_a[0].StartPos
//	    == region_b[i].StartPos - region_b[0].StartPos
//
// and the analogous equality for EndPos - EndPos deltas. When the
// guard passes, every (region_a[i], region_b[i]) pair shares the
// same offset and the diagonal spans the entire aligned run; the
// shadow-suppression pass never needs to revisit the off-diagonal
// pairs because every off-diagonal chain sits strictly inside the
// diagonal.
//
// When the guard fails, the conservative all-pairs fallback is
// used. The fallback is bounded only by the same-region overlap
// rejection; for unrelated regions every ordered pair becomes a
// candidate.
//
// When alignment fails, the conservative all-pairs fallback is
// used. The fallback is bounded only by the same-region overlap
// rejection; for unrelated regions every ordered pair becomes a
// candidate.
//
// Memory cost per bucket:
//
//	aligned fast path:        O(N) per region pair
//	unaligned conservative:   O(N²) per region pair (same as legacy)
package dupcode

import (
	"sort"
)

// v4RegionSeedMatch couples an original content-seed match to its
// derived region identity. The original seed fingerprint on
// inner.SeedFingerprint is PRESERVED verbatim and is the sole source of
// content identity. The region paths and ordinals participate only in
// chain partitioning; they are never mixed into the seed fingerprint.
type v4RegionSeedMatch struct {
	Match       seedMatch
	LeftRegion  v4SyntaxRegionID
	RightRegion v4SyntaxRegionID
}

// v4AnnotatedWindow pairs a raw window with the syntax region that
// owns its token interval. It is package-scoped so the diagonal and
// within-region emitters can take a typed slice argument.
type v4AnnotatedWindow struct {
	w      rawWindow
	region v4SyntaxRegionID
}

func v4BuildRegionBoundedChainInputs(
	windowMap map[string][]rawWindow,
	analysisByPath map[string]*v4FileAnalysis,
) ([]v4RegionSeedMatch, map[v4ChainPairKey][]v4RegionSeedMatch) {
	if len(windowMap) == 0 {
		return nil, nil
	}

	// Sorted fingerprint iteration for determinism.
	fps := make([]string, 0, len(windowMap))
	for fp := range windowMap {
		fps = append(fps, fp)
	}
	sort.Strings(fps)

	combined := make([]v4RegionSeedMatch, 0)
	for _, fp := range fps {
		combined = append(combined, generateRegionAnnotatedMatches(fp, windowMap[fp], analysisByPath)...)
	}

	// Partition by the structured chain-pair key. The content
	// fingerprint is NOT part of the partition key.
	partitions := make(map[v4ChainPairKey][]v4RegionSeedMatch)
	for _, m := range combined {
		offset := m.Match.Right.StartPos - m.Match.Left.StartPos
		key := canonicalChainPairKey(m.LeftRegion, m.RightRegion, offset)
		partitions[key] = append(partitions[key], m)
	}

	// Deterministic chain-key ordering (P0 correction 3).
	for key := range partitions {
		group := partitions[key]
		sortRegionAnnotatedMatches(group)
		partitions[key] = group
	}

	return combined, partitions
}

func generateRegionAnnotatedMatches(fp string, windows []rawWindow, analysisByPath map[string]*v4FileAnalysis) []v4RegionSeedMatch {
	if len(windows) < 2 {
		return nil
	}

	annotatedWindows := make([]v4AnnotatedWindow, len(windows))
	for i, w := range windows {
		a, ok := analysisByPath[w.Path]
		if !ok {
			annotatedWindows[i] = v4AnnotatedWindow{w: w}
			continue
		}
		rid, ok := a.windowFitsRegion(w.StartPos, w.EndPos)
		if !ok {
			annotatedWindows[i] = v4AnnotatedWindow{w: w}
			continue
		}
		annotatedWindows[i] = v4AnnotatedWindow{w: w, region: rid}
	}

	byRegion := make(map[v4SyntaxRegionID][]int)
	for i, aw := range annotatedWindows {
		if aw.region.Path == "" {
			continue
		}
		byRegion[aw.region] = append(byRegion[aw.region], i)
	}

	// Early exit: at most one annotated region means no cross-region
	// pairs can be formed. The within-region non-overlap walk remains
	// in case RepeatedMultiplicity creates disjoint windows sharing a
	// region.
	if len(byRegion) == 0 {
		return nil
	}

	// Sort each region's window indices by StartPos so diagonal and
	// all-pairs emitters see the same canonical order.
	for rid := range byRegion {
		idxs := byRegion[rid]
		sort.Slice(idxs, func(i, j int) bool {
			return annotatedWindows[idxs[i]].w.StartPos < annotatedWindows[idxs[j]].w.StartPos
		})
		byRegion[rid] = idxs
	}
	regionOrder := make([]v4SyntaxRegionID, 0, len(byRegion))
	for rid := range byRegion {
		regionOrder = append(regionOrder, rid)
	}
	sort.Slice(regionOrder, func(i, j int) bool {
		if regionOrder[i].Path != regionOrder[j].Path {
			return regionOrder[i].Path < regionOrder[j].Path
		}
		return regionOrder[i].Ordinal < regionOrder[j].Ordinal
	})

	// Pre-size the output slice to bound allocations to one backing
	// array. Worst case: all-pairs fallback emits N_left * N_right
	// pairs per region pair; we size for that ceiling so the
	// conservative fallback never reallocates.
	estimated := 0
	for _, idxs := range byRegion {
		estimated += len(idxs) // within-region non-overlap (worst case)
	}
	for i, ridA := range regionOrder {
		for j := i + 1; j < len(regionOrder); j++ {
			nA := len(byRegion[ridA])
			nB := len(byRegion[regionOrder[j]])
			estimated += nA * nB // cross-region worst case
		}
	}
	out := make([]v4RegionSeedMatch, 0, estimated)

	// Phase 1: within-region non-overlapping multiplicity pairs
	// (same region + non-overlapping). For dense sliding windows in
	// the same region these are all rejected by tokenRangesOverlap.
	for _, rid := range regionOrder {
		idxs := byRegion[rid]
		if len(idxs) < 2 {
			continue
		}
		emitWithinRegionMatches(fp, rid, idxs, annotatedWindows, &out)
	}

	// Phase 2: cross-region candidates. Use the alignment-guarded
	// diagonal fast path when the per-region occurrence sequences
	// are positionally aligned; fall back to the conservative
	// all-pairs loop otherwise. The guard's correctness is proved
	// invariant-by-invariant: equal lengths, equal per-index
	// StartPos deltas, equal per-index EndPos deltas.
	for i, ridA := range regionOrder {
		for j := i + 1; j < len(regionOrder); j++ {
			ridB := regionOrder[j]
			idxA := byRegion[ridA]
			idxB := byRegion[ridB]
			if regionsArePositionallyAligned(idxA, idxB, annotatedWindows) {
				emitCrossRegionDiagonalMatches(fp, ridA, ridB, idxA, idxB, annotatedWindows, &out)
			} else {
				emitCrossRegionAllPairsMatches(fp, ridA, ridB, idxA, idxB, annotatedWindows, &out)
			}
		}
	}

	return out
}

// regionsArePositionallyAligned reports whether the two sorted
// per-region window-index sequences are position-by-position aligned:
//
//   - len(idxA) == len(idxB),
//   - for every i: windows[idxA[i]].StartPos - windows[idxA[0]].StartPos ==
//     windows[idxB[i]].StartPos - windows[idxB[0]].StartPos,
//   - the analogous equality on EndPos - EndPos deltas.
//
// When this predicate is true, every corresponding (idxA[i], idxB[i])
// pair shares the same offset, and the diagonal spans the entire
// aligned run. The guard does NOT depend on map iteration order, the
// fingerprint bucket ordering, or the developer's working tree.
func regionsArePositionallyAligned(idxA, idxB []int, annotatedWindows []v4AnnotatedWindow) bool {
	if len(idxA) != len(idxB) {
		return false
	}
	if len(idxA) < 1 {
		return true
	}
	baseA := annotatedWindows[idxA[0]]
	baseB := annotatedWindows[idxB[0]]
	deltaStartA := baseA.w.StartPos
	deltaEndA := baseA.w.EndPos
	deltaStartB := baseB.w.StartPos
	deltaEndB := baseB.w.EndPos
	for i := 1; i < len(idxA); i++ {
		wA := annotatedWindows[idxA[i]]
		wB := annotatedWindows[idxB[i]]
		if wA.w.StartPos-deltaStartA != wB.w.StartPos-deltaStartB {
			return false
		}
		if wA.w.EndPos-deltaEndA != wB.w.EndPos-deltaEndB {
			return false
		}
	}
	return true
}

// emitWithinRegionMatches emits one seed match per non-overlapping
// (i, j) pair of windows sharing the same region. The legacy all-pairs
// implementation did the same work but allocated an O(N²) slice;
// this helper emits exactly the kept pairs in a single forward walk.
func emitWithinRegionMatches(
	fp string,
	rid v4SyntaxRegionID,
	idxs []int,
	annotatedWindows []v4AnnotatedWindow,
	out *[]v4RegionSeedMatch,
) {
	for i := 0; i < len(idxs); i++ {
		aIdx := idxs[i]
		a := annotatedWindows[aIdx]
		for j := i + 1; j < len(idxs); j++ {
			bIdx := idxs[j]
			b := annotatedWindows[bIdx]
			// Same-region pairs require disjoint token ranges; this
			// blocks self-overlap and exact-duplicate self-pairs while
			// preserving the within-region RepeatedMultiplicity case.
			if tokenRangesOverlap(a.w, b.w) {
				continue
			}
			left, right := a.w, b.w
			leftRegion, rightRegion := rid, rid
			if a.w.Path > b.w.Path ||
				(a.w.Path == b.w.Path && a.w.StartPos > b.w.StartPos) {
				left, right = b.w, a.w
			}
			*out = append(*out, v4RegionSeedMatch{
				Match: seedMatch{
					SeedFingerprint: fp,
					Left:            left,
					Right:           right,
					Offset:          right.StartPos - left.StartPos,
				},
				LeftRegion:  leftRegion,
				RightRegion: rightRegion,
			})
		}
	}
}

// emitCrossRegionDiagonalMatches emits the diagonal matches
// (region_a[i], region_b[i]) for each i. The diagonal spans every
// window position in both regions and feeds the smallest-offset
// partition per shadow group. This helper MUST only be called after
// `regionsArePositionallyAligned` returns true; the invariant is
// what preserves canonical chain geometry for the dense-sliding
// production case.
func emitCrossRegionDiagonalMatches(
	fp string,
	ridA, ridB v4SyntaxRegionID,
	idxA, idxB []int,
	annotatedWindows []v4AnnotatedWindow,
	out *[]v4RegionSeedMatch,
) {
	n := len(idxA)
	if len(idxB) < n {
		n = len(idxB)
	}
	for k := 0; k < n; k++ {
		wA := annotatedWindows[idxA[k]]
		wB := annotatedWindows[idxB[k]]
		left, right := wA.w, wB.w
		leftRegion, rightRegion := ridA, ridB
		if ridA.Path > ridB.Path ||
			(ridA.Path == ridB.Path && ridA.Ordinal > ridB.Ordinal) {
			left, right = wB.w, wA.w
			leftRegion, rightRegion = ridB, ridA
		}
		*out = append(*out, v4RegionSeedMatch{
			Match: seedMatch{
				SeedFingerprint: fp,
				Left:            left,
				Right:           right,
				Offset:          right.StartPos - left.StartPos,
			},
			LeftRegion:  leftRegion,
			RightRegion: rightRegion,
		})
	}
}

// emitCrossRegionAllPairsMatches is the conservative fallback
// used when the per-region occurrence sequences are not
// positionally aligned. It emits one match per (i, j) pair from
// two distinct regions, applying only the same-region-overlap
// rejection. Memory is O(N_left * N_right) per region pair; the
// guard that triggers it is documented at the call site.
func emitCrossRegionAllPairsMatches(
	fp string,
	ridA, ridB v4SyntaxRegionID,
	idxA, idxB []int,
	annotatedWindows []v4AnnotatedWindow,
	out *[]v4RegionSeedMatch,
) {
	for i := 0; i < len(idxA); i++ {
		wA := annotatedWindows[idxA[i]]
		for j := 0; j < len(idxB); j++ {
			wB := annotatedWindows[idxB[j]]
			left, right := wA.w, wB.w
			leftRegion, rightRegion := ridA, ridB
			if ridA.Path > ridB.Path ||
				(ridA.Path == ridB.Path && ridA.Ordinal > ridB.Ordinal) {
				left, right = wB.w, wA.w
				leftRegion, rightRegion = ridB, ridA
			}
			*out = append(*out, v4RegionSeedMatch{
				Match: seedMatch{
					SeedFingerprint: fp,
					Left:            left,
					Right:           right,
					Offset:          right.StartPos - left.StartPos,
				},
				LeftRegion:  leftRegion,
				RightRegion: rightRegion,
			})
		}
	}
}

// sortRegionAnnotatedMatches sorts matches inside a chain-partition
// deterministically. The comparator uses:
//
//	Left.Path, Left.StartPos, Left.EndPos,
//	Right.Path, Right.StartPos, Right.EndPos,
//	original seed fingerprint (content identity)
//
// Each projected value is compared with strict ordering so distinct
// matches never compare equal.
func sortRegionAnnotatedMatches(group []v4RegionSeedMatch) {
	sort.Slice(group, func(i, j int) bool {
		if group[i].Match.Left.Path != group[j].Match.Left.Path {
			return group[i].Match.Left.Path < group[j].Match.Left.Path
		}
		if group[i].Match.Left.StartPos != group[j].Match.Left.StartPos {
			return group[i].Match.Left.StartPos < group[j].Match.Left.StartPos
		}
		if group[i].Match.Left.EndPos != group[j].Match.Left.EndPos {
			return group[i].Match.Left.EndPos < group[j].Match.Left.EndPos
		}
		if group[i].Match.Right.Path != group[j].Match.Right.Path {
			return group[i].Match.Right.Path < group[j].Match.Right.Path
		}
		if group[i].Match.Right.StartPos != group[j].Match.Right.StartPos {
			return group[i].Match.Right.StartPos < group[j].Match.Right.StartPos
		}
		if group[i].Match.Right.EndPos != group[j].Match.Right.EndPos {
			return group[i].Match.Right.EndPos < group[j].Match.Right.EndPos
		}
		return group[i].Match.SeedFingerprint < group[j].Match.SeedFingerprint
	})
}

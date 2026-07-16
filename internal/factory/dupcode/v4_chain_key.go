// Package dupcode provides V4 chain-key construction.
//
// V4 chains are partitioned by an explicit chain-pair key derived from
// the chain's actual left and right ranges:
//
//   - LeftRegion   : v4SyntaxRegionID of the left side
//   - RightRegion  : v4SyntaxRegionID of the right side
//   - Offset       : constant token offset between left and right windows
//
// The chain-pair key replaces the legacy PathSet-based grouping, which
// conflated paths and ranges and used the presence of a `|` delimiter
// as a within-file heuristic.
//
// Content identity is kept SEPARATE from region metadata. The original
// seed fingerprint is preserved on the match so N-way merging can
// derive a content identity that is independent of the file pair,
// region ordinals, and pair orientation. Region paths and ordinals are
// ATTACHED to the match but never encoded into the seed fingerprint.
//
// Orientation policy: chains are canonicalised so the region with the
// lexically smaller path occupies the Left slot; when both sides share
// a path, the smaller ordinal occupies the Left slot. Both paths AND
// both ranges are swapped together; paths are never sorted
// independently of their ranges.
package dupcode

import (
	"sort"
)

// v4ChainPairKey is the explicit partition key for V4 chains.
//
// Orientation is canonical: when LeftRegion.Path == RightRegion.Path,
// LeftRegion.Ordinal <= RightRegion.Ordinal. Otherwise LeftRegion.Path
// <= RightRegion.Path. Both paths and their ranges are swapped in lock
// step; paths are NEVER sorted independently of their ranges.
//
// The chain-pair key does NOT contain the content fingerprint. The
// content fingerprint is attached to each underlying seed match and
// contributes to maximal-chain content identity after the chain is
// finalized.
type v4ChainPairKey struct {
	LeftRegion  v4SyntaxRegionID
	RightRegion v4SyntaxRegionID
	Offset      int
}

// String returns a deterministic representation for logging.
func (k v4ChainPairKey) String() string {
	return k.LeftRegion.String() + "/" + k.RightRegion.String() + "@" + itoa(k.Offset)
}

// canonicalChainPairKey orients (left, right) so the lexically smaller
// region occupies the Left slot. When both sides share a path, the
// smaller ordinal is placed on the Left side. Ranges and paths are
// moved together; the path order is never computed without swapping
// the ranges in lock step.
func canonicalChainPairKey(leftRegion, rightRegion v4SyntaxRegionID, offset int) v4ChainPairKey {
	if leftRegion.Path < rightRegion.Path ||
		(leftRegion.Path == rightRegion.Path && leftRegion.Ordinal <= rightRegion.Ordinal) {
		return v4ChainPairKey{LeftRegion: leftRegion, RightRegion: rightRegion, Offset: offset}
	}
	return v4ChainPairKey{
		LeftRegion:  rightRegion,
		RightRegion: leftRegion,
		Offset:      -offset,
	}
}

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

// v4BuildRegionBoundedChainInputs walks every fingerprint bucket,
// generates region-annotated matches for every bucket, combines them,
// partitions them by the structured chain-pair key, and sorts matches
// inside each partition. It returns the partitions plus a flat slice
// of region-annotated matches for downstream consumers.
//
// Aggregating ALL fingerprint buckets before partitioning (P0
// correction 2) ensures that adjacent windows with different seed
// fingerprints but the same region pair can extend into one maximal
// chain.
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

// generateRegionAnnotatedMatches walks every window pair for a single
// fingerprint and emits region-annotated matches that lie inside a
// region pair.
//
// Windows whose owning region is empty (Path=="") are rejected so that
// chains cannot start inside the package declaration or in inter-function
// gaps. Same-region non-overlapping windows are allowed so within-file
// multiplicity (RepeatedMultiplicity) remains detectable.
func generateRegionAnnotatedMatches(fp string, windows []rawWindow, analysisByPath map[string]*v4FileAnalysis) []v4RegionSeedMatch {
	if len(windows) < 2 {
		return nil
	}

	type annotated struct {
		w      rawWindow
		region v4SyntaxRegionID
	}
	annotatedWindows := make([]annotated, len(windows))
	for i, w := range windows {
		a, ok := analysisByPath[w.Path]
		if !ok {
			annotatedWindows[i] = annotated{w: w}
			continue
		}
		rid, ok := a.windowFitsRegion(w.StartPos, w.EndPos)
		if !ok {
			annotatedWindows[i] = annotated{w: w}
			continue
		}
		annotatedWindows[i] = annotated{w: w, region: rid}
	}

	var out []v4RegionSeedMatch
	for i := 0; i < len(annotatedWindows); i++ {
		for j := i + 1; j < len(annotatedWindows); j++ {
			a := annotatedWindows[i]
			b := annotatedWindows[j]
			if a.region.Path == "" || b.region.Path == "" {
				continue
			}
			// Same-region pairs require disjoint token ranges. This
			// blocks self-overlap and exact-duplicate self-pairs
			// while still allowing distinct within-region clones
			// (RepeatedMultiplicity B1 vs B2 case).
			if a.region == b.region && tokenRangesOverlap(a.w, b.w) {
				continue
			}
			left, right := a.w, b.w
			leftRegion, rightRegion := a.region, b.region
			if a.region.Path > b.region.Path ||
				(a.region.Path == b.region.Path && a.region.Ordinal > b.region.Ordinal) {
				left, right = b.w, a.w
				leftRegion, rightRegion = b.region, a.region
			}
			out = append(out, v4RegionSeedMatch{
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
	return out
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

// sortChainPairKeys sorts a slice of chain-pair keys by the total
// order required by P0 correction 3.
func sortChainPairKeys(keys []v4ChainPairKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].LeftRegion.Path != keys[j].LeftRegion.Path {
			return keys[i].LeftRegion.Path < keys[j].LeftRegion.Path
		}
		if keys[i].LeftRegion.Ordinal != keys[j].LeftRegion.Ordinal {
			return keys[i].LeftRegion.Ordinal < keys[j].LeftRegion.Ordinal
		}
		if keys[i].RightRegion.Path != keys[j].RightRegion.Path {
			return keys[i].RightRegion.Path < keys[j].RightRegion.Path
		}
		if keys[i].RightRegion.Ordinal != keys[j].RightRegion.Ordinal {
			return keys[i].RightRegion.Ordinal < keys[j].RightRegion.Ordinal
		}
		return keys[i].Offset < keys[j].Offset
	})
}

// v4RegionBoundedChains builds maximal clone chains from a window map
// using region-bounded chain construction.
//
// All fingerprint buckets contribute (P0 correction 2). Matches are
// partitioned by the structured chain-pair key (P0 correction 1).
// Chain extension runs per partition in deterministic order (P0
// correction 3). Chains extending across compatible regions within the
// same region pair and offset form one maximal chain.
func v4RegionBoundedChains(windowMap map[string][]rawWindow, analysisByPath map[string]*v4FileAnalysis) []cloneChain {
	if len(windowMap) == 0 {
		return nil
	}
	_, partitions := v4BuildRegionBoundedChainInputs(windowMap, analysisByPath)
	if len(partitions) == 0 {
		return nil
	}

	var keys []v4ChainPairKey
	for k := range partitions {
		keys = append(keys, k)
	}
	sortChainPairKeys(keys)

	var allChains []cloneChain
	for _, key := range keys {
		group := partitions[key]
		for _, chain := range extendRegionBoundedChain(group) {
			allChains = append(allChains, chain)
		}
	}
	return allChains
}

// extendRegionBoundedChain walks one sorted chain-partition and emits
// maximal contiguous chains.
//
// Two region-annotated matches are chainable when they belong to the
// same partition (region pair, offset) AND the next match's Left and
// Right StartPos each lie within one token of the previous match's
// Left and Right EndPos. This is the same adjacency rule used by the
// legacy non-region chain constructor, but bounded to the partition
// so chains cannot cross region boundaries.
func extendRegionBoundedChain(group []v4RegionSeedMatch) []cloneChain {
	if len(group) == 0 {
		return nil
	}

	var chains []cloneChain
	var current []seedMatch

	flush := func() {
		if len(current) > 0 {
			if c := v4FinalizeChain(current); c != nil {
				chains = append(chains, *c)
			}
			current = nil
		}
	}

	for _, m := range group {
		if len(current) == 0 {
			current = append(current, m.Match)
			continue
		}
		prev := current[len(current)-1]
		canChain := m.Match.Left.StartPos <= prev.Left.EndPos+1 &&
			m.Match.Right.StartPos <= prev.Right.EndPos+1
		if canChain {
			current = append(current, m.Match)
		} else {
			flush()
			current = append(current, m.Match)
		}
	}
	flush()
	return chains
}

// tokenRangesOverlap reports whether two raw-window token ranges share
// at least one token position. Adjacent ranges (start <= end+1) do
// NOT overlap; the predicate is the strict inclusive overlap used by
// chain containment and self-match filtering.
//
// This is the canonical symmetric overlap predicate. Every other
// overlap check in the V4 pipeline MUST route through it; duplicating
// asymmetric expressions is forbidden.
func tokenRangesOverlap(left, right rawWindow) bool {
	return max(left.StartPos, right.StartPos) <= min(left.EndPos, right.EndPos)
}

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

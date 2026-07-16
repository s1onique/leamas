// Package dupcode provides V4 shadow-suppression and chain filtering logic.
//
// V4 emits one maximal finding per physical clone relation. After the
// region-bounded chain construction produces a set of chains for each
// (region-pair, offset) key, shadow suppression removes chains whose
// positional extents are entirely contained inside another chain with
// the same region-pair key. This is the structural rule that removes
// shifted sliding-window variants of the same underlying clone body.
//
// Within-file chains whose LeftRange and RightRange overlap in file
// position are also removed: they are sub-window self-matches produced
// by sliding windows over a single function body. Real within-file
// multiplicity chains (RepeatedMultiplicity B1 vs B2) have disjoint
// LeftRange and RightRange and survive.
//
// All overlap checks route through the canonical symmetric
// tokenRangesOverlap predicate. Direct asymmetric overlap expressions
// such as left.EndPos >= right.StartPos are forbidden.
//
// Strict containment is defined consistently in both chainContainsRange
// and chainRangeRelationBetween: the outer chain must contain the inner
// chain on BOTH sides AND at least one corresponding range must be
// strictly larger.
//
// When ranges are equal, exactly one survivor is retained by a total
// deterministic tie-break. The tie-break includes the canonical
// chain-pair key, the chain's offset, TokenSpan, LineSpan, content
// hash, and the chain's original stable input ordinal. Using "first
// after unstable sort" is forbidden.
package dupcode

import (
	"sort"
	"strconv"
	"strings"
)

// v4SuppressShadowChains removes chains whose positional extents are
// entirely contained inside another chain sharing the same
// chain-pair key.
//
// Within-file chains whose LeftRange and RightRange overlap in file
// position are also removed: they are sub-window self-matches produced
// by sliding windows over a single function body, not real clone
// relations.
//
// The single-chain case is NOT short-circuited. A lone chain may still
// be an overlapping self-match and must run structural validation.
//
// analysesByPath maps file paths to their v4FileAnalysis; when non-nil
// it is used to compute region-bounded chain keys. Region identity is
// REQUIRED to prevent two independent function bodies in the same file
// pair from being treated as a single chain pair.
func v4SuppressShadowChains(chains []cloneChain) []cloneChain {
	return v4SuppressShadowChainsRegionBounded(chains, nil)
}

// v4SuppressShadowChainsRegionBounded is the region-aware form of
// v4SuppressShadowChains. When analysesByPath is non-nil, the chain
// group key includes both region ordinals in addition to the file pair,
// so chains from different regions of the same file pair are
// partitioned into separate shadow groups.
func v4SuppressShadowChainsRegionBounded(chains []cloneChain, analysesByPath map[string]*v4FileAnalysis) []cloneChain {
	if len(chains) == 0 {
		return nil
	}

	type ref struct {
		idx          int
		chain        cloneChain
		shadow       bool
		key          v4ShadowGroupKey
		inputOrdinal int
	}
	groups := make(map[v4ShadowGroupKey][]*ref)
	var groupOrder []v4ShadowGroupKey
	for i := range chains {
		key := chainPairKeyForChain(chains[i], analysesByPath)
		if _, ok := groups[key]; !ok {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], &ref{idx: i, chain: chains[i], key: key, inputOrdinal: i})
	}
	sort.Slice(groupOrder, func(i, j int) bool {
		return compareV4ShadowGroupKeys(groupOrder[i], groupOrder[j]) < 0
	})

	for _, key := range groupOrder {
		members := groups[key]
		sort.Slice(members, func(i, j int) bool {
			if members[i].chain.LeftRange.StartPos != members[j].chain.LeftRange.StartPos {
				return members[i].chain.LeftRange.StartPos < members[j].chain.LeftRange.StartPos
			}
			if members[i].chain.LeftRange.EndPos != members[j].chain.LeftRange.EndPos {
				return members[i].chain.LeftRange.EndPos < members[j].chain.LeftRange.EndPos
			}
			return members[i].inputOrdinal < members[j].inputOrdinal
		})

		// Equal-range chains: keep only one survivor by the
		// deterministic tie-break below.
		n := len(members)
		for i := 0; i < n; i++ {
			if members[i].shadow {
				continue
			}
			for j := i + 1; j < n; j++ {
				if members[j].shadow {
					continue
				}
				rel := chainRangeRelationBetween(members[i].chain, members[j].chain)
				switch rel {
				case chainRangeEqual:
					if !deterministicEqualTieBreak(members[i].chain, members[j].chain, members[i].inputOrdinal, members[j].inputOrdinal) {
						members[i].shadow = true
					} else {
						members[j].shadow = true
					}
				case chainRangeStrictlyContains:
					members[j].shadow = true
				}
			}
		}

		// Lone within-file overlap self-match filter. A single chain
		// may still be an overlapping self-match.
		withinFile := isWithinFileChain(members[0].chain)
		if withinFile {
			for _, m := range members {
				if m.shadow {
					continue
				}
				if tokenRangesOverlap(
					rawWindow{StartPos: m.chain.LeftRange.StartPos, EndPos: m.chain.LeftRange.EndPos},
					rawWindow{StartPos: m.chain.RightRange.StartPos, EndPos: m.chain.RightRange.EndPos}) {
					m.shadow = true
				}
			}
		}

		// Cross-chain containment (after the equal-range pass so we
		// don't redundantly mark equal-pair members).
		for i := 0; i < n; i++ {
			if members[i].shadow {
				continue
			}
			for j := 0; j < n; j++ {
				if i == j || members[j].shadow {
					continue
				}
				if chainContainsRange(members[j].chain, members[i].chain) {
					members[i].shadow = true
					break
				}
			}
		}
	}

	survivors := make([]cloneChain, 0, len(chains))
	for i := range chains {
		key := chainPairKeyForChain(chains[i], analysesByPath)
		for _, member := range groups[key] {
			if member.idx == i && !member.shadow {
				survivors = append(survivors, chains[i])
				break
			}
		}
	}
	return survivors
}

// chainPairKeyForChain returns the deterministic group key for one
// chain. The key is derived from the chain's actual left and right
// ranges, NOT from a PathSet or a delimiter heuristic.
//
// Orientation is canonical: the lexically smaller path occupies the
// Left slot; when both sides share a path, the smaller range
// (LeftRange.StartPos <= RightRange.StartPos) occupies the Left slot.
// Both paths AND their ranges are swapped together; paths are NEVER
// sorted independently of their ranges.
//
// The key intentionally does NOT include the constant token offset.
// Shifted sliding-window variants of the same underlying clone body
// share the same file pair but differ in offset; they must collide
// on this key so the within-group containment rule can suppress them.
//
// When analysesByPath is non-nil, the key also includes each side's
// region ordinal. Two chains from independent function bodies in the
// same file pair thus fall into separate shadow groups and cannot
// suppress each other.
// chainPaths extracts the canonical (leftPath, rightPath) pair for a
// chain. When the chain's matches disagree on a path, the first
// match's left/right paths are returned, then canonicalised so the
// lexically smaller path occupies the Left slot. When both sides
// share a path, the smaller range occupies the Left slot. Both paths
// AND their ranges are moved together; the path order is never
// computed without swapping the ranges in lock step.
func chainPaths(c cloneChain) (string, string) {
	if len(c.Matches) == 0 {
		return "", ""
	}
	leftPath, rightPath := c.Matches[0].Left.Path, c.Matches[0].Right.Path
	leftStart, rightStart := c.Matches[0].Left.StartPos, c.Matches[0].Right.StartPos
	if leftPath > rightPath || (leftPath == rightPath && leftStart > rightStart) {
		leftPath, rightPath = rightPath, leftPath
	}
	return leftPath, rightPath
}

// isWithinFileChain reports whether the chain's left and right sides
// refer to the same file. This is the direct, structural test for
// within-file chains; it does NOT inspect any encoded PathSet.
func isWithinFileChain(c cloneChain) bool {
	leftPath, rightPath := chainPaths(c)
	return leftPath == rightPath
}

// chainRangeRelation reports the positional relationship between two
// chains on both sides. The classification is used to distinguish
// strict containment from exact duplicates from incomparable overlap.
//
// Strict containment: outerChain contains innerChain on BOTH sides
// AND outerChain spans are strictly larger on at least one side.
//
// Exact duplicate: outerChain and innerChain share identical ranges
// on BOTH sides. The duplicate is dropped, retaining exactly one
// deterministic survivor.
//
// Incomparable overlap: chains overlap on at least one side but do
// not fully contain each other; both chains survive pending later
// region logic.
type chainRangeRelation uint8

const (
	chainRangeUnrelated chainRangeRelation = iota
	chainRangeStrictlyContains
	chainRangeEqual
	chainRangeIncomparableOverlap
)

// chainContainsRange reports whether outerChain's positional extents
// entirely contain innerChain's positional extents on BOTH sides, with
// at least one side strictly larger. This is the strict-containment
// predicate used to mark an inner chain as a shadow of an outer chain.
//
// Equal-range chains do NOT satisfy chainContainsRange; equal chains
// fall under chainRangeEqual and resolve through the deterministic
// tie-break rule.
func chainContainsRange(outerChain, innerChain cloneChain) bool {
	if innerChain.LeftRange.StartPos < outerChain.LeftRange.StartPos {
		return false
	}
	if innerChain.LeftRange.EndPos > outerChain.LeftRange.EndPos {
		return false
	}
	if innerChain.RightRange.StartPos < outerChain.RightRange.StartPos {
		return false
	}
	if innerChain.RightRange.EndPos > outerChain.RightRange.EndPos {
		return false
	}
	return innerChain.LeftRange.StartPos != outerChain.LeftRange.StartPos ||
		innerChain.LeftRange.EndPos != outerChain.LeftRange.EndPos ||
		innerChain.RightRange.StartPos != outerChain.RightRange.StartPos ||
		innerChain.RightRange.EndPos != outerChain.RightRange.EndPos
}

// chainRangeRelationBetween classifies the positional relationship
// between outer and inner chains.
//
//   - chainRangeEqual: identical ranges on both sides. The duplicate is
//     dropped, retaining exactly one deterministic survivor.
//   - chainRangeStrictlyContains: outerChain strictly contains innerChain
//     on BOTH sides. innerChain is a shadow and is dropped.
//   - chainRangeUnrelated: outerChain does not contain innerChain on
//     either side. innerChain survives.
//   - chainRangeIncomparableOverlap: chains overlap on at least one
//     side but do not fully contain each other. Both chains survive
//     pending later region logic.
func chainRangeRelationBetween(outerChain, innerChain cloneChain) chainRangeRelation {
	leftContainsStart := innerChain.LeftRange.StartPos >= outerChain.LeftRange.StartPos
	leftContainsEnd := innerChain.LeftRange.EndPos <= outerChain.LeftRange.EndPos
	rightContainsStart := innerChain.RightRange.StartPos >= outerChain.RightRange.StartPos
	rightContainsEnd := innerChain.RightRange.EndPos <= outerChain.RightRange.EndPos

	strictLeft := innerChain.LeftRange.StartPos > outerChain.LeftRange.StartPos ||
		innerChain.LeftRange.EndPos < outerChain.LeftRange.EndPos
	strictRight := innerChain.RightRange.StartPos > outerChain.RightRange.StartPos ||
		innerChain.RightRange.EndPos < outerChain.RightRange.EndPos

	leftEqual := innerChain.LeftRange.StartPos == outerChain.LeftRange.StartPos &&
		innerChain.LeftRange.EndPos == outerChain.LeftRange.EndPos
	rightEqual := innerChain.RightRange.StartPos == outerChain.RightRange.StartPos &&
		innerChain.RightRange.EndPos == outerChain.RightRange.EndPos

	if leftEqual && rightEqual {
		return chainRangeEqual
	}

	if leftContainsStart && leftContainsEnd && rightContainsStart && rightContainsEnd {
		// Strict containment requires STRICT larger on at least one
		// side. Equal-side / strictly-contained-other-side chains
		// are still strict containment.
		if strictLeft || strictRight {
			return chainRangeStrictlyContains
		}
		return chainRangeUnrelated
	}

	leftIntersect := tokenRangesOverlap(
		rawWindow{StartPos: outerChain.LeftRange.StartPos, EndPos: outerChain.LeftRange.EndPos},
		rawWindow{StartPos: innerChain.LeftRange.StartPos, EndPos: innerChain.LeftRange.EndPos})
	rightIntersect := tokenRangesOverlap(
		rawWindow{StartPos: outerChain.RightRange.StartPos, EndPos: outerChain.RightRange.EndPos},
		rawWindow{StartPos: innerChain.RightRange.StartPos, EndPos: innerChain.RightRange.EndPos})
	if leftIntersect || rightIntersect {
		return chainRangeIncomparableOverlap
	}

	return chainRangeUnrelated
}

// chainContentHash returns the content hash used by the equal-range
// tie-break. It is computed lazily.
func chainContentHash(c cloneChain) string {
	if c.ContentHash != "" {
		return c.ContentHash
	}
	return computeContentHash(c.Matches)
}

// deterministicEqualTieBreak returns true if chain `a` should survive
// over chain `b` when both are equal-range duplicates. The comparator
// is total and never produces equal results for distinct chains.
//
// Tie-break order:
//
//  1. canonical region-pair key (path, ordinal, path, ordinal) — already
//     equal in callers since they share a shadow-suppression group key
//  2. offset (smaller wins)
//  3. TokenSpan (smaller wins)
//  4. LineSpan (smaller wins)
//  5. content hash (lex smaller wins)
//  6. original input ordinal (smaller wins)
//  7. canonical chain string (last resort)
func deterministicEqualTieBreak(a, b cloneChain, aOrdinal, bOrdinal int) bool {
	if a.Offset != b.Offset {
		return a.Offset < b.Offset
	}
	if a.TokenSpan != b.TokenSpan {
		return a.TokenSpan < b.TokenSpan
	}
	if a.LineSpan != b.LineSpan {
		return a.LineSpan < b.LineSpan
	}
	ha := chainContentHash(a)
	hb := chainContentHash(b)
	if ha != hb {
		return ha < hb
	}
	if aOrdinal != bOrdinal {
		return aOrdinal < bOrdinal
	}
	return canonicalChainString(a) < canonicalChainString(b)
}

// canonicalChainString returns a deterministic canonical string for
// the chain. It is used only when every other tie-break value is
// equal so the result is deterministic.
func canonicalChainString(c cloneChain) string {
	parts := []string{
		c.PathSet,
		"l=" + strconv.Itoa(c.LeftRange.StartPos) + "-" + strconv.Itoa(c.LeftRange.EndPos),
		"r=" + strconv.Itoa(c.RightRange.StartPos) + "-" + strconv.Itoa(c.RightRange.EndPos),
	}
	return strings.Join(parts, "|")
}

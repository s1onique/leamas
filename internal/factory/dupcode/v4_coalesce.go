// Package dupcode provides V4 chain construction and production merge
// seam.
//
// v4CoalesceFindings is the V4 algorithm for finding maximal clones.
// The chain construction runs over a fingerprint-bucketed window map
// and emits one maximal finding per physical clone relation.
//
// The production merge seam (v4InternalFindingsFromChains) preserves
// this geometry through shadow suppression, N-way merge, dedup,
// occurrence sorting, and finding ordering. The public
// v4FindingsFromChains path projects the internal findings into the
// existing public coalesced type so callers and baselines see no
// schema change.
package dupcode

import (
	"sort"
	"strings"
)

// v4FinalizeChain computes the exact maximal ranges and content hash from a chain of matches.
func v4FinalizeChain(matches []seedMatch) *cloneChain {
	if len(matches) == 0 {
		return nil
	}

	leftMinStart := matches[0].Left.StartPos
	leftMaxEnd := matches[0].Left.EndPos
	rightMinStart := matches[0].Right.StartPos
	rightMaxEnd := matches[0].Right.EndPos
	maxLeftLine := matches[0].Left.EndLine
	maxRightLine := matches[0].Right.EndLine
	minLeftLine := matches[0].Left.StartLine
	minRightLine := matches[0].Right.StartLine

	for _, m := range matches {
		if m.Left.StartPos < leftMinStart {
			leftMinStart = m.Left.StartPos
		}
		if m.Left.EndPos > leftMaxEnd {
			leftMaxEnd = m.Left.EndPos
		}
		if m.Left.EndLine > maxLeftLine {
			maxLeftLine = m.Left.EndLine
		}
		if m.Left.StartLine < minLeftLine {
			minLeftLine = m.Left.StartLine
		}
		if m.Right.StartPos < rightMinStart {
			rightMinStart = m.Right.StartPos
		}
		if m.Right.EndPos > rightMaxEnd {
			rightMaxEnd = m.Right.EndPos
		}
		if m.Right.EndLine > maxRightLine {
			maxRightLine = m.Right.EndLine
		}
		if m.Right.StartLine < minRightLine {
			minRightLine = m.Right.StartLine
		}
	}

	leftSpan := leftMaxEnd - leftMinStart + 1
	rightSpan := rightMaxEnd - rightMinStart + 1
	if leftSpan != rightSpan {
		return nil
	}

	maxLineSpan := maxLeftLine - minLeftLine + 1
	rightLineSpan := maxRightLine - minRightLine + 1
	if rightLineSpan > maxLineSpan {
		maxLineSpan = rightLineSpan
	}

	pathSet := make(map[string]bool)
	for _, m := range matches {
		pathSet[m.Left.Path] = true
		pathSet[m.Right.Path] = true
	}
	var paths []string
	for p := range pathSet {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	contentHash := computeContentHash(matches)

	return &cloneChain{
		Matches:     matches,
		TokenSpan:   leftSpan,
		LineSpan:    maxLineSpan,
		PathSet:     strings.Join(paths, "|"),
		LeftRange:   tokenRange{StartPos: leftMinStart, EndPos: leftMaxEnd},
		RightRange:  tokenRange{StartPos: rightMinStart, EndPos: rightMaxEnd},
		Offset:      matches[0].Offset,
		ContentHash: contentHash,
	}
}

// v4FingerprintFromChain computes the V4 stable fingerprint from a chain.
func v4FingerprintFromChain(chain cloneChain) string {
	if chain.ContentHash == "" {
		chain.ContentHash = computeContentHash(chain.Matches)
	}
	return v4SeedFingerprint(chain.ContentHash)
}

// v4InternalFindingsFromChains is the production V4 merge seam.
//
// v4InternalFindingsFromChains drops shifted shadow chains, emits one
// internal finding per surviving chain, and merges findings that share
// the same (StableFingerprint, TokenCount) into a single N-way finding
// with deduplicated occurrences.
func v4InternalFindingsFromChains(chains []cloneChain) []v4InternalFinding {
	if len(chains) == 0 {
		return nil
	}

	chains = v4SuppressShadowChains(chains)

	var findings []v4InternalFinding
	for _, chain := range chains {
		occurrences := v4OccurrenceFromChain(chain)
		if len(occurrences) < 2 {
			continue
		}

		stableFP := v4FingerprintFromChain(chain)

		findings = append(findings, v4InternalFinding{
			StableFingerprint: stableFP,
			TokenCount:        chain.TokenSpan,
			LineCount:         chain.LineSpan,
			Occurrences:       occurrences,
		})
	}

	findings = v4MergeFindings(findings)

	sortV4InternalFindings(findings)

	return findings
}

// v4FindingsFromChains projects the production-owned internal findings into
// the existing public coalesced type.
func v4FindingsFromChains(chains []cloneChain) []coalescedFinding {
	if len(chains) == 0 {
		return nil
	}
	findings := v4InternalFindingsFromChains(chains)
	result := make([]coalescedFinding, len(findings))
	for i, f := range findings {
		result[i] = coalescedFinding{
			Fingerprint:       truncateFingerprint(f.StableFingerprint),
			StableFingerprint: f.StableFingerprint,
			SeedFingerprint:   "",
			TokenCount:        f.TokenCount,
			LineCount:         f.LineCount,
			Occurrences:       convertOccurrences(f.Occurrences),
		}
	}
	return result
}

// convertOccurrences converts internal maximalOccurrence to public Occurrence.
func convertOccurrences(occs []maximalOccurrence) []Occurrence {
	result := make([]Occurrence, len(occs))
	for i, o := range occs {
		result[i] = Occurrence{
			Path:      o.Path,
			StartLine: o.StartLine,
			EndLine:   o.EndLine,
		}
	}
	return result
}

// canonicalOccurrenceSetFromV4 returns a deterministic string representation of v4 occurrences.
func canonicalOccurrenceSetFromV4(occs []maximalOccurrence) string {
	sorted := make([]maximalOccurrence, len(occs))
	copy(sorted, occs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		if sorted[i].StartLine != sorted[j].StartLine {
			return sorted[i].StartLine < sorted[j].StartLine
		}
		if sorted[i].EndLine != sorted[j].EndLine {
			return sorted[i].EndLine < sorted[j].EndLine
		}
		if sorted[i].StartPos != sorted[j].StartPos {
			return sorted[i].StartPos < sorted[j].StartPos
		}
		return sorted[i].EndPos < sorted[j].EndPos
	})
	var parts []string
	for _, o := range sorted {
		parts = append(parts, strings.Join([]string{
			o.Path,
			itoa(o.StartLine),
			itoa(o.EndLine),
			itoa(o.StartPos),
			itoa(o.EndPos),
		}, ":"))
	}
	return strings.Join(parts, "|")
}

// v4CoalesceFindings is the V4 algorithm for finding maximal clones.
func v4CoalesceFindings(windowMap map[string][]rawWindow, fingerprintTokens map[string]int) []coalescedFinding {
	var fps []string
	for fp := range windowMap {
		fps = append(fps, fp)
	}
	sort.Strings(fps)

	var allMatches []seedMatch
	for _, fp := range fps {
		matches := buildSeedMatches(fp, windowMap[fp])
		allMatches = append(allMatches, matches...)
	}

	if len(allMatches) == 0 {
		return nil
	}

	chains := v4BuildChainsWithPartitioning(allMatches)

	if len(chains) == 0 {
		return nil
	}

	return v4FindingsFromChains(chains)
}

// v4BuildChainsWithPartitioning constructs maximal clone chains using V4 semantics.
func v4BuildChainsWithPartitioning(matches []seedMatch) []cloneChain {
	if len(matches) == 0 {
		return nil
	}

	chainGroups := groupMatchesByChainKey(matches)

	var keys []chainKey
	for k := range chainGroups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].LeftPath != keys[j].LeftPath {
			return keys[i].LeftPath < keys[j].LeftPath
		}
		if keys[i].RightPath != keys[j].RightPath {
			return keys[i].RightPath < keys[j].RightPath
		}
		return keys[i].Offset < keys[j].Offset
	})

	var allChains []cloneChain
	for _, k := range keys {
		groupMatches := chainGroups[k]
		sort.Slice(groupMatches, func(i, j int) bool {
			if groupMatches[i].Left.StartPos != groupMatches[j].Left.StartPos {
				return groupMatches[i].Left.StartPos < groupMatches[j].Left.StartPos
			}
			if groupMatches[i].Right.StartPos != groupMatches[j].Right.StartPos {
				return groupMatches[i].Right.StartPos < groupMatches[j].Right.StartPos
			}
			return groupMatches[i].SeedFingerprint < groupMatches[j].SeedFingerprint
		})

		var currentChain []seedMatch

		flushChain := func() {
			if len(currentChain) > 0 {
				if chain := v4FinalizeChain(currentChain); chain != nil {
					allChains = append(allChains, *chain)
				}
				currentChain = nil
			}
		}

		for _, m := range groupMatches {
			if len(currentChain) == 0 {
				currentChain = append(currentChain, m)
				continue
			}

			prev := currentChain[len(currentChain)-1]

			canChain := m.Left.StartPos <= prev.Left.EndPos+1 &&
				m.Right.StartPos <= prev.Right.EndPos+1

			if canChain {
				currentChain = append(currentChain, m)
			} else {
				flushChain()
				currentChain = append(currentChain, m)
			}
		}
		flushChain()
	}

	return allChains
}

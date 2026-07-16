// Package dupcode provides duplicate code detection for Go source files.
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

	// Compute exact union ranges for both sides
	leftMinStart := matches[0].Left.StartPos
	leftMaxEnd := matches[0].Left.EndPos
	rightMinStart := matches[0].Right.StartPos
	rightMaxEnd := matches[0].Right.EndPos
	maxLeftLine := matches[0].Left.EndLine
	maxRightLine := matches[0].Right.EndLine

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
		if m.Right.StartPos < rightMinStart {
			rightMinStart = m.Right.StartPos
		}
		if m.Right.EndPos > rightMaxEnd {
			rightMaxEnd = m.Right.EndPos
		}
		if m.Right.EndLine > maxRightLine {
			maxRightLine = m.Right.EndLine
		}
	}

	// Token span should be equal on both sides for valid chain
	leftSpan := leftMaxEnd - leftMinStart + 1
	rightSpan := rightMaxEnd - rightMinStart + 1
	if leftSpan != rightSpan {
		return nil
	}

	// Line span is the max of both sides
	maxLineSpan := maxLeftLine - matches[0].Left.StartLine + 1
	rightLineSpan := maxRightLine - matches[0].Right.StartLine + 1
	if rightLineSpan > maxLineSpan {
		maxLineSpan = rightLineSpan
	}

	// Build path set
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

	// Compute content hash from all constituent seed fingerprints
	// This uniquely identifies the complete maximal clone body
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
	// V4: Fingerprint excludes pathSet to enable N-way merging
	return v4SeedFingerprint(chain.ContentHash)
}

// v4FindingsFromChains converts clone chains to coalesced findings using V4 semantics.
// Uses internal v4Finding type to preserve token positions through N-way merging.
func v4FindingsFromChains(chains []cloneChain) []coalescedFinding {
	if len(chains) == 0 {
		return nil
	}

	var findings []v4Finding
	for _, chain := range chains {
		// Each chain gets its own occurrences (coalesced per file)
		occurrences := v4OccurrenceFromChain(chain)
		if len(occurrences) < 2 {
			continue
		}

		stableFP := v4FingerprintFromChain(chain)

		findings = append(findings, v4Finding{
			StableFingerprint: stableFP,
			TokenCount:        chain.TokenSpan,
			LineCount:         chain.LineSpan,
			Occurrences:       occurrences,
		})
	}

	// Merge by stable fingerprint (content-based) for N-way clones
	findings = v4MergeFindings(findings)

	// Sort deterministically by total order:
	// StableFingerprint -> TokenCount -> LineCount -> canonical occurrence sequence
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].StableFingerprint != findings[j].StableFingerprint {
			return findings[i].StableFingerprint < findings[j].StableFingerprint
		}
		if findings[i].TokenCount != findings[j].TokenCount {
			return findings[i].TokenCount < findings[j].TokenCount
		}
		if findings[i].LineCount != findings[j].LineCount {
			return findings[i].LineCount < findings[j].LineCount
		}
		// Compare canonical occurrence sequences
		return canonicalOccurrenceSetFromV4(findings[i].Occurrences) < canonicalOccurrenceSetFromV4(findings[j].Occurrences)
	})

	// Convert to public type
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

// v4BuildChainsWithPartitioning constructs maximal clone chains using V4 semantics.
func v4BuildChainsWithPartitioning(matches []seedMatch) []cloneChain {
	if len(matches) == 0 {
		return nil
	}

	// Group by chain key (left path, right path, offset) for chaining
	chainGroups := groupMatchesByChainKey(matches)

	// Sort keys to ensure deterministic iteration order
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
		// Sort matches by position for deterministic processing
		sort.Slice(groupMatches, func(i, j int) bool {
			if groupMatches[i].Left.StartPos != groupMatches[j].Left.StartPos {
				return groupMatches[i].Left.StartPos < groupMatches[j].Left.StartPos
			}
			if groupMatches[i].Right.StartPos != groupMatches[j].Right.StartPos {
				return groupMatches[i].Right.StartPos < groupMatches[j].Right.StartPos
			}
			return groupMatches[i].SeedFingerprint < groupMatches[j].SeedFingerprint
		})

		// Chain within this group
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

			// Can chain if both sides advance contiguously
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
	// Step 1: Collect all seed matches across all fingerprints
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

	// Step 2: Chain across all matches (partitioning by file pair + offset)
	chains := v4BuildChainsWithPartitioning(allMatches)

	if len(chains) == 0 {
		return nil
	}

	// Step 3: Convert chains to findings using V4 semantics
	return v4FindingsFromChains(chains)
}

// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"sort"
	"strings"
)

// buildSeedMatches generates seed matches from raw windows for a fingerprint.
// Matches all window pairs across different files AND within the same file.
// This enables detection of repeated occurrences in a single file.
func buildSeedMatches(fp string, windows []rawWindow) []seedMatch {
	if len(windows) < 2 {
		return nil
	}

	// Group windows by path
	pathWindows := make(map[string][]rawWindow)
	for _, w := range windows {
		pathWindows[w.Path] = append(pathWindows[w.Path], w)
	}

	var matches []seedMatch

	// For each path, match windows within same file AND across different files
	for _, fileWindows := range pathWindows {
		// Within-file matches for repeated occurrences - require non-overlapping
		if len(fileWindows) >= 2 {
			// Sort by StartPos first to ensure consistent ordering
			sorted := make([]rawWindow, len(fileWindows))
			copy(sorted, fileWindows)
			sort.Slice(sorted, func(i, j int) bool {
				return sorted[i].StartPos < sorted[j].StartPos
			})

			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					lw, rw := sorted[i], sorted[j]
					// Require non-overlapping for within-file matches
					if lw.EndPos >= rw.StartPos {
						continue
					}
					// Lexicographic ordering for Left/Right
					if lw.Path > rw.Path || (lw.Path == rw.Path && lw.StartPos > rw.StartPos) {
						lw, rw = rw, lw
					}
					matches = append(matches, seedMatch{
						SeedFingerprint: fp,
						Left:            lw,
						Right:           rw,
						Offset:          rw.StartPos - lw.StartPos,
					})
				}
			}
		}
	}

	// Cross-file matches
	var paths []string
	for path := range pathWindows {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			leftPath, rightPath := paths[i], paths[j]
			leftWindows := pathWindows[leftPath]
			rightWindows := pathWindows[rightPath]

			for _, lw := range leftWindows {
				for _, rw := range rightWindows {
					matches = append(matches, seedMatch{
						SeedFingerprint: fp,
						Left:            lw,
						Right:           rw,
						Offset:          rw.StartPos - lw.StartPos,
					})
				}
			}
		}
	}

	return matches
}

// groupMatchesByChainKey partitions matches by chain key for chaining.
// V4: Uses (LeftPath, RightPath, Offset) as the partition key, NOT SeedFingerprint.
// This allows consecutive sliding windows with different fingerprints to chain together.
func groupMatchesByChainKey(matches []seedMatch) map[chainKey][]seedMatch {
	groups := make(map[chainKey][]seedMatch)
	for _, m := range matches {
		key := chainKey{
			LeftPath:  m.Left.Path,
			RightPath: m.Right.Path,
			Offset:    m.Offset,
		}
		groups[key] = append(groups[key], m)
	}
	return groups
}

// finalizeChain computes the exact maximal ranges from a chain of matches.
func finalizeChain(matches []seedMatch) *cloneChain {
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

	return &cloneChain{
		Matches:    matches,
		TokenSpan:  leftSpan,
		LineSpan:   maxLineSpan,
		PathSet:    strings.Join(paths, "|"),
		LeftRange:  tokenRange{StartPos: leftMinStart, EndPos: leftMaxEnd},
		RightRange: tokenRange{StartPos: rightMinStart, EndPos: rightMaxEnd},
		Offset:     matches[0].Offset,
	}
}

// buildChains constructs maximal clone chains from seed matches.
func buildChains(matches []seedMatch) []cloneChain {
	if len(matches) == 0 {
		return nil
	}

	// Sort by (left path, left start, right path, right start) for deterministic processing
	sorted := make([]seedMatch, len(matches))
	copy(sorted, matches)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Left.Path != sorted[j].Left.Path {
			return sorted[i].Left.Path < sorted[j].Left.Path
		}
		if sorted[i].Left.StartPos != sorted[j].Left.StartPos {
			return sorted[i].Left.StartPos < sorted[j].Left.StartPos
		}
		if sorted[i].Right.Path != sorted[j].Right.Path {
			return sorted[i].Right.Path < sorted[j].Right.Path
		}
		return sorted[i].Right.StartPos < sorted[j].Right.StartPos
	})

	var chains []cloneChain
	var currentChain []seedMatch

	flushChain := func() {
		if len(currentChain) > 0 {
			if chain := finalizeChain(currentChain); chain != nil {
				chains = append(chains, *chain)
			}
			currentChain = nil
		}
	}

	for _, m := range sorted {
		if len(currentChain) == 0 {
			currentChain = append(currentChain, m)
			continue
		}

		prev := currentChain[len(currentChain)-1]

		// Can chain if:
		// 1. Same file pair
		// 2. Same offset
		// 3. Both sides advance contiguously
		canChain := prev.Left.Path == m.Left.Path &&
			prev.Right.Path == m.Right.Path &&
			prev.Offset == m.Offset &&
			m.Left.StartPos <= prev.Left.EndPos+1 &&
			m.Right.StartPos <= prev.Right.EndPos+1

		if canChain {
			currentChain = append(currentChain, m)
		} else {
			flushChain()
			currentChain = append(currentChain, m)
		}
	}
	flushChain()

	return chains
}

// buildChainsWithPartitioning constructs maximal clone chains from seed matches.
func buildChainsWithPartitioning(matches []seedMatch) []cloneChain {
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
		// Total order: LeftPath → RightPath → Offset
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
		// Sort matches by (left start, right start) for deterministic processing
		sort.Slice(groupMatches, func(i, j int) bool {
			if groupMatches[i].Left.StartPos != groupMatches[j].Left.StartPos {
				return groupMatches[i].Left.StartPos < groupMatches[j].Left.StartPos
			}
			return groupMatches[i].Right.StartPos < groupMatches[j].Right.StartPos
		})

		// Chain within this group
		var currentChain []seedMatch

		flushChain := func() {
			if len(currentChain) > 0 {
				if chain := finalizeChain(currentChain); chain != nil {
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

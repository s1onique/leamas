// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"sort"
	"strings"
)

// computeMaxTokenSpan computes the UNION token span across all windows.
// Returns maxEndPos - minStartPos + 1.
func computeMaxTokenSpan(windows []rawWindow) int {
	if len(windows) == 0 {
		return 0
	}
	minStart := windows[0].StartPos
	maxEnd := windows[0].EndPos
	for _, w := range windows {
		if w.StartPos < minStart {
			minStart = w.StartPos
		}
		if w.EndPos > maxEnd {
			maxEnd = w.EndPos
		}
	}
	return maxEnd - minStart + 1
}

// coalesceWindows merges overlapping/adjacent windows per file.
func coalesceWindows(windows []rawWindow) []rawWindow {
	if len(windows) == 0 {
		return nil
	}
	// Group by file
	fileGroups := make(map[string][]rawWindow)
	for _, w := range windows {
		fileGroups[w.Path] = append(fileGroups[w.Path], w)
	}
	var result []rawWindow
	for _, fileWins := range fileGroups {
		if len(fileWins) == 0 {
			continue
		}
		sort.Slice(fileWins, func(i, j int) bool {
			if fileWins[i].StartLine != fileWins[j].StartLine {
				return fileWins[i].StartLine < fileWins[j].StartLine
			}
			return fileWins[i].StartPos < fileWins[j].StartPos
		})
		current := fileWins[0]
		for i := 1; i < len(fileWins); i++ {
			w := fileWins[i]
			if w.StartLine <= current.EndLine+1 {
				if w.StartLine < current.StartLine {
					current.StartLine = w.StartLine
					current.StartPos = w.StartPos
				}
				if w.EndLine > current.EndLine {
					current.EndLine = w.EndLine
					current.EndPos = w.EndPos
				}
			} else {
				result = append(result, current)
				current = w
			}
		}
		result = append(result, current)
	}
	return result
}

// coalesceOccurrences merges overlapping/adjacent occurrence ranges.
func coalesceOccurrences(occs []Occurrence) []Occurrence {
	if len(occs) == 0 {
		return nil
	}
	sort.Slice(occs, func(i, j int) bool {
		return occs[i].StartLine < occs[j].StartLine
	})
	var result []Occurrence
	current := occs[0]
	for i := 1; i < len(occs); i++ {
		occ := occs[i]
		if occ.StartLine <= current.EndLine+1 {
			if occ.EndLine > current.EndLine {
				current.EndLine = occ.EndLine
			}
		} else {
			result = append(result, current)
			current = occ
		}
	}
	result = append(result, current)
	return result
}

// canonicalOccurrenceSet returns a deterministic string representation.
func canonicalOccurrenceSet(occs []Occurrence) string {
	sorted := make([]Occurrence, len(occs))
	copy(sorted, occs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path != sorted[j].Path {
			return sorted[i].Path < sorted[j].Path
		}
		return sorted[i].StartLine < sorted[j].StartLine
	})
	var parts []string
	for _, o := range sorted {
		parts = append(parts, fmt.Sprintf("%s:%d-%d", o.Path, o.StartLine, o.EndLine))
	}
	return strings.Join(parts, "|")
}

// compareOccurrences compares two occurrences for sorting.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func compareOccurrences(a, b Occurrence) int {
	if a.Path != b.Path {
		if a.Path < b.Path {
			return -1
		}
		return 1
	}
	if a.StartLine != b.StartLine {
		if a.StartLine < b.StartLine {
			return -1
		}
		return 1
	}
	if a.EndLine != b.EndLine {
		if a.EndLine < b.EndLine {
			return -1
		}
		return 1
	}
	return 0
}

// mergeOccurrences merges two occurrence slices, removes duplicates, and coalesces overlaps.
func mergeOccurrences(a, b []Occurrence) []Occurrence {
	seen := make(map[string]bool)
	var combined []Occurrence
	for _, o := range a {
		key := fmt.Sprintf("%s:%d-%d", o.Path, o.StartLine, o.EndLine)
		if !seen[key] {
			seen[key] = true
			combined = append(combined, o)
		}
	}
	for _, o := range b {
		key := fmt.Sprintf("%s:%d-%d", o.Path, o.StartLine, o.EndLine)
		if !seen[key] {
			seen[key] = true
			combined = append(combined, o)
		}
	}
	// Coalesce overlapping occurrences per file
	fileGroups := make(map[string][]Occurrence)
	for _, o := range combined {
		fileGroups[o.Path] = append(fileGroups[o.Path], o)
	}
	// Sort file paths for deterministic iteration
	var paths []string
	for p := range fileGroups {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var result []Occurrence
	for _, path := range paths {
		occs := fileGroups[path]
		sort.Slice(occs, func(i, j int) bool {
			return occs[i].StartLine < occs[j].StartLine
		})
		if len(occs) == 0 {
			continue
		}
		current := occs[0]
		for i := 1; i < len(occs); i++ {
			o := occs[i]
			if o.StartLine <= current.EndLine+1 {
				if o.EndLine > current.EndLine {
					current.EndLine = o.EndLine
				}
			} else {
				result = append(result, current)
				current = o
			}
		}
		result = append(result, current)
	}
	return result
}

// computeLineCount returns the max line count from occurrences.
func computeLineCount(occs []Occurrence) int {
	if len(occs) == 0 {
		return 0
	}
	maxCount := 0
	for _, o := range occs {
		count := o.EndLine - o.StartLine + 1
		if count > maxCount {
			maxCount = count
		}
	}
	return maxCount
}

// itoa converts an int to a string.
func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// chainMatches groups matches into chains where adjacent windows
// overlap or are adjacent on BOTH sides simultaneously.
func chainMatches(matches []windowMatch) [][]windowMatch {
	if len(matches) == 0 {
		return nil
	}
	if len(matches) == 1 {
		return [][]windowMatch{matches}
	}

	var chains [][]windowMatch
	var currentChain []windowMatch
	currentChain = append(currentChain, matches[0])

	for i := 1; i < len(matches); i++ {
		prev := currentChain[len(currentChain)-1]
		curr := matches[i]

		leftContiguous := curr.Left.StartLine <= prev.Left.EndLine+1
		rightContiguous := curr.Right.StartLine <= prev.Right.EndLine+1

		if leftContiguous && rightContiguous {
			currentChain = append(currentChain, curr)
		} else {
			chains = append(chains, currentChain)
			currentChain = []windowMatch{curr}
		}
	}
	if len(currentChain) > 0 {
		chains = append(chains, currentChain)
	}

	return chains
}

// buildComponentFromChain builds an alignedComponent from a chain of matches.
func buildComponentFromChain(fp string, chain []windowMatch) *alignedComponent {
	if len(chain) == 0 {
		return nil
	}

	comp := alignedComponent{
		Fingerprint:   chain[0].Fingerprint,
		Occurrences:   make(map[string][]tokenRange),
		MaxTokenCount: 0,
		MaxLineCount:  0,
	}

	fileWindows := make(map[string][]rawWindow)
	for _, m := range chain {
		fileWindows[m.Left.Path] = append(fileWindows[m.Left.Path], m.Left)
		fileWindows[m.Right.Path] = append(fileWindows[m.Right.Path], m.Right)
	}

	maxTokenSpan := 0
	maxLineSpan := 0
	for path, wins := range fileWindows {
		sort.Slice(wins, func(i, j int) bool {
			if wins[i].StartLine != wins[j].StartLine {
				return wins[i].StartLine < wins[j].StartLine
			}
			return wins[i].StartPos < wins[j].StartPos
		})

		var coalesced []rawWindow
		if len(wins) > 0 {
			current := wins[0]
			for i := 1; i < len(wins); i++ {
				w := wins[i]
				if w.StartLine <= current.EndLine+1 {
					if w.StartLine < current.StartLine {
						current.StartLine = w.StartLine
						current.StartPos = w.StartPos
					}
					if w.EndLine > current.EndLine {
						current.EndLine = w.EndLine
						current.EndPos = w.EndPos
					}
				} else {
					coalesced = append(coalesced, current)
					current = w
				}
			}
			coalesced = append(coalesced, current)
		}

		for _, w := range coalesced {
			comp.Occurrences[path] = append(comp.Occurrences[path], tokenRange{
				StartPos: w.StartLine,
				EndPos:   w.EndLine,
			})
			tokenSpan := w.EndPos - w.StartPos + 1
			lineSpan := w.EndLine - w.StartLine + 1
			if tokenSpan > maxTokenSpan {
				maxTokenSpan = tokenSpan
			}
			if lineSpan > maxLineSpan {
				maxLineSpan = lineSpan
			}
		}
	}
	comp.MaxTokenCount = maxTokenSpan
	comp.MaxLineCount = maxLineSpan

	return &comp
}

// canonicalPathSet returns the canonical (sorted, joined) path set for a finding.
func canonicalPathSet(occs []Occurrence) string {
	paths := make([]string, len(occs))
	for i, occ := range occs {
		paths[i] = occ.Path
	}
	sort.Strings(paths)
	return strings.Join(paths, "|")
}

// computeCoalescedFingerprint creates a stable fingerprint using AlgorithmVersion.
// DEPRECATED: Use v3SeedFingerprint from v3.go for algorithm v3.
func computeCoalescedFingerprint(tokenFP, pathSet string) string {
	return v3SeedFingerprint(tokenFP, pathSet)
}

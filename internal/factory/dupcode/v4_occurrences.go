// Package dupcode provides duplicate code detection for Go source files.
package dupcode

import (
	"fmt"
	"sort"
)

// maximalOccurrence tracks occurrences with token positions for deduplication.
type maximalOccurrence struct {
	Path      string
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

// v4InternalFinding is the production-owned V4 finding representation that
// retains token positions through merge and public projection.
type v4InternalFinding struct {
	StableFingerprint string
	TokenCount        int
	LineCount         int
	Occurrences       []maximalOccurrence
}

// v4Finding is retained as a package-private compatibility alias for existing
// focused tests. The merge implementation uses v4InternalFinding values.
type v4Finding = v4InternalFinding

// v4OccurrenceFromChain converts a cloneChain to maximalOccurrences.
// Preserves token positions for later deduplication during N-way merging.
func v4OccurrenceFromChain(chain cloneChain) []maximalOccurrence {
	// Collect all windows grouped by file
	fileWindows := make(map[string][]rawWindow)

	for _, m := range chain.Matches {
		fileWindows[m.Left.Path] = append(fileWindows[m.Left.Path], m.Left)
		fileWindows[m.Right.Path] = append(fileWindows[m.Right.Path], m.Right)
	}

	// For each file, coalesce overlapping token ranges into distinct occurrences
	var allOccurrences []maximalOccurrence
	for path, windows := range fileWindows {
		// Sort by token start position
		sort.Slice(windows, func(i, j int) bool {
			return windows[i].StartPos < windows[j].StartPos
		})

		// Coalesce using token positions and track line bounds
		distinctRanges := coalesceWithLineBounds(windows)

		// Convert to maximalOccurrence with token positions preserved
		for _, item := range distinctRanges {
			allOccurrences = append(allOccurrences, maximalOccurrence{
				Path:      path,
				StartPos:  item.StartPos,
				EndPos:    item.EndPos,
				StartLine: item.StartLine,
				EndLine:   item.EndLine,
			})
		}
	}

	// Sort occurrences deterministically by path and token position
	sort.Slice(allOccurrences, func(i, j int) bool {
		if allOccurrences[i].Path != allOccurrences[j].Path {
			return allOccurrences[i].Path < allOccurrences[j].Path
		}
		if allOccurrences[i].StartPos != allOccurrences[j].StartPos {
			return allOccurrences[i].StartPos < allOccurrences[j].StartPos
		}
		return allOccurrences[i].EndPos < allOccurrences[j].EndPos
	})

	return allOccurrences
}

// occurrenceWithLines tracks token range with line bounds for output.
type occurrenceWithLines struct {
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

// coalesceWithLineBounds coalesces overlapping windows and tracks line bounds.
func coalesceWithLineBounds(windows []rawWindow) []occurrenceWithLines {
	if len(windows) == 0 {
		return nil
	}

	var result []occurrenceWithLines
	current := occurrenceWithLines{
		StartPos:  windows[0].StartPos,
		EndPos:    windows[0].EndPos,
		StartLine: windows[0].StartLine,
		EndLine:   windows[0].EndLine,
	}

	for i := 1; i < len(windows); i++ {
		w := windows[i]
		// Overlap or adjacent in token space?
		if w.StartPos <= current.EndPos+1 {
			// Extend current range and update line bounds
			if w.EndPos > current.EndPos {
				current.EndPos = w.EndPos
			}
			if w.EndLine > current.EndLine {
				current.EndLine = w.EndLine
			}
		} else {
			// Start new range
			result = append(result, current)
			current = occurrenceWithLines{
				StartPos:  w.StartPos,
				EndPos:    w.EndPos,
				StartLine: w.StartLine,
				EndLine:   w.EndLine,
			}
		}
	}
	result = append(result, current)

	return result
}

// mergeKey identifies findings that can be merged together.
type mergeKey struct {
	StableFingerprint string
	TokenCount        int
}

// v4MergeFindings merges v4Findings representing the same N-way clone.
// Merges by (stable fingerprint, token count) and deduplicates by token position.
func v4MergeFindings(findings []v4InternalFinding) []v4InternalFinding {
	if len(findings) <= 1 {
		return findings
	}

	// Group by (stable fingerprint, token count)
	byKey := make(map[mergeKey][]v4InternalFinding)
	for _, f := range findings {
		key := mergeKey{
			StableFingerprint: f.StableFingerprint,
			TokenCount:        f.TokenCount,
		}
		byKey[key] = append(byKey[key], f)
	}

	// Sort keys for deterministic iteration
	var keys []mergeKey
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].StableFingerprint != keys[j].StableFingerprint {
			return keys[i].StableFingerprint < keys[j].StableFingerprint
		}
		return keys[i].TokenCount < keys[j].TokenCount
	})

	var merged []v4InternalFinding
	for _, key := range keys {
		group := byKey[key]
		if len(group) == 0 {
			continue
		}
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}

		// Merge all occurrences from all findings with same key
		merged = append(merged, v4MergeToNWayClone(group))
	}

	return merged
}

// v4MergeToNWayClone merges pair findings into an N-way clone.
// Deduplicates by token position (Path + StartPos + EndPos).
// Invariant: callers (v4MergeFindings) group by (StableFingerprint, TokenCount),
// so all group members are guaranteed to share TokenCount.
// A mismatch indicates a programming error and must not silently lose findings.
func v4MergeToNWayClone(group []v4InternalFinding) v4InternalFinding {
	if len(group) == 0 {
		return v4InternalFinding{}
	}

	if len(group) == 1 {
		return group[0]
	}

	// Invariant check: all members must have the same TokenCount.
	firstTokenCount := group[0].TokenCount
	for i := 1; i < len(group); i++ {
		if group[i].TokenCount != firstTokenCount {
			panic(fmt.Sprintf("dupcode: inconsistent token counts in v4 merge group: %d vs %d",
				firstTokenCount, group[i].TokenCount))
		}
	}

	// Collect all unique occurrences by token position (not just line)
	seen := make(map[string]bool)
	var allOccs []maximalOccurrence
	for _, f := range group {
		for _, occ := range f.Occurrences {
			// Use token position for deduplication
			key := fmt.Sprintf("%s:%d:%d", occ.Path, occ.StartPos, occ.EndPos)
			if !seen[key] {
				seen[key] = true
				allOccs = append(allOccs, occ)
			}
		}
	}

	// Sort occurrences by path and token position
	sort.Slice(allOccs, func(i, j int) bool {
		if allOccs[i].Path != allOccs[j].Path {
			return allOccs[i].Path < allOccs[j].Path
		}
		if allOccs[i].StartPos != allOccs[j].StartPos {
			return allOccs[i].StartPos < allOccs[j].StartPos
		}
		return allOccs[i].EndPos < allOccs[j].EndPos
	})

	// Create merged finding, taking the maximum LineCount across the group
	// since the same normalized token body may occupy different numbers of
	// lines in different files.
	merged := group[0]
	for _, f := range group[1:] {
		if f.LineCount > merged.LineCount {
			merged.LineCount = f.LineCount
		}
	}
	merged.Occurrences = allOccs

	return merged
}

// projectToPublicOccurrence converts maximalOccurrence to public Occurrence.
func projectToPublicOccurrence(mo maximalOccurrence) Occurrence {
	return Occurrence{
		Path:      mo.Path,
		StartLine: mo.StartLine,
		EndLine:   mo.EndLine,
	}
}

// convertV4FindingToFinding converts a production internal finding to public
// Finding. Public projection intentionally drops token positions.
func convertV4FindingToFinding(f v4InternalFinding) Finding {
	occurrences := make([]Occurrence, len(f.Occurrences))
	for i, mo := range f.Occurrences {
		occurrences[i] = projectToPublicOccurrence(mo)
	}

	return Finding{
		Fingerprint: truncateFingerprint(f.StableFingerprint),
		TokenCount:  f.TokenCount,
		LineCount:   f.LineCount,
		Occurrences: occurrences,
	}
}

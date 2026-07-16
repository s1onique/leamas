// Package dupcode provides V4 occurrence and merge logic.
//
// The V4 chain representation carries Left and Right side occurrences. The
// production-owned merge seam (v4InternalFindingsFromChains) preserves the
// token positions of each occurrence through dedup, N-way merge, and the
// public Finding projection.
//
// Occurrence identity is the structured key (Path, StartPos, EndPos);
// line geometry is NOT part of identity. Two occurrences sharing that
// key MUST agree on StartLine and EndLine; the
// assertOccurrenceIdentityInvariants check fails closed when they do
// not. The check is invoked BEFORE every deduplication so a latent
// inconsistency cannot be masked by dedup.
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

// v4InternalFinding is the production-owned V4 finding representation.
type v4InternalFinding struct {
	StableFingerprint string
	TokenCount        int
	LineCount         int
	Occurrences       []maximalOccurrence
}

// v4Finding is retained as a package-private compatibility alias for existing
// focused tests.
type v4Finding = v4InternalFinding

// maximalOccurrenceKey is the structured identity key for V4 occurrences.
// It replaces the previous colon-delimited string encoding so the key
// participates in tests, comparisons, and map lookups as a real Go value.
type maximalOccurrenceKey struct {
	Path     string
	StartPos int
	EndPos   int
}

// v4OccurrenceFromChain extracts occurrences from a chain.
//
// Each chain match contributes its Left and Right windows to SEPARATE
// per-file maps. The maps are coalesced independently so that distinct
// non-overlapping occurrences in the same file (RepeatedMultiplicity B1/B2
// case) are preserved rather than collapsed into one block.
//
// Cross-side same-range occurrences (same Path, StartPos, EndPos) are
// deduplicated so the same physical occurrence attached to both Left and
// Right sides is recorded once.
//
// The returned slice is sorted deterministically by (Path, StartPos,
// EndPos).
func v4OccurrenceFromChain(chain cloneChain) []maximalOccurrence {
	fileLeftWindows := make(map[string][]rawWindow)
	fileRightWindows := make(map[string][]rawWindow)

	for _, m := range chain.Matches {
		fileLeftWindows[m.Left.Path] = append(fileLeftWindows[m.Left.Path], m.Left)
		fileRightWindows[m.Right.Path] = append(fileRightWindows[m.Right.Path], m.Right)
	}

	var allOccurrences []maximalOccurrence

	for path, wins := range fileLeftWindows {
		coalesced := coalesceLeftRightWindows(path, wins)
		allOccurrences = append(allOccurrences, coalesced...)
	}
	for path, wins := range fileRightWindows {
		coalesced := coalesceLeftRightWindows(path, wins)
		allOccurrences = append(allOccurrences, coalesced...)
	}

	// Invariant check BEFORE dedup so inconsistencies cannot be
	// masked.
	assertOccurrenceIdentityInvariants(allOccurrences)

	seen := make(map[maximalOccurrenceKey]bool)
	deduped := allOccurrences[:0]
	for _, o := range allOccurrences {
		key := maximalOccurrenceKey{Path: o.Path, StartPos: o.StartPos, EndPos: o.EndPos}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, o)
	}

	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Path != deduped[j].Path {
			return deduped[i].Path < deduped[j].Path
		}
		if deduped[i].StartPos != deduped[j].StartPos {
			return deduped[i].StartPos < deduped[j].StartPos
		}
		return deduped[i].EndPos < deduped[j].EndPos
	})

	return deduped
}

// occurrenceKey produces the structured deduplication key for
// maximalOccurrence.
//
// The identity is (Path, StartPos, EndPos). Line geometry is NOT
// included. When two occurrences share the same token-position key but
// disagree on StartLine or EndLine, the invariant check in
// assertOccurrenceIdentityInvariants fails closed.
func occurrenceKey(o maximalOccurrence) maximalOccurrenceKey {
	return maximalOccurrenceKey{Path: o.Path, StartPos: o.StartPos, EndPos: o.EndPos}
}

// assertOccurrenceIdentityInvariants fails closed when two occurrences
// share the same (Path, StartPos, EndPos) token-position key but
// disagree on StartLine or EndLine. The geometry is part of the public
// projection; inconsistent geometry on one span is an internal
// invariant violation.
func validateOccurrenceIdentityInvariants(occs []maximalOccurrence) error {
	canonical := make(map[maximalOccurrenceKey]maximalOccurrence)
	for _, o := range occs {
		key := occurrenceKey(o)
		if prev, ok := canonical[key]; ok {
			if prev.StartLine != o.StartLine || prev.EndLine != o.EndLine {
				return fmt.Errorf("dupcode: occurrence identity invariant violated for %s:%d:%d: prev line=%d-%d next line=%d-%d",
					o.Path, o.StartPos, o.EndPos,
					prev.StartLine, prev.EndLine,
					o.StartLine, o.EndLine)
			}
			continue
		}
		canonical[key] = o
	}
	return nil
}

func assertOccurrenceIdentityInvariants(occs []maximalOccurrence) {
	if err := validateOccurrenceIdentityInvariants(occs); err != nil {
		panic(err)
	}
}

// coalesceLeftRightWindows coalesces one side's windows into distinct
// occurrence blocks.
func coalesceLeftRightWindows(path string, windows []rawWindow) []maximalOccurrence {
	if len(windows) == 0 {
		return nil
	}

	sorted := make([]rawWindow, len(windows))
	copy(sorted, windows)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartPos < sorted[j].StartPos
	})

	var result []maximalOccurrence
	current := occurrenceWithLines{
		StartPos:  sorted[0].StartPos,
		EndPos:    sorted[0].EndPos,
		StartLine: sorted[0].StartLine,
		EndLine:   sorted[0].EndLine,
	}

	for i := 1; i < len(sorted); i++ {
		w := sorted[i]
		if w.StartPos <= current.EndPos+1 {
			if w.EndPos > current.EndPos {
				current.EndPos = w.EndPos
			}
			if w.EndLine > current.EndLine {
				current.EndLine = w.EndLine
			}
		} else {
			result = append(result, maximalOccurrence{
				Path:      path,
				StartPos:  current.StartPos,
				EndPos:    current.EndPos,
				StartLine: current.StartLine,
				EndLine:   current.EndLine,
			})
			current = occurrenceWithLines{
				StartPos:  w.StartPos,
				EndPos:    w.EndPos,
				StartLine: w.StartLine,
				EndLine:   w.EndLine,
			}
		}
	}
	result = append(result, maximalOccurrence{
		Path:      path,
		StartPos:  current.StartPos,
		EndPos:    current.EndPos,
		StartLine: current.StartLine,
		EndLine:   current.EndLine,
	})

	return result
}

// occurrenceWithLines tracks token range with line bounds.
type occurrenceWithLines struct {
	StartPos  int
	EndPos    int
	StartLine int
	EndLine   int
}

// mergeKey identifies findings that can be merged together.
type mergeKey struct {
	StableFingerprint string
	TokenCount        int
}

// v4MergeFindings merges v4Findings representing the same N-way clone.
func v4MergeFindings(findings []v4InternalFinding) []v4InternalFinding {
	if len(findings) <= 1 {
		return findings
	}

	byKey := make(map[mergeKey][]v4InternalFinding)
	for _, f := range findings {
		key := mergeKey{
			StableFingerprint: f.StableFingerprint,
			TokenCount:        f.TokenCount,
		}
		byKey[key] = append(byKey[key], f)
	}

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
		merged = append(merged, v4MergeToNWayCloneLegacy(group))
	}

	return merged
}

// v4MergeToNWayCloneLegacy preserves the old package-private chain
// helper's permissive projection. It rewrites conflicting line geometry
// to a canonical value before invoking the panic-based
// v4MergeToNWayClone, deliberately bypassing the
// validateOccurrenceIdentityInvariants check. The checked production
// pipeline (v4BuildInternalFindings / v4BuildInternalFindingsChecked)
// NEVER calls this function.
//
// Capability boundary: v4MergeToNWayCloneLegacy is reachable only from
// the legacy chain construction path (v4InternalFindingsFromChains
// -> v4MergeFindings). v4InternalFindingsFromChains is itself called
// only from v4FindingsFromChains and v4CoalesceFindings. Neither
// function participates in the production CheckRepo path; production
// routes through v4BuildInternalFindingsChecked which delegates to
// v4MaterializeComponents instead of v4InternalFindingsFromChains.
//
// If a future refactor must reconnect the legacy path to production,
// this function MUST be either deleted or replaced with an
// error-returning variant that fails closed on a geometry conflict.
// Until then, every comment and helper around this function must
// remain unambiguous about its test-only capability.
func v4MergeToNWayCloneLegacy(group []v4InternalFinding) v4InternalFinding {
	copyGroup := make([]v4InternalFinding, len(group))
	copy(copyGroup, group)
	canonical := make(map[maximalOccurrenceKey]maximalOccurrence)
	for i := range copyGroup {
		copyGroup[i].Occurrences = append([]maximalOccurrence(nil), copyGroup[i].Occurrences...)
		for j, occurrence := range copyGroup[i].Occurrences {
			key := occurrenceKey(occurrence)
			if previous, ok := canonical[key]; ok {
				copyGroup[i].Occurrences[j].StartLine = previous.StartLine
				copyGroup[i].Occurrences[j].EndLine = previous.EndLine
			} else {
				canonical[key] = occurrence
			}
		}
	}
	return v4MergeToNWayClone(copyGroup)
}

// v4MergeToNWayClone merges pair findings into an N-way clone.
func v4MergeToNWayClone(group []v4InternalFinding) v4InternalFinding {
	if len(group) == 0 {
		return v4InternalFinding{}
	}

	if len(group) == 1 {
		return group[0]
	}

	firstTokenCount := group[0].TokenCount
	for i := 1; i < len(group); i++ {
		if group[i].TokenCount != firstTokenCount {
			panic(fmt.Sprintf("dupcode: inconsistent token counts in v4 merge group: %d vs %d",
				firstTokenCount, group[i].TokenCount))
		}
	}

	// Flatten the complete merge group first. The invariant must run once
	// across all members before any identity-based deduplication can mask a
	// cross-finding line conflict.
	var allOccs []maximalOccurrence
	for _, f := range group {
		allOccs = append(allOccs, f.Occurrences...)
	}
	assertOccurrenceIdentityInvariants(allOccs)

	seen := make(map[maximalOccurrenceKey]bool)
	deduped := allOccs[:0]
	for _, occ := range allOccs {
		key := occurrenceKey(occ)
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, occ)
	}
	allOccs = deduped

	sort.Slice(allOccs, func(i, j int) bool {
		if allOccs[i].Path != allOccs[j].Path {
			return allOccs[i].Path < allOccs[j].Path
		}
		if allOccs[i].StartPos != allOccs[j].StartPos {
			return allOccs[i].StartPos < allOccs[j].StartPos
		}
		return allOccs[i].EndPos < allOccs[j].EndPos
	})

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

// convertV4FindingToFinding converts a production internal finding to public Finding.
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

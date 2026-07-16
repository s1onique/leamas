package dupcode

import "sort"

// compareV4InternalFindings is the one total publication comparator.
func compareV4InternalFindings(left, right v4InternalFinding) int {
	if left.StableFingerprint != right.StableFingerprint {
		return compareStrings(left.StableFingerprint, right.StableFingerprint)
	}
	if left.TokenCount != right.TokenCount {
		return compareInts(left.TokenCount, right.TokenCount)
	}
	if left.LineCount != right.LineCount {
		return compareInts(left.LineCount, right.LineCount)
	}
	return compareV4OccurrenceSequences(left.Occurrences, right.Occurrences)
}

func compareV4OccurrenceSequences(left, right []maximalOccurrence) int {
	leftSorted := append([]maximalOccurrence(nil), left...)
	rightSorted := append([]maximalOccurrence(nil), right...)
	sort.Slice(leftSorted, func(i, j int) bool {
		return compareV4PublicationOccurrences(leftSorted[i], leftSorted[j]) < 0
	})
	sort.Slice(rightSorted, func(i, j int) bool {
		return compareV4PublicationOccurrences(rightSorted[i], rightSorted[j]) < 0
	})
	limit := len(leftSorted)
	if len(rightSorted) < limit {
		limit = len(rightSorted)
	}
	for i := 0; i < limit; i++ {
		if order := compareV4PublicationOccurrences(leftSorted[i], rightSorted[i]); order != 0 {
			return order
		}
	}
	return compareInts(len(leftSorted), len(rightSorted))
}

func compareV4PublicationOccurrences(left, right maximalOccurrence) int {
	if left.Path != right.Path {
		return compareStrings(left.Path, right.Path)
	}
	if left.StartLine != right.StartLine {
		return compareInts(left.StartLine, right.StartLine)
	}
	if left.EndLine != right.EndLine {
		return compareInts(left.EndLine, right.EndLine)
	}
	if left.StartPos != right.StartPos {
		return compareInts(left.StartPos, right.StartPos)
	}
	return compareInts(left.EndPos, right.EndPos)
}

func sortV4InternalFindings(findings []v4InternalFinding) {
	sort.Slice(findings, func(i, j int) bool {
		return compareV4InternalFindings(findings[i], findings[j]) < 0
	})
}

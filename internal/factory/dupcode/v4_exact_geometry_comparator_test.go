// Package dupcode verifies total test-owned geometry canonicalization.
package dupcode

import "testing"

func TestCanonicalizeInternalOccurrences_TotalComparator(t *testing.T) {
	later := exactInternalOccurrenceGeometry{
		Path:      "same.go",
		StartPos:  10,
		EndPos:    20,
		StartLine: 8,
		EndLine:   12,
	}
	earlier := exactInternalOccurrenceGeometry{
		Path:      "same.go",
		StartPos:  10,
		EndPos:    20,
		StartLine: 7,
		EndLine:   13,
	}

	got := canonicalizeInternalOccurrences([]exactInternalOccurrenceGeometry{later, earlier})
	if len(got) != 2 {
		t.Fatalf("canonical occurrence count = %d, want 2", len(got))
	}
	if got[0] != earlier || got[1] != later {
		t.Fatalf("canonical line-geometry order = %+v, want [%+v %+v]", got, earlier, later)
	}

	sameStartLaterEnd := earlier
	sameStartLaterEnd.EndLine = 14
	got = canonicalizeInternalOccurrences([]exactInternalOccurrenceGeometry{sameStartLaterEnd, earlier})
	if got[0] != earlier || got[1] != sameStartLaterEnd {
		t.Fatalf("canonical EndLine tie-break order = %+v, want [%+v %+v]",
			got, earlier, sameStartLaterEnd)
	}
}

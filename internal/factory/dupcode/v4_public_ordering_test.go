package dupcode

import (
	"reflect"
	"testing"
)

func TestV4PublicOrdering_EqualFingerprintAndTokenCountUsesLineGeometry(t *testing.T) {
	left := v4InternalFinding{
		StableFingerprint: "same",
		TokenCount:        400,
		LineCount:         20,
		Occurrences:       []maximalOccurrence{{Path: "a.go", StartLine: 10, EndLine: 29, StartPos: 100, EndPos: 499}},
	}
	right := left
	right.LineCount = 21
	right.Occurrences = []maximalOccurrence{{Path: "a.go", StartLine: 11, EndLine: 31, StartPos: 100, EndPos: 499}}
	if compareV4InternalFindings(left, right) >= 0 {
		t.Fatalf("line geometry did not participate in total order: left=%+v right=%+v", left, right)
	}
}

func TestV4PublicOrdering_EqualPrefixUsesOccurrenceGeometry(t *testing.T) {
	left := v4InternalFinding{
		StableFingerprint: "same", TokenCount: 400, LineCount: 20,
		Occurrences: []maximalOccurrence{{Path: "a.go", StartLine: 10, EndLine: 29, StartPos: 100, EndPos: 499}},
	}
	right := left
	right.Occurrences = []maximalOccurrence{{Path: "a.go", StartLine: 10, EndLine: 29, StartPos: 101, EndPos: 500}}
	if compareV4InternalFindings(left, right) >= 0 {
		t.Fatalf("token geometry did not participate in total order")
	}
}

func TestV4PublicOrdering_ProjectionDoesNotResort(t *testing.T) {
	findings := []v4InternalFinding{
		{StableFingerprint: "b", TokenCount: 1, LineCount: 1},
		{StableFingerprint: "a", TokenCount: 1, LineCount: 1},
	}
	sortV4InternalFindings(findings)
	if got := findings[0].StableFingerprint; got != "a" {
		t.Fatalf("canonical sort did not use total comparator: %q", got)
	}
	if reflect.DeepEqual(findings[0], findings[1]) {
		t.Fatal("distinct ordered findings unexpectedly equal")
	}
}

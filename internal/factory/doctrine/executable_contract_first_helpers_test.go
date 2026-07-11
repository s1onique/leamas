package doctrine

import (
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// insertCRLFInMarkedBlock rewrites the marked block to use CRLF line endings.
func insertCRLFInMarkedBlock(s string) string {
	beginIdx := strings.Index(s, ecfBeginMarker)
	endIdx := strings.Index(s, ecfEndMarker)
	if beginIdx == -1 || endIdx == -1 {
		return s
	}
	start := beginIdx + len(ecfBeginMarker)
	if start >= endIdx {
		return s
	}
	before := s[:start]
	block := s[start:endIdx]
	after := s[endIdx:]

	// Convert LF inside the block to CRLF.
	block = strings.ReplaceAll(block, "\n", "\r\n")
	return before + block + after
}

func TestInsertCRLFInMarkedBlock(t *testing.T) {
	input := ecfBeginMarker + "\nline1\nline2\n" + ecfEndMarker
	result := insertCRLFInMarkedBlock(input)
	if !strings.Contains(result, "\r\n") {
		t.Error("expected CRLF in marked block")
	}
}

func TestExtractECFMarkedBlock(t *testing.T) {
	content := "before " + ecfBeginMarker + "\nbody\n" + ecfEndMarker + " after"
	got := extractECFMarkedBlock(content)
	if got != "body" {
		t.Errorf("expected 'body', got %q", got)
	}
}

func TestNormalizeECFContent(t *testing.T) {
	if got := normalizeECFContent("a\r\nb\r\n"); got != "a\nb" {
		t.Errorf("CRLF normalization failed: %q", got)
	}
	if got := normalizeECFContent("a\nb\n"); got != "a\nb" {
		t.Errorf("trailing newline trim failed: %q", got)
	}
}

func TestHasNestedECFMarks(t *testing.T) {
	plain := ecfBeginMarker + "\nA\n" + ecfEndMarker
	if hasNestedECFMarks(plain) {
		t.Error("non-nested content should not be reported as nested")
	}
	nested := ecfBeginMarker + "\n" + ecfBeginMarker + "\nA\n" + ecfEndMarker + "\n" + ecfEndMarker
	if !hasNestedECFMarks(nested) {
		t.Error("nested content should be detected")
	}
}

func TestCountOccurrences(t *testing.T) {
	if got := countOccurrences("aXbXc", "X"); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
	if got := countOccurrences("aaa", ""); got != 0 {
		t.Errorf("empty substring should return 0, got %d", got)
	}
}

func TestSortFindings(t *testing.T) {
	findings := []checks.Finding{
		{Path: "b.go", Kind: "ERR1", Message: "msg1"},
		{Path: "a.go", Kind: "ERR2", Message: "msg2"},
	}
	checks.SortFindings(findings)
	if findings[0].Path != "a.go" || findings[1].Path != "b.go" {
		t.Error("expected sorted by path")
	}
}

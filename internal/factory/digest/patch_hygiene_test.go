// Package digest provides targeted digest generation for Git repositories.
package digest

import (
	"strings"
	"testing"
)

func TestRenderPatchHygiene_PassStableOrder(t *testing.T) {
	ph := PatchHygiene{
		GitDiffCheck:     PatchHygienePass,
		WhitespaceErrors: 0,
		ConflictMarkers:  0,
		DiagnosticLines:  0,
		Diagnostics:      nil,
	}

	result := RenderPatchHygiene(ph)

	expectedOrder := []string{
		"git_diff_check",
		"whitespace_errors",
		"conflict_markers",
		"diagnostic_lines",
	}

	for i, key := range expectedOrder {
		if i > 0 {
			idx := strings.Index(result, key)
			prevIdx := strings.Index(result, expectedOrder[i-1])
			if idx <= prevIdx {
				t.Errorf("key %q should come after %q", key, expectedOrder[i-1])
			}
		}
	}

	if !strings.Contains(result, "git_diff_check=pass") {
		t.Error("expected pass status")
	}
	if strings.Contains(result, "diagnostics:") {
		t.Error("should not contain diagnostics section when diagnostic_lines=0")
	}
}

func TestRenderPatchHygiene_FailWithDiagnostics(t *testing.T) {
	ph := PatchHygiene{
		GitDiffCheck:     PatchHygieneFail,
		WhitespaceErrors: 2,
		ConflictMarkers:  1,
		DiagnosticLines:  3,
		Diagnostics: []string{
			"file.go:12: trailing whitespace.",
			"file.go:20: leftover conflict marker",
			"README.md:4: trailing whitespace.",
		},
	}

	result := RenderPatchHygiene(ph)

	if !strings.Contains(result, "git_diff_check=fail") {
		t.Error("expected fail status")
	}
	if !strings.Contains(result, "whitespace_errors=2") {
		t.Error("expected 2 whitespace errors")
	}
	if !strings.Contains(result, "conflict_markers=1") {
		t.Error("expected 1 conflict marker")
	}
	if !strings.Contains(result, "diagnostic_lines=3") {
		t.Error("expected 3 diagnostic lines")
	}
	if !strings.Contains(result, "diagnostics:") {
		t.Error("should contain diagnostics section")
	}
	if !strings.Contains(result, "trailing whitespace") {
		t.Error("should contain whitespace diagnostics")
	}
	if !strings.Contains(result, "conflict marker") {
		t.Error("should contain conflict marker diagnostics")
	}
}

func TestRenderPatchHygiene_Unavailable(t *testing.T) {
	ph := PatchHygiene{
		GitDiffCheck:     PatchHygieneUnavailable,
		WhitespaceErrors: 0,
		ConflictMarkers:  0,
		DiagnosticLines:  1,
		Diagnostics: []string{
			"git diff --check unavailable: some error",
		},
	}

	result := RenderPatchHygiene(ph)

	if !strings.Contains(result, "git_diff_check=unavailable") {
		t.Error("expected unavailable status")
	}
}

func TestParsePatchHygieneDiagnostics_CountsWhitespace(t *testing.T) {
	output := `file.go:12: trailing whitespace.
another.go:20: trailing whitespace.
`

	diagnostics := ParsePatchHygieneDiagnostics(output, "")

	if len(diagnostics) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(diagnostics))
	}

	for _, diag := range diagnostics {
		if isConflictMarker(diag) {
			t.Error("should not be conflict marker")
		}
	}
}

func TestParsePatchHygieneDiagnostics_CountsConflictMarkers(t *testing.T) {
	output := `file.go:12: leftover conflict marker
another.go:20: leftover conflict marker
`

	diagnostics := ParsePatchHygieneDiagnostics(output, "")

	if len(diagnostics) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(diagnostics))
	}

	for _, diag := range diagnostics {
		if !isConflictMarker(diag) {
			t.Error("should be conflict marker")
		}
	}
}

func TestPatchHygieneDiagnostics_AreBounded(t *testing.T) {
	// Generate more than MaxPatchHygieneDiagnostics lines
	var lines []string
	for i := 0; i < MaxPatchHygieneDiagnostics+10; i++ {
		lines = append(lines, "file.go:12: trailing whitespace.")
	}
	output := strings.Join(lines, "\n")

	diagnostics := ParsePatchHygieneDiagnostics(output, "")

	if len(diagnostics) != MaxPatchHygieneDiagnostics {
		t.Errorf("expected %d diagnostics, got %d", MaxPatchHygieneDiagnostics, len(diagnostics))
	}
}

func TestPatchHygieneDiagnostics_NormalizeRepoRoot(t *testing.T) {
	repoRoot := "/Users/test/repo"
	output := "/Users/test/repo/internal/file.go:12: trailing whitespace."

	diagnostics := ParsePatchHygieneDiagnostics(output, repoRoot)

	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}

	if strings.Contains(diagnostics[0], repoRoot) {
		t.Error("repo root should be normalized")
	}
	if !strings.Contains(diagnostics[0], "<repo>") {
		t.Error("should contain <repo> placeholder")
	}
}

func TestTruncateLine(t *testing.T) {
	longLine := strings.Repeat("a", MaxDiagnosticLineLength+100)
	truncated := truncateLine(longLine, MaxDiagnosticLineLength)

	if len(truncated) > MaxDiagnosticLineLength+3 {
		t.Errorf("truncated line too long: %d", len(truncated))
	}
	if !strings.HasSuffix(truncated, "...") {
		t.Error("truncated line should end with ...")
	}

	shortLine := "short line"
	if truncateLine(shortLine, MaxDiagnosticLineLength) != shortLine {
		t.Error("short line should not be truncated")
	}
}

func TestMergePatchHygiene(t *testing.T) {
	a := PatchHygiene{
		GitDiffCheck:     PatchHygienePass,
		WhitespaceErrors: 1,
		ConflictMarkers:  0,
		DiagnosticLines:  1,
		Diagnostics:      []string{"a.go:1: trailing whitespace."},
	}

	b := PatchHygiene{
		GitDiffCheck:     PatchHygieneFail,
		WhitespaceErrors: 2,
		ConflictMarkers:  1,
		DiagnosticLines:  3,
		Diagnostics: []string{
			"b.go:1: trailing whitespace.",
			"c.go:1: trailing whitespace.",
			"c.go:5: leftover conflict marker",
		},
	}

	result := mergePatchHygiene(a, b)

	if result.GitDiffCheck != PatchHygieneFail {
		t.Error("merged status should be fail")
	}
	if result.WhitespaceErrors != 3 {
		t.Errorf("expected 3 whitespace errors, got %d", result.WhitespaceErrors)
	}
	if result.ConflictMarkers != 1 {
		t.Errorf("expected 1 conflict marker, got %d", result.ConflictMarkers)
	}
	if result.DiagnosticLines != 4 {
		t.Errorf("expected 4 diagnostic lines, got %d", result.DiagnosticLines)
	}
}

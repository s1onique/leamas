package agentcontext

import (
	"strings"
	"testing"
)

// TestIsDirective verifies the directive classifier used by the
// prose scanner.
func TestIsDirective(t *testing.T) {
	cases := []struct {
		unit string
		want bool
	}{
		{"run make gate", true},
		{"execute make factorize", true},
		{"commit completed work", true},
		{"push changes", true},
		{"create a tag", true},
		{"use make gate", true},
		{"make gate", true},
		{"make factorize", true},
		{"make gate because it is required", true},
		{"make factorize if the work is ready", true},
		{"always run make gate", true},
		{"please run make gate", true},
		{"then run make gate", true},
		{"next, run make gate", true},
		{"now run make gate", true},
		{"finally run make gate", true},
		{"before finishing, run make gate", true},
		{"after tests, run make gate", true},
		{"for final validation, run make gate", true},
		{"you must run make gate", true},
		{"you should run make gate", true},
		{"agents must run make gate", true},
		{"the repository gate validates state", false},
		{"make factorize is a tier-3 command", false},
		{"git commit is a publication boundary", false},
		{"git push is disabled by default", false},
		{"git tags are immutable lifecycle identities", false},
		{"history rewrite is forbidden", false},
		{"the make gate command", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.unit, func(t *testing.T) {
			stripped := stripAllDirectivePrefixes(c.unit)
			if got := isDirective(stripped); got != c.want {
				t.Fatalf("isDirective(stripped(%q) = %q) = %v, want %v", c.unit, stripped, got, c.want)
			}
		})
	}
}

// TestFindProtectedOccurrences verifies the per-occurrence detector.
func TestFindProtectedOccurrences(t *testing.T) {
	occs := findProtectedOccurrences("run make gate and push the commit and tag the commit")
	seen := make(map[ProtectedOpKind]bool)
	for _, o := range occs {
		seen[o.Op] = true
	}
	if !seen[OpMakeGate] || !seen[OpGitPush] || !seen[OpGitTag] {
		t.Fatalf("expected make_gate, git_push, git_tag; got: %+v", occs)
	}
}

// TestFindProtectedOccurrences_RespectsWordBoundaries verifies that
// "git tag" does not match across "git tags".
func TestFindProtectedOccurrences_RespectsWordBoundaries(t *testing.T) {
	unit := "git tags provide immutable lifecycle identity"
	occs := findProtectedOccurrences(unit)
	if len(occs) == 0 {
		t.Fatalf("expected at least one occurrence for git tag in git tags provide")
	}
	for _, o := range occs {
		if o.Form != "git tags" {
			t.Fatalf("expected occ form to be git tags (plural), got %q", o.Form)
		}
	}
}

// TestStripBacktickDelimitersPreservesContent verifies that
// inline-code backtick delimiters are stripped while the wrapped
// content is preserved.
func TestStripBacktickDelimitersPreservesContent(t *testing.T) {
	in := "Run `make gate` after implementation."
	got := stripBacktickDelimiters(in)
	if !strings.Contains(got, "make gate") {
		t.Fatalf("expected backtick content to be preserved, got: %q", got)
	}
}

// TestJoinParagraphLines verifies that soft-wrapped lines within a
// paragraph are joined into a single string with bullet prefixes
// stripped.
func TestJoinParagraphLines(t *testing.T) {
	got := joinParagraphLines("line one\nline two\n- bullet one\n- bullet two")
	if !strings.Contains(got, "line one") || !strings.Contains(got, "bullet two") {
		t.Fatalf("expected joined paragraph, got: %q", got)
	}
}

// TestParagraphHasImperativeVerb verifies the imperative-verb pattern
// helper.
func TestParagraphHasImperativeVerb(t *testing.T) {
	if !ParagraphHasImperativeVerb("run this now") {
		t.Fatalf("expected imperative verb detection")
	}
	if ParagraphHasImperativeVerb("benign description") {
		t.Fatalf("did not expect verb detection")
	}
}

// TestFormMatchesAtStart verifies the word-boundary check for
// protected-op forms.
func TestFormMatchesAtStart(t *testing.T) {
	if !formMatchesAtStart("make gate now", "make gate") {
		t.Fatalf("expected match with space boundary")
	}
	if !formMatchesAtStart("make gate", "make gate") {
		t.Fatalf("expected match at end of string")
	}
	if formMatchesAtStart("make gates", "make gate") {
		t.Fatalf("did not expect match across word boundary")
	}
	if formMatchesAtStart("make gater", "make gate") {
		t.Fatalf("did not expect match across word boundary")
	}
}

// TestStripAllDirectivePrefixes verifies iterative prefix stripping.
func TestStripAllDirectivePrefixes(t *testing.T) {
	cases := []struct {
		in     string
		expect string
	}{
		{"always run make gate", "run make gate"},
		{"please run make gate", "run make gate"},
		{"then run make gate", "run make gate"},
		{"you must always run make gate", "run make gate"},
		{"before finishing, run make gate", "run make gate"},
		{"after tests, run make gate", "run make gate"},
		{"for final validation, run make gate", "run make gate"},
		{"run make gate", "run make gate"},
	}
	for _, c := range cases {
		got := stripAllDirectivePrefixes(c.in)
		if got != c.expect {
			t.Fatalf("stripAllDirectivePrefixes(%q) = %q, want %q", c.in, got, c.expect)
		}
	}
}

// TestSplitCoarseClauses verifies coarse-clause segmentation.
func TestSplitCoarseClauses(t *testing.T) {
	cases := []struct {
		in     string
		expect []string
	}{
		{"Do not force-push but commit completed work", []string{"Do not force-push", "commit completed work"}},
		{"Run make gate then commit completed work", []string{"Run make gate", "commit completed work"}},
		{"Do not git push; commit completed work", []string{"Do not git push", "commit completed work"}},
		{"Run make gate and run make factorize", []string{"Run make gate and run make factorize"}},
	}
	for _, c := range cases {
		got := splitCoarseClauses(c.in)
		if len(got) != len(c.expect) {
			t.Fatalf("splitCoarseClauses(%q) = %v, want %v", c.in, got, c.expect)
		}
		for i := range got {
			if got[i] != c.expect[i] {
				t.Fatalf("splitCoarseClauses(%q)[%d] = %q, want %q", c.in, i, got[i], c.expect[i])
			}
		}
	}
}

// TestSplitCoordination verifies that "and" splits only when the
// clause has more than one protected operation.
func TestSplitCoordination(t *testing.T) {
	// Single op: no split.
	got := splitCoordination("run make gate")
	if len(got) != 1 {
		t.Fatalf("expected single clause, got %v", got)
	}
	// Multi ops: split.
	got = splitCoordination("run make gate and commit completed work")
	if len(got) != 2 {
		t.Fatalf("expected two sub-clauses, got %v", got)
	}
}

// TestCountMarkers verifies the canonical marker counter.
func TestCountMarkers(t *testing.T) {
	b, e := CountMarkers("<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->\n<!-- LEAMAS:AUTHORITY-CONTRACT:END -->\n")
	if b != 1 || e != 1 {
		t.Fatalf("expected 1/1, got %d/%d", b, e)
	}
	b, e = CountMarkers("<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->\n<!-- LEAMAS:AUTHORITY-CONTRACT:BEGIN -->\nend\n")
	if b != 2 {
		t.Fatalf("expected 2 BEGIN, got %d", b)
	}
}

// TestNextWordIsDescriptivePredicate verifies the helper used by
// isDirective.
func TestNextWordIsDescriptivePredicate(t *testing.T) {
	if !nextWordIsDescriptivePredicate("make gate is an expensive command", "make gate") {
		t.Fatalf("expected descriptive for 'is' after form")
	}
	if nextWordIsDescriptivePredicate("make gate because it is ready", "make gate") {
		t.Fatalf("expected non-descriptive for 'because' after form")
	}
	if nextWordIsDescriptivePredicate("make gate", "make gate") {
		t.Fatalf("expected false for bare form")
	}
}
package agentcontext

import (
	"strings"
	"testing"
)

// TestUnitHasImperativeIntent covers the imperative-intent
// classifier used by the prose scanner.
func TestUnitHasImperativeIntent(t *testing.T) {
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
		{"the repository gate validates state", false},
		{"make factorize is a tier-3 command", false},
		{"git push is publication", false},
		{"history rewrite is forbidden", false},
		{"the make gate command", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.unit, func(t *testing.T) {
			got := unitHasImperativeIntent(c.unit)
			if got != c.want {
				t.Fatalf("unitHasImperativeIntent(%q) = %v, want %v", c.unit, got, c.want)
			}
		})
	}
}

// TestUnitsMentionProtectedOps_AllOpsDetected verifies that a unit
// containing multiple protected operations returns all of them.
func TestUnitsMentionProtectedOps_AllOpsDetected(t *testing.T) {
	lower := "run make gate and push the commit and tag the commit"
	ops := unitsMentionProtectedOps(lower)
	seen := make(map[ProtectedOpKind]bool)
	for _, m := range ops {
		seen[m.op] = true
	}
	if !seen[OpMakeGate] || !seen[OpGitPush] || !seen[OpGitTag] {
		t.Fatalf("expected make_gate, git_push, git_tag; got: %+v", ops)
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

// TestUnitHasDescriptiveVerb verifies the descriptive-verb detector.
func TestUnitHasDescriptiveVerb(t *testing.T) {
	if !unitHasDescriptiveVerb("the make gate is a command") {
		t.Fatalf("expected detection for unit with 'is'")
	}
	if !unitHasDescriptiveVerb("git push is publication") {
		t.Fatalf("expected detection for unit with 'is'")
	}
	if unitHasDescriptiveVerb("run make gate") {
		t.Fatalf("did not expect detection for unit without descriptive verb")
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

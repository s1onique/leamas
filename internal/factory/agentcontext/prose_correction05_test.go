package agentcontext

import (
	"testing"
)

// PHASE: CORRECTION05 regression coverage.
//
// These tests exercise the new authority-source naming rules,
// case-independent period boundaries, occurrence canonicalization,
// multi-occurrence fail-closed, and exact operation identity.
// All tests use the production FindUnguardedProtectedOps function.

// TestCorrection05_AuthoritySource_NamedACT_Verifies reviews the
// required authority-source matrix. Every ACT guard must name the
// ACT authority source. Guards that do not name an ACT must NOT
// produce findings on otherwise unguarded prose.
func TestCorrection05_AuthoritySource_NamedACT_Verifies(t *testing.T) {
	// These guards contain "current ACT" or "the ACT" and must
	// suppress findings.
	pairs := []struct{ guard, body string }{
		{
			"Run make gate only when explicitly authorized by the current ACT.",
			"Run make gate only when explicitly authorized by the current ACT.",
		},
		{
			"Run make gate when the ACT authorizes this exact command.",
			"Run make gate when the ACT authorizes this exact command.",
		},
		{
			"Run make factorize only when the current ACT authorizes it.",
			"Run make factorize only when the current ACT authorizes it.",
		},
		{
			"Commit only when the current ACT delegates commit authority.",
			"Commit only when the current ACT delegates commit authority.",
		},
		{
			"Push only when delegated by the current ACT.",
			"Push only when delegated by the current ACT.",
		},
		{
			"Create a tag only when the ACT delegates tag authority.",
			"Create a tag only when the ACT delegates tag authority.",
		},
		{
			"Run make gate when authorized by the validated closure plan.",
			"Run make gate when authorized by the validated closure plan.",
		},
	}
	for _, p := range pairs {
		findings := FindUnguardedProtectedOps("AGENTS.md", p.body)
		if len(findings) != 0 {
			t.Errorf("expected no findings for %q, got %d", p.guard, len(findings))
		}
	}

	// These guards do NOT name an ACT and must NOT be accepted as
	// authority. They produce findings.
	noAuthRows := []string{
		"Run make gate when explicitly authorized.",
		"Run make gate when delegated.",
		"Run make gate only when that verification tier is explicitly authorized.",
		"Run make gate because deployment explicitly authorizes that exact command.",
		"Run make gate when policy authorizes that exact command.",
		"Commit completed work when explicitly authorized.",
		"Push the commit when deployment is delegated.",
	}
	for _, body := range noAuthRows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) == 0 {
			t.Errorf("expected findings for %q, got 0", body)
		}
	}
}

// TestCorrection05_PeriodBoundary_CaseIndependent verifies that period
// boundaries do not depend on capitalization. "Do not force-push.
// commit completed work." must split into two normative units just
// like "Do not force-push. Commit completed work." does.
func TestCorrection05_PeriodBoundary_CaseIndependent(t *testing.T) {
	// These MUST split into two units.
	rows := []string{
		"Do not force-push. Commit completed work.",
		"Do not force-push. commit completed work.",
		"Run make gate only when authorized by the current ACT. push the commit.",
		"Do not run make factorize. run make gate.",
	}
	for _, body := range rows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) == 0 {
			t.Errorf("expected split findings for %q", body)
		}
	}

	// These MUST preserve internal dots (no splitting).
	noSplitRows := []string{
		"docs/report.md",
		".factory/gate-summary.json",
		"version v1.2",
		"example.com",
		"foo.bar/baz",
	}
	for _, body := range noSplitRows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) != 0 {
			t.Errorf("expected no findings for %q, got %d", body, len(findings))
		}
	}
}

// TestCorrection05_OccurrenceCanonicalization verifies that aliases
// collapse into a single logical occurrence.
func TestCorrection05_OccurrenceCanonicalization(t *testing.T) {
	rows := []struct {
		body  string
		want  int // expected occurrences
	}{
		{"commit completed work", 1},
		{"push the commit", 1},
		{"create a tag", 1},
		{"make a commit", 1},
		{"make gate", 1},
		{"make factorize", 1},
		{"git push", 1},
	}
	for _, r := range rows {
		body := "Run " + r.body + " only when delegated by the current ACT."
		occs := findProtectedOccurrences(r.body)
		if len(occs) != r.want {
			t.Errorf("expected %d occurrence(s) for %q, got %d", r.want, r.body, len(occs))
		}
		// Verifying the umbrella test: a sub-clause with a named
		// ACT guard and a single logical occurrence is exempt.
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) != 0 {
			t.Errorf("expected no findings for %q, got %d", body, len(findings))
		}
	}

	// Repeated logical occurrences must be preserved.
	body := "make gate make gate"
	occs := findProtectedOccurrences(body)
	if len(occs) != 2 {
		t.Errorf("expected 2 occurrences for %q, got %d", body, len(occs))
	}
	body = "push the commit then push changes"
	occs = findProtectedOccurrences(body)
	if len(occs) != 2 {
		t.Errorf("expected 2 occurrences for %q, got %d", body, len(occs))
	}
}

// TestCorrection05_MultiOccurrenceFailClosed verifies that
// sub-clauses with multiple distinct protected operations fail
// closed even when a guard is present.
func TestCorrection05_MultiOccurrenceFailClosed(t *testing.T) {
	rows := []string{
		"Run make gate only when authorized by the current ACT, make gate after tests.",
		"Run make gate and make factorize only when the ACT authorizes make gate.",
		"Commit completed work, push the commit when the ACT delegates commit authority.",
		"Do not run make gate, run make gate.",
		"Do not push the commit, create a tag.",
		"Run make gate, push changes, create a tag when the ACT authorizes make gate.",
	}
	for _, body := range rows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) == 0 {
			t.Errorf("expected fail-closed findings for %q", body)
		}
	}
}

// TestCorrection05_NegationRules verifies that single-occurrence
// negation is accepted and multi-occurrence negation fails closed.
func TestCorrection05_NegationRules(t *testing.T) {
	// Single-occurrence negation is PASS.
	rows := []string{
		"Do not run make gate.",
		"Never force-push.",
		"Do not push the commit.",
		"Do not create a tag.",
	}
	for _, body := range rows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) != 0 {
			t.Errorf("expected no findings for %q, got %d", body, len(findings))
		}
	}

	// Multi-occurrence negation is FAIL CLOSED.
	failRows := []string{
		"Do not run make gate, run make gate.",
		"Do not force-push, commit completed work.",
		"Do not push the commit and create a tag.",
	}
	for _, body := range failRows {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) == 0 {
			t.Errorf("expected fail-closed findings for %q", body)
		}
	}
}

// TestCorrection05_OperationIdentity verifies that each protected
// directive produces exactly one operation kind, no phantom or
// alias-duplicate operations.
func TestCorrection05_OperationIdentity(t *testing.T) {
	rows := []struct {
		body string
		ops  map[ProtectedOpKind]int
	}{
		{"Create a tag.", map[ProtectedOpKind]int{OpGitTag: 1}},
		{"Create a commit.", map[ProtectedOpKind]int{OpGitCommit: 1}},
		{"Make a commit.", map[ProtectedOpKind]int{OpGitCommit: 1}},
		{"Run make gate.", map[ProtectedOpKind]int{OpMakeGate: 1}},
		{"Run make factorize.", map[ProtectedOpKind]int{OpMakeFactorize: 1}},
		{"Push changes.", map[ProtectedOpKind]int{OpGitPush: 1}},
		{"Tag the commit.", map[ProtectedOpKind]int{OpGitTag: 1}},
	}
	for _, r := range rows {
		occs := findProtectedOccurrences(r.body)
		got := make(map[ProtectedOpKind]int)
		for _, o := range occs {
			got[o.Op]++
		}
		if len(got) != len(r.ops) {
			t.Errorf("expected %d distinct ops for %q, got %d (got=%v want=%v)",
				len(r.ops), r.body, len(got), got, r.ops)
		}
		for op, count := range r.ops {
			if got[op] != count {
				t.Errorf("expected %d %s for %q, got %d", count, op, r.body, got[op])
			}
		}
	}
}

// TestCorrection05_TokenBoundary verifies that protected forms
// do not accidentally match across word boundaries.
func TestCorrection05_TokenBoundary(t *testing.T) {
	// These should NOT match protected forms.
	notProtected := []string{
		"run make gatekeeper",
		"run make gater",
		"run make gatecount",
		"CGO_ENABLED=0 make gate-fast",
		"git tags provide immutable lifecycle identity",
	}
	for _, body := range notProtected {
		findings := FindUnguardedProtectedOps("AGENTS.md", body)
		if len(findings) != 0 {
			t.Errorf("expected no findings for %q, got %d", body, len(findings))
		}
	}
}

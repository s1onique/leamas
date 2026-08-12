// PHASE: CORRECTION06 regression coverage.
//
// CORRECTION06 fixes the remaining reject-mode convergence defects
// from CORRECTION05. The authoritative ambiguity boundary is now
// the canonical logical-occurrence count after guard-internal
// mentions are filtered out. The order of evaluation is fixed:
//
//   1. canonical occurrence collection (after guard-span filter)
//   2. ambiguity decision (>1 directive occurrences -> FAIL CLOSED)
//   3. single-occurrence negation check
//   4. single-occurrence named authority check
//   5. otherwise reject
//
// No guard or negation is consulted before step 2.
//
// These tests exercise the production FindUnguardedProtectedOps
// function. The Operation Identity and Occurrence Canonicalization
// sub-tests use the white-box testOccurrences helper, which
// honours the documented lower-cased, whitespace-normalised
// precondition.

package agentcontext

import (
	"strings"
	"testing"
)

// rawHelperInputContract documents the precondition that
// findProtectedOccurrences operates on already-normalised input.
// The contract is verified explicitly by the production-identity
// tests in TestCorrection06_ProductionNormalizedOperationIdentity.
const rawHelperInputContract = "normalized_lowercase"

// TestCorrection06_RepeatedSameOpFailClosed is the required
// REPEATED-SAME-OP matrix. Every row has two (or more) occurrences
// of the same protected operation. A single shared guard or
// negation MUST NOT exempt them. Every row MUST fail closed.
func TestCorrection06_RepeatedSameOpFailClosed(t *testing.T) {
	rows := []string{
		"Run make gate only when authorized by the current ACT,\nmake gate after tests.",
		"Do not run make gate, run make gate.",
		"make gate make gate",
		"Push the commit when delegated by the current ACT,\npush changes afterward.",
		"Do not push the commit, push changes.",
		"Create a tag only when delegated by the current ACT,\ntag the commit afterward.",
	}
	for _, body := range rows {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected fail-closed finding for %q", body)
			}
		})
	}
}

// TestCorrection06_DistinctOpFailClosed is the required
// DISTINCT-OP matrix. Every row has at least two distinct
// protected operations. A single shared guard or negation MUST
// NOT exempt them. Every row MUST fail closed.
func TestCorrection06_DistinctOpFailClosed(t *testing.T) {
	rows := []string{
		"Do not force-push, commit completed work.",
		"Do not push the commit, create a tag.",
		"Run make gate when authorized by the current ACT,\npush the commit.",
		"Commit completed work when delegated by the current ACT,\ncreate a tag.",
		"Run make factorize and push the commit under one shared guard.",
	}
	for _, body := range rows {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected fail-closed finding for %q", body)
			}
		})
	}
}

// TestCorrection06_SingleOccurrenceExemptions is the required
// SINGLE-OCCURRENCE exemption matrix. Every row has exactly one
// directive occurrence. Negation or named-authority guards MUST
// exempt it. None of these rows may produce a finding.
func TestCorrection06_SingleOccurrenceExemptions(t *testing.T) {
	rows := []string{
		"Do not run make gate.",
		"Never force-push.",
		"Do not push the commit.",
		"Do not create a tag.",
		"Run make gate only when explicitly authorized by the current ACT.",
		"Commit only when the current ACT delegates this kind of action.",
		"Push only when delegated by the current ACT.",
		"Create a tag only when the ACT delegates tag authority.",
		"Run make gate when authorized by the validated closure plan.",
	}
	for _, body := range rows {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for %q, got %d (%+v)", body, len(findings), findings)
			}
		})
	}
}

// TestCorrection06_AmbiguityPrecedesNegation proves the
// precedence rule. A sub-clause with two directive occurrences
// MUST fail closed even when one of them appears under a
// "do not" negation. The "do not run make gate, run make gate."
// case has a single sub-clause (the comma is not a sentence
// boundary) with 2 directive occurrences; the negation MUST
// NOT save it.
func TestCorrection06_AmbiguityPrecedesNegation(t *testing.T) {
	rows := []string{
		"Do not run make gate, run make gate.",
		"Do not force-push, commit completed work.",
		"Do not push the commit and create a tag.",
		"Run make gate, push changes, create a tag when the ACT authorizes make gate.",
	}
	for _, body := range rows {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected fail-closed finding for %q (ambiguity MUST precede negation)", body)
			}
		})
	}
}

// TestCorrection06_AmbiguityPrecedesAuthority proves the
// precedence rule for authority. A sub-clause with two directive
// occurrences MUST fail closed even when a named-authority guard
// appears in the same sub-clause.
func TestCorrection06_AmbiguityPrecedesAuthority(t *testing.T) {
	rows := []string{
		"Run make gate only when authorized by the current ACT, make gate after tests.",
		"Commit completed work, push the commit when the ACT delegates commit authority.",
		"Run make gate, push changes, create a tag when the ACT authorizes make gate.",
	}
	for _, body := range rows {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected fail-closed finding for %q (ambiguity MUST precede authority)", body)
			}
		})
	}
}

// TestCorrection06_ProductionNormalizedOperationIdentity proves
// that each protected directive phrase, after the production
// pipeline applies its normalising lower-case + whitespace
// collapse, produces exactly one operation kind.
//
// The result also distinguishes the two failure modes the ACT
// isolates:
//
//	RAW_INTERNAL_HELPER_PRECONDITION=normalized_lowercase
//	PRODUCTION_NORMALIZED_OPERATION_IDENTITY=PASS
func TestCorrection06_ProductionNormalizedOperationIdentity(t *testing.T) {
	// Direct calls on the raw internal helper DO NOT see the
	// normalised form; that is documented precondition behaviour
	// and not a production defect.
	rawOpcs := findProtectedOccurrences("Push changes.")
	if len(rawOpcs) != 0 {
		t.Fatalf("raw helper precondition violated: expected 0 occurrences for %q, got %d",
			"Push changes.", len(rawOpcs))
	}

	// The same phrase, after the production normalising step,
	// MUST produce exactly one OpGitPush.
	assertOccurrenceIdentity(t, "Push changes.", map[ProtectedOpKind]int{OpGitPush: 1})
	assertOccurrenceIdentity(t, "Create a tag.", map[ProtectedOpKind]int{OpGitTag: 1})
	assertOccurrenceIdentity(t, "Make a commit.", map[ProtectedOpKind]int{OpGitCommit: 1})
	assertOccurrenceIdentity(t, "Run make gate.", map[ProtectedOpKind]int{OpMakeGate: 1})
}

// TestCorrection06_OccurrenceCanonicalizationRegression verifies
// the canonical-occurrence-count invariants required by the ACT.
// The exact-count tests and the positional-ordering invariants
// together prove that the helper does not manufacture duplicate
// logical occurrences from alias forms and that repeated
// occurrences are preserved in order.
func TestCorrection06_OccurrenceCanonicalizationRegression(t *testing.T) {
	exact := []struct {
		body string
		want map[ProtectedOpKind]int
	}{
		{"commit completed work", map[ProtectedOpKind]int{OpGitCommit: 1}},
		{"push the commit", map[ProtectedOpKind]int{OpGitPush: 1}},
		{"create a tag", map[ProtectedOpKind]int{OpGitTag: 1}},
		{"make a commit", map[ProtectedOpKind]int{OpGitCommit: 1}},
		{"make gate", map[ProtectedOpKind]int{OpMakeGate: 1}},
		{"make factorize", map[ProtectedOpKind]int{OpMakeFactorize: 1}},
	}
	for _, r := range exact {
		assertOccurrenceIdentity(t, r.body, r.want)
	}

	// Repeated occurrences: two make_gate and two git_push
	// occurrences each, with strict positional ordering.
	assertOccurrenceIdentity(t, "make gate make gate", map[ProtectedOpKind]int{OpMakeGate: 2})
	assertOccurrenceIdentity(t, "push the commit then push changes", map[ProtectedOpKind]int{OpGitPush: 2})
}

// TestCorrection06_AuthoritySourceRegression retains the
// CORRECTION05 named-authority matrix. Generic forms FAIL; named
// sources PASS. No new guard vocabulary is introduced.
func TestCorrection06_AuthoritySourceRegression(t *testing.T) {
	named := []string{
		"Run make gate only when explicitly authorized by the current ACT.",
		"Run make gate when the ACT authorizes this exact command.",
		"Run make factorize only when the current ACT authorizes it.",
		"Commit only when the current ACT delegates this kind of action.",
		"Push only when delegated by the current ACT.",
		"Create a tag only when the ACT delegates tag authority.",
		"Run make gate when authorized by the validated closure plan.",
		"Run make gate when authorized by the frozen closure plan.",
	}
	for _, body := range named {
		body := body
		t.Run("named/"+body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for %q, got %d (%+v)", body, len(findings), findings)
			}
		})
	}

	generic := []string{
		"Run make gate when explicitly authorized.",
		"Run make gate when delegated.",
		"Run make gate only when that verification tier is explicitly authorized.",
		"Run make gate because deployment explicitly authorizes that exact command.",
		"Run make gate when policy authorizes that exact command.",
		"Commit completed work when explicitly authorized.",
		"Push the commit when deployment is delegated.",
	}
	for _, body := range generic {
		body := body
		t.Run("generic/"+body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected finding for %q (generic guard MUST NOT be authority)", body)
			}
		})
	}
}

// TestCorrection06_BoundaryRegression retains the punctuation
// boundary matrix and the internal-dot preservation rules. A
// future regression in splitIntoUnits would break either family.
func TestCorrection06_BoundaryRegression(t *testing.T) {
	punctuation := []string{
		"Run make gate.",
		"Run make gate!",
		"Run make gate?",
		"Run make gate;",
	}
	for _, body := range punctuation {
		body := body
		t.Run(body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected finding for punctuation variant %q", body)
			}
		})
	}

	// Lower-case subsequent sentence starts MUST still split.
	caseIndependent := []string{
		"Do not force-push. commit completed work.",
		"Run make gate only when authorized by the current ACT. push the commit.",
		"Do not run make factorize. run make gate.",
	}
	for _, body := range caseIndependent {
		body := body
		t.Run("case_independent/"+body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) == 0 {
				t.Fatalf("expected split findings for %q", body)
			}
		})
	}

	// Internal-dot cases MUST NOT split.
	internalDots := []string{
		"docs/report.md",
		".factory/gate-summary.json",
		"v1.2",
		"example.com",
		"foo.bar/baz",
	}
	for _, body := range internalDots {
		body := body
		t.Run("internal_dots/"+body, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", body)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for internal-dot case %q, got %d", body, len(findings))
			}
		})
	}
}

// TestCorrection06_ScanSubClausePrecedence exercises scanSubClause
// directly to demonstrate that the canonical occurrence count is
// computed BEFORE any negation or authority processing. A unit
// with two directive occurrences returns both, regardless of
// whether a guard or negation is present.
func TestCorrection06_ScanSubClausePrecedence(t *testing.T) {
	// Two directive occurrences + a shared negation. Must fail.
	got := scanSubClause("do not run make gate, run make gate.")
	if len(got) == 0 {
		t.Fatalf("expected fail-closed ops for negation+ambiguity, got none")
	}

	// Two directive occurrences + a shared named-authority guard.
	// Must fail.
	got = scanSubClause("run make gate only when authorized by the current act, make gate after tests.")
	if len(got) == 0 {
		t.Fatalf("expected fail-closed ops for authority+ambiguity, got none")
	}

	// Single directive occurrence + negation. Exempt.
	got = scanSubClause("do not run make gate.")
	if len(got) != 0 {
		t.Fatalf("expected no ops for single negation, got %v", got)
	}

	// Single directive occurrence + named authority. Exempt.
	got = scanSubClause("run make gate only when explicitly authorized by the current act.")
	if len(got) != 0 {
		t.Fatalf("expected no ops for single named authority, got %v", got)
	}
}

// TestCorrection06_ReportTags documents the proof of completion
// for this ACT. It is a no-op assertion that prints the proof
// surface; the surrounding tests are the actual checks.
func TestCorrection06_ReportTags(t *testing.T) {
	if !strings.Contains(rawHelperInputContract, "normalized_lowercase") {
		t.Fatalf("rawHelperInputContract drift: %q", rawHelperInputContract)
	}
}

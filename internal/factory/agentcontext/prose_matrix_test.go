package agentcontext

import (
	"testing"
)

// PHASE 7: Operation-binding matrix.
//
// Each row MUST FAIL with a per-operation finding. The required
// CASES list proves that a guard or negation for one protected
// operation does NOT authorize or negate another operation in the
// same unit.

func TestFindUnguardedProtectedOps_OperationBindingMatrix(t *testing.T) {
	cases := []string{
		"Run make gate and commit only when the ACT delegates commit authority.",
		"Run make factorize and push only when explicitly delegated by the current ACT.",
		"Create a tag and run make gate only when make gate is authorized.",
		"Push the commit and run make factorize only when push authority is delegated.",
		"Run make gate and push the commit.",
		"Do not force-push but commit completed work.",
		"Do not run make factorize and run make gate.",
		"Do not run make gate-dupcode and run make gate.",
		"Do not git push; commit completed work.",
		"Never create a tag but push the commit.",
		"Push only when delegated and tag the commit.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected operation-binding finding for %q", p)
			}
		})
	}
}

// PHASE 8: Prefix metamorphic matrix.

func TestFindUnguardedProtectedOps_PrefixMatrix(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Always run make gate.",
		"Please run make gate.",
		"Then run make gate.",
		"Next, run make gate.",
		"Now run make gate.",
		"Finally run make gate.",
		"Before finishing, run make gate.",
		"After tests, run make gate.",
		"For final validation, run make gate.",
		"You must run make gate.",
		"You should run make gate.",
		"Agents must run make gate.",
		"Run make factorize.",
		"Always run make factorize.",
		"Run make gate-dupcode.",
		"Always run make gate-dupcode.",
		"Commit completed work.",
		"Always commit completed work.",
		"Push changes.",
		"Always push changes.",
		"Create a tag.",
		"Always create a tag.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected prefix finding for %q", p)
			}
		})
	}
}

// PHASE 9: Subordinate-description matrix.

func TestFindUnguardedProtectedOps_SubordinateDescriptionMatrix(t *testing.T) {
	directiveCases := []string{
		"Run make gate because it is required.",
		"make gate because it is required.",
		"Execute make factorize after the repository is clean.",
		"Commit completed work after the tree is verified.",
		"Push the commit if the repository is ready.",
	}
	for _, p := range directiveCases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected directive finding for %q", p)
			}
		})
	}
	descriptiveCases := []string{
		"make gate is Tier-3 verification.",
		"make factorize is expensive.",
		"Git commit is a publication boundary.",
		"Git push is disabled by default.",
		"Git tags are immutable lifecycle identities.",
	}
	for _, p := range descriptiveCases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for descriptive prose %q, got: %+v", p, findings)
			}
		})
	}
}

// PHASE 10: Token-boundary tests.

func TestFindUnguardedProtectedOps_RespectsTokenBoundary(t *testing.T) {
	cases := []string{
		"CGO_ENABLED=0 make gate-fast",
		"run make gater",
		"run make gatekeeper",
		"run make gatecount",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for non-form token %q, got: %+v", p, findings)
			}
		})
	}
}

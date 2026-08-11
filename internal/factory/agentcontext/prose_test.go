package agentcontext

import (
	"testing"
)

// PHASE 1: Inline backtick preservation.
// Backticks MUST NOT bypass protected-operation detection.

func TestFindUnguardedProtectedOps_BacktickedGateFails(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Run `make gate`.",
		"Run  `make gate`  .",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_BacktickedFactorizeFails(t *testing.T) {
	cases := []string{
		"Execute make factorize.",
		"Execute `make factorize`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_BacktickedGateDupcodeFails(t *testing.T) {
	cases := []string{
		"Run make gate-dupcode.",
		"Run `make gate-dupcode`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_BacktickedCommitFails(t *testing.T) {
	cases := []string{
		"Run git commit.",
		"Run `git commit`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_BacktickedPushFails(t *testing.T) {
	cases := []string{
		"Execute git push.",
		"Execute `git push`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_BacktickedTagFails(t *testing.T) {
	cases := []string{
		"Run git tag.",
		"Run `git tag`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

// Fenced code blocks remain an excluded region.
func TestFindUnguardedProtectedOps_FencedCodeBlockTolerated(t *testing.T) {
	content := "Examples that are NOT run automatically:\n\n```\nrun make gate\ncommit the changes\npush the commit\n```\n"
	findings := FindUnguardedProtectedOps("AGENTS.md", content)
	if len(findings) != 0 {
		t.Fatalf("expected no findings inside fenced code block, got: %+v", findings)
	}
}

// PHASE 2: Per-operation guard/negation binding.

func TestFindUnguardedProtectedOps_DoNotDoesNotAuthorizeUnrelatedRun(t *testing.T) {
	para := "Do not skip tests. Run make gate."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected unguarded finding for %q (do-not in unit 1 must not authorize make gate in unit 2)", para)
	}
}

func TestFindUnguardedProtectedOps_CommitGuardDoesNotAuthorizeMakeGate(t *testing.T) {
	para := "Run make gate. Commit only when the ACT delegates commit authority."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected unguarded finding for %q (commit guard must not authorize make gate)", para)
	}
}

func TestFindUnguardedProtectedOps_NeverForcePushDoesNotAuthorizePush(t *testing.T) {
	para := "Never force-push. Push the commit."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected unguarded finding for %q (force-push negation must not authorize push)", para)
	}
}

func TestFindUnguardedProtectedOps_CommitGuardDoesNotAuthorizePush(t *testing.T) {
	para := "Commit completed work. Push only when explicitly delegated by the current ACT."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected unguarded finding for %q (commit guard must not authorize push)", para)
	}
}

// PHASE 3: Imperative intent required.

func TestFindUnguardedProtectedOps_DescriptiveProseAccepted(t *testing.T) {
	cases := []string{
		"The repository gate validates canonical state.",
		"The make gate command is Tier-3 verification.",
		"make factorize is a Tier-3 command.",
		"Git commit is a publication boundary.",
		"Git push is publication.",
		"Git tags provide immutable lifecycle identity.",
		"History rewrite is forbidden by persistent context.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for descriptive prose %q, got: %+v", p, findings)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_ImperativeFail(t *testing.T) {
	cases := []string{
		"Run the repository gate.",
		"Execute make gate.",
		"Use make factorize.",
		"Commit completed work.",
		"Push changes.",
		"Create a tag.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

// PHASE 4: Guarded equivalents accepted.

func TestFindUnguardedProtectedOps_GuardedAccepts(t *testing.T) {
	cases := []string{
		"Run make gate only when explicitly authorized by the current ACT.",
		"When delegated by the current ACT, run make factorize.",
		"Do not run make gate-dupcode unless the current ACT explicitly authorizes that exact command.",
		"Commit only when the ACT delegates commit authority.",
		"Push only when explicitly delegated by the current ACT.",
		"Create a tag only when the ACT delegates tag authority.",
		"Run the repository gate only when that verification tier is explicitly authorized.",
		"Do not run make gate.",
		"Never execute make factorize without explicit ACT authority.",
		"Do not git push.",
		"Do not create a tag.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for guarded %q, got: %+v", p, findings)
			}
		})
	}
}

// PHASE 5: Backticked guarded equivalents also accepted.

func TestFindUnguardedProtectedOps_BacktickedGuardedAccepts(t *testing.T) {
	cases := []string{
		"Run `make gate` only when explicitly authorized by the current ACT.",
		"Run `make factorize` only when explicitly authorized by the current ACT.",
		"Run `make gate-dupcode` only when explicitly authorized by the current ACT.",
		"Run `git push` only when delegated by the current ACT.",
		"Run `git commit` only when the ACT delegates commit authority.",
		"Run `git tag` only when the ACT delegates tag authority.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for backticked guarded %q, got: %+v", p, findings)
			}
		})
	}
}

// PHASE 6: Multi-operation ambiguity fails closed.

func TestFindUnguardedProtectedOps_MultiOperationAmbiguityFails(t *testing.T) {
	para := "Run make gate and push changes only when push is authorized."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected unguarded finding for mixed-authority %q", para)
	}
}

// PHASE 7: Metamorphic variation.

func TestFindUnguardedProtectedOps_MetamorphicCapitalization(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Run MAKE GATE.",
		"Run Make Gate.",
		"RUN MAKE GATE.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_MetamorphicWhitespace(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Run  make  gate.",
		"  Run make gate.  ",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_MetamorphicBullet(t *testing.T) {
	cases := []string{
		"- Run make gate.",
		"* Execute make gate.",
		"1. Run `make gate`.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

func TestFindUnguardedProtectedOps_MetamorphicPunctuation(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Run make gate!",
		"Run make gate?",
		"Run make gate;",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) == 0 {
				t.Fatalf("expected unguarded finding for %q", p)
			}
		})
	}
}

// PHASE 8: Soft-wrapped paragraphs are joined.

func TestFindUnguardedProtectedOps_SoftWrappedParagraph(t *testing.T) {
	// Strongly-typed imperative protected operation that starts on
	// one line and continues on the next line in a soft-wrapped
	// paragraph.
	para := "Never move or force-push ACT tags.\nAnd never delete the manifest."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	// "Never move or force-push ACT tags" has "never " (negation),
	// so it is NOT flagged. "And never delete the manifest" has no
	// protected op, so it is NOT flagged.
	if len(findings) != 0 {
		t.Fatalf("expected no findings for fully-negated paragraph, got: %+v", findings)
	}
}

func TestFindUnguardedProtectedOps_SoftWrappedImperative(t *testing.T) {
	// An imperative that is split across soft-wrapped lines must
	// still be flagged.
	para := "Run make gate\nnow and commit completed work\nlater."
	findings := FindUnguardedProtectedOps("AGENTS.md", para)
	if len(findings) == 0 {
		t.Fatalf("expected finding for soft-wrapped imperative %q", para)
	}
}

// PHASE 9: helpers.

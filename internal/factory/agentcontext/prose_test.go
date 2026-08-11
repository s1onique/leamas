package agentcontext

import (
	"strings"
	"testing"
)

// unguardedSentence returns a paragraph that contains a protected
// operation framed by an imperative verb and lacks any guard phrase.
func unguardedSentence(op string) string {
	return "Run " + op + " for final validation."
}

// guardedSentence returns a paragraph that contains a protected
// operation framed by an imperative verb and includes a guard phrase.
func guardedSentence(op string) string {
	return "Run " + op + " only when explicitly authorized by the current ACT."
}

func TestFindUnguardedProtectedOps_UnguardedGateFails(t *testing.T) {
	cases := []string{
		"Run make gate on every task.",
		"Execute make gate after implementation.",
		"Invoke make gate before finishing.",
		"Use make gate for final validation.",
		"Run the make gate step.",
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

func TestFindUnguardedProtectedOps_UnguardedFactorizeFails(t *testing.T) {
	cases := []string{
		"Run make factorize after editing.",
		"Execute make factorize before completion.",
		"Invoke make factorize to confirm results.",
		"Use make factorize for canonical authority.",
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

func TestFindUnguardedProtectedOps_UnguardedGateDupcodeFails(t *testing.T) {
	cases := []string{
		"Run make gate-dupcode after dupcode changes.",
		"Execute make gate-dupcode when code is ready.",
		"Invoke make gate-dupcode to validate.",
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

func TestFindUnguardedProtectedOps_UnguardedRepositoryGateFails(t *testing.T) {
	cases := []string{
		"Run the repository gate.",
		"Execute the repository gate after GREEN.",
		"Use the repository gate for final validation.",
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

func TestFindUnguardedProtectedOps_UnguardedCommitFails(t *testing.T) {
	cases := []string{
		"Commit completed work.",
		"Commit the changes when finished.",
		"Create a commit after tests.",
		"Make a commit for successful tasks.",
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

func TestFindUnguardedProtectedOps_UnguardedPushFails(t *testing.T) {
	cases := []string{
		"Push the commit.",
		"Push changes after verification.",
		"Push successful work automatically.",
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

func TestFindUnguardedProtectedOps_UnguardedTagFails(t *testing.T) {
	cases := []string{
		"Tag successful ACTs.",
		"Create a tag after committing.",
		"Tag the commit after merge.",
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

func TestFindUnguardedProtectedOps_GuardedAccepts(t *testing.T) {
	cases := []string{
		"Run make gate only when explicitly authorized by the current ACT.",
		"When delegated by the current ACT, run make factorize.",
		"Do not run make gate-dupcode unless the current ACT explicitly authorizes that exact command.",
		"Commit only when the ACT delegates commit authority.",
		"Push only when explicitly delegated by the current ACT.",
		"Create a tag only when the current ACT delegates tag authority.",
		"Run the repository gate only when that verification tier is explicitly authorized.",
	}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for %q, got: %+v", p, findings)
			}
		})
	}
}

// TestFindUnguardedProtectedOps_MetamorphicCapitalization varies
// capitalization of the protected operation. The dangerous unguarded
// variants MUST fail regardless of capitalization.
func TestFindUnguardedProtectedOps_MetamorphicCapitalization(t *testing.T) {
	cases := []string{
		"Run make gate.",
		"Run MAKE GATE.",
		"Run Make Gate.",
		"Run make Gate.",
		"Run MAKE gate.",
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

// TestFindUnguardedProtectedOps_MetamorphicBackticks verifies that
// the scanner handles inline backticks correctly. Inside backticks,
// the protected operation is treated as a code example and is NOT
// flagged. Outside backticks, the same imperative line is flagged.
func TestFindUnguardedProtectedOps_MetamorphicBackticks(t *testing.T) {
	t.Run("inline_backticks_in_code_span_tolerated", func(t *testing.T) {
		// Plain prose mentioning `make gate` inside inline backticks
		// describes a code example and should not be flagged.
		content := "Use the `make gate` command when it is authorized."
		findings := FindUnguardedProtectedOps("AGENTS.md", content)
		if len(findings) != 0 {
			t.Fatalf("expected no findings for inline backtick example, got: %+v", findings)
		}
	})

	t.Run("fenced_code_block_tolerated", func(t *testing.T) {
		// Example commands inside a fenced block are not flagged.
		content := "Examples that are NOT run automatically:\n\n```\nrun make gate\ncommit the changes\npush the commit\n```\n"
		findings := FindUnguardedProtectedOps("AGENTS.md", content)
		if len(findings) != 0 {
			t.Fatalf("expected no findings inside fenced code block, got: %+v", findings)
		}
	})

	t.Run("unguarded_outside_backticks_still_fails", func(t *testing.T) {
		// The paragraph OUTSIDE the backticks still must be flagged.
		content := "Run make gate now."
		findings := FindUnguardedProtectedOps("AGENTS.md", content)
		if len(findings) == 0 {
			t.Fatalf("expected unguarded finding outside backticks")
		}
	})
}

// TestFindUnguardedProtectedOps_MetamorphicPunctuation verifies that
// terminal punctuation and extra whitespace do not defeat detection.
func TestFindUnguardedProtectedOps_MetamorphicPunctuation(t *testing.T) {
	cases := []string{
		"Run make gate",
		"Run  make  gate",
		"Run make gate.",
		"Run make gate!",
		"Run make gate;",
		"Run make gate?",
		" - Run make gate.",
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

// TestFindUnguardedProtectedOps_MetamorphicBulletForm varies the
// Markdown bullet form. The unprotected operation is still flagged.
func TestFindUnguardedProtectedOps_MetamorphicBulletForm(t *testing.T) {
	cases := []string{
		"- Run make gate.",
		"* Run make gate.",
		"1. Run make gate.",
		"Run make gate.",
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

func TestFindUnguardedProtectedOps_ContractBlockStripped(t *testing.T) {
	// Mentions of protected operations inside the contract block
	// (e.g. the `commit=explicit` line) MUST NOT be flagged.
	c := minimalValidBlock()
	findings := FindUnguardedProtectedOps("AGENTS.md", "Here is the contract.\n\n"+c+"\n\nNo prose contains the protected operation.\n")
	if len(findings) != 0 {
		t.Fatalf("expected no findings; contract block must be stripped, got: %+v", findings)
	}
}

func TestFindUnguardedProtectedOps_NoProtectedOpAccepts(t *testing.T) {
	// Plain prose that does not mention a protected operation is
	// accepted regardless of phrasing.
	benign := []string{
		"Follow the current ACT.",
		"Use Bash for tiny glue.",
		"Verify changes before completing the task.",
		"Apply consistent style across the codebase.",
	}
	for _, p := range benign {
		p := p
		t.Run(p, func(t *testing.T) {
			findings := FindUnguardedProtectedOps("AGENTS.md", p)
			if len(findings) != 0 {
				t.Fatalf("expected no findings for benign prose %q", p)
			}
		})
	}
}

func TestParagraphHasImperativeVerb(t *testing.T) {
	if !ParagraphHasImperativeVerb(strings.ToLower(unguardedSentence("make gate"))) {
		t.Fatalf("expected verb detection")
	}
	if ParagraphHasImperativeVerb("benign prose with no imperative verb") {
		t.Fatalf("did not expect verb detection")
	}
}
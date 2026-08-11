package agentcontext

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// matrixCase is one row of the authority-delegation matrix.
//
// WantPass indicates whether the fixture should produce zero findings.
// The fixture is built by starting from a known-valid AGENTS.md and
// .clinerules/leamas.md and then applying a single textual mutation
// (or no mutation).
type matrixCase struct {
	Name     string
	Mutate   func(content string) string
	WantPass bool
}

// TestCheckRepo_AuthorityDelegationMatrix exercises the production
// CheckRepo function against a focused set of adversarial fixtures.
//
// Every PASS case exercises the real production verifier; every FAIL
// case asserts that an adversarial or unguarded mutation is rejected
// deterministically.
func TestCheckRepo_AuthorityDelegationMatrix(t *testing.T) {
	cases := []matrixCase{
		// PASS: valid guarded fixtures.
		{
			Name:     "valid_guarded_agents_and_cline",
			Mutate:   func(s string) string { return s },
			WantPass: true,
		},

		// PASS: guarded mention of make factorize (already covered
		// by the valid fixture, but here we tighten by also checking
		// the .clinerules case below).

		// FAIL: missing the explicit-authority rule.
		{
			Name: "missing_explicit_authority_rule",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "Follow the current ACT's explicit authority.", "")
			},
			WantPass: false,
		},

		// FAIL: missing the NOT RUN rule.
		{
			Name: "missing_not_run_rule",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "report the command as not run", "report the command status")
			},
			WantPass: false,
		},

		// FAIL: missing the commit-delegation rule.
		{
			Name: "missing_commit_delegation_rule",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "Do not infer commit authority from permission to edit or test.", "")
			},
			WantPass: false,
		},

		// FAIL: missing the push-delegation rule.
		{
			Name: "missing_push_delegation_rule",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "Do not infer push authority from permission to commit.", "")
			},
			WantPass: false,
		},

		// FAIL: unguarded make factorize.
		{
			Name: "unguarded_make_factorize",
			Mutate: func(s string) string {
				return s + "\n- Always run make factorize.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded make gate-dupcode.
		{
			Name: "unguarded_make_gate_dupcode",
			Mutate: func(s string) string {
				return s + "\n- Always run make gate-dupcode before commit.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded make gate.
		{
			Name: "unguarded_make_gate",
			Mutate: func(s string) string {
				return s + "\n- Always run make gate.\n"
			},
			WantPass: false,
		},

		// FAIL: "always commit when tests pass".
		{
			Name: "auto_commit_when_tests_pass",
			Mutate: func(s string) string {
				return s + "\n- Always commit when tests pass.\n"
			},
			WantPass: false,
		},

		// FAIL: "commit all changes after task".
		{
			Name: "commit_all_changes_after",
			Mutate: func(s string) string {
				return s + "\n- Commit all changes after task completion.\n"
			},
			WantPass: false,
		},

		// FAIL: "push successful work automatically".
		{
			Name: "push_successful_work_automatically",
			Mutate: func(s string) string {
				return s + "\n- Push successful work automatically.\n"
			},
			WantPass: false,
		},

		// FAIL: "make gate before every commit".
		{
			Name: "make_gate_before_every_commit",
			Mutate: func(s string) string {
				return s + "\n- Run make gate before every commit.\n"
			},
			WantPass: false,
		},

		// FAIL: "make factorize before every commit".
		{
			Name: "make_factorize_before_every_commit",
			Mutate: func(s string) string {
				return s + "\n- Run make factorize before every commit.\n"
			},
			WantPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			tmp := t.TempDir()
			writeFixture(t, tmp, "AGENTS.md", tc.Mutate(validAgentsMD()))
			writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), tc.Mutate(validClineMD()))
			writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"),
				"# Agent Context Files\nLeamas uses checked-in agent context files.\n")

			findings, err := CheckRepo(tmp)
			if err != nil {
				t.Fatalf("CheckRepo: %v", err)
			}

			gotPass := len(findings) == 0
			if gotPass != tc.WantPass {
				var summary []string
				for _, f := range findings {
					summary = append(summary, fmt.Sprintf("%s:%s:%s", f.Path, f.Kind, f.Message))
				}
				t.Fatalf("want_pass=%v got_pass=%v findings=%s",
					tc.WantPass, gotPass, strings.Join(summary, "\n"))
			}
		})
	}
}

// TestCheckRepo_ClineMatrix exercises the same matrix but only against
// the .clinerules/leamas.md fixture. This proves that the Cline rule
// alone enforces its own contract independently of AGENTS.md.
func TestCheckRepo_ClineMatrix(t *testing.T) {
	cases := []struct {
		Name     string
		Mutate   func(content string) string
		WantPass bool
	}{
		{
			Name:     "valid_guarded_cline",
			Mutate:   func(s string) string { return s },
			WantPass: true,
		},
		{
			Name: "missing_cline_execution_authority",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "## Execution Authority\n\nThe current ACT is authoritative.\n\n", "")
			},
			WantPass: false,
		},
		{
			Name: "missing_not_run_rule_in_cline",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "report it as not run", "report it as a failure")
			},
			WantPass: false,
		},
		{
			Name: "unguarded_make_factorize_in_cline",
			Mutate: func(s string) string {
				return s + "\n- Always run make factorize.\n"
			},
			WantPass: false,
		},
		{
			Name: "unguarded_make_gate_in_cline",
			Mutate: func(s string) string {
				return s + "\n- Always run make gate.\n"
			},
			WantPass: false,
		},
		{
			Name: "auto_commit_in_cline",
			Mutate: func(s string) string {
				return s + "\n- Always commit when tests pass.\n"
			},
			WantPass: false,
		},
		{
			Name: "push_successful_work_in_cline",
			Mutate: func(s string) string {
				return s + "\n- Push successful work automatically.\n"
			},
			WantPass: false,
		},
		{
			Name: "force_push_successful_in_cline",
			Mutate: func(s string) string {
				return s + "\n- Force-push successful work.\n"
			},
			WantPass: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			tmp := t.TempDir()
			writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
			writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), tc.Mutate(validClineMD()))
			writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"),
				"# Agent Context Files\nLeamas uses checked-in agent context files.\n")

			findings, err := CheckRepo(tmp)
			if err != nil {
				t.Fatalf("CheckRepo: %v", err)
			}

			gotPass := len(findings) == 0
			if gotPass != tc.WantPass {
				var summary []string
				for _, f := range findings {
					summary = append(summary, fmt.Sprintf("%s:%s:%s", f.Path, f.Kind, f.Message))
				}
				t.Fatalf("want_pass=%v got_pass=%v findings=%s",
					tc.WantPass, gotPass, strings.Join(summary, "\n"))
			}
		})
	}
}

// TestCheckRepo_CheckpointDistinguished verifies that the checkpoint
// keyword, when paired with the explicit "not Git commits" rule, is
// accepted; when paired with an unguarded auto-commit clause it is
// rejected. This guards the D3 doctrine directly.
func TestCheckRepo_CheckpointDistinguished(t *testing.T) {
	t.Run("checkpoint_distinguished_passes", func(t *testing.T) {
		tmp := t.TempDir()
		writeFixture(t, tmp, "AGENTS.md", validAgentsMD())
		writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
		writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

		findings, err := CheckRepo(tmp)
		if err != nil {
			t.Fatalf("CheckRepo: %v", err)
		}
		if len(findings) != 0 {
			t.Fatalf("expected PASS, got findings: %+v", findings)
		}
	})

	t.Run("checkpoint_with_auto_commit_fails", func(t *testing.T) {
		tmp := t.TempDir()
		original := "Successful tests do not imply commit authority. Commit only when the ACT delegates commit authority."
		replacement := original + "\n\nEditor checkpoints imply commit authority.\n- Always commit when tests pass."
		bad := strings.ReplaceAll(validAgentsMD(), original, replacement)
		writeFixture(t, tmp, "AGENTS.md", bad)
		writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
		writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

		findings, err := CheckRepo(tmp)
		if err != nil {
			t.Fatalf("CheckRepo: %v", err)
		}
		if len(findings) == 0 {
			t.Fatalf("expected FAIL because 'always commit when tests pass' is unguarded")
		}
	})
}

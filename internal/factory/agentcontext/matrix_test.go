package agentcontext

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// matrixCase is one row of the authority-delegation matrix.
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

		// FAIL: contract block removed.
		{
			Name: "contract_block_removed",
			Mutate: func(s string) string {
				return removeContractBlock(s)
			},
			WantPass: false,
		},

		// FAIL: contract has wrong value.
		{
			Name: "contract_commit_implicit",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "commit=explicit", "commit=act")
			},
			WantPass: false,
		},

		// FAIL: contract has unknown key.
		{
			Name: "contract_unknown_key",
			Mutate: func(s string) string {
				return strings.Replace(s, "<!-- LEAMAS:AUTHORITY-CONTRACT:END -->", "rogue=true\n<!-- LEAMAS:AUTHORITY-CONTRACT:END -->", 1)
			},
			WantPass: false,
		},

		// FAIL: contract has duplicate key.
		{
			Name: "contract_duplicate_key",
			Mutate: func(s string) string {
				return strings.Replace(s, "commit=explicit", "commit=explicit\ncommit=explicit", 1)
			},
			WantPass: false,
		},

		// FAIL: unguarded make gate.
		{
			Name: "unguarded_make_gate_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nRun make gate now.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded make factorize.
		{
			Name: "unguarded_make_factorize_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nExecute make factorize.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded make gate-dupcode.
		{
			Name: "unguarded_make_gate_dupcode_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nRun make gate-dupcode.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded repository gate.
		{
			Name: "unguarded_repository_gate_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nRun the repository gate.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded commit.
		{
			Name: "unguarded_commit_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nCommit completed work.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded push.
		{
			Name: "unguarded_push_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nPush the commit.\n"
			},
			WantPass: false,
		},

		// FAIL: unguarded tag.
		{
			Name: "unguarded_tag_in_prose",
			Mutate: func(s string) string {
				return s + "\n\nTag successful ACTs.\n"
			},
			WantPass: false,
		},

		// FAIL: forced run.
		{
			Name: "forced_run_with_no_contract",
			Mutate: func(s string) string {
				return strings.ReplaceAll(s, "commit=explicit", "commit=act") +
					"\n\nRun make gate.\n"
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

// TestCheckRepo_CrossFileContractMismatch applies a different
// mutation to AGENTS.md and .clinerules/leamas.md so the shared
// contract semantics disagree. The verifier MUST reject this even
// when each file alone is well-formed.
func TestCheckRepo_CrossFileContractMismatch(t *testing.T) {
	tmp := t.TempDir()

	// AGENTS.md uses commit=explicit (canonical).
	agentsContent := validAgentsMD()
	// .clinerules/leamas.md uses commit=act (mutated shared semantics).
	clineContent := strings.ReplaceAll(validClineMD(), "commit=explicit", "commit=act")

	writeFixture(t, tmp, "AGENTS.md", agentsContent)
	writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), clineContent)
	writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"),
		"# Agent Context Files\nLeamas uses checked-in agent context files.\n")

	findings, err := CheckRepo(tmp)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	// Both files have valid contracts, so the per-file malformed /
	// semantic-invalid findings should NOT be present. However, the
	// shared-semantics mismatch MUST be reported.
	foundMismatch := false
	for _, f := range findings {
		if f.Kind == "contract_shared_semantics_mismatch" {
			foundMismatch = true
			break
		}
	}
	if !foundMismatch {
		t.Fatalf("expected contract_shared_semantics_mismatch finding, got: %+v", findings)
	}
}

// TestCheckRepo_ClineMatrix exercises the same matrix but only
// mutates the .clinerules/leamas.md fixture. This proves that the
// Cline file alone enforces its own contract independently.
func TestCheckRepo_ClineMatrix(t *testing.T) {
	cases := []matrixCase{
		{
			Name:     "valid_guarded_cline",
			Mutate:   func(s string) string { return s },
			WantPass: true,
		},
		{
			Name: "contract_block_removed_in_cline",
			Mutate: func(s string) string {
				return removeContractBlock(s)
			},
			WantPass: false,
		},
		{
			Name: "unguarded_make_gate_in_cline",
			Mutate: func(s string) string {
				return s + "\n\nRun make gate.\n"
			},
			WantPass: false,
		},
		{
			Name: "unguarded_commit_in_cline",
			Mutate: func(s string) string {
				return s + "\n\nCommit completed work.\n"
			},
			WantPass: false,
		},
		{
			Name: "unguarded_push_in_cline",
			Mutate: func(s string) string {
				return s + "\n\nPush the commit.\n"
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
// keyword, when paired with a valid contract and bounded prose, is
// accepted; when paired with an unguarded auto-commit clause it is
// rejected.
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

	t.Run("unguarded_commit_in_prose_fails", func(t *testing.T) {
		tmp := t.TempDir()
		bad := strings.ReplaceAll(validAgentsMD(),
			"Successful tests do not imply commit authority. Commit only when the ACT delegates commit authority.",
			"Editor checkpoints imply commit authority.\n\nCommit completed work.")
		writeFixture(t, tmp, "AGENTS.md", bad)
		writeFixture(t, tmp, filepath.Join(".clinerules", "leamas.md"), validClineMD())
		writeFixture(t, tmp, filepath.Join("docs", "factory", "agent-context-files.md"), "policy")

		findings, err := CheckRepo(tmp)
		if err != nil {
			t.Fatalf("CheckRepo: %v", err)
		}
		if len(findings) == 0 {
			t.Fatalf("expected FAIL because 'Commit completed work.' is unguarded")
		}
	})
}

// removeContractBlock removes the structured authority contract from
// the supplied content. Useful for adversarial mutation tests.
func removeContractBlock(content string) string {
	beginIdx := strings.Index(content, ContractBeginMarker)
	endIdx := strings.Index(content, ContractEndMarker)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return content
	}
	return content[:beginIdx] + content[endIdx+len(ContractEndMarker):]
}

func TestRemoveContractBlock(t *testing.T) {
	c := minimalValidBlock()
	got := removeContractBlock(c)
	if strings.Contains(got, ContractBeginMarker) || strings.Contains(got, ContractEndMarker) {
		t.Fatalf("contract markers not removed: %s", got)
	}
}

package agentcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// findRepoRoot walks up from the test directory until it finds a
// directory that contains both AGENTS.md and .clinerules/leamas.md.
func findRepoRoot(start string) (string, error) {
	dir := start
	for i := 0; i < 16; i++ {
		agents := filepath.Join(dir, "AGENTS.md")
		cline := filepath.Join(dir, ".clinerules", "leamas.md")
		if _, err := os.Stat(agents); err == nil {
			if _, err := os.Stat(cline); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root from %s", start)
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find repo root from %s (walk limit reached)", start)
}

// TestAgentContextAuthorityDelegationContract is the umbrella test
// for ACT-LEAMAS-FACTORY-AGENT-AUTHORITY-DELEGATION-CONTRACT01-CORRECTION01.
//
// It reads the real repository AGENTS.md and .clinerules/leamas.md
// and runs them through the production CheckRepo function. It MUST
// fail if a future edit reintroduces an unguarded imperative grant of
// a protected operation, an implicit commit/push/tag authority, or a
// shared-semantics mismatch between the two files.
//
// The test is intentionally repository-relative and does NOT hard-code
// any specific Git commit SHA.
func TestAgentContextAuthorityDelegationContract(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, err := findRepoRoot(wd)
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	// 1) Real production CheckRepo.
	findings, err := CheckRepo(root)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("real repo agent-context files violate authority-delegation contract:\n%+v", findings)
	}

	// 2) Parse the structured contracts from both files and require
	// shared-semantics equality.
	agentsBytes, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	clineBytes, err := os.ReadFile(filepath.Join(root, ".clinerules", "leamas.md"))
	if err != nil {
		t.Fatalf("read .clinerules/leamas.md: %v", err)
	}

	agentsContract, agentsErr := ParseContractBlock(string(agentsBytes))
	clineContract, clineErr := ParseContractBlock(string(clineBytes))
	if agentsErr != nil {
		t.Fatalf("AGENTS.md contract malformed: %v", agentsErr)
	}
	if clineErr != nil {
		t.Fatalf(".clinerules/leamas.md contract malformed: %v", clineErr)
	}
	if !agentsContract.IsValidContractSemantics() {
		t.Fatalf("AGENTS.md contract semantics are doctrinally invalid: %+v", agentsContract)
	}
	if !clineContract.IsValidContractSemantics() {
		t.Fatalf(".clinerules/leamas.md contract semantics are doctrinally invalid: %+v", clineContract)
	}
	if !SharedSemanticsEqual(agentsContract, clineContract) {
		t.Fatalf("shared authority semantics disagree between AGENTS.md and .clinerules/leamas.md\nAGENTS: %+v\nCline:  %+v", agentsContract, clineContract)
	}
}
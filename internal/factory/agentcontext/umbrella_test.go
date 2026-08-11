package agentcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findRepoRoot walks up from the test directory until it finds a
// directory that contains both AGENTS.md and .clinerules/leamas.md.
// Returns an error if it reaches the filesystem root without finding
// both files.
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
// for ACT-LEAMAS-FACTORY-AGENT-AUTHORITY-DELEGATION-CONTRACT01.
//
// It reads the real repository AGENTS.md and .clinerules/leamas.md
// and runs them through the production CheckRepo function. It MUST
// fail if a future edit reintroduces an unguarded expensive-gate
// instruction or an implicit commit/push/tag authority grant.
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

	findings, err := CheckRepo(root)
	if err != nil {
		t.Fatalf("CheckRepo: %v", err)
	}

	if len(findings) != 0 {
		var summary []string
		for _, f := range findings {
			summary = append(summary, fmt.Sprintf("%s:%s:%s", f.Path, f.Kind, f.Message))
		}
		t.Fatalf("real repo agent-context files violate authority-delegation contract:\n%s",
			strings.Join(summary, "\n"))
	}

	// Defense in depth: re-read the files and assert that the
	// dangerous substrings are absent at the textual level. This
	// guards against any future regression in the anchor tables
	// themselves.
	dangerous := []string{
		"Always run make factorize",
		"Always run make gate-dupcode",
		"Always run make gate",
		"always commit when tests pass",
		"automatically push",
		"push successful work automatically",
	}

	for _, rel := range []string{"AGENTS.md", filepath.Join(".clinerules", "leamas.md")} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		lower := strings.ToLower(string(data))
		for _, needle := range dangerous {
			if strings.Contains(lower, strings.ToLower(needle)) {
				t.Errorf("%s contains dangerous unguarded phrase %q", rel, needle)
			}
		}
	}
}

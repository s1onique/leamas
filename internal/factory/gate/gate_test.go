package gate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllVerifiers(t *testing.T) {
	verifiers := AllVerifiers()
	if len(verifiers) == 0 {
		t.Error("AllVerifiers should return verifiers")
	}

	// Check all have names
	for _, v := range verifiers {
		if v.Name == "" {
			t.Error("verifier should have a name")
		}
		if v.Run == nil {
			t.Error("verifier should have a Run function")
		}
	}
}

// findRepoRoot walks up from the current working directory looking for go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root containing go.mod not found")
		}
		dir = parent
	}
}

func TestRunFactorize(t *testing.T) {
	// Find the actual repo root, not the package directory.
	// A go test binary runs with its package source directory as CWD,
	// so "." would resolve to internal/factory/gate, not the repo root.
	repoRoot := findRepoRoot(t)
	if code := RunFactorize(repoRoot); code != 0 {
		t.Fatalf("RunFactorize(%q) returned %d", repoRoot, code)
	}
}

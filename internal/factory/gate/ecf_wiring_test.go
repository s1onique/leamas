package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// TestAllVerifiersIncludesECF proves that the executable-contract-first
// verifier is wired into the gate's factorization path by name.
func TestAllVerifiersIncludesECF(t *testing.T) {
	verifiers := AllVerifiers()
	found := false
	for _, v := range verifiers {
		if v.Name == "executable-contract-first" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("executable-contract-first verifier not registered in AllVerifiers()")
	}
}

// TestECFFactorizationWiring_DriftDetected exercises the wired verifier
// through the gate's AllVerifiers() entry point, asserting that
// projection drift is reported as a finding on the real path.
func TestECFFactorizationWiring_DriftDetected(t *testing.T) {
	tmpDir := t.TempDir()

	// Build a minimal but valid ECF repository.
	mustMkdirAll(t, filepath.Join(tmpDir, "docs/doctrine"))
	mustMkdirAll(t, filepath.Join(tmpDir, ".github"))
	mustMkdirAll(t, filepath.Join(tmpDir, "docs/templates"))

	canonDoctrine := "# Executable Contract First\n\n## Purpose\n\nDefine behavior.\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "docs/doctrine/executable-contract-first.md"), []byte(canonDoctrine), 0644); err != nil {
		t.Fatalf("write doctrine: %v", err)
	}

	canonAgent := "## Executable Contract First\n\nFor every task:\n\n1. Inspect.\n2. Plan.\n3. RED.\n4. GREEN.\n5. Refactor.\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "docs/doctrine/executable-contract-first-agent.md"), []byte(canonAgent), 0644); err != nil {
		t.Fatalf("write agent: %v", err)
	}

	// AGENTS.md exists but its marked block is drifted (different text).
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	drifted := "# AGENTS\n\n<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->\n## Drift\n\nDifferent content.\n<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->\n"
	if err := os.WriteFile(agentsPath, []byte(drifted), 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	// Copilot exists with valid marker content (matching canonical).
	copilotPath := filepath.Join(tmpDir, ".github/copilot-instructions.md")
	copilotContent := "<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:BEGIN -->\n" + canonAgent + "\n<!-- LEAMAS:EXECUTABLE-CONTRACT-FIRST:END -->\n"
	if err := os.WriteFile(copilotPath, []byte(copilotContent), 0644); err != nil {
		t.Fatalf("write copilot: %v", err)
	}

	// ACT template exists with required sections.
	actContent := "# ACT\n\n## Executable contract\n\n### Stable boundary\n\n### Test matrix\n\n### RED evidence\n\n### GREEN evidence\n\n### Exceptions\n\nNone.\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "docs/templates/act.md"), []byte(actContent), 0644); err != nil {
		t.Fatalf("write act.md: %v", err)
	}

	// Locate the verifier via AllVerifiers() and run it through the
	// factorization entry point.
	verifiers := AllVerifiers()
	var runFn func(string) []checks.Finding
	for _, v := range verifiers {
		if v.Name == "executable-contract-first" {
			runFn = v.Run
			break
		}
	}
	if runFn == nil {
		t.Fatal("executable-contract-first not registered")
	}

	findings := runFn(tmpDir)

	// Must report the drift on the AGENTS.md path.
	hasDrift := false
	for _, f := range findings {
		if f.Kind == "ECF007" && f.Path == "AGENTS.md" {
			hasDrift = true
		}
	}
	if !hasDrift {
		var codes []string
		for _, f := range findings {
			codes = append(codes, f.Kind+":"+f.Path)
		}
		t.Errorf("expected ECF007 drift on AGENTS.md through gate wiring, got: %s",
			strings.Join(codes, ", "))
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

package docs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckInventory(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with missing files
	findings := CheckInventory(tmpDir)
	if len(findings) != len(RequiredFactoryDocs) {
		t.Errorf("expected %d findings for empty dir, got %d", len(RequiredFactoryDocs), len(findings))
	}

	// Create one of the required files
	requiredPath := filepath.Join(tmpDir, "docs/adr/README.md")
	if err := os.MkdirAll(filepath.Dir(requiredPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(requiredPath, []byte("# ADR README"), 0644); err != nil {
		t.Fatal(err)
	}

	findings = CheckInventory(tmpDir)
	if len(findings) != len(RequiredFactoryDocs)-1 {
		t.Errorf("expected %d findings, got %d", len(RequiredFactoryDocs)-1, len(findings))
	}
}

func TestCheckADRStructure(t *testing.T) {
	tmpDir := t.TempDir()
	adrDir := filepath.Join(tmpDir, "docs", "adr")

	if err := os.MkdirAll(adrDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Test with missing ADR directory - should return error
	findings := CheckADRStructure(t.TempDir())
	if len(findings) == 0 {
		t.Error("expected findings for missing ADR directory")
	}

	// Create ADR without required sections
	incompleteADR := filepath.Join(adrDir, "0001-test.md")
	if err := os.WriteFile(incompleteADR, []byte("# ADR 0001\n\nJust some content."), 0644); err != nil {
		t.Fatal(err)
	}

	findings = CheckADRStructure(tmpDir)
	if len(findings) != len(RequiredADRSections) {
		t.Errorf("expected %d findings for incomplete ADR, got %d", len(RequiredADRSections), len(findings))
	}

	// Create ADR with all required sections
	completeADR := filepath.Join(adrDir, "0002-test.md")
	completeContent := `# ADR 0002

## Status
Accepted

## Context
Some context

## Decision
Some decision
`
	if err := os.WriteFile(completeADR, []byte(completeContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings = CheckADRStructure(tmpDir)
	if len(findings) != len(RequiredADRSections) {
		t.Errorf("expected %d findings after adding complete ADR, got %d", len(RequiredADRSections), len(findings))
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create all required factory docs with minimal content
	for _, file := range RequiredFactoryDocs {
		path := filepath.Join(tmpDir, file)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# Test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create ADRs with proper structure (overwriting placeholders)
	adrDir := filepath.Join(tmpDir, "docs", "adr")
	adrFiles := []string{
		"0001-local-first-single-binary.md",
		"0002-go-only-for-v0.md",
		"0003-web-first-local-cockpit.md",
		"0004-no-oidc-until-shared-rig.md",
		"0005-not-an-llm-gateway.md",
		"0006-filesystem-run-bundles.md",
	}
	for _, adr := range adrFiles {
		path := filepath.Join(adrDir, adr)
		content := `# ADR

## Status
Accepted

## Context
Test

## Decision
Test
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	findings := CheckRepo(tmpDir)
	if len(findings) != 0 {
		for _, f := range findings {
			t.Logf("unexpected finding: %s: %s: %s", f.Path, f.Kind, f.Message)
		}
		t.Errorf("expected no findings for complete factory docs, got %d", len(findings))
	}
}

func TestCheckRepoWithRealRepo(t *testing.T) {
	// Test with current repository structure
	findings := CheckRepo(".")
	// We don't know the exact state, just check it runs without error
	if findings == nil {
		t.Error("CheckRepo should return findings, not nil")
	}
}

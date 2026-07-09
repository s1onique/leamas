package boundary

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWitnessProxyRejectsDatabaseSQL verifies that witness proxy rejects database/sql.
func TestWitnessProxyRejectsDatabaseSQL(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	"database/sql"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-witness",
		Dir:               pkgDir,
		AllowedImports:    witnessAllowedImports,
		ForbiddenImports:  witnessForbiddenImports,
		ForbiddenContains: witnessForbiddenContains,
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "database/sql" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected database/sql to be rejected for witness policy")
	}
}

// TestWitnessProxyRejectsProviderSubstring verifies that witness proxy rejects provider imports.
func TestWitnessProxyRejectsProviderSubstring(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	_ "github.com/someone/openai-sdk"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-witness",
		Dir:               pkgDir,
		AllowedImports:    witnessAllowedImports,
		ForbiddenImports:  witnessForbiddenImports,
		ForbiddenContains: witnessForbiddenContains,
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "github.com/someone/openai-sdk" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected openai provider import to be rejected for witness policy")
	}
}

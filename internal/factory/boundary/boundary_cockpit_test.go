package boundary

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCockpitRejectsHttputil verifies that cockpit rejects net/http/httputil.
func TestCockpitRejectsHttputil(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	"net/http/httputil"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-cockpit",
		Dir:               pkgDir,
		AllowedImports:    cockpitAllowedImports,
		ForbiddenImports:  cockpitForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(cockpitForbiddenContains),
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "net/http/httputil" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected httputil to be rejected for cockpit policy")
	}
}

// TestCockpitRejectsDatabaseSQL verifies that cockpit rejects database/sql.
func TestCockpitRejectsDatabaseSQL(t *testing.T) {
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
		Name:              "test-cockpit",
		Dir:               pkgDir,
		AllowedImports:    cockpitAllowedImports,
		ForbiddenImports:  cockpitForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(cockpitForbiddenContains),
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
		t.Error("expected database/sql to be rejected for cockpit policy")
	}
}

// TestCockpitRejectsAuthImports verifies that cockpit rejects auth/session-like imports.
func TestCockpitRejectsAuthImports(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	_ "github.com/someone/session-manager"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-cockpit",
		Dir:               pkgDir,
		AllowedImports:    cockpitAllowedImports,
		ForbiddenImports:  cockpitForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(cockpitForbiddenContains),
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "github.com/someone/session-manager" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected session provider import to be rejected for cockpit policy")
	}
}

package boundary

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHulkPackageRejectsNetHTTP verifies that hulk packages reject net/http.
func TestHulkPackageRejectsNetHTTP(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	"net/http"
	"sort"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-hulk",
		Dir:               pkgDir,
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "net/http" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected net/http to be rejected for hulk policy")
	}
}

// TestHulkPackageRejectsTime verifies that hulk packages reject time.
func TestHulkPackageRejectsTime(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	"time"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-hulk",
		Dir:               pkgDir,
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
	}

	findings := checkPackage(policy, pkgDir)

	found := false
	for _, f := range findings {
		if f.Import == "time" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected time to be rejected for hulk policy")
	}
}

// TestHulkPackageRejectsDatabaseSQL verifies that hulk packages reject database/sql.
func TestHulkPackageRejectsDatabaseSQL(t *testing.T) {
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
		Name:              "test-hulk",
		Dir:               pkgDir,
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
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
		t.Error("expected database/sql to be rejected for hulk policy")
	}
}

// TestTestFilesIgnored verifies that *_test.go files are ignored.
func TestTestFilesIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "testpkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	testFile := filepath.Join(pkgDir, "test.go")
	content := `package testpkg

import (
	"net/http"
)
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	testFileTest := filepath.Join(pkgDir, "test_test.go")
	contentTest := `package testpkg

import (
	"net/http"
	"testing"
)

func TestSomething(t *testing.T) {
	_ = http.StatusOK
}
`
	if err := os.WriteFile(testFileTest, []byte(contentTest), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	policy := PackagePolicy{
		Name:              "test-hulk",
		Dir:               pkgDir,
		AllowedImports:    hulkAllowedImports,
		ForbiddenImports:  hulkForbiddenImports,
		ForbiddenContains: forbiddenContainsToMap(hulkForbiddenContains),
	}

	findings := checkPackage(policy, pkgDir)

	httpImportCount := 0
	for _, f := range findings {
		if f.Import == "net/http" {
			httpImportCount++
		}
	}

	if httpImportCount != 1 {
		t.Errorf("expected 1 violation (from test.go), got %d; test file should be ignored", httpImportCount)
	}
}

package forbidden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

func TestCheckForbiddenPatternsCmdScope(t *testing.T) {
	tmpDir := t.TempDir()

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	forbiddenFile := filepath.Join(cmdDir, "auth.go")
	forbiddenContent := `package app

import "database/sql"

func Connect() {
	// TODO: add OIDC support
}
`
	if err := os.WriteFile(forbiddenFile, []byte(forbiddenContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for forbidden patterns in cmd/")
	}

	found := false
	for _, f := range findings {
		if f.Kind == "forbidden_pattern" && f.Message == "found forbidden pattern: database/sql import" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find database/sql pattern")
	}
}

func TestCheckForbiddenPatternsInternalFactoryExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	factoryDir := filepath.Join(tmpDir, "internal", "factory")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatal(err)
	}

	factoryFile := filepath.Join(factoryDir, "oidc_check.go")
	factoryContent := `package factory

// CheckOIDC verifies no OIDC usage in production code.
func CheckOIDC() {}
`
	if err := os.WriteFile(factoryFile, []byte(factoryContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cleanFile := filepath.Join(cmdDir, "clean.go")
	cleanContent := `package app

func Clean() {}
`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	foundErrors := false
	for _, f := range findings {
		if f.Severity == checks.SeverityError {
			foundErrors = true
			break
		}
	}
	if foundErrors {
		t.Errorf("expected no error findings, got %d", len(findings))
	}
}

func TestCheckForbiddenPatternsTestFilesExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(cmdDir, "auth_test.go")
	testContent := `package app

import "testing"

func TestOIDCSupport(t *testing.T) {
	t.Log("Testing OIDC integration")
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	for _, f := range findings {
		if f.Path == "cmd/app/auth_test.go" {
			t.Error("found forbidden pattern in _test.go file")
		}
	}
}

func TestCheckForbiddenPatternsDocsAllowed(t *testing.T) {
	tmpDir := t.TempDir()

	docsDir := filepath.Join(tmpDir, "docs", "doctrine")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	docFile := filepath.Join(docsDir, "security.md")
	docContent := `# Security Doctrine

We explicitly forbid OIDC in production code.
`
	if err := os.WriteFile(docFile, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cleanFile := filepath.Join(cmdDir, "clean.go")
	cleanContent := `package app

func Clean() {}
`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	foundErrors := false
	for _, f := range findings {
		if f.Severity == checks.SeverityError {
			foundErrors = true
			break
		}
	}
	if foundErrors {
		t.Errorf("expected no error findings, got %d", len(findings))
	}
}

func TestCheckDatabaseImports(t *testing.T) {
	tmpDir := t.TempDir()

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	dbFile := filepath.Join(cmdDir, "db.go")
	dbContent := `package app

import _ "github.com/lib/pq"

func Init() {}
`
	if err := os.WriteFile(dbFile, []byte(dbContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckDatabaseImports(tmpDir)
	if len(findings) == 0 {
		t.Error("expected findings for database imports")
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	cmdDir := filepath.Join(tmpDir, "cmd", "app")
	if err := os.MkdirAll(cmdDir, 0755); err != nil {
		t.Fatal(err)
	}

	cleanFile := filepath.Join(cmdDir, "clean.go")
	cleanContent := `package app

func Clean() {}
`
	if err := os.WriteFile(cleanFile, []byte(cleanContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckRepo(tmpDir)
	foundErrors := false
	for _, f := range findings {
		if f.Severity == checks.SeverityError {
			foundErrors = true
			break
		}
	}
	if foundErrors {
		t.Errorf("expected no error findings, got %d", len(findings))
	}
}

// TestScriptsForbiddenPatternDetected verifies that scripts/ containing forbidden patterns are detected.
// This is a regression test for ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1.
func TestScriptsForbiddenPatternDetected(t *testing.T) {
	tmpDir := t.TempDir()

	scriptsDir := filepath.Join(tmpDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	badScript := filepath.Join(scriptsDir, "bad.sh")
	badContent := `#!/bin/bash
# Setup OIDC authentication
echo "Setting up OIDC..."
`
	if err := os.WriteFile(badScript, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	foundOIDC := false
	for _, f := range findings {
		if f.Kind == "forbidden_pattern" && strings.Contains(f.Message, "OIDC") {
			foundOIDC = true
			break
		}
	}
	if !foundOIDC {
		t.Error("expected to find forbidden OIDC pattern in scripts/bad.sh")
	}
}

// TestGithooksForbiddenPatternDetected verifies that githooks/ containing forbidden patterns are detected.
// This is a regression test for ACT-LEAMAS-FACTORY-FORBIDDEN-SCOPE-CONTRACT01-R1.
func TestGithooksForbiddenPatternDetected(t *testing.T) {
	tmpDir := t.TempDir()

	hooksDir := filepath.Join(tmpDir, "githooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatal(err)
	}

	badHook := filepath.Join(hooksDir, "pre-push")
	badContent := `#!/bin/sh
# Check for OIDC tokens
echo "Checking OIDC tokens..."
`
	if err := os.WriteFile(badHook, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	foundOIDC := false
	for _, f := range findings {
		if f.Kind == "forbidden_pattern" && strings.Contains(f.Message, "OIDC") {
			foundOIDC = true
			break
		}
	}
	if !foundOIDC {
		t.Error("expected to find forbidden OIDC pattern in githooks/pre-push")
	}
}

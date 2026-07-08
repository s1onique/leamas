package forbidden

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

func TestContainsForbidden(t *testing.T) {
	tests := []struct {
		pattern string
		content string
		want    bool
	}{
		{"OIDC|oidc", "using OIDC for auth", true},
		{"OIDC|oidc", "no auth here", false},
		{"RBAC|rbac", "has RBAC permissions", true},
		{"database/sql", `import "database/sql"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := containsForbidden(tt.content, tt.pattern)
			if got != tt.want {
				t.Errorf("containsForbidden(%q, %q) = %v, want %v", tt.content, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIsInAllowedDir(t *testing.T) {
	tests := []struct {
		path  string
		allow bool
	}{
		{"docs/doctrine/test.md", true},
		{"./docs/doctrine/test.md", true},
		{"docs/adr/0001-test.md", true},
		{"internal/foo.go", false},
		{"cmd/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isInAllowedDir(tt.path)
			if got != tt.allow {
				t.Errorf("isInAllowedDir(%q) = %v, want %v", tt.path, got, tt.allow)
			}
		})
	}
}

func TestCheckForbiddenPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cmd directory with a file containing forbidden pattern
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
		t.Error("expected findings for forbidden patterns")
	}

	// Verify it found database/sql
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

func TestCheckDatabaseImports(t *testing.T) {
	tmpDir := t.TempDir()

	// Create cmd directory with database import
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

func TestCheckForbiddenPatternsAllowsDocs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create docs/doctrine with forbidden content (should be allowed)
	docsDir := filepath.Join(tmpDir, "docs", "doctrine")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}

	docFile := filepath.Join(docsDir, "security.md")
	docContent := `# Security Doctrine

We explicitly forbid OIDC in production code.
Multi-tenancy is not a goal.
`
	if err := os.WriteFile(docFile, []byte(docContent), 0644); err != nil {
		t.Fatal(err)
	}

	findings := CheckForbiddenPatterns(tmpDir)
	// Should not find any findings because the file is in allowed directory
	for _, f := range findings {
		if f.Path == filepath.Join("docs", "doctrine", "security.md") {
			t.Error("found forbidden pattern in allowed directory docs/doctrine")
		}
	}
}

func TestCheckRepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a clean cmd package
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
	// Should have no findings for clean code
	foundErrors := false
	for _, f := range findings {
		if f.Severity == checks.SeverityError {
			foundErrors = true
			break
		}
	}
	if foundErrors {
		t.Errorf("expected no error findings for clean code, got %d", len(findings))
	}
}

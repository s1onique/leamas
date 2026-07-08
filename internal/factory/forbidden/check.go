// Package forbidden provides verification for forbidden patterns in production code.
//
// # SCAN BOUNDARY CONTRACT
//
// This verifier enforces the forbidden-pattern scan boundary defined in the Factory doctrine.
//
// SCAN:
//   - cmd/
//   - internal/ (except internal/factory/)
//   - scripts/
//   - githooks/
//   - AGENTS.md
//   - .clinerules/
//
// ALLOW (forbidden-policy terms permitted):
//   - internal/factory/       - Factory verification code must mention forbidden terms
//   - docs/doctrine/          - Doctrine documents discuss policy
//   - docs/adr/               - Architecture decision records
//   - docs/factory/           - Factory documentation
//   - docs/close-reports/     - Close reports
//   - *_test.go               - Test files
//   - testdata/               - Test fixtures
package forbidden

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// ForbiddenPattern represents a forbidden pattern to check.
type ForbiddenPattern struct {
	Pattern string
	Desc    string
}

// ForbiddenPatterns lists patterns forbidden in production code.
var ForbiddenPatterns = []ForbiddenPattern{
	{Pattern: `OIDC|oidc`, Desc: "OIDC implementation"},
	{Pattern: `OAuth|oauth`, Desc: "OAuth implementation"},
	{Pattern: `RBAC|rbac`, Desc: "RBAC implementation"},
	{Pattern: `ABAC|abac`, Desc: "ABAC implementation"},
	{Pattern: `multi.tenant|multitenancy|multi_tenant`, Desc: "Multi-tenancy"},
	{Pattern: `tenant|tenants`, Desc: "Tenancy reference"},
	{Pattern: `postgres|postgresql|mysql|mariadb|sqlite`, Desc: "Database storage"},
	{Pattern: `mongodb|dynamodb|cassandra`, Desc: "NoSQL database"},
	{Pattern: `redis|memcached|cockroachdb`, Desc: "Cache/database"},
	{Pattern: `LiteLLM|litellm`, Desc: "LiteLLM/provider routing"},
	{Pattern: `database/sql`, Desc: "database/sql import"},
	{Pattern: `github.com/lib/pq`, Desc: "PostgreSQL driver"},
	{Pattern: `github.com/go-sql-driver`, Desc: "SQL driver"},
}

// AllowedDirs are directories where forbidden patterns are allowed by policy.
var AllowedDirs = []string{
	"internal/factory",
	"docs/doctrine",
	"docs/adr",
	"docs/factory",
	"docs/close-reports",
	"testdata",
}

// ScanDirs are the directories to scan for forbidden patterns.
var ScanDirs = []string{"cmd", "internal", "scripts", "githooks"}

// ScanFiles are specific files to scan.
var ScanFiles = []string{"AGENTS.md", ".clinerules/leamas.md"}

// SkipPatterns are path patterns to skip during scanning.
var SkipPatterns = []string{"vendor", ".git", "testdata"}

// CheckForbiddenPatterns scans Go source files for forbidden patterns.
func CheckForbiddenPatterns(root string) []checks.Finding {
	var findings []checks.Finding

	for _, dir := range ScanDirs {
		scanPath := filepath.Join(root, dir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				name := d.Name()
				for _, skip := range SkipPatterns {
					if name == skip {
						return filepath.SkipDir
					}
				}
				return nil
			}

			relPath, _ := filepath.Rel(root, path)

			// Skip internal/factory
			if strings.HasPrefix(relPath, "internal/factory") ||
				strings.HasPrefix(relPath, "internal\\factory") {
				return nil
			}

			// Skip non-Go files and test files
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			checkFilePatterns(relPath, string(data), &findings)
			return nil
		})
		if err != nil {
			continue
		}
	}

	// Scan special files
	for _, file := range ScanFiles {
		path := filepath.Join(root, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		for _, pattern := range ForbiddenPatterns {
			if containsForbidden(content, pattern.Pattern) {
				if pattern.Pattern == "database/sql" ||
					pattern.Pattern == "github.com/lib/pq" ||
					pattern.Pattern == "github.com/go-sql-driver" {
					findings = append(findings, checks.Finding{
						Path:     file,
						Kind:     "forbidden_import",
						Message:  "database driver import: " + pattern.Desc,
						Severity: checks.SeverityError,
					})
				}
			}
		}
	}

	checks.SortFindings(findings)
	return findings
}

// containsForbidden checks if content contains the pattern.
func containsForbidden(content, pattern string) bool {
	parts := strings.Split(pattern, "|")
	for _, part := range parts {
		if strings.Contains(content, part) {
			return true
		}
	}
	return false
}

// checkFilePatterns checks a file's content for forbidden patterns.
func checkFilePatterns(path string, content string, findings *[]checks.Finding) {
	for _, pattern := range ForbiddenPatterns {
		if !containsForbidden(content, pattern.Pattern) {
			continue
		}
		if isInAllowedDir(path) {
			continue
		}
		*findings = append(*findings, checks.Finding{
			Path:     path,
			Kind:     "forbidden_pattern",
			Message:  "found forbidden pattern: " + pattern.Desc,
			Severity: checks.SeverityError,
		})
	}
}

// isInAllowedDir checks if a path is in an allowed directory.
func isInAllowedDir(path string) bool {
	for _, dir := range AllowedDirs {
		if strings.HasPrefix(path, dir) || strings.HasPrefix(path, "./"+dir) {
			return true
		}
	}
	return false
}

// CheckRepo runs all forbidden pattern checks.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding
	findings = append(findings, CheckForbiddenPatterns(root)...)
	findings = append(findings, CheckDatabaseImports(root)...)
	checks.SortFindings(findings)
	return findings
}

// CheckFile scans a single file for forbidden patterns.
func CheckFile(path string) []checks.Finding {
	var findings []checks.Finding
	data, err := os.ReadFile(path)
	if err != nil {
		return findings
	}
	content := string(data)
	for _, pattern := range ForbiddenPatterns {
		if !containsForbidden(content, pattern.Pattern) {
			continue
		}
		relPath, _ := filepath.Rel(".", path)
		if isInAllowedDir(relPath) {
			continue
		}
		findings = append(findings, checks.Finding{
			Path:     path,
			Kind:     "forbidden_pattern",
			Message:  "found forbidden pattern: " + pattern.Desc,
			Severity: checks.SeverityError,
		})
	}
	return findings
}

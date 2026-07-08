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
	"AGENTS.md",
	".clinerules",
}

// ScanDirs are the directories to scan for forbidden patterns.
var ScanDirs = []string{"cmd", "internal", "scripts", "githooks"}

// ScanFiles are specific files to scan.
var ScanFiles = []string{}

// SkipPatterns are path patterns to skip during scanning.
var SkipPatterns = []string{"vendor", ".git", "testdata"}

// shouldSkipDir returns true if the directory should be skipped during scanning.
func shouldSkipDir(name string) bool {
	skipDirs := []string{"vendor", ".git"}
	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	return false
}

// isGoProductionFile returns true if the path is a Go production file to scan.
func isGoProductionFile(relPath string) bool {
	// Must be a .go file
	if !strings.HasSuffix(relPath, ".go") {
		return false
	}
	// Must not be a test file
	if strings.HasSuffix(relPath, "_test.go") {
		return false
	}
	// cmd/** is scanned for Go files
	if strings.HasPrefix(relPath, "cmd/") || strings.HasPrefix(relPath, "cmd\\") {
		return true
	}
	// internal/** is scanned except internal/factory/**
	if strings.HasPrefix(relPath, "internal/") || strings.HasPrefix(relPath, "internal\\") {
		if strings.HasPrefix(relPath, "internal/factory/") || strings.HasPrefix(relPath, "internal/factory\\") ||
			strings.HasPrefix(relPath, "internal\\factory/") || strings.HasPrefix(relPath, "internal\\factory\\") {
			return false
		}
		return true
	}
	return false
}

// isTextPolicyFile returns true if the path is a text/script file to scan.
func isTextPolicyFile(relPath string) bool {
	// Scan scripts/** for all text files
	if strings.HasPrefix(relPath, "scripts/") || strings.HasPrefix(relPath, "scripts\\") {
		return true
	}
	// Scan githooks/** for all text files
	if strings.HasPrefix(relPath, "githooks/") || strings.HasPrefix(relPath, "githooks\\") {
		return true
	}
	return false
}

// shouldScanFile returns true if the file should be scanned for forbidden patterns.
func shouldScanFile(relPath string) bool {
	// Check if it's an allowed directory (policies permitted there)
	if isInAllowedDir(relPath) {
		return false
	}
	// Check Go production files
	if isGoProductionFile(relPath) {
		return true
	}
	// Check text/script files
	if isTextPolicyFile(relPath) {
		return true
	}
	return false
}

// CheckForbiddenPatterns scans files for forbidden patterns according to the scan boundary contract.
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
				if shouldSkipDir(name) {
					return filepath.SkipDir
				}
				return nil
			}

			relPath, _ := filepath.Rel(root, path)

			if !shouldScanFile(relPath) {
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

	// Scan special files (AGENTS.md, .clinerules/leamas.md)
	for _, file := range ScanFiles {
		path := filepath.Join(root, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		checkFilePatterns(file, string(data), &findings)
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

// Package forbidden provides verification for forbidden patterns in production code.
package forbidden

import (
	"bufio"
	"bytes"
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

// AllowedDirs are directories where forbidden patterns are allowed.
var AllowedDirs = []string{
	"docs/doctrine",
	"docs/doctrine/README.md", // README is documentation, not doctrine
	"docs/adr",
	"docs/factory",
	"docs/close-reports",
	"internal/factory", // Factory verification code is excluded from pattern checks
}

// CheckForbiddenPatterns scans Go source files for forbidden patterns.
func CheckForbiddenPatterns(root string) []checks.Finding {
	var findings []checks.Finding

	// Define directories to scan (production code only, not factory verification)
	scanDirs := []string{"cmd", "githooks"}

	// Also scan AGENTS.md and .clinerules
	scanFiles := []string{"AGENTS.md", ".clinerules/leamas.md"}

	for _, dir := range scanDirs {
		scanPath := filepath.Join(root, dir)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			if d.IsDir() {
				// Skip vendor, .git, etc.
				name := d.Name()
				if name == "vendor" || name == ".git" || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			// Only scan Go files (not test files for patterns)
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Skip test files
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)
			checkFilePatterns(relPath, string(data), &findings)

			return nil
		})
		if err != nil {
			continue
		}
	}

	// Scan special files
	for _, file := range scanFiles {
		path := filepath.Join(root, file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Check for patterns that are NEVER allowed even in docs
		// This catches places that should not mention these terms at all
		content := string(data)
		for _, pattern := range ForbiddenPatterns {
			if containsForbidden(content, pattern.Pattern) {
				// For these files, check if it's a database import
				if pattern.Pattern == "database/sql" || pattern.Pattern == "github.com/lib/pq" || pattern.Pattern == "github.com/go-sql-driver" {
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
	// Simple contains check for basic patterns
	// In a more complex version, could use regex
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

		// Check if file is in an allowed directory
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

// CheckDatabaseImports specifically checks for database driver imports in Go files.
func CheckDatabaseImports(root string) []checks.Finding {
	var findings []checks.Finding

	dbImportPatterns := []struct {
		Pattern string
		Desc    string
	}{
		{`"database/sql"`, "database/sql import"},
		{`"github.com/lib/pq"`, "lib/pq PostgreSQL driver"},
		{`"github.com/go-sql-driver/mysql"`, "MySQL driver"},
		{`"github.com/go-sql-driver/sqlite"`, "SQLite driver"},
		{`"github.com/mattn/go-sqlite3"`, "go-sqlite3 driver"},
	}

	scanDirs := []string{"cmd"}

	for _, dir := range scanDirs {
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
				if name == "vendor" || name == ".git" || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)
			content := string(data)

			for _, imp := range dbImportPatterns {
				if strings.Contains(content, imp.Pattern) {
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "forbidden_import",
						Message:  "database driver import: " + imp.Desc,
						Severity: checks.SeverityError,
					})
				}
			}

			return nil
		})
		if err != nil {
			continue
		}
	}

	checks.SortFindings(findings)
	return findings
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

	// Check each pattern
	for _, pattern := range ForbiddenPatterns {
		if !containsForbidden(content, pattern.Pattern) {
			continue
		}

		// Check if file is in allowed directory
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

// ScanGoFiles walks directories and checks Go files for forbidden patterns.
func ScanGoFiles(root string, dirs []string) ([]checks.Finding, error) {
	var findings []checks.Finding

	for _, dir := range dirs {
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
				if name == "vendor" || name == ".git" || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			relPath, _ := filepath.Rel(root, path)

			// Use scanner for line-aware output
			scanner := bufio.NewScanner(bytes.NewReader(data))
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()

				for _, pattern := range ForbiddenPatterns {
					if !containsForbidden(line, pattern.Pattern) {
						continue
					}

					if isInAllowedDir(relPath) {
						continue
					}

					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "forbidden_pattern",
						Message:  "line " + string(rune('0'+lineNum/10)) + string(rune('0'+lineNum%10)) + ": forbidden pattern: " + pattern.Desc,
						Severity: checks.SeverityError,
					})
				}
			}

			return nil
		})
		if err != nil {
			return findings, err
		}
	}

	checks.SortFindings(findings)
	return findings, nil
}

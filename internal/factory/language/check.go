// Package language provides verification for single-language enforcement (Go-only).
package language

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// AllowedShellDirs are directories where shell scripts are allowed.
var AllowedShellDirs = []string{
	"scripts",
	"githooks",
}

// ProductionDirs are directories that must contain only Go.
var ProductionDirs = []string{
	"cmd",
	"internal",
}

// ForbiddenNodeFiles lists Node.js package files that are not allowed.
var ForbiddenNodeFiles = []string{
	"package.json",
	"package-lock.json",
	"pnpm-lock.yaml",
	"yarn.lock",
}

// CheckLanguageCompliance verifies Go-only policy.
func CheckLanguageCompliance(root string) []checks.Finding {
	var findings []checks.Finding

	// Check production directories for non-Go files
	findings = append(findings, checkProductionDirs(root)...)

	// Check for Node.js package files anywhere
	findings = append(findings, checkForbiddenNodeFiles(root)...)

	// Check shell scripts are only in allowed dirs
	findings = append(findings, checkShellScriptLocations(root)...)

	checks.SortFindings(findings)
	return findings
}

// checkProductionDirs verifies cmd/ and internal/ contain only Go files.
func checkProductionDirs(root string) []checks.Finding {
	var findings []checks.Finding

	// Non-Go extensions that are forbidden in production dirs
	forbiddenExtensions := []string{
		".py", ".js", ".ts", ".jsx", ".tsx",
		".java", ".rs", ".c", ".cpp", ".h",
		".rb", ".php", ".swift", ".kt",
	}

	for _, dir := range ProductionDirs {
		dirPath := filepath.Join(root, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			// Directory doesn't exist, skip
			continue
		}

		err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() {
				// Skip vendor, testdata, etc.
				name := d.Name()
				if name == "vendor" || name == ".git" || name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			// Check if it's a non-Go file
			ext := strings.ToLower(filepath.Ext(path))
			for _, forbidden := range forbiddenExtensions {
				if ext == forbidden {
					relPath, _ := filepath.Rel(root, path)
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "forbidden_file",
						Message:  "non-Go file in production directory: " + ext,
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

	return findings
}

// checkForbiddenNodeFiles checks for Node.js package files.
func checkForbiddenNodeFiles(root string) []checks.Finding {
	var findings []checks.Finding

	for _, file := range ForbiddenNodeFiles {
		path := filepath.Join(root, file)
		if _, err := os.Stat(path); err == nil {
			findings = append(findings, checks.Finding{
				Path:     file,
				Kind:     "forbidden_file",
				Message:  "Node.js package file not permitted",
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

// checkShellScriptLocations verifies shell scripts are only in allowed directories.
func checkShellScriptLocations(root string) []checks.Finding {
	var findings []checks.Finding

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			// Skip ignored directories
			if name == ".git" || name == "vendor" || name == "build" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for shell scripts
		ext := filepath.Ext(path)
		if ext != ".sh" {
			return nil
		}

		relPath, _ := filepath.Rel(root, path)

		// Check if script is in allowed directory
		inAllowedDir := false
		for _, dir := range AllowedShellDirs {
			if strings.HasPrefix(relPath, dir) || strings.HasPrefix(relPath, "./"+dir) {
				inAllowedDir = true
				break
			}
		}

		if !inAllowedDir {
			findings = append(findings, checks.Finding{
				Path:     relPath,
				Kind:     "forbidden_location",
				Message:  "shell script outside allowed directories (scripts/, githooks/)",
				Severity: checks.SeverityError,
			})
		}

		return nil
	})
	if err != nil {
		return findings
	}

	return findings
}

// CheckPythonFiles scans for Python files anywhere (they're strictly forbidden).
func CheckPythonFiles(root string) []checks.Finding {
	var findings []checks.Finding

	// Directories to ignore
	ignoredDirs := map[string]bool{
		".git":  true,
		"build": true,
		"bin":   true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if ignoredDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".py" {
			relPath, _ := filepath.Rel(root, path)
			findings = append(findings, checks.Finding{
				Path:     relPath,
				Kind:     "forbidden",
				Message:  "Python file strictly forbidden",
				Severity: checks.SeverityError,
			})
		}

		return nil
	})
	if err != nil {
		return findings
	}

	return findings
}

// CheckRepo runs all language checks.
func CheckRepo(root string) []checks.Finding {
	var findings []checks.Finding

	findings = append(findings, CheckLanguageCompliance(root)...)
	findings = append(findings, CheckPythonFiles(root)...)

	checks.SortFindings(findings)
	return findings
}

// Package boundary provides verification for domain boundary import policies.
package boundary

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// checkPackage scans a single package directory for boundary violations.
func checkPackage(policy PackagePolicy, dirPath, repoRoot string) []Finding {
	var findings []Finding

	fset := token.NewFileSet()

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			// Skip testdata and vendor directories
			name := d.Name()
			if name == "testdata" || name == "vendor" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan .go files, skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileFindings := checkFile(policy, path, repoRoot, fset)
		findings = append(findings, fileFindings...)
		return nil
	})

	if err != nil {
		return findings
	}

	return findings
}

// checkCLIFile scans a single CLI runtime file for boundary violations.
func checkCLIFile(policy FilePolicy, filePath, repoRoot string) []Finding {
	var findings []Finding

	fset := token.NewFileSet()

	// Only scan .go files, skip test files
	if strings.HasSuffix(filePath, "_test.go") {
		return findings
	}

	fileFindings := checkFileForCLI(policy, filePath, repoRoot, fset)
	findings = append(findings, fileFindings...)

	return findings
}

// checkFile parses a single Go file and checks its imports using PackagePolicy.
func checkFile(policy PackagePolicy, filePath, repoRoot string, fset *token.FileSet) []Finding {
	var findings []Finding

	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return findings
	}

	relPath, _ := filepath.Rel(repoRoot, filePath)

	for _, imp := range f.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if import is explicitly forbidden
		if reason, forbidden := policy.ForbiddenImports[importPath]; forbidden {
			findings = append(findings, Finding{
				File:   relPath,
				Import: importPath,
				Reason: reason,
			})
			continue
		}

		// Check if import path contains forbidden substrings (deterministic order)
		for _, substring := range policy.ForbiddenContains {
			if strings.Contains(importPath, substring) {
				findings = append(findings, Finding{
					File:   relPath,
					Import: importPath,
					Reason: forbiddenContainsReason(substring),
				})
				break // Only report first match for this import
			}
		}
	}

	return findings
}

// checkFileForCLI parses a single CLI runtime file and checks its imports using FilePolicy.
func checkFileForCLI(policy FilePolicy, filePath, repoRoot string, fset *token.FileSet) []Finding {
	var findings []Finding

	f, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return findings
	}

	relPath, _ := filepath.Rel(repoRoot, filePath)

	for _, imp := range f.Imports {
		if imp.Path == nil {
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if import is explicitly forbidden
		if reason, forbidden := policy.ForbiddenImports[importPath]; forbidden {
			findings = append(findings, Finding{
				File:   relPath,
				Import: importPath,
				Reason: reason,
			})
			continue
		}

		// Check if this is an internal import that is forbidden
		if reason, forbidden := cliRuntimeForbiddenInternal[importPath]; forbidden {
			findings = append(findings, Finding{
				File:   relPath,
				Import: importPath,
				Reason: reason,
			})
			continue
		}

		// Check if this is an internal import that is NOT in the allowed list
		if strings.HasPrefix(importPath, "github.com/s1onique/leamas/internal/") {
			if !cliRuntimeAllowedInternal[importPath] {
				findings = append(findings, Finding{
					File:   relPath,
					Import: importPath,
					Reason: "CLI runtime must only import allowed internal packages",
				})
				continue
			}
		}

		// Check if import path contains forbidden substrings (deterministic order)
		for _, substring := range policy.ForbiddenContains {
			if strings.Contains(importPath, substring) {
				findings = append(findings, Finding{
					File:   relPath,
					Import: importPath,
					Reason: forbiddenContainsReason(substring),
				})
				break // Only report first match for this import
			}
		}
	}

	return findings
}

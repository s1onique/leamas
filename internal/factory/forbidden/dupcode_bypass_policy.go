// SPDX-License-Identifier: Apache-2.0

// Package forbidden provides AST-based static analysis for security policies.
package forbidden

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// DupcodeProtectedPrefixes are import path prefixes for protected dupcode packages.
var DupcodeProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// CanonicalAdapterPath is the ONLY package allowed to call protected dupcode packages.
const CanonicalAdapterPath = "github.com/s1onique/leamas/internal/factory/protectedverifier"

// DupcodeAllowedPaths are the canonical paths allowed to call protected dupcode packages.
// Only these specific paths may import dupcode; no wildcard matches.
// NOTE: This is the single source of truth for enforcement in isAllowedCaller.
var DupcodeAllowedPaths = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",           // dupcode itself
	"github.com/s1onique/leamas/internal/factory/protectedverifier", // canonical adapter (ONLY production entry point)
	"github.com/s1onique/leamas/internal/factory/gate",              // gate orchestration (internal infrastructure)
	"github.com/s1onique/leamas/internal/factory/forbidden",         // bypass policy enforcement
}

// BypassError represents a policy violation for direct protected access.
type BypassError struct {
	File       string
	Line       int
	ImportPath string
	Selector   string
	Message    string
}

func (e *BypassError) Error() string {
	return fmt.Sprintf("%s:%d: unauthorized call to %s.%s: %s",
		e.File, e.Line, e.ImportPath, e.Selector, e.Message)
}

// DupcodeBypassPolicy checks that production code does not directly call
// protected dupcode capabilities outside the canonical adapter.
type DupcodeBypassPolicy struct {
	allowedPaths map[string]bool
	repoRoot     string
}

// NewDupcodeBypassPolicy creates a policy with default allowed paths.
func NewDupcodeBypassPolicy() *DupcodeBypassPolicy {
	allowed := make(map[string]bool)
	for _, path := range DupcodeAllowedPaths {
		allowed[path] = true
	}

	// Get the current working directory as repo root
	cwd, _ := os.Getwd()
	return &DupcodeBypassPolicy{
		allowedPaths: allowed,
		repoRoot:     cwd,
	}
}

// NewDupcodeBypassPolicyWithRoot creates a policy with a specific repo root.
func NewDupcodeBypassPolicyWithRoot(repoRoot string) *DupcodeBypassPolicy {
	allowed := make(map[string]bool)
	for _, path := range DupcodeAllowedPaths {
		allowed[path] = true
	}
	return &DupcodeBypassPolicy{
		allowedPaths: allowed,
		repoRoot:     repoRoot,
	}
}

// CheckFile analyzes a single Go source file for policy violations.
// Returns a BypassError if unauthorized access is found.
// Returns nil if the file is not subject to bypass checking or has no violations.
func (p *DupcodeBypassPolicy) CheckFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return &BypassError{
			File:    filename,
			Message: fmt.Sprintf("parse error: %v", err),
		}
	}

	// Check for dot imports
	for _, imp := range node.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			return &BypassError{
				File:    filename,
				Line:    fset.Position(imp.Pos()).Line,
				Message: "dot imports are forbidden",
			}
		}
	}

	// Build import path -> alias map
	importAliases := make(map[string]string)
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil && imp.Name.Name != "" {
			importAliases[imp.Name.Name] = path
		} else {
			// Default import - use last path segment as alias
			parts := strings.Split(path, "/")
			alias := parts[len(parts)-1]
			importAliases[alias] = path
		}
	}

	// Check protected package usage
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if !p.isProtectedPackage(path) {
			continue
		}

		// This import is to a protected package
		// Check if this file's directory is allowed to use it
		if !p.isAllowedCaller(filename, path) {
			return &BypassError{
				File:       filename,
				Line:       fset.Position(imp.Pos()).Line,
				ImportPath: path,
				Message:    fmt.Sprintf("file in %s is not allowed to import protected package %s", p.getFileDirectory(filename), path),
			}
		}
	}

	// Walk the AST to check for selector expressions
	var foundErr error
	ast.Inspect(node, func(n ast.Node) bool {
		if foundErr != nil {
			return false
		}

		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Get the identifier (the base of the selector)
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		// Look up the import path for this identifier
		importPath, ok := importAliases[ident.Name]
		if !ok {
			return true
		}

		// Check if this import path is protected
		if !p.isProtectedPackage(importPath) {
			return true
		}

		// Check if the file's directory is allowed to call this selector
		if !p.isAllowedCaller(filename, importPath) {
			foundErr = &BypassError{
				File:       filename,
				Line:       fset.Position(sel.Pos()).Line,
				ImportPath: importPath,
				Selector:   sel.Sel.Name,
				Message:    fmt.Sprintf("file in %s is not allowed to call protected selector %s.%s", p.getFileDirectory(filename), importPath, sel.Sel.Name),
			}
			return false
		}

		return true
	})

	return foundErr
}

func (p *DupcodeBypassPolicy) getFileDirectory(filename string) string {
	dir := filepath.Dir(filename)

	// If repo root is set, compute relative path
	if p.repoRoot != "" {
		rel, err := filepath.Rel(p.repoRoot, dir)
		if err == nil {
			return rel
		}
	}

	// Fallback: try to compute relative to current working directory
	rel, err := filepath.Rel(".", dir)
	if err != nil {
		return dir
	}
	return rel
}

func (p *DupcodeBypassPolicy) isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// isAllowedCaller checks if the file at the given path is allowed to call the protected package.
// Authorization is based on the canonical repository-relative directory path.
// Only DupcodeAllowedPaths are permitted - this is the single source of truth.
func (p *DupcodeBypassPolicy) isAllowedCaller(filePath, protectedPackage string) bool {
	// Get the repository-relative directory
	relDir := p.getFileDirectory(filePath)

	// Normalize to forward slashes for consistent comparison
	relDir = filepath.ToSlash(relDir)

	// Check against allowed paths from DupcodeAllowedPaths
	// Only these specific directories may import dupcode
	allowedDirs := []struct {
		path        string
		description string
	}{
		{"internal/factory/protectedverifier", "canonical adapter"},
		{"internal/factory/dupcode", "dupcode itself"},
		{"internal/factory/gate", "gate orchestration (internal infrastructure)"},
	}

	for _, allowed := range allowedDirs {
		if relDir == allowed.path || strings.HasPrefix(relDir, allowed.path+"/") {
			return true
		}
	}

	// No other paths are allowed - fail closed
	return false
}

// CheckDupcodeBypass scans the repository for unauthorized access to protected dupcode packages.
// This is a fail-closed policy: any scan error produces a finding.
func CheckDupcodeBypass(root string) []checks.Finding {
	var findings []checks.Finding

	// Get absolute path to root
	absRoot, err := filepath.Abs(root)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     root,
			Kind:     "dupcode_bypass_scan_error",
			Message:  fmt.Sprintf("failed to resolve root path: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	policy := NewDupcodeBypassPolicyWithRoot(absRoot)

	// Scan production Go files in cmd/ and internal/
	// This does NOT skip internal/factory - we need to verify it only contains allowed calls
	scanDirs := []string{"cmd", "internal"}

	for _, dir := range scanDirs {
		scanPath := filepath.Join(absRoot, dir)
		info, err := os.Stat(scanPath)
		if err != nil {
			// Directory doesn't exist - skip silently (OK for partial checkouts/tests)
			if os.IsNotExist(err) {
				continue
			}
			// Inaccessible directory - fail closed
			findings = append(findings, checks.Finding{
				Path:     dir,
				Kind:     "dupcode_bypass_scan_error",
				Message:  fmt.Sprintf("failed to scan %s: %v", dir, err),
				Severity: checks.SeverityError,
			})
			continue
		}
		if !info.IsDir() {
			// Not a directory - skip
			continue
		}

		walkErr := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Fail closed: walk error
				relPath, _ := filepath.Rel(absRoot, path)
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_bypass_walk_error",
					Message:  fmt.Sprintf("walk error: %v", err),
					Severity: checks.SeverityError,
				})
				return nil // Continue scanning other files
			}

			if info.IsDir() {
				return nil
			}

			// Only scan production Go files
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			// Check this file for bypass violations
			checkErr := policy.CheckFile(path)
			if checkErr != nil {
				var bypassErr *BypassError
				if errors.As(checkErr, &bypassErr) {
					relPath, _ := filepath.Rel(absRoot, bypassErr.File)
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "dupcode_bypass",
						Message:  bypassErr.Message,
						Severity: checks.SeverityError,
					})
				} else {
					// Other error - treat as scan failure
					relPath, _ := filepath.Rel(absRoot, path)
					findings = append(findings, checks.Finding{
						Path:     relPath,
						Kind:     "dupcode_bypass_scan_error",
						Message:  fmt.Sprintf("scan error: %v", checkErr),
						Severity: checks.SeverityError,
					})
				}
			}

			return nil
		})

		if walkErr != nil {
			// Walk failed completely
			findings = append(findings, checks.Finding{
				Path:     dir,
				Kind:     "dupcode_bypass_walk_error",
				Message:  fmt.Sprintf("directory walk failed: %v", walkErr),
				Severity: checks.SeverityError,
			})
		}
	}

	return findings
}

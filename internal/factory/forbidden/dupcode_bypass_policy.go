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
)

// DupcodeProtectedPackages lists packages that contain protected dupcode capabilities.
var DupcodeProtectedPackages = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

// DupcodeAllowedPackages lists packages that are allowed to call protected dupcode packages.
var DupcodeAllowedPackages = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",           // dupcode itself
	"github.com/s1onique/leamas/internal/factory/protectedverifier", // canonical adapter
	"github.com/s1onique/leamas/internal/factory/gate",              // gate package
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
	allowedPackages map[string]bool
}

// NewDupcodeBypassPolicy creates a policy with default allowed packages.
func NewDupcodeBypassPolicy() *DupcodeBypassPolicy {
	allowed := make(map[string]bool)
	for _, pkg := range DupcodeAllowedPackages {
		allowed[pkg] = true
	}
	return &DupcodeBypassPolicy{allowedPackages: allowed}
}

// CheckFile analyzes a single Go source file for policy violations.
func (p *DupcodeBypassPolicy) CheckFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse error in %s: %w", filename, err)
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
		if imp.Name != nil && imp.Name.Name != "" {
			// Named import (alias)
			path := strings.Trim(imp.Path.Value, "\"")
			importAliases[imp.Name.Name] = path
		} else {
			// Default import
			path := strings.Trim(imp.Path.Value, "\"")
			// Extract last element of path as potential alias
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
		// Check if this file's package is allowed to use it
		if !p.isAllowedCaller(node.Name.Name, path) {
			return &BypassError{
				File:       filename,
				Line:       fset.Position(imp.Pos()).Line,
				ImportPath: path,
				Message:    fmt.Sprintf("package %s is not allowed to import protected package %s", node.Name.Name, path),
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

		// Check if the file's package is allowed to call this selector
		if !p.isAllowedCaller(node.Name.Name, importPath) {
			foundErr = &BypassError{
				File:       filename,
				Line:       fset.Position(sel.Pos()).Line,
				ImportPath: importPath,
				Selector:   sel.Sel.Name,
				Message:    fmt.Sprintf("unauthorized call to protected selector %s.%s from package %s", importPath, sel.Sel.Name, node.Name.Name),
			}
			return false
		}

		return true
	})

	return foundErr
}

// CheckPackageDir analyzes all Go files in a directory.
func (p *DupcodeBypassPolicy) CheckPackageDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		if err := p.CheckFile(path); err != nil {
			return err
		}

		return nil
	})
}

// CheckRepository scans the repository for policy violations.
func (p *DupcodeBypassPolicy) CheckRepository(root string) error {
	// Check internal/factory directory, excluding tests and the allowed packages
	return filepath.Walk(filepath.Join(root, "internal", "factory"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip allowed packages
			base := filepath.Base(path)
			if base == "dupcode" || base == "protectedverifier" || base == "forbidden" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip non-production files
		if strings.Contains(path, "_test") {
			return nil
		}

		if err := p.CheckFile(path); err != nil {
			var bypassErr *BypassError
			if errors.As(err, &bypassErr) {
				return bypassErr
			}
			return err
		}

		return nil
	})
}

func (p *DupcodeBypassPolicy) isProtectedPackage(path string) bool {
	for _, protected := range DupcodeProtectedPackages {
		if path == protected || strings.HasPrefix(path, protected+"/") {
			return true
		}
	}
	return false
}

func (p *DupcodeBypassPolicy) isAllowedCaller(callerPackage, protectedPackage string) bool {
	// The protected package itself is always allowed
	if callerPackage == "dupcode" {
		return true
	}

	// Check allowed packages - both full path and package name
	if p.allowedPackages[callerPackage] {
		return true
	}

	// Also check by package name (last segment of import path)
	for _, allowed := range DupcodeAllowedPackages {
		parts := strings.Split(allowed, "/")
		pkgName := parts[len(parts)-1]
		if pkgName == callerPackage {
			return true
		}
	}

	return false
}

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
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
)

// Note: DupcodeProtectedPrefixes, CanonicalAdapterPath, DupcodeAllowedPaths, and BypassError
// are defined in dupcode_bypass_policy.go and shared across all bypass policies.

// ExclusionPolicy describes what files are excluded from AST scanning.
type ExclusionPolicy struct {
	SkipTestFiles   bool
	SkipVendorDir   bool
	SkipDotGitDir   bool
	SkipTestdataDir bool
	SkipNonGoFiles  bool
}

// DefaultExclusionPolicy is the standard exclusion policy.
var DefaultExclusionPolicy = ExclusionPolicy{
	SkipTestFiles:   true,
	SkipVendorDir:   true,
	SkipDotGitDir:   true,
	SkipTestdataDir: true,
	SkipNonGoFiles:  true,
}

// WalkErrorPolicy describes how walk errors are handled.
type WalkErrorPolicy struct {
	FailClosed bool
}

// DefaultWalkErrorPolicy fails closed on walk errors.
var DefaultWalkErrorPolicy = WalkErrorPolicy{
	FailClosed: true,
}

// ParseErrorPolicy describes how parse errors are handled.
type ParseErrorPolicy struct {
	FailClosed bool
}

// DefaultParseErrorPolicy fails closed on parse errors.
var DefaultParseErrorPolicy = ParseErrorPolicy{
	FailClosed: true,
}

// CanonicalDupcodeBypassPolicy is a repository-wide AST policy for detecting bypasses.
type CanonicalDupcodeBypassPolicy struct {
	allowedPaths     map[string]bool
	repoRoot         string
	modulePath       string
	exclusionPolicy  ExclusionPolicy
	walkErrorPolicy  WalkErrorPolicy
	parseErrorPolicy ParseErrorPolicy
	rootResolver     *reporoot.RootResolver
}

// NewCanonicalDupcodeBypassPolicy creates a policy with canonical root resolution.
func NewCanonicalDupcodeBypassPolicy(repoRoot string, modulePath string) (*CanonicalDupcodeBypassPolicy, error) {
	resolver := reporoot.New()

	// Resolve to canonical root
	canonicalRoot, err := resolver.Resolve(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve canonical root: %w", err)
	}

	allowed := make(map[string]bool)
	for _, path := range DupcodeAllowedPaths {
		allowed[path] = true
	}

	return &CanonicalDupcodeBypassPolicy{
		allowedPaths:     allowed,
		repoRoot:         canonicalRoot,
		modulePath:       modulePath,
		exclusionPolicy:  DefaultExclusionPolicy,
		walkErrorPolicy:  DefaultWalkErrorPolicy,
		parseErrorPolicy: DefaultParseErrorPolicy,
		rootResolver:     resolver,
	}, nil
}

// CanonicalCheckDupcodeBypass performs a repository-wide AST scan for dupcode bypasses.
// This uses canonical root resolution and deterministic file discovery.
func CanonicalCheckDupcodeBypass(inputPath string, modulePath string) []checks.Finding {
	var findings []checks.Finding

	resolver := reporoot.New()

	// Resolve to canonical root
	canonicalRoot, err := resolver.Resolve(inputPath)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     inputPath,
			Kind:     "canonical_root_error",
			Message:  fmt.Sprintf("failed to resolve canonical root: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Create policy with canonical root
	policy, err := NewCanonicalDupcodeBypassPolicy(canonicalRoot, modulePath)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     inputPath,
			Kind:     "policy_init_error",
			Message:  fmt.Sprintf("failed to create policy: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Discover and scan production files
	prodFiles, err := policy.DiscoverProductionFiles()
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     canonicalRoot,
			Kind:     "file_discovery_error",
			Message:  fmt.Sprintf("failed to discover production files: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Scan each file
	for _, file := range prodFiles {
		err := policy.CheckFile(file)
		if err != nil {
			var bypassErr *BypassError
			if errors.As(err, &bypassErr) {
				relPath, _ := filepath.Rel(canonicalRoot, bypassErr.File)
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_bypass",
					Message:  bypassErr.Message,
					Severity: checks.SeverityError,
				})
			} else {
				relPath, _ := filepath.Rel(canonicalRoot, file)
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_scan_error",
					Message:  fmt.Sprintf("scan error: %v", err),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Sort findings for determinism
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings
}

// DiscoverProductionFiles finds all production Go files in the repository.
func (p *CanonicalDupcodeBypassPolicy) DiscoverProductionFiles() ([]string, error) {
	var files []string

	scanDirs := []string{"cmd", "internal"}

	for _, dir := range scanDirs {
		scanPath := filepath.Join(p.repoRoot, dir)
		info, err := os.Stat(scanPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to stat %s: %w", dir, err)
		}
		if !info.IsDir() {
			continue
		}

		err = filepath.WalkDir(scanPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if p.walkErrorPolicy.FailClosed {
					return err
				}
				return nil
			}

			if d.IsDir() {
				name := d.Name()
				if p.exclusionPolicy.SkipDotGitDir && name == ".git" {
					return filepath.SkipDir
				}
				if p.exclusionPolicy.SkipVendorDir && name == "vendor" {
					return filepath.SkipDir
				}
				if p.exclusionPolicy.SkipTestdataDir && name == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}

			// Check file eligibility
			if p.exclusionPolicy.SkipNonGoFiles && !strings.HasSuffix(path, ".go") {
				return nil
			}
			if p.exclusionPolicy.SkipTestFiles && strings.HasSuffix(path, "_test.go") {
				return nil
			}

			files = append(files, path)
			return nil
		})

		if err != nil && p.walkErrorPolicy.FailClosed {
			return nil, fmt.Errorf("walk error for %s: %w", dir, err)
		}
	}

	// Sort for determinism
	sort.Strings(files)
	return files, nil
}

// CheckFile analyzes a single Go source file for policy violations.
func (p *CanonicalDupcodeBypassPolicy) CheckFile(filename string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		if p.parseErrorPolicy.FailClosed {
			return &BypassError{
				File:    filename,
				Message: fmt.Sprintf("parse error: %v", err),
			}
		}
		return nil
	}

	// Check for dot imports of protected packages
	for _, imp := range node.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			path := strings.Trim(imp.Path.Value, "\"")
			if p.isProtectedPackage(path) {
				return &BypassError{
					File:    filename,
					Line:    fset.Position(imp.Pos()).Line,
					Message: "dot imports of protected packages are forbidden",
				}
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
			parts := strings.Split(path, "/")
			alias := parts[len(parts)-1]
			importAliases[alias] = path
		}
	}

	// Check protected package imports
	for _, imp := range node.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if !p.isProtectedPackage(path) {
			continue
		}

		if !p.isAllowedCaller(filename, path) {
			relDir := p.relativeDir(filename)
			return &BypassError{
				File:       filename,
				Line:       fset.Position(imp.Pos()).Line,
				ImportPath: path,
				Message:    fmt.Sprintf("file in %s is not allowed to import protected package %s", relDir, path),
			}
		}
	}

	// Walk AST for selector expressions
	var foundErr error
	ast.Inspect(node, func(n ast.Node) bool {
		if foundErr != nil {
			return false
		}

		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		importPath, ok := importAliases[ident.Name]
		if !ok {
			return true
		}

		if !p.isProtectedPackage(importPath) {
			return true
		}

		if !p.isAllowedCaller(filename, importPath) {
			relDir := p.relativeDir(filename)
			foundErr = &BypassError{
				File:       filename,
				Line:       fset.Position(sel.Pos()).Line,
				ImportPath: importPath,
				Selector:   sel.Sel.Name,
				Message:    fmt.Sprintf("file in %s is not allowed to call protected selector %s.%s", relDir, importPath, sel.Sel.Name),
			}
			return false
		}

		return true
	})

	return foundErr
}

func (p *CanonicalDupcodeBypassPolicy) relativeDir(filename string) string {
	dir := filepath.Dir(filename)
	rel, err := filepath.Rel(p.repoRoot, dir)
	if err != nil {
		return dir
	}
	return rel
}

func (p *CanonicalDupcodeBypassPolicy) isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func (p *CanonicalDupcodeBypassPolicy) isAllowedCaller(filePath, protectedPackage string) bool {
	relDir := p.relativeDir(filePath)
	relDir = filepath.ToSlash(relDir)

	// Convert directory path to import path
	importPath := p.rootResolver.ImportPathFromRelPath(relDir, p.modulePath)

	return p.allowedPaths[importPath]
}

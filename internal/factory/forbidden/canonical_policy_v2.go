// SPDX-License-Identifier: Apache-2.0

// Package forbidden provides AST-based static analysis for security policies.
package forbidden

import (
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
)

// Note: DupcodeProtectedPrefixes, CanonicalAdapterPath, DupcodeAllowedPaths, and BypassError
// are defined in dupcode_bypass_policy.go and shared across all bypass policies.

// ProtectedSymbols defines which symbols are protected within allowed packages.
var ProtectedSymbols = map[string][]string{
	"github.com/s1onique/leamas/internal/factory/dupcode": {
		"CheckRepo", "CheckReport", "LoadBaseline", "VerifyBaseline",
		"WriteBaseline", "CompareToBaseline",
	},
}

// CanonicalDupcodeBypassPolicyV2 is a symbol-aware AST policy for detecting bypasses.
type CanonicalDupcodeBypassPolicyV2 struct {
	allowedPaths     map[string]bool
	repoRoot         string
	modulePath       string
	exclusionPolicy  ExclusionPolicy
	walkErrorPolicy  WalkErrorPolicy
	parseErrorPolicy ParseErrorPolicy
	rootResolver     *reporoot.RootResolver
}

// NewCanonicalDupcodeBypassPolicyV2 creates a symbol-aware policy.
func NewCanonicalDupcodeBypassPolicyV2(repoRoot string, modulePath string) (*CanonicalDupcodeBypassPolicyV2, error) {
	resolver := reporoot.New()

	canonicalRoot, err := resolver.Resolve(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve canonical root: %w", err)
	}

	allowed := make(map[string]bool)
	for _, path := range DupcodeAllowedPaths {
		allowed[path] = true
	}

	return &CanonicalDupcodeBypassPolicyV2{
		allowedPaths:     allowed,
		repoRoot:         canonicalRoot,
		modulePath:       modulePath,
		exclusionPolicy:  DefaultExclusionPolicy,
		walkErrorPolicy:  DefaultWalkErrorPolicy,
		parseErrorPolicy: DefaultParseErrorPolicy,
		rootResolver:     resolver,
	}, nil
}

// CanonicalCheckDupcodeBypassV2 performs repository-wide symbol-aware AST scanning.
func CanonicalCheckDupcodeBypassV2(inputPath string, modulePath string) []checks.Finding {
	var findings []checks.Finding

	resolver := reporoot.New()

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

	policy, err := NewCanonicalDupcodeBypassPolicyV2(canonicalRoot, modulePath)
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     inputPath,
			Kind:     "policy_init_error",
			Message:  fmt.Sprintf("failed to create policy: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Repository-wide discovery: walk from repo root
	prodFiles, err := policy.DiscoverProductionFilesRepoWide()
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     canonicalRoot,
			Kind:     "file_discovery_error",
			Message:  fmt.Sprintf("failed to discover production files: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Scan each file with symbol awareness
	for _, file := range prodFiles {
		fileFindings := policy.CheckFileWithSymbols(file)
		findings = append(findings, fileFindings...)
	}

	// Sort for determinism
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Kind < findings[j].Kind
	})

	return findings
}

// DiscoverProductionFilesRepoWide discovers all production Go files from repo root.
func (p *CanonicalDupcodeBypassPolicyV2) DiscoverProductionFilesRepoWide() ([]string, error) {
	var files []string

	err := filepath.WalkDir(p.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if p.walkErrorPolicy.FailClosed {
				return err
			}
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil && p.walkErrorPolicy.FailClosed {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

// CheckFileWithSymbols performs symbol-aware analysis on a file.
func (p *CanonicalDupcodeBypassPolicyV2) CheckFileWithSymbols(filename string) []checks.Finding {
	var findings []checks.Finding

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		if p.parseErrorPolicy.FailClosed {
			findings = append(findings, checks.Finding{
				Path:     p.relativePath(filename),
				Kind:     "dupcode_parse_error",
				Message:  fmt.Sprintf("parse error: %v", err),
				Severity: checks.SeverityError,
			})
		}
		return findings
	}

	// Type-check to get symbol information
	conf := types.Config{
		Importer: importer.ForCompiler(fset, "source", nil),
	}

	_, err = conf.Check(filename, fset, []*ast.File{node}, nil)
	// Continue even if type check fails - we still want AST analysis

	// Build import alias map
	importAliases := make(map[string]string) // alias -> import path
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

	// Check for dot imports of protected packages
	for _, imp := range node.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			path := strings.Trim(imp.Path.Value, "\"")
			if p.isProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path:     p.relativePath(filename),
					Kind:     "dupcode_bypass",
					Message:  fmt.Sprintf("dot imports of protected package %s are forbidden", path),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Check if caller is allowed
	callerPath := p.callerImportPath(filename)
	if !p.isAllowedCaller(callerPath) {
		// Check for protected imports
		for _, imp := range node.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if p.isProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path:     p.relativePath(filename),
					Kind:     "dupcode_bypass",
					Message:  fmt.Sprintf("file in %s is not allowed to import protected package %s", callerPath, path),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Walk AST for selector and call expressions with symbol awareness
	ast.Inspect(node, func(n ast.Node) bool {
		// Check selector expressions: x.ProtectedSymbol
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				alias := ident.Name
				importPath, hasImport := importAliases[alias]

				if hasImport && p.isProtectedPackage(importPath) {
					symbolName := sel.Sel.Name

					// Check if this is a protected symbol
					if p.isProtectedSymbol(importPath, symbolName) {
						// Only flag if caller is not allowed
						if !p.isAllowedCaller(callerPath) {
							line := fset.Position(sel.Pos()).Line
							findings = append(findings, checks.Finding{
								Path:     p.relativePath(filename),
								Kind:     "dupcode_bypass",
								Message:  fmt.Sprintf("line %d: unauthorized call to %s.%s", line, importPath, symbolName),
								Severity: checks.SeverityError,
							})
						}
					}
				}
			}
		}

		// Check call expressions: ProtectedSymbol(...)
		if call, ok := n.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok {
				symbolName := ident.Name

				// Check if this is a protected symbol call
				if p.isProtectedSymbolCall(symbolName, callerPath) {
					line := fset.Position(call.Pos()).Line
					findings = append(findings, checks.Finding{
						Path:     p.relativePath(filename),
						Kind:     "dupcode_bypass",
						Message:  fmt.Sprintf("line %d: unauthorized call to protected symbol %s", line, symbolName),
						Severity: checks.SeverityError,
					})
				}
			}
		}

		return true
	})

	return findings
}

func (p *CanonicalDupcodeBypassPolicyV2) relativePath(filename string) string {
	rel, err := filepath.Rel(p.repoRoot, filename)
	if err != nil {
		return filename
	}
	return rel
}

func (p *CanonicalDupcodeBypassPolicyV2) callerImportPath(filePath string) string {
	relDir := filepath.Dir(filePath)
	rel, err := filepath.Rel(p.repoRoot, relDir)
	if err != nil {
		return relDir
	}
	return p.rootResolver.ImportPathFromRelPath(rel, p.modulePath)
}

func (p *CanonicalDupcodeBypassPolicyV2) isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func (p *CanonicalDupcodeBypassPolicyV2) isAllowedCaller(callerPath string) bool {
	return p.allowedPaths[callerPath]
}

func (p *CanonicalDupcodeBypassPolicyV2) isProtectedSymbol(pkgPath, symbolName string) bool {
	symbols, ok := ProtectedSymbols[pkgPath]
	if !ok {
		return false
	}
	for _, s := range symbols {
		if s == symbolName {
			return true
		}
	}
	return false
}

func (p *CanonicalDupcodeBypassPolicyV2) isProtectedSymbolCall(symbolName, callerPath string) bool {
	// Check if any protected package has this symbol
	for _, symbols := range ProtectedSymbols {
		for _, s := range symbols {
			if s == symbolName {
				// Only flag if not in allowed caller
				if !p.isAllowedCaller(callerPath) {
					return true
				}
			}
		}
	}
	return false
}

// ScanError represents a non-bypass scan error.
type ScanError struct {
	File    string
	Message string
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("%s: scan error: %s", e.File, e.Message)
}

// IsBypassError checks if an error is a bypass violation.
func IsBypassError(err error) bool {
	var bypassErr *BypassError
	return errors.As(err, &bypassErr)
}

// IsScanError checks if an error is a scan failure.
func IsScanError(err error) bool {
	var scanErr *ScanError
	return errors.As(err, &scanErr)
}

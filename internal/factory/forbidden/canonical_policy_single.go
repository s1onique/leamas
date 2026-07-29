// SPDX-License-Identifier: Apache-2.0

// Package forbidden provides AST-based static analysis for security policies.
package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
	"golang.org/x/tools/go/packages"
)

// Note: DupcodeProtectedPrefixes, DupcodeAllowedPaths, and BypassError
// are defined in dupcode_bypass_policy.go and shared across all bypass policies.

// ProtectedSymbol represents an exact protected symbol.
type ProtectedSymbol struct {
	PackagePath string
	Name        string
}

// ProtectedSymbols defines exact protected symbols and their allowed callers.
var ProtectedSymbols = []ProtectedSymbol{
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo"},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport"},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline"},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline"},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline"},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline"},
}

// ApprovedCaller defines an exact authorized caller edge.
type ApprovedCaller struct {
	PackagePath string
	Function    string
	Callee      ProtectedSymbol
}

// ApprovedCallers defines exact approved caller-to-callee edges.
// Only these specific edges are allowed; all other calls are violations.
var ApprovedCallers = []ApprovedCaller{
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline"}},
	// Dupcode package itself may call its own functions
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo"}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Function: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport"}},
}

// DupcodeBypassPolicy is the canonical dupcode bypass policy.
type DupcodeBypassPolicy struct {
	repoRoot   string
	modulePath string
	resolver   *reporoot.RootResolver
}

// NewDupcodeBypassPolicy creates a new policy.
func NewDupcodeBypassPolicy(repoRoot, modulePath string) (*DupcodeBypassPolicy, error) {
	resolver := reporoot.New()
	canonicalRoot, err := resolver.Resolve(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve canonical root: %w", err)
	}
	return &DupcodeBypassPolicy{
		repoRoot:   canonicalRoot,
		modulePath: modulePath,
		resolver:   resolver,
	}, nil
}

// CanonicalCheckDupcodeBypass is the single exported entry point for canonical dupcode bypass checking.
// This function uses module-aware type checking with go/packages for accurate symbol identity resolution.
func CanonicalCheckDupcodeBypass(root string, modulePath string) []checks.Finding {
	var findings []checks.Finding

	policy, err := NewDupcodeBypassPolicy(root, modulePath)
	if err != nil {
		return []checks.Finding{
			{Path: root, Kind: "canonical_root_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	// Load packages module-aware
	pkgs, err := policy.loadPackages()
	if err != nil {
		findings = append(findings, checks.Finding{
			Path:     policy.repoRoot,
			Kind:     "dupcode_package_load_error",
			Message:  fmt.Sprintf("failed to load packages: %v", err),
			Severity: checks.SeverityError,
		})
		return findings
	}

	// Build symbol table: object -> protected symbol info
	protectedSyms := policy.buildProtectedSymbolTable()

	// Analyze each package
	for _, pkg := range pkgs {
		if pkg.Types == nil || pkg.TypesInfo == nil {
			findings = append(findings, checks.Finding{
				Path:     policy.repoRoot,
				Kind:     "dupcode_type_info_error",
				Message:  fmt.Sprintf("package %s missing type info", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			continue
		}

		// Check each file in the package
		for i, file := range pkg.Syntax {
			if i >= len(pkg.GoFiles) {
				continue
			}
			filename := pkg.GoFiles[i]
			fileFindings := policy.analyzeFile(pkg, filename, file, protectedSyms)
			findings = append(findings, fileFindings...)
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

// loadPackages loads all packages using go/packages with full type information.
func (p *DupcodeBypassPolicy) loadPackages() ([]*packages.Package, error) {
	cfg := &packages.Config{
		Dir:   p.repoRoot,
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Tests: false,
		Fset:  token.NewFileSet(),
	}

	// Load all packages under repo root
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	// Filter to only production packages (not test packages)
	var result []*packages.Package
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}
		// Skip vendor
		if strings.Contains(pkg.PkgPath, "vendor/") {
			continue
		}
		result = append(result, pkg)
	}

	return result, nil
}

// protectedSymInfo holds information about a protected symbol.
type protectedSymInfo struct {
	pkgPath string
	name    string
	obj     interface{} // *types.Func
}

// buildProtectedSymbolTable builds a map of protected symbols for fast lookup.
func (p *DupcodeBypassPolicy) buildProtectedSymbolTable() map[string]protectedSymInfo {
	result := make(map[string]protectedSymInfo)
	for _, sym := range ProtectedSymbols {
		key := sym.PackagePath + "." + sym.Name
		result[key] = protectedSymInfo{pkgPath: sym.PackagePath, name: sym.Name}
	}
	return result
}

// isProtectedCall checks if a call is to a protected symbol from an unauthorized caller.
func (p *DupcodeBypassPolicy) isProtectedCall(pkg *packages.Package, callerPath string, callerFunc string, pkgPath string, symName string) bool {
	// Build key for the callee
	calleeKey := pkgPath + "." + symName

	// Check if this is a protected symbol
	_, isProtected := ProtectedSymbolsMap()[calleeKey]
	if !isProtected {
		return false
	}

	// Check if this is an approved caller
	return !IsApprovedCaller(callerPath, callerFunc, pkgPath, symName)
}

// ProtectedSymbolsMap returns a map for fast lookup.
func ProtectedSymbolsMap() map[string]bool {
	result := make(map[string]bool)
	for _, sym := range ProtectedSymbols {
		key := sym.PackagePath + "." + sym.Name
		result[key] = true
	}
	return result
}

// IsApprovedCaller checks if a caller-callee pair is approved.
func IsApprovedCaller(callerPath, callerFunc, calleePath, calleeName string) bool {
	for _, ac := range ApprovedCallers {
		if ac.PackagePath == callerPath && ac.Callee.PackagePath == calleePath && ac.Callee.Name == calleeName {
			// Empty function means any function in package is allowed
			if ac.Function == "" || ac.Function == callerFunc {
				return true
			}
		}
	}
	return false
}

// analyzeFile analyzes a single file for dupcode bypass violations.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File, protectedSyms map[string]protectedSymInfo) []checks.Finding {
	var findings []checks.Finding

	relPath, _ := filepath.Rel(p.repoRoot, filename)
	callerPath := p.callerImportPath(filename)

	// Check for dot imports of protected packages
	for _, imp := range file.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			path := strings.Trim(imp.Path.Value, "\"")
			if isProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_bypass",
					Message:  fmt.Sprintf("dot import of protected package %s is forbidden", path),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Check for direct imports of protected packages from unauthorized callers
	if p.isProtectedCall(pkg, callerPath, "", callerPath, "") {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			if isProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_bypass",
					Message:  fmt.Sprintf("import of protected package %s from unauthorized package %s", path, callerPath),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Walk AST for call expressions with proper symbol resolution
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Get line number
		line := p.fsetPosition(pkg, call.Pos())

		// Check for selector expressions: x.Method()
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			// Use type info to resolve the actual object
			obj := pkg.TypesInfo.ObjectOf(ident)
			if obj == nil {
				return true
			}

			// Check if it's a package identifier
			pkgName, ok := obj.(*types.PkgName)
			if !ok {
				return true
			}

			importedPath := pkgName.Imported().Path()
			symName := sel.Sel.Name

			// Check if this is a protected call from unauthorized caller
			if p.isProtectedCall(pkg, callerPath, "", importedPath, symName) {
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_bypass",
					Message:  fmt.Sprintf("line %d: unauthorized call to %s.%s from %s", line, importedPath, symName, callerPath),
					Severity: checks.SeverityError,
				})
			}
			return true
		}

		// Check for direct function calls: Function()
		if ident, ok := call.Fun.(*ast.Ident); ok {
			// Use type info to resolve the actual object
			obj := pkg.TypesInfo.ObjectOf(ident)
			if obj == nil {
				return true
			}

			// Check if it's a function from a protected package
			if fn, ok := obj.(*types.Func); ok {
				sig := fn.Type().(*types.Signature)
				if sig != nil && sig.Recv() == nil {
					// Package-level function
					fnPkg := fn.Pkg()
					if fnPkg != nil {
						fnPkgPath := fnPkg.Path()
						symName := fn.Name()

						if p.isProtectedCall(pkg, callerPath, "", fnPkgPath, symName) {
							findings = append(findings, checks.Finding{
								Path:     relPath,
								Kind:     "dupcode_bypass",
								Message:  fmt.Sprintf("line %d: unauthorized call to %s.%s from %s", line, fnPkgPath, symName, callerPath),
								Severity: checks.SeverityError,
							})
						}
					}
				}
			}
			return true
		}

		return true
	})

	return findings
}

func (p *DupcodeBypassPolicy) fsetPosition(pkg *packages.Package, pos token.Pos) int {
	if pkg.Fset != nil {
		return pkg.Fset.Position(pos).Line
	}
	return 0
}

func (p *DupcodeBypassPolicy) callerImportPath(filePath string) string {
	relDir := filepath.Dir(filePath)
	rel, err := filepath.Rel(p.repoRoot, relDir)
	if err != nil {
		return relDir
	}
	return p.resolver.ImportPathFromRelPath(rel, p.modulePath)
}

func isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// FindProtectedSymbol finds a protected symbol by package path and name.
func FindProtectedSymbol(pkgPath, name string) *ProtectedSymbol {
	for _, sym := range ProtectedSymbols {
		if sym.PackagePath == pkgPath && sym.Name == name {
			return &sym
		}
	}
	return nil
}

// LegacyCheckDupcodeBypass performs the legacy (V1) dupcode bypass check.
// Deprecated: Use CanonicalCheckDupcodeBypass instead.
func LegacyCheckDupcodeBypass(root string) []checks.Finding {
	// This runs the legacy AST-based scanner without type information
	return CheckDupcodeBypass(root)
}

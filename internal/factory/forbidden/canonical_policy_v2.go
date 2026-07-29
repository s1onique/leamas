// SPDX-License-Identifier: Apache-2.0

// Package forbidden provides AST-based static analysis for security policies.
package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/forbidden/reporoot"
	"golang.org/x/tools/go/packages"
)

// ProtectedSymbolKind is the kind of a protected symbol.
type ProtectedSymbolKind string

const (
	// ProtectedPackageFunction is a package-level function.
	ProtectedPackageFunction ProtectedSymbolKind = "package_function"
	// ProtectedMethod is a method with a receiver.
	ProtectedMethod ProtectedSymbolKind = "method"
)

// ProtectedSymbol represents an exact protected declaration.
type ProtectedSymbol struct {
	PackagePath string
	Name        string
	Kind        ProtectedSymbolKind
	// Receiver is required for ProtectedMethod.
	Receiver string
}

// ApprovedCaller defines an exact approved caller edge.
type ApprovedCaller struct {
	PackagePath string
	Function    string
	Receiver    string
	Callee      ProtectedSymbol
}

// DupcodeProtectedPrefixes defines protected package prefixes.
var DupcodeProtectedPrefixes = []string{
	"github.com/s1onique/leamas/internal/factory/dupcode",
}

func isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

// ApprovedCallers defines exact approved caller-to-callee edges.
// No wildcards: function must be set.
var ApprovedCallers = []ApprovedCaller{
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport", Kind: ProtectedPackageFunction}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline", Kind: ProtectedPackageFunction}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline", Kind: ProtectedPackageFunction}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline", Kind: ProtectedPackageFunction}},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/protectedverifier", Function: "Adapter", Receiver: "", Callee: ProtectedSymbol{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline", Kind: ProtectedPackageFunction}},
}

// CallerIdentity identifies an enclosing caller.
type CallerIdentity struct {
	PackagePath string
	Function    string
	Receiver    string
	Kind        string // package_function, method, function_literal, package_init, variable_initializer
}

// IsApprovedCaller checks if a caller-callee edge is approved. Function and Receiver must both match exactly.
func IsApprovedCaller(caller CallerIdentity, callee ProtectedSymbol) bool {
	for _, ac := range ApprovedCallers {
		if ac.PackagePath != caller.PackagePath {
			continue
		}
		if ac.Callee.PackagePath != callee.PackagePath ||
			ac.Callee.Name != callee.Name ||
			ac.Callee.Kind != callee.Kind {
			continue
		}
		// Function must match exactly (no wildcards)
		if ac.Function != caller.Function {
			continue
		}
		// Receiver must match exactly
		if ac.Receiver != caller.Receiver {
			continue
		}
		return true
	}
	return false
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

// CanonicalCheckDupcodeBypass is the single exported entry point.
func CanonicalCheckDupcodeBypass(root string, modulePath string) []checks.Finding {
	policy, err := NewDupcodeBypassPolicy(root, modulePath)
	if err != nil {
		return []checks.Finding{
			{Path: root, Kind: "canonical_root_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	// Phase 1: discover eligible files from filesystem
	discovered, err := policy.DiscoverProductionFilesRepoWide()
	if err != nil {
		return []checks.Finding{
			{Path: policy.repoRoot, Kind: "dupcode_discovery_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	// Phase 2: load packages
	cfg := &packages.Config{
		Dir:   policy.repoRoot,
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedModule,
		Tests: false,
		Fset:  token.NewFileSet(),
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return []checks.Finding{
			{Path: policy.repoRoot, Kind: "dupcode_package_load_error", Message: err.Error(), Severity: checks.SeverityError},
		}
	}

	// Phase 3: validate each package
	var findings []checks.Finding
	var typedFiles []string
	ignoredFiles := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}

		// Track ignored files
		for _, ignored := range pkg.IgnoredFiles {
			rel, _ := filepath.Rel(policy.repoRoot, ignored)
			ignoredFiles[rel] = true
		}

		// Phase 3a: fail closed on package errors
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_package_metadata_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
		}
		if len(pkg.TypeErrors) > 0 {
			for _, e := range pkg.TypeErrors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_type_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
		}
		if pkg.IllTyped {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_error",
				Message:  fmt.Sprintf("%s: ill-typed package", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
		}
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_info_error",
				Message:  fmt.Sprintf("%s: missing type info or file set", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			continue
		}

		// Phase 3b: analyze each compiled file using CompiledGoFiles
		for i, file := range pkg.Syntax {
			if i >= len(pkg.CompiledGoFiles) {
				continue
			}
			filename := pkg.CompiledGoFiles[i]
			rel, _ := filepath.Rel(policy.repoRoot, filename)
			typedFiles = append(typedFiles, rel)
			fileFindings := policy.analyzeFile(pkg, filename, file)
			findings = append(findings, fileFindings...)
		}
	}

	// Phase 4: reconcile discovery with typed files
	typedSet := make(map[string]bool)
	for _, f := range typedFiles {
		typedSet[f] = true
	}
	for _, d := range discovered {
		if !typedSet[d] && !ignoredFiles[d] {
			findings = append(findings, checks.Finding{
				Path: d, Kind: "dupcode_analysis_coverage_error",
				Message:  fmt.Sprintf("eligible file not in typed analysis: %s", d),
				Severity: checks.SeverityError,
			})
		}
	}

	// Phase 5: deterministic sort
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Message < findings[j].Message
	})

	return findings
}

// DiscoverProductionFilesRepoWide walks the repo and returns eligible files.
func (p *DupcodeBypassPolicy) DiscoverProductionFilesRepoWide() ([]string, error) {
	var files []string

	err := filepath.WalkDir(p.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == "node_modules" {
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

		rel, _ := filepath.Rel(p.repoRoot, path)
		files = append(files, rel)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

// analyzeFile analyzes a single file for dupcode bypass violations.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File) []checks.Finding {
	var findings []checks.Finding

	relPath, _ := filepath.Rel(p.repoRoot, filename)
	callerPath := pkg.PkgPath

	// Track files we've already reported function-value findings for (to avoid duplicates)
	reportedValueUses := make(map[token.Pos]bool)

	// Track enclosing caller using a stack
	type callerFrame struct {
		identity CallerIdentity
	}
	var stack []callerFrame

	push := func(id CallerIdentity) {
		stack = append(stack, callerFrame{id})
	}
	pop := func() {
		if len(stack) > 0 {
			stack = stack[:len(stack)]
		}
	}
	currentCaller := func() CallerIdentity {
		if len(stack) == 0 {
			return CallerIdentity{PackagePath: callerPath, Function: "<init>", Kind: "package_init"}
		}
		return stack[len(stack)-1].identity
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// Push function/method caller
			id := CallerIdentity{
				PackagePath: callerPath,
				Function:    x.Name.Name,
				Kind:        "package_function",
			}
			if x.Recv != nil {
				id.Receiver = recvTypeName(x.Recv)
				id.Kind = "method"
			}
			push(id)
			defer pop()
			return true

		case *ast.CallExpr:
			line := pkg.Fset.Position(x.Pos()).Line
			caller := currentCaller()

			// Selector call: x.Method()
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					baseObj := pkg.TypesInfo.ObjectOf(ident)
					if pkgName, ok := baseObj.(*types.PkgName); ok {
						importedPath := pkgName.Imported().Path()
						selObj := pkg.TypesInfo.ObjectOf(sel.Sel)
						if fn, ok := selObj.(*types.Func); ok {
							fnPkg := fn.Pkg()
							if fnPkg != nil {
								callee := ProtectedSymbol{
									PackagePath: fnPkg.Path(),
									Name:        fn.Name(),
									Kind:        ProtectedPackageFunction,
								}
								if fn.Type() != nil {
									if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
										callee.Kind = ProtectedMethod
										callee.Receiver = recvTypeNameFromType(sig.Recv())
									}
								}

								if p.isProtectedCall(caller, callee) {
									findings = append(findings, checks.Finding{
										Path: relPath, Kind: "dupcode_bypass",
										Message:  fmt.Sprintf("line %d: %s.%s called by %s.%s", line, importedPath, fn.Name(), caller.PackagePath, caller.Function),
										Severity: checks.SeverityError,
									})
								}
							}
						}
					}
				}
				return true
			}

			// Direct call: Function()
			if ident, ok := x.Fun.(*ast.Ident); ok {
				obj := pkg.TypesInfo.ObjectOf(ident)
				if fn, ok := obj.(*types.Func); ok {
					fnPkg := fn.Pkg()
					if fnPkg != nil {
						callee := ProtectedSymbol{
							PackagePath: fnPkg.Path(),
							Name:        fn.Name(),
							Kind:        ProtectedPackageFunction,
						}
						if fn.Type() != nil {
							if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
								callee.Kind = ProtectedMethod
								callee.Receiver = recvTypeNameFromType(sig.Recv())
							}
						}

						if p.isProtectedCall(caller, callee) {
							findings = append(findings, checks.Finding{
								Path: relPath, Kind: "dupcode_bypass",
								Message:  fmt.Sprintf("line %d: %s.%s called by %s.%s", line, fnPkg.Path(), fn.Name(), caller.PackagePath, caller.Function),
								Severity: checks.SeverityError,
							})
						}
					}
				}
			}
			return true

		case *ast.SelectorExpr:
			// Check for protected function-value capture: dupcode.CheckRepo (not called)
			if x.Sel.Name == "CheckRepo" || x.Sel.Name == "CheckReport" {
				if ident, ok := x.X.(*ast.Ident); ok {
					baseObj := pkg.TypesInfo.ObjectOf(ident)
					if pkgName, ok := baseObj.(*types.PkgName); ok {
						importedPath := pkgName.Imported().Path()
						if isProtectedPackage(importedPath) {
							// Confirm it's a protected symbol
							if _, ok := ProtectedSymbolsMap()[importedPath+"."+x.Sel.Name]; ok {
								if !reportedValueUses[x.Pos()] {
									reportedValueUses[x.Pos()] = true
									line := pkg.Fset.Position(x.Pos()).Line
									caller := currentCaller()
									findings = append(findings, checks.Finding{
										Path: relPath, Kind: "dupcode_protected_function_value",
										Message:  fmt.Sprintf("line %d: protected function value %s.%s captured by %s.%s", line, importedPath, x.Sel.Name, caller.PackagePath, caller.Function),
										Severity: checks.SeverityError,
									})
								}
							}
						}
					}
				}
			}
			return true

		case *ast.GenDecl:
			// Track variable initializers
			if x.Tok == token.VAR {
				for _, spec := range x.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok {
						if vspec.Values != nil {
							id := CallerIdentity{
								PackagePath: callerPath,
								Function:    "<var-init:" + vspec.Names[0].Name + ">",
								Kind:        "variable_initializer",
							}
							push(id)
							defer pop()
						}
					}
				}
			}
			return true

		case *ast.FuncLit:
			// Function literal
			id := CallerIdentity{
				PackagePath: callerPath,
				Function:    currentCaller().Function + "@literal",
				Kind:        "function_literal",
			}
			push(id)
			defer pop()
			return true
		}

		return true
	})

	return findings
}

func recvTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}

func recvTypeNameFromType(recv *types.Var) string {
	if recv == nil {
		return ""
	}
	if named, ok := recv.Type().(*types.Named); ok {
		return named.Obj().Name()
	}
	if ptr, ok := recv.Type().(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			return named.Obj().Name()
		}
	}
	return ""
}

func (p *DupcodeBypassPolicy) isProtectedCall(caller CallerIdentity, callee ProtectedSymbol) bool {
	key := callee.PackagePath + "." + callee.Name
	if _, ok := ProtectedSymbolsMap()[key]; !ok {
		return false
	}
	return !IsApprovedCaller(caller, callee)
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

// ProtectedSymbols defines exact protected symbols.
var ProtectedSymbols = []ProtectedSymbol{
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckRepo", Kind: ProtectedPackageFunction},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CheckReport", Kind: ProtectedPackageFunction},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "LoadBaseline", Kind: ProtectedPackageFunction},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "VerifyBaseline", Kind: ProtectedPackageFunction},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "WriteBaseline", Kind: ProtectedPackageFunction},
	{PackagePath: "github.com/s1onique/leamas/internal/factory/dupcode", Name: "CompareToBaseline", Kind: ProtectedPackageFunction},
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

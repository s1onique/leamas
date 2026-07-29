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
	ProtectedPackageFunction ProtectedSymbolKind = "package_function"
	ProtectedMethod          ProtectedSymbolKind = "method"
)

// ProtectedSymbol represents an exact protected declaration.
type ProtectedSymbol struct {
	PackagePath string
	Name        string
	Kind        ProtectedSymbolKind
	Receiver    string
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
	Kind        string
}

// IsApprovedCaller checks if a caller-callee edge is approved.
func IsApprovedCaller(caller CallerIdentity, callee ProtectedSymbol) bool {
	for _, ac := range ApprovedCallers {
		if ac.PackagePath != caller.PackagePath {
			continue
		}
		if ac.Callee.PackagePath != callee.PackagePath ||
			ac.Callee.Name != callee.Name ||
			ac.Callee.Kind != callee.Kind ||
			ac.Callee.Receiver != callee.Receiver {
			continue
		}
		// Function and Receiver must match exactly (no wildcards)
		if ac.Function == "" || ac.Function != caller.Function {
			continue
		}
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

	// Phase 1: discover eligible files from filesystem (fail-closed on errors)
	discovered, walkErr := policy.DiscoverProductionFilesRepoWide()
	if walkErr != nil {
		return []checks.Finding{
			{Path: policy.repoRoot, Kind: "dupcode_discovery_error", Message: walkErr.Error(), Severity: checks.SeverityError},
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
	typedFiles := make(map[string]bool)
	ignoredFiles := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}

		for _, ignored := range pkg.IgnoredFiles {
			rel, err := filepath.Rel(policy.repoRoot, ignored)
			if err != nil {
				continue
			}
			ignoredFiles[filepath.ToSlash(rel)] = true
		}

		// Fail closed on package errors
		packageInvalid := false
		if len(pkg.Errors) > 0 {
			for _, e := range pkg.Errors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_package_metadata_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
			packageInvalid = true
		}
		if len(pkg.TypeErrors) > 0 {
			for _, e := range pkg.TypeErrors {
				findings = append(findings, checks.Finding{
					Path: policy.repoRoot, Kind: "dupcode_type_error",
					Message:  fmt.Sprintf("%s: %v", pkg.PkgPath, e),
					Severity: checks.SeverityError,
				})
			}
			packageInvalid = true
		}
		if pkg.IllTyped {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_error",
				Message:  fmt.Sprintf("%s: ill-typed package", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			packageInvalid = true
		}
		if pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			findings = append(findings, checks.Finding{
				Path: policy.repoRoot, Kind: "dupcode_type_info_error",
				Message:  fmt.Sprintf("%s: missing type info or file set", pkg.PkgPath),
				Severity: checks.SeverityError,
			})
			packageInvalid = true
		}

		// Skip semantic enforcement if package is invalid
		if packageInvalid {
			continue
		}

		// Phase 3b: analyze each syntax tree via Fset filename mapping
		syntaxFiles := make(map[string]*ast.File)
		for _, file := range pkg.Syntax {
			pos := pkg.Fset.PositionFor(file.Pos(), true)
			if pos.Filename == "" {
				continue
			}
			syntaxFiles[filepath.ToSlash(pos.Filename)] = file
		}

		// Verify each CompiledGoFile has a syntax tree
		for _, compiled := range pkg.CompiledGoFiles {
			rel, err := filepath.Rel(policy.repoRoot, compiled)
			if err != nil {
				continue
			}
			relSlash := filepath.ToSlash(rel)
			if _, ok := syntaxFiles[filepath.ToSlash(compiled)]; !ok {
				// CompiledGoFiles missing syntax tree — try using rel path
				if _, ok := syntaxFiles[relSlash]; !ok {
					findings = append(findings, checks.Finding{
						Path: relSlash, Kind: "dupcode_analysis_coverage_error",
						Message:  fmt.Sprintf("compiled file missing syntax: %s", relSlash),
						Severity: checks.SeverityError,
					})
				}
			}
		}

		for filename, file := range syntaxFiles {
			rel, err := filepath.Rel(policy.repoRoot, filename)
			if err != nil {
				continue
			}
			relSlash := filepath.ToSlash(rel)
			typedFiles[relSlash] = true

			fileFindings := policy.analyzeFile(pkg, filename, file)
			findings = append(findings, fileFindings...)
		}
	}

	// Phase 4: reconcile discovery with typed files
	for _, d := range discovered {
		if !typedFiles[d] {
			reason := "missing_from_packages"
			if ignoredFiles[d] {
				reason = "build_ignored"
			}
			findings = append(findings, checks.Finding{
				Path: d, Kind: "dupcode_analysis_coverage_error",
				Message:  fmt.Sprintf("eligible file not analyzed (reason=%s): %s", reason, d),
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
// Fails closed on traversal errors.
func (p *DupcodeBypassPolicy) DiscoverProductionFilesRepoWide() ([]string, error) {
	var files []string

	err := filepath.WalkDir(p.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
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

		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(p.repoRoot, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	sort.Strings(files)
	return files, nil
}

// callerIdentity derives the enclosing caller using the ancestor stack.
// Precedence (nearest first): function literal > method > package function > variable initializer > package init
func callerIdentity(pkgPath string, ancestors []ast.Node, fset *token.FileSet) CallerIdentity {
	for i := len(ancestors) - 1; i >= 0; i-- {
		switch a := ancestors[i].(type) {
		case *ast.FuncLit:
			pos := fset.Position(a.Pos())
			return CallerIdentity{
				PackagePath: pkgPath,
				Function:    fmt.Sprintf("func@%d:%d", pos.Line, pos.Column),
				Kind:        "function_literal",
			}
		case *ast.FuncDecl:
			id := CallerIdentity{
				PackagePath: pkgPath,
				Function:    a.Name.Name,
				Kind:        "package_function",
			}
			if a.Recv != nil {
				id.Receiver = recvTypeNameFromAST(a.Recv)
				id.Kind = "method"
			}
			return id
		}
	}
	// Check for variable initializer
	for i := len(ancestors) - 1; i >= 0; i-- {
		if gd, ok := ancestors[i].(*ast.GenDecl); ok {
			if gd.Tok == token.VAR {
				for _, spec := range gd.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok && vspec.Values != nil {
						return CallerIdentity{
							PackagePath: pkgPath,
							Function:    "<var-init:" + vspec.Names[0].Name + ">",
							Kind:        "variable_initializer",
						}
					}
				}
			}
		}
	}
	return CallerIdentity{PackagePath: pkgPath, Function: "<init>", Kind: "package_init"}
}

// resolveProtectedUse resolves a protected object from an expression.
func (p *DupcodeBypassPolicy) resolveProtectedUse(pkg *packages.Package, expr ast.Expr) (ProtectedSymbol, bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			baseObj := pkg.TypesInfo.ObjectOf(ident)
			if pkgName, ok := baseObj.(*types.PkgName); ok {
				importedPath := pkgName.Imported().Path()
				selObj := pkg.TypesInfo.ObjectOf(e.Sel)
				if fn, ok := selObj.(*types.Func); ok {
					fnPkg := fn.Pkg()
					if fnPkg != nil && fnPkg.Path() == importedPath {
						callee := ProtectedSymbol{
							PackagePath: fnPkg.Path(),
							Name:        fn.Name(),
							Kind:        ProtectedPackageFunction,
						}
						if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
							callee.Kind = ProtectedMethod
							callee.Receiver = recvTypeNameFromSig(sig.Recv())
						}
						// Verify this is a configured protected symbol
						if _, ok := ProtectedSymbolsMap()[callee.PackagePath+"."+callee.Name]; ok {
							return callee, true
						}
					}
				}
			}
		}
	case *ast.Ident:
		obj := pkg.TypesInfo.ObjectOf(e)
		if fn, ok := obj.(*types.Func); ok {
			fnPkg := fn.Pkg()
			if fnPkg != nil {
				callee := ProtectedSymbol{
					PackagePath: fnPkg.Path(),
					Name:        fn.Name(),
					Kind:        ProtectedPackageFunction,
				}
				if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
					callee.Kind = ProtectedMethod
					callee.Receiver = recvTypeNameFromSig(sig.Recv())
				}
				if _, ok := ProtectedSymbolsMap()[callee.PackagePath+"."+callee.Name]; ok {
					return callee, true
				}
			}
		}
	}
	return ProtectedSymbol{}, false
}

// analyzeFile uses PreorderStack to track caller via ancestor stack.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File) []checks.Finding {
	var findings []checks.Finding
	relPath, _ := filepath.Rel(p.repoRoot, filename)

	// Track which CallExpr.Fun selectors/idents we've already classified as calls
	// (so we don't double-report as function-value captures)
	calledPos := make(map[token.Pos]bool)

	// First pass: collect all CallExpr.Fun positions
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			calledPos[call.Fun.Pos()] = true
		}
		return true
	})

	// Use PreorderStack for proper caller tracking
	ast.PreorderStack(file, nil, func(n ast.Node, ancestors []ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)
			callee, ok := p.resolveProtectedUse(pkg, node.Fun)
			if !ok {
				return true
			}
			if IsApprovedCaller(caller, callee) {
				return true
			}
			line := pkg.Fset.Position(node.Pos()).Line
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "dupcode_bypass",
				Message:  fmt.Sprintf("line %d: %s.%s called by %s.%s", line, callee.PackagePath, callee.Name, caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
		case *ast.SelectorExpr, *ast.Ident:
			// Skip if this expression is used as CallExpr.Fun (already reported as direct call)
			if calledPos[node.Pos()] {
				return true
			}
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)
			callee, ok := p.resolveProtectedUse(pkg, node.(ast.Expr))
			if !ok {
				return true
			}
			if IsApprovedCaller(caller, callee) {
				return true
			}
			line := pkg.Fset.Position(node.Pos()).Line
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: "dupcode_protected_function_value",
				Message:  fmt.Sprintf("line %d: protected function value %s.%s captured by %s.%s", line, callee.PackagePath, callee.Name, caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
		}
		return true
	})

	return findings
}

func recvTypeNameFromAST(recv *ast.FieldList) string {
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

func recvTypeNameFromSig(recv *types.Var) string {
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

// ProtectedSymbolsMap returns a map for fast lookup (package.path + . + name).
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

func isProtectedPackage(path string) bool {
	for _, prefix := range DupcodeProtectedPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

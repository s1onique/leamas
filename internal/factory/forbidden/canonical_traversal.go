// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/s1onique/leamas/internal/factory/checks"
	"golang.org/x/tools/go/packages"
)

// callerIdentity derives the enclosing caller using the ancestor stack.
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
	for i := len(ancestors) - 1; i >= 0; i-- {
		if gd, ok := ancestors[i].(*ast.GenDecl); ok {
			if gd.Tok == token.VAR {
				for _, spec := range gd.Specs {
					if vspec, ok := spec.(*ast.ValueSpec); ok && vspec.Values != nil {
						if len(vspec.Names) > 0 {
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
	}
	return CallerIdentity{PackagePath: pkgPath, Function: "<init>", Kind: "package_init"}
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

// referenceClass classifies a protected reference in source.
type referenceClass string

const (
	refDirectCall       referenceClass = "DIRECT_CALL"
	refFunctionValue    referenceClass = "FUNCTION_VALUE"
	refMethodValue      referenceClass = "METHOD_VALUE"
	refMethodExpression referenceClass = "METHOD_EXPRESSION"
	refPackageVariable  referenceClass = "PACKAGE_VARIABLE_REFERENCE"
	refDeclaration      referenceClass = "DECLARATION"
)

// resolveProtectedUse resolves a protected object (any layer) from an expression.
// It only processes USE-site identifiers, never declarations.
func (p *DupcodeBypassPolicy) resolveProtectedUse(pkg *packages.Package, expr ast.Expr, class referenceClass) (ProtectedSymbol, bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// Case 1: package-qualified selector (protectedverifier.X)
		if ident, ok := e.X.(*ast.Ident); ok {
			// USE site only - use Uses map
			useObj, isUse := pkg.TypesInfo.Uses[ident]
			if !isUse || useObj == nil {
				return ProtectedSymbol{}, false
			}
			if pkgName, ok := useObj.(*types.PkgName); ok {
				importedPath := pkgName.Imported().Path()
				selObj, isUseSel := pkg.TypesInfo.Uses[e.Sel]
				if !isUseSel || selObj == nil {
					return ProtectedSymbol{}, false
				}
				if fn, ok := selObj.(*types.Func); ok {
					fnPkg := fn.Pkg()
					if fnPkg != nil && fnPkg.Path() == importedPath {
						if sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), fn.Name()); sym != nil {
							callee := *sym
							applyReceiver(&callee, fn)
							return callee, true
						}
						if sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name()); sym != nil {
							callee := *sym
							applyReceiver(&callee, fn)
							return callee, true
						}
					}
				}
				if v, ok := selObj.(*types.Var); ok {
					vPkg := v.Pkg()
					if vPkg != nil && vPkg.Path() == importedPath {
						if sym := findProtectedSymbolVariable(AuthorityLayerAdapter, vPkg.Path(), v.Name()); sym != nil {
							return *sym, true
						}
						if sym := findProtectedSymbolVariable(AuthorityLayerRaw, vPkg.Path(), v.Name()); sym != nil {
							return *sym, true
						}
					}
				}
			}
		}
		// Case 2: method selection (runner.Method or type.Method expression)
		if sel, ok := pkg.TypesInfo.Selections[e]; ok {
			if fn, ok := sel.Obj().(*types.Func); ok {
				fnPkg := fn.Pkg()
				if fnPkg != nil {
					if sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name()); sym != nil {
						callee := *sym
						recv := recvFromSelection(sel)
						if recv != nil {
							callee.Receiver = recvTypeNameFromSig(recv)
						}
						return callee, true
					}
				}
			}
		}
	case *ast.Ident:
		// USE site only - skip identifiers that are declarations (Defs map)
		if _, isDef := pkg.TypesInfo.Defs[e]; isDef {
			return ProtectedSymbol{}, false
		}
		useObj, isUse := pkg.TypesInfo.Uses[e]
		if !isUse || useObj == nil {
			return ProtectedSymbol{}, false
		}
		if fn, ok := useObj.(*types.Func); ok {
			fnPkg := fn.Pkg()
			if fnPkg != nil {
				if sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), fn.Name()); sym != nil {
					callee := *sym
					applyReceiver(&callee, fn)
					return callee, true
				}
				if sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name()); sym != nil {
					callee := *sym
					applyReceiver(&callee, fn)
					return callee, true
				}
			}
		}
		if v, ok := useObj.(*types.Var); ok {
			vPkg := v.Pkg()
			if vPkg != nil {
				if sym := findProtectedSymbolVariable(AuthorityLayerAdapter, vPkg.Path(), v.Name()); sym != nil {
					return *sym, true
				}
			}
		}
	}
	return ProtectedSymbol{}, false
}

// applyReceiver sets the receiver if the function has one.
func applyReceiver(callee *ProtectedSymbol, fn *types.Func) {
	if fn.Type() != nil {
		if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
			callee.Kind = ProtectedMethod
			callee.Receiver = recvTypeNameFromSig(sig.Recv())
		}
	}
}

// recvFromSelection extracts the receiver from a types.Selection.
func recvFromSelection(sel *types.Selection) *types.Var {
	switch sel.Kind() {
	case types.MethodVal, types.MethodExpr:
		if fn, ok := sel.Obj().(*types.Func); ok {
			if sig, ok := fn.Type().(*types.Signature); ok {
				return sig.Recv()
			}
		}
	}
	return nil
}

// findProtectedSymbol looks up a protected function symbol across both layers.
func findProtectedSymbol(layer AuthorityLayer, pkgPath, name string) *ProtectedSymbol {
	for _, sym := range ProtectedSymbols {
		if sym.Layer == layer && sym.PackagePath == pkgPath && sym.Name == name {
			return &sym
		}
	}
	for _, sym := range AdapterProtectedSymbols {
		if sym.Layer == layer && sym.PackagePath == pkgPath && sym.Name == name {
			return &sym
		}
	}
	return nil
}

// findProtectedSymbolVariable looks up a protected variable symbol across both layers.
func findProtectedSymbolVariable(layer AuthorityLayer, pkgPath, name string) *ProtectedSymbol {
	for _, sym := range AdapterProtectedSymbols {
		if sym.Layer == layer && sym.PackagePath == pkgPath && sym.Name == name {
			return &sym
		}
	}
	return nil
}

// analyzeFile uses PreorderStack to track caller via ancestor stack.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File) []checks.Finding {
	var findings []checks.Finding
	relPath, _ := filepath.Rel(p.repoRoot, filename)

	// Track which direct-call callee expressions have already been classified,
	// so they don't produce a second function-value finding.
	calledCallees := make(map[token.Pos]bool)

	// Phase 1: detect dot imports (independent of use detection).
	for _, imp := range file.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			path := strings.Trim(imp.Path.Value, "\"")
			if isProtectedPackage(path) || isAdapterProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path:     relPath,
					Kind:     "dupcode_dot_import",
					Message:  fmt.Sprintf("line %d: dot import of protected package %s is forbidden", pkg.Fset.Position(imp.Pos()).Line, path),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Phase 2: classify protected references.
	ast.PreorderStack(file, nil, func(n ast.Node, ancestors []ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Direct call
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)
			callee, ok := p.resolveProtectedUse(pkg, node.Fun, refDirectCall)
			if !ok {
				return true
			}
			if IsApprovedCaller(caller, callee) {
				return true
			}
			line := pkg.Fset.Position(node.Pos()).Line
			kind := "dupcode_bypass"
			if callee.Layer == AuthorityLayerAdapter {
				kind = "dupcode_adapter_bypass"
			}
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: kind,
				Message:  fmt.Sprintf("line %d: %s.%s called by %s.%s", line, callee.PackagePath, callee.Name, caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
			// Mark the callee expression position so a child selector
			// or ident does not produce a duplicate function-value finding.
			calledCallees[node.Fun.Pos()] = true
			// Prune descent into the callee expression.
			return true
		case *ast.SelectorExpr, *ast.Ident:
			// Skip if this expression is the callee of a direct call (already reported)
			if calledCallees[node.Pos()] {
				return true
			}
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)
			class := refFunctionValue
			if _, ok := node.(*ast.SelectorExpr); ok {
				class = refMethodValue
			}
			callee, ok := p.resolveProtectedUse(pkg, node.(ast.Expr), class)
			if !ok {
				return true
			}
			if IsApprovedCaller(caller, callee) {
				return true
			}
			line := pkg.Fset.Position(node.Pos()).Line
			kind := "dupcode_protected_function_value"
			if callee.Layer == AuthorityLayerAdapter {
				kind = "dupcode_adapter_function_value"
			}
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: kind,
				Message:  fmt.Sprintf("line %d: protected function value %s.%s captured by %s.%s", line, callee.PackagePath, callee.Name, caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
		}
		return true
	})

	return findings
}

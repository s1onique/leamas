// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"

	"github.com/s1onique/leamas/internal/factory/checks"
	"golang.org/x/tools/go/packages"
)

// callerIdentity derives the enclosing caller using the ancestor stack.
// Precedence (nearest first): function literal > method > package function > variable initializer > package init.
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

// resolveProtectedUse resolves a protected object (any layer) from an expression.
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
						// Check raw layer
						if sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), fn.Name()); sym != nil {
							callee := *sym
							if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
								callee.Kind = ProtectedMethod
								callee.Receiver = recvTypeNameFromSig(sig.Recv())
							}
							return callee, true
						}
						// Check adapter layer
						if sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name()); sym != nil {
							callee := *sym
							if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
								callee.Kind = ProtectedMethod
								callee.Receiver = recvTypeNameFromSig(sig.Recv())
							}
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
				if sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), fn.Name()); sym != nil {
					callee := *sym
					if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
						callee.Kind = ProtectedMethod
						callee.Receiver = recvTypeNameFromSig(sig.Recv())
					}
					return callee, true
				}
				if sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name()); sym != nil {
					callee := *sym
					if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
						callee.Kind = ProtectedMethod
						callee.Receiver = recvTypeNameFromSig(sig.Recv())
					}
					return callee, true
				}
			}
		}
	}
	return ProtectedSymbol{}, false
}

// findProtectedSymbol looks up a protected symbol across both layers.
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

// analyzeFile uses PreorderStack to track caller via ancestor stack.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File) []checks.Finding {
	var findings []checks.Finding
	relPath, _ := filepath.Rel(p.repoRoot, filename)

	calledPos := make(map[token.Pos]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			calledPos[call.Fun.Pos()] = true
		}
		return true
	})

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
			kind := "dupcode_bypass"
			if callee.Layer == AuthorityLayerAdapter {
				kind = "dupcode_adapter_bypass"
			}
			findings = append(findings, checks.Finding{
				Path: relPath, Kind: kind,
				Message:  fmt.Sprintf("line %d: %s.%s called by %s.%s", line, callee.PackagePath, callee.Name, caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
		case *ast.SelectorExpr, *ast.Ident:
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

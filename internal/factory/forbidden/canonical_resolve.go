// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

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
//
// The class argument states the structural role of the observed expression.
// When the resolved types.Object does not match the requested class the
// resolution is rejected; this enforces the configured layer/kind semantics
// independent of descriptive metadata.
func (p *DupcodeBypassPolicy) resolveProtectedUse(pkg *packages.Package, expr ast.Expr, class referenceClass) (ProtectedSymbol, bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// Case 1: package-qualified selector (protectedverifier.X)
		if ident, ok := e.X.(*ast.Ident); ok {
			useObj, isUse := pkg.TypesInfo.Uses[ident]
			if !isUse || useObj == nil {
				return ProtectedSymbol{}, false
			}
			pkgName, ok := useObj.(*types.PkgName)
			if !ok {
				// X is not a package identifier — fall through to method selection.
			} else {
				importedPath := pkgName.Imported().Path()
				selObj, isUseSel := pkg.TypesInfo.Uses[e.Sel]
				if !isUseSel || selObj == nil {
					return ProtectedSymbol{}, false
				}
				switch sel := selObj.(type) {
				case *types.Func:
					fnPkg := sel.Pkg()
					if fnPkg == nil || fnPkg.Path() != importedPath {
						return ProtectedSymbol{}, false
					}
					sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), sel.Name())
					if sym == nil {
						sym = findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), sel.Name())
					}
					if sym == nil {
						return ProtectedSymbol{}, false
					}
					callee := *sym
					if !classMatchesFunc(class, sel) {
						return ProtectedSymbol{}, false
					}
					if sig, ok := sel.Type().(*types.Signature); ok && sig.Recv() != nil {
						callee.Kind = ProtectedMethod
						callee.Receiver = recvTypeNameFromSig(sig.Recv())
					}
					return callee, true
				case *types.Var:
					vPkg := sel.Pkg()
					if vPkg == nil || vPkg.Path() != importedPath {
						return ProtectedSymbol{}, false
					}
					sym := findProtectedSymbolVariable(AuthorityLayerAdapter, vPkg.Path(), sel.Name())
					if sym == nil {
						sym = findProtectedSymbolVariable(AuthorityLayerRaw, vPkg.Path(), sel.Name())
					}
					if sym == nil {
						return ProtectedSymbol{}, false
					}
					if class != refPackageVariable && class != refFunctionValue && class != refDirectCall {
						return ProtectedSymbol{}, false
					}
					return *sym, true
				}
			}
		}
		// Case 2: method selection (runner.Method or type.Method expression)
		sel, ok := pkg.TypesInfo.Selections[e]
		if !ok {
			return ProtectedSymbol{}, false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			return ProtectedSymbol{}, false
		}
		fnPkg := fn.Pkg()
		if fnPkg == nil {
			return ProtectedSymbol{}, false
		}
		sym := findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), fn.Name())
		if sym == nil {
			return ProtectedSymbol{}, false
		}
		// Validate class against actual selection kind.
		switch class {
		case refDirectCall:
			if sel.Kind() != types.MethodVal {
				return ProtectedSymbol{}, false
			}
		case refMethodValue:
			if sel.Kind() != types.MethodVal {
				return ProtectedSymbol{}, false
			}
		case refMethodExpression:
			if sel.Kind() != types.MethodExpr {
				return ProtectedSymbol{}, false
			}
		default:
			return ProtectedSymbol{}, false
		}
		callee := *sym
		if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
			callee.Receiver = recvTypeNameFromSig(sig.Recv())
		}
		return callee, true

	case *ast.Ident:
		// USE site only - skip identifiers that are declarations (Defs map)
		if _, isDef := pkg.TypesInfo.Defs[e]; isDef {
			return ProtectedSymbol{}, false
		}
		useObj, isUse := pkg.TypesInfo.Uses[e]
		if !isUse || useObj == nil {
			return ProtectedSymbol{}, false
		}
		switch o := useObj.(type) {
		case *types.Func:
			fnPkg := o.Pkg()
			if fnPkg == nil {
				return ProtectedSymbol{}, false
			}
			sym := findProtectedSymbol(AuthorityLayerRaw, fnPkg.Path(), o.Name())
			if sym == nil {
				sym = findProtectedSymbol(AuthorityLayerAdapter, fnPkg.Path(), o.Name())
			}
			if sym == nil {
				return ProtectedSymbol{}, false
			}
			if !classMatchesFunc(class, o) {
				return ProtectedSymbol{}, false
			}
			callee := *sym
			if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
				callee.Kind = ProtectedMethod
				callee.Receiver = recvTypeNameFromSig(sig.Recv())
			}
			return callee, true
		case *types.Var:
			vPkg := o.Pkg()
			if vPkg == nil {
				return ProtectedSymbol{}, false
			}
			sym := findProtectedSymbolVariable(AuthorityLayerAdapter, vPkg.Path(), o.Name())
			if sym == nil {
				sym = findProtectedSymbolVariable(AuthorityLayerRaw, vPkg.Path(), o.Name())
			}
			if sym == nil {
				return ProtectedSymbol{}, false
			}
			if class != refPackageVariable && class != refFunctionValue && class != refDirectCall {
				return ProtectedSymbol{}, false
			}
			return *sym, true
		}
	}
	return ProtectedSymbol{}, false
}

// classMatchesFunc validates that the requested referenceClass is consistent
// with the actual declaration type. Package functions can only be called as
// direct calls or captured as function values; methods can be called, captured
// as method values, or referenced as method expressions.
func classMatchesFunc(class referenceClass, fn *types.Func) bool {
	sig, _ := fn.Type().(*types.Signature)
	hasRecv := sig != nil && sig.Recv() != nil
	switch class {
	case refDirectCall, refFunctionValue:
		// Either form is allowed; we don't constrain on declaration shape
		// here because the resolver already located a configured symbol.
		return true
	case refMethodValue, refMethodExpression:
		return hasRecv
	case refPackageVariable:
		return false
	default:
		return false
	}
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

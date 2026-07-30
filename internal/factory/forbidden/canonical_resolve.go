// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func (a *canonicalAnalysis) protectedObjectForExpr(
	pkg *packages.Package,
	expression ast.Expr,
) (types.Object, ProtectedSymbol, bool) {
	switch typed := expression.(type) {
	case *ast.ParenExpr:
		return a.protectedObjectForExpr(pkg, typed.X)
	case *ast.IndexExpr:
		return a.protectedObjectForExpr(pkg, typed.X)
	case *ast.IndexListExpr:
		return a.protectedObjectForExpr(pkg, typed.X)
	case *ast.SelectorExpr:
		if selection := pkg.TypesInfo.Selections[typed]; selection != nil {
			object := selection.Obj()
			symbol, ok := a.protectedByObject[object]
			return object, symbol, ok
		}
		object := pkg.TypesInfo.Uses[typed.Sel]
		if object == nil {
			return nil, ProtectedSymbol{}, false
		}
		symbol, ok := a.protectedByObject[object]
		return object, symbol, ok
	case *ast.Ident:
		if pkg.TypesInfo.Defs[typed] != nil {
			return nil, ProtectedSymbol{}, false
		}
		object := pkg.TypesInfo.Uses[typed]
		if object == nil {
			return nil, ProtectedSymbol{}, false
		}
		symbol, ok := a.protectedByObject[object]
		return object, symbol, ok
	default:
		return nil, ProtectedSymbol{}, false
	}
}

func referenceClassFor(
	pkg *packages.Package,
	expression ast.Expr,
	object types.Object,
	direct bool,
) ReferenceClass {
	if _, variable := object.(*types.Var); variable {
		return refPackageVariable
	}
	if direct {
		if selector, ok := unwrappedSelector(expression); ok {
			if selection := pkg.TypesInfo.Selections[selector]; selection != nil && selection.Kind() == types.MethodExpr {
				return refMethodExpression
			}
		}
		return refDirectCall
	}
	if function, ok := object.(*types.Func); ok {
		signature, _ := function.Type().(*types.Signature)
		if signature != nil && signature.Recv() != nil {
			if selector, ok := unwrappedSelector(expression); ok {
				if selection := pkg.TypesInfo.Selections[selector]; selection != nil && selection.Kind() == types.MethodExpr {
					return refMethodExpression
				}
			}
			return refMethodValue
		}
	}
	return refFunctionValue
}

func unwrappedSelector(expression ast.Expr) (*ast.SelectorExpr, bool) {
	switch typed := expression.(type) {
	case *ast.SelectorExpr:
		return typed, true
	case *ast.ParenExpr:
		return unwrappedSelector(typed.X)
	case *ast.IndexExpr:
		return unwrappedSelector(typed.X)
	case *ast.IndexListExpr:
		return unwrappedSelector(typed.X)
	default:
		return nil, false
	}
}

func classMatchesFunc(class ReferenceClass, function *types.Func) bool {
	signature, _ := function.Type().(*types.Signature)
	hasReceiver := signature != nil && signature.Recv() != nil
	switch class {
	case refDirectCall, refFunctionValue, refDotImport:
		return true
	case refMethodValue, refMethodExpression:
		return hasReceiver
	default:
		return false
	}
}

func findProtectedSymbol(layer AuthorityLayer, pkgPath, name string) *ProtectedSymbol {
	for _, symbol := range allProtectedSymbols() {
		if symbol.Layer == layer && symbol.PackagePath == pkgPath && symbol.Name == name {
			copy := symbol
			return &copy
		}
	}
	return nil
}

func findProtectedSymbolVariable(layer AuthorityLayer, pkgPath, name string) *ProtectedSymbol {
	for _, symbol := range allProtectedSymbols() {
		if symbol.Layer == layer && symbol.PackagePath == pkgPath && symbol.Name == name && symbol.Kind == ProtectedPackageVariable {
			copy := symbol
			return &copy
		}
	}
	return nil
}

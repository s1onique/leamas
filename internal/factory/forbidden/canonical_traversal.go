// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/packages"
)

func (a *canonicalAnalysis) scanProtectedReferences() {
	for _, pkg := range a.packages {
		for _, syntax := range a.syntaxFiles(pkg) {
			a.scanFile(pkg, syntax.filename, syntax.file)
		}
	}
}

func (a *canonicalAnalysis) scanFile(pkg *packages.Package, filename string, file *ast.File) {
	path := a.relativePath(filename)
	dotImports := a.classifyDotImports(pkg, path, file)

	ast.PreorderStack(file, nil, func(node ast.Node, ancestors []ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CallExpr:
			object, symbol, protected := a.protectedObjectForExpr(pkg, typed.Fun)
			if !protected {
				return true
			}
			a.markDirectCallee(pkg.Fset, typed.Fun)
			class := referenceClassFor(pkg, typed.Fun, object, true)
			if objectFromDotImport(object, dotImports) {
				class = refDotImport
			}
			a.observeEdge(pkg, path, typed.Fun.Pos(), ancestors, object, symbol, class)
		case *ast.SelectorExpr:
			if a.insideDirectCallee(pkg.Fset, typed.Pos()) {
				return true
			}
			object, symbol, protected := a.protectedObjectForExpr(pkg, typed)
			if !protected {
				return true
			}
			class := referenceClassFor(pkg, typed, object, false)
			if objectFromDotImport(object, dotImports) {
				class = refDotImport
			}
			a.observeEdge(pkg, path, typed.Pos(), ancestors, object, symbol, class)
		case *ast.Ident:
			if a.insideDirectCallee(pkg.Fset, typed.Pos()) || identifierIsSelectorChild(typed, ancestors) {
				return true
			}
			object, symbol, protected := a.protectedObjectForExpr(pkg, typed)
			if !protected {
				return true
			}
			class := referenceClassFor(pkg, typed, object, false)
			if objectFromDotImport(object, dotImports) {
				class = refDotImport
			}
			a.observeEdge(pkg, path, typed.Pos(), ancestors, object, symbol, class)
		}
		return true
	})
}

func (a *canonicalAnalysis) classifyDotImports(
	pkg *packages.Package,
	path string,
	file *ast.File,
) map[string]bool {
	dotImports := make(map[string]bool)
	for _, declaration := range file.Imports {
		if declaration.Name == nil || declaration.Name.Name != "." {
			continue
		}
		importPath, err := strconv.Unquote(declaration.Path.Value)
		position := pkg.Fset.PositionFor(declaration.Pos(), true)
		if err != nil {
			a.addFinding(path, "dupcode_import_path_error", fmt.Sprintf("malformed import literal: %v", err), position, CallerIdentity{}, ProtectedSymbol{}, refDotImport)
			continue
		}
		if !a.configuresPackage(importPath) {
			continue
		}
		dotImports[importPath] = true
		a.addFinding(path, "dupcode_dot_import", fmt.Sprintf("dot import of protected package %s is forbidden", importPath), position, CallerIdentity{}, ProtectedSymbol{PackagePath: importPath}, refDotImport)
	}
	return dotImports
}

func (a *canonicalAnalysis) configuresPackage(path string) bool {
	for _, symbol := range a.config.protected {
		if symbol.PackagePath == path {
			return true
		}
	}
	return false
}

func objectFromDotImport(object types.Object, dotImports map[string]bool) bool {
	return object != nil && object.Pkg() != nil && dotImports[object.Pkg().Path()]
}

// observeEdge records one globally typed protected source reference.
// When the resolved caller is an anonymous function literal:
//
//   - the actual caller identity remains CallerKindFunctionLiteral
//     with a nil callerObject (a function literal has no declared
//     name in the package symbol table and therefore no caller
//     authority object);
//   - the enclosing named declaration (if any) is captured separately
//     on the edge for diagnostic and cascade-isolation purposes only.
//
// The outer caller identity and outer caller object NEVER participate
// in ordinary approval matching, cardinality accounting, or
// stale-approval checks. They are used only when the edge's internal
// reference class is invalid, to poison matching approval states.
func (a *canonicalAnalysis) observeEdge(
	pkg *packages.Package,
	path string,
	position token.Pos,
	ancestors []ast.Node,
	calleeObject types.Object,
	callee ProtectedSymbol,
	class ReferenceClass,
) {
	caller, callerObject := a.callerForUse(pkg, ancestors, position)
	edge := ObservedEdge{
		Caller:         caller,
		Callee:         callee,
		ReferenceClass: class,
		Path:           path,
		Position:       pkg.Fset.PositionFor(position, true),
		callerObject:   callerObject,
		calleeObject:   calleeObject,
	}
	if caller.Kind == CallerKindFunctionLiteral {
		if outer, outerObject, ok := outerNamedCallerFromAncestors(pkg, ancestors); ok {
			edge.outerCaller = outer
			edge.hasOuterCaller = true
			edge.outerCallerObject = outerObject
		}
	}
	a.observedEdges = append(a.observedEdges, edge)
}

func (a *canonicalAnalysis) markDirectCallee(fileSet *token.FileSet, expression ast.Expr) {
	if fileSet == nil || expression == nil {
		return
	}
	a.directCalleeRanges[fileSet] = append(a.directCalleeRanges[fileSet], sourceRange{
		start: expression.Pos(),
		end:   expression.End(),
	})
}

func (a *canonicalAnalysis) insideDirectCallee(fileSet *token.FileSet, position token.Pos) bool {
	for _, source := range a.directCalleeRanges[fileSet] {
		if position >= source.start && position <= source.end {
			return true
		}
	}
	return false
}

func identifierIsSelectorChild(identifier *ast.Ident, ancestors []ast.Node) bool {
	if len(ancestors) == 0 {
		return false
	}
	selector, ok := ancestors[len(ancestors)-1].(*ast.SelectorExpr)
	return ok && (selector.Sel == identifier || selector.X == identifier)
}

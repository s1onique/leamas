// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func (a *canonicalAnalysis) resolveCallerDeclarations() {
	for _, pkg := range a.packages {
		for _, syntax := range a.syntaxFiles(pkg) {
			for _, declaration := range syntax.file.Decls {
				switch typed := declaration.(type) {
				case *ast.FuncDecl:
					object, _ := pkg.TypesInfo.Defs[typed.Name].(*types.Func)
					if object == nil {
						continue
					}
					identity := callerIdentityFromFunction(pkg.PkgPath, object)
					a.addCallerCandidate(identity, object)
				case *ast.GenDecl:
					if typed.Tok != token.VAR {
						continue
					}
					for _, specification := range typed.Specs {
						value, ok := specification.(*ast.ValueSpec)
						if !ok || len(value.Values) == 0 {
							continue
						}
						for _, name := range value.Names {
							object, _ := pkg.TypesInfo.Defs[name].(*types.Var)
							if object == nil || object.Parent() != pkg.Types.Scope() {
								continue
							}
							identity := CallerIdentity{
								PackagePath: pkg.PkgPath,
								Function:    "<var-init:" + name.Name + ">",
								Kind:        CallerKindVariableInitializer,
							}
							a.addCallerCandidate(identity, object)
						}
					}
				}
			}
		}
	}
	for identity, candidates := range a.callerCandidates {
		if len(candidates) == 1 {
			a.callersByIdentity[identity] = candidates[0]
		}
	}
}

func callerIdentityFromFunction(pkgPath string, function *types.Func) CallerIdentity {
	identity := CallerIdentity{
		PackagePath: pkgPath,
		Function:    function.Name(),
		Kind:        CallerKindPackageFunction,
	}
	if function.Name() == "init" {
		identity.Function = "<init>"
		identity.Kind = CallerKindPackageInit
		return identity
	}
	signature, _ := function.Type().(*types.Signature)
	if signature != nil && signature.Recv() != nil {
		identity.Receiver = recvTypeNameFromSig(signature.Recv())
		identity.Kind = CallerKindMethod
	}
	return identity
}

func (a *canonicalAnalysis) addCallerCandidate(identity CallerIdentity, object types.Object) {
	for _, existing := range a.callerCandidates[identity] {
		if existing == object {
			return
		}
	}
	a.callerCandidates[identity] = append(a.callerCandidates[identity], object)
}

// callerForUse finds the nearest enclosing executable scope for a
// protected source reference. The semantic boundary is fixed:
//
//	named declaration      → exact configured approval
//	anonymous function literal → separate execution scope
//	                            cannot inherit outer approval
//	                            cannot be configured by source coordinates
//	                            protected use fails closed
//
// When a *ast.FuncLit occurs between the protected use and an outer
// *ast.FuncDecl, the function literal is the caller scope. The traversal
// MUST NOT skip the FuncLit and attribute the use to the outer function.
//
// The returned types.Object is non-nil only for named declarations:
// package functions, methods, and variable initializers. For a function
// literal the caller has no declaration object; the caller object is
// nil. The outerCallerObject on the edge is the only object that may
// be used for cascade isolation.
func (a *canonicalAnalysis) callerForUse(
	pkg *packages.Package,
	ancestors []ast.Node,
	position token.Pos,
) (CallerIdentity, types.Object) {
	for index := len(ancestors) - 1; index >= 0; index-- {
		switch typed := ancestors[index].(type) {
		case *ast.FuncLit:
			// Anonymous function literal is the nearest enclosing
			// executable scope. It never inherits the approval of an
			// outer named declaration. The returned Function field
			// carries a position-based identifier for diagnostics
			// only; the strict approval schema rejects CallerKind =
			// function_literal outright, so this identifier can never
			// be a configurable approval key. The caller object is
			// intentionally nil — a function literal has no
			// declaration object to reuse as its caller identity.
			return functionLiteralCallerIdentity(pkg, typed), nil
		case *ast.FuncDecl:
			function, _ := pkg.TypesInfo.Defs[typed.Name].(*types.Func)
			if function == nil {
				break
			}
			return callerIdentityFromFunction(pkg.PkgPath, function), function
		}
	}
	for index := len(ancestors) - 1; index >= 0; index-- {
		value, ok := ancestors[index].(*ast.ValueSpec)
		if !ok || len(value.Values) == 0 || len(value.Names) == 0 {
			continue
		}
		nameIndex := initializerNameIndex(value, position)
		if nameIndex >= len(value.Names) {
			nameIndex = len(value.Names) - 1
		}
		variable, _ := pkg.TypesInfo.Defs[value.Names[nameIndex]].(*types.Var)
		if variable == nil {
			break
		}
		identity := CallerIdentity{
			PackagePath: pkg.PkgPath,
			Function:    "<var-init:" + value.Names[nameIndex].Name + ">",
			Kind:        CallerKindVariableInitializer,
		}
		return identity, variable
	}
	return CallerIdentity{PackagePath: pkg.PkgPath, Function: "<unknown>", Kind: CallerKindFunctionLiteral}, nil
}

// functionLiteralCallerIdentity returns the observed CallerIdentity for
// an anonymous function literal. The Function field encodes the literal's
// source position as a stable diagnostic identifier. The identifier is
// rejected by the strict approval schema because CallerKind is
// CallerKindFunctionLiteral; it is therefore observable but never
// configurable.
func functionLiteralCallerIdentity(pkg *packages.Package, lit *ast.FuncLit) CallerIdentity {
	position := pkg.Fset.PositionFor(lit.Pos(), true)
	return CallerIdentity{
		PackagePath: pkg.PkgPath,
		Function:    fmt.Sprintf("<func-literal:%d:%d>", position.Line, position.Column),
		Kind:        CallerKindFunctionLiteral,
	}
}

// outerNamedCallerFromAncestors walks the ancestor stack from the
// innermost node outward, returning the first named *ast.FuncDecl that
// would have been the caller had the function literal not been present.
// The returned identity is used for diagnostics; the returned object is
// used only for invariant-cascade isolation (e.g. to poison matching
// approval states when an anonymous edge carries an invalid internal
// reference class). The outer object never participates in ordinary
// approval matching, cardinality accounting, or stale-approval checks.
//
// For an enclosing named package function or method, the returned
// object is the exact resolved *types.Func. For a package variable
// initializer with no enclosing FuncDecl, the returned object is nil
// and the boolean is false.
func outerNamedCallerFromAncestors(
	pkg *packages.Package,
	ancestors []ast.Node,
) (CallerIdentity, types.Object, bool) {
	for index := len(ancestors) - 1; index >= 0; index-- {
		declaration, ok := ancestors[index].(*ast.FuncDecl)
		if !ok {
			continue
		}
		function, _ := pkg.TypesInfo.Defs[declaration.Name].(*types.Func)
		if function == nil {
			break
		}
		return callerIdentityFromFunction(pkg.PkgPath, function), function, true
	}
	return CallerIdentity{}, nil, false
}

func initializerNameIndex(value *ast.ValueSpec, position token.Pos) int {
	for index, expression := range value.Values {
		if position >= expression.Pos() && position <= expression.End() {
			return index
		}
	}
	return 0
}

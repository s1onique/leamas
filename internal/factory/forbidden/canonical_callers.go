// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
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

func (a *canonicalAnalysis) callerForUse(
	pkg *packages.Package,
	ancestors []ast.Node,
	position token.Pos,
) (CallerIdentity, types.Object) {
	for index := len(ancestors) - 1; index >= 0; index-- {
		declaration, ok := ancestors[index].(*ast.FuncDecl)
		if !ok {
			continue
		}
		function, _ := pkg.TypesInfo.Defs[declaration.Name].(*types.Func)
		if function == nil {
			break
		}
		return callerIdentityFromFunction(pkg.PkgPath, function), function
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

func initializerNameIndex(value *ast.ValueSpec, position token.Pos) int {
	for index, expression := range value.Values {
		if position >= expression.Pos() && position <= expression.End() {
			return index
		}
	}
	return 0
}

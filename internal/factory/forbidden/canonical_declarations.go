// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// resolveProtectedDeclarations binds every configured symbol to exactly one
// declaration object from this analysis's coherent package graph.
func (a *canonicalAnalysis) resolveProtectedDeclarations() {
	seenConfig := make(map[ProtectedSymbol]bool)
	for _, symbol := range a.config.protected {
		if seenConfig[symbol] {
			a.symbolFinding("authority_policy_duplicate_symbol", symbol, nil, "duplicate protected symbol configuration")
			continue
		}
		seenConfig[symbol] = true
		if !validAuthorityLayer(symbol.Layer) {
			a.symbolFinding("authority_policy_kind_mismatch", symbol, nil, "unknown authority layer")
			continue
		}

		packages := a.packagesByPath(symbol.PackagePath)
		if len(packages) == 0 {
			a.symbolFinding("authority_policy_symbol_missing", symbol, nil, "configured package is absent from repository graph")
			continue
		}
		if len(packages) > 1 {
			a.symbolFinding("authority_policy_symbol_ambiguous", symbol, nil, "configured package resolves more than once")
			continue
		}
		candidates := declarationCandidates(packages[0], symbol.Name)
		exact := objectsMatchingSymbol(packages[0], candidates, symbol)
		switch len(exact) {
		case 1:
			object := exact[0]
			if previous, exists := a.protectedByObject[object]; exists && previous != symbol {
				a.symbolFinding("authority_policy_duplicate_symbol", symbol, object, "multiple configurations resolve to one declaration object")
				continue
			}
			a.protectedByObject[object] = symbol
			a.objectByProtected[symbol] = object
		case 0:
			a.reportSymbolMismatch(packages[0], symbol, candidates)
		default:
			a.symbolFinding("authority_policy_symbol_ambiguous", symbol, exact[0], "configured symbol resolves to multiple declaration objects")
		}
	}
}

func validAuthorityLayer(layer AuthorityLayer) bool {
	switch layer {
	case AuthorityLayerRaw, AuthorityLayerAdapter, AuthorityLayerGate:
		return true
	default:
		return false
	}
}

func (a *canonicalAnalysis) packagesByPath(path string) []*packages.Package {
	var matches []*packages.Package
	for _, pkg := range a.packages {
		if pkg.PkgPath == path {
			matches = append(matches, pkg)
		}
	}
	return matches
}

func declarationCandidates(pkg *packages.Package, name string) []types.Object {
	seen := make(map[types.Object]bool)
	var candidates []types.Object
	for _, object := range pkg.TypesInfo.Defs {
		if object == nil || object.Name() != name || object.Pkg() == nil || object.Pkg().Path() != pkg.PkgPath {
			continue
		}
		if !seen[object] {
			seen[object] = true
			candidates = append(candidates, object)
		}
	}
	return candidates
}

func objectsMatchingSymbol(pkg *packages.Package, candidates []types.Object, symbol ProtectedSymbol) []types.Object {
	var matches []types.Object
	for _, object := range candidates {
		if objectMatchesSymbol(pkg, object, symbol) {
			matches = append(matches, object)
		}
	}
	return matches
}

func objectMatchesSymbol(pkg *packages.Package, object types.Object, symbol ProtectedSymbol) bool {
	switch symbol.Kind {
	case ProtectedPackageFunction:
		function, ok := object.(*types.Func)
		if !ok {
			return false
		}
		signature, _ := function.Type().(*types.Signature)
		return signature != nil && signature.Recv() == nil && function.Parent() == pkg.Types.Scope()
	case ProtectedMethod:
		function, ok := object.(*types.Func)
		if !ok {
			return false
		}
		signature, _ := function.Type().(*types.Signature)
		return signature != nil && signature.Recv() != nil && recvTypeNameFromSig(signature.Recv()) == symbol.Receiver
	case ProtectedPackageVariable:
		variable, ok := object.(*types.Var)
		return ok && variable.Parent() == pkg.Types.Scope()
	default:
		return false
	}
}

func (a *canonicalAnalysis) reportSymbolMismatch(pkg *packages.Package, symbol ProtectedSymbol, candidates []types.Object) {
	if len(candidates) == 0 {
		a.symbolFinding("authority_policy_symbol_missing", symbol, nil, "configured declaration not found")
		return
	}
	if symbol.Kind == ProtectedMethod {
		var methods []types.Object
		for _, candidate := range candidates {
			function, ok := candidate.(*types.Func)
			if !ok {
				continue
			}
			signature, _ := function.Type().(*types.Signature)
			if signature != nil && signature.Recv() != nil {
				methods = append(methods, candidate)
			}
		}
		if symbol.Receiver == "" && len(methods) > 1 {
			a.symbolFinding("authority_policy_symbol_ambiguous", symbol, methods[0], "method name resolves on multiple receivers")
			return
		}
		if len(methods) > 0 {
			a.symbolFinding("authority_policy_receiver_mismatch", symbol, methods[0], "method receiver does not match configuration")
			return
		}
	}
	if symbol.Kind == ProtectedPackageVariable {
		for _, candidate := range candidates {
			if _, ok := candidate.(*types.Var); ok {
				a.symbolFinding("authority_policy_scope_mismatch", symbol, candidate, "configured variable is not in package scope")
				return
			}
		}
	}
	a.symbolFinding("authority_policy_kind_mismatch", symbol, candidates[0], fmt.Sprintf("declaration kind does not match %s", symbol.Kind))
}

func (a *canonicalAnalysis) symbolFinding(kind string, symbol ProtectedSymbol, object types.Object, detail string) {
	position := token.Position{}
	path := symbol.PackagePath
	if object != nil {
		for _, pkg := range a.packagesByPath(symbol.PackagePath) {
			position = pkg.Fset.PositionFor(object.Pos(), true)
			if position.Filename != "" {
				path = a.relativePath(position.Filename)
			}
		}
	}
	message := fmt.Sprintf("%s: %s (%s)", protectedSymbolString(symbol), detail, kind)
	a.addFinding(path, kind, message, position, CallerIdentity{}, symbol, refDeclaration)
}

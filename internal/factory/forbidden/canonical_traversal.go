// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"

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

// analyzeFile uses PreorderStack to track caller via ancestor stack and to
// structurally classify protected references.
func (p *DupcodeBypassPolicy) analyzeFile(pkg *packages.Package, filename string, file *ast.File) []checks.Finding {
	var findings []checks.Finding
	relPath, _ := filepath.Rel(p.repoRoot, filename)

	// Phase 0: validate configured protected declarations before any use scan.
	findings = append(findings, p.resolveProtectedDeclarations(pkg, file)...)

	// Phase 1: detect dot imports with proper Go-literal decoding.
	for _, imp := range file.Imports {
		if imp.Name != nil && imp.Name.Name == "." {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				findings = append(findings, checks.Finding{
					Path: relPath, Kind: "dupcode_import_path_error",
					Message: fmt.Sprintf("line %d: malformed import literal: %v",
						pkg.Fset.Position(imp.Pos()).Line, err),
					Severity: checks.SeverityError,
				})
				continue
			}
			if isProtectedPackage(path) || isAdapterProtectedPackage(path) {
				findings = append(findings, checks.Finding{
					Path: relPath, Kind: "dupcode_dot_import",
					Message: fmt.Sprintf("line %d: dot import of protected package %s is forbidden",
						pkg.Fset.Position(imp.Pos()).Line, path),
					Severity: checks.SeverityError,
				})
			}
		}
	}

	// Phase 2: classify protected references with structural parent-role detection.
	ast.PreorderStack(file, nil, func(n ast.Node, ancestors []ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)

			class := refDirectCall
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if s, ok := pkg.TypesInfo.Selections[sel]; ok && s.Kind() == types.MethodExpr {
					class = refMethodExpression
				}
			}

			callee, ok := p.resolveProtectedUse(pkg, node.Fun, class)
			if !ok {
				return true
			}
			// Same-package internal calls are not a policy bypass.
			if caller.PackagePath == callee.PackagePath {
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
				Message: fmt.Sprintf("line %d: %s.%s called by %s.%s",
					line, callee.PackagePath, callee.Name,
					caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
			return true

		case *ast.SelectorExpr, *ast.Ident:
			if isCalleeOfCallExpr(node, ancestors) {
				return true
			}
			caller := callerIdentity(pkg.PkgPath, ancestors, pkg.Fset)

			var class referenceClass
			switch node.(type) {
			case *ast.SelectorExpr:
				class = refMethodValue
			case *ast.Ident:
				class = refFunctionValue
			default:
				return true
			}

			callee, ok := p.resolveProtectedUse(pkg, node.(ast.Expr), class)
			if !ok {
				return true
			}
			// Same-package internal references are not a policy bypass.
			if caller.PackagePath == callee.PackagePath {
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
				Message: fmt.Sprintf("line %d: protected function value %s.%s captured by %s.%s",
					line, callee.PackagePath, callee.Name,
					caller.PackagePath, caller.Function),
				Severity: checks.SeverityError,
			})
		}
		return true
	})

	return findings
}

// isCalleeOfCallExpr reports whether the node is the Fun expression of an
// enclosing CallExpr in the ancestors stack. Used to suppress duplicate
// function-value findings for the callee of a direct call already reported.
func isCalleeOfCallExpr(node ast.Node, ancestors []ast.Node) bool {
	for i := len(ancestors) - 1; i >= 0; i-- {
		if ce, ok := ancestors[i].(*ast.CallExpr); ok {
			if ce.Fun == node {
				return true
			}
			return false
		}
	}
	return false
}

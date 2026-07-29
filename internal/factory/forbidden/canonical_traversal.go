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
			// Same-package skip REMOVED. Every legitimate same-package
			// protected use must be approved via an exact edge in
			// ApprovedCallers / AdapterApprovedCallers.
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
			// Mark the entire CallExpr.Fun subtree as belonging to this
			// direct call, so children are not reclassified as function
			// values. Do NOT prune call arguments.
			markDirectCalleeSubtree(pkg.Fset, node)
			return true

		case *ast.SelectorExpr, *ast.Ident:
			if isInsideDirectCalleeSubtree(pkg, node) {
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

// directCalleeRange represents the byte range of a CallExpr.Fun subtree.
type directCalleeRange struct {
	start token.Pos
	end   token.Pos
}

// perFileSetCalleeRanges tracks direct-callee ranges per FileSet.
type perFileSetCalleeRanges struct {
	ranges []directCalleeRange
}

func (d *perFileSetCalleeRanges) contains(p token.Pos) bool {
	for _, r := range d.ranges {
		if p >= r.start && p <= r.end {
			return true
		}
	}
	return false
}

// packageDirectCalleeRanges stores per-package direct-callee ranges,
// keyed by *token.FileSet (a pointer comparable type).
var packageDirectCalleeRanges = map[*token.FileSet]*perFileSetCalleeRanges{}

func markDirectCalleeSubtree(fset *token.FileSet, call *ast.CallExpr) {
	if fset == nil || call == nil || call.Fun == nil {
		return
	}
	d, ok := packageDirectCalleeRanges[fset]
	if !ok {
		d = &perFileSetCalleeRanges{}
		packageDirectCalleeRanges[fset] = d
	}
	d.ranges = append(d.ranges, directCalleeRange{
		start: call.Fun.Pos(),
		end:   call.Fun.End(),
	})
}

func isInsideDirectCalleeSubtree(pkg *packages.Package, n ast.Node) bool {
	if pkg == nil || pkg.Fset == nil || n == nil {
		return false
	}
	d, ok := packageDirectCalleeRanges[pkg.Fset]
	if !ok {
		return false
	}
	return d.contains(n.Pos())
}

// resetDirectCalleeRanges clears the per-file-set ranges. Tests may call this
// between scans to keep state isolated.
func resetDirectCalleeRanges() {
	packageDirectCalleeRanges = map[*token.FileSet]*perFileSetCalleeRanges{}
}

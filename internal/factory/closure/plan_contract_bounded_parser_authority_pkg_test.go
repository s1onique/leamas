package closure

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// plan_contract_bounded_parser_authority_pkg_test.go holds the
// shared AST inspection helpers used by the bounded parser
// authority test suite. Splitting the helpers into a focused
// file keeps the test driver below the LLM-friendly 400-line
// threshold while every helper remains reviewable in one place.

// containsMaxPlanBytesComparison walks the AST of body and
// reports whether any binary comparison has exactly
// MaxPlanBytes on one side and `len(data)` on the other. The
// check accepts the four documented comparison forms:
//
//	len(data) > MaxPlanBytes
//	len(data) >= MaxPlanBytes
//	MaxPlanBytes < len(data)
//	MaxPlanBytes <= len(data)
//
// The detector uses AST predicates (isMaxPlanBytes,
// isLenOfData) instead of substring matching so it cannot be
// fooled by:
//   - len(other) > MaxPlanBytes
//   - lenData > MaxPlanBytes
//   - len(data) > otherMaxPlanBytes
//   - helper(len(data)) > MaxPlanBytes
func containsMaxPlanBytesComparison(body ast.Node) bool {
	if body == nil {
		return false
	}
	cmpOps := map[token.Token]bool{
		token.GTR: true, token.GEQ: true,
		token.LSS: true, token.LEQ: true,
	}
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if !cmpOps[bin.Op] {
			return true
		}
		leftIsAuth := isMaxPlanBytes(bin.X)
		rightIsAuth := isMaxPlanBytes(bin.Y)
		leftIsLen := isLenOfData(bin.X)
		rightIsLen := isLenOfData(bin.Y)
		if (leftIsAuth && rightIsLen) || (rightIsAuth && leftIsLen) {
			found = true
			return false
		}
		return true
	})
	return found
}

// isMaxPlanBytes reports whether expr is the identifier
// MaxPlanBytes. The detector requires the exact authority
// constant; any other identifier is rejected.
func isMaxPlanBytes(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "MaxPlanBytes"
}

// isLenOfData reports whether expr is exactly a call
// `len(data)`: an *ast.CallExpr with Fun = identifier "len"
// and exactly one argument that is the identifier "data".
// Anything else, including wrapper calls like helper(len(data)),
// isLen(other), or lenData, is rejected.
func isLenOfData(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if ident.Name != "len" {
		return false
	}
	if len(call.Args) != 1 {
		return false
	}
	argIdent, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return false
	}
	return argIdent.Name == "data"
}

// loadClosureSources parses every non-test Go file in the
// closure package directory and returns the AST roots. Parser
// errors are surfaced to the caller so the audit never falls
// open on malformed source.
func loadClosureSources(t *testing.T) ([]*ast.File, error) {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			paths = append(paths, name)
		}
	}
	fset := token.NewFileSet()
	var roots []*ast.File
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		roots = append(roots, file)
	}
	return roots, nil
}

// callSite records the function and file of a single call-site
// for a target identifier.
type callSite struct {
	function string
	file     string
}

// functionInfo records a function's name, file, and the set of
// identifier-named callees inside its body.
type functionInfo struct {
	name  string
	file  string
	calls []string
}

// collectCallsInPackage returns every call site of `target` in
// the closure package. The file is parsed as Go source so the
// result is unaffected by reflection or build tags.
func collectCallsInPackage(t *testing.T, target string) []callSite {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", closureFilter, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	var sites []callSite
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if !callExprNamesTarget(call, target) {
					return true
				}
				enclosing := enclosingFunctionName(file, call.Pos(), call.End())
				sites = append(sites, callSite{function: enclosing, file: shortFile(fset, file)})
				return true
			})
		}
	}
	return sites
}

// collectPublicByteEntryFunctions returns every exported function
// whose first parameter is []byte. The call list is the set of
// identifier-named callees inside its body.
func collectPublicByteEntryFunctions(t *testing.T) []functionInfo {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", closureFilter, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}
	var out []functionInfo
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !ast.IsExported(fn.Name.Name) {
					continue
				}
				if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
					continue
				}
				first := fn.Type.Params.List[0]
				if !isByteSliceType(first.Type) {
					continue
				}
				out = append(out, functionInfo{
					name:  fn.Name.Name,
					file:  shortFile(fset, file),
					calls: collectCallTargets(fn),
				})
			}
		}
	}
	return out
}

// closureFilter is the parser filter: include every non-test Go
// file in the package root. The schema sub-package is in a
// separate directory so ParseDir does not traverse it.
func closureFilter(info os.FileInfo) bool {
	name := info.Name()
	if strings.HasSuffix(name, "_test.go") {
		return false
	}
	return strings.HasSuffix(name, ".go")
}

// callExprNamesTarget reports whether a call expression's
// function expression resolves to the bare identifier `target`.
func callExprNamesTarget(call *ast.CallExpr, target string) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == target
}

// enclosingFunctionName finds the innermost function declaration
// enclosing a node positioned between start and end. Returns
// "<top-level>" when no function encloses the node.
func enclosingFunctionName(file *ast.File, start, end token.Pos) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Pos() <= start && end <= fn.End() {
			return fn.Name.Name
		}
	}
	return "<top-level>"
}

// collectCallTargets returns the set of identifier-named
// functions called within the given function declaration body.
func collectCallTargets(fn *ast.FuncDecl) []string {
	var out []string
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		out = append(out, ident.Name)
		return true
	})
	return out
}

// isByteSliceType reports whether an AST expression denotes
// []byte or []uint8 (byte is an alias for uint8 in Go).
func isByteSliceType(expr ast.Expr) bool {
	arr, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}
	return isByteType(arr.Elt)
}

// isByteType reports whether an AST expression is the
// predeclared `byte` or its alias `uint8`.
func isByteType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "byte" || ident.Name == "uint8"
}

// shortFile returns the relative file path recorded in the
// FileSet for the given file node.
func shortFile(fset *token.FileSet, file *ast.File) string {
	tok := fset.File(file.Pos())
	if tok == nil {
		return "<unknown>"
	}
	return tok.Name()
}

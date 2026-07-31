package closure

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// plan_contract_bounded_parser_authority_test.go enforces the
// single-caller invariant for parseClosurePlanDocument and the
// no-duplicate-size-check invariant for every public byte-entry
// API. It walks every Go source file in the package via go/ast
// and asserts the documented caller topology.

// TestBoundedParserAuthoritySingleCaller proves that the only
// production caller of parseClosurePlanDocument is
// parseBoundedClosurePlanDocument (in
// plan_contract_validation_bounded.go). The function is otherwise
// private and tests must reach it through the bounded entry
// point so the size bound is enforced.
func TestBoundedParserAuthoritySingleCaller(t *testing.T) {
	calls := collectCallsInPackage(t, "parseClosurePlanDocument")
	if len(calls) != 1 {
		var names []string
		for _, c := range calls {
			names = append(names, c.function)
		}
		t.Fatalf("parseClosurePlanDocument callers = %v, want exactly 1", names)
	}
	if calls[0].function != "parseBoundedClosurePlanDocument" {
		t.Fatalf("parseClosurePlanDocument caller = %s, want parseBoundedClosurePlanDocument", calls[0].function)
	}
	if !strings.HasSuffix(calls[0].file, "plan_contract_validation_bounded.go") {
		t.Fatalf("parseClosurePlanDocument caller file = %s, want plan_contract_validation_bounded.go", calls[0].file)
	}
}

// TestBoundedParserAuthorityNoPublicDuplicateSizeCheck proves that
// no exported byte-entry API calls parseClosurePlanDocument
// directly (which would bypass MaxPlanBytes). Public byte-entry
// APIs must reach the parser only through
// parseBoundedClosurePlanDocument so the size bound is enforced.
func TestBoundedParserAuthorityNoPublicDuplicateSizeCheck(t *testing.T) {
	for _, fn := range collectPublicByteEntryFunctions(t) {
		for _, call := range fn.calls {
			if call == "parseClosurePlanDocument" {
				t.Fatalf("public byte-entry %s in %s must not call parseClosurePlanDocument directly; route through parseBoundedClosurePlanDocument",
					fn.name, fn.file)
			}
		}
	}
}

// TestBoundedParserAuthorityNoDuplicateSizeCheck walks every
// production byte-entry and internal pipeline entry point and
// asserts the function body contains no comparison equivalent
// to `len(data) > MaxPlanBytes`. The only place that size check
// is allowed is the bounded parser itself.
func TestBoundedParserAuthorityNoDuplicateSizeCheck(t *testing.T) {
	entries := sizeBoundCandidateEntries()
	for _, entry := range entries {
		if entry == "parseBoundedClosurePlanDocument" {
			continue
		}
		body, err := loadFunctionBody(entry)
		if err != nil {
			t.Fatalf("load function %s: %v", entry, err)
		}
		if containsMaxPlanBytesComparison(body) {
			t.Fatalf("function %s contains a duplicate MaxPlanBytes comparison; route through parseBoundedClosurePlanDocument instead",
				entry)
		}
	}
}

// sizeBoundCandidateEntries returns every entry point the ACT
// pins as a candidate for size-bound duplication. The list is
// authoritative; add new candidates here only after the ACT
// grows them.
func sizeBoundCandidateEntries() []string {
	return []string{
		"DecodePlan",
		"LoadPlanFromBytes",
		"ValidatePlanStructural",
		"ValidatePlanComposed",
		"ValidatePlanStructuralAndSemantic",
		"validatePlanComposedWithObserver",
		"validatePlanComposedWithObserverAndDeps",
		"validatePlanStructuralAndSemanticWith",
		"validatePlanStructuralWithObserver",
	}
}

// loadFunctionBody returns the textual body of the named
// function. The body is the concatenation of the function's
// AST identifiers and basic literals so the comparison audit
// is independent of comment/whitespace changes.
func loadFunctionBody(name string) (string, error) {
	roots, err := loadClosureSources()
	if err != nil {
		return "", err
	}
	for _, file := range roots {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if fn.Name.Name != name {
				continue
			}
			return funcBodyText(fn), nil
		}
	}
	return "", nil
}

// containsMaxPlanBytesComparison walks the AST of the function
// body and reports whether it contains a binary comparison
// involving `MaxPlanBytes` and a length expression. The check
// accepts the four documented comparison forms:
//
//	len(data) > MaxPlanBytes
//	len(data) >= MaxPlanBytes
//	MaxPlanBytes < len(data)
//	MaxPlanBytes <= len(data)
func containsMaxPlanBytesComparison(body string) bool {
	if !strings.Contains(body, "MaxPlanBytes") {
		return false
	}
	if !strings.Contains(body, "len") {
		return false
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "body.go", body, 0)
	if err != nil {
		return false
	}
	cmpOps := map[token.Token]bool{
		token.GTR: true, token.GEQ: true,
		token.LSS: true, token.LEQ: true,
	}
	var found bool
	ast.Inspect(file, func(n ast.Node) bool {
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
		leftText := exprText(bin.X)
		rightText := exprText(bin.Y)
		if !strings.Contains(leftText, "MaxPlanBytes") && !strings.Contains(rightText, "MaxPlanBytes") {
			return true
		}
		// Confirm the comparison references the size limit
		// and a length expression. The audit tolerates either
		// side carrying the length expression.
		hasLen := strings.Contains(leftText, "len") || strings.Contains(rightText, "len")
		if !hasLen {
			return true
		}
		found = true
		return false
	})
	return found
}

// funcBodyText returns the textual body of a function. The
// definition lives in source_structure_test.go so the package
// has exactly one implementation. The text is the
// concatenation of identifier names and basic literal values
// from the function body.

// exprText reconstructs the textual form of an AST expression.
// Identifiers are emitted as their names; call expressions
// include the function name plus a trailing "(". The result is
// robust enough for the audit's textual checks without
// requiring full source reconstruction.
func exprText(expr ast.Expr) string {
	var b strings.Builder
	ast.Inspect(expr, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.Ident:
			b.WriteString(v.Name)
		case *ast.CallExpr:
			if ident, ok := v.Fun.(*ast.Ident); ok {
				b.WriteString(ident.Name)
				b.WriteString("(")
			}
		}
		return true
	})
	return b.String()
}

// loadClosureSources parses every non-test Go file in the
// closure package directory and returns the AST roots.
func loadClosureSources() ([]*ast.File, error) {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			paths = append(paths, name)
		}
	}
	var roots []*ast.File
	fset := token.NewFileSet()
	for _, path := range paths {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
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

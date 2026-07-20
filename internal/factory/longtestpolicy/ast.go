// Package longtestpolicy provides a verifier that checks long-test policy compliance.
package longtestpolicy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

const longtestCanonicalPath = "github.com/s1onique/leamas/internal/factory/longtest"

type CallSite struct {
	ID          string
	File        string
	PkgPath     string
	TestFunc    string
	Line        int
	ValidTest   bool
}

type ScanResult struct {
	LiteralCalls []CallSite
	Malformed   []CallSite
}

func scanTestFileAST(path, pkgPath string) (*ScanResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	result := &ScanResult{LiteralCalls: []CallSite{}, Malformed: []CallSite{}}

	longtestIdent, testingIdent := findImportIdents(file)
	if longtestIdent == "" {
		return result, nil
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		funcName := getFuncName(fn)
		validTest := isValidGoTest(fn, testingIdent)
		scanFuncBody(fset, fn.Body, path, pkgPath, funcName, validTest, longtestIdent, result)
	}

	return result, nil
}

func findImportIdents(file *ast.File) (longtestIdent, testingIdent string) {
	for _, imp := range file.Imports {
		impPath := trimQuote(imp.Path.Value)
		if impPath == longtestCanonicalPath {
			longtestIdent = getImportName(imp)
		}
		if impPath == "testing" {
			testingIdent = getImportName(imp)
		}
	}
	return
}

func scanFuncBody(fset *token.FileSet, body *ast.BlockStmt, path, pkgPath, funcName string, validTest bool, longtestIdent string, result *ScanResult) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "RequireLongTest" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != longtestIdent {
			return true
		}
		pos := fset.Position(call.Pos())
		if len(call.Args) < 2 {
			result.Malformed = append(result.Malformed, CallSite{
				ID: "", File: path, PkgPath: pkgPath, TestFunc: funcName, Line: pos.Line, ValidTest: validTest,
			})
			return true
		}
		lit, ok := call.Args[1].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			result.Malformed = append(result.Malformed, CallSite{
				ID: "", File: path, PkgPath: pkgPath, TestFunc: funcName, Line: pos.Line, ValidTest: validTest,
			})
			return true
		}
		id, err := strconv.Unquote(lit.Value)
		if err != nil {
			result.Malformed = append(result.Malformed, CallSite{
				ID: "", File: path, PkgPath: pkgPath, TestFunc: funcName, Line: pos.Line, ValidTest: validTest,
			})
			return true
		}
		result.LiteralCalls = append(result.LiteralCalls, CallSite{
			ID: id, File: path, PkgPath: pkgPath, TestFunc: funcName, Line: pos.Line, ValidTest: validTest,
		})
		return true
	})
}

func getFuncName(fn *ast.FuncDecl) string {
	if fn.Name == nil {
		return ""
	}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return "method(" + fn.Name.Name + ")"
	}
	return fn.Name.Name
}

// parameterCount returns the number of logical parameters in a field list.
// Each Field may declare multiple comma-separated names, e.g., "a, b int".
func parameterCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	count := 0
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func isValidGoTest(fn *ast.FuncDecl, testingIdent string) bool {
	if fn == nil || fn.Name == nil {
		return false
	}
	name := fn.Name.Name
	if len(name) <= 4 || name[:4] != "Test" {
		return false
	}
	if name[4] >= 'a' && name[4] <= 'z' {
		return false
	}
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return false
	}
	if parameterCount(fn.Type.Params) != 1 {
		return false
	}
	if testingIdent == "" {
		testingIdent = "testing"
	}
	param := fn.Type.Params.List[0]
	if !isTestingTType(param.Type, testingIdent) {
		return false
	}
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		return false
	}
	return true
}

func isTestingTType(expr ast.Expr, testingIdent string) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != testingIdent {
		return false
	}
	return sel.Sel.Name == "T"
}

func getImportName(imp *ast.ImportSpec) string {
	if imp.Name != nil {
		return imp.Name.String()
	}
	path := trimQuote(imp.Path.Value)
	parts := splitPath(path)
	return parts[len(parts)-1]
}

func trimQuote(s string) string { return s[1 : len(s)-1] }

func splitPath(p string) []string {
	var parts []string
	start := 0
	for i, c := range p {
		if c == '/' {
			parts = append(parts, p[start:i])
			start = i + 1
		}
	}
	return append(parts, p[start:])
}

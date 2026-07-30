package closure

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// hygieneFiles are the production source files this test parses.
// Adding the dead helper back into any of these files fails the
// test. The test is an AST-level guard, not a grep, so it cannot
// be defeated by renaming or by introducing a comment that
// references the dead identifier.
var hygieneFiles = []string{
	"plan_contract_validation.go",
	"plan_contract_validation_bounded.go",
	"plan_contract_validation_composed.go",
	"plan_contract_validation_fields.go",
}

// TestFirstSemanticErrorHelperRemoved proves the dead helper
// firstSemanticError is not declared in any production source
// file. The test parses every file in hygieneFiles with
// go/parser and walks the AST; it fails on any FuncDecl whose
// name is firstSemanticError.
func TestFirstSemanticErrorHelperRemoved(t *testing.T) {
	fset := token.NewFileSet()
	for _, name := range hygieneFiles {
		astFile, err := parser.ParseFile(fset, name, nil,
			parser.ParseComments)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range astFile.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "firstSemanticError" {
				t.Fatalf("dead helper firstSemanticError must not be declared in %s",
					name)
			}
		}
	}
}

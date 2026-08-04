package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthorityOnePlanRouter(t *testing.T) {
	// Prove: one plan router exists in factory_close_plan.go
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var routers int
	var runFactoryClosePlanFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "runFactoryClosePlan" {
				runFactoryClosePlanFunc = fn
				routers++
			}
		}
	}

	if routers != 1 {
		t.Errorf("routers = %d, want 1", routers)
	}

	if runFactoryClosePlanFunc == nil {
		t.Fatal("runFactoryClosePlan not found")
	}

	// Verify it has a switch statement routing to schema/example/validate
	body := runFactoryClosePlanFunc.Body
	var hasSwitch bool
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.SwitchStmt); ok {
			hasSwitch = true
		}
		return true
	})
	if !hasSwitch {
		t.Error("runFactoryClosePlan missing switch statement")
	}
}

func TestAuthorityOneSchemaImplementation(t *testing.T) {
	// Prove: exactly one schema implementation exists
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_schema.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var schemaFuncs int
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "runFactoryClosePlanSchema") {
				schemaFuncs++
			}
		}
	}

	if schemaFuncs != 1 {
		t.Errorf("schemaFuncs = %d, want 1", schemaFuncs)
	}
}

func TestAuthorityOneExampleImplementation(t *testing.T) {
	// Prove: exactly two example implementations exist:
	// - runFactoryClosePlanExample (production adapter)
	// - runFactoryClosePlanExampleWith (testable handler)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_example.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var exampleFuncs int
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "runFactoryClosePlanExample") {
				exampleFuncs++
			}
		}
	}

	if exampleFuncs != 2 {
		t.Errorf("exampleFuncs = %d, want 2 (adapter + handler)", exampleFuncs)
	}
}

func TestAuthorityOneValidateImplementation(t *testing.T) {
	// Prove: exactly two validate implementations exist:
	// - runFactoryClosePlanValidate (production adapter)
	// - runFactoryClosePlanValidateWith (testable handler)
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_validate.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var validateFuncs int
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if strings.HasPrefix(fn.Name.Name, "runFactoryClosePlanValidate") {
				validateFuncs++
			}
		}
	}

	if validateFuncs != 2 {
		t.Errorf("validateFuncs = %d, want 2 (adapter + handler)", validateFuncs)
	}
}

func TestAuthoritySchemaGeneratorCalledOnce(t *testing.T) {
	// Prove: closure.JSONSchema is called exactly once in the schema handler
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_schema.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var jsonSchemaCalls int
	ast.Inspect(node, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "JSONSchema" {
					jsonSchemaCalls++
				}
			}
		}
		return true
	})

	if jsonSchemaCalls != 1 {
		t.Errorf("JSONSchema calls = %d, want 1", jsonSchemaCalls)
	}
}

func TestAuthorityExampleValidateComposedCalledOnce(t *testing.T) {
	// Prove: ValidatePlanComposed is called exactly once in the example handler body
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_example.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	// Find the runFactoryClosePlanExampleWith function
	var handlerFunc *ast.FuncDecl
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "runFactoryClosePlanExampleWith" {
				handlerFunc = fn
				break
			}
		}
	}

	if handlerFunc == nil {
		t.Fatal("runFactoryClosePlanExampleWith not found")
	}

	var validateCalls int
	ast.Inspect(handlerFunc.Body, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Validate" {
					validateCalls++
				}
			}
		}
		return true
	})

	if validateCalls != 1 {
		t.Errorf("Validate calls in runFactoryClosePlanExampleWith = %d, want 1", validateCalls)
	}
}

func TestAuthorityValidateComposedCalledOnce(t *testing.T) {
	// Prove: closure.ValidatePlanComposed is called exactly once in validate handler
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "factory_close_plan_validate.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var validateCalls int
	ast.Inspect(node, func(n ast.Node) bool {
		if ce, ok := n.(*ast.CallExpr); ok {
			if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "ValidatePlanComposed" {
					validateCalls++
				}
			}
		}
		return true
	})

	if validateCalls != 1 {
		t.Errorf("ValidatePlanComposed calls = %d, want 1", validateCalls)
	}
}

func TestAuthorityZeroOsExitInHandlers(t *testing.T) {
	// Prove: zero os.Exit calls in handler files
	for _, file := range []string{
		"factory_close_plan.go",
		"factory_close_plan_schema.go",
		"factory_close_plan_example.go",
		"factory_close_plan_validate.go",
	} {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		var osExitCalls int
		ast.Inspect(node, func(n ast.Node) bool {
			if ce, ok := n.(*ast.CallExpr); ok {
				if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
					if ident, ok := sel.X.(*ast.Ident); ok {
						if ident.Name == "os" && sel.Sel.Name == "Exit" {
							osExitCalls++
						}
					}
				}
			}
			return true
		})

		if osExitCalls > 0 {
			t.Errorf("%s has %d os.Exit calls, want 0", filepath.Base(file), osExitCalls)
		}
	}
}

func TestAuthorityOsStdinScoped(t *testing.T) {
	// Prove: os.Stdin is only used in the production adapter, not handlers
	// runFactoryClosePlanValidate should have exactly one os.Stdin
	// runFactoryClosePlanValidateWith should have zero
	// schema/example handlers should have zero

	// Check production adapter has exactly one os.Stdin
	fset := token.NewFileSet()
	validateFile, err := parser.ParseFile(fset, "factory_close_plan_validate.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	var adapterStdin, handlerStdin int
	for _, decl := range validateFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			countStdin := func(node *ast.FuncDecl) int {
				var count int
				ast.Inspect(node.Body, func(n ast.Node) bool {
					if sel, ok := n.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok {
							if ident.Name == "os" && sel.Sel.Name == "Stdin" {
								count++
							}
						}
					}
					return true
				})
				return count
			}

			switch fn.Name.Name {
			case "runFactoryClosePlanValidate":
				adapterStdin = countStdin(fn)
			case "runFactoryClosePlanValidateWith":
				handlerStdin = countStdin(fn)
			}
		}
	}

	if adapterStdin != 1 {
		t.Errorf("runFactoryClosePlanValidate has %d os.Stdin, want 1", adapterStdin)
	}
	if handlerStdin != 0 {
		t.Errorf("runFactoryClosePlanValidateWith has %d os.Stdin, want 0", handlerStdin)
	}

	// Check schema and example handlers have zero os.Stdin
	for _, file := range []string{"factory_close_plan_schema.go", "factory_close_plan_example.go"} {
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}

		var stdinCount int
		ast.Inspect(node, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "os" && sel.Sel.Name == "Stdin" {
						stdinCount++
					}
				}
			}
			return true
		})

		if stdinCount > 0 {
			t.Errorf("%s has %d os.Stdin, want 0", filepath.Base(file), stdinCount)
		}
	}
}

func TestAuthorityExactDeclarationNames(t *testing.T) {
	// Prove: exact declaration names are used
	fset := token.NewFileSet()

	// Schema: must have runFactoryClosePlanSchema (production)
	schemaFile, err := parser.ParseFile(fset, "factory_close_plan_schema.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var hasSchemaAdapter bool
	for _, decl := range schemaFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "runFactoryClosePlanSchema" && fn.Body != nil {
				hasSchemaAdapter = true
			}
		}
	}
	if !hasSchemaAdapter {
		t.Error("missing runFactoryClosePlanSchema with non-nil body")
	}

	// Example: must have runFactoryClosePlanExample (production) and runFactoryClosePlanExampleWith (handler)
	exampleFile, err := parser.ParseFile(fset, "factory_close_plan_example.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var hasExampleAdapter, hasExampleHandler bool
	for _, decl := range exampleFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "runFactoryClosePlanExample" && fn.Body != nil {
				hasExampleAdapter = true
			}
			if fn.Name.Name == "runFactoryClosePlanExampleWith" && fn.Body != nil {
				hasExampleHandler = true
			}
		}
	}
	if !hasExampleAdapter {
		t.Error("missing runFactoryClosePlanExample with non-nil body")
	}
	if !hasExampleHandler {
		t.Error("missing runFactoryClosePlanExampleWith with non-nil body")
	}

	// Validate: must have runFactoryClosePlanValidate (production) and runFactoryClosePlanValidateWith (handler)
	validateFile, err := parser.ParseFile(fset, "factory_close_plan_validate.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var hasValidateAdapter, hasValidateHandler bool
	for _, decl := range validateFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name.Name == "runFactoryClosePlanValidate" && fn.Body != nil {
				hasValidateAdapter = true
			}
			if fn.Name.Name == "runFactoryClosePlanValidateWith" && fn.Body != nil {
				hasValidateHandler = true
			}
		}
	}
	if !hasValidateAdapter {
		t.Error("missing runFactoryClosePlanValidate with non-nil body")
	}
	if !hasValidateHandler {
		t.Error("missing runFactoryClosePlanValidateWith with non-nil body")
	}
}

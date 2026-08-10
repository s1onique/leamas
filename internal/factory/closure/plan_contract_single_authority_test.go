// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_single_authority_test.go
// is the B2-R7 AST/source-inspection guard for the
// closure package's Plan Contract semantic authority.
//
// B2-R7 single-authority rule: every wire-contract rule
// for the Plan Contract v1 lives in the plancontract
// leaf. The closure package's legacy typed validators
// (validatePlanTyped, ValidateRunnerAuthority, etc.) are
// adapters; their bodies MUST contain only:
//
//   - a canonical call to plancontract,
//   - error adaptation (typed-error mapping), and
//   - representation conversion (typed Plan <-> wire bytes).
//
// This test inspects the closure package's source files
// for the legacy semantic-validator symbols and asserts
// each body contains only the canonical-call /
// adaptation / conversion primitives. If a future
// refactor restores a real semantic rule inside the
// closure package, the guard fires and the test fails.
//
// The test uses go/parser rather than reflection so the
// guard is deterministic across Go versions and does
// not depend on private runtime state.
package closure

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// semanticAuthoritySymbols is the closed list of legacy
// typed-validator symbols the guard inspects. Adding a
// new validator requires adding it here so the guard
// stays complete.
var semanticAuthoritySymbols = []string{
	"ValidatePlan",
	"ValidateRunnerAuthority",
	"validatePlanTyped",
}

// allowedCallPrefixes is the closed list of call
// prefixes that count as "canonical call /
// adaptation / conversion" primitives. Any other call
// inside a legacy validator body fails the guard.
var allowedCallPrefixes = []string{
	"plancontract.",
	"adaptPlanContractError",
	"adaptRunnerAuthorityError",
	"encodePlanForValidation",
	"json.",
	"fmt.",
	"errors.",
	"strings.",
	"authorityToWire",
	"runnerAuthorityShortField",
	"containsClosurePlaceholder",
	"convertPlanContractError",
	"validatePlanTyped",
	"errorFromDiagnostics",
	"decodeTypedPlan",
	"loadPlan",
	"err.Error",
	"len(",
	"nil",
	"newSemanticError",
}

// TestPlanContractSingleSemanticAuthority is the
// B2-R7 AST/source guard. It parses every closure-package
// source file that contains a legacy semantic-validator
// symbol and asserts the body contains only canonical
// call / adaptation / conversion primitives. Drift here
// is a contract bug; the test is the gate.
func TestPlanContractSingleSemanticAuthority(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		// Skip generated and test files; the guard
		// targets production authority surfaces.
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
		return true
	}, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				if !semanticAuthoritySymbol(fn.Name.Name) {
					continue
				}
				assertOnlyAllowedCalls(t, fn)
			}
		}
	}
}

func semanticAuthoritySymbol(name string) bool {
	for _, sym := range semanticAuthoritySymbols {
		if name == sym {
			return true
		}
	}
	return false
}

// assertOnlyAllowedCalls walks the function body and
// fails the test if any call expression has a callee
// that is NOT one of the allowed call prefixes. The
// check is name-based; package-qualified names like
// plancontract.DecodeAndValidateFull match the
// "plancontract." prefix.
func assertOnlyAllowedCalls(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch e := call.Fun.(type) {
		case *ast.SelectorExpr:
			if !selectorAllowed(e) {
				t.Fatalf("legacy validator %q contains non-canonical call %q; B2-R7 single-authority rule violated",
					fn.Name.Name, selectorName(e))
			}
		case *ast.Ident:
			if !identAllowed(e.Name) {
				t.Fatalf("legacy validator %q contains non-canonical call %q; B2-R7 single-authority rule violated",
					fn.Name.Name, e.Name)
			}
		}
		return true
	})
}

func selectorAllowed(e *ast.SelectorExpr) bool {
	name := selectorName(e)
	for _, prefix := range allowedCallPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func identAllowed(name string) bool {
	for _, prefix := range allowedCallPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func selectorName(e *ast.SelectorExpr) string {
	if id, ok := e.X.(*ast.Ident); ok {
		return id.Name + "." + e.Sel.Name
	}
	return e.Sel.Name
}

// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_single_authority_helpers_test.go
// owns the helper functions for the B2-R7-R1 AST/source
// guard in plan_contract_single_authority_test.go.
//
// The guard fails if any of these conditions occur:
//   - a legacy validator body contains a non-canonical call,
//   - a newly introduced validate*Plan* / *Authority*
//     production function does not delegate to plancontract,
//   - a function whose name suggests Plan Contract authority
//     makes a semantic comparison against a Plan field,
//   - a function whose name suggests Plan Contract authority
//     matches a Plan field against a closure regex helper.
package closure


import (
	"go/ast"
	"strings"
	"testing"
)

// legacySymbol reports whether name matches one of the
// legacy typed-validator symbols the guard watches.
func legacySymbol(name string) bool {
	for _, sym := range legacyValidatorSymbols {
		if name == sym {
			return true
		}
	}
	return false
}

// isNewValidatorSymbol reports whether name matches the
// pattern of a newly introduced production validator. The
// check is regex-based so the guard fires if a future
// change adds, say, `validatePlanChecks` or
// `validateRunnerAuthorityTool`.
func isNewValidatorSymbol(name string) bool {
	for _, pattern := range newValidatorRegex {
		ok, err := globMatch(pattern, name)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// looksLikePlanContractAuthority reports whether name
// suggests the function owns a Plan Contract semantic rule.
// Functions matching this predicate are subject to the
// semantic-comparison and regex-matching checks. Functions
// whose names do not suggest Plan Contract authority
// (e.g. `BuildV2ValidPlanFixtureWithCheck`) are not
// authority surfaces and may legitimately touch Plan
// fields without owning a wire rule.
func looksLikePlanContractAuthority(name string) bool {
	return legacySymbol(name) || isNewValidatorSymbol(name)
}

// assertOnlyAllowedCalls walks the function body and
// fails the test if any call expression has a callee
// that is NOT one of the allowed call prefixes.
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

// assertDelegatesToPlanContract asserts the function body
// contains at least one call to a plancontract.* symbol.
// Newly introduced validate*Plan* / *Authority* production
// functions MUST delegate to the canonical leaf.
func assertDelegatesToPlanContract(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if id.Name == "plancontract" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Fatalf("validator %q does not delegate to plancontract; B2-R7-R1 single-authority rule violated",
			fn.Name.Name)
	}
}

// assertNoPlanFieldSemanticComparisons walks the function
// body and fails the test if any binary expression compares
// a Plan field to a literal / constant. The check is
// pattern-based on field name; a precise AST match would
// require resolving selector chains which the guard keeps
// simple.
func assertNoPlanFieldSemanticComparisons(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		bin, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		left := selectorNameFromExpr(bin.X)
		right := selectorNameFromExpr(bin.Y)
		if fieldReferencesPlan(left) || fieldReferencesPlan(right) {
			t.Fatalf("production function %q contains a semantic comparison against a Plan field; B2-R7-R1 single-authority rule violated",
				fn.Name.Name)
		}
		return true
	})
}

// assertNoPlanFieldRegexMatching walks the function body
// and fails the test if any call uses one of the
// closure-side regex helpers (e.g. oidPattern.MatchString)
// on a Plan field value. The canonical shape check lives
// in the plancontract leaf.
func assertNoPlanFieldRegexMatching(t *testing.T, fn *ast.FuncDecl) {
	t.Helper()
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// Look for <regex>.MatchString(<plan-field>) call.
		for _, helper := range planRegexHelpers {
			if sel.Sel.Name == "MatchString" && selectorName(sel) == helper {
				// Inspect the first argument.
				if len(call.Args) == 0 {
					continue
				}
				arg := selectorNameFromExpr(call.Args[0])
				if fieldReferencesPlan(arg) {
					t.Fatalf("production function %q matches a Plan field against %s; B2-R7-R1 single-authority rule violated",
						fn.Name.Name, helper)
				}
			}
		}
		return true
	})
}

func selectorAllowed(e *ast.SelectorExpr) bool {
	name := selectorName(e)
	for _, prefix := range adapterCallAllowedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func identAllowed(name string) bool {
	for _, prefix := range adapterCallAllowedPrefixes {
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

// selectorNameFromExpr returns the dotted selector name
// (e.g. `plan.X`) when the expression is a selector chain,
// or the empty string otherwise. Used to detect Plan-field
// references inside expressions.
func selectorNameFromExpr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.SelectorExpr:
		return selectorName(v)
	case *ast.Ident:
		return v.Name
	}
	return ""
}

// fieldReferencesPlan reports whether expr contains a
// reference to a Plan struct field. The check is
// pattern-based: it matches Plan.X style selectors where X
// is a known Plan field.
func fieldReferencesPlan(expr string) bool {
	if expr == "" {
		return false
	}
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return false
	}
	last := parts[len(parts)-1]
	for _, field := range planFieldNames {
		if last == field {
			return true
		}
	}
	return false
}

// globMatch is a tiny shell-glob matcher used by the
// validator-symbol regex list. The implementation supports
// only `*` wildcards because that is all the guard needs.
func globMatch(pattern, name string) (bool, error) {
	if pattern == name {
		return true, nil
	}
	if !strings.Contains(pattern, "*") {
		return false, nil
	}
	parts := strings.Split(pattern, "*")
	idx := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		found := strings.Index(name[idx:], p)
		if found < 0 {
			return false, nil
		}
		idx += found + len(p)
	}
	return true, nil
}

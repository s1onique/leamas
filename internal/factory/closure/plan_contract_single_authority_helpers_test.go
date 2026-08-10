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

// selectorNameFromExpr returns the dotted selector chain
// (e.g. `plan.Baseline.CommitOID`) by walking any depth
// of nested SelectorExpr nodes. Returns the empty string
// for non-selector expressions. The walker follows every
// selector chain depth so the guard can detect nested-field
// references like `plan.Baseline.CommitOID == ...` that
// a single-level selector walk would miss.
//
// HEURISTIC qualifier: a selector chain whose root is not
// an Ident (e.g. `f().X`) is not resolvable by static AST
// inspection; the guard accepts that gap and matches
// only selector chains that terminate at a named root.
func selectorNameFromExpr(e ast.Expr) string {
	var parts []string
	cur := e
	for {
		sel, ok := cur.(*ast.SelectorExpr)
		if !ok {
			break
		}
		parts = append([]string{sel.Sel.Name}, parts...)
		cur = sel.X
	}
	if len(parts) == 0 {
		if id, ok := e.(*ast.Ident); ok {
			return id.Name
		}
		return ""
	}
	if id, ok := cur.(*ast.Ident); ok {
		parts = append([]string{id.Name}, parts...)
	}
	out := ""
	for i, p := range parts {
		if i == 0 {
			out = p
		} else {
			out += "." + p
		}
	}
	return out
}

// fieldReferencesPlan reports whether expr contains a
// reference to any known Plan struct field. The check
// walks every part of the dotted chain so nested fields
// like `plan.Baseline.CommitOID` are detected via the
// `Baseline` part even if `CommitOID` is not in the
// watch list. The watch list deliberately covers the
// top-level Plan fields; a future expansion can add
// sub-field names without changing the guard.
func fieldReferencesPlan(expr string) bool {
	if expr == "" {
		return false
	}
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		for _, field := range planFieldNames {
			if p == field {
				return true
			}
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

// TestSelectorNameFromExprFollowsNestedSelectors is the
// B2-R7-R2 unit test for the nested-selector walker. The
// walker must surface the full chain for any depth of
// SelectorExpr nodes so the guard can detect
// plan.Baseline.CommitOID-shaped references.
func TestSelectorNameFromExprFollowsNestedSelectors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"plan", "plan"},
		{"plan.X", "plan.X"},
		{"plan.Baseline.CommitOID", "plan.Baseline.CommitOID"},
		{"plan.Execution.Mode", "plan.Execution.Mode"},
		{"x.y.z.w", "x.y.z.w"},
	}
	for _, c := range cases {
		got := resolveSelectorChainForTest(c.in)
		if got != c.want {
			t.Fatalf("resolveSelectorChainForTest(%q) = %q, want %q",
				c.in, got, c.want)
		}
	}
}

// resolveSelectorChainForTest is a thin wrapper that
// reconstructs the AST node from a dotted name and runs
// the production walker. Used only by the unit test so
// the walker has a deterministic round-trip check.
func resolveSelectorChainForTest(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return ""
	}
	var expr ast.Expr = ast.NewIdent(parts[0])
	for _, p := range parts[1:] {
		expr = &ast.SelectorExpr{X: expr, Sel: ast.NewIdent(p)}
	}
	return selectorNameFromExpr(expr)
}

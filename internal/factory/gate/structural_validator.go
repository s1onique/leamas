// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"go/ast"
	"go/token"
)

// typedDispatchRule describes the dispatch contract for an internal
// *With entry point: there must be exactly one dispatcher.Dispatch
// call, no post-dispatch protected call, and only the direct
// `binder.BindRunner()` may appear inside the dispatch expression's
// argument list.
type typedDispatchRule struct {
	// FunctionName is the package-internal entry-point function name.
	FunctionName string

	// AllowedArgCalls is the exact set of CallExpr expressions permitted
	// to appear INSIDE the dispatcher.Dispatch argument list. Each is
	// matched by its last selector-segment name (or bare identifier for
	// non-selector calls). Anything else inside the argument list is
	// rejected.
	//
	// For example, allowing {"BindRunner"} permits
	// `dispatcher.Dispatch(..., binder.BindRunner())` but rejects
	// `dispatcher.Dispatch(..., helper(binder.BindRunner()))` and
	// `dispatcher.Dispatch(..., func() { NewDupcodeRunner() }())`.
	AllowedArgCalls []string
}

// typedDelegationRule describes the public wrapper contract: it must
// delegate exactly once to the matching internal *With function and
// must contain no direct call to a protected operation.
type typedDelegationRule struct {
	// PublicName is the public wrapper function name.
	PublicName string
	// InternalName is the matching internal *With name.
	InternalName string
}

// StructuralValidatorResult aggregates every error produced by a single
// structural-validation pass.
type StructuralValidatorResult struct {
	Errors []error
}

// OK reports whether the structural validation passed.
func (r StructuralValidatorResult) OK() bool { return len(r.Errors) == 0 }

// typedDispatchContract captures the rules for every internal entry
// point and every public wrapper in the gate package.
type typedDispatchContract struct {
	InternalRules []typedDispatchRule
	PublicRules   []typedDelegationRule
}

// DefaultTypedDispatchContract returns the canonical contract for the
// gate package. It is the single source of truth that both the
// production-source test and the adversarial fixtures exercise.
func DefaultTypedDispatchContract() typedDispatchContract {
	return typedDispatchContract{
		InternalRules: []typedDispatchRule{
			{FunctionName: "dispatchDupcodeVerifyTypedWith", AllowedArgCalls: []string{"BindRunner"}},
			{FunctionName: "dispatchDupcodeBaselineVerifyTypedWith", AllowedArgCalls: []string{"BindRunner"}},
			{FunctionName: "dispatchDupcodeUpdateBaselineTypedWith", AllowedArgCalls: []string{"BindRunner"}},
		},
		PublicRules: []typedDelegationRule{
			{PublicName: "DispatchDupcodeVerifyTyped", InternalName: "dispatchDupcodeVerifyTypedWith"},
			{PublicName: "DispatchDupcodeBaselineVerifyTyped", InternalName: "dispatchDupcodeBaselineVerifyTypedWith"},
			{PublicName: "DispatchDupcodeUpdateBaselineTyped", InternalName: "dispatchDupcodeUpdateBaselineTypedWith"},
		},
	}
}

// protectedCallNamesForStructural is the canonical list of symbol names
// that must NOT appear in the post-Dispatch region of any typed entry
// point or anywhere inside a public wrapper. They are matched against
// the LAST selector segment (or bare identifier) of a CallExpr.
var protectedCallNamesForStructural = []string{
	"run",
	"BindRunner",
	"NewRunner",
	"NewDupcodeRunner",
	"LoadBaseline",
	"RunCheckReport",
	"VerifyBaseline",
	"WriteBaseline",
	"CompareToBaseline",
}

// validateTypedDispatchSource runs one structural-validation pass over
// the parsed production files using the supplied contract. It returns
// every violation found; an empty error slice means the contract holds.
//
// The validator is fail-closed. It rejects:
//
//	missing internal declaration
//	duplicate internal declaration
//	nil internal body
//	zero dispatcher.Dispatch calls
//	two dispatcher.Dispatch calls
//	protected call after dispatch
//	protected call later in the same statement
//	disallowed call inside the dispatch argument list
//	missing public wrapper
//	duplicate public wrapper
//	zero public delegations
//	two public delegations
//	direct protected call from public wrapper
func validateTypedDispatchSource(
	files map[string]*ast.File,
	fset *token.FileSet,
	contract typedDispatchContract,
) StructuralValidatorResult {
	var result StructuralValidatorResult

	// Index declarations by name.
	hits := map[string][]astDecl{}
	for fname, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			hits[fn.Name.Name] = append(hits[fn.Name.Name], astDecl{
				file: fname,
				fset: fset,
				fn:   fn,
			})
		}
	}

	checkInternal := func(name string, allowedArgCalls []string) astDecl {
		found := hits[name]
		if len(found) == 0 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q is missing", name))
			return astDecl{}
		}
		if len(found) > 1 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q appears %d times, want exactly 1", name, len(found)))
			return astDecl{}
		}
		if found[0].fn.Body == nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q has nil body", name))
			return astDecl{}
		}
		validateInternalEntryPoint(found[0], allowedArgCalls, &result)
		return found[0]
	}

	for _, rule := range contract.InternalRules {
		checkInternal(rule.FunctionName, rule.AllowedArgCalls)
	}

	for _, rule := range contract.PublicRules {
		found := hits[rule.PublicName]
		if len(found) == 0 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q is missing", rule.PublicName))
			continue
		}
		if len(found) > 1 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q appears %d times, want exactly 1", rule.PublicName, len(found)))
			continue
		}
		if found[0].fn.Body == nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q has nil body", rule.PublicName))
			continue
		}
		validatePublicWrapper(found[0], rule.InternalName, &result)
	}

	return result
}

// astDecl bundles an *ast.FuncDecl with its FileSet and source file
// name. It is local to the validator.
type astDecl struct {
	file string
	fset *token.FileSet
	fn   *ast.FuncDecl
}

// validateInternalEntryPoint checks that fn has exactly one
// dispatcher.Dispatch call, no post-dispatch protected calls, and
// only the allowed argument-list calls. Every violation is appended
// to result.
func validateInternalEntryPoint(d astDecl, allowedArgCalls []string, result *StructuralValidatorResult) {
	calls := dispatchCallsFor(d.fn)
	if len(calls) == 0 {
		result.Errors = append(result.Errors,
			fmt.Errorf("%s: dispatcher.Dispatch call not found", d.fn.Name.Name))
		return
	}
	if len(calls) > 1 {
		result.Errors = append(result.Errors,
			fmt.Errorf("%s: dispatcher.Dispatch called %d times, want exactly 1",
				d.fn.Name.Name, len(calls)))
		return
	}
	dispatchCall := calls[0]

	// Walk ONLY the immediate arguments of the dispatch call (not the
	// whole body). Preceding nodes like DispatcherForVerifier or
	// Errorf are part of separate statements and must not be flagged.
	// Anything that is a direct argument of dispatch(...) and is not
	// in allowedArgCalls is rejected.
	allowed := map[string]bool{}
	for _, n := range allowedArgCalls {
		allowed[n] = true
	}
	for _, arg := range dispatchCall.args {
		if arg == nil {
			continue
		}
		// Recursively inspect the argument expression: any nested
		// CallExpr is part of the argument tree. Only the top-level
		// call expression must match the allowed set.
		ast.Inspect(arg, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			// The dispatch call itself is wrapped by the parser
			// differently; we never inspect inside dispatcher.Dispatch
			// here because the dispatcher call expression IS the
			// outer node.
			if call.Pos() == dispatchCall.pos {
				return false
			}
			name := callName(call)
			if name == "" {
				return true
			}
			// Inline closures and nested helpers must not hide
			// protected calls inside the dispatch argument list.
			// We permit only direct CallExpr expressions whose name
			// is in the allowed set.
			if !allowed[name] {
				pos := d.fset.Position(call.Pos())
				result.Errors = append(result.Errors,
					fmt.Errorf("%s: disallowed call %q at line %d col %d inside dispatcher.Dispatch argument list (allowed: %v)",
						d.fn.Name.Name, name, pos.Line, pos.Column, allowedArgCalls))
			}
			return true
		})
	}

	// Walk every node and flag any protected call whose position is
	// strictly AFTER dispatch.end. Anything inside dispatch(end-of-call)
	// is fine (it ran before dispatch returned); anything after is
	// not.
	ast.Inspect(d.fn.Body, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if call.Pos() == dispatchCall.pos {
			return true
		}
		if call.Pos() < dispatchCall.end {
			return true
		}
		name := callName(call)
		if name == "" {
			return true
		}
		for _, bad := range protectedCallNamesForStructural {
			if name == bad {
				pos := d.fset.Position(call.Pos())
				result.Errors = append(result.Errors,
					fmt.Errorf("%s: protected call %q at line %d col %d after dispatcher.Dispatch",
						d.fn.Name.Name, name, pos.Line, pos.Column))
			}
		}
		return true
	})

	// Additionally flag any protected call in any top-level statement
	// that begins AFTER the dispatch call's enclosing statement.
	dispatchStmtIdx := findStmtForPos(d.fn.Body, dispatchCall.pos)
	if dispatchStmtIdx < 0 {
		return
	}
	for j := dispatchStmtIdx + 1; j < len(d.fn.Body.List); j++ {
		ast.Inspect(d.fn.Body.List[j], func(n ast.Node) bool {
			if n == nil {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := callName(call)
			for _, bad := range protectedCallNamesForStructural {
				if name == bad {
					result.Errors = append(result.Errors,
						fmt.Errorf("%s: post-Dispatch statement at index %d contains protected call %q",
							d.fn.Name.Name, j, bad))
				}
			}
			return true
		})
	}
}

// validatePublicWrapper checks that fn delegates exactly once to
// internalName and contains no direct call to a protected operation.
func validatePublicWrapper(d astDecl, internalName string, result *StructuralValidatorResult) {
	delegateCount := 0
	ast.Inspect(d.fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == internalName {
			delegateCount++
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != internalName {
			return true
		}
		delegateCount++
		return true
	})
	if delegateCount != 1 {
		result.Errors = append(result.Errors,
			fmt.Errorf("%s: must delegate exactly once to %s, got %d calls",
				d.fn.Name.Name, internalName, delegateCount))
	}
	ast.Inspect(d.fn.Body, func(n ast.Node) bool {
		if n == nil {
			return true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		for _, bad := range protectedCallNamesForStructural {
			if name == bad {
				result.Errors = append(result.Errors,
					fmt.Errorf("%s: directly calls protected operation %q",
						d.fn.Name.Name, bad))
			}
		}
		return true
	})
}

// dispatchCallInfo records a single dispatcher.Dispatch call's token

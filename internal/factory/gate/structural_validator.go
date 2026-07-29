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
//
// The AllowedReceivers map keys on the selector name of a permitted
// CallExpr and gives the required receiver identifier. An empty entry
// accepts any receiver; a populated entry fails closed if the receiver
// identifier does not match. This catches the case where an attacker
// (or a careless refactor) renames the receiver while leaving the
// selector name unchanged, e.g. evil.BindRunner() vs binder.BindRunner().
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

	// AllowedReceivers maps a permitted selector name to the exact
	// receiver identifier that must appear in the AST. An empty entry
	// accepts any receiver. A populated entry fails closed when the
	// receiver identifier does not match.
	//
	// For example, {"BindRunner": "binder"} permits
	// `dispatcher.Dispatch(..., binder.BindRunner())` but rejects
	// `dispatcher.Dispatch(..., evil.BindRunner())` even though both
	// share the same selector name.
	AllowedReceivers map[string]string
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

// structuralValidatorResult aggregates every error produced by a
// single structural-validation pass. It is intentionally unexported
// because production callers have no need to inspect the result; the
// package's own self-validation test is the sole consumer.
type structuralValidatorResult struct {
	Errors []error
}

// OK reports whether the structural validation passed.
func (r structuralValidatorResult) OK() bool { return len(r.Errors) == 0 }

// typedDispatchContract captures the rules for every internal entry
// point and every public wrapper in the gate package.
type typedDispatchContract struct {
	InternalRules []typedDispatchRule
	PublicRules   []typedDelegationRule
}

// defaultTypedDispatchContract returns the canonical contract for the
// gate package. It is the single source of truth that both the
// production-source test and the adversarial fixtures exercise.
//
// It is unexported because it is consumed only by structural
// validation tests within the gate package.
func defaultTypedDispatchContract() typedDispatchContract {
	return typedDispatchContract{
		InternalRules: []typedDispatchRule{
			{
				FunctionName:     "dispatchDupcodeVerifyTypedWith",
				AllowedArgCalls:  []string{"BindRunner"},
				AllowedReceivers: map[string]string{"BindRunner": "binder"},
			},
			{
				FunctionName:     "dispatchDupcodeBaselineVerifyTypedWith",
				AllowedArgCalls:  []string{"BindRunner"},
				AllowedReceivers: map[string]string{"BindRunner": "binder"},
			},
			{
				FunctionName:     "dispatchDupcodeUpdateBaselineTypedWith",
				AllowedArgCalls:  []string{"BindRunner"},
				AllowedReceivers: map[string]string{"BindRunner": "binder"},
			},
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
//	disallowed receiver inside the dispatch argument list
//	missing public wrapper
//	duplicate public wrapper
//	zero public delegations
//	two public delegations
//	direct protected call from public wrapper
func validateTypedDispatchSource(
	files map[string]*ast.File,
	fset *token.FileSet,
	contract typedDispatchContract,
) structuralValidatorResult {
	var result structuralValidatorResult

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

	checkInternal := func(rule typedDispatchRule) astDecl {
		found := hits[rule.FunctionName]
		if len(found) == 0 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q is missing", rule.FunctionName))
			return astDecl{}
		}
		if len(found) > 1 {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q appears %d times, want exactly 1", rule.FunctionName, len(found)))
			return astDecl{}
		}
		if found[0].fn.Body == nil {
			result.Errors = append(result.Errors,
				fmt.Errorf("required declaration %q has nil body", rule.FunctionName))
			return astDecl{}
		}
		validateInternalEntryPoint(found[0], rule.AllowedArgCalls, rule.AllowedReceivers, &result)
		return found[0]
	}

	for _, rule := range contract.InternalRules {
		checkInternal(rule)
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
// only the allowed argument-list calls (matched by selector name AND
// receiver identity). Every violation is appended to result.
func validateInternalEntryPoint(
	d astDecl,
	allowedArgCalls []string,
	allowedReceivers map[string]string,
	result *structuralValidatorResult,
) {
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
			if !allowed[name] {
				pos := d.fset.Position(call.Pos())
				result.Errors = append(result.Errors,
					fmt.Errorf("%s: disallowed call %q at line %d col %d inside dispatcher.Dispatch argument list (allowed: %v)",
						d.fn.Name.Name, name, pos.Line, pos.Column, allowedArgCalls))
				return true
			}
			// Selector name matches; now confirm the receiver
			// identity matches the contract. A malicious or careless
			// refactor that renames the receiver (e.g. evil.BindRunner)
			// while keeping the selector name unchanged is rejected
			// here.
			if want, ok := allowedReceivers[name]; ok && want != "" {
				got := callReceiverName(call)
				if got != want {
					pos := d.fset.Position(call.Pos())
					result.Errors = append(result.Errors,
						fmt.Errorf("%s: %q at line %d col %d has receiver %q, want %q (selector name matches but receiver identity differs)",
							d.fn.Name.Name, name, pos.Line, pos.Column, got, want))
				}
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
func validatePublicWrapper(d astDecl, internalName string, result *structuralValidatorResult) {
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
// interval and its direct argument expressions. It is the validator's
// internal representation.
type dispatchCallInfo struct {
	pos  token.Pos
	end  token.Pos
	args []ast.Expr
}

// dispatchCallsFor returns every dispatcher.Dispatch call in fn that
// uses the bare identifier "dispatcher" as the receiver.
func dispatchCallsFor(fn *ast.FuncDecl) []dispatchCallInfo {
	var out []dispatchCallInfo
	if fn == nil || fn.Body == nil {
		return out
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Dispatch" {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != "dispatcher" {
			return true
		}
		out = append(out, dispatchCallInfo{
			pos:  call.Pos(),
			end:  call.End(),
			args: call.Args,
		})
		return true
	})
	return out
}

// callName returns the canonical name used for protected-call and
// allowed-call matching: the last selector segment for SelectorExpr
// calls, the bare identifier for plain function calls, or "" when the
// call is on something we do not match (e.g. method expressions).
func callName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// callReceiverName returns the receiver identifier of a selector
// call expression (e.g. `binder` for `binder.BindRunner()`) or "" for
// a bare identifier call. It is used by the validator to confirm the
// receiver identity against AllowedReceivers.
func callReceiverName(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

// findStmtForPos returns the index of the top-level statement in
// body.List whose token range contains pos, or -1 when no enclosing
// statement is found.
func findStmtForPos(body *ast.BlockStmt, pos token.Pos) int {
	for i, s := range body.List {
		if pos >= s.Pos() && pos <= s.End() {
			return i
		}
	}
	return -1
}

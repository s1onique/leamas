// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"go/ast"
	"go/token"
)

// typedDispatchRule is the dispatch contract for an internal *With
// entry point: there must be exactly one dispatcher.Dispatch call,
// no post-dispatch protected call, and only the direct
// `binder.BindRunner()` may appear inside the dispatch argument list.
//
// AllowedReceivers maps a permitted selector name to the exact
// receiver identifier that must appear in the AST. An empty entry
// accepts any receiver. A populated entry fails closed when the
// receiver identifier does not match — catching renames like
// evil.BindRunner() that preserve the selector name.
type typedDispatchRule struct {
	FunctionName     string
	AllowedArgCalls  []string
	AllowedReceivers map[string]string
}

type typedDelegationRule struct {
	PublicName   string
	InternalName string
}

type structuralValidatorResult struct {
	Errors []error
}

func (r structuralValidatorResult) OK() bool { return len(r.Errors) == 0 }

type typedDispatchContract struct {
	InternalRules []typedDispatchRule
	PublicRules   []typedDelegationRule
}

func defaultTypedDispatchContract() typedDispatchContract {
	return typedDispatchContract{
		InternalRules: []typedDispatchRule{
			{FunctionName: "dispatchDupcodeVerifyTypedWith", AllowedArgCalls: []string{"BindRunner"}, AllowedReceivers: map[string]string{"BindRunner": "binder"}},
			{FunctionName: "dispatchDupcodeBaselineVerifyTypedWith", AllowedArgCalls: []string{"BindRunner"}, AllowedReceivers: map[string]string{"BindRunner": "binder"}},
			{FunctionName: "dispatchDupcodeUpdateBaselineTypedWith", AllowedArgCalls: []string{"BindRunner"}, AllowedReceivers: map[string]string{"BindRunner": "binder"}},
		},
		PublicRules: []typedDelegationRule{
			{PublicName: "DispatchDupcodeVerifyTyped", InternalName: "dispatchDupcodeVerifyTypedWith"},
			{PublicName: "DispatchDupcodeBaselineVerifyTyped", InternalName: "dispatchDupcodeBaselineVerifyTypedWith"},
			{PublicName: "DispatchDupcodeUpdateBaselineTyped", InternalName: "dispatchDupcodeUpdateBaselineTypedWith"},
		},
	}
}

// protectedCallNamesForStructural is the canonical list of symbol names
// that must NOT appear in the post-Dispatch region or inside any
// public wrapper. Matched against the LAST selector segment of a
// CallExpr.
var protectedCallNamesForStructural = []string{
	"run", "BindRunner", "NewRunner", "NewDupcodeRunner",
	"LoadBaseline", "RunCheckReport", "VerifyBaseline",
	"WriteBaseline", "CompareToBaseline",
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

	allowed := map[string]bool{}
	for _, n := range allowedArgCalls {
		allowed[n] = true
	}
	for _, arg := range dispatchCall.args {
		if arg == nil {
			continue
		}
		ast.Inspect(arg, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
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
	// strictly AFTER dispatch.end.
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

// validateTypedDispatchSource lives in
// structural_validator_validate.go to keep this file under the
// LLM-friendly line threshold.

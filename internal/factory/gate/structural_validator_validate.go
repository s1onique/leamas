// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"fmt"
	"go/ast"
	"go/token"
)

// validateTypedDispatchSource runs one structural-validation pass over
// the parsed production files using the supplied contract. It returns
// every violation found; an empty error slice means the contract holds.
//
// The validator is fail-closed. It rejects:
//   - missing internal declaration
//   - duplicate internal declaration
//   - nil internal body
//   - zero dispatcher.Dispatch calls
//   - two dispatcher.Dispatch calls
//   - protected call after dispatch
//   - protected call later in the same statement
//   - disallowed call inside the dispatch argument list
//   - disallowed receiver inside the dispatch argument list
//   - missing public wrapper
//   - duplicate public wrapper
//   - zero public delegations
//   - two public delegations
//   - direct protected call from public wrapper
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

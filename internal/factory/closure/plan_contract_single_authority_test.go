// SPDX-License-Identifier: Apache-2.0

// Package closure - plan_contract_single_authority_test.go
// is the B2-R7-R1 AST/source-inspection guard for the
// closure package's Plan Contract semantic authority.
//
// See plan_contract_single_authority_helpers_test.go for
// the supporting helper functions and the guard
// documentation.
package closure



// Package closure - plan_contract_single_authority_test.go
// is the B2-R7-R1 AST/source-inspection guard for the
// closure package's Plan Contract semantic authority.
//
// B2-R7-R1 single-authority rule: every wire-contract rule
// for the Plan Contract v1 lives in the plancontract leaf.
// The closure package's legacy typed validators
// (validatePlanTyped, ValidateRunnerAuthority, etc.) are
// adapters; their bodies MUST contain only:
//
//   - a canonical call to plancontract,
//   - error adaptation (typed-error mapping), and
//   - representation conversion (typed Plan <-> wire bytes).
//
// This test inspects the closure package's source files
// for:
//
//   - legacy typed-validator symbols that contain
//     non-canonical calls,
//   - newly introduced validate*Plan / *Authority
//     production functions that do not delegate to
//     plancontract,
//   - semantic comparisons against Plan fields
//     (e.g. `if plan.X != Y`) that indicate an
//     independent semantic rule,
//   - direct regexp matching against Plan fields
//     (e.g. `oidPattern.MatchString(plan.X)`) that
//     indicate an independent shape rule.
//
// Drift in any of these categories is a contract bug; the
// guard fails. The guard uses go/parser so it is
// deterministic across Go versions and does not depend on
// private runtime state.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// legacyValidatorSymbols is the closed list of legacy
// typed-validator symbols the guard inspects. The named
// functions MUST contain only canonical calls (or stdlib
// calls); any other call fails the guard.
var legacyValidatorSymbols = []string{
	"ValidatePlan",
	"ValidateRunnerAuthority",
	"validatePlanTyped",
}

// newValidatorRegex matches newly introduced production
// functions whose names suggest a Plan Contract semantic
// authority. Adding a function with such a name without
// delegating to plancontract triggers the guard.
//
// The patterns are deliberately precise so the guard does
// not fire on unrelated validators (e.g.
// `validateV2EvidenceAuthority` is for the v2 evidence
// domain, not the Plan Contract v1 wire format).
var newValidatorRegex = []string{
	"validatePlan",
	"validatePlan[A-Z][a-zA-Z]*",
	"validateAuthority",
	"validateAuthority[A-Z][a-zA-Z]*",
	"validate[A-Z][a-zA-Z]*Authority",
}

// adapterCallAllowedPrefixes is the closed list of call
// prefixes that count as "canonical call / adaptation /
// conversion" primitives for legacy validators. Stdlib
// helpers (json, fmt, errors, strings) are allowed because
// the adapter needs them to marshal / format errors.
var adapterCallAllowedPrefixes = []string{
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
	"newSemanticError",
	"isPolicyMissingField",
	"errorsAsDecodeError",
	"missingPolicyFields",
}

// planFieldNames is the closed list of Plan field names
// the guard watches for semantic-comparison drift. A
// production function that compares a Plan field to a
// literal or constant is a candidate for an independent
// semantic rule that the closure package must not own.
var planFieldNames = []string{
	"ContractVersion",
	"ActID",
	"Mode",
	"Execution",
	"Policy",
	"RunnerAuthority",
	"Baseline",
	"Checks",
	"Artifacts",
}

// planRegexHelpers is the closed list of regex variables
// in the closure package whose MatchString against a Plan
// field indicates an independent shape rule.
var planRegexHelpers = []string{
	"oidPattern",
	"actIDPattern",
	"itemIDPattern",
	"environmentNamePattern",
}

// nonPlanValidatorAllowlist is the closed set of
// pre-existing production validators that operate on a
// non-Plan-contract domain (e.g. v2 evidence authority).
// The guard skips these because their semantic authority
// lives outside the Plan Contract v1 wire format.
var nonPlanValidatorAllowlist = map[string]bool{
	"validateV2EvidenceAuthority": true,
}

// isNonPlanValidator reports whether name belongs to the
// allowlist of validators that operate on a non-Plan-contract
// domain. The guard skips these so unrelated domain validators
// do not trip the single-authority rule.
func isNonPlanValidator(name string) bool {
	return nonPlanValidatorAllowlist[name]
}

// TestPlanContractSingleSemanticAuthority is the
// B2-R7-R1 AST/source guard. It walks every closure-package
// production source file and asserts:
//
//   - every legacy validator body contains only canonical
//     calls,
//   - every newly introduced validate*Plan* / *Authority*
//     production function delegates to plancontract,
//   - no production function compares Plan fields to
//     literals,
//   - no production function matches Plan fields against
//     the closure regex helpers.
func TestPlanContractSingleSemanticAuthority(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
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
				// Legacy validator symbol: only allow
				// canonical calls in the body.
				if legacySymbol(fn.Name.Name) {
					assertOnlyAllowedCalls(t, fn)
				}
				// Newly introduced validator symbol:
				// MUST delegate to plancontract (unless it
				// is a pre-existing non-Plan validator in
				// the explicit allowlist).
				if isNewValidatorSymbol(fn.Name.Name) && !isNonPlanValidator(fn.Name.Name) {
					assertDelegatesToPlanContract(t, fn)
				}
				// Semantic comparisons against Plan fields
				// and direct regexp matching are only
				// checked for functions whose name suggests
				// they own a Plan Contract semantic rule.
				// Other production functions (fixture
				// builders, helpers) are not authority
				// surfaces and may legitimately compare
				// Plan fields.
				if looksLikePlanContractAuthority(fn.Name.Name) {
					assertNoPlanFieldSemanticComparisons(t, fn)
					assertNoPlanFieldRegexMatching(t, fn)
				}
			}
		}
	}
}

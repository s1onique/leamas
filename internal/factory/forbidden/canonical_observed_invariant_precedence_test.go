// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/s1onique/leamas/internal/factory/checks"
)

// TestInvPrecedenceInvalidAnonymousEdgePoisonMatchingApproval locks the
// precedence that an invalid internal reference class wins over the
// anonymous-caller policy. The matching approval is poisoned; the
// stale / cardinality / ordinary-bypass cascade is suppressed; the
// anonymous-caller finding is NOT emitted (the invariant takes
// precedence).
func TestInvPrecedenceInvalidAnonymousEdgePoisonMatchingApproval(t *testing.T) {
	callerObj := types.NewVar(token.Pos(0), nil, "caller", nil)
	calleeObj := types.NewVar(token.Pos(0), nil, "callee", nil)

	approval := ApprovedCaller{
		PackagePath:    "example.test/policy/caller",
		Function:       "Allowed",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "example.test/policy/protected",
			Name:        "Cap",
			Kind:        ProtectedPackageFunction,
		},
	}

	// Simulate an anonymous edge with an invalid reference class.
	invalidEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: "example.test/policy/caller",
			Function:    "<func-literal:5:3>",
			Kind:        CallerKindFunctionLiteral,
		},
		Callee:         approval.Callee,
		ReferenceClass: ReferenceClass("DECLARATION"), // invalid
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 5, Column: 3},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis([]ApprovedCaller{approval}, []ObservedEdge{invalidEdge})
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	// Mutate the state to claim a matching caller object so the
	// invariant-violation path actually finds a candidate to poison.
	// In this seam we directly construct the invalid edge with the
	// same objects the approval state references.
	analysis.observedEdges[0] = invalidEdge

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0 (invalid class wins)", got)
	}
	for _, kind := range []string{
		"authority_policy_stale_approval",
		"authority_policy_reference_class_mismatch",
		"authority_policy_edge_cardinality_mismatch",
		"dupcode_bypass",
		"dupcode_adapter_bypass",
		"dupcode_protected_function_value",
	} {
		if count := findingKindsCount(analysis.findings, kind); count != 0 {
			t.Errorf("%s count = %d, want 0 (cascade suppressed)", kind, count)
		}
	}
	state := &analysis.approvalStates[0]
	if !state.observedInvariantFailure {
		t.Errorf("observedInvariantFailure = false, want true")
	}
	if state.validated {
		t.Errorf("validated = true, want false (poisoned)")
	}
}

// TestInvPrecedenceInvalidAnonymousEdgeWithoutApproval locks the
// invariant precedence when no approval matches the edge. The invalid
// class still wins. No cascade.
func TestInvPrecedenceInvalidAnonymousEdgeWithoutApproval(t *testing.T) {
	callerObj := types.NewVar(token.Pos(0), nil, "caller", nil)
	calleeObj := types.NewVar(token.Pos(0), nil, "callee", nil)

	invalidEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: "example.test/policy/orphan",
			Function:    "<func-literal:7:5>",
			Kind:        CallerKindFunctionLiteral,
		},
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "example.test/policy/protected",
			Name:        "Cap",
			Kind:        ProtectedPackageFunction,
		},
		ReferenceClass: ReferenceClass("BOGUS"),
		Path:           "example.test/policy/orphan/orphan.go",
		Position:       token.Position{Line: 7, Column: 5},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis(nil, []ObservedEdge{invalidEdge})
	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0 (invalid class wins)", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_stale_approval"); got != 0 {
		t.Errorf("stale_approval count = %d, want 0 (no cascade)", got)
	}
}

// TestInvPrecedenceValidAnonymousDirectCallPreserved locks the
// preservation of the anonymous-caller finding when the observed
// reference class is internally valid. The implementation must remain
// on the anonymous-caller path and not regress into the invalid-class
// branch.
func TestInvPrecedenceValidAnonymousDirectCallPreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		p.Cap()
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Allowed", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 1 {
		t.Errorf("anonymous_caller count = %d, want 1", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "dupcode_bypass"); got != 0 {
		t.Errorf("dupcode_bypass count = %d, want 0", got)
	}
}

// TestInvPrecedenceValidAnonymousFunctionValuePreserved locks the
// preservation of FUNCTION_VALUE classification through the invariant
// precedence. The anonymous-caller finding is emitted with the
// FUNCTION_VALUE class.
func TestInvPrecedenceValidAnonymousFunctionValuePreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() {
	func() {
		f := p.Cap
		_ = f
	}()
}
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	approval := fixtureApproval(fixture.packagePath("caller"), "Allowed", "", CallerKindPackageFunction, symbol, refDirectCall)
	result := fixture.run([]ProtectedSymbol{symbol}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 1 {
		t.Errorf("anonymous_caller count = %d, want 1", got)
	}
	functionValueEdge := 0
	for _, edge := range result.ObservedEdges {
		if edge.Caller.Kind == CallerKindFunctionLiteral &&
			edge.ReferenceClass == refFunctionValue {
			functionValueEdge++
		}
	}
	if functionValueEdge != 1 {
		t.Errorf("function-value edges inside literal = %d, want 1", functionValueEdge)
	}
}

// TestInvPrecedenceValidNamedEdgePreserved confirms named-caller
// matching and validation remain unchanged. The named direct call is
// matched; no anonymous finding; no cascade.
func TestInvPrecedenceValidNamedEdgePreserved(t *testing.T) {
	fixture, protected, callerPkg := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindCount(result.Findings, "dupcode_bypass"); got != 0 {
		t.Errorf("dupcode_bypass count = %d, want 0", got)
	}
	if result.Stats.ValidatedApprovals != 1 {
		t.Errorf("validated approvals = %d, want 1", result.Stats.ValidatedApprovals)
	}
}

// TestInvPrecedenceDotImportPreserved confirms DOT_IMPORT remains
// categorically observed and reported without internal-invariant
// findings.
func TestInvPrecedenceDotImportPreserved(t *testing.T) {
	fixture := newCanonicalFixture(t)
	protectedPkg := fixture.packagePath("protected")
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("dotcaller/dot.go", `package dotcaller
import . "example.test/policy/protected"
func Dot() { Cap() }
`)
	symbol := fixtureSymbol(AuthorityLayerRaw, protectedPkg, "Cap", ProtectedPackageFunction, "")
	result := fixture.run([]ProtectedSymbol{symbol}, nil)

	if got := findingKindCount(result.Findings, "dupcode_dot_import"); got != 1 {
		t.Errorf("dupcode_dot_import count = %d, want 1", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0 (DOT_IMPORT is internally valid)", got)
	}
	if got := findingKindCount(result.Findings, "authority_policy_anonymous_caller"); got != 0 {
		t.Errorf("anonymous_caller count = %d, want 0", got)
	}
}

// findingKindCount returns the count of checks.Finding with the given
// kind. It is independent of the canonical-finding seam used by other
// pipeline tests.
func findingKindCount(findings []checks.Finding, kind string) int {
	count := 0
	for _, finding := range findings {
		if finding.Kind == kind {
			count++
		}
	}
	return count
}

// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"testing"

	"go/token"
	"go/types"
)

// buildPipelineAnalysis creates a minimal canonicalAnalysis populated with
// the supplied approvals and observed edges. The seam is intentionally
// narrow: it exists entirely in _test.go so the production API surface
// is not extended. Callers may then run analysis.validateObservedEdges()
// and analysis.validateConfiguredApprovals() in isolation.
func buildPipelineAnalysis(
	approvals []ApprovedCaller,
	edges []ObservedEdge,
) *canonicalAnalysis {
	a := &canonicalAnalysis{
		approvalStates: make([]resolvedApproval, len(approvals)),
		observedEdges:  edges,
	}
	for index, approval := range approvals {
		a.approvalStates[index] = resolvedApproval{approval: approval, valid: true}
	}
	return a
}

// withCallerCalleeObjects sets callerObject and calleeObject on a single
// approval state so the test seam can exercise object-identity matching.
func withCallerCalleeObjects(
	state *resolvedApproval,
	callerObject types.Object,
	calleeObject types.Object,
) {
	state.callerObject = callerObject
	state.calleeObject = calleeObject
}

func findingKindsCount(findings []canonicalFinding, kind string) int {
	count := 0
	for _, finding := range findings {
		if finding.finding.Kind == kind {
			count++
		}
	}
	return count
}

// TestObservedReferenceClassPipelineIsolation exercises the actual
// analysis pipeline. An invalid observed reference class must emit its
// invariant finding, poison the relevant approval state, and never let
// the poisoned state participate in stale-approval, cardinality, or
// reference-class-mismatch accounting.
func TestObservedReferenceClassPipelineIsolation(t *testing.T) {
	// Use any two distinct sentinel objects so the approval state can be
	// matched by identity without going through real package loading.
	var (
		callerObj types.Object = types.NewVar(token.Pos(0), nil, "caller", nil)
		calleeObj types.Object = types.NewVar(token.Pos(0), nil, "callee", nil)
	)

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

	// One observed edge with the same caller/callee objects but an
	// invalid internal reference class.
	invalidEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: approval.PackagePath,
			Function:    approval.Function,
			Kind:        approval.CallerKind,
		},
		Callee:         approval.Callee,
		ReferenceClass: refDeclaration,
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 10, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis([]ApprovedCaller{approval}, []ObservedEdge{invalidEdge})
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1", got)
	}
	for _, kind := range []string{
		"authority_policy_stale_approval",
		"authority_policy_reference_class_mismatch",
		"authority_policy_edge_cardinality_mismatch",
		"dupcode_bypass",
		"dupcode_protected_function_value",
		"dupcode_adapter_bypass",
		"dupcode_adapter_function_value",
		"dupcode_gate_bypass",
	} {
		if count := findingKindsCount(analysis.findings, kind); count != 0 {
			t.Errorf("%s count = %d, want 0 (cascade isolated)", kind, count)
		}
	}

	state := &analysis.approvalStates[0]
	if state.matches != 0 {
		t.Errorf("matches = %d, want 0", state.matches)
	}
	if state.validated {
		t.Errorf("validated = true, want false (poisoned)")
	}
	if !state.observedInvariantFailure {
		t.Errorf("observedInvariantFailure = false, want true")
	}
	if state.referenceMismatch {
		t.Errorf("referenceMismatch = true, want false")
	}
}

// TestObservedReferenceClassPipelineInvalidEdgeWithoutApproval emits the
// invariant finding even when no approval state matches the edge's
// caller/callee objects. The edge must never produce a stale approval
// finding because no approval is connected to it.
func TestObservedReferenceClassPipelineInvalidEdgeWithoutApproval(t *testing.T) {
	callerObj := types.NewVar(token.Pos(0), nil, "caller", nil)
	calleeObj := types.NewVar(token.Pos(0), nil, "callee", nil)

	invalidEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: "example.test/policy/orphan",
			Function:    "Orphan",
			Kind:        CallerKindPackageFunction,
		},
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "example.test/policy/protected",
			Name:        "Cap",
			Kind:        ProtectedPackageFunction,
		},
		ReferenceClass: ReferenceClass("BOGUS"),
		Path:           "example.test/policy/orphan/orphan.go",
		Position:       token.Position{Line: 5, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis(nil, []ObservedEdge{invalidEdge})

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_stale_approval"); got != 0 {
		t.Errorf("stale_approval count = %d, want 0 (no approval to be stale)", got)
	}
	for _, kind := range []string{
		"dupcode_bypass",
		"dupcode_protected_function_value",
		"dupcode_adapter_bypass",
		"dupcode_adapter_function_value",
		"dupcode_gate_bypass",
	} {
		if count := findingKindsCount(analysis.findings, kind); count != 0 {
			t.Errorf("%s count = %d, want 0 (no ordinary bypass)", kind, count)
		}
	}
}

// TestObservedReferenceClassPipelineMultipleApprovalsPoisoned ensures
// defensive behavior when two valid approval states share the same
// caller/callee objects: both are poisoned by one invalid edge, and the
// invariant finding is emitted exactly once for the one edge.
func TestObservedReferenceClassPipelineMultipleApprovalsPoisoned(t *testing.T) {
	callerObj := types.NewVar(token.Pos(0), nil, "caller", nil)
	calleeObj := types.NewVar(token.Pos(0), nil, "callee", nil)

	makeApproval := func() ApprovedCaller {
		return ApprovedCaller{
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
	}

	analysis := buildPipelineAnalysis(
		[]ApprovedCaller{makeApproval(), makeApproval()},
		nil,
	)
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)
	withCallerCalleeObjects(&analysis.approvalStates[1], callerObj, calleeObj)

	invalidEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: "example.test/policy/caller",
			Function:    "Allowed",
			Kind:        CallerKindPackageFunction,
		},
		Callee: ProtectedSymbol{
			Layer:       AuthorityLayerRaw,
			PackagePath: "example.test/policy/protected",
			Name:        "Cap",
			Kind:        ProtectedPackageFunction,
		},
		ReferenceClass: refDeclaration,
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 10, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}
	analysis.observedEdges = []ObservedEdge{invalidEdge}

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1 (one per edge, not per approval)", got)
	}
	for index, state := range analysis.approvalStates {
		if !state.observedInvariantFailure {
			t.Errorf("approval %d not poisoned", index)
		}
		if state.validated {
			t.Errorf("approval %d validated, want false", index)
		}
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_stale_approval"); got != 0 {
		t.Errorf("stale_approval count = %d, want 0", got)
	}
}

// TestObservedReferenceClassPipelineMixedEdges exercises the fail-closed
// result when a valid matching observed edge coexists with an invalid one
// that shares the same caller/callee objects. The presence of one valid
// match must not erase the internal invariant violation; the approval is
// not validated and no stale/cardinality finding is emitted.
func TestObservedReferenceClassPipelineMixedEdges(t *testing.T) {
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

	makeEdge := func(class ReferenceClass) ObservedEdge {
		return ObservedEdge{
			Caller: CallerIdentity{
				PackagePath: approval.PackagePath,
				Function:    approval.Function,
				Kind:        approval.CallerKind,
			},
			Callee:         approval.Callee,
			ReferenceClass: class,
			Path:           "example.test/policy/caller/caller.go",
			Position:       token.Position{Line: 10, Column: 1},
			callerObject:   callerObj,
			calleeObject:   calleeObj,
		}
	}

	analysis := buildPipelineAnalysis(
		[]ApprovedCaller{approval},
		[]ObservedEdge{makeEdge(refDirectCall), makeEdge(refDeclaration)},
	)
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 1 {
		t.Errorf("observed_reference_class_invalid count = %d, want 1", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_stale_approval"); got != 0 {
		t.Errorf("stale_approval count = %d, want 0 (poisoned suppresses stale)", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_edge_cardinality_mismatch"); got != 0 {
		t.Errorf("edge_cardinality_mismatch count = %d, want 0 (poisoned suppresses cardinality)", got)
	}
	state := &analysis.approvalStates[0]
	if state.validated {
		t.Errorf("validated = true, want false (poisoned)")
	}
	if !state.observedInvariantFailure {
		t.Errorf("observedInvariantFailure = false, want true")
	}
}

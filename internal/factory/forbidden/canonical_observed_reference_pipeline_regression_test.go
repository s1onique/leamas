// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"testing"

	"go/token"
	"go/types"
)

// TestObservedReferenceClassPipelineValidExactMatch locks the regression
// that valid exact-match behavior still works: one valid observed edge
// increments matches, sets validated = true, and emits no cascade.
func TestObservedReferenceClassPipelineValidExactMatch(t *testing.T) {
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
	validEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: approval.PackagePath,
			Function:    approval.Function,
			Kind:        approval.CallerKind,
		},
		Callee:         approval.Callee,
		ReferenceClass: refDirectCall,
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 10, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis([]ApprovedCaller{approval}, []ObservedEdge{validEdge})
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	state := &analysis.approvalStates[0]
	if state.matches != 1 {
		t.Errorf("matches = %d, want 1", state.matches)
	}
	if !state.validated {
		t.Errorf("validated = false, want true")
	}
	if state.observedInvariantFailure {
		t.Errorf("observedInvariantFailure = true, want false")
	}
	for _, kind := range []string{
		"authority_policy_observed_reference_class_invalid",
		"authority_policy_reference_class_mismatch",
		"authority_policy_stale_approval",
		"authority_policy_edge_cardinality_mismatch",
	} {
		if count := findingKindsCount(analysis.findings, kind); count != 0 {
			t.Errorf("%s count = %d, want 0", kind, count)
		}
	}
}

// TestObservedReferenceClassPipelineWrongValidReference locks the
// regression that a wrong-but-valid observed reference class still
// produces authority_policy_reference_class_mismatch and never a stale
// cascade.
func TestObservedReferenceClassPipelineWrongValidReference(t *testing.T) {
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
	wrongEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: approval.PackagePath,
			Function:    approval.Function,
			Kind:        approval.CallerKind,
		},
		Callee:         approval.Callee,
		ReferenceClass: refFunctionValue,
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 10, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis([]ApprovedCaller{approval}, []ObservedEdge{wrongEdge})
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_reference_class_mismatch"); got != 1 {
		t.Errorf("reference_class_mismatch count = %d, want 1", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0", got)
	}
	if got := findingKindsCount(analysis.findings, "authority_policy_stale_approval"); got != 0 {
		t.Errorf("stale_approval count = %d, want 0", got)
	}
}

// TestObservedReferenceClassPipelineDotImportStillCategorical locks the
// regression that DOT_IMPORT remains a valid observed class. DOT_IMPORT
// must not produce authority_policy_observed_reference_class_invalid and
// the categorical dot-import/bypass policy must continue to apply.
func TestObservedReferenceClassPipelineDotImportStillCategorical(t *testing.T) {
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
	dotEdge := ObservedEdge{
		Caller: CallerIdentity{
			PackagePath: approval.PackagePath,
			Function:    approval.Function,
			Kind:        approval.CallerKind,
		},
		Callee:         approval.Callee,
		ReferenceClass: refDotImport,
		Path:           "example.test/policy/caller/caller.go",
		Position:       token.Position{Line: 10, Column: 1},
		callerObject:   callerObj,
		calleeObject:   calleeObj,
	}

	analysis := buildPipelineAnalysis([]ApprovedCaller{approval}, []ObservedEdge{dotEdge})
	withCallerCalleeObjects(&analysis.approvalStates[0], callerObj, calleeObj)

	analysis.validateObservedEdges()
	analysis.validateConfiguredApprovals()

	if got := findingKindsCount(analysis.findings, "authority_policy_observed_reference_class_invalid"); got != 0 {
		t.Errorf("observed_reference_class_invalid count = %d, want 0 (DOT_IMPORT is valid observed class)", got)
	}
}

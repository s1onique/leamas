// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/token"
	"go/types"
	"strings"
)

type resolvedApproval struct {
	approval          ApprovedCaller
	callerObject      types.Object
	calleeObject      types.Object
	matches           int
	valid             bool
	duplicate         bool
	referenceMismatch bool
	validated         bool
}

func (a *canonicalAnalysis) resolveConfiguredApprovals() {
	normalized := make([]ApprovedCaller, len(a.config.approvals))
	counts := make(map[ApprovedCaller]int)
	for index, approval := range a.config.approvals {
		normalized[index] = normalizeApproval(approval)
		counts[normalized[index]]++
	}

	a.approvalStates = make([]resolvedApproval, len(normalized))
	for index, approval := range normalized {
		state := resolvedApproval{approval: approval}
		if counts[approval] > 1 {
			state.duplicate = true
			a.approvalFinding("authority_policy_duplicate_approval", approval, "approval appears more than once")
			a.approvalStates[index] = state
			continue
		}
		if invalidCallerApproval(approval) {
			a.approvalFinding("authority_policy_caller_missing", approval, "caller identity is empty, wildcarded, or unstable")
			a.approvalStates[index] = state
			continue
		}
		candidates := a.callerCandidates[approvalCallerIdentity(approval)]
		switch len(candidates) {
		case 0:
			a.approvalFinding("authority_policy_caller_missing", approval, "configured caller declaration not found")
			a.approvalStates[index] = state
			continue
		case 1:
			state.callerObject = candidates[0]
		default:
			a.approvalFinding("authority_policy_caller_ambiguous", approval, "configured caller resolves more than once")
			a.approvalStates[index] = state
			continue
		}
		calleeObject := a.objectByProtected[approval.Callee]
		if calleeObject == nil {
			a.approvalFinding("authority_policy_callee_missing", approval, "configured callee declaration did not resolve")
			a.approvalStates[index] = state
			continue
		}
		if !validApprovalReferenceClass(approval.ReferenceClass) {
			a.approvalFinding("authority_policy_reference_class_mismatch", approval, "unknown or declaration-only reference class")
			a.approvalStates[index] = state
			continue
		}
		state.calleeObject = calleeObject
		state.valid = true
		a.approvalStates[index] = state
	}
}

func normalizeApproval(approval ApprovedCaller) ApprovedCaller {
	if approval.CallerKind == "" {
		approval.CallerKind = CallerKindPackageFunction
		if approval.Receiver != "" {
			approval.CallerKind = CallerKindMethod
		}
	}
	if approval.ReferenceClass == "" {
		approval.ReferenceClass = refDirectCall
	}
	if approval.Cardinality <= 0 {
		approval.Cardinality = 1
	}
	return approval
}

func invalidCallerApproval(approval ApprovedCaller) bool {
	return approval.PackagePath == "" || approval.Function == "" ||
		strings.ContainsAny(approval.Function, "*@") ||
		strings.Contains(approval.PackagePath, "*") ||
		strings.Contains(approval.Receiver, "*")
}

func approvalCallerIdentity(approval ApprovedCaller) CallerIdentity {
	return CallerIdentity{
		PackagePath: approval.PackagePath,
		Function:    approval.Function,
		Receiver:    approval.Receiver,
		Kind:        approval.CallerKind,
	}
}

func validApprovalReferenceClass(class ReferenceClass) bool {
	switch class {
	case refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable, refDotImport:
		return true
	default:
		return false
	}
}

func (a *canonicalAnalysis) validateObservedEdges() {
	for edgeIndex := range a.observedEdges {
		edge := &a.observedEdges[edgeIndex]
		var exact []*resolvedApproval
		var wrongReference []*resolvedApproval
		for index := range a.approvalStates {
			state := &a.approvalStates[index]
			if !state.valid || state.callerObject != edge.callerObject || state.calleeObject != edge.calleeObject {
				continue
			}
			if state.approval.ReferenceClass == edge.ReferenceClass {
				exact = append(exact, state)
			} else {
				wrongReference = append(wrongReference, state)
			}
		}
		if len(exact) == 1 {
			exact[0].matches++
			continue
		}
		if len(wrongReference) > 0 {
			for _, state := range wrongReference {
				state.referenceMismatch = true
			}
			a.edgeFinding("authority_policy_reference_class_mismatch", *edge, "observed reference class does not match configured approval")
			continue
		}
		a.edgeFinding(bypassFindingKind(*edge), *edge, "protected source edge has no exact approval")
	}
}

func (a *canonicalAnalysis) validateConfiguredApprovals() {
	for index := range a.approvalStates {
		state := &a.approvalStates[index]
		if !state.valid || state.referenceMismatch {
			continue
		}
		switch {
		case state.matches == 0:
			a.approvalFinding("authority_policy_stale_approval", state.approval, "configured approval has no matching source edge")
		case state.matches != state.approval.Cardinality:
			a.approvalFinding(
				"authority_policy_edge_cardinality_mismatch",
				state.approval,
				fmt.Sprintf("observed %d matching edges, want %d", state.matches, state.approval.Cardinality),
			)
		default:
			state.validated = true
		}
	}
}

func bypassFindingKind(edge ObservedEdge) string {
	if edge.Callee.Layer == AuthorityLayerGate {
		return "dupcode_gate_bypass"
	}
	if edge.ReferenceClass != refDirectCall && edge.ReferenceClass != refMethodExpression && edge.ReferenceClass != refDotImport {
		if edge.Callee.Layer == AuthorityLayerAdapter {
			return "dupcode_adapter_function_value"
		}
		return "dupcode_protected_function_value"
	}
	if edge.Callee.Layer == AuthorityLayerAdapter {
		return "dupcode_adapter_bypass"
	}
	return "dupcode_bypass"
}

func (a *canonicalAnalysis) approvalFinding(kind string, approval ApprovedCaller, detail string) {
	caller := approvalCallerIdentity(approval)
	message := fmt.Sprintf("approval %s -> %s [%s]: %s", callerIdentityString(caller), protectedSymbolString(approval.Callee), approval.ReferenceClass, detail)
	a.addFinding(approval.PackagePath, kind, message, token.Position{}, caller, approval.Callee, approval.ReferenceClass)
}

func (a *canonicalAnalysis) edgeFinding(kind string, edge ObservedEdge, detail string) {
	message := fmt.Sprintf(
		"line %d:%d: %s -> %s [%s]: %s",
		edge.Position.Line,
		edge.Position.Column,
		callerIdentityString(edge.Caller),
		protectedSymbolString(edge.Callee),
		edge.ReferenceClass,
		detail,
	)
	a.addFinding(edge.Path, kind, message, edge.Position, edge.Caller, edge.Callee, edge.ReferenceClass)
}

func countValidatedApprovals(states []resolvedApproval) int {
	count := 0
	for _, state := range states {
		if state.validated {
			count++
		}
	}
	return count
}

// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"fmt"
	"go/token"
	"go/types"
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

// resolveConfiguredApprovals validates every configured approval against the
// strict schema, marks malformed records as invalid without inferring any
// field, then resolves only the schema-valid records against the canonical
// package graph.
//
// The required order is fixed:
//
//	allocate one approval state per configured record
//	→ validate schema (no mutation, no inference)
//	→ mark malformed states invalid and emit typed schema findings
//	→ count exact schema-valid records
//	→ detect exact duplicates
//	→ resolve callers
//	→ resolve callees
//	→ mark valid
//
// Malformed approvals MUST NOT participate in:
//   - duplicate normalization / duplicate detection
//   - caller candidate lookup
//   - callee lookup
//   - observed-edge matching
//   - stale-approval checking
//   - cardinality checking
//
// A malformed schema finding MUST NOT cascade into caller_missing,
// callee_missing, stale_approval, or edge_cardinality_mismatch findings.
func (a *canonicalAnalysis) resolveConfiguredApprovals() {
	a.approvalStates = make([]resolvedApproval, len(a.config.approvals))
	schemaValid := make([]bool, len(a.config.approvals))
	counts := make(map[ApprovedCaller]int)

	// Phase 1: allocate, validate schema, count schema-valid records.
	for index, approval := range a.config.approvals {
		a.approvalStates[index] = resolvedApproval{approval: approval}
		issues := validateApprovalSchema(approval)
		if len(issues) > 0 {
			for _, issue := range issues {
				a.schemaApprovalFinding(issue, approval)
			}
			schemaValid[index] = false
			continue
		}
		schemaValid[index] = true
		counts[approval]++
	}

	// Phase 2: duplicate detection, caller/callee resolution for the
	// schema-valid records only.
	for index, approval := range a.config.approvals {
		if !schemaValid[index] {
			continue
		}
		state := &a.approvalStates[index]
		if counts[approval] > 1 {
			state.duplicate = true
			a.approvalFinding("authority_policy_duplicate_approval", approval, "approval appears more than once")
			continue
		}
		candidates := a.callerCandidates[approvalCallerIdentity(approval)]
		switch len(candidates) {
		case 0:
			a.approvalFinding("authority_policy_caller_missing", approval, "configured caller declaration not found")
			continue
		case 1:
			state.callerObject = candidates[0]
		default:
			a.approvalFinding("authority_policy_caller_ambiguous", approval, "configured caller resolves more than once")
			continue
		}
		calleeObject := a.objectByProtected[approval.Callee]
		if calleeObject == nil {
			a.approvalFinding("authority_policy_callee_missing", approval, "configured callee declaration did not resolve")
			continue
		}
		state.calleeObject = calleeObject
		state.valid = true
	}
}

func approvalCallerIdentity(approval ApprovedCaller) CallerIdentity {
	return CallerIdentity{
		PackagePath: approval.PackagePath,
		Function:    approval.Function,
		Receiver:    approval.Receiver,
		Kind:        approval.CallerKind,
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

// schemaApprovalFinding emits a typed schema finding without mutating or
// normalizing the supplied approval. The detail field is the message category
// from validateApprovalSchema so the emitted finding is deterministic.
func (a *canonicalAnalysis) schemaApprovalFinding(issue approvalSchemaIssue, approval ApprovedCaller) {
	caller := approvalCallerIdentity(approval)
	message := fmt.Sprintf(
		"approval %s -> %s: field=%s %s",
		callerIdentityString(caller),
		protectedSymbolString(approval.Callee),
		issue.Field,
		issue.Message,
	)
	a.addFinding(approval.PackagePath, issue.Kind, message, token.Position{}, caller, approval.Callee, approval.ReferenceClass)
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

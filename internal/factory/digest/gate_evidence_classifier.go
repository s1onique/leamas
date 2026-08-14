// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_classifier.go implements the
// pure EvaluateGateEvidenceBinding function. The classifier is
// total — every input combination produces a typed
// GateEvidenceBinding record. It is intentionally fail-closed:
// ambiguous input always produces authoritative_for_digest=false.
//
// The classifier never reads the filesystem, never spawns Git
// subprocesses, and never consults the clock. The caller MUST
// resolve all identities before invocation.
package digest

import "strings"

// EvaluateGateEvidenceBinding is the pure binding classifier.
// It performs no I/O, no Git subprocesses, and no time lookup.
// Identities MUST be resolved before invocation.
//
// Inputs:
//
//	gate           - the binding-relevant extract of the gate summary
//	digest         - the binding-relevant extract of the digest
//	                 authority already resolved from the working tree
//	sourceValidity - SourceAbsent (no file) / SourceInvalid
//	                 (file present but malformed/unreadable) /
//	                 SourceValid (parseable + normalizable).
//	                 The ACT explicitly distinguishes ABSENT from
//	                 INVALID; they are NOT collapsed.
//
// The function is total: every input combination produces a
// typed GateEvidenceBinding record. The returned struct is
// safe for direct use by renderers and tests.
func EvaluateGateEvidenceBinding(gate GateSummaryIdentity, digest DigestAuthority, sourceValidity GateSourceValidity) GateEvidenceBinding {
	result := GateEvidenceBinding{
		Status:                 BindingNotApplicable,
		SourceValidity:         sourceValidity,
		StateBinding:           StateUnbound,
		ScopeBinding:           ScopeNotApplicable,
		AuthoritativeForDigest: false,
		WarningCode:            GateBindingWarningCodeNotApplicable,
		TreeBinding:            StateUnbound,
	}

	// Stage 1: handle ABSENT, INVALID, and UNKNOWN sources
	// BEFORE any binding math. The ACT explicitly distinguishes
	// these:
	//
	//   missing summary    != present malformed summary
	//   unknown validity    != accepted validity
	//
	//   SourceAbsent  -> NOT_APPLICABLE
	//   SourceInvalid -> EVIDENCE_INVALID
	//   any other int  -> EVIDENCE_INVALID (fail closed)
	//
	// The default branch is critical: GateSourceValidity is
	// an int enum, and any future or accidental numeric value
	// MUST NOT silently reach the valid-evidence path. The
	// doctrine is "AMBIGUOUS_BINDING_FAILS_CLOSED=true";
	// unknown values are definitionally ambiguous.
	//
	// The renderer is responsible for the source_status
	// label (missing / invalid / read / decode / normalize).
	switch sourceValidity {
	case SourceAbsent:
		result.Status = BindingNotApplicable
		result.StateBinding = StateUnbound
		result.ScopeBinding = ScopeNotApplicable
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeNotApplicable
		return result
	case SourceInvalid:
		result.Status = BindingEvidenceInvalid
		result.StateBinding = StateUnbound
		result.ScopeBinding = ScopeUnbound
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeInvalidBinding
		return result
	case SourceValid:
		// proceed to the binding math below
	default:
		// Any unrecognized value is treated as INVALID
		// evidence. The classifier never silently promotes
		// unknown values to AUTHORITATIVE.
		result.Status = BindingEvidenceInvalid
		result.StateBinding = StateUnbound
		result.ScopeBinding = ScopeUnbound
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeInvalidBinding
		return result
	}

	// Stage 2: legacy shapes lack execution identity. We
	// classify them as LEGACY_UNBOUND without further
	// comparison; the historical verdict remains visible
	// as reported_overall_status.
	if !gate.HasExecutionIdentity {
		result.Status = BindingLegacyUnbound
		result.StateBinding = StateUnbound
		result.ScopeBinding = ScopeUnbound
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeLegacyUnbound
		return result
	}

	// Stage 3: dirty digest subjects cannot be bound by a
	// commit-only gate summary. We refuse before any state
	// comparison because HEAD identity does not identify
	// uncommitted changes.
	if digest.Dirty {
		result.Status = BindingDirtySubjectUnbound
		result.StateBinding = StateUnbound
		result.ScopeBinding = digestScopeBinding(gate, digest)
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeDirtySubjectUnbound
		return result
	}

	// Stage 4: state binding. Compare gate execution
	// identity to digest subject identity. Both must be
	// present and equal.
	stateBinding := compareStateBinding(gate, digest)
	result.StateBinding = stateBinding
	result.TreeBinding = compareTreeBinding(gate, digest)

	if stateBinding != StateMatch {
		result.Status = BindingStateMismatch
		result.ScopeBinding = StateBindingToScope(stateBinding, digestScopeBinding(gate, digest))
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeStateMismatch
		return result
	}

	// Stage 5: scope binding. Only meaningful when the
	// digest has a known ActID. Three terminal cases:
	//
	//   - both gate and digest expose identical scope: MATCH
	//   - both expose scope but they differ: MISMATCH
	//   - either side lacks scope: UNBOUND (NOT new
	//     authority, but not contradicted)
	scopeBinding := digestScopeBinding(gate, digest)
	result.ScopeBinding = scopeBinding

	switch scopeBinding {
	case ScopeMismatch:
		result.Status = BindingScopeMismatch
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeScopeMismatch
		return result
	case ScopeUnbound:
		// State matches but no scope can be proven.
		result.Status = BindingStateMatchScopeUnbound
		result.AuthoritativeForDigest = false
		result.WarningCode = GateBindingWarningCodeStateMatchScopeUnbound
		return result
	default:
		// Scope MATCH: state MATCH + scope MATCH
		// + digest authority resolved = AUTHORITATIVE.
		result.Status = BindingAuthoritative
		result.AuthoritativeForDigest = true
		result.WarningCode = GateBindingWarningCodeNone
		return result
	}
}

// compareStateBinding returns MATCH iff the gate summary's
// execution head OID equals the digest's resolved subject
// commit OID. Any other case is MISMATCH. The tree OID is
// recorded separately in TreeBinding for diagnostic purposes;
// the classifier never promotes same-tree evidence to authority.
func compareStateBinding(gate GateSummaryIdentity, digest DigestAuthority) StateBindingStatus {
	if gate.ExecutionHeadOID == "" || digest.SubjectCommitOID == "" {
		return StateMismatch
	}
	if strings.EqualFold(gate.ExecutionHeadOID, digest.SubjectCommitOID) {
		return StateMatch
	}
	return StateMismatch
}

// compareTreeBinding returns the tree-component verdict. It
// is recorded for diagnostic visibility only.
func compareTreeBinding(gate GateSummaryIdentity, digest DigestAuthority) StateBindingStatus {
	if gate.ExecutionTreeOID == "" || digest.SubjectTreeOID == "" {
		return StateUnbound
	}
	if strings.EqualFold(gate.ExecutionTreeOID, digest.SubjectTreeOID) {
		return StateMatch
	}
	return StateMismatch
}

// digestScopeBinding derives the scope verdict from the
// gate scope identity and the digest ActID. The contract
// is conservative:
//
//	both absent     -> NOT_APPLICABLE
//	one absent      -> UNBOUND
//	both present    -> MATCH iff EqualFold
//	                  -> MISMATCH otherwise
//
// CRITICAL: only gate.ScopeID can establish scope match.
// gate.ParentAct is recorded as provenance metadata but
// is NOT a fallback scope identifier. A gate summary that
// reports itself as a child of ACT-A (parent_act=ACT-A,
// scope_id="") does not prove that the gate was executed
// FOR ACT-A; it only proves the gate's parent is ACT-A.
// Substituting parent_act for scope_id would silently
// elevate provenance metadata into authority claims,
// which is exactly the false-positive path the ACT exists
// to prevent.
func digestScopeBinding(gate GateSummaryIdentity, digest DigestAuthority) ScopeBindingStatus {
	// Only gate.ScopeID is used for scope comparison. ParentAct
	// is intentionally NOT consulted here. See the comment
	// above for the rationale.
	gateScope := strings.TrimSpace(gate.ScopeID)
	digestScope := strings.TrimSpace(digest.ActID)

	if gateScope == "" && digestScope == "" {
		return ScopeNotApplicable
	}
	if gateScope == "" || digestScope == "" {
		return ScopeUnbound
	}
	if strings.EqualFold(gateScope, digestScope) {
		return ScopeMatch
	}
	return ScopeMismatch
}

// StateBindingToScope composes the rendered scope binding
// from the state- and scope-binding dimensions. When the
// state already mismatched, the scope verdict is reported
// only if it independently mismatches; otherwise the
// overall binding is MISMATCH and the scope is left at
// UNBOUND.
func StateBindingToScope(state StateBindingStatus, scope ScopeBindingStatus) ScopeBindingStatus {
	if state == StateMismatch {
		if scope == ScopeMismatch {
			return ScopeMismatch
		}
		return ScopeUnbound
	}
	return scope
}

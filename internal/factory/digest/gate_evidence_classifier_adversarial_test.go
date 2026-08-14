// SPDX-License-Identifier: Apache-2.0

// Package digest: gate_evidence_classifier_adversarial_test.go
// contains the adversarial input rows for the gate-evidence
// binding classifier. These rows verify that the classifier
// fails closed on:
//   - SourceAbsent vs SourceInvalid distinction (rows 13-16)
//   - Unknown GateSourceValidity enum values (row 17)
//   - parent_act not substituting for scope_id (row 18)
//
// The split keeps the core state/scope matrix in
// gate_evidence_classifier_test.go under the 400-line LLM
// friendliness limit, with the adversarial coverage here.
package digest

import "testing"

// TestEvaluateGateEvidenceBinding_AdversarialMatrix covers the
// adversarial inputs that the classifier must reject or
// classify as STATE_MATCH_SCOPE_UNBOUND rather than
// AUTHORITATIVE.
func TestEvaluateGateEvidenceBinding_AdversarialMatrix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		gate           GateSummaryIdentity
		digest         DigestAuthority
		sourceValidity GateSourceValidity
		wantStatus     GateBindingStatus
		wantState      StateBindingStatus
		wantScope      ScopeBindingStatus
		wantAuth       bool
		wantWarn       string
	}{
		// Row 13: ABSENT source (no file at all). Distinct
		// from INVALID: there is no evidence to bind.
		{
			name:           "row13_source_absent",
			sourceValidity: SourceAbsent,
			wantStatus:     BindingNotApplicable,
			wantState:      StateUnbound,
			wantScope:      ScopeNotApplicable,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeNotApplicable,
		},
		// Row 14: INVALID source (file present but
		// malformed/unreadable). Distinct from ABSENT: there
		// IS evidence, but it is corrupt. The ACT requires
		// these to remain distinct.
		{
			name:           "row14_source_invalid",
			sourceValidity: SourceInvalid,
			wantStatus:     BindingEvidenceInvalid,
			wantState:      StateUnbound,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeInvalidBinding,
		},
		// Row 15: ABSENT source with otherwise-bindable
		// digest authority. Authority MUST remain false.
		{
			name: "row15_source_absent_with_authority",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			sourceValidity: SourceAbsent,
			wantStatus:     BindingNotApplicable,
			wantState:      StateUnbound,
			wantScope:      ScopeNotApplicable,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeNotApplicable,
		},
		// Row 16: INVALID source with otherwise-bindable
		// digest authority. Authority MUST remain false and
		// binding MUST be EVIDENCE_INVALID, not NOT_APPLICABLE.
		{
			name: "row16_source_invalid_with_authority",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			sourceValidity: SourceInvalid,
			wantStatus:     BindingEvidenceInvalid,
			wantState:      StateUnbound,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeInvalidBinding,
		},
		// Row 17: UNKNOWN source validity (numeric value
		// outside the three accepted enum values). The
		// classifier MUST fail closed: any future or
		// accidental numeric value cannot silently reach
		// the valid-evidence path. AMBIGUOUS_BINDING_FAILS_CLOSED.
		{
			name:           "row17_unknown_source_validity",
			sourceValidity: GateSourceValidity(99),
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			// The gate is irrelevant: unknown validity must
			// short-circuit before any binding math.
			wantStatus: BindingEvidenceInvalid,
			wantState:  StateUnbound,
			wantScope:  ScopeUnbound,
			wantAuth:   false,
			wantWarn:   GateBindingWarningCodeInvalidBinding,
		},
		// Row 18: parent_act ONLY (no scope_id) with the
		// digest's ActID matching the parent. This is the
		// exact false-positive path CORRECTION02 closes:
		// parent_act is provenance metadata, NOT a scope
		// authority claim. The classifier must report
		// STATE_MATCH_SCOPE_UNBOUND, not AUTHORITATIVE.
		{
			name: "row18_parent_act_only_no_scope_match",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              "", // no scope_id
				ParentAct:            testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true, // parent_act counts as scope identity presence
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA, // matches parent_act
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMatchScopeUnbound,
			wantState:      StateMatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMatchScopeUnbound,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateGateEvidenceBinding(tc.gate, tc.digest, tc.sourceValidity)
			if got.SourceValidity != tc.sourceValidity {
				t.Errorf("SourceValidity: got %q want %q", got.SourceValidity, tc.sourceValidity)
			}
			if got.Status != tc.wantStatus {
				t.Errorf("Status: got %q want %q", got.Status, tc.wantStatus)
			}
			if got.StateBinding != tc.wantState {
				t.Errorf("StateBinding: got %q want %q", got.StateBinding, tc.wantState)
			}
			if got.ScopeBinding != tc.wantScope {
				t.Errorf("ScopeBinding: got %q want %q", got.ScopeBinding, tc.wantScope)
			}
			if got.AuthoritativeForDigest != tc.wantAuth {
				t.Errorf("AuthoritativeForDigest: got %v want %v", got.AuthoritativeForDigest, tc.wantAuth)
			}
			if got.WarningCode != tc.wantWarn {
				t.Errorf("WarningCode: got %q want %q", got.WarningCode, tc.wantWarn)
			}
		})
	}
}

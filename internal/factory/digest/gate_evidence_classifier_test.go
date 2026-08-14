// SPDX-License-Identifier: Apache-2.0

package digest

import "testing"

// OID literals used across the matrix. They are well-formed
// full 40-char hex strings so the EqualFold branches exercise
// the canonical comparison path.
const (
	testOIDX = "0123456789abcdef0123456789abcdef01234567"
	testOIDY = "89abcdef0123456789abcdef0123456789abcdef01"
	testOIDT = "fedcba9876543210fedcba9876543210fedcba98"
	testActA = "ACT-LEAMAS-DEMO-01"
	testActB = "ACT-LEAMAS-DEMO-02"
)

// TestEvaluateGateEvidenceBinding_Matrix walks the required
// classification matrix from ACT-LEAMAS-DIGEST-GATE-EVIDENCE-
// AUTHORITY-BINDING01 sections 28 and 32-35. Every row asserts
// the four binding dimensions plus the warning code.
func TestEvaluateGateEvidenceBinding_Matrix(t *testing.T) {
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
		// Row 1: state MATCH, scope MATCH => AUTHORITATIVE.
		{
			name: "row1_state_match_scope_match",
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
			sourceValidity: SourceValid,
			wantStatus:     BindingAuthoritative,
			wantState:      StateMatch,
			wantScope:      ScopeMatch,
			wantAuth:       true,
			wantWarn:       GateBindingWarningCodeNone,
		},
		// Row 2: state MISMATCH, scope MATCH => STATE_MISMATCH.
		{
			name: "row2_state_mismatch",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDY,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMismatch,
			wantState:      StateMismatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMismatch,
		},
		// Row 3: state MATCH, scope MISMATCH => SCOPE_MISMATCH.
		{
			name: "row3_scope_mismatch",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActB,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingScopeMismatch,
			wantState:      StateMatch,
			wantScope:      ScopeMismatch,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeScopeMismatch,
		},
		// Row 4: state MATCH, scope UNBOUND (gate scope absent).
		{
			name: "row4_state_match_scope_unbound",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				HasExecutionIdentity: true,
				HasScopeIdentity:     false,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMatchScopeUnbound,
			wantState:      StateMatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMatchScopeUnbound,
		},
		// Row 5: absent summary => NOT_APPLICABLE.
		{
			name:           "row5_legacy_unbound",
			sourceValidity: SourceAbsent,
			wantStatus:     BindingNotApplicable,
			wantState:      StateUnbound,
			wantScope:      ScopeNotApplicable,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeNotApplicable,
		},
		// Row 6: dirty digest, commit-only gate => DIRTY_SUBJECT_UNBOUND.
		{
			name: "row6_dirty_subject_unbound",
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
				Dirty:            true,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingDirtySubjectUnbound,
			wantState:      StateUnbound,
			wantScope:      ScopeMatch,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeDirtySubjectUnbound,
		},
		// Row 7: malformed summary, summaryValid=true but
		// no execution identity => LEGACY_UNBOUND.
		{
			name: "row7_legacy_v1_summary",
			gate: GateSummaryIdentity{
				SchemaVersion:        1,
				OverallStatus:        "pass",
				HasExecutionIdentity: false,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingLegacyUnbound,
			wantState:      StateUnbound,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeLegacyUnbound,
		},
		// Row 8: same-tree, different commit => STATE_MISMATCH
		// (the conservative contract).
		{
			name: "row8_same_tree_different_commit",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ExecutionTreeOID:     testOIDT,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: testOIDY,
				SubjectTreeOID:   testOIDT,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMismatch,
			wantState:      StateMismatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMismatch,
		},

		// Row 9: historical-range, gate head matches the
		// digest's right endpoint B, even though repo HEAD is C.
		{
			name: "row9_historical_range_match",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDY,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				Mode:             "range",
				Range:            "A.." + testOIDY,
				SubjectCommitOID: testOIDY,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingAuthoritative,
			wantState:      StateMatch,
			wantScope:      ScopeMatch,
			wantAuth:       true,
			wantWarn:       GateBindingWarningCodeNone,
		},
		// Row 10: historical-range, gate head is ambient HEAD=C
		// but digest subject is B => not authoritative.
		{
			name: "row10_historical_range_ambient_head_mismatch",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDT,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				Mode:             "range",
				Range:            "A..B",
				SubjectCommitOID: testOIDX,
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMismatch,
			wantState:      StateMismatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMismatch,
		},
		// Row 11: explicit-range, no ACT-ID resolved, gate
		// matches state => STATE_MATCH_SCOPE_UNBOUND.
		{
			name: "row11_explicit_range_no_act",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				Mode:             "range",
				Range:            "A..B",
				SubjectCommitOID: testOIDX,
				ActID:            "",
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMatchScopeUnbound,
			wantState:      StateMatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMatchScopeUnbound,
		},
		// Row 12: malformed digest subject (no SubjectCommitOID)
		// cannot be proved authoritative.
		{
			name: "row12_digest_subject_unresolved",
			gate: GateSummaryIdentity{
				SchemaVersion:        2,
				ExecutionHeadOID:     testOIDX,
				ScopeID:              testActA,
				HasExecutionIdentity: true,
				HasScopeIdentity:     true,
			},
			digest: DigestAuthority{
				SubjectCommitOID: "",
				ActID:            testActA,
			},
			sourceValidity: SourceValid,
			wantStatus:     BindingStateMismatch,
			wantState:      StateMismatch,
			wantScope:      ScopeUnbound,
			wantAuth:       false,
			wantWarn:       GateBindingWarningCodeStateMismatch,
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

// TestEvaluateGateEvidenceBinding_DoesNotConsultClock proves
// the classifier is deterministic across varying generated_at
// values. Wall-clock time MUST NOT establish authority.
func TestEvaluateGateEvidenceBinding_DoesNotConsultClock(t *testing.T) {
	gate := GateSummaryIdentity{
		SchemaVersion:        2,
		ExecutionHeadOID:     testOIDX,
		ScopeID:              testActA,
		HasExecutionIdentity: true,
		HasScopeIdentity:     true,
	}
	digest := DigestAuthority{
		SubjectCommitOID: testOIDX,
		ActID:            testActA,
	}
	for _, ts := range []string{
		"2026-07-18T18:29:21Z",
		"2026-08-14T12:01:39Z",
		"2099-01-01T00:00:00Z",
	} {
		gate.GeneratedAt = ts
		got := EvaluateGateEvidenceBinding(gate, digest, SourceValid)
		if got.Status != BindingAuthoritative {
			t.Errorf("varying generated_at must not change status; got %q for %q", got.Status, ts)
		}
		if !got.AuthoritativeForDigest {
			t.Errorf("varying generated_at must not flip authority; got false for %q", ts)
		}
	}
}

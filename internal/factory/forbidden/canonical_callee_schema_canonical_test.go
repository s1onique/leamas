// SPDX-License-Identifier: Apache-2.0

package forbidden

import "testing"

// validCalleeSchemaBaseline returns a caller-side schema-valid approval
// baseline paired with a schema-valid callee. The fixture tests below
// mutate one callee field at a time so the resulting schema finding is
// fully attributable to that field.
func validCalleeSchemaBaseline() ApprovedCaller {
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

// TestCanonicalCalleeSchemaFailuresAreTyped runs every malformed callee
// fixture case through the real canonical analysis and asserts:
//
//   - the typed callee-schema finding is emitted
//   - exactly zero validated approvals
//   - the malformed-schema finding does NOT cascade into caller_missing,
//     callee_missing, stale_approval, edge_cardinality_mismatch, or
//     reference_class_mismatch
func TestCanonicalCalleeSchemaFailuresAreTyped(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*ApprovedCaller)
		wantKind string
	}{
		{
			name: "missing callee layer",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Layer = ""
			},
			wantKind: "authority_policy_approval_callee_layer_invalid",
		},
		{
			name: "missing callee package",
			mutate: func(a *ApprovedCaller) {
				a.Callee.PackagePath = ""
			},
			wantKind: "authority_policy_approval_callee_identity_invalid",
		},
		{
			name: "missing callee name",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Name = ""
			},
			wantKind: "authority_policy_approval_callee_identity_invalid",
		},
		{
			name: "missing callee kind",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Kind = ""
			},
			wantKind: "authority_policy_approval_callee_kind_invalid",
		},
		{
			name: "method without callee receiver",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Kind = ProtectedMethod
				a.Callee.Receiver = ""
			},
			wantKind: "authority_policy_approval_callee_receiver_invalid",
		},
		{
			name: "package function with callee receiver",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Kind = ProtectedPackageFunction
				a.Callee.Receiver = "DupcodeRunner"
			},
			wantKind: "authority_policy_approval_callee_receiver_invalid",
		},
		{
			name: "wildcard callee identity",
			mutate: func(a *ApprovedCaller) {
				a.Callee.Name = "*"
			},
			wantKind: "authority_policy_approval_callee_identity_invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newCanonicalFixture(t)
			fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
			fixture.write("caller/caller.go", `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
			protected := fixtureSymbol(
				AuthorityLayerRaw,
				fixture.packagePath("protected"),
				"Cap",
				ProtectedPackageFunction,
				"",
			)
			approval := validCalleeSchemaBaseline()
			approval.PackagePath = fixture.packagePath("caller")
			tc.mutate(&approval)
			// The baseline already matches the fixture package paths;
			// the case deliberately mutated exactly one callee field.
			_ = fixture

			result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})
			requireFindingKind(t, result.Findings, tc.wantKind)
			if result.Stats.ValidatedApprovals != 0 {
				t.Errorf("validated approvals = %d, want 0 (malformed schema)", result.Stats.ValidatedApprovals)
			}
			rejectFindingKind(t, result.Findings, "authority_policy_caller_missing")
			rejectFindingKind(t, result.Findings, "authority_policy_callee_missing")
			rejectFindingKind(t, result.Findings, "authority_policy_stale_approval")
			rejectFindingKind(t, result.Findings, "authority_policy_edge_cardinality_mismatch")
			rejectFindingKind(t, result.Findings, "authority_policy_reference_class_mismatch")
		})
	}
}

// TestCanonicalWellFormedMissingCalleeRetainsSemantic distinguishes the
// schema-failure path from the well-formed-but-nonexistent-callee path.
// A schema-valid approval whose callee identity is well-formed but does not
// match any configured/resolved protected declaration must continue
// producing authority_policy_callee_missing.
func TestCanonicalWellFormedMissingCalleeRetainsSemantic(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", `package caller
func Allowed() {}
`)
	protected := fixtureSymbol(
		AuthorityLayerRaw,
		fixture.packagePath("protected"),
		"Cap",
		ProtectedPackageFunction,
		"",
	)
	approval := validCalleeSchemaBaseline()
	approval.PackagePath = fixture.packagePath("caller")
	// Schema-valid callee pointing to a name that does not exist in the
	// configured protected-symbol list.
	approval.Callee.Name = "DoesNotExist"
	approval.Callee.PackagePath = fixture.packagePath("protected")
	approval.Callee.Layer = AuthorityLayerRaw
	approval.Callee.Kind = ProtectedPackageFunction

	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})
	requireFindingKind(t, result.Findings, "authority_policy_callee_missing")
	rejectFindingKind(t, result.Findings, "authority_policy_approval_callee_layer_invalid")
	rejectFindingKind(t, result.Findings, "authority_policy_approval_callee_identity_invalid")
	rejectFindingKind(t, result.Findings, "authority_policy_approval_callee_kind_invalid")
	rejectFindingKind(t, result.Findings, "authority_policy_approval_callee_receiver_invalid")
	if result.Stats.ValidatedApprovals != 0 {
		t.Errorf("validated approvals = %d, want 0", result.Stats.ValidatedApprovals)
	}
}

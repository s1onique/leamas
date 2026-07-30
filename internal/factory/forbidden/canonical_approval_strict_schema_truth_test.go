// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// TestCanonicalApprovalSchemaFailuresAreTyped runs every malformed-fixture
// case through the actual fixture analysis and asserts:
//
//   - the typed schema finding is emitted
//   - exactly zero validated approvals
//   - the malformed-schema finding does NOT cascade into caller_missing,
//     callee_missing, stale_approval, or edge_cardinality_mismatch
//   - a protected source use may still produce its ordinary bypass finding
//     because no valid approval exists; that is acceptable.
func TestCanonicalApprovalSchemaFailuresAreTyped(t *testing.T) {
	cases := []struct {
		name            string
		source          string
		approval        ApprovedCaller
		wantKind        string
		wantCascadeFree bool
	}{
		{
			name: "missing CallerKind",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "",
				CallerKind:     "",
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDirectCall,
				Cardinality:    1,
			},
			wantKind: "authority_policy_approval_caller_kind_invalid",
		},
		{
			name: "missing ReferenceClass",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath: "example.test/policy/caller",
				Function:    "Allowed",
				Receiver:    "",
				CallerKind:  CallerKindPackageFunction,
				Callee:      protectedSymbolFixture(),
				Cardinality: 1,
			},
			wantKind: "authority_policy_approval_reference_class_invalid",
		},
		{
			name: "DOT_IMPORT approval",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "",
				CallerKind:     CallerKindPackageFunction,
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDotImport,
				Cardinality:    1,
			},
			wantKind: "authority_policy_approval_reference_class_invalid",
		},
		{
			name: "zero Cardinality",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "",
				CallerKind:     CallerKindPackageFunction,
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDirectCall,
				Cardinality:    0,
			},
			wantKind: "authority_policy_approval_cardinality_invalid",
		},
		{
			name: "method without Receiver",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "",
				CallerKind:     CallerKindMethod,
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDirectCall,
				Cardinality:    1,
			},
			wantKind: "authority_policy_approval_receiver_invalid",
		},
		{
			name: "package function with Receiver",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "SomeReceiver",
				CallerKind:     CallerKindPackageFunction,
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDirectCall,
				Cardinality:    1,
			},
			wantKind: "authority_policy_approval_receiver_invalid",
		},
		{
			name: "wildcard caller identity",
			source: `package caller
func Existing() {}
`,
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "*",
				Receiver:       "",
				CallerKind:     CallerKindPackageFunction,
				Callee:         protectedSymbolFixture(),
				ReferenceClass: refDirectCall,
				Cardinality:    1,
			},
			wantKind: "authority_policy_approval_identity_invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newCanonicalFixture(t)
			fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
			fixture.write("caller/caller.go", tc.source)
			protected := fixtureSymbol(
				AuthorityLayerRaw,
				fixture.packagePath("protected"),
				"Cap",
				ProtectedPackageFunction,
				"",
			)
			approval := tc.approval
			approval.PackagePath = fixture.packagePath("caller")
			approval.Callee = protected

			result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})
			requireFindingKind(t, result.Findings, tc.wantKind)
			if result.Stats.ValidatedApprovals != 0 {
				t.Errorf("validated approvals = %d, want 0 (malformed schema)", result.Stats.ValidatedApprovals)
			}
			rejectFindingKind(t, result.Findings, "authority_policy_caller_missing")
			rejectFindingKind(t, result.Findings, "authority_policy_callee_missing")
			rejectFindingKind(t, result.Findings, "authority_policy_stale_approval")
			rejectFindingKind(t, result.Findings, "authority_policy_edge_cardinality_mismatch")
		})
	}
}

// TestCanonicalApprovalDuplicateAndEdgeBehaviorArePreserved locks the
// existing duplicate and edge-behavior semantics so the strict-schema
// migration does not regress them:
//
//   - two identical valid explicit approvals produce
//     authority_policy_duplicate_approval.
//   - one valid approval with a wrong reference class for the observed
//     edge produces authority_policy_reference_class_mismatch.
//   - one valid approval with the wrong cardinality produces
//     authority_policy_edge_cardinality_mismatch.
//   - one valid stale approval produces authority_policy_stale_approval.
func TestCanonicalApprovalDuplicateAndEdgeBehaviorArePreserved(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		approvals func(string, ProtectedSymbol) []ApprovedCaller
		wantKind  string
	}{
		{
			name: "two identical valid explicit approvals",
			source: `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				approval := fixtureApproval(pkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
				return []ApprovedCaller{approval, approval}
			},
			wantKind: "authority_policy_duplicate_approval",
		},
		{
			name: "wrong reference class for observed edge",
			source: `package caller
import p "example.test/policy/protected"
func Capture() { _ = p.Cap }
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "Capture", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_reference_class_mismatch",
		},
		{
			name: "wrong cardinality",
			source: `package caller
import p "example.test/policy/protected"
func Twice() { p.Cap(); p.Cap() }
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "Twice", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_edge_cardinality_mismatch",
		},
		{
			name: "valid stale approval",
			source: `package caller
func Idle() {}
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "Idle", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_stale_approval",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newCanonicalFixture(t)
			fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
			fixture.write("caller/caller.go", tc.source)
			protected := fixtureSymbol(
				AuthorityLayerRaw,
				fixture.packagePath("protected"),
				"Cap",
				ProtectedPackageFunction,
				"",
			)
			callerPkg := fixture.packagePath("caller")
			result := fixture.run([]ProtectedSymbol{protected}, tc.approvals(callerPkg, protected))
			requireFindingKind(t, result.Findings, tc.wantKind)
		})
	}
}

// protectedSymbolFixture returns the canonical protected-symbol shape used by
// TestCanonicalApprovalSchemaFailuresAreTyped so each case can construct an
// approval with the right callee identity in a single literal.
func protectedSymbolFixture() ProtectedSymbol {
	return ProtectedSymbol{
		Layer:       AuthorityLayerRaw,
		PackagePath: "example.test/policy/protected",
		Name:        "Cap",
		Kind:        ProtectedPackageFunction,
	}
}

// guard against accidental drift between the schema validator and the
// fixture harness: every approval used by the schema-failure cases must be
// rejected with the same issue list when run through validateApprovalSchema
// directly. This locks the unit tests, fixture tests, and runtime analysis
// to a single source of truth.
func TestSchemaFailuresMatchValidatorIssues(t *testing.T) {
	cases := []struct {
		name     string
		approval ApprovedCaller
		want     []approvalSchemaIssue
	}{
		{
			name: "missing CallerKind",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				CallerKind:     "",
				ReferenceClass: refDirectCall,
				Cardinality:    1,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_caller_kind_invalid", Field: "CallerKind", Message: "caller_kind_empty",
			}},
		},
		{
			name: "missing ReferenceClass",
			approval: ApprovedCaller{
				PackagePath: "example.test/policy/caller",
				Function:    "Allowed",
				CallerKind:  CallerKindPackageFunction,
				Cardinality: 1,
				Callee:      protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_reference_class_invalid", Field: "ReferenceClass", Message: "reference_class_empty",
			}},
		},
		{
			name: "DOT_IMPORT approval",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				CallerKind:     CallerKindPackageFunction,
				ReferenceClass: refDotImport,
				Cardinality:    1,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_reference_class_invalid", Field: "ReferenceClass", Message: "reference_class_dot_import",
			}},
		},
		{
			name: "zero Cardinality",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				CallerKind:     CallerKindPackageFunction,
				ReferenceClass: refDirectCall,
				Cardinality:    0,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_cardinality_invalid", Field: "Cardinality", Message: "cardinality_zero",
			}},
		},
		{
			name: "method without Receiver",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				CallerKind:     CallerKindMethod,
				ReferenceClass: refDirectCall,
				Cardinality:    1,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_receiver_invalid", Field: "Receiver", Message: "receiver_required_for_method",
			}},
		},
		{
			name: "package function with Receiver",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "Allowed",
				Receiver:       "SomeReceiver",
				CallerKind:     CallerKindPackageFunction,
				ReferenceClass: refDirectCall,
				Cardinality:    1,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_receiver_invalid", Field: "Receiver", Message: "receiver_must_be_empty",
			}},
		},
		{
			name: "wildcard caller identity",
			approval: ApprovedCaller{
				PackagePath:    "example.test/policy/caller",
				Function:       "*",
				CallerKind:     CallerKindPackageFunction,
				ReferenceClass: refDirectCall,
				Cardinality:    1,
				Callee:         protectedSymbolFixture(),
			},
			want: []approvalSchemaIssue{{
				Kind: "authority_policy_approval_identity_invalid", Field: "Function", Message: "function_wildcard",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateApprovalSchema(tc.approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema issues = %+v, want %+v", got, tc.want)
			}
		})
	}
}

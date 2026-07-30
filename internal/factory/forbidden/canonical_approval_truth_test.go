// SPDX-License-Identifier: Apache-2.0

package forbidden

import "testing"

func approvalFixture(t *testing.T, callerSource string) (*canonicalFixture, ProtectedSymbol, string) {
	t.Helper()
	fixture := newCanonicalFixture(t)
	fixture.write("protected/protected.go", `package protected
func Cap() {}
`)
	fixture.write("caller/caller.go", callerSource)
	protected := fixtureSymbol(
		AuthorityLayerRaw,
		fixture.packagePath("protected"),
		"Cap",
		ProtectedPackageFunction,
		"",
	)
	return fixture, protected, fixture.packagePath("caller")
}

func TestCanonicalApprovalBidirectionalTruth(t *testing.T) {
	fixture, protected, callerPkg := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func Allowed() { p.Cap() }
`)
	approval := fixtureApproval(callerPkg, "Allowed", "", CallerKindPackageFunction, protected, refDirectCall)
	result := fixture.run([]ProtectedSymbol{protected}, []ApprovedCaller{approval})

	for _, kind := range []string{
		"dupcode_bypass",
		"authority_policy_stale_approval",
		"authority_policy_duplicate_approval",
		"authority_policy_caller_missing",
		"authority_policy_callee_missing",
		"authority_policy_reference_class_mismatch",
		"authority_policy_edge_cardinality_mismatch",
	} {
		rejectFindingKind(t, result.Findings, kind)
	}
	if result.Stats.ConfiguredApprovals != 1 ||
		result.Stats.ObservedEdges != 1 ||
		result.Stats.ValidatedApprovals != 1 {
		t.Fatalf("approval stats = %+v, want configured/observed/validated = 1/1/1", result.Stats)
	}
}

func TestCanonicalApprovalFailures(t *testing.T) {
	cases := []struct {
		name      string
		source    string
		approvals func(string, ProtectedSymbol) []ApprovedCaller
		wantKind  string
	}{
		{
			name: "stale approval",
			source: `package caller
func Idle() {}
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "Idle", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_stale_approval",
		},
		{
			name: "duplicate approval",
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
			name: "missing caller",
			source: `package caller
func Existing() {}
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "Missing", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_caller_missing",
		},
		{
			name: "wrong reference class",
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
			name: "edge cardinality",
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
			name: "wildcard caller identity",
			source: `package caller
func Existing() {}
`,
			approvals: func(pkg string, protected ProtectedSymbol) []ApprovedCaller {
				return []ApprovedCaller{fixtureApproval(pkg, "*", "", CallerKindPackageFunction, protected, refDirectCall)}
			},
			wantKind: "authority_policy_approval_identity_invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture, protected, callerPkg := approvalFixture(t, tc.source)
			result := fixture.run([]ProtectedSymbol{protected}, tc.approvals(callerPkg, protected))
			requireFindingKind(t, result.Findings, tc.wantKind)
		})
	}
}

func TestCanonicalApprovalCalleeMissing(t *testing.T) {
	fixture := newCanonicalFixture(t)
	fixture.write("caller/caller.go", "package caller\nfunc Existing() {}\n")
	missing := fixtureSymbol(AuthorityLayerRaw, fixture.packagePath("protected"), "Missing", ProtectedPackageFunction, "")
	approval := fixtureApproval(
		fixture.packagePath("caller"),
		"Existing",
		"",
		CallerKindPackageFunction,
		missing,
		refDirectCall,
	)
	result := fixture.run([]ProtectedSymbol{missing}, []ApprovedCaller{approval})
	requireFindingKind(t, result.Findings, "authority_policy_symbol_missing")
	requireFindingKind(t, result.Findings, "authority_policy_callee_missing")
}

func TestCanonicalApprovalDistinctCallersValidateIndependently(t *testing.T) {
	fixture, protected, callerPkg := approvalFixture(t, `package caller
import p "example.test/policy/protected"
func First() { p.Cap() }
func Second() { p.Cap() }
`)
	approvals := []ApprovedCaller{
		fixtureApproval(callerPkg, "First", "", CallerKindPackageFunction, protected, refDirectCall),
		fixtureApproval(callerPkg, "Second", "", CallerKindPackageFunction, protected, refDirectCall),
	}
	result := fixture.run([]ProtectedSymbol{protected}, approvals)
	if result.Stats.ObservedEdges != 2 || result.Stats.ValidatedApprovals != 2 {
		t.Fatalf("stats = %+v, want two observed and two validated", result.Stats)
	}
	rejectFindingKind(t, result.Findings, "authority_policy_stale_approval")
}

// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// validCalleeBaseline returns a strictly schema-valid callee used as the
// baseline by callee-schema failure cases. Every case mutates exactly one
// callee field so the resulting issue is fully attributable to that field.
func validCalleeBaseline() ProtectedSymbol {
	return ProtectedSymbol{
		Layer:       AuthorityLayerRaw,
		PackagePath: "example.test/policy/protected",
		Name:        "Cap",
		Kind:        ProtectedPackageFunction,
	}
}

// validCalleeBaselineApproval wraps validCalleeBaseline in a strictly
// schema-valid approval so callee-side failure tests can mutate one callee
// field at a time without producing caller-side noise.
func validCalleeBaselineApproval() ApprovedCaller {
	return ApprovedCaller{
		PackagePath:    "example.test/policy/caller",
		Function:       "Allowed",
		Receiver:       "",
		CallerKind:     CallerKindPackageFunction,
		ReferenceClass: refDirectCall,
		Cardinality:    1,
		Callee:         validCalleeBaseline(),
	}
}

// TestStrictSchemaCalleeValidCoverage exercises one positive case per
// required callee shape so the validator's callee allow-list is explicitly
// verified.
func TestStrictSchemaCalleeValidCoverage(t *testing.T) {
	methodCallee := validCalleeBaseline()
	methodCallee.Kind = ProtectedMethod
	methodCallee.Receiver = "DupcodeRunner"

	gateCallee := validCalleeBaseline()
	gateCallee.Layer = AuthorityLayerGate
	gateCallee.PackagePath = "example.test/policy/gate"

	adapterMethod := validCalleeBaseline()
	adapterMethod.Layer = AuthorityLayerAdapter

	variableCallee := validCalleeBaseline()
	variableCallee.Kind = ProtectedPackageVariable

	cases := []struct {
		name   string
		callee ProtectedSymbol
	}{
		{"valid raw package function", validCalleeBaseline()},
		{"valid raw method", methodCallee},
		{"valid gate package function", gateCallee},
		{"valid adapter package function", adapterMethod},
		{"valid raw package variable", variableCallee},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validCalleeBaselineApproval()
			approval.Callee = tc.callee
			if issues := validateApprovalSchema(approval); len(issues) != 0 {
				t.Errorf("approval returned issues %+v, want none", issues)
			}
		})
	}
}

// TestStrictSchemaCalleeLayerFailures exercises Callee.Layer validation:
// empty, unknown, and case/whitespace variants are rejected.
func TestStrictSchemaCalleeLayerFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProtectedSymbol)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty Callee.Layer",
			mutate: func(s *ProtectedSymbol) { s.Layer = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_layer_invalid",
				Field:   "Callee.Layer",
				Message: "callee_layer_empty",
			}},
		},
		{
			name:   "unknown Callee.Layer",
			mutate: func(s *ProtectedSymbol) { s.Layer = AuthorityLayer("mystery_layer") },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_layer_invalid",
				Field:   "Callee.Layer",
				Message: "callee_layer_unknown",
			}},
		},
		{
			name:   "case variant Callee.Layer",
			mutate: func(s *ProtectedSymbol) { s.Layer = AuthorityLayer("RAW_DUPCODE") },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_layer_invalid",
				Field:   "Callee.Layer",
				Message: "callee_layer_unknown",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validCalleeBaselineApproval()
			tc.mutate(&approval.Callee)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCalleePackagePathFailures exercises Callee.PackagePath:
// empty and wildcard are rejected.
func TestStrictSchemaCalleePackagePathFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProtectedSymbol)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty Callee.PackagePath",
			mutate: func(s *ProtectedSymbol) { s.PackagePath = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_identity_invalid",
				Field:   "Callee.PackagePath",
				Message: "callee_package_path_empty",
			}},
		},
		{
			name:   "wildcard Callee.PackagePath",
			mutate: func(s *ProtectedSymbol) { s.PackagePath = "example.*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_identity_invalid",
				Field:   "Callee.PackagePath",
				Message: "callee_package_path_wildcard",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validCalleeBaselineApproval()
			tc.mutate(&approval.Callee)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCalleeNameFailures exercises Callee.Name: empty,
// wildcard, and source-coordinate markers are rejected.
func TestStrictSchemaCalleeNameFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProtectedSymbol)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty Callee.Name",
			mutate: func(s *ProtectedSymbol) { s.Name = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_identity_invalid",
				Field:   "Callee.Name",
				Message: "callee_name_empty",
			}},
		},
		{
			name:   "wildcard Callee.Name",
			mutate: func(s *ProtectedSymbol) { s.Name = "*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_identity_invalid",
				Field:   "Callee.Name",
				Message: "callee_name_wildcard",
			}},
		},
		{
			name:   "source-coordinate Callee.Name",
			mutate: func(s *ProtectedSymbol) { s.Name = "Cap@10:1" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_identity_invalid",
				Field:   "Callee.Name",
				Message: "callee_name_source_coordinate",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validCalleeBaselineApproval()
			tc.mutate(&approval.Callee)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCalleeKindFailures exercises Callee.Kind: empty and
// unknown values are rejected.
func TestStrictSchemaCalleeKindFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ProtectedSymbol)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty Callee.Kind",
			mutate: func(s *ProtectedSymbol) { s.Kind = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_kind_invalid",
				Field:   "Callee.Kind",
				Message: "callee_kind_empty",
			}},
		},
		{
			name:   "unknown Callee.Kind",
			mutate: func(s *ProtectedSymbol) { s.Kind = ProtectedSymbolKind("mystery_kind") },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_kind_invalid",
				Field:   "Callee.Kind",
				Message: "callee_kind_unknown",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validCalleeBaselineApproval()
			tc.mutate(&approval.Callee)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCalleeReceiverFailures exercises Callee.Receiver rules
// per callee kind. Methods require a non-empty, wildcard-free receiver;
// package_function and package_variable require an empty receiver.
func TestStrictSchemaCalleeReceiverFailures(t *testing.T) {
	methodBase := func() ProtectedSymbol {
		callee := validCalleeBaseline()
		callee.Kind = ProtectedMethod
		callee.Receiver = "DupcodeRunner"
		return callee
	}

	cases := []struct {
		name   string
		mutate func(*ProtectedSymbol)
		want   []approvalSchemaIssue
	}{
		{
			name:   "method without Callee.Receiver",
			mutate: func(s *ProtectedSymbol) { s.Receiver = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_receiver_invalid",
				Field:   "Callee.Receiver",
				Message: "callee_receiver_required_for_method",
			}},
		},
		{
			name:   "method with wildcard Callee.Receiver",
			mutate: func(s *ProtectedSymbol) { s.Receiver = "*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_receiver_invalid",
				Field:   "Callee.Receiver",
				Message: "callee_receiver_wildcard_for_method",
			}},
		},
		{
			name: "package function with Callee.Receiver",
			mutate: func(s *ProtectedSymbol) {
				s.Kind = ProtectedPackageFunction
				s.Receiver = "DupcodeRunner"
			},
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_receiver_invalid",
				Field:   "Callee.Receiver",
				Message: "callee_receiver_must_be_empty",
			}},
		},
		{
			name: "package variable with Callee.Receiver",
			mutate: func(s *ProtectedSymbol) {
				s.Kind = ProtectedPackageVariable
				s.Receiver = "DupcodeRunner"
			},
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_callee_receiver_invalid",
				Field:   "Callee.Receiver",
				Message: "callee_receiver_must_be_empty",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			callee := methodBase()
			tc.mutate(&callee)
			approval := validCalleeBaselineApproval()
			approval.Callee = callee
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCalleeFieldOrdering locks the deterministic field order
// for the complete callee-side issue list.
//
// Every caller-side field is set to a baseline value that passes, and every
// callee-side field is set to a value that fails, so the resulting issue
// list demonstrates the expected callee-side order:
//
//	Callee.Layer
//	Callee.PackagePath
//	Callee.Name
//	Callee.Kind
//	Callee.Receiver
func TestStrictSchemaCalleeFieldOrdering(t *testing.T) {
	methodCallee := ProtectedSymbol{
		Layer:       "",
		PackagePath: "",
		Name:        "",
		Kind:        "",
		Receiver:    "",
	}
	approval := validCalleeBaselineApproval()
	approval.Callee = methodCallee
	issues := validateApprovalSchema(approval)
	wantFields := []string{
		"Callee.Layer",
		"Callee.PackagePath",
		"Callee.Name",
		"Callee.Kind",
	}
	// The receiver check is skipped because Kind is empty (unknown kind).
	if len(issues) != len(wantFields) {
		t.Fatalf("got %d issues, want %d: %+v", len(issues), len(wantFields), issues)
	}
	for index, want := range wantFields {
		if issues[index].Field != want {
			t.Errorf("issue %d field = %q, want %q", index, issues[index].Field, want)
		}
	}
}

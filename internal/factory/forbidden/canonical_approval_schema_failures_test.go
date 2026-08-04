// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// TestStrictSchemaReceiverFailures exercises the receiver rules per
// caller kind. Methods require a non-empty, wildcard-free receiver;
// package_function, variable_initializer, and package_init must have
// an empty receiver.
func TestStrictSchemaReceiverFailures(t *testing.T) {
	methodBase := func() ApprovedCaller {
		approval := validBaseApproval()
		approval.Receiver = "DupcodeRunner"
		approval.CallerKind = CallerKindMethod
		return approval
	}

	cases := []struct {
		name   string
		mutate func(*ApprovedCaller)
		want   []approvalSchemaIssue
	}{
		{
			name:   "method without Receiver",
			mutate: func(a *ApprovedCaller) { a.Receiver = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_receiver_invalid",
				Field:   "Receiver",
				Message: "receiver_required_for_method",
			}},
		},
		{
			name:   "method with wildcard Receiver",
			mutate: func(a *ApprovedCaller) { a.Receiver = "*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_receiver_invalid",
				Field:   "Receiver",
				Message: "receiver_wildcard_for_method",
			}},
		},
		{
			name: "package function with Receiver",
			mutate: func(a *ApprovedCaller) {
				a.CallerKind = CallerKindPackageFunction
				a.Receiver = "DupcodeRunner"
			},
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_receiver_invalid",
				Field:   "Receiver",
				Message: "receiver_must_be_empty",
			}},
		},
		{
			name: "variable initializer with Receiver",
			mutate: func(a *ApprovedCaller) {
				a.CallerKind = CallerKindVariableInitializer
				a.Receiver = "DupcodeRunner"
			},
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_receiver_invalid",
				Field:   "Receiver",
				Message: "receiver_must_be_empty",
			}},
		},
		{
			name: "package init with Receiver",
			mutate: func(a *ApprovedCaller) {
				a.CallerKind = CallerKindPackageInit
				a.Receiver = "DupcodeRunner"
			},
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_receiver_invalid",
				Field:   "Receiver",
				Message: "receiver_must_be_empty",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := methodBase()
			tc.mutate(&approval)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaReferenceClassFailures exercises reference class
// validation: empty, unknown, DOT_IMPORT, and DECLARATION must all be
// rejected. Allowed classes pass.
func TestStrictSchemaReferenceClassFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ApprovedCaller)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty ReferenceClass",
			mutate: func(a *ApprovedCaller) { a.ReferenceClass = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_reference_class_invalid",
				Field:   "ReferenceClass",
				Message: "reference_class_empty",
			}},
		},
		{
			name:   "unknown ReferenceClass",
			mutate: func(a *ApprovedCaller) { a.ReferenceClass = ReferenceClass("BOGUS") },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_reference_class_invalid",
				Field:   "ReferenceClass",
				Message: "reference_class_unknown",
			}},
		},
		{
			name:   "DOT_IMPORT",
			mutate: func(a *ApprovedCaller) { a.ReferenceClass = refDotImport },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_reference_class_invalid",
				Field:   "ReferenceClass",
				Message: "reference_class_dot_import",
			}},
		},
		{
			name:   "DECLARATION",
			mutate: func(a *ApprovedCaller) { a.ReferenceClass = refDeclaration },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_reference_class_invalid",
				Field:   "ReferenceClass",
				Message: "reference_class_declaration",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validBaseApproval()
			tc.mutate(&approval)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestStrictSchemaCardinalityFailures exercises zero and negative
// Cardinality. Positive values must continue to pass.
func TestStrictSchemaCardinalityFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ApprovedCaller)
		want   []approvalSchemaIssue
	}{
		{
			name:   "zero Cardinality",
			mutate: func(a *ApprovedCaller) { a.Cardinality = 0 },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_cardinality_invalid",
				Field:   "Cardinality",
				Message: "cardinality_zero",
			}},
		},
		{
			name:   "negative Cardinality",
			mutate: func(a *ApprovedCaller) { a.Cardinality = -3 },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_cardinality_invalid",
				Field:   "Cardinality",
				Message: "cardinality_negative",
			}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validBaseApproval()
			tc.mutate(&approval)
			got := validateApprovalSchema(approval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("validateApprovalSchema = %+v, want %+v", got, tc.want)
			}
		})
	}

	// Positive cardinalities remain valid.
	for _, n := range []int{1, 2, 7} {
		approval := validBaseApproval()
		approval.Cardinality = n
		if issues := validateApprovalSchema(approval); len(issues) != 0 {
			t.Errorf("cardinality %d produced issues %+v, want none", n, issues)
		}
	}
}

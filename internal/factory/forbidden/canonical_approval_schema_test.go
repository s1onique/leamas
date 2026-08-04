// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// validBaseApproval returns a strictly valid approval baseline used by
// the strict-schema unit tests. Every test case mutates a single field
// of this baseline so the issue list it produces is fully attributable
// to that field.
func validBaseApproval() ApprovedCaller {
	return ApprovedCaller{
		PackagePath:    "example.test/policy/caller",
		Function:       "Allowed",
		Receiver:       "",
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

// issue lists that should be emitted from each valid baseline variant.
// Each list is constructed inline so tests stay self-contained.

// TestStrictSchemaValidBaseline ensures the unmodified baseline passes
// with zero issues, which is the precondition for every field-level
// failure case.
func TestStrictSchemaValidBaseline(t *testing.T) {
	if issues := validateApprovalSchema(validBaseApproval()); len(issues) != 0 {
		t.Fatalf("baseline returned issues %+v, want none", issues)
	}
}

// TestStrictSchemaValidCoverage exercises one positive case per required
// approval shape so the validator's allow-list is explicitly verified.
func TestStrictSchemaValidCoverage(t *testing.T) {
	method := validBaseApproval()
	method.Receiver = "DupcodeRunner"
	method.CallerKind = CallerKindMethod
	method.Function = "RunCap"

	methodValue := method
	methodValue.ReferenceClass = refMethodValue

	methodExpression := method
	methodExpression.ReferenceClass = refMethodExpression

	functionValue := validBaseApproval()
	functionValue.ReferenceClass = refFunctionValue

	packageVariable := validBaseApproval()
	packageVariable.ReferenceClass = refPackageVariable

	cases := []struct {
		name     string
		approval ApprovedCaller
	}{
		{"valid package-function direct call", validBaseApproval()},
		{"valid method direct call", method},
		{"valid package-function function value", functionValue},
		{"valid method value", methodValue},
		{"valid method expression", methodExpression},
		{"valid package-variable reference", packageVariable},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if issues := validateApprovalSchema(tc.approval); len(issues) != 0 {
				t.Errorf("approval returned issues %+v, want none", issues)
			}
		})
	}
}

// TestStrictSchemaIdentityFailures exercises PackagePath and Function
// identity issues. Each issue list is asserted in the exact field order
// produced by the validator: PackagePath first, Function second.
func TestStrictSchemaIdentityFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ApprovedCaller)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty PackagePath",
			mutate: func(a *ApprovedCaller) { a.PackagePath = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_identity_invalid",
				Field:   "PackagePath",
				Message: "package_path_empty",
			}},
		},
		{
			name:   "wildcard PackagePath",
			mutate: func(a *ApprovedCaller) { a.PackagePath = "example.*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_identity_invalid",
				Field:   "PackagePath",
				Message: "package_path_wildcard",
			}},
		},
		{
			name:   "empty Function",
			mutate: func(a *ApprovedCaller) { a.Function = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_identity_invalid",
				Field:   "Function",
				Message: "function_empty",
			}},
		},
		{
			name:   "wildcard Function",
			mutate: func(a *ApprovedCaller) { a.Function = "*" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_identity_invalid",
				Field:   "Function",
				Message: "function_wildcard",
			}},
		},
		{
			name:   "source-coordinate Function",
			mutate: func(a *ApprovedCaller) { a.Function = "Allowed@10:1" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_identity_invalid",
				Field:   "Function",
				Message: "function_source_coordinate",
			}},
		},
		{
			name: "empty PackagePath and Function",
			mutate: func(a *ApprovedCaller) {
				a.PackagePath = ""
				a.Function = ""
			},
			want: []approvalSchemaIssue{
				{
					Kind:    "authority_policy_approval_identity_invalid",
					Field:   "PackagePath",
					Message: "package_path_empty",
				},
				{
					Kind:    "authority_policy_approval_identity_invalid",
					Field:   "Function",
					Message: "function_empty",
				},
			},
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

// TestStrictSchemaCallerKindFailures exercises CallerKind validation:
// empty, unknown, function-literal, whitespace, and case variants.
func TestStrictSchemaCallerKindFailures(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ApprovedCaller)
		want   []approvalSchemaIssue
	}{
		{
			name:   "empty CallerKind",
			mutate: func(a *ApprovedCaller) { a.CallerKind = "" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_caller_kind_invalid",
				Field:   "CallerKind",
				Message: "caller_kind_empty",
			}},
		},
		{
			name:   "unknown CallerKind",
			mutate: func(a *ApprovedCaller) { a.CallerKind = "unusual_kind" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_caller_kind_invalid",
				Field:   "CallerKind",
				Message: "caller_kind_unknown",
			}},
		},
		{
			name:   "function-literal CallerKind",
			mutate: func(a *ApprovedCaller) { a.CallerKind = CallerKindFunctionLiteral },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_caller_kind_invalid",
				Field:   "CallerKind",
				Message: "caller_kind_unknown",
			}},
		},
		{
			name:   "whitespace CallerKind",
			mutate: func(a *ApprovedCaller) { a.CallerKind = " " },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_caller_kind_invalid",
				Field:   "CallerKind",
				Message: "caller_kind_unknown",
			}},
		},
		{
			name:   "case variant CallerKind",
			mutate: func(a *ApprovedCaller) { a.CallerKind = "Method" },
			want: []approvalSchemaIssue{{
				Kind:    "authority_policy_approval_caller_kind_invalid",
				Field:   "CallerKind",
				Message: "caller_kind_unknown",
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

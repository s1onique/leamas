// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// TestStrictSchemaFieldOrdering locks the deterministic field order of
// the issue list to: PackagePath, Function, CallerKind, Receiver,
// ReferenceClass, Cardinality. Each subtest mutates a single additional
// field so the produced issue list demonstrates the expected order.
func TestStrictSchemaFieldOrdering(t *testing.T) {
	cases := []struct {
		name       string
		mutate     func(*ApprovedCaller)
		wantFields []string
	}{
		{
			name: "package-function receiver missing exercises receiver field after CallerKind",
			mutate: func(a *ApprovedCaller) {
				a.PackagePath = ""
				a.Function = "*"
				a.CallerKind = CallerKindPackageFunction
				a.Receiver = "DupcodeRunner"
				a.ReferenceClass = refDotImport
				a.Cardinality = 0
			},
			wantFields: []string{
				"PackagePath", "Function", "Receiver", "ReferenceClass", "Cardinality",
			},
		},
		{
			name: "unknown CallerKind exercises CallerKind before receiver (receiver skipped)",
			mutate: func(a *ApprovedCaller) {
				a.PackagePath = ""
				a.Function = "*"
				a.CallerKind = "bogus"
				a.Receiver = "DupcodeRunner"
				a.ReferenceClass = refDotImport
				a.Cardinality = 0
			},
			wantFields: []string{
				"PackagePath", "Function", "CallerKind", "ReferenceClass", "Cardinality",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			approval := validBaseApproval()
			tc.mutate(&approval)
			issues := validateApprovalSchema(approval)
			if len(issues) != len(tc.wantFields) {
				t.Fatalf("got %d issues, want %d: %+v", len(issues), len(tc.wantFields), issues)
			}
			for index, want := range tc.wantFields {
				if issues[index].Field != want {
					t.Errorf("issue %d field = %q, want %q", index, issues[index].Field, want)
				}
			}
		})
	}
}

// TestStrictSchemaPureNoMutation locks the contract that the validator
// never mutates the input. It compares the input before and after the
// call to make sure all fields are preserved verbatim.
func TestStrictSchemaPureNoMutation(t *testing.T) {
	original := validBaseApproval()
	copy := original
	_ = validateApprovalSchema(original)
	if !reflect.DeepEqual(original, copy) {
		t.Errorf("validateApprovalSchema mutated input: before=%+v after=%+v", copy, original)
	}
}

// TestValidApprovalReferenceClassAndObservedClass locks the split
// between approval-side and observed-side class validation. DOT_IMPORT
// is allowed only for observed edges; DECLARATION is rejected by both.
func TestValidApprovalReferenceClassAndObservedClass(t *testing.T) {
	if validApprovalReferenceClass(refDotImport) {
		t.Errorf("validApprovalReferenceClass(DOT_IMPORT) = true, want false")
	}
	if validApprovalReferenceClass(refDeclaration) {
		t.Errorf("validApprovalReferenceClass(DECLARATION) = true, want false")
	}
	if !validObservedReferenceClass(refDotImport) {
		t.Errorf("validObservedReferenceClass(DOT_IMPORT) = false, want true")
	}
	if validObservedReferenceClass(refDeclaration) {
		t.Errorf("validObservedReferenceClass(DECLARATION) = true, want false")
	}

	// The approved classes are accepted by both helpers.
	for _, class := range []ReferenceClass{
		refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable,
	} {
		if !validApprovalReferenceClass(class) {
			t.Errorf("validApprovalReferenceClass(%s) = false, want true", class)
		}
		if !validObservedReferenceClass(class) {
			t.Errorf("validObservedReferenceClass(%s) = false, want true", class)
		}
	}
}

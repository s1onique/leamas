// SPDX-License-Identifier: Apache-2.0

package forbidden

import (
	"reflect"
	"testing"
)

// TestValidObservedReferenceClassContract locks the split between the
// observed-class allow-list and the approval-class allow-list. The
// observed helper must accept DOT_IMPORT (which traversal may observe)
// but must reject DECLARATION and unknown or empty classes.
func TestValidObservedReferenceClassContract(t *testing.T) {
	validObserved := []ReferenceClass{
		refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable, refDotImport,
	}
	for _, class := range validObserved {
		if !validObservedReferenceClass(class) {
			t.Errorf("validObservedReferenceClass(%s) = false, want true", class)
		}
	}

	invalidObserved := []ReferenceClass{
		"",
		refDeclaration,
		ReferenceClass("BOGUS"),
	}
	for _, class := range invalidObserved {
		if validObservedReferenceClass(class) {
			t.Errorf("validObservedReferenceClass(%s) = true, want false", class)
		}
	}
}

// TestValidApprovalReferenceClassRejectsDotImport locks the policy that
// DOT_IMPORT may be observed by traversal but is never approvable.
func TestValidApprovalReferenceClassRejectsDotImport(t *testing.T) {
	if validApprovalReferenceClass(refDotImport) {
		t.Errorf("validApprovalReferenceClass(DOT_IMPORT) = true, want false")
	}
	if validApprovalReferenceClass(refDeclaration) {
		t.Errorf("validApprovalReferenceClass(DECLARATION) = true, want false")
	}

	approved := []ReferenceClass{
		refDirectCall, refFunctionValue, refMethodValue, refMethodExpression, refPackageVariable,
	}
	for _, class := range approved {
		if !validApprovalReferenceClass(class) {
			t.Errorf("validApprovalReferenceClass(%s) = false, want true", class)
		}
	}
}

// TestValidateObservedEdgesRejectsInvalidClasses wires
// validObservedReferenceClass into the per-edge validation loop. The
// invariant check must run before approval matching so an internal edge
// with an invalid class never participates in reference-class or
// cardinality accounting.
func TestValidateObservedEdgesRejectsInvalidClasses(t *testing.T) {
	cases := []struct {
		name  string
		class ReferenceClass
		want  bool
	}{
		{"DIRECT_CALL is valid", refDirectCall, true},
		{"FUNCTION_VALUE is valid", refFunctionValue, true},
		{"METHOD_VALUE is valid", refMethodValue, true},
		{"METHOD_EXPRESSION is valid", refMethodExpression, true},
		{"PACKAGE_VARIABLE_REFERENCE is valid", refPackageVariable, true},
		{"DOT_IMPORT is valid observed class", refDotImport, true},
		{"empty class is invalid", "", false},
		{"unknown class is invalid", ReferenceClass("BOGUS"), false},
		{"DECLARATION is invalid", refDeclaration, false},
	}

	want := []bool{true, true, true, true, true, true, false, false, false}
	got := []bool{}
	for _, tc := range cases {
		got = append(got, validObservedReferenceClass(tc.class))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("validObservedReferenceClass results = %+v, want %+v", got, want)
	}
}

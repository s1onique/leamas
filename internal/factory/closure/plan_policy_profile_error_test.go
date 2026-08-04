package closure

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestPlanPolicyProfileErrorInvalidKindError verifies that Error() produces
// stable output for invalid PolicyProfileErrorKind values.
func TestPlanPolicyProfileErrorInvalidKindError(t *testing.T) {
	// Valid kinds are: 0 (Unknown), 1 (Unimplemented), 2 (MissingCheck)
	// Invalid kinds are anything outside this range
	invalidKinds := []PolicyProfileErrorKind{
		PolicyProfileErrorKind(-1),  // negative
		PolicyProfileErrorKind(3),   // just beyond known values
		PolicyProfileErrorKind(100), // far beyond known values
	}

	for _, kind := range invalidKinds {
		err := &PolicyProfileError{
			ProfileName: "test-profile",
			CheckID:     "test-check",
			Kind:        kind,
		}

		msg := err.Error()
		if msg == "" {
			t.Errorf("Kind=%d: Error() returned empty string", kind)
		}

		// Error message must be stable (same input -> same output)
		msg2 := err.Error()
		if msg != msg2 {
			t.Errorf("Kind=%d: Error() not stable: first=%q, second=%q", kind, msg, msg2)
		}

		// Error message should identify the numeric kind
		wantSubstr := fmt.Sprintf("kind=%d", kind)
		if !strings.Contains(msg, wantSubstr) {
			t.Errorf("Kind=%d: Error() = %q, want substring %q", kind, msg, wantSubstr)
		}
	}
}

// TestPlanPolicyProfileErrorInvalidKindPlanDiagnosticsCount1 verifies that
// PlanDiagnostics() returns exactly 1 diagnostic for invalid kinds.
func TestPlanPolicyProfileErrorInvalidKindPlanDiagnosticsCount1(t *testing.T) {
	invalidKinds := []PolicyProfileErrorKind{
		PolicyProfileErrorKind(-1),
		PolicyProfileErrorKind(3),
		PolicyProfileErrorKind(100),
	}

	for _, kind := range invalidKinds {
		err := &PolicyProfileError{
			ProfileName: "test-profile",
			CheckID:     "test-check",
			Kind:        kind,
		}

		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Errorf("Kind=%d: len(PlanDiagnostics()) = %d, want 1", kind, len(diags))
		}
	}
}

// TestPlanPolicyProfileErrorInvalidKindPlanDiagnosticsCount20 verifies that
// 20 different invalid PolicyProfileErrorKind values each produce exactly
// 1 diagnostic.
func TestPlanPolicyProfileErrorInvalidKindPlanDiagnosticsCount20(t *testing.T) {
	// Generate 20 invalid kinds: mix of negatives and large values
	// Valid kinds are 0, 1, 2
	invalidKinds := make([]PolicyProfileErrorKind, 20)
	for i := range invalidKinds {
		if i%2 == 0 {
			invalidKinds[i] = PolicyProfileErrorKind(-1 - i)
		} else {
			// Values that don't match any known case (not 0, 1, or 2)
			invalidKinds[i] = PolicyProfileErrorKind(3 + i)
		}
	}

	for _, kind := range invalidKinds {
		err := &PolicyProfileError{
			ProfileName: fmt.Sprintf("profile-%d", kind),
			CheckID:     fmt.Sprintf("check-%d", kind),
			Kind:        kind,
		}

		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Errorf("Kind=%d: len(PlanDiagnostics()) = %d, want 1", kind, len(diags))
		}
	}
}

// TestPlanPolicyProfileErrorInvalidKindContract verifies the stable invariant
// contract for invalid kinds:
// - Empty InstancePath
// - PropertyName identifies invalid numeric kind
// - Non-empty Code, Keyword, Message
func TestPlanPolicyProfileErrorInvalidKindContract(t *testing.T) {
	invalidKinds := []PolicyProfileErrorKind{
		PolicyProfileErrorKind(-1),
		PolicyProfileErrorKind(3),
		PolicyProfileErrorKind(100),
	}

	for _, kind := range invalidKinds {
		t.Run(fmt.Sprintf("Kind=%d", kind), func(t *testing.T) {
			err := &PolicyProfileError{
				ProfileName: "test-profile",
				CheckID:     "test-check",
				Kind:        kind,
			}

			diags := err.PlanDiagnostics()
			if len(diags) != 1 {
				t.Fatalf("len(diags) = %d, want 1", len(diags))
			}

			diag := diags[0]

			// Contract: Empty InstancePath
			if diag.InstancePath != "" {
				t.Errorf("InstancePath = %q, want empty", diag.InstancePath)
			}

			// Contract: PropertyName identifies invalid numeric kind
			wantPropName := fmt.Sprintf("PolicyProfileErrorKind=%d", kind)
			if diag.PropertyName != wantPropName {
				t.Errorf("PropertyName = %q, want %q", diag.PropertyName, wantPropName)
			}

			// Contract: Non-empty Code
			if diag.Code == "" {
				t.Error("Code is empty, want non-empty")
			}

			// Contract: Non-empty Keyword
			if diag.Keyword == "" {
				t.Error("Keyword is empty, want non-empty")
			}

			// Contract: Non-empty Message
			if diag.Message == "" {
				t.Error("Message is empty, want non-empty")
			}

			// Verify message matches Error()
			if diag.Message != err.Error() {
				t.Errorf("Message = %q, want %q (err.Error())", diag.Message, err.Error())
			}
		})
	}
}

// TestPlanPolicyProfileErrorInvalidKindStability verifies that Error() and
// PlanDiagnostics() produce stable output across multiple calls.
func TestPlanPolicyProfileErrorInvalidKindStability(t *testing.T) {
	err := &PolicyProfileError{
		ProfileName: "stable-profile",
		CheckID:     "stable-check",
		Kind:        PolicyProfileErrorKind(99),
	}

	// Call Error() 3 times
	errMsg1 := err.Error()
	errMsg2 := err.Error()
	errMsg3 := err.Error()

	if errMsg1 != errMsg2 || errMsg2 != errMsg3 {
		t.Errorf("Error() not stable across calls: %q, %q, %q", errMsg1, errMsg2, errMsg3)
	}

	// Call PlanDiagnostics() 3 times
	diags1 := err.PlanDiagnostics()
	diags2 := err.PlanDiagnostics()
	diags3 := err.PlanDiagnostics()

	if len(diags1) != 1 || len(diags2) != 1 || len(diags3) != 1 {
		t.Errorf("PlanDiagnostics() count not stable: %d, %d, %d", len(diags1), len(diags2), len(diags3))
	}

	// Each call should produce semantically equivalent diagnostics
	for i, d1 := range diags1 {
		d2 := diags2[i]
		d3 := diags3[i]

		if d1.InstancePath != d2.InstancePath || d2.InstancePath != d3.InstancePath {
			t.Errorf("InstancePath not stable: %q, %q, %q", d1.InstancePath, d2.InstancePath, d3.InstancePath)
		}
		if d1.PropertyName != d2.PropertyName || d2.PropertyName != d3.PropertyName {
			t.Errorf("PropertyName not stable: %q, %q, %q", d1.PropertyName, d2.PropertyName, d3.PropertyName)
		}
		if d1.Message != d2.Message || d2.Message != d3.Message {
			t.Errorf("Message not stable: %q, %q, %q", d1.Message, d2.Message, d3.Message)
		}
		if d1.Code != d2.Code || d2.Code != d3.Code {
			t.Errorf("Code not stable: %q, %q, %q", d1.Code, d2.Code, d3.Code)
		}
		if d1.Keyword != d2.Keyword || d2.Keyword != d3.Keyword {
			t.Errorf("Keyword not stable: %q, %q, %q", d1.Keyword, d2.Keyword, d3.Keyword)
		}
	}
}

// TestPlanPolicyProfileErrorInvalidKindUnwrap verifies error unwrapping works.
func TestPlanPolicyProfileErrorInvalidKindUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &PolicyProfileError{
		ProfileName: "test-profile",
		CheckID:     "test-check",
		Kind:        PolicyProfileErrorKind(-1),
		Cause:       cause,
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) should be true")
	}

	var profileErr *PolicyProfileError
	if !errors.As(err, &profileErr) {
		t.Error("errors.As should extract PolicyProfileError")
	}
}

// TestPlanPolicyProfileErrorInvalidKindIsolation verifies that mutations
// to returned diagnostics don't affect subsequent calls.
func TestPlanPolicyProfileErrorInvalidKindIsolation(t *testing.T) {
	err := &PolicyProfileError{
		ProfileName: "isolate-profile",
		CheckID:     "isolate-check",
		Kind:        PolicyProfileErrorKind(50),
	}

	diags1 := err.PlanDiagnostics()
	originalPropName := diags1[0].PropertyName

	// Mutate the returned diagnostic
	diags1[0].PropertyName = "MUTATED"

	// Second call should return original values
	diags2 := err.PlanDiagnostics()

	if diags2[0].PropertyName != originalPropName {
		t.Errorf("Isolation violated: PropertyName = %q, want %q", diags2[0].PropertyName, originalPropName)
	}
}

// TestPlanPolicyProfileErrorInvalidKindDifferentValues verifies that different
// invalid kinds produce different PropertyName values.
func TestPlanPolicyProfileErrorInvalidKindDifferentValues(t *testing.T) {
	kinds := []PolicyProfileErrorKind{
		PolicyProfileErrorKind(-5),
		PolicyProfileErrorKind(5),
		PolicyProfileErrorKind(10),
		PolicyProfileErrorKind(1000),
	}

	propNames := make(map[string]bool)
	for _, kind := range kinds {
		err := &PolicyProfileError{
			ProfileName: "test",
			CheckID:     "test",
			Kind:        kind,
		}
		diags := err.PlanDiagnostics()
		propName := diags[0].PropertyName

		if propNames[propName] {
			t.Errorf("Duplicate PropertyName %q for different Kind=%d", propName, kind)
		}
		propNames[propName] = true
	}
}

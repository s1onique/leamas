package closure

import (
	"errors"
	"testing"
)

// TestEnvironmentSemanticDeterminism validates that environment validation produces
// identical diagnostics regardless of map iteration order by creating equivalent
// invalid environment maps in opposite insertion orders and comparing ValidatePlan results.
func TestEnvironmentSemanticDeterminism(t *testing.T) {
	testCases := []struct {
		name        string
		envForward  map[string]string
		envReverse  map[string]string
		wantPath    string
		wantCode    PlanValidationCode
		wantKeyword PlanValidationKeyword
		wantMsg     string
	}{
		{
			name:        "invalid key starts with digit - forward order",
			envForward:  map[string]string{"1INVALID": "value"},
			envReverse:  map[string]string{"1INVALID": "value"},
			wantPath:    "/checks/0/environment/1INVALID",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "1INVALID"`,
		},
		{
			name:        "invalid key contains dash - forward order",
			envForward:  map[string]string{"INVALID-KEY": "value"},
			envReverse:  map[string]string{"INVALID-KEY": "value"},
			wantPath:    "/checks/0/environment/INVALID-KEY",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "INVALID-KEY"`,
		},
		{
			name:        "key with slash - RFC 6901 encoded",
			envForward:  map[string]string{"PATH/TO": "value"},
			envReverse:  map[string]string{"PATH/TO": "value"},
			wantPath:    "/checks/0/environment/PATH~1TO",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "PATH/TO"`,
		},
		{
			name:        "key with tilde - RFC 6901 encoded",
			envForward:  map[string]string{"USER~NAME": "value"},
			envReverse:  map[string]string{"USER~NAME": "value"},
			wantPath:    "/checks/0/environment/USER~0NAME",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "USER~NAME"`,
		},
		{
			name:        "key with both slash and tilde - RFC 6901 encoded",
			envForward:  map[string]string{"PATH/~HOME": "value"},
			envReverse:  map[string]string{"PATH/~HOME": "value"},
			wantPath:    "/checks/0/environment/PATH~1~0HOME",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "PATH/~HOME"`,
		},
		{
			name:        "multiple invalid keys - different insertion orders produce same first error",
			envForward:  map[string]string{"FIRST-KEY": "v1", "SECOND-KEY": "v2"},
			envReverse:  map[string]string{"SECOND-KEY": "v2", "FIRST-KEY": "v1"},
			wantPath:    "/checks/0/environment/FIRST-KEY",
			wantCode:    PlanCodeSemanticConstraintFailed,
			wantKeyword: KeywordType,
			wantMsg:     `contains invalid entry "FIRST-KEY"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan := validSemanticPlanFixture(t)

			// Set forward order environment
			plan.Checks[0].Environment = tc.envForward
			errForward := ValidatePlan(plan)

			// Reset plan and set reverse order environment
			plan = validSemanticPlanFixture(t)
			plan.Checks[0].Environment = tc.envReverse
			errReverse := ValidatePlan(plan)

			// Both should produce errors
			if errForward == nil {
				t.Fatal("ValidatePlan(forward) = nil, want error")
			}
			if errReverse == nil {
				t.Fatal("ValidatePlan(reverse) = nil, want error")
			}

			// Extract diagnostics from both errors
			var sourceForward, sourceReverse planDiagnosticSource
			if !errors.As(errForward, &sourceForward) {
				t.Fatal("errForward does not implement planDiagnosticSource")
			}
			if !errors.As(errReverse, &sourceReverse) {
				t.Fatal("errReverse does not implement planDiagnosticSource")
			}

			diagsForward := sourceForward.PlanDiagnostics()
			diagsReverse := sourceReverse.PlanDiagnostics()

			// Both should have at least one diagnostic
			if len(diagsForward) == 0 {
				t.Fatal("forward diagnostics empty")
			}
			if len(diagsReverse) == 0 {
				t.Fatal("reverse diagnostics empty")
			}

			// Compare error types - both should be the same concrete type
			typeForward := errorTypeName(errForward)
			typeReverse := errorTypeName(errReverse)
			if typeForward != typeReverse {
				t.Errorf("error type mismatch: forward=%s, reverse=%s", typeForward, typeReverse)
			}

			// Compare the first diagnostic's properties (sorted order ensures consistency)
			diagF := diagsForward[0]
			diagR := diagsReverse[0]

			// Require identical concrete error type
			if diagF.InstancePath != diagR.InstancePath {
				t.Errorf("InstancePath mismatch: forward=%q, reverse=%q", diagF.InstancePath, diagR.InstancePath)
			}
			if diagF.Code != diagR.Code {
				t.Errorf("Code mismatch: forward=%v, reverse=%v", diagF.Code, diagR.Code)
			}
			if diagF.Keyword != diagR.Keyword {
				t.Errorf("Keyword mismatch: forward=%v, reverse=%v", diagF.Keyword, diagR.Keyword)
			}
			if diagF.PropertyName != diagR.PropertyName {
				t.Errorf("PropertyName mismatch: forward=%q, reverse=%q", diagF.PropertyName, diagR.PropertyName)
			}
			if diagF.Message != diagR.Message {
				t.Errorf("Message mismatch: forward=%q, reverse=%q", diagF.Message, diagR.Message)
			}

			// Verify expected values match
			if diagF.InstancePath != tc.wantPath {
				t.Errorf("InstancePath = %q, want %q", diagF.InstancePath, tc.wantPath)
			}
			if diagF.Code != tc.wantCode {
				t.Errorf("Code = %v, want %v", diagF.Code, tc.wantCode)
			}
			if diagF.Keyword != tc.wantKeyword {
				t.Errorf("Keyword = %v, want %v", diagF.Keyword, tc.wantKeyword)
			}
			if diagF.Message != tc.wantMsg && !contains(diagF.Message, tc.wantMsg) {
				t.Errorf("Message = %q, want to contain %q", diagF.Message, tc.wantMsg)
			}
		})
	}
}

// TestEnvironmentKeyWithSlashAndTilde verifies that an environment key containing
// both / and ~ characters produces an exact RFC 6901 JSON Pointer path.
func TestEnvironmentKeyWithSlashAndTilde(t *testing.T) {
	plan := validSemanticPlanFixture(t)
	// Key with both / and ~ should be encoded per RFC 6901
	// RFC 6901 encoding: ~ → ~0, / → ~1 (order matters: encode ~ first, then /)
	plan.Checks[0].Environment = map[string]string{"DATA/~PATH/FILE": "value"}

	err := ValidatePlan(plan)
	if err == nil {
		t.Fatal("ValidatePlan() = nil, want error for invalid key with / and ~")
	}

	var source planDiagnosticSource
	if !errors.As(err, &source) {
		t.Fatal("err does not implement planDiagnosticSource")
	}

	diags := source.PlanDiagnostics()
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diags))
	}

	diag := diags[0]

	// Verify exact RFC 6901 encoding:
	// Original key: DATA/~PATH/FILE
	// After encoding ~ → ~0: DATA/~0PATH/FILE
	// After encoding / → ~1: DATA~1~0PATH~1FILE
	wantPath := "/checks/0/environment/DATA~1~0PATH~1FILE"
	if diag.InstancePath != wantPath {
		t.Errorf("InstancePath = %q, want %q (RFC 6901 encoded)", diag.InstancePath, wantPath)
	}

	// Verify code
	if diag.Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
	}

	// Verify keyword
	if diag.Keyword != KeywordType {
		t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordType)
	}

	// Verify message contains the original key (not the encoded form)
	wantMsg := `checks[0].environment contains invalid entry "DATA/~PATH/FILE"`
	if diag.Message != wantMsg {
		t.Errorf("Message = %q, want %q", diag.Message, wantMsg)
	}
}

// TestEnvironmentValidationOrderIndependent verifies that the order of keys in the
// environment map does not affect which invalid key is reported when using
// sorted iteration.
func TestEnvironmentValidationOrderIndependent(t *testing.T) {
	// Create two equivalent invalid environments with different insertion orders
	// but containing the same set of invalid keys.
	// The validation sorts keys and reports the first one that fails.
	// "BAD~TILDE" sorts before "WORST/KEY" (B < W), so BAD~TILDE is reported first.
	envForward := map[string]string{
		"BAD~TILDE": "v1",
		"WORST/KEY": "v2",
	}

	envReverse := map[string]string{
		"WORST/KEY": "v2",
		"BAD~TILDE": "v1",
	}

	plan := validSemanticPlanFixture(t)
	plan.Checks[0].Environment = envForward
	errForward := ValidatePlan(plan)

	plan = validSemanticPlanFixture(t)
	plan.Checks[0].Environment = envReverse
	errReverse := ValidatePlan(plan)

	if errForward == nil || errReverse == nil {
		t.Fatal("both plans should produce validation errors")
	}

	// Both should implement planDiagnosticSource
	var sourceForward, sourceReverse planDiagnosticSource
	if !errors.As(errForward, &sourceForward) || !errors.As(errReverse, &sourceReverse) {
		t.Fatal("errors should implement planDiagnosticSource")
	}

	diagsForward := sourceForward.PlanDiagnostics()
	diagsReverse := sourceReverse.PlanDiagnostics()

	// Both should have exactly 1 diagnostic (validation stops at first error)
	if len(diagsForward) != 1 {
		t.Errorf("forward got %d diagnostics, want 1", len(diagsForward))
	}
	if len(diagsReverse) != 1 {
		t.Errorf("reverse got %d diagnostics, want 1", len(diagsReverse))
	}

	// Verify the diagnostics are identical regardless of insertion order.
	diagF := diagsForward[0]
	diagR := diagsReverse[0]

	// Require identical concrete error type, path, code, keyword, property, and message
	if errorTypeName(errForward) != errorTypeName(errReverse) {
		t.Errorf("error type mismatch: forward=%s, reverse=%s",
			errorTypeName(errForward), errorTypeName(errReverse))
	}

	if diagF.InstancePath != diagR.InstancePath {
		t.Errorf("InstancePath mismatch: forward=%q, reverse=%q", diagF.InstancePath, diagR.InstancePath)
	}
	if diagF.Code != diagR.Code {
		t.Errorf("Code mismatch: forward=%v, reverse=%v", diagF.Code, diagR.Code)
	}
	if diagF.Keyword != diagR.Keyword {
		t.Errorf("Keyword mismatch: forward=%v, reverse=%v", diagF.Keyword, diagR.Keyword)
	}
	if diagF.PropertyName != diagR.PropertyName {
		t.Errorf("PropertyName mismatch: forward=%q, reverse=%q", diagF.PropertyName, diagR.PropertyName)
	}
	if diagF.Message != diagR.Message {
		t.Errorf("Message mismatch: forward=%q, reverse=%q", diagF.Message, diagR.Message)
	}

	// Verify the path is correctly RFC 6901 encoded
	// BAD~TILDE → BAD~0TILDE
	wantPath := "/checks/0/environment/BAD~0TILDE"
	if diagF.InstancePath != wantPath {
		t.Errorf("InstancePath = %q, want %q", diagF.InstancePath, wantPath)
	}

	// Verify the message contains the original key
	wantMsg := `checks[0].environment contains invalid entry "BAD~TILDE"`
	if diagF.Message != wantMsg {
		t.Errorf("Message = %q, want %q", diagF.Message, wantMsg)
	}
}

// errorTypeName returns the concrete type name of an error.
func errorTypeName(err error) string {
	if _, ok := err.(*PlanSemanticError); ok {
		return "PlanSemanticError"
	}
	if _, ok := err.(*PlanSemanticMultiError); ok {
		return "PlanSemanticMultiError"
	}
	return "unknown"
}

// contains returns true if s contains substr.
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr))
}

// findSubstring implements a simple substring search.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

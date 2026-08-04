package closure

import (
	"errors"
	"slices"
	"testing"
)

// TestPlanPolicyRequiredErrorPlanDiagnosticsEmpty verifies that an empty
// Missing slice returns a non-nil empty slice.
func TestPlanPolicyRequiredErrorPlanDiagnosticsEmpty(t *testing.T) {
	err := &PlanPolicyRequiredError{Missing: []string{}}
	diags := err.PlanDiagnostics()

	if diags == nil {
		t.Fatal("PlanDiagnostics() returned nil, want non-nil empty slice")
	}
	if len(diags) != 0 {
		t.Fatalf("len(diags) = %d, want 0", len(diags))
	}

	// Verify it implements planDiagnosticSource
	var source planDiagnosticSource
	if !errors.As(err, &source) {
		t.Error("PlanPolicyRequiredError should implement planDiagnosticSource")
	}
}

// TestPlanPolicyRequiredErrorPlanDiagnosticsReversedOrder verifies that
// diagnostics are ordered by PolicyFieldOrder regardless of the input order.
func TestPlanPolicyRequiredErrorPlanDiagnosticsReversedOrder(t *testing.T) {
	// Input is reversed order of PolicyFieldOrder
	reversed := []string{
		"require_diff_check",
		"forbid_tracked_full_digests",
		"require_clean_after",
		"require_clean_before",
	}
	err := &PlanPolicyRequiredError{Missing: reversed}
	diags := err.PlanDiagnostics()

	if len(diags) != 4 {
		t.Fatalf("len(diags) = %d, want 4", len(diags))
	}

	// Expected order is PolicyFieldOrder: require_clean_before, require_clean_after,
	// forbid_tracked_full_digests, require_diff_check
	wantPaths := []string{
		"/policy/require_clean_before",
		"/policy/require_clean_after",
		"/policy/forbid_tracked_full_digests",
		"/policy/require_diff_check",
	}
	wantProps := []string{
		"require_clean_before",
		"require_clean_after",
		"forbid_tracked_full_digests",
		"require_diff_check",
	}

	for i, diag := range diags {
		if diag.InstancePath != wantPaths[i] {
			t.Errorf("diags[%d].InstancePath = %q, want %q", i, diag.InstancePath, wantPaths[i])
		}
		if diag.PropertyName != wantProps[i] {
			t.Errorf("diags[%d].PropertyName = %q, want %q", i, diag.PropertyName, wantProps[i])
		}
		if diag.Code != PlanCodeRequiredPropertyMissing {
			t.Errorf("diags[%d].Code = %v, want %v", i, diag.Code, PlanCodeRequiredPropertyMissing)
		}
		if diag.Keyword != KeywordRequired {
			t.Errorf("diags[%d].Keyword = %v, want %v", i, diag.Keyword, KeywordRequired)
		}
	}
}

// TestPlanPolicyRequiredErrorPlanDiagnosticsDuplicates verifies that duplicate
// fields in Missing are deduplicated.
func TestPlanPolicyRequiredErrorPlanDiagnosticsDuplicates(t *testing.T) {
	// Contains duplicates of two fields
	withDupes := []string{
		"require_clean_before",
		"require_clean_after",
		"require_clean_before", // duplicate
		"require_diff_check",
		"require_clean_after", // duplicate
		"forbid_tracked_full_digests",
	}
	err := &PlanPolicyRequiredError{Missing: withDupes}
	diags := err.PlanDiagnostics()

	// 4 unique fields (all duplicates should be removed)
	if len(diags) != 4 {
		t.Fatalf("len(diags) = %d, want 4 (duplicates should be deduplicated)", len(diags))
	}

	// Verify each field appears exactly once in diagnostics
	fieldCount := make(map[string]int)
	for _, diag := range diags {
		fieldCount[diag.PropertyName]++
	}
	for field, count := range fieldCount {
		if count != 1 {
			t.Errorf("field %q appears %d times in diagnostics, want 1", field, count)
		}
	}
}

// TestPlanPolicyRequiredErrorPlanDiagnosticsMixedFields verifies that
// mixed known and unknown fields produce correct diagnostics.
func TestPlanPolicyRequiredErrorPlanDiagnosticsMixedFields(t *testing.T) {
	mixed := []string{
		"unknown_field",
		"require_clean_before",
		"another_unknown",
		"require_diff_check",
	}
	err := &PlanPolicyRequiredError{Missing: mixed}
	diags := err.PlanDiagnostics()

	// 2 known fields + 1 diagnostic for all unknown fields = 3 diagnostics
	if len(diags) != 3 {
		t.Fatalf("len(diags) = %d, want 3 (2 known + 1 for unknown)", len(diags))
	}

	// First two should be known fields in order
	if diags[0].InstancePath != "/policy/require_clean_before" {
		t.Errorf("diags[0].InstancePath = %q, want %q", diags[0].InstancePath, "/policy/require_clean_before")
	}
	if diags[0].Code != PlanCodeRequiredPropertyMissing {
		t.Errorf("diags[0].Code = %v, want %v", diags[0].Code, PlanCodeRequiredPropertyMissing)
	}

	if diags[1].InstancePath != "/policy/require_diff_check" {
		t.Errorf("diags[1].InstancePath = %q, want %q", diags[1].InstancePath, "/policy/require_diff_check")
	}
	if diags[1].Code != PlanCodeRequiredPropertyMissing {
		t.Errorf("diags[1].Code = %v, want %v", diags[1].Code, PlanCodeRequiredPropertyMissing)
	}

	// Third should be the unknown fields diagnostic
	unknownDiag := diags[2]
	if unknownDiag.Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("unknown diag Code = %v, want %v", unknownDiag.Code, PlanCodeSemanticConstraintFailed)
	}
	if unknownDiag.Keyword != KeywordType {
		t.Errorf("unknown diag Keyword = %v, want %v", unknownDiag.Keyword, KeywordType)
	}
	// Unknown fields should be sorted alphabetically
	if unknownDiag.InstancePath != "" {
		t.Errorf("unknown diag InstancePath = %q, want empty", unknownDiag.InstancePath)
	}
	// "another_unknown" comes before "unknown_field" alphabetically
	wantPropName := "another_unknown,unknown_field"
	if unknownDiag.PropertyName != wantPropName {
		t.Errorf("unknown diag PropertyName = %q, want %q", unknownDiag.PropertyName, wantPropName)
	}
	wantMsg := "unknown policy field(s): another_unknown, unknown_field"
	if unknownDiag.Message != wantMsg {
		t.Errorf("unknown diag Message = %q, want %q", unknownDiag.Message, wantMsg)
	}
}

// TestPlanPolicyRequiredErrorPlanDiagnostics20UnknownFields verifies that
// 20 unknown fields produce exactly 1 diagnostic with all fields.
func TestPlanPolicyRequiredErrorPlanDiagnostics20UnknownFields(t *testing.T) {
	unknowns := make([]string, 20)
	for i := 0; i < 20; i++ {
		unknowns[i] = "unknown_field_" + string(rune('a'+i))
	}
	err := &PlanPolicyRequiredError{Missing: unknowns}
	diags := err.PlanDiagnostics()

	if len(diags) != 1 {
		t.Fatalf("len(diags) = %d, want 1 (all unknown fields in one diagnostic)", len(diags))
	}

	diag := diags[0]
	if diag.Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
	}
	if diag.Keyword != KeywordType {
		t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordType)
	}

	// The fields should be sorted alphabetically
	sortedUnknowns := make([]string, 20)
	copy(sortedUnknowns, unknowns)
	slices.Sort(sortedUnknowns)

	// Build the expected PropertyName (comma-separated, sorted alphabetically)
	wantPropName := ""
	for i, f := range sortedUnknowns {
		if i > 0 {
			wantPropName += ","
		}
		wantPropName += f
	}
	if diag.PropertyName != wantPropName {
		t.Errorf("PropertyName = %q, want %q", diag.PropertyName, wantPropName)
	}
}

// TestPlanPolicyRequiredErrorPlanDiagnosticsAllFieldsMissing verifies that
// when all 4 policy fields are missing, exactly 4 diagnostics are returned.
func TestPlanPolicyRequiredErrorPlanDiagnosticsAllFieldsMissing(t *testing.T) {
	allMissing := []string{
		"require_clean_before",
		"require_clean_after",
		"forbid_tracked_full_digests",
		"require_diff_check",
	}
	err := &PlanPolicyRequiredError{Missing: allMissing}
	diags := err.PlanDiagnostics()

	if len(diags) != 4 {
		t.Fatalf("len(diags) = %d, want 4", len(diags))
	}

	// Verify each field has correct diagnostic properties
	wantFields := []struct {
		path     string
		propName string
		code     PlanValidationCode
		keyword  PlanValidationKeyword
	}{
		{"/policy/require_clean_before", "require_clean_before", PlanCodeRequiredPropertyMissing, KeywordRequired},
		{"/policy/require_clean_after", "require_clean_after", PlanCodeRequiredPropertyMissing, KeywordRequired},
		{"/policy/forbid_tracked_full_digests", "forbid_tracked_full_digests", PlanCodeRequiredPropertyMissing, KeywordRequired},
		{"/policy/require_diff_check", "require_diff_check", PlanCodeRequiredPropertyMissing, KeywordRequired},
	}

	for i, want := range wantFields {
		diag := diags[i]
		if diag.InstancePath != want.path {
			t.Errorf("diags[%d].InstancePath = %q, want %q", i, diag.InstancePath, want.path)
		}
		if diag.PropertyName != want.propName {
			t.Errorf("diags[%d].PropertyName = %q, want %q", i, diag.PropertyName, want.propName)
		}
		if diag.Code != want.code {
			t.Errorf("diags[%d].Code = %v, want %v", i, diag.Code, want.code)
		}
		if diag.Keyword != want.keyword {
			t.Errorf("diags[%d].Keyword = %v, want %v", i, diag.Keyword, want.keyword)
		}
		// SchemaPath should match InstancePath for policy errors
		if diag.SchemaPath != want.path {
			t.Errorf("diags[%d].SchemaPath = %q, want %q", i, diag.SchemaPath, want.path)
		}
	}
}

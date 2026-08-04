package closure

import (
	"errors"
	"testing"
)

// TestSemanticTaxonomyStaleEntries verifies that implementation doesn't have
// stale entries not in the documented taxonomy.
func TestSemanticTaxonomyStaleEntries(t *testing.T) {
	t.Run("NoStalePolicyFields", func(t *testing.T) {
		order := PolicyFieldOrder()
		validFields := map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		}

		for _, field := range order {
			if !validFields[field] {
				t.Errorf("PolicyFieldOrder contains stale field %q not in taxonomy", field)
			}
		}
	})

	t.Run("NoStaleRunnerAuthorityPlanFields", func(t *testing.T) {
		// These are the documented plan-declaration paths
		documentedPlanFields := map[string]string{
			"mode":                "/runner_authority/mode",
			"tool":                "/runner_authority/tool",
			"tool.revision":       "/runner_authority/tool/revision",
			"tool.tree_oid":       "/runner_authority/tool/tree_oid",
			"tool.binary_sha256":  "/runner_authority/tool/binary_sha256",
			"tool.version":        "/runner_authority/tool/version",
			"tool.tag_name":       "/runner_authority/tool/tag_name",
			"tool.tag_object_oid": "/runner_authority/tool/tag_object_oid",
		}

		for field, wantPath := range documentedPlanFields {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if isRuntime {
				t.Errorf("documented plan field %q incorrectly returned isRuntime=true", field)
				continue
			}
			if path != wantPath {
				t.Errorf("documented plan field %q: implementation path = %q, want %q", field, path, wantPath)
			}
		}
	})
}

// TestSemanticTaxonomyPathsFailOnDrift verifies that if the implementation
// changes paths, codes, or keywords without updating the taxonomy, tests fail.
func TestSemanticTaxonomyPathsFailOnDrift(t *testing.T) {
	t.Run("PolicyDiagnosticsUseCorrectPaths", func(t *testing.T) {
		err := &PlanPolicyRequiredError{
			Missing: []string{"require_clean_before"},
		}
		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}

		diag := diags[0]
		wantPath := "/policy/require_clean_before"
		if diag.InstancePath != wantPath {
			t.Errorf("InstancePath = %q, want %q", diag.InstancePath, wantPath)
		}
		if diag.Code != PlanCodeRequiredPropertyMissing {
			t.Errorf("Code = %v, want %v", diag.Code, PlanCodeRequiredPropertyMissing)
		}
		if diag.Keyword != KeywordRequired {
			t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordRequired)
		}
	})

	t.Run("ExecutionModeDiagnosticsUseCorrectPath", func(t *testing.T) {
		err := &ExecutionModeError{
			Path:      "/execution/mode",
			Value:     "invalid",
			Presence:  ExecutionModePresentUnknown,
			Supported: SupportedExecutionModes(),
		}
		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}

		diag := diags[0]
		wantPath := "/execution/mode"
		if diag.InstancePath != wantPath {
			t.Errorf("InstancePath = %q, want %q", diag.InstancePath, wantPath)
		}
		if diag.Code != PlanCodeSemanticConstraintFailed {
			t.Errorf("Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
		}
		if diag.Keyword != KeywordEnum {
			t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordEnum)
		}
	})

	t.Run("RunnerAuthorityPlanDiagnosticsUseCorrectPath", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "mode", Message: "unknown mode"}
		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}

		diag := diags[0]
		wantPath := "/runner_authority/mode"
		if diag.InstancePath != wantPath {
			t.Errorf("InstancePath = %q, want %q", diag.InstancePath, wantPath)
		}
		if diag.Code != PlanCodeSemanticConstraintFailed {
			t.Errorf("Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
		}
		if diag.Keyword != KeywordType {
			t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordType)
		}
	})

	t.Run("RunnerAuthorityToolDiagnosticsUseCorrectPath", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "tool.revision", Message: "revision required"}
		diags := err.PlanDiagnostics()

		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}

		diag := diags[0]
		wantPath := "/runner_authority/tool/revision"
		if diag.InstancePath != wantPath {
			t.Errorf("InstancePath = %q, want %q", diag.InstancePath, wantPath)
		}
	})
}

// TestSemanticTaxonomyRequirePredeterminedDigestRemoved verifies that
// require_predetermined_digest has been removed from the taxonomy per
// the directive to eliminate it from policy fields.
func TestSemanticTaxonomyRequirePredeterminedDigestRemoved(t *testing.T) {
	t.Run("RequirePredeterminedDigestNotInPolicyFields", func(t *testing.T) {
		order := PolicyFieldOrder()
		for _, field := range order {
			if field == "require_predetermined_digest" {
				t.Errorf("require_predetermined_digest still exists in PolicyFieldOrder; it should be removed")
			}
		}
	})

	t.Run("RequirePredeterminedDigestNotInContractDescriptor", func(t *testing.T) {
		// Verify the contract descriptor doesn't include require_predetermined_digest
		policyFields := []string{
			"require_clean_before",
			"require_clean_after",
			"forbid_tracked_full_digests",
			"require_diff_check",
		}

		// Check PolicyFieldOrder matches expected fields exactly
		order := PolicyFieldOrder()
		if len(order) != len(policyFields) {
			t.Fatalf("PolicyFieldOrder length = %d, want %d (require_predetermined_digest should not be present)",
				len(order), len(policyFields))
		}

		for i, expected := range policyFields {
			if got := order[i]; got != expected {
				t.Errorf("PolicyFieldOrder()[%d] = %q, want %q", i, got, expected)
			}
		}
	})
}

// TestSemanticTaxonomyValidatePlanIntegration verifies that ValidatePlan
// produces diagnostics that match the documented taxonomy.
func TestSemanticTaxonomyValidatePlanIntegration(t *testing.T) {
	t.Run("MissingPolicyFieldProducesDocumentedDiagnostic", func(t *testing.T) {
		plan := validSemanticPlanFixture(t)
		plan.Policy.RequireDiffCheck = nil // Remove required field

		err := ValidatePlan(plan)
		if err == nil {
			t.Fatal("ValidatePlan() = nil, want error for missing policy field")
		}

		var source planDiagnosticSource
		if !errors.As(err, &source) {
			t.Fatal("error does not implement planDiagnosticSource")
		}

		diags := source.PlanDiagnostics()
		found := false
		for _, diag := range diags {
			if diag.InstancePath == "/policy/require_diff_check" {
				found = true
				if diag.Code != PlanCodeRequiredPropertyMissing {
					t.Errorf("/policy/require_diff_check: Code = %v, want %v", diag.Code, PlanCodeRequiredPropertyMissing)
				}
				if diag.Keyword != KeywordRequired {
					t.Errorf("/policy/require_diff_check: Keyword = %v, want %v", diag.Keyword, KeywordRequired)
				}
				break
			}
		}
		if !found {
			t.Error("/policy/require_diff_check not found in diagnostics")
		}
	})

	t.Run("ActIDPlaceholderProducesDocumentedDiagnostic", func(t *testing.T) {
		plan := validSemanticPlanFixture(t)
		plan.ActID = "TODO" // Invalid placeholder

		err := ValidatePlan(plan)
		if err == nil {
			t.Fatal("ValidatePlan() = nil, want error for placeholder ACT ID")
		}

		var source planDiagnosticSource
		if !errors.As(err, &source) {
			t.Fatal("error does not implement planDiagnosticSource")
		}

		diags := source.PlanDiagnostics()
		found := false
		for _, diag := range diags {
			if diag.InstancePath == "/act_id" {
				found = true
				if diag.Code != PlanCodeSemanticConstraintFailed {
					t.Errorf("/act_id: Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
				}
				break
			}
		}
		if !found {
			t.Error("/act_id not found in diagnostics")
		}
	})
}

// TestSemanticTaxonomyDocumentationCompliance verifies that the implementation
// stays in sync with docs/factory/semantic-taxonomy.md.
func TestSemanticTaxonomyDocumentationCompliance(t *testing.T) {
	taxonomy := canonicalTaxonomy()

	// Check that all documented execution mode path exists
	t.Run("ExecutionModePathImplemented", func(t *testing.T) {
		for _, entry := range taxonomy {
			if entry.SemanticPath == "/execution/mode" {
				if entry.Code != PlanCodeSemanticConstraintFailed {
					t.Errorf("/execution/mode: documented code %v doesn't match implementation %v",
						entry.Code, PlanCodeSemanticConstraintFailed)
				}
				if entry.Keyword != KeywordEnum {
					t.Errorf("/execution/mode: documented keyword %v doesn't match implementation %v",
						entry.Keyword, KeywordEnum)
				}
				return
			}
		}
		t.Error("/execution/mode not found in taxonomy")
	})

	// Check that all documented check paths are covered
	t.Run("CheckPathsImplemented", func(t *testing.T) {
		checkPaths := []string{
			"/act_id",
			"/baseline/commit_oid",
			"/baseline/tree_oid",
		}

		for _, path := range checkPaths {
			found := false
			for _, entry := range taxonomy {
				if entry.SemanticPath == path {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("check path %q not found in taxonomy", path)
			}
		}
	})

	// Check that structural taxonomy entries are covered
	t.Run("StructuralPathsImplemented", func(t *testing.T) {
		structuralPaths := []struct {
			path    string
			code    PlanValidationCode
			keyword PlanValidationKeyword
		}{
			{"/checks", PlanCodeSemanticConstraintFailed, KeywordMinItems},
			{"/artifacts", PlanCodeSemanticConstraintFailed, KeywordMinItems},
		}

		for _, tc := range structuralPaths {
			found := false
			for _, entry := range taxonomy {
				if entry.SemanticPath == tc.path && entry.Code == tc.code {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("structural path %q with code %v not found in taxonomy", tc.path, tc.code)
			}
		}
	})

	// Verify runtime identities are marked correctly in implementation
	t.Run("RuntimeIdentitiesMarkedCorrectly", func(t *testing.T) {
		runtimeFields := []string{
			"vcs.revision",
			"vcs.modified",
			"binary_sha256",
			"target.subject",
			"target.tree",
		}

		for _, field := range runtimeFields {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if !isRuntime {
				t.Errorf("runtime field %q: isRuntime = false, want true", field)
			}
			if path != "" {
				t.Errorf("runtime field %q: path = %q, want empty string", field, path)
			}
		}
	})
}

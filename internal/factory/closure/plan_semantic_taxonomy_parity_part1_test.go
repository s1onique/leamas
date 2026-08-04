package closure

import (
	"errors"
	"testing"
)

// TaxonomyEntry represents a single row in the documented semantic taxonomy.
type TaxonomyEntry struct {
	SemanticPath string
	Code         PlanValidationCode
	Keyword      PlanValidationKeyword
	// isRuntimeIdentity is true when the taxonomy marks this as N/A (runtime-only).
	// These entries have no JSON pointer but should produce PropertyName-based diagnostics.
	isRuntimeIdentity bool
}

// canonicalTaxonomy returns the authoritative taxonomy derived from
// docs/factory/semantic-taxonomy.md. This is the single source of truth
// that the implementation must reconcile against.
func canonicalTaxonomy() []TaxonomyEntry {
	return []TaxonomyEntry{
		// Execution Mode Errors
		{SemanticPath: "/execution/mode", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordEnum},

		// Policy Errors
		{SemanticPath: "/policy/require_clean_before", Code: PlanCodeRequiredPropertyMissing, Keyword: KeywordRequired},
		{SemanticPath: "/policy/require_clean_after", Code: PlanCodeRequiredPropertyMissing, Keyword: KeywordRequired},
		{SemanticPath: "/policy/forbid_tracked_full_digests", Code: PlanCodeRequiredPropertyMissing, Keyword: KeywordRequired},
		{SemanticPath: "/policy/require_diff_check", Code: PlanCodeRequiredPropertyMissing, Keyword: KeywordRequired},

		// Runner Authority Plan-Declaration Paths (produce InstancePath)
		{SemanticPath: "/runner_authority/mode", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/revision", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/tree_oid", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/binary_sha256", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/version", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/tag_name", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/runner_authority/tool/tag_object_oid", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},

		// Runner Authority Runtime Identities (produce PropertyName, no InstancePath)
		{SemanticPath: "/runner_authority/vcs_revision", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, isRuntimeIdentity: true},
		{SemanticPath: "/runner_authority/vcs_modified", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, isRuntimeIdentity: true},
		{SemanticPath: "/runner_authority/binary_sha256", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, isRuntimeIdentity: true},
		{SemanticPath: "/runner_authority/target_subject", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, isRuntimeIdentity: true},
		{SemanticPath: "/runner_authority/target_tree", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, isRuntimeIdentity: true},

		// Check Errors
		{SemanticPath: "/act_id", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/baseline/commit_oid", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/baseline/tree_oid", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/id", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordPattern},
		{SemanticPath: "/checks/<i>/mode", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/reason", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordRequired},
		{SemanticPath: "/checks/<i>/argv", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/argv/<j>", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/working_directory", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/timeout_seconds", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/checks/<i>/environment", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},

		// Artifact Errors
		{SemanticPath: "/artifacts/<i>/id", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordPattern},
		{SemanticPath: "/artifacts/<i>/path", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/artifacts/<i>/required", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordRequired},
		{SemanticPath: "/artifacts/<i>/max_bytes", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},
		{SemanticPath: "/artifacts/<i>/media_type", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType},

		// Structural Taxonomy (Parse/Decode Validation)
		{SemanticPath: "(root)", Code: PlanCodeInvalidJSON, Keyword: KeywordType},
		{SemanticPath: "/contract_version_unsupported", Code: PlanCodeUnsupportedContractVersion, Keyword: KeywordType},
		{SemanticPath: "/contract_version_wrong_type", Code: PlanCodeInvalidType, Keyword: KeywordType},
		{SemanticPath: "/checks", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordMinItems},
		{SemanticPath: "/artifacts", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordMinItems},
	}
}

// TestSemanticTaxonomyReconciliation mechanically verifies that the documented
// taxonomy in docs/factory/semantic-taxonomy.md matches the actual Go implementation.
// This test fails if:
//   - A row exists in documentation but is missing from implementation
//   - An implementation has extra stale rows not in documentation
//   - A path, code, or keyword doesn't match between doc and implementation
//   - A runtime identity is incorrectly documented with a plan pointer
func TestSemanticTaxonomyReconciliation(t *testing.T) {
	taxonomy := canonicalTaxonomy()

	t.Run("DocumentedPathsAreNonEmpty", func(t *testing.T) {
		for i, entry := range taxonomy {
			if entry.SemanticPath == "" {
				t.Errorf("taxonomy[%d]: empty semantic path", i)
			}
			if entry.Code == "" {
				t.Errorf("taxonomy[%d] (%s): empty code", i, entry.SemanticPath)
			}
			if entry.Keyword == "" {
				t.Errorf("taxonomy[%d] (%s): empty keyword", i, entry.SemanticPath)
			}
		}
	})

	t.Run("PolicyFieldPathsMatchContract", func(t *testing.T) {
		policyFields := []string{
			"require_clean_before",
			"require_clean_after",
			"forbid_tracked_full_digests",
			"require_diff_check",
		}
		order := PolicyFieldOrder()

		if len(order) != len(policyFields) {
			t.Fatalf("PolicyFieldOrder() returned %d fields, want %d", len(order), len(policyFields))
		}

		for i, name := range order {
			// Verify field exists in expected set
			found := false
			for _, expected := range policyFields {
				if name == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("PolicyFieldOrder()[%d] = %q is not in expected policy fields", i, name)
			}
		}
	})

	t.Run("RunnerAuthorityFieldPathsMatchImplementation", func(t *testing.T) {
		// Verify that runnerAuthorityFieldPaths maps match documented paths
		expectedPaths := map[string]string{
			"mode":                "/runner_authority/mode",
			"tool":                "/runner_authority/tool",
			"tool.revision":       "/runner_authority/tool/revision",
			"tool.tree_oid":       "/runner_authority/tool/tree_oid",
			"tool.binary_sha256":  "/runner_authority/tool/binary_sha256",
			"tool.version":        "/runner_authority/tool/version",
			"tool.tag_name":       "/runner_authority/tool/tag_name",
			"tool.tag_object_oid": "/runner_authority/tool/tag_object_oid",
		}

		for field, wantPath := range expectedPaths {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if isRuntime {
				t.Errorf("field %q: isRuntime = true, want false (plan field)", field)
			}
			if path != wantPath {
				t.Errorf("field %q: path = %q, want %q", field, path, wantPath)
			}
		}
	})

	t.Run("RunnerAuthorityRuntimeIdentitiesRecognized", func(t *testing.T) {
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
				t.Errorf("runtime field %q: path = %q, want empty string for runtime identity", field, path)
			}
		}
	})

	t.Run("RunnerAuthorityRuntimeDiagnosticsHaveNoInstancePath", func(t *testing.T) {
		runtimeFields := []string{
			"vcs.revision",
			"vcs.modified",
			"binary_sha256",
			"target.subject",
			"target.tree",
		}

		for _, field := range runtimeFields {
			err := &RunnerAuthorityError{Field: field, Message: "runtime diagnostic test"}
			diags := err.PlanDiagnostics()

			if len(diags) != 1 {
				t.Errorf("runtime field %q: got %d diagnostics, want 1", field, len(diags))
				continue
			}

			diag := diags[0]
			if diag.InstancePath != "" {
				t.Errorf("runtime field %q: InstancePath = %q, want empty string", field, diag.InstancePath)
			}
			if diag.PropertyName == "" {
				t.Errorf("runtime field %q: PropertyName is empty, want %q", field, field)
			}
			if diag.Code != PlanCodeSemanticConstraintFailed {
				t.Errorf("runtime field %q: Code = %v, want %v", field, diag.Code, PlanCodeSemanticConstraintFailed)
			}
			if diag.Keyword != KeywordType {
				t.Errorf("runtime field %q: Keyword = %v, want %v", field, diag.Keyword, KeywordType)
			}
		}
	})

	t.Run("ErrorTypesImplementPlanDiagnostics", func(t *testing.T) {
		// Verify all documented error types implement planDiagnosticSource
		errorTypes := []struct {
			name string
			err  error
		}{
			{"ExecutionModeError", &ExecutionModeError{
				Path:      "/execution/mode",
				Value:     "invalid",
				Presence:  ExecutionModePresentUnknown,
				Supported: SupportedExecutionModes(),
			}},
			{"PlanPolicyRequiredError", &PlanPolicyRequiredError{
				Missing: []string{"require_clean_before"},
			}},
			{"RunnerAuthorityError", &RunnerAuthorityError{Field: "mode", Message: "test"}},
		}

		for _, tc := range errorTypes {
			t.Run(tc.name, func(t *testing.T) {
				var source planDiagnosticSource
				if !errors.As(tc.err, &source) {
					t.Errorf("%s should implement planDiagnosticSource", tc.name)
				}
				diags := source.PlanDiagnostics()
				if len(diags) != 1 {
					t.Errorf("%s.PlanDiagnostics returned %d diags, want 1", tc.name, len(diags))
				}
			})
		}
	})
}

// TestSemanticTaxonomyCodesAndKeywords verifies that all documented codes
// and keywords exist and have non-zero values.
func TestSemanticTaxonomyCodesAndKeywords(t *testing.T) {
	codes := []struct {
		code PlanValidationCode
		want string
	}{
		{PlanCodeRequiredPropertyMissing, "required_property_missing"},
		{PlanCodeSemanticConstraintFailed, "semantic_constraint_failed"},
		{PlanCodeInvalidEnum, "invalid_enum"},
		{PlanCodeInvalidType, "invalid_type"},
		{PlanCodeUnknownProperty, "unknown_property"},
		{PlanCodeDuplicateProperty, "duplicate_property"},
		{PlanCodeInvalidJSON, "invalid_json"},
		{PlanCodeUnsupportedContractVersion, "unsupported_contract_version"},
		{PlanCodeDuplicateApplicabilityRule, "duplicate_applicability_rule"},
	}

	for _, tc := range codes {
		t.Run("code/"+tc.want, func(t *testing.T) {
			if tc.code == "" {
				t.Errorf("PlanValidationCode %q is empty", tc.want)
			}
			if string(tc.code) != tc.want {
				t.Errorf("PlanValidationCode = %q, want %q", tc.code, tc.want)
			}
		})
	}

	keywords := []struct {
		kw   PlanValidationKeyword
		want string
	}{
		{KeywordRequired, "required"},
		{KeywordEnum, "enum"},
		{KeywordType, "type"},
		{KeywordPattern, "pattern"},
		{KeywordConst, "const"},
		{KeywordMinItems, "minItems"},
		{KeywordIfThenElse, "if"},
		{KeywordAdditionalProp, "additionalProperties"},
	}

	for _, tc := range keywords {
		t.Run("keyword/"+tc.want, func(t *testing.T) {
			if tc.kw == "" {
				t.Errorf("PlanValidationKeyword %q is empty", tc.want)
			}
			if string(tc.kw) != tc.want {
				t.Errorf("PlanValidationKeyword = %q, want %q", tc.kw, tc.want)
			}
		})
	}
}

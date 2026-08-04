package closure

import (
	"errors"
	"testing"
)

// TestRunnerAuthorityDiagnosticIdentityClosed verifies that the runner authority
// diagnostic identity function is closed: it returns known paths for known plan-declaration
// fields and a signal for runtime identities and unknown fields.
func TestRunnerAuthorityDiagnosticIdentityClosed(t *testing.T) {
	// Plan-declaration fields produce JSON pointer paths
	planFields := []string{
		"mode",
		"tool",
		"tool.revision",
		"tool.tree_oid",
		"tool.binary_sha256",
		"tool.version",
		"tool.tag_name",
		"tool.tag_object_oid",
	}

	for _, field := range planFields {
		t.Run("plan/"+field, func(t *testing.T) {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if isRuntime {
				t.Errorf("plan field %q: isRuntime = true, want false", field)
			}
			if path == "" {
				t.Errorf("plan field %q: path is empty, want non-empty JSON pointer", field)
			}
		})
	}

	// Runtime identities produce no path (empty string, isRuntime=true)
	runtimeFields := []string{
		"vcs.revision",
		"vcs.modified",
		"binary_sha256",
		"target.subject",
		"target.tree",
	}

	for _, field := range runtimeFields {
		t.Run("runtime/"+field, func(t *testing.T) {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if !isRuntime {
				t.Errorf("runtime field %q: isRuntime = false, want true", field)
			}
			if path != "" {
				t.Errorf("runtime field %q: path = %q, want empty string", field, path)
			}
		})
	}

	// Unknown fields produce empty path and isRuntime=false
	unknownFields := []string{
		"unknown_field",
		"misspelled",
		"vcsrevision",  // missing dot
		"toolrevision", // missing dot
		"mode_extra",
		"modexecution",
		"TOOL.REVISION", // uppercase
	}

	for _, field := range unknownFields {
		t.Run("unknown/"+field, func(t *testing.T) {
			path, isRuntime := runnerAuthorityDiagnosticIdentity(field)
			if isRuntime {
				t.Errorf("unknown field %q: isRuntime = true, want false", field)
			}
			if path != "" {
				t.Errorf("unknown field %q: path = %q, want empty string", field, path)
			}
		})
	}
}

// TestRunnerAuthorityErrorPlanDiagnostics verifies that RunnerAuthorityError
// produces planDiagnosticSource with typed diagnostics.
func TestRunnerAuthorityErrorPlanDiagnostics(t *testing.T) {
	cases := []struct {
		field        string
		wantPath     string
		wantKnown    bool // true for plan fields, false for runtime/unknown
		wantPropName string
	}{
		// Plan-declaration fields produce InstancePath
		{"mode", "/runner_authority/mode", true, ""},
		{"tool", "/runner_authority/tool", true, ""},
		{"tool.revision", "/runner_authority/tool/revision", true, ""},
		{"tool.tree_oid", "/runner_authority/tool/tree_oid", true, ""},
		{"tool.binary_sha256", "/runner_authority/tool/binary_sha256", true, ""},
		{"tool.version", "/runner_authority/tool/version", true, ""},
		{"tool.tag_name", "/runner_authority/tool/tag_name", true, ""},
		{"tool.tag_object_oid", "/runner_authority/tool/tag_object_oid", true, ""},

		// Runtime identities produce PropertyName, no InstancePath
		{"vcs.revision", "", false, "vcs.revision"},
		{"vcs.modified", "", false, "vcs.modified"},
		{"binary_sha256", "", false, "binary_sha256"},
		{"target.subject", "", false, "target.subject"},
		{"target.tree", "", false, "target.tree"},

		// Unknown fields produce PropertyName, no InstancePath
		{"unknown_field", "", false, "unknown_field"},
		{"misspelled", "", false, "misspelled"},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			err := &RunnerAuthorityError{Field: tc.field, Message: "test error"}
			diags := err.PlanDiagnostics()

			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics, want 1", len(diags))
			}

			diag := diags[0]

			if tc.wantKnown {
				// Plan field: expect InstancePath
				if diag.InstancePath != tc.wantPath {
					t.Errorf("InstancePath = %q, want %q", diag.InstancePath, tc.wantPath)
				}
				if diag.PropertyName != "" {
					t.Errorf("PropertyName = %q, want empty for plan field", diag.PropertyName)
				}
			} else {
				// Runtime or unknown: expect no InstancePath, expect PropertyName
				if diag.InstancePath != "" {
					t.Errorf("InstancePath = %q, want empty for %s field", diag.InstancePath, tc.field)
				}
				if diag.PropertyName != tc.wantPropName {
					t.Errorf("PropertyName = %q, want %q", diag.PropertyName, tc.wantPropName)
				}
			}

			if diag.Code != PlanCodeSemanticConstraintFailed {
				t.Errorf("Code = %v, want %v", diag.Code, PlanCodeSemanticConstraintFailed)
			}
			if diag.Keyword != KeywordType {
				t.Errorf("Keyword = %v, want %v", diag.Keyword, KeywordType)
			}

			// Verify it implements planDiagnosticSource
			var source planDiagnosticSource
			if !errors.As(err, &source) {
				t.Error("RunnerAuthorityError should implement planDiagnosticSource")
			}
		})
	}
}

// TestRunnerAuthorityErrorUnwrap verifies error unwrapping.
func TestRunnerAuthorityErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &RunnerAuthorityError{Field: "mode", Message: "test", Cause: cause}
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) should be true")
	}
	var runner *RunnerAuthorityError
	if !errors.As(err, &runner) {
		t.Error("errors.As should extract RunnerAuthorityError")
	}
}

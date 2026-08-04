package closure

import (
	"testing"
)

// TestJSONPointerHelpers tests the RFC 6901 JSON Pointer helpers.
func TestJSONPointerHelpers(t *testing.T) {
	t.Run("jsonPointer empty", func(t *testing.T) {
		got := jsonPointer()
		if got != "" {
			t.Errorf("jsonPointer() = %q, want %q", got, "")
		}
	})

	t.Run("jsonPointer single segment", func(t *testing.T) {
		got := jsonPointer("foo")
		if got != "/foo" {
			t.Errorf("jsonPointer(\"foo\") = %q, want %q", got, "/foo")
		}
	})

	t.Run("jsonPointer multiple segments", func(t *testing.T) {
		got := jsonPointer("foo", "bar", "baz")
		if got != "/foo/bar/baz" {
			t.Errorf("jsonPointer(\"foo\", \"bar\", \"baz\") = %q, want %q", got, "/foo/bar/baz")
		}
	})

	t.Run("jsonPointerIndex positive", func(t *testing.T) {
		got := jsonPointerIndex("checks", 0)
		if got != "/checks/0" {
			t.Errorf("jsonPointerIndex(\"checks\", 0) = %q, want %q", got, "/checks/0")
		}
	})

	t.Run("jsonPointerIndex large index", func(t *testing.T) {
		got := jsonPointerIndex("artifacts", 99)
		if got != "/artifacts/99" {
			t.Errorf("jsonPointerIndex(\"artifacts\", 99) = %q, want %q", got, "/artifacts/99")
		}
	})

	t.Run("jsonPointerIndex negative fails closed", func(t *testing.T) {
		got := jsonPointerIndex("checks", -1)
		if got != "" {
			t.Errorf("jsonPointerIndex(\"checks\", -1) = %q, want %q (fail closed)", got, "")
		}
	})

	t.Run("jsonPointerCheckID", func(t *testing.T) {
		got := jsonPointerCheckID(0, "id")
		if got != "/checks/0/id" {
			t.Errorf("jsonPointerCheckID(0, \"id\") = %q, want %q", got, "/checks/0/id")
		}
	})

	t.Run("jsonPointerArtifactID", func(t *testing.T) {
		got := jsonPointerArtifactID(1, "path")
		if got != "/artifacts/1/path" {
			t.Errorf("jsonPointerArtifactID(1, \"path\") = %q, want %q", got, "/artifacts/1/path")
		}
	})

	t.Run("jsonPointerArgvElement", func(t *testing.T) {
		got := jsonPointerArgvElement(0, 0)
		if got != "/checks/0/argv/0" {
			t.Errorf("jsonPointerArgvElement(0, 0) = %q, want %q", got, "/checks/0/argv/0")
		}
	})
}

// TestJSONPointerToken tests RFC 6901 token encoding.
func TestJSONPointerToken(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo/bar", "foo~1bar"},
		{"foo~bar", "foo~0bar"},
		{"foo/bar~baz", "foo~1bar~0baz"},
		{"a/b~c/d", "a~1b~0c~1d"},
		{"", ""},
		{"~", "~0"},
		{"/", "~1"},
		{"~1", "~01"},
		{"~0", "~00"},
		// Note: "a~1b~0c" encodes both ~ chars per RFC 6901.
		// ~ → ~0, / → ~1. No / in this string, so only ~ is encoded.
		{"a~1b~0c", "a~01b~00c"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := jsonPointerToken(c.input)
			if got != c.expected {
				t.Errorf("jsonPointerToken(%q) = %q, want %q", c.input, got, c.expected)
			}
		})
	}
}

// TestSemanticPathMatrix tests that every semantic error type produces
// the expected InstancePath for every known error path in the taxonomy.
func TestSemanticPathMatrix(t *testing.T) {
	// ExecutionModeError paths
	t.Run("ExecutionModeError PlanDiagnostics", func(t *testing.T) {
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
		if diags[0].InstancePath != "/execution/mode" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/execution/mode")
		}
		if diags[0].Code != PlanCodeSemanticConstraintFailed {
			t.Errorf("Code = %v, want %v", diags[0].Code, PlanCodeSemanticConstraintFailed)
		}
		if diags[0].Keyword != KeywordEnum {
			t.Errorf("Keyword = %v, want %v", diags[0].Keyword, KeywordEnum)
		}
	})

	// PlanPolicyRequiredError paths
	t.Run("PlanPolicyRequiredError PlanDiagnostics", func(t *testing.T) {
		err := &PlanPolicyRequiredError{
			Missing: []string{"require_clean_before", "require_diff_check"},
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 2 {
			t.Fatalf("got %d diagnostics, want 2", len(diags))
		}
		wantPaths := []string{"/policy/require_clean_before", "/policy/require_diff_check"}
		for i, want := range wantPaths {
			if diags[i].InstancePath != want {
				t.Errorf("diagnostic[%d].InstancePath = %q, want %q", i, diags[i].InstancePath, want)
			}
		}
	})

	// RunnerAuthorityError paths
	t.Run("RunnerAuthorityError PlanDiagnostics mode", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "mode",
			Message: "unknown mode",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/mode" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/mode")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics tool.revision", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "tool.revision",
			Message: "revision is required",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/tool/revision" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/tool/revision")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics tool.binary_sha256", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "tool.binary_sha256",
			Message: "binary_sha256 is required",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/tool/binary_sha256" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/tool/binary_sha256")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics binary_sha256", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "binary_sha256",
			Message: "binary_sha256 mismatch",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/binary_sha256" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/binary_sha256")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics vcs.revision", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "vcs.revision",
			Message: "vcs.revision mismatch",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/vcs_revision" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/vcs_revision")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics vcs.modified", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "vcs.modified",
			Message: "modified sources",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/vcs_modified" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/vcs_modified")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics target.subject", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "target.subject",
			Message: "target subject empty",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/target_subject" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/target_subject")
		}
	})

	t.Run("RunnerAuthorityError PlanDiagnostics target.tree", func(t *testing.T) {
		err := &RunnerAuthorityError{
			Field:   "target.tree",
			Message: "target tree empty",
		}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/target_tree" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/target_tree")
		}
	})

	// PlanSemanticError paths
	t.Run("PlanSemanticError PlanDiagnostics", func(t *testing.T) {
		err := newSemanticError(
			"/checks/0/id",
			PlanCodeSemanticConstraintFailed,
			KeywordPattern,
			"invalid check id",
			nil,
		)
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/checks/0/id" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/checks/0/id")
		}
		if diags[0].Code != PlanCodeSemanticConstraintFailed {
			t.Errorf("Code = %v, want %v", diags[0].Code, PlanCodeSemanticConstraintFailed)
		}
		if diags[0].Keyword != KeywordPattern {
			t.Errorf("Keyword = %v, want %v", diags[0].Keyword, KeywordPattern)
		}
	})

	// PlanSemanticMultiError paths
	t.Run("PlanSemanticMultiError PlanDiagnostics", func(t *testing.T) {
		err := newSemanticMultiError([]PlanValidationError{
			{
				InstancePath: "/checks/0/id",
				Code:        PlanCodeSemanticConstraintFailed,
				Keyword:     KeywordPattern,
				Message:     "invalid check id",
			},
			{
				InstancePath: "/checks/1/mode",
				Code:        PlanCodeSemanticConstraintFailed,
				Keyword:     KeywordType,
				Message:     "unknown check mode",
			},
		}, nil)
		diags := err.PlanDiagnostics()
		if len(diags) != 2 {
			t.Fatalf("got %d diagnostics, want 2", len(diags))
		}
		if diags[0].InstancePath != "/checks/0/id" {
			t.Errorf("diagnostic[0].InstancePath = %q, want %q", diags[0].InstancePath, "/checks/0/id")
		}
		if diags[1].InstancePath != "/checks/1/mode" {
			t.Errorf("diagnostic[1].InstancePath = %q, want %q", diags[1].InstancePath, "/checks/1/mode")
		}
	})
}

// TestSemanticDiagnosticsNil tests that semanticDiagnostics handles nil.
func TestSemanticDiagnosticsNil(t *testing.T) {
	diags := semanticDiagnostics(nil)
	if diags == nil {
		t.Error("semanticDiagnostics(nil) returned nil, want empty slice")
	}
	if len(diags) != 0 {
		t.Errorf("semanticDiagnostics(nil) returned %d diagnostics, want 0", len(diags))
	}
}

// TestSemanticDiagnosticsUnknownError tests that unknown errors get a root fallback.
func TestSemanticDiagnosticsUnknownError(t *testing.T) {
	err := &unknownError{msg: "something went wrong"}
	diags := semanticDiagnostics(err)
	if len(diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diags))
	}
	if diags[0].InstancePath != "" {
		t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "")
	}
	if diags[0].Code != PlanCodeSemanticConstraintFailed {
		t.Errorf("Code = %v, want %v", diags[0].Code, PlanCodeSemanticConstraintFailed)
	}
	if diags[0].Message != "something went wrong" {
		t.Errorf("Message = %q, want %q", diags[0].Message, "something went wrong")
	}
}

// unknownError is a test helper that implements error but not planDiagnosticSource.
type unknownError struct {
	msg string
}

func (e *unknownError) Error() string {
	return e.msg
}

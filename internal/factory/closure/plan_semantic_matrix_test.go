package closure

import (
	"errors"
	"fmt"
	"testing"
)

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
	t.Run("RunnerAuthorityError mode", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "mode", Message: "unknown mode"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/mode" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/mode")
		}
	})

	t.Run("RunnerAuthorityError tool.revision", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "tool.revision", Message: "revision required"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/tool/revision" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/runner_authority/tool/revision")
		}
	})

	t.Run("RunnerAuthorityError tool.binary_sha256", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "tool.binary_sha256", Message: "sha256 required"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/tool/binary_sha256" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/tool/binary_sha256")
		}
	})

	t.Run("RunnerAuthorityError binary_sha256", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "binary_sha256", Message: "mismatch"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/binary_sha256" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/binary_sha256")
		}
	})

	t.Run("RunnerAuthorityError vcs.revision", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "vcs.revision", Message: "mismatch"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/vcs_revision" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/vcs_revision")
		}
	})

	t.Run("RunnerAuthorityError vcs.modified", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "vcs.modified", Message: "modified"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/vcs_modified" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/vcs_modified")
		}
	})

	t.Run("RunnerAuthorityError target.subject", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "target.subject", Message: "empty"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/target_subject" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/target_subject")
		}
	})

	t.Run("RunnerAuthorityError target.tree", func(t *testing.T) {
		err := &RunnerAuthorityError{Field: "target.tree", Message: "empty"}
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/runner_authority/target_tree" {
			t.Errorf("got %q, want %q", diags[0].InstancePath, "/runner_authority/target_tree")
		}
	})

	// PlanSemanticError paths
	t.Run("PlanSemanticError PlanDiagnostics", func(t *testing.T) {
		err := newSemanticError("/checks/0/id", PlanCodeSemanticConstraintFailed, KeywordPattern, "invalid", nil)
		diags := err.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/checks/0/id" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/checks/0/id")
		}
	})

	// PlanSemanticMultiError paths
	t.Run("PlanSemanticMultiError PlanDiagnostics", func(t *testing.T) {
		err := newSemanticMultiError([]PlanValidationError{
			{InstancePath: "/checks/0/id", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordPattern, Message: "invalid"},
			{InstancePath: "/checks/1/mode", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, Message: "unknown"},
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

func (e *unknownError) Error() string { return e.msg }

// TestErrorsIs tests errors.Is through every Cause-bearing type.
func TestErrorsIs(t *testing.T) {
	t.Run("PlanSemanticError Unwrap", func(t *testing.T) {
		cause := &unknownError{msg: "underlying error"}
		err := newSemanticError("/foo", PlanCodeSemanticConstraintFailed, KeywordType, "test", cause)
		if !errors.Is(err, cause) {
			t.Error("errors.Is(err, cause) = false, want true")
		}
	})

	t.Run("PlanSemanticMultiError Unwrap", func(t *testing.T) {
		cause := &unknownError{msg: "underlying error"}
		err := newSemanticMultiError([]PlanValidationError{{InstancePath: "/foo", Code: PlanCodeSemanticConstraintFailed, Keyword: KeywordType, Message: "test"}}, cause)
		if !errors.Is(err, cause) {
			t.Error("errors.Is(err, cause) = false, want true")
		}
	})

	t.Run("RunnerAuthorityError Unwrap", func(t *testing.T) {
		cause := &unknownError{msg: "underlying error"}
		err := &RunnerAuthorityError{Field: "mode", Message: "test", Cause: cause}
		if !errors.Is(err, cause) {
			t.Error("errors.Is(err, cause) = false, want true")
		}
	})

	t.Run("ExecutionModeError no cause", func(t *testing.T) {
		err := &ExecutionModeError{Path: "/mode", Value: "bad", Presence: ExecutionModePresentUnknown, Supported: SupportedExecutionModes()}
		if errors.Is(err, errors.New("anything")) {
			t.Error("errors.Is(err, anything) = true, want false (no cause)")
		}
	})
}

// TestErrorsAs tests errors.As through wrapping.
func TestErrorsAs(t *testing.T) {
	t.Run("wrapped PlanSemanticError", func(t *testing.T) {
		inner := newSemanticError("/foo", PlanCodeSemanticConstraintFailed, KeywordType, "inner", nil)
		wrapper := fmt.Errorf("wrapper: %w", inner)
		var semantic *PlanSemanticError
		if !errors.As(wrapper, &semantic) {
			t.Error("errors.As(wrapper, &semantic) = false, want true")
		}
		if semantic.Diagnostic.InstancePath != "/foo" {
			t.Errorf("InstancePath = %q, want %q", semantic.Diagnostic.InstancePath, "/foo")
		}
	})

	t.Run("wrapped RunnerAuthorityError", func(t *testing.T) {
		inner := &RunnerAuthorityError{Field: "mode", Message: "inner"}
		wrapper := fmt.Errorf("wrapper: %w", inner)
		var runner *RunnerAuthorityError
		if !errors.As(wrapper, &runner) {
			t.Error("errors.As(wrapper, &runner) = false, want true")
		}
		if runner.Field != "mode" {
			t.Errorf("Field = %q, want %q", runner.Field, "mode")
		}
	})

	t.Run("wrapped ExecutionModeError", func(t *testing.T) {
		inner := &ExecutionModeError{Path: "/mode", Value: "bad", Presence: ExecutionModePresentUnknown, Supported: SupportedExecutionModes()}
		wrapper := fmt.Errorf("wrapper: %w", inner)
		var exec *ExecutionModeError
		if !errors.As(wrapper, &exec) {
			t.Error("errors.As(wrapper, &exec) = false, want true")
		}
		if exec.Presence != ExecutionModePresentUnknown {
			t.Errorf("Presence = %v, want %v", exec.Presence, ExecutionModePresentUnknown)
		}
	})
}

// TestAcceptedValuesMutationIsolation tests AcceptedValues mutation isolation.
func TestAcceptedValuesMutationIsolation(t *testing.T) {
	t.Run("ExecutionModeError AcceptedValues isolation", func(t *testing.T) {
		orig := &ExecutionModeError{Path: "/mode", Value: "bad", Presence: ExecutionModePresentUnknown, Supported: SupportedExecutionModes()}
		diags := orig.PlanDiagnostics()
		diags[0].AcceptedValues[0] = "hacked"
		diags2 := orig.PlanDiagnostics()
		if diags2[0].AcceptedValues[0] == "hacked" {
			t.Error("AcceptedValues mutation leaked through to original")
		}
	})

	t.Run("clonePlanValidationError AcceptedValues isolation", func(t *testing.T) {
		orig := PlanValidationError{
			InstancePath:   "/foo",
			Code:           PlanCodeSemanticConstraintFailed,
			Keyword:        KeywordEnum,
			AcceptedValues: []string{"a", "b", "c"},
		}
		cloned := clonePlanValidationError(orig)
		cloned.AcceptedValues[0] = "hacked"
		if orig.AcceptedValues[0] == "hacked" {
			t.Error("AcceptedValues mutation leaked through to original")
		}
	})
}

// TestWrappedPlanDiagnosticSource tests errors.As for planDiagnosticSource.
func TestWrappedPlanDiagnosticSource(t *testing.T) {
	t.Run("wrapped source preserves PlanDiagnostics", func(t *testing.T) {
		orig := &ExecutionModeError{Path: "/execution/mode", Value: "bad", Presence: ExecutionModePresentUnknown, Supported: SupportedExecutionModes()}
		wrapper := fmt.Errorf("validation failed: %w", orig)
		var source planDiagnosticSource
		if !errors.As(wrapper, &source) {
			t.Error("errors.As(wrapper, &source) = false, want true")
		}
		diags := source.PlanDiagnostics()
		if len(diags) != 1 {
			t.Fatalf("got %d diagnostics, want 1", len(diags))
		}
		if diags[0].InstancePath != "/execution/mode" {
			t.Errorf("InstancePath = %q, want %q", diags[0].InstancePath, "/execution/mode")
		}
	})
}

// TestNilDiagnosticSourceOutput tests that nil or empty outputs become [].
func TestNilDiagnosticSourceOutput(t *testing.T) {
	t.Run("PlanSemanticMultiError empty diagnostics", func(t *testing.T) {
		err := newSemanticMultiError(nil, nil)
		diags := err.PlanDiagnostics()
		if diags == nil {
			t.Error("PlanDiagnostics() returned nil, want non-nil empty slice")
		}
		if len(diags) != 0 {
			t.Errorf("len(diags) = %d, want 0", len(diags))
		}
	})

	t.Run("clonePlanValidationErrors nil input", func(t *testing.T) {
		result := clonePlanValidationErrors(nil)
		if result == nil {
			t.Error("clonePlanValidationErrors(nil) returned nil, want non-nil empty slice")
		}
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})

	t.Run("semanticDiagnostics nil", func(t *testing.T) {
		diags := semanticDiagnostics(nil)
		if diags == nil {
			t.Error("semanticDiagnostics(nil) returned nil, want non-nil empty slice")
		}
		if len(diags) != 0 {
			t.Errorf("len(diags) = %d, want 0", len(diags))
		}
	})
}

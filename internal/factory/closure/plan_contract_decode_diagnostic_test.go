package closure

import (
	"errors"
	"strings"
	"testing"
)

// TestTypedDecodeDiagnostic pins the decode-stage diagnostic
// taxonomy produced by typedDecodeDiagnostic. The function is
// the unit-level boundary between a typed-decode error and the
// composed result's decode_errors array. The unit proof does
// not require a real typed-decode seam; it documents the
// stable shape so future seams can be tested against it.
func TestTypedDecodeDiagnostic(t *testing.T) {
	sentinel := errors.New("typed decode: boom")
	diag := typedDecodeDiagnostic(sentinel)
	if diag.Code != PlanCodeInvalidType {
		t.Fatalf("Code = %v, want %v",
			diag.Code, PlanCodeInvalidType)
	}
	if diag.Keyword != KeywordType {
		t.Fatalf("Keyword = %v, want %v",
			diag.Keyword, KeywordType)
	}
	if diag.InstancePath != "" {
		t.Fatalf("InstancePath = %q, want empty",
			diag.InstancePath)
	}
	if diag.SchemaPath != "" {
		t.Fatalf("SchemaPath = %q, want empty",
			diag.SchemaPath)
	}
	if !strings.Contains(diag.Message, "boom") {
		t.Fatalf("Message = %q, must contain sentinel",
			diag.Message)
	}
}

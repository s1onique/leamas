package gate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGateSummary_Missing(t *testing.T) {
	result := RenderGateSummary(nil, nil)

	if !strings.Contains(result, "source_status=missing") {
		t.Errorf("expected source_status=missing, got:\n%s", result)
	}
	if !strings.Contains(result, "overall_status=unavailable") {
		t.Errorf("expected overall_status=unavailable, got:\n%s", result)
	}
	if strings.Contains(result, "diagnostics:") {
		t.Errorf("expected no diagnostics for missing, got:\n%s", result)
	}
}

func TestRenderGateSummary_Invalid(t *testing.T) {
	testErr := errors.New("json: invalid character '}' looking for beginning of value")
	result := RenderGateSummary(nil, testErr)

	if !strings.Contains(result, "source_status=invalid") {
		t.Errorf("expected source_status=invalid, got:\n%s", result)
	}
	if !strings.Contains(result, "overall_status=unavailable") {
		t.Errorf("expected overall_status=unavailable, got:\n%s", result)
	}
	if !strings.Contains(result, "diagnostics:") {
		t.Errorf("expected diagnostics for invalid, got:\n%s", result)
	}
	// Error message should be sanitized (no newlines)
	if strings.Contains(result, "\n") {
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "  -") {
				if strings.Count(line, "\n") > 0 {
					t.Errorf("diagnostics line should not contain newlines, got: %q", line)
				}
			}
		}
	}
}

func TestRenderGateSummary_Present(t *testing.T) {
	summary := &GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-10T00:00:00Z",
		OverallStatus: "pass",
		Checks: []Check{
			{Name: "go_test", Status: CheckStatusPass, DurationMs: 100, Evidence: "go test ./..."},
			{Name: "go_vet", Status: CheckStatusPass, DurationMs: 50, Evidence: "go vet ./..."},
		},
	}

	result := RenderGateSummary(summary, nil)

	if !strings.Contains(result, "source_status=present") {
		t.Errorf("expected source_status=present, got:\n%s", result)
	}
	if !strings.Contains(result, "schema_version=1") {
		t.Errorf("expected schema_version=1, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_total=2") {
		t.Errorf("expected checks_total=2, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_passed=2") {
		t.Errorf("expected checks_passed=2, got:\n%s", result)
	}
}

func TestSanitizeString_Newlines(t *testing.T) {
	input := "line1\nline2\r\nline3"
	result := sanitizeString(input)

	if strings.Contains(result, "\n") || strings.Contains(result, "\r") {
		t.Errorf("sanitized string should not contain newlines, got: %q", result)
	}
}

func TestSanitizeString_MaxLength(t *testing.T) {
	// Create a string longer than MaxEvidenceLength
	input := strings.Repeat("a", MaxEvidenceLength+100)
	result := sanitizeString(input)

	if len(result) > MaxEvidenceLength {
		t.Errorf("sanitized string should be truncated to %d, got %d", MaxEvidenceLength, len(result))
	}
}

func TestSanitizeString_Whitespace(t *testing.T) {
	input := "  multiple   spaces   here  "
	result := sanitizeString(input)

	// Should not have leading/trailing whitespace
	if result != strings.TrimSpace(result) {
		t.Errorf("sanitized string should be trimmed, got: %q", result)
	}
	// Should not have multiple spaces
	if strings.Contains(result, "  ") {
		t.Errorf("sanitized string should not have multiple spaces, got: %q", result)
	}
}

func TestStatusCounts(t *testing.T) {
	summary := &GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-10T00:00:00Z",
		OverallStatus: "fail",
		Checks: []Check{
			{Name: "pass_test", Status: CheckStatusPass},
			{Name: "fail_test", Status: CheckStatusFail},
			{Name: "skip_test", Status: CheckStatusSkip},
			{Name: "unavail_test", Status: CheckStatusUnavailable},
		},
	}

	result := RenderGateSummary(summary, nil)

	if !strings.Contains(result, "checks_total=4") {
		t.Errorf("expected checks_total=4, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_passed=1") {
		t.Errorf("expected checks_passed=1, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_failed=1") {
		t.Errorf("expected checks_failed=1, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_skipped=1") {
		t.Errorf("expected checks_skipped=1, got:\n%s", result)
	}
	if !strings.Contains(result, "checks_unavailable=1") {
		t.Errorf("expected checks_unavailable=1, got:\n%s", result)
	}
}

func TestWriteGateSummary_CreatesParentDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write to a path with a non-existent parent directory
	outputPath := filepath.Join(tmpDir, "subdir", "gate-summary.json")
	err = WriteGateSummary(tmpDir, outputPath)
	if err != nil {
		t.Errorf("WriteGateSummary should create parent directory: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("WriteGateSummary should create the file")
	}
}

func TestGateSummaryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	existingPath := filepath.Join(tmpDir, "exists.json")
	nonExistingPath := filepath.Join(tmpDir, "not-exists.json")

	// Create existing file
	if err := os.WriteFile(existingPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if !GateSummaryExists(existingPath) {
		t.Errorf("GateSummaryExists should return true for existing file")
	}
	if GateSummaryExists(nonExistingPath) {
		t.Errorf("GateSummaryExists should return false for non-existing file")
	}
}

func TestParseGateSummaryStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"pass", "overall_status=pass\n", "pass"},
		{"fail", "overall_status=fail\n", "fail"},
		{"unavailable", "overall_status=unavailable\n", "unavailable"},
		{"empty", "", "unavailable"},
		{"no_match", "source=foo\n", "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGateSummaryStatus(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseGateSummarySourceStatus(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"present", "source_status=present\n", "present"},
		{"missing", "source_status=missing\n", "missing"},
		{"invalid", "source_status=invalid\n", "invalid"},
		{"empty", "", "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseGateSummarySourceStatus(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestReadGateSummary_InvalidSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid schema version
	invalidPath := filepath.Join(tmpDir, "invalid-schema.json")
	invalidJSON := `{"schema_version": 99, "generated_at": "2026-07-10T00:00:00Z", "overall_status": "pass", "checks": []}`
	if err := os.WriteFile(invalidPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = ReadGateSummary(invalidPath)
	if err == nil {
		t.Errorf("ReadGateSummary should return error for invalid schema version")
	}
}

func TestReadGateSummary_InvalidStatus(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write invalid check status - should be sanitized to unavailable
	invalidPath := filepath.Join(tmpDir, "invalid-status.json")
	invalidJSON := `{"schema_version": 1, "generated_at": "2026-07-10T00:00:00Z", "overall_status": "pass", "checks": [{"name": "test", "status": "invalid_status"}]}`
	if err := os.WriteFile(invalidPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	summary, err := ReadGateSummary(invalidPath)
	if err != nil {
		t.Errorf("ReadGateSummary should normalize invalid status, not error: %v", err)
	}
	if summary != nil && summary.Checks[0].Status != CheckStatusUnavailable {
		t.Errorf("expected status to be normalized to unavailable, got %s", summary.Checks[0].Status)
	}
}

func TestReadGateSummary_Valid(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gate-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write valid gate summary
	validPath := filepath.Join(tmpDir, "valid.json")
	validJSON := `{
		"schema_version": 1,
		"generated_at": "2026-07-10T00:00:00Z",
		"tool": "test",
		"overall_status": "pass",
		"checks": [
			{"name": "go_test", "status": "pass", "duration_ms": 100, "evidence": "go test ./..."}
		]
	}`
	if err := os.WriteFile(validPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	summary, err := ReadGateSummary(validPath)
	if err != nil {
		t.Errorf("ReadGateSummary should not return error for valid file: %v", err)
	}
	if summary.SchemaVersion != 1 {
		t.Errorf("expected schema version 1, got %d", summary.SchemaVersion)
	}
	if len(summary.Checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(summary.Checks))
	}
}

func TestDeterministicCheckOrder(t *testing.T) {
	summary := &GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   "2026-07-10T00:00:00Z",
		OverallStatus: "pass",
		Checks: []Check{
			{Name: "zebra", Status: CheckStatusPass},
			{Name: "apple", Status: CheckStatusPass},
			{Name: "middle", Status: CheckStatusPass},
		},
	}

	result1 := RenderGateSummary(summary, nil)
	result2 := RenderGateSummary(summary, nil)

	// Results should be identical
	if result1 != result2 {
		t.Errorf("RenderGateSummary should produce deterministic output")
	}

	// Check for name presence - use longer unique strings
	if !strings.Contains(result1, "name=apple status=pass") {
		t.Fatalf("expected apple in output: %s", result1)
	}
	if !strings.Contains(result1, "name=middle status=pass") {
		t.Fatalf("expected middle in output: %s", result1)
	}
	if !strings.Contains(result1, "name=zebra status=pass") {
		t.Fatalf("expected zebra in output: %s", result1)
	}

	// Find positions of each name
	appleIdx := strings.Index(result1, "name=apple status=pass")
	middleIdx := strings.Index(result1, "name=middle status=pass")
	zebraIdx := strings.Index(result1, "name=zebra status=pass")

	if appleIdx == -1 || middleIdx == -1 || zebraIdx == -1 {
		t.Fatalf("could not find all names")
	}

	if appleIdx > middleIdx || middleIdx > zebraIdx {
		t.Errorf("checks should be sorted alphabetically by name")
	}
}

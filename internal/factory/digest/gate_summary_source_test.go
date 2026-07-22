// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGateSummarySourceMissing tests rendering when the file is absent.
func TestGateSummarySourceMissing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	section := buildGateSummarySection(tmpDir)

	if !strings.Contains(section, "source_status=missing") {
		t.Errorf("expected source_status=missing, got:\n%s", section)
	}
	if !strings.Contains(section, "schema_version=0") {
		t.Errorf("expected schema_version=0, got:\n%s", section)
	}
	if !strings.Contains(section, "overall_status=unavailable") {
		t.Errorf("expected overall_status=unavailable, got:\n%s", section)
	}
	if !strings.Contains(section, "failure_stage=") {
		t.Errorf("expected failure_stage field, got:\n%s", section)
	}
}

// TestGateSummarySourceInvalidRead tests rendering when file cannot be read.
func TestGateSummarySourceInvalidRead(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create a directory named gate-summary.json (not a file)
	factoryDir := filepath.Join(tmpDir, ".factory")
	if err := os.MkdirAll(filepath.Join(factoryDir, "gate-summary.json"), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	section := buildGateSummarySection(tmpDir)

	if !strings.Contains(section, "source_status=invalid") {
		t.Errorf("expected source_status=invalid, got:\n%s", section)
	}
	if !strings.Contains(section, "failure_stage=read") {
		t.Errorf("expected failure_stage=read, got:\n%s", section)
	}
	if !strings.Contains(section, "DG_GATE_SUMMARY_READ_FAILED") {
		t.Errorf("expected DG_GATE_SUMMARY_READ_FAILED, got:\n%s", section)
	}
	// Must use stable diagnostic path
	expectedDiag := "code=DG_GATE_SUMMARY_READ_FAILED path=/.factory/gate-summary.json"
	if !strings.Contains(section, expectedDiag) {
		t.Fatalf("missing stable read diagnostic, expected %q in:\n%s", expectedDiag, section)
	}
	// Must not contain tmpDir
	if strings.Contains(section, tmpDir) {
		t.Errorf("must not contain tmpDir %q in diagnostics:\n%s", tmpDir, section)
	}
}

// TestGateSummarySourceInvalidDecode tests rendering when JSON is malformed.
func TestGateSummarySourceInvalidDecode(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte("{invalid json"))

	section := buildGateSummarySection(tmpDir)

	if !strings.Contains(section, "source_status=invalid") {
		t.Errorf("expected source_status=invalid, got:\n%s", section)
	}
	if !strings.Contains(section, "failure_stage=decode") {
		t.Errorf("expected failure_stage=decode, got:\n%s", section)
	}
	if !strings.Contains(section, "diagnostics_total=") {
		t.Errorf("expected diagnostics_total, got:\n%s", section)
	}
}

// TestGateSummarySourceInvalidNormalize tests rendering when decode succeeds but normalize fails.
func TestGateSummarySourceInvalidNormalize(t *testing.T) {
	t.Parallel()

	// Valid JSON but semantically invalid - v2 with duplicate check names (same name+scope)
	tmpDir := t.TempDir()
	invalidV2 := `{
		"schema_version": 2,
		"generated_at": "2024-01-01T00:00:00Z",
		"scope_id": "TEST",
		"scope_status": "CLOSED",
		"scope_disposition": "done",
		"parent_act": "TEST",
		"parent_status": "CLOSED",
		"parent_disposition": "done",
		"overall_status": "pass",
		"overall_disposition": "done",
		"execution_head_oid": "0123456789abcdef0123456789abcdef01234567",
		"execution_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"subject_tree_oid": "0123456789abcdef0123456789abcdef01234567",
		"worktree_clean_before": true,
		"worktree_clean_after": true,
		"checks": [
			{
				"name": "check",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "check.sh",
				"detail": "check test",
				"extras": {
					"argv": ["check.sh"],
					"exit_code": 0,
					"duration_ms": 100,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			},
			{
				"name": "check",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "check.sh",
				"detail": "check test 2",
				"extras": {
					"argv": ["check.sh"],
					"exit_code": 0,
					"duration_ms": 200,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`
	writeGateSummaryFile(t, tmpDir, []byte(invalidV2))

	section := buildGateSummarySection(tmpDir)

	if !strings.Contains(section, "source_status=invalid") {
		t.Errorf("expected source_status=invalid, got:\n%s", section)
	}
	if !strings.Contains(section, "failure_stage=normalize") {
		t.Errorf("expected failure_stage=normalize, got:\n%s", section)
	}
	if !strings.Contains(section, "schema_version=2") {
		t.Errorf("expected schema_version=2, got:\n%s", section)
	}
	if !strings.Contains(section, "overall_status=unavailable") {
		t.Errorf("expected overall_status=unavailable, got:\n%s", section)
	}
	if !strings.Contains(section, "checks_total=0") {
		t.Errorf("expected checks_total=0, got:\n%s", section)
	}
	// No partial scope fields should be emitted
	if strings.Contains(section, "scope_id=") {
		t.Errorf("no partial scope_id should be emitted, got:\n%s", section)
	}
	if strings.Contains(section, "parent_act=") {
		t.Errorf("no partial parent_act should be emitted, got:\n%s", section)
	}
	if strings.Contains(section, "checks:") {
		t.Errorf("no partial checks section should be emitted, got:\n%s", section)
	}
}

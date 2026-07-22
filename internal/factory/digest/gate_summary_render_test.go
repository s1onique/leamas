// SPDX-License-Identifier: Apache-2.0

package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test fixtures directory
var gatesummaryTestdataDir = filepath.Join("..", "..", "gatesummary", "testdata", "valid")

// Golden files directory
var goldenDir = filepath.Join("testdata")

func readTestFixture(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(gatesummaryTestdataDir, name))
}

func readGolden(name string) (string, error) {
	content, err := os.ReadFile(filepath.Join(goldenDir, name+".golden.txt"))
	return string(content), err
}

func writeGateSummaryFile(t *testing.T, tmpDir string, content []byte) {
	t.Helper()
	factoryDir := filepath.Join(tmpDir, ".factory")
	if err := os.MkdirAll(factoryDir, 0755); err != nil {
		t.Fatalf("failed to create .factory directory: %v", err)
	}
	gsPath := filepath.Join(factoryDir, "gate-summary.json")
	if err := os.WriteFile(gsPath, content, 0644); err != nil {
		t.Fatalf("failed to write gate-summary.json: %v", err)
	}
}

// TestGateSummaryV1Minimal tests rendering a minimal v1 summary.
func TestGateSummaryV1Minimal(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v1-minimal.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v1-minimal")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryV1Full tests rendering a full v1 summary.
func TestGateSummaryV1Full(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v1-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v1-full")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryV2Minimal tests rendering a minimal v2 summary.
func TestGateSummaryV2Minimal(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-minimal.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v2-minimal")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryV2ClineMM tests the ClineMM topology.
func TestGateSummaryV2ClineMM(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-clinemm-microc3.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v2-clinemm-microc3")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryV2Full tests v2-full.
func TestGateSummaryV2Full(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v2-full")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryV2LeamasSelfHosted tests the Leamas self-hosted topology.
func TestGateSummaryV2LeamasSelfHosted(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-leamas-self-hosted.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section := buildGateSummarySection(tmpDir)
	golden, err := readGolden("v2-leamas-self-hosted")
	if err != nil {
		t.Fatalf("failed to read golden: %v", err)
	}
	if section != golden {
		t.Errorf("literal mismatch:\n--- GOT ---\n%s\n--- EXPECTED ---\n%s", section, golden)
	}
}

// TestGateSummaryDeterminism20x tests that 20 renders produce identical output.
func TestGateSummaryDeterminism20x(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-clinemm-microc3.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	var first string
	for i := 0; i < 20; i++ {
		section := buildGateSummarySection(tmpDir)
		if i == 0 {
			first = section
		} else if section != first {
			t.Errorf("render %d differs from render 1", i+1)
		}
	}
}

// TestGateSummaryCheckOrderDeterministic tests that check order is deterministic.
func TestGateSummaryCheckOrderDeterministic(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section1 := buildGateSummarySection(tmpDir)
	section2 := buildGateSummarySection(tmpDir)
	if section1 != section2 {
		t.Errorf("identical input should produce identical output")
	}
	names := findCheckNames(section1)
	if !isSorted(names) {
		t.Errorf("check names should be sorted, got: %v", names)
	}
}

func findCheckNames(section string) []string {
	var names []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "  - name=") {
			val := strings.TrimPrefix(line, "  - name=")
			if idx := strings.Index(val, " "); idx != -1 {
				val = val[:idx]
			}
			names = append(names, val)
		}
	}
	return names
}

func isSorted(ss []string) bool {
	for i := 1; i < len(ss); i++ {
		if ss[i] < ss[i-1] {
			return false
		}
	}
	return true
}

// TestGateSummaryNoSourceMutation tests that repeated reads don't mutate the source.
func TestGateSummaryNoSourceMutation(t *testing.T) {
	t.Parallel()
	fixture, err := readTestFixture("v2-full.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, fixture)
	section1 := buildGateSummarySection(tmpDir)
	section2 := buildGateSummarySection(tmpDir)
	section3 := buildGateSummarySection(tmpDir)
	if section1 != section2 || section2 != section3 {
		t.Errorf("subsequent renders differ, suggesting source mutation")
	}
}

// TestGateSummarySameNameDeterministic tests that checks with same name are ordered deterministically.
func TestGateSummarySameNameDeterministic(t *testing.T) {
	t.Parallel()
	// Valid v1 fixture with same-name checks
	fixture := `{
		"schema_version": 1,
		"generated_at": "2024-01-01T00:00:00Z",
		"overall_status": "pass",
		"checks": [
			{"name": "check", "status": "pass"},
			{"name": "check", "status": "fail"},
			{"name": "check", "status": "pass"}
		]
	}`
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte(fixture))
	section := buildGateSummarySection(tmpDir)
	if !strings.Contains(section, "source_status=present") {
		t.Errorf("expected source_status=present, got:\n%s", section)
	}
	if !strings.Contains(section, "checks_total=3") {
		t.Errorf("expected checks_total=3, got:\n%s", section)
	}
	// Render twice and compare
	section2 := buildGateSummarySection(tmpDir)
	if section != section2 {
		t.Errorf("same-name checks should produce deterministic output")
	}
}

// TestGateSummaryV1ZeroDurationOmitted tests that v1 zero duration is omitted (legacy behavior).
func TestGateSummaryV1ZeroDurationOmitted(t *testing.T) {
	t.Parallel()
	fixture := `{
		"schema_version": 1,
		"generated_at": "2024-01-01T00:00:00Z",
		"overall_status": "pass",
		"checks": [
			{"name": "instant", "status": "pass", "duration_ms": 0}
		]
	}`
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte(fixture))
	section := buildGateSummarySection(tmpDir)
	// V1: zero duration should be omitted, not rendered as duration_ms=0
	if strings.Contains(section, "duration_ms=") {
		t.Errorf("v1 zero duration should be omitted, got:\n%s", section)
	}
	if !strings.Contains(section, "evidence=instant") {
		t.Errorf("v1 empty evidence should fall back to name, got:\n%s", section)
	}
}

// TestGateSummaryV2ZeroDurationRendered tests that v2 zero duration is rendered as duration_ms=0.
func TestGateSummaryV2ZeroDurationRendered(t *testing.T) {
	t.Parallel()
	fixture := `{
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
				"name": "instant",
				"scope": "ROOT",
				"status": "pass",
				"evidence": "instant.sh",
				"detail": "instant test",
				"extras": {
					"argv": ["instant.sh"],
					"exit_code": 0,
					"duration_ms": 0,
					"stdout_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					"stderr_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
				}
			}
		]
	}`
	tmpDir := t.TempDir()
	writeGateSummaryFile(t, tmpDir, []byte(fixture))
	section := buildGateSummarySection(tmpDir)
	// V2: zero duration should be rendered as duration_ms=0
	if !strings.Contains(section, "duration_ms=0") {
		t.Errorf("v2 zero duration should be rendered as duration_ms=0, got:\n%s", section)
	}
}

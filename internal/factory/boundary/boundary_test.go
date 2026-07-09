// Package boundary provides verification for domain boundary import policies.
package boundary

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot returns the absolute path to the repository root for tests.
func repoRoot() string {
	// internal/factory/boundary -> internal/factory -> internal -> repo root
	return filepath.Join(os.Getenv("PWD"), "..", "..", "..")
}

// TestCurrentRepoPoliciesPass verifies that current protected packages pass.
func TestCurrentRepoPoliciesPass(t *testing.T) {
	result := Check(repoRoot())
	if !result.OK() {
		for _, f := range result.Findings {
			t.Errorf("boundary violation: %s imports %s: %s", f.File, f.Import, f.Reason)
		}
	}
}

// TestHulkRunbundleAllowsSort verifies that hulk runbundle allows sort import.
func TestHulkRunbundleAllowsSort(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/hulk/runbundle/runbundle.go" && f.Import == "sort" {
			t.Errorf("runbundle should allow sort import, but got violation: %s", f.Reason)
		}
	}
}

// TestHulkClaimevidenceAllowsSort verifies that hulk claimevidence allows sort import.
func TestHulkClaimevidenceAllowsSort(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/hulk/claimevidence/claimevidence.go" && f.Import == "sort" {
			t.Errorf("claimevidence should allow sort import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessProxyAllowsNetHTTP verifies that witness proxy allows net/http.
func TestWitnessProxyAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/witness/proxy/proxy.go" && f.Import == "net/http" {
			t.Errorf("witness proxy should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestWitnessProxyAllowsHttputil verifies that witness proxy allows net/http/httputil.
func TestWitnessProxyAllowsHttputil(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/witness/proxy/proxy.go" && f.Import == "net/http/httputil" {
			t.Errorf("witness proxy should allow httputil import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsEmbed verifies that cockpit allows embed.
func TestCockpitAllowsEmbed(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "embed" {
			t.Errorf("cockpit should allow embed import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsEncodingJSON verifies that cockpit allows encoding/json.
func TestCockpitAllowsEncodingJSON(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "encoding/json" {
			t.Errorf("cockpit should allow encoding/json import, but got violation: %s", f.Reason)
		}
	}
}

// TestCockpitAllowsNetHTTP verifies that cockpit allows net/http.
func TestCockpitAllowsNetHTTP(t *testing.T) {
	result := Check(".")
	for _, f := range result.Findings {
		if f.File == "internal/web/cockpit/cockpit.go" && f.Import == "net/http" {
			t.Errorf("cockpit should allow net/http import, but got violation: %s", f.Reason)
		}
	}
}

// TestFindingsDeterministic verifies that findings are in deterministic order.
func TestFindingsDeterministic(t *testing.T) {
	result1 := Check(".")
	result2 := Check(".")

	if len(result1.Findings) != len(result2.Findings) {
		t.Fatalf("different number of findings: %d vs %d", len(result1.Findings), len(result2.Findings))
	}

	for i := range result1.Findings {
		if result1.Findings[i] != result2.Findings[i] {
			t.Errorf("finding at index %d differs:\n  first:  %+v\n  second: %+v", i, result1.Findings[i], result2.Findings[i])
		}
	}
}

// TestResultOK verifies the Result.OK() method.
func TestResultOK(t *testing.T) {
	emptyResult := Result{}
	if !emptyResult.OK() {
		t.Error("empty result should be OK")
	}

	nonEmptyResult := Result{
		Findings: []Finding{
			{File: "test.go", Import: "net/http", Reason: "forbidden"},
		},
	}
	if nonEmptyResult.OK() {
		t.Error("non-empty result should not be OK")
	}
}

// TestMissingDirectoryDetection verifies that missing protected directories are detected.
func TestMissingDirectoryDetection(t *testing.T) {
	tmpDir := t.TempDir()

	result := Check(tmpDir)

	// All 4 protected packages + 2 CLI runtime files should be reported as missing = 6
	if len(result.Findings) != 6 {
		t.Fatalf("expected 6 missing findings, got %d: %v", len(result.Findings), result.Findings)
	}

	foundDirs := make(map[string]bool)
	foundFiles := make(map[string]bool)
	for _, f := range result.Findings {
		if f.Import == "(missing directory)" {
			foundDirs[f.File] = true
		} else if f.Import == "(missing file)" {
			foundFiles[f.File] = true
		}
	}

	expectedDirs := []string{
		"internal/hulk/runbundle",
		"internal/hulk/claimevidence",
		"internal/witness/proxy",
		"internal/web/cockpit",
	}
	for _, dir := range expectedDirs {
		if !foundDirs[dir] {
			t.Errorf("expected missing directory finding for %s", dir)
		}
	}

	expectedFiles := []string{
		"cmd/leamas/cockpit.go",
		"cmd/leamas/witness.go",
	}
	for _, file := range expectedFiles {
		if !foundFiles[file] {
			t.Errorf("expected missing file finding for %s", file)
		}
	}
}

// TestMissingCLIRuntimeFileDetection verifies that missing CLI runtime files are detected.
func TestMissingCLIRuntimeFileDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create protected directories to avoid missing directory findings
	createDir(t, tmpDir, "internal/hulk/runbundle")
	createDir(t, tmpDir, "internal/hulk/claimevidence")
	createDir(t, tmpDir, "internal/witness/proxy")
	createDir(t, tmpDir, "internal/web/cockpit")

	result := Check(tmpDir)

	// All 2 CLI runtime files should be reported as missing
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 missing CLI file findings, got %d: %v", len(result.Findings), result.Findings)
	}

	foundFiles := make(map[string]bool)
	for _, f := range result.Findings {
		if f.Import != "(missing file)" {
			t.Errorf("expected missing file import, got: %s", f.Import)
		}
		if f.Reason != "expected CLI runtime file does not exist" {
			t.Errorf("unexpected reason: %s", f.Reason)
		}
		foundFiles[f.File] = true
	}

	expectedFiles := []string{
		"cmd/leamas/cockpit.go",
		"cmd/leamas/witness.go",
	}
	for _, file := range expectedFiles {
		if !foundFiles[file] {
			t.Errorf("expected missing CLI file finding for %s", file)
		}
	}
}

// Helper functions for creating test files

func createDir(t *testing.T, baseDir, relPath string) {
	t.Helper()
	path := filepath.Join(baseDir, relPath)
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func createProtectedDirs(t *testing.T, baseDir string) {
	t.Helper()
	createDir(t, baseDir, "internal/hulk/runbundle")
	createDir(t, baseDir, "internal/hulk/claimevidence")
	createDir(t, baseDir, "internal/witness/proxy")
	createDir(t, baseDir, "internal/web/cockpit")
}

func createProtectedDirsWithCLI(t *testing.T, baseDir string) {
	t.Helper()
	createDir(t, baseDir, "internal/hulk/runbundle")
	createDir(t, baseDir, "internal/hulk/claimevidence")
	createDir(t, baseDir, "internal/witness/proxy")
	createDir(t, baseDir, "internal/web/cockpit")
	createDir(t, baseDir, "cmd/leamas")
}

func createCLIRuntimeFile(t *testing.T, baseDir, relPath, content string) {
	t.Helper()
	path := filepath.Join(baseDir, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func createHulkFile(t *testing.T, baseDir, relPath, content string) {
	t.Helper()
	createCLIRuntimeFile(t, baseDir, relPath, content)
}

func createWitnessFile(t *testing.T, baseDir, relPath, content string) {
	t.Helper()
	createCLIRuntimeFile(t, baseDir, relPath, content)
}

func createCockpitFile(t *testing.T, baseDir, relPath, content string) {
	t.Helper()
	createCLIRuntimeFile(t, baseDir, relPath, content)
}

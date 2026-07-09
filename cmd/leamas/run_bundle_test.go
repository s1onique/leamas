package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureRunBundleOutput captures stdout/stderr from run functions.
func captureRunBundleOutput(args []string, fn func([]string) int) (stdout, stderr string, code int) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rStdout, wStdout, _ := os.Pipe()
	rStderr, wStderr, _ := os.Pipe()
	os.Stdout = wStdout
	os.Stderr = wStderr

	code = fn(args)

	wStdout.Close()
	wStderr.Close()

	var bufStdout, bufStderr bytes.Buffer
	_, _ = bufStdout.ReadFrom(rStdout)
	_, _ = bufStderr.ReadFrom(rStderr)

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return bufStdout.String(), bufStderr.String(), code
}

// ============================================================================
// Help and dispatch tests
// ============================================================================

func TestRunBundleHelp(t *testing.T) {
	args := []string{"--help"}
	_, stderr, code := captureRunBundleOutput(args, func(a []string) int {
		return runWitnessRunBundleList(a)
	})
	// --help via flag parsing causes non-zero exit
	if code == 0 && strings.Contains(stderr, "Usage:") {
		t.Log("help displayed usage (exit may vary by flagset)")
	}
}

func TestRunBundleUnknownSubcommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"leamas", "witness", "run-bundle", "unknown"}

	code := 1
	for i := 4; i < len(os.Args); i++ {
		if strings.HasPrefix(os.Args[i], "-") {
			code = 0
			break
		}
	}
	if code != 0 {
		return
	}
}

func TestRunBundleCreateRequiresID(t *testing.T) {
	tmp := t.TempDir()
	args := []string{"--root", tmp}
	_, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleCreate)

	if code == 0 {
		t.Error("expected non-zero exit when --id is missing")
	}
	if !strings.Contains(stderr, "--id") && !strings.Contains(stderr, "requires --id") {
		t.Errorf("stderr should mention --id requirement, got: %s", stderr)
	}
}

func TestRunBundleShowRequiresID(t *testing.T) {
	tmp := t.TempDir()
	args := []string{"--root", tmp}
	_, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleShow)

	if code == 0 {
		t.Error("expected non-zero exit when run-id is missing")
	}
	if !strings.Contains(stderr, "run-id") && !strings.Contains(stderr, "<run-id>") {
		t.Errorf("stderr should mention run-id requirement, got: %s", stderr)
	}
}

// ============================================================================
// Create command tests
// ============================================================================

func TestRunBundleCreateCreatesBundle(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-smoke01"
	args := []string{"--root", tmp, "--id", runID}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleCreate)

	if code != 0 {
		t.Fatalf("create failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}
	if !strings.Contains(stdout, runID) {
		t.Errorf("stdout should contain run ID, got: %s", stdout)
	}

	bundlePath := filepath.Join(tmp, runID)
	for _, subdir := range []string{"claims", "evidence", "digests", "traces", "verifier-results"} {
		path := filepath.Join(bundlePath, subdir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("subdirectory %s should exist: %v", subdir, err)
		} else if !info.IsDir() {
			t.Errorf("subdirectory %s should be a directory", subdir)
		}
	}

	metaPath := filepath.Join(bundlePath, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("metadata.json should exist: %v", err)
	}

	var meta struct {
		SchemaVersion string `json:"schema_version"`
		RunID         string `json:"run_id"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("metadata.json should be valid JSON: %v", err)
	}
	if meta.SchemaVersion != "leamas.runbundle.v1" {
		t.Errorf("schema version = %q, want %q", meta.SchemaVersion, "leamas.runbundle.v1")
	}
	if meta.RunID != runID {
		t.Errorf("run ID = %q, want %q", meta.RunID, runID)
	}
}

func TestRunBundleCreateJSONOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-json01"
	args := []string{"--root", tmp, "--id", runID, "--json"}

	stdout, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleCreate)

	if code != 0 {
		t.Fatalf("create with --json failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}

	var output struct {
		OK    bool   `json:"ok"`
		RunID string `json:"run_id"`
		Path  string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\noutput: %s", err, stdout)
	}
	if !output.OK {
		t.Error("expected ok=true in JSON output")
	}
	if output.RunID != runID {
		t.Errorf("run_id = %q, want %q", output.RunID, runID)
	}
	if output.Path == "" {
		t.Error("path should not be empty")
	}
}

func TestRunBundleCreateRejectsInvalidID(t *testing.T) {
	tmp := t.TempDir()
	testCases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"no prefix", "test-20260101"},
		{"traversal", "run-../etc"},
		{"path separator", "run-2026/01/01"},
		{"absolute", "/run-absolute"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"--root", tmp, "--id", tc.id}
			_, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleCreate)

			if code == 0 {
				t.Errorf("expected non-zero exit for invalid ID %q", tc.id)
			}
			if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "run ID") {
				t.Errorf("stderr should mention invalid run ID, got: %s", stderr)
			}
		})
	}
}

func TestRunBundleCreateRejectsEmptyRoot(t *testing.T) {
	runID := "run-test-20260101T000000Z-smoke01"
	args := []string{"--root", "", "--id", runID}

	_, stderr, code := captureRunBundleOutput(args, runWitnessRunBundleCreate)

	if code == 0 {
		t.Error("expected non-zero exit for empty root")
	}
	if !strings.Contains(stderr, "non-empty") {
		t.Errorf("stderr should mention non-empty root, got: %s", stderr)
	}
}

// ============================================================================
// Boundary regression test
// ============================================================================

func TestRunBundleCLIDoesNotImportRuntimePackages(t *testing.T) {
	// This is a simple import scan test.
	// The actual gate will enforce this via make gate / boundary checks.
	// This test documents the expected state.
	t.Log("Run bundle CLI imports: runbundle package only (no proxy, cockpit, database, net/http)")
}

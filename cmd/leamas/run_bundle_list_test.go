package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/witness/runbundle"
)

// captureRunBundleListOutput captures stdout/stderr from list command.
func captureRunBundleListOutput(args []string) (stdout, stderr string, code int) {
	return captureRunBundleListFn(args, runWitnessRunBundleList)
}

// captureRunBundleListFn is a test helper for list command.
func captureRunBundleListFn(args []string, fn func([]string) int) (stdout, stderr string, code int) {
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

func TestRunBundleListEmptyRoot(t *testing.T) {
	tmp := t.TempDir()
	args := []string{"--root", tmp}

	stdout, stderr, code := captureRunBundleListOutput(args)

	if code != 0 {
		t.Fatalf("list failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, "no run bundles") {
		t.Errorf("stdout should say 'no run bundles found', got: %s", stdout)
	}
}

func TestRunBundleListShowsCreatedBundles(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-list01"
	_, err := runbundle.Create(runbundle.CreateOptions{
		Root:  tmp,
		RunID: runbundle.RunID(runID),
	})
	if err != nil {
		t.Fatalf("failed to create test bundle: %v", err)
	}

	args := []string{"--root", tmp}
	stdout, stderr, code := captureRunBundleListOutput(args)

	if code != 0 {
		t.Fatalf("list failed with code %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(stdout, runID) {
		t.Errorf("stdout should contain created run ID %q, got: %s", runID, stdout)
	}
}

func TestRunBundleListJSONOutput(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-listjson01"
	_, err := runbundle.Create(runbundle.CreateOptions{
		Root:  tmp,
		RunID: runbundle.RunID(runID),
	})
	if err != nil {
		t.Fatalf("failed to create test bundle: %v", err)
	}

	args := []string{"--root", tmp, "--json"}
	stdout, stderr, code := captureRunBundleListOutput(args)

	if code != 0 {
		t.Fatalf("list --json failed with code %d, stderr: %s", code, stderr)
	}

	var output struct {
		OK      string `json:"ok"`
		Root    string `json:"root"`
		Bundles []struct {
			RunID         string `json:"run_id"`
			CreatedAt     string `json:"created_at"`
			Path          string `json:"path"`
			SchemaVersion string `json:"schema_version"`
		} `json:"bundles"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\noutput: %s", err, stdout)
	}
	if output.OK != "true" {
		t.Errorf("ok = %q, want %q", output.OK, "true")
	}
	if len(output.Bundles) != 1 {
		t.Fatalf("bundles count = %d, want 1", len(output.Bundles))
	}
	if output.Bundles[0].RunID != runID {
		t.Errorf("bundles[0].run_id = %q, want %q", output.Bundles[0].RunID, runID)
	}
}

func TestRunBundleListIgnoresNonBundles(t *testing.T) {
	tmp := t.TempDir()

	regularFile := filepath.Join(tmp, "regular-file.txt")
	if err := os.WriteFile(regularFile, []byte("not a bundle"), 0644); err != nil {
		t.Fatalf("failed to create regular file: %v", err)
	}

	args := []string{"--root", tmp}
	stdout, stderr, code := captureRunBundleListOutput(args)

	if code != 0 {
		t.Fatalf("list failed with code %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "regular-file") {
		t.Errorf("stdout should not contain non-bundle file, got: %s", stdout)
	}
}

func TestRunBundleListSkipsInvalidBundles(t *testing.T) {
	tmp := t.TempDir()

	invalidDir := filepath.Join(tmp, "run-invalid-20260101T000000Z-bad")
	if err := os.MkdirAll(invalidDir, 0755); err != nil {
		t.Fatalf("failed to create invalid bundle dir: %v", err)
	}

	validID := "run-test-20260101T000000Z-good"
	_, err := runbundle.Create(runbundle.CreateOptions{
		Root:  tmp,
		RunID: runbundle.RunID(validID),
	})
	if err != nil {
		t.Fatalf("failed to create valid bundle: %v", err)
	}

	args := []string{"--root", tmp}
	stdout, stderr, code := captureRunBundleListOutput(args)

	if code != 0 {
		t.Fatalf("list failed with code %d, stderr: %s", code, stderr)
	}
	if strings.Contains(stdout, "run-invalid") {
		t.Errorf("stdout should not contain invalid bundle, got: %s", stdout)
	}
	if !strings.Contains(stdout, validID) {
		t.Errorf("stdout should contain valid bundle %q, got: %s", validID, stdout)
	}
}

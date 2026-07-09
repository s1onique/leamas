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

// captureRunBundleShowOutput captures stdout/stderr from show command.
func captureRunBundleShowOutput(args []string) (stdout, stderr string, code int) {
	return captureRunBundleShowFn(args, runWitnessRunBundleShow)
}

// captureRunBundleShowFn is a test helper for show command.
func captureRunBundleShowFn(args []string, fn func([]string) int) (stdout, stderr string, code int) {
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

func TestRunBundleShowDisplaysMetadata(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-show01"
	_, err := runbundle.Create(runbundle.CreateOptions{
		Root:    tmp,
		RunID:   runbundle.RunID(runID),
		Version: "v1.2.3",
	})
	if err != nil {
		t.Fatalf("failed to create test bundle: %v", err)
	}

	args := []string{"--root", tmp, runID}
	stdout, stderr, code := captureRunBundleShowOutput(args)

	if code != 0 {
		t.Fatalf("show failed with code %d, stderr: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr on success, got: %s", stderr)
	}

	if !strings.Contains(stdout, runID) {
		t.Errorf("stdout should contain run ID, got: %s", stdout)
	}
	if !strings.Contains(stdout, "leamas.runbundle.v1") {
		t.Errorf("stdout should contain schema version, got: %s", stdout)
	}
	if !strings.Contains(stdout, "local_only=true") {
		t.Errorf("stdout should contain doctrine flags, got: %s", stdout)
	}
}

func TestRunBundleShowJSONOutput(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-showjson01"
	_, err := runbundle.Create(runbundle.CreateOptions{
		Root:    tmp,
		RunID:   runbundle.RunID(runID),
		Version: "v1.0.0",
	})
	if err != nil {
		t.Fatalf("failed to create test bundle: %v", err)
	}

	args := []string{"--root", tmp, "--json", runID}
	stdout, stderr, code := captureRunBundleShowOutput(args)

	if code != 0 {
		t.Fatalf("show --json failed with code %d, stderr: %s", code, stderr)
	}

	var output struct {
		OK       bool   `json:"ok"`
		Path     string `json:"path"`
		Metadata struct {
			SchemaVersion string `json:"schema_version"`
			RunID         string `json:"run_id"`
			CreatedAt     string `json:"created_at"`
			Tool          struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"tool"`
			Doctrine struct {
				LocalOnly  bool `json:"local_only"`
				ReadOnly   bool `json:"read_only"`
				NoDatabase bool `json:"no_database"`
			} `json:"doctrine"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("stdout should be valid JSON: %v\noutput: %s", err, stdout)
	}
	if !output.OK {
		t.Error("expected ok=true in JSON output")
	}
	if output.Metadata.RunID != runID {
		t.Errorf("metadata.run_id = %q, want %q", output.Metadata.RunID, runID)
	}
	if !output.Metadata.Doctrine.LocalOnly {
		t.Error("expected doctrine.local_only=true")
	}
}

func TestRunBundleShowRejectsInvalidID(t *testing.T) {
	tmp := t.TempDir()
	invalidIDs := []string{"", "bad", "../etc", "run-../traversal"}

	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			args := []string{"--root", tmp, id}
			_, stderr, code := captureRunBundleShowOutput(args)

			if code == 0 {
				t.Errorf("expected non-zero exit for invalid ID %q", id)
			}
			if !strings.Contains(stderr, "invalid") && !strings.Contains(stderr, "run ID") {
				t.Errorf("stderr should mention invalid run ID, got: %s", stderr)
			}
		})
	}
}

func TestRunBundleShowRejectsMissingMetadata(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-nometa"
	noMetaPath := filepath.Join(tmp, runID)
	if err := os.MkdirAll(noMetaPath, 0755); err != nil {
		t.Fatalf("failed to create dir without metadata: %v", err)
	}

	args := []string{"--root", tmp, runID}
	_, stderr, code := captureRunBundleShowOutput(args)

	if code == 0 {
		t.Error("expected non-zero exit for missing metadata")
	}
	if !strings.Contains(stderr, "not found") && !strings.Contains(stderr, "metadata") {
		t.Errorf("stderr should mention missing metadata, got: %s", stderr)
	}
}

func TestRunBundleShowRejectsSchemaMismatch(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-badschema"
	bundlePath := filepath.Join(tmp, runID)
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	meta := map[string]interface{}{
		"schema_version": "wrong.version",
		"run_id":         runID,
	}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(bundlePath, "metadata.json"), data, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	args := []string{"--root", tmp, runID}
	_, stderr, code := captureRunBundleShowOutput(args)

	if code == 0 {
		t.Error("expected non-zero exit for schema mismatch")
	}
	if !strings.Contains(stderr, "schema") && !strings.Contains(stderr, "mismatch") {
		t.Errorf("stderr should mention schema mismatch, got: %s", stderr)
	}
}

func TestRunBundleShowRejectsRunIDMismatch(t *testing.T) {
	tmp := t.TempDir()

	runID := "run-test-20260101T000000Z-good"
	actualDir := filepath.Join(tmp, runID)
	if err := os.MkdirAll(actualDir, 0755); err != nil {
		t.Fatalf("failed to create bundle dir: %v", err)
	}

	meta := map[string]interface{}{
		"schema_version": "leamas.runbundle.v1",
		"run_id":         "run-different-id",
		"created_at":     "2026-01-01T00:00:00Z",
		"tool":           map[string]string{"name": "leamas"},
		"doctrine":       map[string]bool{"local_only": true, "read_only": true, "no_database": true},
	}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(actualDir, "metadata.json"), data, 0644); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	args := []string{"--root", tmp, runID}
	_, stderr, code := captureRunBundleShowOutput(args)

	if code == 0 {
		t.Error("expected non-zero exit for run ID mismatch")
	}
	if !strings.Contains(stderr, "run ID") && !strings.Contains(stderr, "mismatch") {
		t.Errorf("stderr should mention run ID mismatch, got: %s", stderr)
	}
}

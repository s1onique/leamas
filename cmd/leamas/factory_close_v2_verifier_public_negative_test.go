// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_verifier_public_negative_test.go covers
// Phase 8 of
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02C:
// the public CLI negative matrix. Every row exercises the
// real `factory close verify-v2-authority` entry point
// and pins the exit-code partition.

// This file does not re-test the unit-level seams (those
// live in v2_verifier_*_test.go inside the closure
// package); the focus is the public surface.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestV2VerifierPublicCLIRejectsDuplicateBeforeObservation
// pins the Phase 1 invariant: a duplicate flag is rejected
// before the CLI even touches the orchestrator. The test
// uses a non-repository --repository path; if the parser
// ever regressed and observed Git first, the exit would
// change from 2 (usage) to 4 (observer).
func TestV2VerifierPublicCLIRejectsDuplicateBeforeObservation(t *testing.T) {
	cases := [][]string{
		{"--repository", "/tmp/nope", "--repository", "/tmp/nope"},
		{"--subject", "0000000000000000000000000000000000000000",
			"--subject", "0000000000000000000000000000000000000000"},
		{"--output", "/tmp/a", "--output", "/tmp/a"},
		{"--json", "--json"},
		{"--help", "--help"},
	}
	for _, args := range cases {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		got := runFactoryCloseVerifyV2Authority(args, stdout, stderr)
		if got != v2VerifierExitUsage {
			t.Fatalf("args=%v must produce exit 2, got %d (stderr=%s)",
				args, got, stderr.String())
		}
	}
}

// TestV2VerifierPublicCLIRejectsUnsafeOutputBeforeObservation
// pins the read-only output authority: an --output path
// inside the repository must surface the
// verifier_output_path_not_detached observer-class
// diagnostic and exit 4 BEFORE any Git observation.
func TestV2VerifierPublicCLIRejectsUnsafeOutputBeforeObservation(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	got := runFactoryCloseVerifyV2Authority([]string{
		"--repository", repo,
		"--subject", "0000000000000000000000000000000000000000",
		"--freeze", "0000000000000000000000000000000000000000",
		"--closure", "0000000000000000000000000000000000000000",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
		"--output", filepath.Join(repo, "out.json"),
	}, stdout, stderr)
	if got != v2VerifierExitObserverBroken {
		t.Fatalf("unsafe output must produce exit 4, got %d (stderr=%s)",
			got, stderr.String())
	}
}

// TestV2VerifierPublicCLIJSONContract pins the JSON
// envelope: --json mode must emit exactly one document,
// the `ok` field reflects the verdict, and the diagnostic
// list is non-empty on failure.
func TestV2VerifierPublicCLIJSONContract(t *testing.T) {
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{
		"--repository", dir,
		"--subject", "0000000000000000000000000000000000000000",
		"--freeze", "0000000000000000000000000000000000000000",
		"--closure", "0000000000000000000000000000000000000000",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
		"--json",
	}, stdout, stderr)
	dec := json.NewDecoder(stdout)
	var doc map[string]any
	if err := dec.Decode(&doc); err != nil {
		t.Fatalf("decode JSON envelope: %v (stdout=%s)", err, stdout.String())
	}
	if _, err := dec.Token(); err == nil {
		t.Fatalf("JSON output must be exactly one document")
	}
	if _, ok := doc["ok"]; !ok {
		t.Fatalf("JSON envelope must include ok field, got %v", doc)
	}
	if diagnostics, ok := doc["verification"].(map[string]any)["diagnostics"]; !ok {
		t.Fatalf("JSON envelope must include diagnostics, got %v", doc["verification"])
	} else if arr, ok := diagnostics.([]any); !ok || len(arr) == 0 {
		t.Fatalf("diagnostics must be non-empty on failure, got %v", diagnostics)
	}
}

// TestV2VerifierPublicCLITextContract pins the text-mode
// surface: success emits exactly one OK line; failure
// emits typed diagnostics on stderr and never emits JSON
// on stdout.
func TestV2VerifierPublicCLITextContract(t *testing.T) {
	dir := t.TempDir()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{
		"--repository", dir,
		"--subject", "0000000000000000000000000000000000000000",
		"--freeze", "0000000000000000000000000000000000000000",
		"--closure", "0000000000000000000000000000000000000000",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
	}, stdout, stderr)
	if stdout.Len() != 0 {
		// On failure, text mode must not write a JSON envelope.
		trimmed := bytes.TrimSpace(stdout.Bytes())
		if trimmed[0] == '{' {
			t.Fatalf("text mode must not emit JSON, got %q", trimmed)
		}
	}
	if stderr.Len() == 0 {
		t.Fatalf("text mode failure must surface diagnostics on stderr")
	}
}

// TestV2VerifierPublicCLINoOutputOnPrepublicationFailure
// pins the read-only output authority: a failed invocation
// MUST NOT publish to the --output file. The hermetic
// repository rejection path is the canonical trigger.
func TestV2VerifierPublicCLINoOutputOnPrepublicationFailure(t *testing.T) {
	repo := t.TempDir()
	outside := t.TempDir()
	outputPath := filepath.Join(outside, "verifier-output.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	_ = runFactoryCloseVerifyV2Authority([]string{
		"--repository", repo,
		"--subject", "0000000000000000000000000000000000000000",
		"--freeze", "0000000000000000000000000000000000000000",
		"--closure", "0000000000000000000000000000000000000000",
		"--plan-path", "plan/plan.json",
		"--manifest-path", "manifest/manifest.json",
		"--output", outputPath,
	}, stdout, stderr)
	if _, err := os.Stat(outputPath); err == nil {
		t.Fatalf("output file must not be published on failure, found %s", outputPath)
	}
}

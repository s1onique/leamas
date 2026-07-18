// Characterization tests for the `leamas witness evidence show` command.
//
// These tests pin the observable behaviour of runWitnessEvidenceShow BEFORE
// the production refactor that introduces a shared `record_show` helper.
// They MUST remain GREEN across the refactor and serve as the contract
// proof for the duplicated geometry in cmd/leamas/claim_commands.go and
// cmd/leamas/evidence_commands.go.
//
// Tests in this file must not be marked t.Parallel(): the capture helper
// swaps package-level os.Stdout/os.Stderr globals.
package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// runEvidenceShowCapture runs runWitnessEvidenceShow with the given argv
// under swapped stdout/stderr and returns the captured streams plus exit code.
func runEvidenceShowCapture(t *testing.T, argv []string) (stdout, stderr string, code int) {
	t.Helper()
	stdout, stderr, code = captureWithCode(t, func() int {
		return runWitnessEvidenceShow(argv)
	})
	return stdout, stderr, code
}

// makeEvidenceBundleForShow creates a minimal run bundle plus a single
// evidence record ready to be shown. Returns (root, runID, evidenceID).
func makeEvidenceBundleForShow(t *testing.T) (string, string, string) {
	t.Helper()
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evishow01"
	_, _, c := captureWithCode(t, func() int {
		return runWitnessRunBundleCreate([]string{"--root", tmp, "--id", runID})
	})
	if c != 0 {
		t.Fatalf("create bundle: code=%d", c)
	}
	evidenceID := "evidence-characterization-001"
	createArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", evidenceID,
		"--kind", "log",
		"--role", "primary",
		"--title", "evidence title for characterization",
		"--summary", "evidence summary for characterization",
		"--relative-path", "logs/run.log",
	}
	_, _, c = captureWithCode(t, func() int {
		return runWitnessEvidenceCreate(createArgs)
	})
	if c != 0 {
		t.Fatalf("create evidence: code=%d", c)
	}
	return tmp, runID, evidenceID
}

// TestEvidenceShow_Text_Success asserts the canonical text-mode rendering
// for a freshly-created evidence record: every line, in order, with no
// trailing extras.
func TestEvidenceShow_Text_Success(t *testing.T) {
	tmp, runID, evidenceID := makeEvidenceBundleForShow(t)

	argv := []string{"--root", tmp, "--run-id", runID, evidenceID}
	stdout, stderr, code := runEvidenceShowCapture(t, argv)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}

	wantLines := []string{
		"Evidence: " + evidenceID,
		"Run: " + runID,
		"Kind: log",
		"Role: primary",
		"Title: evidence title for characterization",
		"Path: logs/run.log",
		"Summary: evidence summary for characterization",
	}
	want := strings.Join(wantLines, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout mismatch:\n--- want ---\n%s\n--- got ---\n%s", want, stdout)
	}
}

// TestEvidenceShow_JSON_Success asserts the JSON output structure, key set,
// and schema-relevant fields.
func TestEvidenceShow_JSON_Success(t *testing.T) {
	tmp, runID, evidenceID := makeEvidenceBundleForShow(t)

	argv := []string{"--root", tmp, "--run-id", runID, "--json", evidenceID}
	stdout, stderr, code := runEvidenceShowCapture(t, argv)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}

	var envelope struct {
		OK       bool        `json:"ok"`
		Evidence interface{} `json:"evidence"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !envelope.OK {
		t.Errorf("envelope.ok = false, want true")
	}
	if envelope.Evidence == nil {
		t.Fatalf("envelope.evidence missing")
	}

	evidenceMap, ok := envelope.Evidence.(map[string]interface{})
	if !ok {
		t.Fatalf("envelope.evidence must be an object, got %T", envelope.Evidence)
	}
	if evidenceMap["id"] != evidenceID {
		t.Errorf("evidence.id = %v, want %q", evidenceMap["id"], evidenceID)
	}
	if evidenceMap["run_id"] != runID {
		t.Errorf("evidence.run_id = %v, want %q", evidenceMap["run_id"], runID)
	}
	if evidenceMap["kind"] != "log" {
		t.Errorf("evidence.kind = %v, want \"log\"", evidenceMap["kind"])
	}
	if evidenceMap["title"] != "evidence title for characterization" {
		t.Errorf("evidence.title mismatch: %v", evidenceMap["title"])
	}
}

// TestEvidenceShow_MissingPositionalArg asserts the exact error and exit
// code when the <evidence-id> positional argument is omitted.
func TestEvidenceShow_MissingPositionalArg(t *testing.T) {
	tmp, runID, _ := makeEvidenceBundleForShow(t)

	stdout, stderr, code := runEvidenceShowCapture(t, []string{"--root", tmp, "--run-id", runID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: evidence show requires <evidence-id>") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestEvidenceShow_MissingRunID asserts the exact error and exit code when
// --run-id is omitted.
func TestEvidenceShow_MissingRunID(t *testing.T) {
	tmp, _, evidenceID := makeEvidenceBundleForShow(t)

	stdout, stderr, code := runEvidenceShowCapture(t, []string{"--root", tmp, evidenceID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: evidence show requires --run-id") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestEvidenceShow_EmptyRoot asserts the exact error when --root is empty.
func TestEvidenceShow_EmptyRoot(t *testing.T) {
	_, runID, evidenceID := makeEvidenceBundleForShow(t)

	stdout, stderr, code := runEvidenceShowCapture(t, []string{"--root", "", "--run-id", runID, evidenceID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: run bundle root must be non-empty") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestEvidenceShow_InvalidEvidenceID asserts the exact error and exit code
// when the supplied evidence ID is rejected by claim.ValidateEvidenceID.
func TestEvidenceShow_InvalidEvidenceID(t *testing.T) {
	tmp, runID, _ := makeEvidenceBundleForShow(t)

	// "bad" lacks the required "evidence-" prefix.
	stdout, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "bad",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: invalid evidence ID:") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestEvidenceShow_RepoNotFound asserts the exact error and exit code when
// the run bundle does not exist on disk.
func TestEvidenceShow_RepoNotFound(t *testing.T) {
	tmp := t.TempDir()
	missingRunID := "run-test-20260101T000000Z-missing002"

	stdout, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", missingRunID, "evidence-missing",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR:") {
		t.Errorf("stderr missing ERROR prefix, got %q", stderr)
	}
}

// TestEvidenceShow_NotFound asserts the exact error and exit code when the
// run bundle exists but the evidence record does not.
func TestEvidenceShow_NotFound(t *testing.T) {
	tmp, runID, _ := makeEvidenceBundleForShow(t)

	stdout, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "evidence-does-not-exist",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: evidence not found: evidence-does-not-exist") {
		t.Errorf("stderr missing expected not-found error, got %q", stderr)
	}
}

// TestEvidenceShow_InvalidRunID asserts the exact error and exit code when
// the --run-id value is not a valid run bundle identifier.
func TestEvidenceShow_InvalidRunID(t *testing.T) {
	tmp := t.TempDir()

	stdout, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", "bad-run-id", "evidence-anything",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: invalid run ID:") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestEvidenceShow_NoFilesystemWrites asserts that `show` performs no
// filesystem mutations in either success or failure modes.
func TestEvidenceShow_NoFilesystemWrites(t *testing.T) {
	tmp, runID, evidenceID := makeEvidenceBundleForShow(t)

	beforeFiles := listJSONFiles(t, tmp)

	_, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, evidenceID,
	})
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, stderr)
	}

	afterFiles := listJSONFiles(t, tmp)
	if !stringSlicesEqual(beforeFiles, afterFiles) {
		t.Fatalf("show created or removed files on success:\nbefore=%v\nafter=%v",
			beforeFiles, afterFiles)
	}

	stdout2, stderr2, code2 := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "evidence-not-present",
	})
	if code2 != 1 {
		t.Fatalf("exit code (missing) = %d, want 1; stderr=%q stdout=%q",
			code2, stderr2, stdout2)
	}
	afterFailFiles := listJSONFiles(t, tmp)
	if !stringSlicesEqual(beforeFiles, afterFailFiles) {
		t.Fatalf("show mutated tree on failure:\nbefore=%v\nafter=%v",
			beforeFiles, afterFailFiles)
	}
}

// TestEvidenceShow_StdoutStderrSeparation asserts that success puts the
// record on stdout and failures put errors on stderr, never the other way
// around.
func TestEvidenceShow_StdoutStderrSeparation(t *testing.T) {
	tmp, runID, evidenceID := makeEvidenceBundleForShow(t)

	stdout, stderr, code := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, evidenceID,
	})
	if code != 0 {
		t.Fatalf("success path: exit=%d stderr=%q", code, stderr)
	}
	if stdout == "" {
		t.Fatalf("success path: stdout must not be empty")
	}
	if stderr != "" {
		t.Fatalf("success path: stderr must be empty, got %q", stderr)
	}

	stdout2, stderr2, code2 := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "evidence-does-not-exist",
	})
	if code2 == 0 {
		t.Fatalf("error path: exit must be non-zero")
	}
	if stdout2 != "" {
		t.Fatalf("error path: stdout must be empty, got %q", stdout2)
	}
	if stderr2 == "" {
		t.Fatalf("error path: stderr must not be empty")
	}
}

// TestEvidenceShow_FlagParseError asserts that an unknown flag exits 1 with
// the error written to stderr (and stdout empty).
func TestEvidenceShow_FlagParseError(t *testing.T) {
	stdout, stderr, code := runEvidenceShowCapture(t, []string{"--unknown-flag"})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on parse error, got %q", stdout)
	}
	if !strings.Contains(stderr, "flag provided but not defined") &&
		!strings.Contains(stderr, "unknown flag") {
		t.Errorf("stderr missing flag-parse error, got %q", stderr)
	}
}

// TestEvidenceShow_DiffersFromClaimShow documents the explicit difference
// between the two show commands: the noun printed in error messages and the
// schema field name in JSON output. These tests pin the contract that the
// refactor must preserve.
func TestEvidenceShow_DiffersFromClaimShow(t *testing.T) {
	tmp, runID, _ := makeEvidenceBundleForShow(t)

	// evidence uses "evidence not found" not "claim not found".
	_, stderr, _ := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "evidence-does-not-exist",
	})
	if strings.Contains(stderr, "claim not found") {
		t.Errorf("evidence show must not print claim-not-found text; got %q", stderr)
	}
	if !strings.Contains(stderr, "evidence not found") {
		t.Errorf("evidence show must print evidence-not-found text; got %q", stderr)
	}

	// evidence uses "evidence" label, not "<claim-id>".
	_, stderr2, _ := runEvidenceShowCapture(t, []string{
		"--root", tmp, "--run-id", runID,
	})
	if strings.Contains(stderr2, "<claim-id>") {
		t.Errorf("evidence show must not reference <claim-id>; got %q", stderr2)
	}
	if !strings.Contains(stderr2, "<evidence-id>") {
		t.Errorf("evidence show must reference <evidence-id>; got %q", stderr2)
	}
}

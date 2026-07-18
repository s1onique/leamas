// Characterization tests for the `leamas witness claim show` command.
//
// These tests pin the observable behaviour of runWitnessClaimShow BEFORE
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

// runClaimShowCapture runs runWitnessClaimShow with the given argv under
// swapped stdout/stderr and returns the captured streams plus exit code.
func runClaimShowCapture(t *testing.T, argv []string) (stdout, stderr string, code int) {
	t.Helper()
	stdout, stderr, code = captureWithCode(t, func() int {
		return runWitnessClaimShow(argv)
	})
	return stdout, stderr, code
}

// makeClaimBundleForShow creates a minimal run bundle plus a single claim
// record ready to be shown. Returns (root, runID, claimID). Output of the
// helper commands is captured and discarded to avoid polluting the test log.
func makeClaimBundleForShow(t *testing.T) (string, string, string) {
	t.Helper()
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-clmshow01"
	_, _, c := captureWithCode(t, func() int {
		return runWitnessRunBundleCreate([]string{"--root", tmp, "--id", runID})
	})
	if c != 0 {
		t.Fatalf("create bundle: code=%d", c)
	}
	claimID := "claim-characterization-001"
	createArgs := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", claimID,
		"--statement", "claim statement for characterization",
		"--notes", "claim notes for characterization",
	}
	_, _, c = captureWithCode(t, func() int {
		return runWitnessClaimCreate(createArgs)
	})
	if c != 0 {
		t.Fatalf("create claim: code=%d", c)
	}
	return tmp, runID, claimID
}

// TestClaimShow_Text_Success asserts the canonical text-mode rendering for a
// freshly-created claim: every line, in order, with no trailing extras.
func TestClaimShow_Text_Success(t *testing.T) {
	tmp, runID, claimID := makeClaimBundleForShow(t)

	argv := []string{"--root", tmp, "--run-id", runID, claimID}
	stdout, stderr, code := runClaimShowCapture(t, argv)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}

	wantLines := []string{
		"Claim: " + claimID,
		"Run: " + runID,
		"Status: open",
		"Verdict: unreviewed",
		"Statement: claim statement for characterization",
		"Notes: claim notes for characterization",
	}
	want := strings.Join(wantLines, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout mismatch:\n--- want ---\n%s\n--- got ---\n%s", want, stdout)
	}
}

// TestClaimShow_JSON_Success asserts the JSON output structure, key set, and
// schema-relevant fields. It does not assert byte-for-byte equality because
// time-based fields exist; it asserts shape.
func TestClaimShow_JSON_Success(t *testing.T) {
	tmp, runID, claimID := makeClaimBundleForShow(t)

	argv := []string{"--root", tmp, "--run-id", runID, "--json", claimID}
	stdout, stderr, code := runClaimShowCapture(t, argv)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr must be empty on success, got %q", stderr)
	}

	var envelope struct {
		OK    bool        `json:"ok"`
		Claim interface{} `json:"claim"`
	}
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !envelope.OK {
		t.Errorf("envelope.ok = false, want true")
	}
	if envelope.Claim == nil {
		t.Fatalf("envelope.claim missing")
	}

	claimMap, ok := envelope.Claim.(map[string]interface{})
	if !ok {
		t.Fatalf("envelope.claim must be an object, got %T", envelope.Claim)
	}
	if claimMap["id"] != claimID {
		t.Errorf("claim.id = %v, want %q", claimMap["id"], claimID)
	}
	if claimMap["run_id"] != runID {
		t.Errorf("claim.run_id = %v, want %q", claimMap["run_id"], runID)
	}
	if claimMap["statement"] != "claim statement for characterization" {
		t.Errorf("claim.statement mismatch: %v", claimMap["statement"])
	}
}

// TestClaimShow_MissingPositionalArg asserts the exact error and exit code
// when the <claim-id> positional argument is omitted.
func TestClaimShow_MissingPositionalArg(t *testing.T) {
	tmp, runID, _ := makeClaimBundleForShow(t)

	stdout, stderr, code := runClaimShowCapture(t, []string{"--root", tmp, "--run-id", runID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: claim show requires <claim-id>") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestClaimShow_MissingRunID asserts the exact error and exit code when
// --run-id is omitted.
func TestClaimShow_MissingRunID(t *testing.T) {
	tmp, _, claimID := makeClaimBundleForShow(t)

	stdout, stderr, code := runClaimShowCapture(t, []string{"--root", tmp, claimID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: claim show requires --run-id") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestClaimShow_EmptyRoot asserts the exact error when --root is empty.
func TestClaimShow_EmptyRoot(t *testing.T) {
	tmp, runID, claimID := makeClaimBundleForShow(t)

	stdout, stderr, code := runClaimShowCapture(t, []string{"--root", "", "--run-id", runID, claimID})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: run bundle root must be non-empty") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
	_ = tmp
}

// TestClaimShow_InvalidClaimID asserts the exact error and exit code when the
// supplied claim ID is rejected by claim.ValidateClaimID.
func TestClaimShow_InvalidClaimID(t *testing.T) {
	tmp, runID, _ := makeClaimBundleForShow(t)

	// "bad" lacks the required "claim-" prefix.
	stdout, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "bad",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: invalid claim ID:") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
}

// TestClaimShow_RepoNotFound asserts the exact error and exit code when the
// run bundle does not exist on disk.
func TestClaimShow_RepoNotFound(t *testing.T) {
	tmp := t.TempDir()
	missingRunID := "run-test-20260101T000000Z-missing001"

	stdout, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", missingRunID, "claim-missing",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	// printRunBundleError formats the path explicitly.
	if !strings.Contains(stderr, "ERROR:") {
		t.Errorf("stderr missing ERROR prefix, got %q", stderr)
	}
}

// TestClaimShow_NotFound asserts the exact error and exit code when the run
// bundle exists but the claim record does not.
func TestClaimShow_NotFound(t *testing.T) {
	tmp, runID, _ := makeClaimBundleForShow(t)

	stdout, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "claim-does-not-exist",
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: claim not found: claim-does-not-exist") {
		t.Errorf("stderr missing expected not-found error, got %q", stderr)
	}
}

// TestClaimShow_InvalidRunID asserts the exact error and exit code when the
// --run-id value is not a valid run bundle identifier.
func TestClaimShow_InvalidRunID(t *testing.T) {
	tmp := t.TempDir()

	stdout, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", "bad-run-id", "claim-anything",
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

// TestClaimShow_NoFilesystemWrites asserts that `show` performs no
// filesystem mutations in either success or failure modes.
func TestClaimShow_NoFilesystemWrites(t *testing.T) {
	tmp, runID, claimID := makeClaimBundleForShow(t)

	// Capture the directory listing before the show call.
	beforeFiles := listJSONFiles(t, tmp)

	// Successful show.
	_, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, claimID,
	})
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, stderr)
	}

	afterFiles := listJSONFiles(t, tmp)
	if !stringSlicesEqual(beforeFiles, afterFiles) {
		t.Fatalf("show created or removed files on success:\nbefore=%v\nafter=%v",
			beforeFiles, afterFiles)
	}

	// Failing show (claim does not exist) must also leave the tree alone.
	stdout2, stderr2, code2 := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "claim-not-present",
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

// TestClaimShow_StdoutStderrSeparation asserts that success puts the record
// on stdout and failures put errors on stderr, never the other way around.
func TestClaimShow_StdoutStderrSeparation(t *testing.T) {
	tmp, runID, claimID := makeClaimBundleForShow(t)

	// Success path.
	stdout, stderr, code := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, claimID,
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

	// Error path.
	stdout2, stderr2, code2 := runClaimShowCapture(t, []string{
		"--root", tmp, "--run-id", runID, "claim-does-not-exist",
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

// TestClaimShow_FlagParseError asserts that an unknown flag exits 1 with the
// error written to stderr (and stdout empty).
func TestClaimShow_FlagParseError(t *testing.T) {
	stdout, stderr, code := runClaimShowCapture(t, []string{"--unknown-flag"})

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

// (Helpers listJSONFiles and stringSlicesEqual live in cli_test_helpers_test.go.)

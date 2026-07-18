// Error-path unit tests for the runRecordShow shared abstraction in
// record_show.go. The happy-path tests live in record_show_test.go;
// this file owns the validation, not-found, persistence, and
// repo-error scenarios so each file stays within the LLM-friendly
// line budget.
//
// Tests in this file must not be marked t.Parallel(): the capture
// helper swaps package-level os.Stdout/os.Stderr globals.
package main

import (
	"errors"
	"strings"
	"testing"
)

// TestRunRecordShow_ValidationRejectsPathSeparator exercises the spec's
// ValidateID hook. The shared operation must surface the error verbatim
// in the "<kind> ID: <error>" format.
func TestRunRecordShow_ValidationRejectsPathSeparator(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	spec := minimalSpec(nil, nil)

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "evil/path"},
			spec,
		)
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on validation error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: invalid minimal ID:") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
	if !strings.Contains(stderr, "path separator not allowed") {
		t.Errorf("stderr missing underlying validation message, got %q", stderr)
	}
}

// TestRunRecordShow_NotFoundTranslation asserts that errors.Is(err, NotFoundErr)
// drives the "<kind> not found: <id>" message and exit code 1.
func TestRunRecordShow_NotFoundTranslation(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	recs := map[string]minimalRecord{} // empty
	spec := minimalSpec(recs, nil)

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "absent"},
			spec,
		)
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: minimal not found: absent") {
		t.Errorf("stderr missing expected not-found error, got %q", stderr)
	}
}

// TestRunRecordShow_NonNotFoundError asserts that any other read error is
// translated into the "failed to read <kind>: <error>" message and exit 1.
func TestRunRecordShow_NonNotFoundError(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	spec := minimalSpec(nil, errors.New("disk exploded"))

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "anything"},
			spec,
		)
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR: failed to read minimal:") {
		t.Errorf("stderr missing expected error, got %q", stderr)
	}
	if !strings.Contains(stderr, "disk exploded") {
		t.Errorf("stderr missing underlying error message, got %q", stderr)
	}
}

// TestRunRecordShow_SpecPolicyInvoked asserts that the spec's read loader
// is called exactly once per successful invocation and never on validation
// or arg-count failure.
func TestRunRecordShow_SpecPolicyInvoked(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	resetRecordCounter()
	recs := map[string]minimalRecord{"x": {ID: "x", Title: "x"}}
	spec := minimalSpec(recs, nil)

	// Success path: counter increments once.
	_, _, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "x"},
			spec,
		)
	})
	if code != 0 {
		t.Fatalf("success path exit = %d, want 0", code)
	}
	if got := readRecordCounter(); got != 1 {
		t.Errorf("read counter after success = %d, want 1", got)
	}

	// Failure path (validation): counter does NOT increment.
	resetRecordCounter()
	_, _, code = captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "x/extra"},
			spec,
		)
	})
	if code != 1 {
		t.Fatalf("validation failure exit = %d, want 1", code)
	}
	if got := readRecordCounter(); got != 0 {
		t.Errorf("read counter after validation failure = %d, want 0", got)
	}

	// Failure path (missing positional arg): counter does NOT increment.
	resetRecordCounter()
	_, _, code = captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID},
			spec,
		)
	})
	if code != 1 {
		t.Fatalf("missing-arg failure exit = %d, want 1", code)
	}
	if got := readRecordCounter(); got != 0 {
		t.Errorf("read counter after missing-arg failure = %d, want 0", got)
	}
}

// TestRunRecordShow_RepositoryErrorPropagates asserts that a repository
// open error from runbundle.Open is surfaced through the existing
// printRunBundleError path with exit code 1 and a non-empty stderr.
func TestRunRecordShow_RepositoryErrorPropagates(t *testing.T) {
	tmp := t.TempDir()
	missingRunID := "run-test-20260101T000000Z-missingrec001"
	spec := minimalSpec(nil, nil)

	stdout, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", tmp, "--run-id", missingRunID, "anything"},
			spec,
		)
	})

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on repo error, got %q", stdout)
	}
	if !strings.Contains(stderr, "ERROR:") {
		t.Errorf("stderr missing ERROR prefix, got %q", stderr)
	}
}

// TestRunRecordShow_NoBundleMutation asserts that the shared operation
// does not create or delete files anywhere under the run bundle, even on
// a successful read.
func TestRunRecordShow_NoBundleMutation(t *testing.T) {
	root, runID := makeMinimalRunBundleForRecordShow(t)
	recs := map[string]minimalRecord{"a": {ID: "a", Title: "a"}}
	spec := minimalSpec(recs, nil)

	before := listJSONFiles(t, root)

	_, stderr, code := captureWithCode(t, func() int {
		return runRecordShow(
			[]string{"--root", root, "--run-id", runID, "a"},
			spec,
		)
	})
	if code != 0 {
		t.Fatalf("exit code = %d; stderr=%q", code, stderr)
	}

	after := listJSONFiles(t, root)
	if !stringSlicesEqual(before, after) {
		t.Fatalf("shared op mutated bundle:\nbefore=%v\nafter=%v", before, after)
	}
}

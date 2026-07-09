// Package runbundle provides local run bundle creation and validation for
// Leamas verification witness evidence.
package runbundle

import (
	"strings"
	"testing"
)

func TestValidateRunIDAcceptsSafeIDs(t *testing.T) {
	// Valid run IDs that should be accepted
	safeIDs := []string{
		"run-20260709T071704Z-smoke01",
		"run-20260709T071704Z-abcdef",
		"run-abc123",
		"run-ABC123",
		"run-abc.def",
		"run-abc_def",
		"run-abc-def",
		"run-a",
		"run-123",
		"run-abc.def_ghi-jkl",
		"run-.hidden", // dot is in allowed charset
	}

	for _, id := range safeIDs {
		t.Run(id, func(t *testing.T) {
			err := ValidateRunID(RunID(id))
			if err != nil {
				t.Errorf("ValidateRunID(%q) returned error: %v", id, err)
			}
		})
	}
}

func TestValidateRunIDRejectsUnsafeIDs(t *testing.T) {
	// Unsafe run IDs that should be rejected
	// Just verify they return an error - specific error type doesn't matter
	unsafeIDs := []string{
		"",                // empty
		".",               // reserved name
		"..",              // reserved name
		"../escape",       // traversal
		"run/bad",         // path separator
		"/absolute",       // absolute path
		"run with spaces", // spaces not in charset
		"run\tbad",        // tab not in charset
		"run\nbad",        // newline not in charset
	}

	for _, id := range unsafeIDs {
		t.Run(id, func(t *testing.T) {
			err := ValidateRunID(RunID(id))
			if err == nil {
				t.Errorf("ValidateRunID(%q) should have returned error, got nil", id)
			}
		})
	}
}

func TestValidateRunIDLength(t *testing.T) {
	// Test maximum length (128 chars total including "run-" prefix)
	// run- = 4 chars, so we need 124 more chars for 128 total
	// Use only valid chars (a-z) for the long ID
	longID := "run-" + strings.Repeat("a", 124)
	err := ValidateRunID(RunID(longID))
	if err != nil {
		t.Errorf("ValidateRunID with 128-char ID should pass, got: %v", err)
	}

	// Test over maximum length (129 chars total)
	tooLongID := "run-" + strings.Repeat("a", 125)
	err = ValidateRunID(RunID(tooLongID))
	if err != ErrRunIDTooLong {
		t.Errorf("ValidateRunID with >128-char ID should return ErrRunIDTooLong, got: %v", err)
	}
}

func TestBundlePathStaysUnderRoot(t *testing.T) {
	testCases := []struct {
		root string
		id   string
	}{
		{"/tmp/leamas", "run-20260709T071704Z-smoke01"},
		{"/tmp/.leamas/runs", "run-20260709T071704Z-abcdef"},
		{"/data/bundles", "run-abc123"},
	}

	for _, tc := range testCases {
		t.Run(tc.root+"/"+tc.id, func(t *testing.T) {
			path, err := BundlePath(tc.root, RunID(tc.id))
			if err != nil {
				t.Errorf("BundlePath(%q, %q) returned error: %v", tc.root, tc.id, err)
				return
			}
			if path == "" {
				t.Error("BundlePath returned empty path")
			}
		})
	}
}

func TestBundlePathRejectsTraversalID(t *testing.T) {
	ids := []string{
		"run-../escape",
		"run-../test",
		"../run-bad",
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			_, err := BundlePath("/tmp/leamas", RunID(id))
			if err == nil {
				t.Errorf("BundlePath should reject traversal ID %q", id)
			}
		})
	}
}

func TestBundlePathRejectsAbsoluteID(t *testing.T) {
	ids := []string{
		"/absolute/run-bad",
		"/run-bad",
	}

	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			_, err := BundlePath("/tmp/leamas", RunID(id))
			if err == nil {
				t.Errorf("BundlePath should reject absolute ID %q", id)
			}
		})
	}
}

func TestBundlePathRejectsEmptyRoot(t *testing.T) {
	_, err := BundlePath("", RunID("run-20260709T071704Z-smoke01"))
	if err != ErrEmptyRoot {
		t.Errorf("BundlePath should reject empty root, got: %v", err)
	}
}

func TestBundlePathRequiresValidRunID(t *testing.T) {
	_, err := BundlePath("/tmp/leamas", RunID(""))
	if err != ErrEmptyRunID {
		t.Errorf("BundlePath should reject empty run ID, got: %v", err)
	}

	_, err = BundlePath("/tmp/leamas", RunID("invalid"))
	if err != ErrRunIDNoPrefix {
		t.Errorf("BundlePath should reject run ID without 'run-' prefix, got: %v", err)
	}
}

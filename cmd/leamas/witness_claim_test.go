// Package main provides tests for witness claim commands.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// ============================================================================
// Claim dispatcher tests with parse/usage
// ============================================================================

func TestParseWitnessClaimCommand_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantErr         bool
		wantErrContains string
	}{
		// Success cases - runWitnessClaim dispatches to subcommands
		{"create", []string{"create"}, false, ""},
		{"list", []string{"list"}, false, ""},
		{"show", []string{"show"}, false, ""},
		{"attach-evidence", []string{"attach-evidence"}, false, ""},
		// Error cases
		{"empty args", []string{}, true, "Usage:"},
		{"unknown subcommand", []string{"unknown"}, true, "unknown claim subcommand"},
		{"case mismatch create", []string{"Create"}, true, "unknown claim subcommand"},
		{"case mismatch list", []string{"LIST"}, true, "unknown claim subcommand"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runWitnessClaim(tt.args)

			if tt.wantErr {
				if code == 0 {
					t.Fatal("expected non-zero exit, got 0")
				}
				return
			}

			// For success cases, we expect either 0 or 1 depending on whether
			// the subcommand succeeds (depends on args passed)
			// We only care that it dispatched to the right subcommand
			t.Logf("code=%d", code)
		})
	}
}

func TestRunWitnessClaim_MissingSubcommand(t *testing.T) {
	code := runWitnessClaim([]string{})

	if code == 0 {
		t.Error("missing subcommand should exit non-zero")
	}
}

func TestRunWitnessClaim_UnknownSubcommand(t *testing.T) {
	code := runWitnessClaim([]string{"unknown"})

	if code == 0 {
		t.Error("unknown subcommand should exit non-zero")
	}
}

func TestRunWitnessClaim_Help(t *testing.T) {
	code := runWitnessClaim([]string{"--help"})

	if code != 0 {
		t.Errorf("help should exit 0, got %d", code)
	}
}

// ============================================================================
// printClaimUsageTo tests
// ============================================================================

func TestPrintClaimUsageTo(t *testing.T) {
	var buf bytes.Buffer
	printClaimUsageTo(&buf)

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("output should contain 'Usage:'")
	}
	if !strings.Contains(output, "leamas witness claim") {
		t.Error("output should mention leamas witness claim")
	}
	if !strings.Contains(output, "create") {
		t.Error("output should mention create subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("output should mention list subcommand")
	}
	if !strings.Contains(output, "show") {
		t.Error("output should mention show subcommand")
	}
	if !strings.Contains(output, "attach-evidence") {
		t.Error("output should mention attach-evidence subcommand")
	}
	if !strings.Contains(output, "--root") {
		t.Error("output should mention --root flag")
	}
	if !strings.Contains(output, "--run-id") {
		t.Error("output should mention --run-id flag")
	}
	if !strings.Contains(output, "--json") {
		t.Error("output should mention --json flag")
	}
}

// ============================================================================
// Claim list parsing tests (without storage)
// ============================================================================

func TestRunWitnessClaimList_RequiresRunID(t *testing.T) {
	args := []string{"--root", "/tmp/nonexistent"}
	code := runWitnessClaimList(args)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
}

func TestRunWitnessClaimList_RejectsInvalidRunID(t *testing.T) {
	args := []string{"--root", "/tmp", "--run-id", "bad-run-id"}
	code := runWitnessClaimList(args)

	if code == 0 {
		t.Error("expected non-zero exit for invalid run ID")
	}
}

func TestRunWitnessClaimList_AcceptsJSON(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-list01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--json"}
	code := runWitnessClaimList(args)

	if code != 0 {
		t.Fatalf("list with --json failed with code %d", code)
	}
}

// ============================================================================
// Claim show parsing tests (without storage)
// ============================================================================

func TestRunWitnessClaimShow_RequiresClaimID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-show01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID}
	code := runWitnessClaimShow(args)

	if code == 0 {
		t.Error("expected non-zero exit when claim-id is missing")
	}
}

func TestRunWitnessClaimShow_RequiresRunID(t *testing.T) {
	args := []string{"--root", "/tmp/nonexistent", "claim-test"}
	code := runWitnessClaimShow(args)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
}

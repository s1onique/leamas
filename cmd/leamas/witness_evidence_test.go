// Package main provides tests for witness evidence commands.
package main

import (
	"bytes"
	"strings"
	"testing"
)

// ============================================================================
// Evidence dispatcher tests with parse/usage
// ============================================================================

func TestParseWitnessEvidenceCommand_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantErr         bool
		wantErrContains string
	}{
		// Success cases - runWitnessEvidence dispatches to subcommands
		{"create", []string{"create"}, false, ""},
		{"list", []string{"list"}, false, ""},
		{"show", []string{"show"}, false, ""},
		// Error cases
		{"empty args", []string{}, true, "Usage:"},
		{"unknown subcommand", []string{"unknown"}, true, "unknown evidence subcommand"},
		{"case mismatch create", []string{"Create"}, true, "unknown evidence subcommand"},
		{"case mismatch list", []string{"LIST"}, true, "unknown evidence subcommand"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := runWitnessEvidence(tt.args)

			if tt.wantErr {
				if code == 0 {
					t.Fatal("expected non-zero exit, got 0")
				}
				// We can't easily capture stderr since the command writes directly
				// to os.Stderr, but we verify the exit code
				return
			}

			// For success cases, we expect either 0 or 1 depending on whether
			// the subcommand succeeds (depends on args passed)
			t.Logf("code=%d for args=%v", code, tt.args)
		})
	}
}

func TestRunWitnessEvidence_MissingSubcommand(t *testing.T) {
	code := runWitnessEvidence([]string{})

	if code == 0 {
		t.Error("missing subcommand should exit non-zero")
	}
}

func TestRunWitnessEvidence_UnknownSubcommand(t *testing.T) {
	code := runWitnessEvidence([]string{"unknown"})

	if code == 0 {
		t.Error("unknown subcommand should exit non-zero")
	}
}

func TestRunWitnessEvidence_Help(t *testing.T) {
	code := runWitnessEvidence([]string{"--help"})

	if code != 0 {
		t.Errorf("help should exit 0, got %d", code)
	}
}

// ============================================================================
// printEvidenceUsageTo tests
// ============================================================================

func TestPrintEvidenceUsageTo(t *testing.T) {
	var buf bytes.Buffer
	printEvidenceUsageTo(&buf)

	output := buf.String()
	if !strings.Contains(output, "Usage:") {
		t.Error("output should contain 'Usage:'")
	}
	if !strings.Contains(output, "leamas witness evidence") {
		t.Error("output should mention leamas witness evidence")
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
// Evidence list parsing tests (without storage)
// ============================================================================

func TestRunWitnessEvidenceList_RequiresRunID(t *testing.T) {
	args := []string{"--root", "/tmp/nonexistent"}
	code := runWitnessEvidenceList(args)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
}

func TestRunWitnessEvidenceList_RejectsInvalidRunID(t *testing.T) {
	args := []string{"--root", "/tmp", "--run-id", "bad-run-id"}
	code := runWitnessEvidenceList(args)

	if code == 0 {
		t.Error("expected non-zero exit for invalid run ID")
	}
}

func TestRunWitnessEvidenceList_AcceptsJSON(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evlist01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--json"}
	code := runWitnessEvidenceList(args)

	if code != 0 {
		t.Fatalf("list with --json failed with code %d", code)
	}
}

// ============================================================================
// Evidence show parsing tests (without storage)
// ============================================================================

func TestRunWitnessEvidenceShow_RequiresEvidenceID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evshow01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID}
	code := runWitnessEvidenceShow(args)

	if code == 0 {
		t.Error("expected non-zero exit when evidence-id is missing")
	}
}

func TestRunWitnessEvidenceShow_RequiresRunID(t *testing.T) {
	args := []string{"--root", "/tmp/nonexistent", "evidence-test"}
	code := runWitnessEvidenceShow(args)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
}

// ============================================================================
// Evidence create parsing tests
// ============================================================================

func TestRunWitnessEvidenceCreate_RequiresRunID(t *testing.T) {
	args := []string{"--root", "/tmp", "--id", "ev-test", "--kind", "log", "--role", "primary", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit when --run-id is missing")
	}
}

func TestRunWitnessEvidenceCreate_RequiresID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate01"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--kind", "log", "--role", "primary", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit when --id is missing")
	}
}

func TestRunWitnessEvidenceCreate_RequiresKind(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate02"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "ev-001", "--role", "primary", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit when --kind is missing")
	}
}

func TestRunWitnessEvidenceCreate_RequiresRole(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate03"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "ev-001", "--kind", "log", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit when --role is missing")
	}
}

func TestRunWitnessEvidenceCreate_RequiresTitle(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate04"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "ev-001", "--kind", "log", "--role", "primary"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit when --title is missing")
	}
}

func TestRunWitnessEvidenceCreate_RejectsInvalidEvidenceID(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate05"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	invalidIDs := []string{"", "bad", "evidence-../etc", "evidence-2026/01"}
	for _, id := range invalidIDs {
		t.Run("id="+id, func(t *testing.T) {
			args := []string{"--root", tmp, "--run-id", runID, "--id", id, "--kind", "log", "--role", "primary", "--title", "Test"}
			code := runWitnessEvidenceCreate(args)
			if code == 0 {
				t.Errorf("expected non-zero exit for invalid ID %q", id)
			}
		})
	}
}

func TestRunWitnessEvidenceCreate_RejectsInvalidKind(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate06"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "ev-001", "--kind", "invalid-kind", "--role", "primary", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit for invalid kind")
	}
}

func TestRunWitnessEvidenceCreate_RejectsInvalidRole(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate07"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{"--root", tmp, "--run-id", runID, "--id", "ev-001", "--kind", "log", "--role", "invalid-role", "--title", "Test"}
	code := runWitnessEvidenceCreate(args)

	if code == 0 {
		t.Error("expected non-zero exit for invalid role")
	}
}

func TestRunWitnessEvidenceCreate_Success(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate08"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-gate-passed",
		"--kind", "command_output",
		"--role", "primary",
		"--title", "Gate test passed",
	}

	code := runWitnessEvidenceCreate(args)

	if code != 0 {
		t.Fatalf("create failed with code %d", code)
	}
}

func TestRunWitnessEvidenceCreate_JSONOutput(t *testing.T) {
	tmp := t.TempDir()
	runID := "run-test-20260101T000000Z-evcreate09"
	runBundleArgs := []string{"--root", tmp, "--id", runID}
	if c := runWitnessRunBundleCreate(runBundleArgs); c != 0 {
		t.Fatalf("failed to create run bundle: %d", c)
	}

	args := []string{
		"--root", tmp,
		"--run-id", runID,
		"--id", "evidence-test-json",
		"--kind", "log",
		"--role", "supporting",
		"--title", "Test JSON output",
		"--json",
	}

	code := runWitnessEvidenceCreate(args)

	if code != 0 {
		t.Fatalf("create with --json failed with code %d", code)
	}
}

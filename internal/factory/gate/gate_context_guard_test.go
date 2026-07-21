// Package gate provides tests for the gate context guard behavior.
package gate

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

const testTimeout = 30 * time.Second

// TestGateContextGuardTruthTable tests the guard truth table.
func TestGateContextGuardTruthTable(t *testing.T) {
	tests := []struct {
		name               string
		env                map[string]string
		wantOutcome        exectest.Outcome
		wantCode           int
		wantRefuseInStderr bool
		wantErrorInStderr  bool
	}{
		{name: "unset_allows", env: map[string]string{}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "cline_marker_refuses", env: map[string]string{"LEAMAS_GATE_CALLER": "cline"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "codium_marker_refuses", env: map[string]string{"LEAMAS_GATE_CALLER": "codium"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "vscode_marker_refuses", env: map[string]string{"LEAMAS_GATE_CALLER": "vscode"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "editor_marker_refuses", env: map[string]string{"LEAMAS_GATE_CALLER": "editor"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "term_program_vscode_refuses", env: map[string]string{"TERM_PROGRAM": "vscode"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "term_program_vscodium_refuses", env: map[string]string{"TERM_PROGRAM": "vscodium"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "vscode_pid_refuses", env: map[string]string{"VSCODE_PID": "12345"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "cline_with_override_allows", env: map[string]string{"LEAMAS_GATE_CALLER": "cline", "LEAMAS_ALLOW_FULL_GATE": "1"}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "term_program_with_override_allows", env: map[string]string{"TERM_PROGRAM": "vscode", "LEAMAS_ALLOW_FULL_GATE": "1"}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "invalid_caller_fails_closed", env: map[string]string{"LEAMAS_GATE_CALLER": "typo"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "invalid_override_fails_closed", env: map[string]string{"LEAMAS_ALLOW_FULL_GATE": "yes"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "override_0_allows", env: map[string]string{"LEAMAS_GATE_CALLER": "cline", "LEAMAS_ALLOW_FULL_GATE": "0"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "invalid_caller_with_override_fails_closed", env: map[string]string{"LEAMAS_GATE_CALLER": "typo", "LEAMAS_ALLOW_FULL_GATE": "1"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "empty_caller_allows", env: map[string]string{"LEAMAS_GATE_CALLER": ""}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "empty_override_allows", env: map[string]string{"LEAMAS_ALLOW_FULL_GATE": ""}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx, testCancel := context.WithTimeout(ctx, 10*time.Second)
			defer testCancel()

			env := buildTestEnv(t, tt.env)
			dir := findRepoRoot(t)

			result := exectest.RunMake(testCtx, dir, env, "gate-context-guard")

			if result.Outcome == exectest.OutcomeSpawnFailure {
				t.Fatalf("spawn failed: %v", result.SpawnErr)
			}
			if result.Outcome == exectest.OutcomeTimeout {
				t.Fatalf("test timed out")
			}
			if result.Outcome != tt.wantOutcome {
				t.Errorf("outcome = %v, want %v", result.Outcome, tt.wantOutcome)
			}
			if result.ExitCode != tt.wantCode {
				t.Errorf("exit code = %d, want %d", result.ExitCode, tt.wantCode)
			}
			if tt.wantRefuseInStderr {
				if !strings.Contains(string(result.Stderr), "REFUSED") {
					t.Errorf("expected REFUSED in stderr, got: %s", result.Stderr)
				}
				if strings.Contains(string(result.Stdout), "REFUSED") {
					t.Errorf("REFUSED should be in stderr, not stdout")
				}
			}
			if tt.wantErrorInStderr {
				if !strings.Contains(string(result.Stderr), "invalid") {
					t.Errorf("expected 'invalid' error in stderr, got: %s", result.Stderr)
				}
			}
		})
	}
}

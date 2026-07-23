// Package gate provides tests for the factorize context guard behavior.
package gate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

const factorizeTestTimeout = 30 * time.Second

// TestFactorizeContextGuardTruthTable tests the factorize guard truth table.
func TestFactorizeContextGuardTruthTable(t *testing.T) {
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
		{name: "cline_with_override_allows", env: map[string]string{"LEAMAS_GATE_CALLER": "cline", "LEAMAS_ALLOW_FULL_FACTORIZE": "1"}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "term_program_with_override_allows", env: map[string]string{"TERM_PROGRAM": "vscode", "LEAMAS_ALLOW_FULL_FACTORIZE": "1"}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "invalid_caller_fails_closed", env: map[string]string{"LEAMAS_GATE_CALLER": "typo"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "invalid_override_fails_closed", env: map[string]string{"LEAMAS_ALLOW_FULL_FACTORIZE": "yes"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "override_0_does_not_bypass_editor_refusal", env: map[string]string{"LEAMAS_GATE_CALLER": "cline", "LEAMAS_ALLOW_FULL_FACTORIZE": "0"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantRefuseInStderr: true},
		{name: "invalid_caller_with_override_fails_closed", env: map[string]string{"LEAMAS_GATE_CALLER": "typo", "LEAMAS_ALLOW_FULL_FACTORIZE": "1"}, wantOutcome: exectest.OutcomeExitFailure, wantCode: 2, wantErrorInStderr: true},
		{name: "empty_caller_allows", env: map[string]string{"LEAMAS_GATE_CALLER": ""}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
		{name: "empty_override_allows", env: map[string]string{"LEAMAS_ALLOW_FULL_FACTORIZE": ""}, wantOutcome: exectest.OutcomeSuccess, wantCode: 0},
	}

	ctx, cancel := context.WithTimeout(context.Background(), factorizeTestTimeout)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx, testCancel := context.WithTimeout(ctx, 10*time.Second)
			defer testCancel()

			env := buildTestEnv(t, tt.env)
			dir := findRepoRoot(t)

			result := exectest.RunMake(testCtx, dir, env, "factorize-context-guard")

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

// baseEnv provides a deterministic environment that clears all ambient gate variables.
func baseEnv() []string {
	return []string{
		"LEAMAS_GATE_CALLER=",
		"LEAMAS_ALLOW_FULL_FACTORIZE=",
		"TERM_PROGRAM=",
		"VSCODE_PID=",
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
	}
}

// TestFactorizeRoutingWithSentinels verifies the routing topology using sentinel files.
func TestFactorizeRoutingWithSentinels(t *testing.T) {
	tmpDir := t.TempDir()

	const factorizeSentinel = "SENTINEL_FACTORIZE_GATE"

	makefile := `PHONY := guard factorize factorize-canonical

guard:
	@test "$(LEAMAS_GATE_CALLER)" != "editor" -o "$(LEAMAS_ALLOW_FULL_FACTORIZE)" = "1" || (echo "REFUSED" >&2; exit 2)

factorize:
	@$(MAKE) --no-print-directory guard
	@$(MAKE) --no-print-directory factorize-canonical

factorize-canonical:
	@echo "running factorize"
	@echo "factorize" >> ` + factorizeSentinel

	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	sentinelExists := func() bool {
		_, err := os.Stat(filepath.Join(tmpDir, factorizeSentinel))
		return err == nil
	}

	cleanSentinel := func() { os.Remove(filepath.Join(tmpDir, factorizeSentinel)) }

	t.Run("editor+gatedefaultsToRefusal", func(t *testing.T) {
		cleanSentinel()
		env := baseEnv()
		env = append(env, "LEAMAS_GATE_CALLER=editor")
		result := exectest.RunMake(context.Background(), tmpDir, env, "factorize")

		if result.Outcome != exectest.OutcomeExitFailure {
			t.Errorf("expected OutcomeExitFailure, got %v", result.Outcome)
		}
		if result.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", result.ExitCode)
		}
		if sentinelExists() {
			t.Error("factorize sentinel should not exist after refusal")
		}
		if !strings.Contains(string(result.Stderr), "REFUSED") {
			t.Errorf("expected REFUSED in stderr, got: %s", result.Stderr)
		}
	})

	t.Run("editor+overrideRunsFactorize", func(t *testing.T) {
		cleanSentinel()
		env := baseEnv()
		env = append(env, "LEAMAS_GATE_CALLER=editor", "LEAMAS_ALLOW_FULL_FACTORIZE=1")
		result := exectest.RunMake(context.Background(), tmpDir, env, "factorize")

		if result.Outcome != exectest.OutcomeSuccess {
			t.Errorf("expected OutcomeSuccess, got %v", result.Outcome)
		}
		if !sentinelExists() {
			t.Fatal("factorize sentinel should exist after override factorize")
		}
	})

	t.Run("emptyCallerWithOverrideRunsFactorize", func(t *testing.T) {
		cleanSentinel()
		env := baseEnv()
		result := exectest.RunMake(context.Background(), tmpDir, env, "factorize")

		if result.Outcome != exectest.OutcomeSuccess {
			t.Errorf("expected OutcomeSuccess, got %v", result.Outcome)
		}
		if !sentinelExists() {
			t.Error("factorize sentinel should exist after factorize with empty caller")
		}
	})
}

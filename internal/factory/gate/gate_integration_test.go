// Package gate_test provides integration tests for the gate context guard behavior.
package gate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

// buildTestEnv builds a test environment with specified overrides.
func buildTestEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()

	env := []string{
		"LEAMAS_GATE_CALLER=",
		"LEAMAS_ALLOW_FULL_GATE=",
		"TERM_PROGRAM=",
		"VSCODE_PID=",
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
	}

	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

// findRepoRoot finds the repository root directory.
func findRepoRoot(t *testing.T) string {
	// Navigate up to find the repo root (contains Makefile)
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "Makefile")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

// TestPublicGateRefusesInEditorContext verifies the real public target refuses in editor context.
func TestPublicGateRefusesInEditorContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := exectest.RunMake(
		ctx,
		findRepoRoot(t),
		buildTestEnv(t, map[string]string{
			"LEAMAS_GATE_CALLER": "codium",
		}),
		"gate",
	)

	// Must refuse with exit code 2
	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf(
			"outcome = %v, want exit failure; stdout=%q stderr=%q",
			result.Outcome,
			result.Stdout,
			result.Stderr,
		)
	}
	if result.ExitCode != 2 {
		t.Fatalf("exit code = %d, want 2", result.ExitCode)
	}

	stderr := string(result.Stderr)
	if !strings.Contains(stderr, "REFUSED") {
		t.Fatalf("missing refusal diagnostic: %q", stderr)
	}
	if !strings.Contains(stderr, "make gate-fast") {
		t.Fatalf("missing gate-fast guidance: %q", stderr)
	}

	// Must not have started canonical work
	allOutput := string(result.Stdout) + stderr
	for _, forbidden := range []string{
		"Running quality gate (full mode)",
		"DUPCODE LANE",
		"GATE PASSED",
	} {
		if strings.Contains(allOutput, forbidden) {
			t.Fatalf("unexpected canonical marker %q", forbidden)
		}
	}
}

// TestGateTimeoutVerifiesTimeout verifies timeout classification.
func TestGateTimeoutVerifiesTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tmpDir := t.TempDir()
	makefile := `
.PHONY: slow-target
slow-target:
	@sleep 10
	@echo "done"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	result := exectest.RunMake(ctx, tmpDir, nil, "slow-target")

	if result.Outcome != exectest.OutcomeTimeout {
		t.Errorf("outcome = %v, want timeout", result.Outcome)
	}
}

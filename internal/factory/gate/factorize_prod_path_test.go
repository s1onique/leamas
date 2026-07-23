// Package gate provides production-path tests for the factorize context guard.
package gate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution/exectest"
)

// TestFactorizePublicTargetRefusesInEditorContext verifies the real public target refuses.
func TestFactorizePublicTargetRefusesInEditorContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER": "codium",
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize")

	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf("outcome = %v, want exit failure; stdout=%q stderr=%q",
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
	if !strings.Contains(stderr, "LEAMAS_ALLOW_FULL_FACTORIZE=1") {
		t.Fatalf("missing override guidance: %q", stderr)
	}

	allOutput := string(result.Stdout) + stderr
	for _, forbidden := range []string{
		"Running factory factorize",
		"FACTORIZE PASSED",
		"FACTORIZE FAILED",
	} {
		if strings.Contains(allOutput, forbidden) {
			t.Fatalf("unexpected factorize marker %q", forbidden)
		}
	}
}

// TestFactorizeCanonicalRefusesInEditorContext verifies factorize-canonical also refuses.
func TestFactorizeCanonicalRefusesInEditorContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER": "codium",
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize-canonical")

	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf("outcome = %v, want exit failure; stdout=%q stderr=%q",
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

	allOutput := string(result.Stdout) + stderr
	for _, forbidden := range []string{
		"Running factory factorize",
		"FACTORIZE PASSED",
		"FACTORIZE FAILED",
	} {
		if strings.Contains(allOutput, forbidden) {
			t.Fatalf("unexpected factorize marker %q", forbidden)
		}
	}
}

// TestFactorizeParallelRefusesInEditorContext verifies parallel invocation refuses.
func TestFactorizeParallelRefusesInEditorContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER": "codium",
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "-j8", "factorize")

	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf("outcome = %v, want exit failure; stdout=%q stderr=%q",
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

	allOutput := string(result.Stdout) + stderr
	for _, forbidden := range []string{
		"Running factory factorize",
		"FACTORIZE PASSED",
		"FACTORIZE FAILED",
	} {
		if strings.Contains(allOutput, forbidden) {
			t.Fatalf("unexpected factorize marker %q", forbidden)
		}
	}
}

// TestFactorizeCanonicalWithOverrideAllows verifies factorize-canonical with override allows.
func TestFactorizeCanonicalWithOverrideAllows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sentinel := filepath.Join(t.TempDir(), "factorize-ran")
	sentinelCmd := fmt.Sprintf("touch %s && echo FACTORIZE PASSED", sentinel)

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER":          "codium",
		"LEAMAS_ALLOW_FULL_FACTORIZE": "1",
		"FACTORIZE_COMMAND":           sentinelCmd,
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize-canonical")

	if result.Outcome != exectest.OutcomeSuccess {
		t.Fatalf("outcome = %v, want success; stdout=%q stderr=%q",
			result.Outcome,
			result.Stdout,
			result.Stderr,
		)
	}

	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Fatalf("sentinel not created, expected single execution")
	}

	if !strings.Contains(string(result.Stdout), "FACTORIZE PASSED") {
		t.Fatalf("expected FACTORIZE PASSED in output, got stdout=%q stderr=%q",
			result.Stdout, result.Stderr)
	}
}

// TestFactorizePublicTargetWithOverrideAndSentinel verifies factorize with bounded sentinel.
func TestFactorizePublicTargetWithOverrideAndSentinel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sentinel := filepath.Join(t.TempDir(), "factorize-public-ran")
	sentinelCmd := fmt.Sprintf("touch %s && echo FACTORIZE_PUBLIC_PASSED", sentinel)

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER":          "codium",
		"LEAMAS_ALLOW_FULL_FACTORIZE": "1",
		"FACTORIZE_COMMAND":           sentinelCmd,
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize")

	if result.Outcome != exectest.OutcomeSuccess {
		t.Fatalf("outcome = %v, want success; stdout=%q stderr=%q",
			result.Outcome,
			result.Stdout,
			result.Stderr,
		)
	}

	if _, err := os.Stat(sentinel); os.IsNotExist(err) {
		t.Fatalf("sentinel not created, expected single execution")
	}
}

// TestFactorizeRefusalExecutesZeroTimes verifies refusal never executes the command.
func TestFactorizeRefusalExecutesZeroTimes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sentinel := filepath.Join(t.TempDir(), "should-not-exist")
	sentinelCmd := fmt.Sprintf("touch %s && echo SHOULD_NOT_RUN", sentinel)

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER": "codium",
		"FACTORIZE_COMMAND":  sentinelCmd,
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize")

	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf("outcome = %v, want exit failure", result.Outcome)
	}

	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("sentinel was created, expected zero executions on refusal")
	}
}

// TestFactorizeInternalTargetAbsent verifies factorize-internal is not a valid target.
func TestFactorizeInternalTargetAbsent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	env := buildTestEnv(t, map[string]string{
		"LEAMAS_GATE_CALLER":          "codium",
		"LEAMAS_ALLOW_FULL_FACTORIZE": "1",
	})
	dir := findRepoRoot(t)

	result := exectest.RunMake(ctx, dir, env, "factorize-internal")

	if result.ExitCode != 2 {
		t.Fatalf("factorize-internal should not exist or refuse; got exit %d", result.ExitCode)
	}
}

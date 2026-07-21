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

// baseEnv provides a deterministic environment that clears all ambient gate variables
func baseEnv() []string {
	return []string{
		"LEAMAS_GATE_CALLER=",
		"LEAMAS_ALLOW_FULL_GATE=",
		"TERM_PROGRAM=",
		"VSCODE_PID=",
		"HOME=" + os.Getenv("HOME"),
		"PATH=" + os.Getenv("PATH"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
	}
}

// TestRoutingWithSentinels verifies the routing topology using sentinel files.
// This simulates the actual guard behavior from make/long-tests.mk.
func TestRoutingWithSentinels(t *testing.T) {
	// Create a temporary directory with a test Makefile
	tmpDir := t.TempDir()

	// Sentinel file names
	const (
		canonicalSentinel = "SENTINEL_CANONICAL_GATE"
		fastSentinel      = "SENTINEL_FAST_GATE"
		dupcodeSentinel   = "SENTINEL_DUPCODE_GATE"
	)

	// Makefile that uses production-equivalent sequential recursive invocation
	// This matches the actual guard topology in make/long-tests.mk
	makefile := `PHONY := guard gate gate-canonical gate-fast gate-dupcode

guard:
	@test "$(LEAMAS_GATE_CALLER)" != "editor" -o "$(LEAMAS_ALLOW_FULL_GATE)" = "1" || (echo "REFUSED" >&2; exit 2)

gate:
	@$(MAKE) --no-print-directory guard
	@$(MAKE) --no-print-directory gate-canonical

gate-canonical:
	@echo "running canonical gate"
	@echo "canonical" >> SENTINEL_CANONICAL_GATE

gate-fast:
	@echo "running fast gate"
	@echo "fast" >> SENTINEL_FAST_GATE

gate-dupcode:
	@echo "running dupcode gate"
	@echo "dupcode" >> SENTINEL_DUPCODE_GATE
`

	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	// Clean up any existing sentinels
	cleanSentinels := func() {
		for _, s := range []string{canonicalSentinel, fastSentinel, dupcodeSentinel} {
			os.Remove(filepath.Join(tmpDir, s))
		}
	}

	// Helper to check sentinel existence
	sentinelExists := func(name string) bool {
		_, err := os.Stat(filepath.Join(tmpDir, name))
		return err == nil
	}

	t.Run("editor+gatedefaultsToRefusal", func(t *testing.T) {
		cleanSentinels()
		env := append(baseEnv(), "LEAMAS_GATE_CALLER=editor")
		result := exectest.RunMake(context.Background(), tmpDir, env, "gate")

		// Should refuse with exit code 2
		if result.Outcome != exectest.OutcomeExitFailure {
			t.Errorf("expected OutcomeExitFailure, got %v", result.Outcome)
		}
		if result.ExitCode != 2 {
			t.Errorf("expected exit code 2, got %d", result.ExitCode)
		}
		// Canonical sentinel should NOT exist
		if sentinelExists(canonicalSentinel) {
			t.Error("canonical sentinel should not exist after refusal")
		}
		// Should see REFUSED in stderr
		if !strings.Contains(string(result.Stderr), "REFUSED") {
			t.Errorf("expected REFUSED in stderr, got: %s", result.Stderr)
		}
	})

	t.Run("editor+overrideRunsCanonical", func(t *testing.T) {
		cleanSentinels()
		env := append(baseEnv(), "LEAMAS_GATE_CALLER=editor", "LEAMAS_ALLOW_FULL_GATE=1")
		result := exectest.RunMake(context.Background(), tmpDir, env, "gate")

		// Should succeed
		if result.Outcome != exectest.OutcomeSuccess {
			t.Errorf("expected OutcomeSuccess, got %v", result.Outcome)
		}
		// Canonical sentinel should exist exactly once
		if !sentinelExists(canonicalSentinel) {
			t.Fatal("canonical sentinel should exist after override gate")
		}
		// Verify exactly one execution
		contents, err := os.ReadFile(filepath.Join(tmpDir, canonicalSentinel))
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(string(contents), "canonical"); got != 1 {
			t.Fatalf("canonical executions = %d, want 1", got)
		}
	})

	t.Run("gate-fastRunsFastOnly", func(t *testing.T) {
		cleanSentinels()
		env := append(baseEnv(), "LEAMAS_GATE_CALLER=editor")
		result := exectest.RunMake(context.Background(), tmpDir, env, "gate-fast")

		// Should succeed
		if result.Outcome != exectest.OutcomeSuccess {
			t.Errorf("expected OutcomeSuccess, got %v", result.Outcome)
		}
		// Fast sentinel should exist
		if !sentinelExists(fastSentinel) {
			t.Error("fast sentinel should exist after gate-fast")
		}
		// Canonical and dupcode sentinels should NOT exist
		if sentinelExists(canonicalSentinel) {
			t.Error("canonical sentinel should not exist after gate-fast")
		}
		if sentinelExists(dupcodeSentinel) {
			t.Error("dupcode sentinel should not exist after gate-fast")
		}
	})

	t.Run("emptyCallerWithOverrideRunsCanonical", func(t *testing.T) {
		cleanSentinels()
		// Empty caller (base env) is valid
		result := exectest.RunMake(context.Background(), tmpDir, baseEnv(), "gate")

		// Should succeed (empty caller is valid)
		if result.Outcome != exectest.OutcomeSuccess {
			t.Errorf("expected OutcomeSuccess, got %v", result.Outcome)
		}
		// Canonical sentinel should exist
		if !sentinelExists(canonicalSentinel) {
			t.Error("canonical sentinel should exist after gate with empty caller")
		}
	})
}

// TestTimeoutVerifiesTimeout tests that RunMake correctly classifies timeout.
func TestTimeoutVerifiesTimeout(t *testing.T) {
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

	// Run with a timeout shorter than the command (1 second)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result := exectest.RunMake(ctx, tmpDir, nil, "slow-target")

	if result.Outcome != exectest.OutcomeTimeout {
		t.Errorf("expected OutcomeTimeout, got %v", result.Outcome)
	}
}

// TestSpawnFailureVerifiesSpawnFailure tests spawn failure handling.
func TestSpawnFailureVerifiesSpawnFailure(t *testing.T) {
	// Use Run with an absolute path to nonexistent executable
	result := exectest.Run(context.Background(), "", nil, "/nonexistent/absolutely_missing_binary_12345")

	if result.Outcome != exectest.OutcomeSpawnFailure {
		t.Fatalf("outcome = %v, want spawn failure", result.Outcome)
	}
	if result.SpawnErr == nil {
		t.Fatal("SpawnErr is nil")
	}
}

// TestWaitDelayDetection tests WaitDelay classification with retained pipe.
func TestWaitDelayDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a retained-pipe fixture: shell starts background process inheriting stderr.
	// This causes stderr to remain open after shell exits, triggering WaitDelay.
	makefile := `
.PHONY: retained-output
retained-output:
	@$(SHELL) -c 'sleep 5 >&2 &'; exit 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	started := time.Now()
	result := exectest.RunMake(context.Background(), tmpDir, baseEnv(), "retained-output")
	elapsed := time.Since(started)

	// Strict assertions: retained pipe must trigger WaitDelay outcome
	if result.Outcome != exectest.OutcomeWaitDelay {
		t.Fatalf(
			"outcome = %v, want wait delay; stdout=%q stderr=%q",
			result.Outcome,
			result.Stdout,
			result.Stderr,
		)
	}
	if !result.WaitDelay {
		t.Fatal("WaitDelay evidence is false")
	}
	if elapsed > exectest.DefaultWaitDelay+3*time.Second {
		t.Fatalf("elapsed = %v, retained-pipe return was not bounded", elapsed)
	}
}

// TestOutputOverflowStdout tests stdout overflow detection.
func TestOutputOverflowStdout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Makefile that outputs more than the default limit (1 MiB)
	makefile := `
.PHONY: big-output
big-output:
	@yes | head -c 2097152
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	result := exectest.RunMake(context.Background(), tmpDir, nil, "big-output")

	// Strict assertions: overflow must be classified correctly
	if result.Outcome != exectest.OutcomeOutputOverflow {
		t.Fatalf(
			"outcome = %v, want %v; stderr=%q",
			result.Outcome,
			exectest.OutcomeOutputOverflow,
			result.Stderr,
		)
	}
	if result.Overflow == nil {
		t.Fatal("missing overflow evidence")
	}
	if got := len(result.Stdout); got != int(exectest.DefaultOutputLimit) {
		t.Fatalf(
			"captured stdout = %d, want %d",
			got,
			exectest.DefaultOutputLimit,
		)
	}
	if result.Overflow.Observed <= result.Overflow.Limit {
		t.Fatalf(
			"observed = %d, limit = %d",
			result.Overflow.Observed,
			result.Overflow.Limit,
		)
	}
}

// TestCancellationVerifiesCancelled tests explicit cancellation.
func TestCancellationVerifiesCancelled(t *testing.T) {
	tmpDir := t.TempDir()

	makefile := `
.PHONY: long-target
long-target:
	@sleep 60
	@echo "done"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	// Create a context and cancel it after starting the command
	ctx, cancel := context.WithCancel(context.Background())

	// Start the command in a goroutine
	resultCh := make(chan *exectest.Result, 1)
	go func() {
		// Use empty env to avoid inheriting VSCODE_PID which might affect things
		resultCh <- exectest.RunMake(ctx, tmpDir, []string{"TERM_PROGRAM=", "VSCODE_PID="}, "long-target")
	}()

	// Cancel after a brief delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	result := <-resultCh

	if result.Outcome != exectest.OutcomeCancelled {
		t.Errorf("expected OutcomeCancelled, got %v", result.Outcome)
	}
}

// TestFastGateRequiresSuccess explicitly verifies fast-gate failure classification.
func TestFastGateRequiresSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	makefile := `
.PHONY: gate-fast
gate-fast:
	@echo "fast gate"
	@exit 1
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	result := exectest.RunMake(context.Background(), tmpDir, baseEnv(), "gate-fast")

	// A failing target must be classified as exit failure
	if result.Outcome != exectest.OutcomeExitFailure {
		t.Fatalf(
			"outcome = %v, want exit failure; stdout=%q stderr=%q",
			result.Outcome,
			result.Stdout,
			result.Stderr,
		)
	}
	if result.ExitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
}

// TestInvalidCallerFailsClosed verifies invalid caller with override fails.
func TestInvalidCallerFailsClosed(t *testing.T) {
	tmpDir := t.TempDir()

	makefile := `PHONY := guard gate gate-canonical
guard:
	@test "$(LEAMAS_GATE_CALLER)" != "invalid_caller" || (echo "invalid caller value" >&2; exit 2)

gate: guard gate-canonical
	@echo "gate target completed"

gate-canonical:
	@echo "canonical"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	// Invalid caller with override should fail
	result := exectest.RunMake(context.Background(), tmpDir,
		[]string{"LEAMAS_GATE_CALLER=invalid_caller", "LEAMAS_ALLOW_FULL_GATE=1"}, "gate")

	if result.Outcome != exectest.OutcomeExitFailure {
		t.Errorf("expected OutcomeExitFailure for invalid caller, got %v", result.Outcome)
	}
	if !strings.Contains(string(result.Stderr), "invalid") {
		t.Errorf("expected 'invalid' error, got: %s", result.Stderr)
	}
}

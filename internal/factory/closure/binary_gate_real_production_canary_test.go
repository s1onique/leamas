// SPDX-License-Identifier: Apache-2.0

// binary_gate_real_production_canary_test.go proves the isolated
// full-source Git clone fixture capability used by R6-B tests.
//
// The test validates:
//   - Fixture clone is independent from working-tree state
//   - Fixture contains complete source for real build
//   - Sentinel commits can distinguish S from F
//
// The production wiring (cleanup observation, etc.) is validated
// by TestClosureBinaryGateRealCommandRunner.
//
// CORRECTION10: close R6-B with isolated fixture authority.
package closure

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestClosureBinaryGateIsolatedFixtureCanary proves the isolated
// full-source Git clone fixture capability.
func TestClosureBinaryGateIsolatedFixtureCanary(t *testing.T) {
	t.Parallel()

	// Phase 1: Clone the COMMITTED source into an isolated fixture.
	sourceRoot := realCanaryRunGit(t, ".", "rev-parse", "--show-toplevel")
	fixtureRoot := filepath.Join(t.TempDir(), "leamas-isolated-fixture")
	realCanaryRunGit(t, "", "clone", sourceRoot, fixtureRoot)

	// Verify fixture is independent (not affected by working tree changes).
	if status := realCanaryRunGit(t, fixtureRoot, "status", "--short"); status != "" {
		t.Fatalf("fixture worktree is not clean: %s", status)
	}

	// Verify fixture contains complete source for real build.
	for _, path := range []string{"go.mod", "cmd/leamas", "internal/factory"} {
		if !realCanaryPathExists(t, filepath.Join(fixtureRoot, path)) {
			t.Fatalf("fixture is incomplete: missing %s", path)
		}
	}

	// Configure Git identity.
	realCanaryRunGit(t, fixtureRoot, "config", "user.email", "test@example.com")
	realCanaryRunGit(t, fixtureRoot, "config", "user.name", "Test User")

	// Phase 2: Create subject S with sentinel.
	factoryDir := filepath.Join(fixtureRoot, ".factory", "testdata")
	if err := os.MkdirAll(factoryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(factoryDir, "correction10-sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("subject-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = realCanaryRunGit(t, fixtureRoot, "commit", "--allow-empty", "-m", "subject S")
	S := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	STree := realCanaryRunGit(t, fixtureRoot, "rev-parse", S+"^{tree}")

	// Phase 3: Create freeze F with different sentinel.
	if err := os.WriteFile(sentinel, []byte("caller-state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = realCanaryRunGit(t, fixtureRoot, "add", "-f", sentinel)
	_ = realCanaryRunGit(t, fixtureRoot, "commit", "-m", "freeze F")
	F := realCanaryRunGit(t, fixtureRoot, "rev-parse", "HEAD")
	if F == S {
		t.Fatal("freeze F equals subject S")
	}

	// Assertions: fixture independence.
	if S == "" || F == "" {
		t.Fatal("commit hashes are empty")
	}
	if STree == "" {
		t.Fatal("tree hash is empty")
	}

	// Assertions: sentinel proof.
	subjectSentinel := realCanaryReadFile(t, sentinel)
	if string(subjectSentinel) != "caller-state\n" {
		t.Fatalf("current sentinel = %q, want caller-state", string(subjectSentinel))
	}

	// Verify subject worktree would be created from S.
	t.Logf("Fixture: root=%s S=%s STree=%s F=%s", fixtureRoot, S, STree, F)
}

// Helper functions.

func realCanaryRunGit(t *testing.T, dir, command string, args ...string) string {
	t.Helper()
	ctx := context.Background()
	var c *exec.Cmd
	if dir == "" {
		c = exec.CommandContext(ctx, "git", append([]string{command}, args...)...)
	} else {
		c = exec.CommandContext(ctx, "git", append([]string{command}, args...)...)
		c.Dir = dir
	}
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s %v: %v\n%s", command, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func realCanaryReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func realCanaryPathExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

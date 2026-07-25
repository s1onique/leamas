// SPDX-License-Identifier: Apache-2.0

// Package closure: temporarygit_helper_test.go hosts the small,
// test-only repository fixture used by ancestry-semantics tests.
// The fixture builds a temporary Git repo with deterministic
// commits A0, S0, and E0 and exposes a single helper to assert
// merge-base ancestry, so unit tests no longer need access to
// the real production repository's object database.
//
// The helper lives in a _test.go file alongside its tests; the
// v2 test harness drives the package through subprocess
// invocations of the leamas test binary when it needs to
// sanity-check the fixture and the helpers remain package-private.
// Helper names are not exposed in any production code path.
//
// Git is invoked exclusively through authority.DefaultGitRunner
// to satisfy the executable-contract-first gate which forbids
// direct os/exec use outside internal/execution.
package closure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/factory/authority"
)

// temporaryGitFixture is a self-contained Git repository with
// three referenceable commits: A0 (initial), S0 (predecessor
// subject), and E0 (later commit in which the predecessor plan
// first appears). All ancestors of E0 are reachable from E0.
type temporaryGitFixture struct {
	dir  string
	oids map[string]string
}

// newTemporaryGitFixture initializes a fresh Git repository in
// t.TempDir() and creates the three commits. Calling t.Cleanup
// tears down the temp directory.
func newTemporaryGitFixture(t *testing.T) *temporaryGitFixture {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "-q")
	runGitCmd(t, dir, "config", "user.email", "fixture@example.com")
	runGitCmd(t, dir, "config", "user.name", "Fixture")
	runGitCmd(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "a0.txt"), []byte("a0\n"), 0o644); err != nil {
		t.Fatalf("write a0: %v", err)
	}
	runGitCmd(t, dir, "add", "a0.txt")
	runGitCmd(t, dir, "commit", "-q", "-m", "initial A0")
	a0 := strings.TrimSpace(mustGitOutput(t, dir, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(dir, "s0.txt"), []byte("s0\n"), 0o644); err != nil {
		t.Fatalf("write s0: %v", err)
	}
	runGitCmd(t, dir, "add", "s0.txt")
	runGitCmd(t, dir, "commit", "-q", "-m", "predecessor subject S0")
	s0 := strings.TrimSpace(mustGitOutput(t, dir, "rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(dir, "plan.json"),
		[]byte(`{"contract_version": 1}`), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	runGitCmd(t, dir, "add", "plan.json")
	runGitCmd(t, dir, "commit", "-q", "-m", "introduce predecessor plan E0")
	e0 := strings.TrimSpace(mustGitOutput(t, dir, "rev-parse", "HEAD"))
	return &temporaryGitFixture{
		dir: dir,
		oids: map[string]string{
			"A0": a0,
			"S0": s0,
			"E0": e0,
		},
	}
}

// isAncestor asserts ancestor is reachable from descendant via
// `git merge-base --is-ancestor`. Returns false on any error and
// surfaces a string error with stderr context.
func (f *temporaryGitFixture) isAncestor(t *testing.T, ancestor, descendant string) (bool, error) {
	t.Helper()
	_, err := authority.DefaultGitRunner(f.dir, "merge-base", "--is-ancestor",
		ancestor, descendant)
	if err != nil {
		return false, fmt.Errorf("git merge-base --is-ancestor %s %s: %w",
			ancestor, descendant, err)
	}
	return true, nil
}

// runGitCmd is a small wrapper that fails the test on non-zero exit.
func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := authority.DefaultGitRunner(dir, args...); err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
}

// mustGitOutput runs git and fails the test on non-zero exit.
func mustGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := authority.DefaultGitRunner(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return out
}

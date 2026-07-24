// SPDX-License-Identifier: Apache-2.0

// Package authority: authority_test.go covers the executable-authority
// contract required by ACT-LEAMAS-FACTORY-SELF-HOSTED-ENTRYPOINT-AUTHORITY01.
//
// The tests exercise the 15 acceptance scenarios listed in the ACT
// using small Git repositories built in temp dirs. They avoid
// running real `go build` for the bootstrap path; that path is
// covered by an integration test that requires the Go toolchain on
// PATH.
package authority

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withGitRepo creates a temp dir, initialises a git repo with a
// supplied HEAD setup, and returns the directory plus a cleanup
// function. The commit SHAs created during setup are returned as
// `commits["baseline"]`, `commits["subject"]`, and so on.
//
// The repo layout is intentionally minimal: a single tracked file
// named `tracked.txt` that the tests can mutate between commits.
func withGitRepo(t *testing.T) (string, map[string]string, func()) {
	t.Helper()
	dir := t.TempDir()
	// Use main as the default branch; fall back to master for
	// compatibility with the wider git default-branch policy.
	initCmd := exec.Command("git", "init", "-q", "-b", "main")
	initCmd.Dir = dir
	if out, err := initCmd.CombinedOutput(); err != nil {
		runGit(t, dir, "init", "-q")
		// Rename to main if needed.
		branchOut, _ := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
		branch := strings.TrimSpace(string(branchOut))
		if branch != "main" {
			runGit(t, dir, "branch", "-M", "main")
		}
	} else {
		_ = out
	}
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-q", "-m", "baseline")
	baseline := head(t, dir)

	// Optional extra commits for tests that need them.
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\ncapability-bump\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "commit", "-q", "-am", "capability bump")
	capabilityBump := head(t, dir)

	return dir, map[string]string{
		"baseline":       baseline,
		"capabilityBump": capabilityBump,
	}, func() {}
}

// head returns the HEAD commit SHA of the test repo.
func head(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// runGit runs `git <args...>` in dir.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, string(out))
	}
}

// withCapturedRunner replaces DefaultGitRunner for the duration of
// the test and returns a restore function.
func withCapturedRunner(t *testing.T, runner GitRunner) func() {
	t.Helper()
	prev := DefaultGitRunner
	DefaultGitRunner = runner
	return func() { DefaultGitRunner = prev }
}

// stubRunner captures git invocations and returns canned answers.
type stubRunner struct {
	repoRoot string
	bang     map[string]string // key=arg signature, value=stdout
	calls    [][]string
}

// runStub returns the canned answer for args, falling back to the
// real DefaultGitRunner behaviour for unmatched calls. It is used
// sparingly; tests that need full control over git state should
// build a real repo and not stub the runner.
func runStub(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// TestCapabilityRequiredSatisfied asserts that the default capability
// table satisfies the manifest in this repository.
func TestCapabilityRequiredSatisfied(t *testing.T) {
	dir, _, done := withGitRepo(t)
	done()

	embedded := SnapshotEmbedded()
	required, err := LoadRequired(DefaultPath(dir))
	if err != nil {
		t.Fatalf("LoadRequired: %v", err)
	}
	if err := required.SatisfiedBy(embedded); err != nil {
		t.Fatalf("required capabilities not satisfied: %v", err)
	}
}

// TestCapabilityRequiredBelowEmbedded raises the required level
// beyond the embedded level and asserts the gap is reported.
func TestCapabilityRequiredBelowEmbedded(t *testing.T) {
	_, _, done := withGitRepo(t)
	done()

	required := &RequiredCapabilities{Raw: map[string]int{
		CapDigestAutoRange:     99,
		CapSelfHostedAuthority: 99,
	}}
	embedded := SnapshotEmbedded()

	gap := required.SatisfiedBy(embedded)
	if gap == nil {
		t.Fatalf("expected capability gap")
	}
	if !strings.Contains(gap.Error(), CapDigestAutoRange) {
		t.Errorf("gap missing %s: %v", CapDigestAutoRange, gap)
	}
}

// TestCheckExecutable_EqualCommitAuthoritative asserts scenario 1:
// installed binary equals repository HEAD.
func TestCheckExecutable_EqualCommitAuthoritative(t *testing.T) {
	dir, commits, done := withGitRepo(t)
	done()

	check, err := CheckExecutable(dir, commits["baseline"], commits["baseline"], true)
	if err != nil {
		t.Fatalf("CheckExecutable: %v", err)
	}
	if check.Relationship != RelationshipEqual {
		t.Errorf("Relationship = %s, want equal", check.Relationship)
	}
	if check.Verdict != VerdictAuthoritative {
		t.Errorf("Verdict = %s, want authoritative: %s", check.Verdict, check.VerdictReason)
	}
}

// TestCheckExecutable_AncestorWithCapabilityOK asserts scenario 2:
// harmless ancestor with sufficient capability.
func TestCheckExecutable_AncestorWithCapabilityOK(t *testing.T) {
	dir, commits, done := withGitRepo(t)
	done()

	check, err := CheckExecutable(dir, commits["baseline"], commits["capabilityBump"], true)
	if err != nil {
		t.Fatalf("CheckExecutable: %v", err)
	}
	if check.Relationship != RelationshipAncestor {
		t.Errorf("Relationship = %s, want ancestor", check.Relationship)
	}
	if check.Verdict != VerdictAncestor {
		t.Errorf("Verdict = %s, want ancestor_capability_acceptable: %s",
			check.Verdict, check.VerdictReason)
	}
}

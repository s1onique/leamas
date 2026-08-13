// SPDX-License-Identifier: Apache-2.0

// factory_simple_test.go is the bounded test suite for the
// simplified factory CLI pair (factory begin, factory close).
// The tests are deliberately thin: they assert CLI plumbing
// (parsing, validation, JSON envelope shape, dispatch) but
// delegate the authority semantics to the closure package,
// which has its own real-Git proof suite.

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// chdirRepo changes the working directory to repoRoot for the
// duration of the test. Tests that touch a real Git repository
// must run from inside it so closure.RunGitShowToplevel and
// the RealGit client resolve the canonical root.
func chdirRepo(t *testing.T, repoRoot string) func() {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("chdir %s: %v", repoRoot, err)
	}
	return func() { _ = os.Chdir(orig) }
}

func TestFactoryBeginValidAct(t *testing.T) {
	repo := newCLIRealRepo(t)
	defer chdirRepo(t, repo)()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runFactoryBegin([]string{"ACT-PROD"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%s)", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "freeze_commit=") {
		t.Fatalf("output missing freeze_commit: %s", out)
	}
	if !strings.Contains(out, "act_id=ACT-PROD") {
		t.Fatalf("output missing act_id=ACT-PROD: %s", out)
	}
	if !strings.Contains(out, "state=frozen") {
		t.Fatalf("output missing state=frozen: %s", out)
	}
	// Real-Git invariant: a freeze ref must now exist.
	out2, err := exec.Command("git", "-C", repo, "rev-parse", "--verify", "--end-of-options", "refs/factory/freeze/ACT-PROD").Output()
	if err != nil {
		t.Fatalf("freeze ref missing: %v", err)
	}
	f := strings.TrimSpace(string(out2))
	if f == "" {
		t.Fatalf("freeze ref empty")
	}
}

func TestFactoryBeginInvalidAct(t *testing.T) {
	repo := newCLIRealRepo(t)
	defer chdirRepo(t, repo)()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	// Traversal sequence is the canonical rejection case.
	code := runFactoryBegin([]string{"ACT-../escape"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit; got 0 (stdout=%s)", stdout.String())
	}
	// No ref may have been created.
	_, revErr := exec.Command("git", "-C", repo, "rev-parse", "--verify", "--end-of-options", "refs/factory/freeze/ACT-../escape").Output()
	if revErr == nil {
		t.Fatalf("invalid ACT produced freeze ref; refused")
	}
}

func TestFactoryBeginJSON(t *testing.T) {
	repo := newCLIRealRepo(t)
	defer chdirRepo(t, repo)()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runFactoryBegin([]string{"--json", "ACT-PROD"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr=%s)", code, stderr.String())
	}
	var got simpleBeginResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v (stdout=%s)", err, stdout.String())
	}
	if got.ActID != "ACT-PROD" {
		t.Fatalf("act_id = %q, want ACT-PROD", got.ActID)
	}
	if got.FreezeCommit == "" {
		t.Fatalf("freeze_commit empty: %+v", got)
	}
	if got.PlanPath != "docs/closure-plans/ACT-PROD.json" {
		t.Fatalf("plan_path = %q, want canonical", got.PlanPath)
	}
	if got.State != "frozen" {
		t.Fatalf("state = %q, want frozen", got.State)
	}
}

func TestFactoryCloseUnsupportedLane(t *testing.T) {
	repo := newCLIRealRepo(t)
	defer chdirRepo(t, repo)()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runFactorySimpleClose([]string{
		"--act", "ACT-PROD",
		"--subject", strings.Repeat("a", 40),
		"--lane", "slow",
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit for unsupported lane")
	}
	if !strings.Contains(stderr.String(), "unsupported_lane") {
		t.Fatalf("stderr missing unsupported_lane: %s", stderr.String())
	}
}

func TestFactoryCloseMissingRequired(t *testing.T) {
	repo := newCLIRealRepo(t)
	defer chdirRepo(t, repo)()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	code := runFactorySimpleClose([]string{
		"--act", "ACT-PROD",
		// missing --subject and --lane
	}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit for missing required flags")
	}
}

func TestIsLegacyCloseSubcommand(t *testing.T) {
	cases := []struct {
		arg  string
		want bool
	}{
		{"plan", true},
		{"run", true},
		{"tag", true},
		{"verify", true},
		{"status", true},
		{"execute", true},
		{"--act", false},
		{"--subject", false},
		{"--lane", false},
		{"--publish", false},
		{"--json", false},
		{"", false},
	}
	for _, tc := range cases {
		got := isLegacyCloseSubcommand(tc.arg)
		if got != tc.want {
			t.Errorf("isLegacyCloseSubcommand(%q) = %v, want %v", tc.arg, got, tc.want)
		}
	}
}

// newCLIRealRepo builds a minimal real Git repository for
// the CLI tests. It is intentionally lightweight: a single
// commit A on main, no remote.
func newCLIRealRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main", repo},
		{"-C", repo, "config", "user.email", "cli-test@example.invalid"},
		{"-C", repo, "config", "user.name", "cli-test"},
		{"-C", repo, "config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	seed := filepath.Join(repo, "README.md")
	if err := os.WriteFile(seed, []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	for _, args := range [][]string{
		{"-C", repo, "add", "README.md"},
		{"-C", repo, "commit", "-q", "-m", "initial"},
	} {
		cmd := exec.Command("git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return repo
}

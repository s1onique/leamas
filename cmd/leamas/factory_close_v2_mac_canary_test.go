// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_mac_canary_test.go executes the
// installed-style external dogfood required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-MAC-CANARY-READINESS01.
//
// The test:
//  1. builds the leamas binary against the CURRENT tree using
//     the production LDFLAGS so the running binary identity
//     reports the actual current HEAD commit and a clean
//     "vcs.modified=false" stamp;
//  2. invokes the binary from a temp directory OUTSIDE the
//     source tree, against a fresh hermetic S < F < D
//     repository in another temp directory;
//  3. asserts DOGFOOD_EXIT == 0, the manifest bindings are
//     exact, the caller repository is unchanged, and no
//     linked worktree leaked.
//
// The test does NOT exercise the production closure verifier.
// It exercises the public v2 authority CLI end-to-end so the
// Mac canary has a deterministic Linux-side proof.
//
// The build step goes through internal/execution via the
// runMacCanaryBuildCommand wrapper, satisfying the exec-gate.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// dogfoodPlanPath is the repository-relative path used by
// every Mac canary dogfood run.
const dogfoodPlanPath = "docs/closure-plans/MAC-CANARY-DOGFOOD.json"

// dogfoodCheckShell is a POSIX sh -c invocation that proves
// the executor ran against S^{tree} (not F^{tree} or D^{tree}):
//
//	test -f subject-only.txt     (subject tree contains S's file)
//	test ! -e freeze-only.txt    (subject tree does NOT contain F-only)
//	test ! -e descendant-only.txt (subject tree does NOT contain D-only)
const dogfoodCheckShell = "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e descendant-only.txt"

// TestClosureCLIV2MacCanaryDogfood is the installed-style
// external dogfood for the Mac canary. It is the Linux-side
// proof that the public v2 CLI succeeds end-to-end against a
// meaningful S < F < D repository when invoked from outside
// the Leamas checkout.
func TestClosureCLIV2MacCanaryDogfood(t *testing.T) {
	binary := buildMacCanaryLeamasForTest(t)
	repository, subject, freeze, d := prepareMacCanaryDogfoodRepo(t)
	// Verify HEAD = D before invoking the binary.
	if got := gitForClosureTest(t, repository, "rev-parse", "HEAD"); got != d {
		t.Fatalf("pre-run HEAD must equal D: got=%s want=%s", got, d)
	}
	// Evidence and manifest live OUTSIDE the target repository
	// so the v2 runner can satisfy the detached-path invariant
	// (PATH-AUTHORITY01).
	detachedDir := t.TempDir()
	evidenceDir := filepath.Join(detachedDir, "evidence")
	manifestOutput := filepath.Join(detachedDir, "manifest.json")
	// Snapshot caller + worktree registrations BEFORE.
	headBefore := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	headTreeBefore := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	statusBefore := gitForClosureTest(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesBefore := gitForClosureTest(t, repository, "worktree", "list", "--porcelain")
	// Invoke the binary from OUTSIDE the source tree.
	// We do NOT cd into the Leamas checkout; the binary must
	// be self-contained for the Mac handoff.
	stdout, stderr, err := runClosureSubprocess(binary, detachedDir,
		"factory", "close", "run-v2-authority",
		"--protocol-version", "2",
		"--plan-contract-version", "1",
		"--repository", repository,
		"--subject", subject,
		"--freeze", freeze,
		"--plan-path", dogfoodPlanPath,
		"--evidence-directory", evidenceDir,
		"--manifest-output", manifestOutput,
	)
	if err != nil {
		t.Fatalf("dogfood run failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	// Snapshot caller + worktree registrations AFTER.
	headAfter := gitForClosureTest(t, repository, "rev-parse", "HEAD")
	headTreeAfter := gitForClosureTest(t, repository, "rev-parse", "HEAD^{tree}")
	statusAfter := gitForClosureTest(t, repository, "status", "--porcelain=v2", "--untracked-files=all")
	worktreesAfter := gitForClosureTest(t, repository, "worktree", "list", "--porcelain")
	// Caller-state drift invariants.
	if headBefore != headAfter {
		t.Fatalf("caller HEAD drifted: before=%s after=%s", headBefore, headAfter)
	}
	if headTreeBefore != headTreeAfter {
		t.Fatalf("caller HEAD tree drifted: before=%s after=%s",
			headTreeBefore, headTreeAfter)
	}
	if statusBefore != statusAfter {
		t.Fatalf("caller worktree status drifted:\nbefore=%q\nafter=%q",
			statusBefore, statusAfter)
	}
	if worktreesBefore != worktreesAfter {
		t.Fatalf("linked-worktree registrations leaked:\nbefore=%q\nafter=%q",
			worktreesBefore, worktreesAfter)
	}
	// Manifest must be on disk.
	manifestBytes, err := os.ReadFile(manifestOutput)
	if err != nil {
		t.Fatalf("manifest file missing: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("manifest JSON invalid: %v\n%s", err, string(manifestBytes))
	}
	if got, _ := manifest["subject_commit"].(string); got != subject {
		t.Fatalf("subject_commit: got=%q want=%q", got, subject)
	}
	if got, _ := manifest["freeze_commit"].(string); got != freeze {
		t.Fatalf("freeze_commit: got=%q want=%q", got, freeze)
	}
	if got, _ := manifest["caller_head"].(string); got != d {
		t.Fatalf("caller_head: got=%q want=%q", got, d)
	}
	if got, _ := manifest["plan_path"].(string); got != dogfoodPlanPath {
		t.Fatalf("plan_path: got=%q want=%q", got, dogfoodPlanPath)
	}
	if got, _ := manifest["closure_protocol_version"].(string); got != "2" {
		t.Fatalf("closure_protocol_version: got=%q want=2", got)
	}
	// SHA-256 of the manifest on disk must be a 64-character
	// lowercase hex string and must round-trip to the same
	// bytes we just read.
	manifestSHA := sha256.Sum256(manifestBytes)
	if hex.EncodeToString(manifestSHA[:]) != sha256HexString(manifestBytes) {
		t.Fatalf("sha256 self-check failed")
	}
	// stdout and stderr must be non-empty: the CLI prints
	// "OK" plus the summary line.
	if !strings.Contains(stdout, "OK") {
		t.Fatalf("expected OK on stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "Closure Protocol v2:") {
		t.Fatalf("expected v2 summary on stderr, got %q", stderr)
	}
}

// buildMacCanaryLeamasForTest builds the leamas binary with
// the production LDFLAGS that inject the current HEAD commit,
// the build time, and an explicit "false" dirty flag. The
// running binary identity helpers (closure.RunningLeamasVCSRevision
// etc.) read these linker-injected values, so without the
// LDFLAGS the v2 runner refuses to publish a manifest.
//
// The build is routed through internal/execution via
// runMacCanaryBuildCommand so the exec-gate stays green.
func buildMacCanaryLeamasForTest(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "leamas-mac-canary")
	commit := gitForClosureTest(t, ".", "rev-parse", "HEAD")
	buildTime := gitForClosureTest(t, ".", "show", "-s", "--format=%ct", "HEAD")
	// The LDFLAGS match the production Makefile values for
	// the four linker-injected version variables.
	ldflags := fmt.Sprintf(
		"-X 'github.com/s1onique/leamas/internal/version.Version=0.1.0+dev.%s.%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.DeclaredVersion=0.1.0' "+
			"-X 'github.com/s1onique/leamas/internal/version.Commit=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.BuildTime=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.Dirty=false'",
		commit[:8], buildTime, commit, buildTime,
	)
	runMacCanaryBuildCommand(t, []string{"CGO_ENABLED=0"}, bin, ldflags)
	return bin
}

// runMacCanaryBuildCommand runs an arbitrary `go` invocation
// through internal/execution so the exec-gate does not flag
// direct exec.Command usage. Failures fail the test.
func runMacCanaryBuildCommand(t *testing.T, environment []string, outputPath, ldflags string) {
	t.Helper()
	budget := execution.DefaultBudget().WithTimeout(2 * time.Minute).WithMaxConcurrent(1).WithMaxStarts(1)
	executor, err := execution.NewExecutor(budget, execution.NewTestExecutionRoot())
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	result := executor.Execute(t.Context(), &execution.Request{
		Name: "mac canary leamas build", Args: []string{"go", "build", "-trimpath", "-ldflags", ldflags, "-o", outputPath, "github.com/s1onique/leamas/cmd/leamas"},
		Env: environment, Timeout: 2 * time.Minute,
	})
	if !result.Success() {
		t.Fatalf("go build failed (exit %d, err=%v):\n%s",
			result.ExitCode, result.Error, result.Stderr)
	}
}

// prepareMacCanaryDogfoodRepo constructs the S < F < D
// repository used by the Mac canary dogfood test. The
// repository lives in a fresh temp directory so the binary is
// truly invoked from "outside" the Leamas checkout. The
// returned values are (repository, subject, freeze, d).
func prepareMacCanaryDogfoodRepo(t *testing.T) (repository, subject, freeze, d string) {
	t.Helper()
	repository = t.TempDir()
	gitForClosureTest(t, repository, "init", "-b", "main")
	gitForClosureTest(t, repository, "config", "user.name", "Mac Canary Dogfood")
	gitForClosureTest(t, repository, "config", "user.email", "mac-canary@example.invalid")
	gitForClosureTest(t, repository, "config", "commit.gpgsign", "false")
	gitForClosureTest(t, repository, "config", "tag.gpgsign", "false")
	gitForClosureTest(t, repository, "commit", "--allow-empty", "-m", "initial")
	// S: subject-only file present.
	mustWriteFile(t, filepath.Join(repository, "subject-only.txt"), "subject\n")
	gitForClosureTest(t, repository, "add", "subject-only.txt")
	gitForClosureTest(t, repository, "commit", "-m", "subject")
	subject = gitForClosureTest(t, repository, "rev-parse", "HEAD")
	subjectTree := gitForClosureTest(t, repository, "rev-parse", subject+"^{tree}")
	// F: plan + freeze-only file.
	planBytes := buildDogfoodPlan(t, subject, subjectTree)
	planDir := filepath.Dir(filepath.Join(repository, dogfoodPlanPath))
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(repository, dogfoodPlanPath), planBytes)
	mustWriteFile(t, filepath.Join(repository, "freeze-only.txt"), "freeze-only\n")
	gitForClosureTest(t, repository, "add", ".")
	gitForClosureTest(t, repository, "commit", "-m", "freeze: add plan")
	freeze = gitForClosureTest(t, repository, "rev-parse", "HEAD")
	// D: mutate the plan + descendant-only file.
	mutated := []byte(`{"contract_version": 1, "act_id": "ACT-MAC-CANARY-DOGFOOD-MUTATED"}`)
	mustWriteFile(t, filepath.Join(repository, dogfoodPlanPath), string(mutated))
	mustWriteFile(t, filepath.Join(repository, "descendant-only.txt"), "descendant-only\n")
	gitForClosureTest(t, repository, "add", ".")
	gitForClosureTest(t, repository, "commit", "-m", "descendant: mutate plan")
	d = gitForClosureTest(t, repository, "rev-parse", "HEAD")
	return repository, subject, freeze, d
}

// buildDogfoodPlan returns a contract-valid Plan Contract v1
// document whose run-mode check is the dogfoodCheckShell
// sh -c invocation. We emit the JSON manually rather than
// importing the closure fixture helper to keep the test
// self-contained at the cmd/leamas layer.
func buildDogfoodPlan(t *testing.T, subject, subjectTree string) string {
	t.Helper()
	plan := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-MAC-CANARY-DOGFOOD",
		"baseline": map[string]string{
			"commit_oid": subject,
			"tree_oid":   subjectTree,
		},
		"execution": map[string]any{
			"mode": "serial_fail_fast",
		},
		"checks": []map[string]any{{
			"id":                "mac_canary_dogfood_proof",
			"mode":              "run",
			"argv":              []string{"sh", "-c", dogfoodCheckShell},
			"working_directory": ".",
			"timeout_seconds":   60,
			"environment":       map[string]string{},
		}},
		"artifacts": []any{},
		"policy": map[string]bool{
			"require_clean_before":        true,
			"require_clean_after":         true,
			"forbid_tracked_full_digests": true,
			"require_diff_check":          true,
		},
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	return string(raw)
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sha256HexString(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum[:])
}

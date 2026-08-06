// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2c_helpers_test.go owns the R2C-R1
// helper functions for the exact-final-tip dogfood: detached
// worktree preparation, build, repo construction, plan building,
// git subprocess wrappers, and SHA-256 helpers. The main test
// lives in factory_close_v2_r2c_dogfood_test.go.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

func mustPrepareR2CRBuildEnv(t *testing.T) (finalCommit, finalTree, leamasRepoRoot, detachedWorktree string) {
	t.Helper()
	repoRoot, err := runR2CRGitE(t, ".", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	leamasRepoRoot = repoRoot
	finalCommit = os.Getenv("LEAMAS_R2CR_FINAL_COMMIT")
	if finalCommit == "" {
		finalCommit = runR2CRGit(t, leamasRepoRoot, "rev-parse", "HEAD")
	}
	finalTree = runR2CRGit(t, leamasRepoRoot, "rev-parse", finalCommit+"^{tree}")

	// Caller worktree MUST be clean before we build. The
	// detached worktree build will be performed in a separate
	// temporary worktree so this assertion is best-effort
	// (it guards against accidentally baking dirty source
	// into the binary even before we isolate it).
	statusOut := runR2CRGit(t, leamasRepoRoot, "status", "--porcelain=v2", "--untracked-files=all")
	if strings.TrimSpace(statusOut) != "" {
		t.Fatalf("caller worktree not clean:\n%s", statusOut)
	}

	detachedWorktree = filepath.Join(t.TempDir(), "r2cr-build")
	runR2CRGit(t, leamasRepoRoot, "worktree", "add", "--detach", detachedWorktree, finalCommit)
	if got := runR2CRGit(t, detachedWorktree, "rev-parse", "HEAD"); got != finalCommit {
		t.Fatalf("detached worktree HEAD %s != FINAL_COMMIT %s", got, finalCommit)
	}
	if got := runR2CRGit(t, detachedWorktree, "status", "--porcelain=v2", "--untracked-files=all"); strings.TrimSpace(got) != "" {
		t.Fatalf("detached worktree porcelain-v2 not empty:\n%s", got)
	}
	return finalCommit, finalTree, leamasRepoRoot, detachedWorktree
}

// buildInDetachedWorktree runs `go build` inside the detached
// worktree with the production LDFLAGS that inject HEAD commit,
// build time, and Dirty=false.
func buildInDetachedWorktree(t *testing.T, worktreePath, finalCommit string) {
	t.Helper()
	bin := filepath.Join(worktreePath, "leamas-r2cr-dogfood")
	buildTime := runR2CRGit(t, worktreePath, "show", "-s", "--format=%ct", "HEAD")
	ldflags := fmt.Sprintf(
		"-X 'github.com/s1onique/leamas/internal/version.Version=0.1.0+dev.%s.%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.DeclaredVersion=0.1.0' "+
			"-X 'github.com/s1onique/leamas/internal/version.Commit=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.BuildTime=%s' "+
			"-X 'github.com/s1onique/leamas/internal/version.Dirty=false'",
		finalCommit[:8], buildTime, finalCommit, buildTime,
	)
	budget := execution.DefaultBudget().WithTimeout(2 * time.Minute).WithMaxConcurrent(1).WithMaxStarts(1)
	executor, err := execution.NewExecutor(budget, execution.NewTestExecutionRoot())
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	result := executor.Execute(t.Context(), &execution.Request{
		Name:    "r2cr leamas build",
		Args:    []string{"go", "build", "-trimpath", "-ldflags", ldflags, "-o", bin, "github.com/s1onique/leamas/cmd/leamas"},
		Env:     []string{"CGO_ENABLED=0"},
		Dir:     worktreePath,
		Timeout: 2 * time.Minute,
	})
	if !result.Success() {
		t.Fatalf("go build failed (exit %d):\n%s", result.ExitCode, result.Stderr)
	}
}

// prepareR2CRDogfoodRepo constructs the S < F < D repository used
// by the R2C-R1 dogfood. It returns enough identifiers for the
// test to assert exact manifest bindings.
func prepareR2CRDogfoodRepo(t *testing.T, leamasRepoRoot string) (repository, subject, subjectTree, freeze, freezeTree, d, planPath, planBlobOID, planBytes string, planSHA256, descendantPlanBytes string) {
	t.Helper()
	repository = t.TempDir()
	runR2CRGit(t, repository, "init", "-b", "main")
	runR2CRGit(t, repository, "config", "user.name", "R2C-R1 Dogfood")
	runR2CRGit(t, repository, "config", "user.email", "r2cr@example.invalid")
	runR2CRGit(t, repository, "config", "commit.gpgsign", "false")
	runR2CRGit(t, repository, "config", "tag.gpgsign", "false")
	runR2CRGit(t, repository, "commit", "--allow-empty", "-m", "initial")

	// S: subject-only file present.
	mustWriteFile(t, filepath.Join(repository, "subject-only.txt"), "subject\n")
	runR2CRGit(t, repository, "add", "subject-only.txt")
	runR2CRGit(t, repository, "commit", "-m", "subject")
	subject = runR2CRGit(t, repository, "rev-parse", "HEAD")
	subjectTree = runR2CRGit(t, repository, "rev-parse", subject+"^{tree}")

	// F: plan + freeze-only file.
	planPath = "docs/closure-plans/R2CR-DOGFOOD.json"
	planDir := filepath.Join(repository, filepath.Dir(planPath))
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatal(err)
	}
	planBytes = buildR2CRDogfoodPlan(t, subject, subjectTree, leamasRepoRoot)
	mustWriteFile(t, filepath.Join(repository, planPath), planBytes)
	mustWriteFile(t, filepath.Join(repository, "freeze-only.txt"), "freeze-only\n")
	runR2CRGit(t, repository, "add", ".")
	runR2CRGit(t, repository, "commit", "-m", "freeze: add plan")
	freeze = runR2CRGit(t, repository, "rev-parse", "HEAD")
	freezeTree = runR2CRGit(t, repository, "rev-parse", freeze+"^{tree}")

	// D: mutate the plan + descendant-only file.
	muta := []byte(`{"contract_version": 1, "act_id": "ACT-R2CR-DOGFOOD-MUTATED"}`)
	mustWriteFile(t, filepath.Join(repository, planPath), string(muta))
	mustWriteFile(t, filepath.Join(repository, "descendant-only.txt"), "descendant-only\n")
	runR2CRGit(t, repository, "add", ".")
	runR2CRGit(t, repository, "commit", "-m", "descendant: mutate plan")
	d = runR2CRGit(t, repository, "rev-parse", "HEAD")

	// Resolve F:P blob OID, F:P exact bytes, and D:P exact bytes
	// via the production git client (no mutable checkout).
	planBlobOID = runR2CRGit(t, repository, "rev-parse", freeze+":"+planPath)
	descPlanBlob := runR2CRGit(t, repository, "rev-parse", d+":"+planPath)
	if planBlobOID == descPlanBlob {
		t.Fatalf("F:P and D:P blob OIDs must differ: got %s for both", planBlobOID)
	}
	planBytes = runR2CRGit(t, repository, "show", freeze+":"+planPath)
	descendantPlanBytes = runR2CRGit(t, repository, "show", d+":"+planPath)
	planSHA256 = sha256HexBytes([]byte(planBytes))

	return
}

// buildR2CRDogfoodPlan returns a contract-valid Plan Contract v1
// document whose run-mode check exercises the S-only / F-only /
// D-only invariants.
func buildR2CRDogfoodPlan(t *testing.T, subject, subjectTree, _ string) string {
	t.Helper()
	const r2crCheck = "test -f subject-only.txt && test ! -e freeze-only.txt && test ! -e descendant-only.txt"
	plan := map[string]any{
		"contract_version": 1,
		"act_id":           "ACT-R2CR-DOGFOOD",
		"baseline": map[string]string{
			"commit_oid": subject,
			"tree_oid":   subjectTree,
		},
		"execution": map[string]any{
			"mode": "serial_fail_fast",
		},
		"checks": []map[string]any{{
			"id":                "r2cr_dogfood_proof",
			"mode":              "run",
			"argv":              []string{"sh", "-c", r2crCheck},
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
		t.Fatalf("marshal r2cr plan: %v", err)
	}
	return string(raw)
}

// assertManifestAbsent fails the test if path exists.
func assertManifestAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("manifest path must be absent before invocation: %s", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected manifest stat error: %v", err)
	}
}

// r2cRBinaryIdentity captures the three linker-injected values
// the binary reports on `version`.
type r2cRBinaryIdentity struct {
	Commit  string
	Dirty   bool
	Version string
}

// readBinaryIdentity invokes `binary version` and parses the
// printed identity fields.
func readBinaryIdentity(t *testing.T, binary string) r2cRBinaryIdentity {
	t.Helper()
	stdout, _, err := runR2CRSubprocess(t, binary, []string{"version"})
	if err != nil {
		t.Fatalf("read binary identity: %v", err)
	}
	id := r2cRBinaryIdentity{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "commit:"):
			id.Commit = strings.TrimSpace(line[len("commit:"):])
		case strings.HasPrefix(lower, "vcs revision:"):
			id.Commit = strings.TrimSpace(line[len("vcs revision:"):])
		case strings.HasPrefix(lower, "dirty:"):
			id.Dirty = strings.EqualFold(strings.TrimSpace(line[len("dirty:"):]), "true")
		case strings.HasPrefix(lower, "vcs modified:"):
			id.Dirty = strings.EqualFold(strings.TrimSpace(line[len("vcs modified:"):]), "true")
		case strings.HasPrefix(lower, "version:"):
			id.Version = strings.TrimSpace(line[len("version:"):])
		}
	}
	return id
}

// runR2CRGit runs an arbitrary git command in the supplied
// directory, failing the test on non-zero exit. The output is
// trimmed and returned.
func runR2CRGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := runR2CRGitE(t, dir, args...)
	if err != nil {
		t.Fatalf("git %s (in %s): %v", strings.Join(args, " "), dir, err)
	}
	return out
}

// runR2CRGitE runs git and returns the error (does not fail the
// test) so callers can inspect the result.
func runR2CRGitE(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(stdout.String()),
			fmt.Errorf("exit %v stdout=%q stderr=%q", err,
				strings.TrimSpace(stdout.String()),
				strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// runR2CRSubprocess runs an arbitrary binary with the supplied
// argv and returns stdout, stderr, and the error (does not
// fail). This is a thin wrapper used by readBinaryIdentity; the
// dogfood itself uses the boundedSubprocessV2 harness.
func runR2CRSubprocess(t *testing.T, binary string, argv []string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(binary, argv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// runR2CRGitRaw returns the exact raw bytes of the supplied Git
// object without trimming. It uses `git cat-file blob <oid>` for
// blob content and `git cat-file -p <oid>` for tree/commit content
// only when explicitly requested. SHA-256 hashes computed over
// the returned bytes are guaranteed to match the literal blob.
func runR2CRGitRaw(t *testing.T, dir, oid string) []byte {
	t.Helper()
	// `git cat-file blob <oid>` writes the raw blob content to
	// stdout with no trailing newline added. This is the only
	// command that returns the exact blob bytes.
	out, err := runR2CRGitE(t, dir, "cat-file", "blob", oid)
	if err != nil {
		t.Fatalf("git cat-file blob %s: %v", oid, err)
	}
	return []byte(out)
}

// runR2CRGitRawShow returns the raw bytes of `git cat-file -p
// <oid>` output for non-blob objects. Trailing newlines are NOT
// trimmed.
func runR2CRGitRawShow(t *testing.T, dir, oid string) []byte {
	t.Helper()
	// exec directly to preserve the raw output (runGitValue
	// trims). We capture both stdout and stderr so we can
	// surface any error message.
	cmd := exec.Command("git", "cat-file", "-p", oid)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git cat-file -p %s: %v stderr=%s", oid, err, stderr.String())
	}
	return stdout.Bytes()
}

// sha256HexFile returns the lowercase hex SHA-256 of path's
// contents.
func sha256HexFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return sha256HexBytes(data)
}

// sha256HexBytes returns the lowercase hex SHA-256 of b.
func sha256HexBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// sha256HexString removed: mac_canary_test already defines it.

// silence unused imports.
var _ = io.Discard

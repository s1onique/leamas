// SPDX-License-Identifier: Apache-2.0

package main

// factory_close_v2_r2cr_correction02a_helpers_test.go owns
// the build-with-external-output helper and the
// build-source snapshot helper for
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-VERIFIER-MAC-HANDOFF01-CORRECTION02A.
//
// The original `buildInDetachedWorktree` writes the binary
// inside the detached worktree, which makes the worktree
// dirty after the build. The correction02a ACT requires
// the binary to live outside the build worktree so the
// worktree can stay observably clean before and after the
// build. This file introduces the corrected helper and
// keeps the legacy helper for backward compatibility with
// the original test.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// buildInDetachedWorktreeTo is the corrected build helper.
// The source worktree is at worktreePath; the binary is
// written to outputDir/leamas-r2cr-dogfood so the source
// worktree's porcelain-v2 status stays observably clean
// after the build returns.
//
// The function returns the absolute path to the freshly
// built binary. The outputDir is created if it does not
// exist.
func buildInDetachedWorktreeTo(t *testing.T, worktreePath, outputDir, finalCommit string) string {
	t.Helper()
	if err := exec.Command("mkdir", "-p", outputDir).Run(); err != nil {
		t.Fatalf("mkdir %s: %v", outputDir, err)
	}
	bin := filepath.Join(outputDir, "leamas-r2cr-dogfood")
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
		Name:    "r2cr-correction02a leamas build",
		Args:    []string{"go", "build", "-trimpath", "-ldflags", ldflags, "-o", bin, "github.com/s1onique/leamas/cmd/leamas"},
		Env:     []string{"CGO_ENABLED=0"},
		Dir:     worktreePath,
		Timeout: 2 * time.Minute,
	})
	if !result.Success() {
		t.Fatalf("go build failed (exit %d):\n%s", result.ExitCode, result.Stderr)
	}
	return bin
}

// correction02aBuildSourceSnapshot captures the literal
// detached build source state at one moment. The struct
// is the input to the "build source stayed clean"
// assertion. Every field is observed via git, not
// fabricated.
type correction02aBuildSourceSnapshot struct {
	HeadCommit     string
	HeadTree       string
	Detached       bool
	PorcelainV2    string
	PorcelainV2SHA string
	IsEmpty        bool
}

// captureCorrection02aBuildSourceSnapshot reads the
// detached worktree's HEAD commit, HEAD tree, detached
// state, and porcelain-v2 status. The function never
// returns an "empty" status without reading it.
func captureCorrection02aBuildSourceSnapshot(t *testing.T, worktreePath string) correction02aBuildSourceSnapshot {
	t.Helper()
	head := runR2CRGit(t, worktreePath, "rev-parse", "HEAD")
	tree := runR2CRGit(t, worktreePath, "rev-parse", "HEAD^{tree}")
	porcelain := runR2CRGit(t, worktreePath, "status", "--porcelain=v2", "--untracked-files=all")
	// `git symbolic-ref --quiet --short HEAD` exits
	// non-zero when HEAD is detached. We capture that
	// condition via the exit code rather than parsing
	// output.
	detached := isHeadDetached(worktreePath)
	return correction02aBuildSourceSnapshot{
		HeadCommit:     head,
		HeadTree:       tree,
		Detached:       detached,
		PorcelainV2:    porcelain,
		PorcelainV2SHA: sha256HexBytes([]byte(porcelain)),
		IsEmpty:        strings.TrimSpace(porcelain) == "",
	}
}

// isHeadDetached returns true when HEAD in the supplied
// worktree is a detached HEAD. The check is a
// non-fatal `git symbolic-ref --quiet --short HEAD`
// whose non-zero exit indicates a detached HEAD.
func isHeadDetached(worktreePath string) bool {
	cmd := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD")
	cmd.Dir = worktreePath
	return cmd.Run() != nil
}

// buildSFWithPlan creates the S and F commits in the
// supplied repository using a caller-specified plan
// path. The plan contract is the same one
// buildCorrection01SF uses; only the path differs so
// multiple dogfood harnesses can co-exist in the same
// test binary.
func buildSFWithPlan(t *testing.T, repo, _ string, planPath string) (subject, subjectTree, freeze, freezeTree string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(repo, "subject-only.txt"), "subject\n")
	runR2CRGit(t, repo, "add", "subject-only.txt")
	runR2CRGit(t, repo, "commit", "-m", "subject")
	subject = runR2CRGit(t, repo, "rev-parse", "HEAD")
	subjectTree = runR2CRGit(t, repo, "rev-parse", subject+"^{tree}")

	planDir := filepath.Join(repo, filepath.Dir(planPath))
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	planJSON := buildCorrection01Plan(subject, subjectTree)
	mustWriteFile(t, filepath.Join(repo, planPath), planJSON)
	mustWriteFile(t, filepath.Join(repo, "freeze-only.txt"), "freeze-only\n")
	runR2CRGit(t, repo, "add", ".")
	runR2CRGit(t, repo, "commit", "-m", "freeze: add plan")
	freeze = runR2CRGit(t, repo, "rev-parse", "HEAD")
	freezeTree = runR2CRGit(t, repo, "rev-parse", freeze+"^{tree}")
	return subject, subjectTree, freeze, freezeTree
}

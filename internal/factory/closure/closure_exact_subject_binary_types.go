// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary_types.go defines the typed
// request and result shapes for the BuildExactSubjectBinary
// production authority.
//
// BINARY IDENTITY AUTHORITY:
//   BinaryCommit / BinaryModified are the canonical authority
//   observed from the produced binary's own `version --json`
//   output (the same surface the production release artefacts
//   expose). They are NOT derived from `go version -m -json`
//   because cmd/go does not reliably stamp linked worktrees.
//
// NATIVE BUILD-INFO:
//   NativeVCSRevision / NativeVCSModified are auxiliary
//   diagnostics decoded from `go version -m -json <binary>`.
//   Their presence is recorded in NativeVCSRevisionPresent /
//   NativeVCSModifiedPresent. Their absence MUST NOT fail the
//   exact-S authority: linked-worktree builds do not stamp
//   vcs.revision reliably.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// ExactSubjectBinaryRequest is the typed input for the
// BuildExactSubjectBinary production authority.
type ExactSubjectBinaryRequest struct {
	RepositoryRoot string
	SubjectCommit  string
	SubjectTree    string
	OutputRoot     string
	OutputName     string
	// CleanupTimeout overrides the default cleanup timeout
	// when non-zero. The override exists so unit tests can
	// shrink the budget; production callers leave it at zero.
	CleanupTimeout time.Duration
}

// ExactSubjectBinaryResult is the typed successful output of
// the BuildExactSubjectBinary production authority.
//
// The struct is split into five logical groups:
//
//   - Source authority: HEAD, tree, clean, detached
//   - Binary identity: BinaryCommit, BinaryModified
//   - Binary provenance: path, SHA-256, executable
//   - Native build-info: auxiliary diagnostics from
//     `go version -m -json`
//   - Bounded-result proof: per-stage exit / error / output
//     metadata proving every subprocess returned cleanly
//   - Cleanup proof: attempts, success, leak status,
//     inventory closure
type ExactSubjectBinaryResult struct {
	// SOURCE AUTHORITY (observed from the detached build
	// worktree).
	SourceCommit   string
	SourceTree     string
	SourceClean    bool
	SourceDetached bool

	// BINARY IDENTITY (observed from the produced binary's
	// own `version --json` output). These are the canonical
	// B1 predicates. cmd/go VCS stamping is NOT used here.
	BinaryCommit   string
	BinaryModified bool

	// BINARY PROVENANCE.
	BinaryPath                string
	BinarySHA256              string
	Executable                bool
	OutputOutsideAllWorktrees bool

	// NATIVE BUILD-INFO AUXILIARY. Present=true means the
	// key was found; the corresponding *String / *Bool field
	// is meaningful only when Present=true. Absence of the
	// key MUST NOT fail the exact-S authority.
	NativeVCSRevision        string
	NativeVCSRevisionPresent bool
	NativeVCSModified        bool
	NativeVCSModifiedPresent bool

	// BOUNDED-RESULT PROOF: each stage captures the typed
	// execution.Result so umbrella tests can prove every
	// subprocess returned cleanly.
	BuildBounded      bool
	IdentityBounded   bool
	BuildErrorCode    string
	IdentityErrorCode string

	// CLEANUP PROOF: tracks attempts and outcome.
	CleanupAttempted              bool
	CleanupSucceeded              bool
	CleanupError                  string
	CleanupAttempts               int
	CleanupContextFresh           bool
	BuildWorktreeLeak             bool
	PostCleanupInventoryClosed    bool
	PostCleanupInventoryError     string
	PostCleanupInventoryLeakPaths []string
}

// HasBinaryIdentityFailures returns true when any of the
// exact-S binary authority predicates fail. Callers can use
// the bool to decide whether to publish or surface a typed
// diagnostic.
func (r ExactSubjectBinaryResult) HasBinaryIdentityFailures() bool {
	return r.BinaryCommit == "" || r.BinaryModified || !r.Executable || !r.OutputOutsideAllWorktrees
}

// exactBinaryValidate rejects empty required inputs before
// any side effect.
func exactBinaryValidate(req ExactSubjectBinaryRequest) error {
	if req.RepositoryRoot == "" {
		return errors.New("exact-binary: repository root is empty")
	}
	if req.SubjectCommit == "" {
		return errors.New("exact-binary: subject commit is empty")
	}
	if req.SubjectTree == "" {
		return errors.New("exact-binary: subject tree is empty")
	}
	if req.OutputRoot == "" {
		return errors.New("exact-binary: output root is empty")
	}
	return nil
}

// canonicalPath resolves symlinks and returns the absolute
// canonical form of an existing directory path.
func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// canonicalNonExistentPath resolves symlinks for the largest
// existing prefix of the path, then appends the missing
// suffix. This lets the B1 check reject output paths whose
// nearest existing ancestor is a symlink into a worktree or
// the caller repo, even when the final output directory has
// not yet been created.
func canonicalNonExistentPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	head, tail := abs, ""
	cursor := abs
	for {
		if _, err := os.Lstat(cursor); err == nil {
			canonicalCursor, err := filepath.EvalSymlinks(cursor)
			if err != nil {
				return "", err
			}
			return filepath.Join(canonicalCursor, tail), nil
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			return head, nil
		}
		base := filepath.Base(cursor)
		if tail == "" {
			tail = base
		} else {
			tail = filepath.Join(base, tail)
		}
		cursor = parent
	}
}

// exactBinaryResolveOutputRoots rejects outputs that land
// inside the caller repo or any linked worktree and returns
// the canonical (output, caller) paths. The rejection uses
// filepath.EvalSymlinks so the output path is resolved
// against symlinked ancestors, not just lexical.
func exactBinaryResolveOutputRoots(repositoryRoot, outputRoot string, worktreePaths []string) (string, string, string, error) {
	absCallerRepo, err := canonicalPath(repositoryRoot)
	if err != nil {
		return "", "", "", fmt.Errorf("exact-binary: canonical caller repo: %w", err)
	}
	canonicalWorktreePaths := make([]string, 0, len(worktreePaths))
	for _, wt := range worktreePaths {
		cwp, err := canonicalPath(wt)
		if err != nil {
			return "", "", "", fmt.Errorf("exact-binary: canonical worktree %s: %w", wt, err)
		}
		canonicalWorktreePaths = append(canonicalWorktreePaths, cwp)
	}
	canonicalOutput, err := canonicalNonExistentPath(outputRoot)
	if err != nil {
		return "", "", "", fmt.Errorf("exact-binary: canonical output root: %w", err)
	}
	if pathInsideOrEqual(canonicalOutput, absCallerRepo) {
		return "", "", "", fmt.Errorf("exact-binary: output root %q is inside caller repo %q", canonicalOutput, absCallerRepo)
	}
	for _, wt := range canonicalWorktreePaths {
		if pathInsideOrEqual(canonicalOutput, wt) {
			return "", "", "", fmt.Errorf("exact-binary: output root %q is inside linked worktree %q", canonicalOutput, wt)
		}
	}
	if err := os.MkdirAll(canonicalOutput, 0o755); err != nil {
		return "", "", "", fmt.Errorf("exact-binary: mkdir output root: %w", err)
	}
	return canonicalOutput, absCallerRepo, "", nil
}

// pathInsideOrEqual tests whether the canonical child is
// inside or equal to the canonical parent. Both inputs MUST
// be canonicalised before this call.
func pathInsideOrEqual(canonicalChild, canonicalParent string) bool {
	rel, err := filepath.Rel(canonicalParent, canonicalChild)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// exactBinaryVerifySource asserts the build worktree state
// before the build.
func exactBinaryVerifySource(ctx context.Context, git gitClient, buildWorktree, subjectCommit, subjectTree string) error {
	headResult, err := runGitValue(ctx, git, buildWorktree, "rev-parse", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("exact-binary: rev-parse HEAD: %w", err)
	}
	if headResult != subjectCommit {
		return fmt.Errorf("exact-binary: HEAD %s != subject %s", headResult, subjectCommit)
	}
	treeResult, err := runGitValue(ctx, git, buildWorktree, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return fmt.Errorf("exact-binary: rev-parse HEAD^{tree}: %w", err)
	}
	if treeResult != subjectTree {
		return fmt.Errorf("exact-binary: tree %s != subject tree %s", treeResult, subjectTree)
	}
	if err := exactBinaryRequireDetachedHEAD(ctx, git, buildWorktree); err != nil {
		return err
	}
	statusResult, err := runGitValue(ctx, git, buildWorktree, "status", "--porcelain=v2")
	if err != nil {
		return fmt.Errorf("exact-binary: status porcelain: %w", err)
	}
	if statusResult != "" {
		return fmt.Errorf("exact-binary: build worktree is dirty: %s", statusResult)
	}
	return nil
}

// exactBinaryRequireDetachedHEAD asserts that the worktree at
// worktreePath has a detached HEAD. `git symbolic-ref -q HEAD`
// returns 1 for a detached HEAD and 0 for a symbolic-ref HEAD.
func exactBinaryRequireDetachedHEAD(ctx context.Context, git gitClient, worktreePath string) error {
	res := git.Run(ctx, worktreePath, "symbolic-ref", "-q", "HEAD")
	if res.ExitCode != 1 {
		return fmt.Errorf("exact-binary: HEAD must be detached at %s (symbolic-ref exit=%d, expected 1)",
			worktreePath, res.ExitCode)
	}
	return nil
}

// exactBinaryVerifySourceAfterBuild asserts the build
// worktree state after the build.
func exactBinaryVerifySourceAfterBuild(ctx context.Context, git gitClient, buildWorktree, subjectCommit, subjectTree string) error {
	headAfter, err := runGitValue(ctx, git, buildWorktree, "rev-parse", "HEAD^{commit}")
	if err != nil || headAfter != subjectCommit {
		return fmt.Errorf("exact-binary: post-build HEAD drift: %s vs %s (err=%v)",
			headAfter, subjectCommit, err)
	}
	treeAfter, err := runGitValue(ctx, git, buildWorktree, "rev-parse", "HEAD^{tree}")
	if err != nil || treeAfter != subjectTree {
		return fmt.Errorf("exact-binary: post-build tree drift: %s vs %s (err=%v)",
			treeAfter, subjectTree, err)
	}
	if err := exactBinaryRequireDetachedHEAD(ctx, git, buildWorktree); err != nil {
		return fmt.Errorf("exact-binary: post-build %w", err)
	}
	statusAfter, err := runGitValue(ctx, git, buildWorktree, "status", "--porcelain=v2")
	if err != nil || statusAfter != "" {
		return fmt.Errorf("exact-binary: post-build status dirty: %q (err=%v)",
			statusAfter, err)
	}
	return nil
}

// exactBinaryBuildBudget bounds the build subprocess.
func exactBinaryBuildBudget() *execution.Budget {
	return execution.DefaultBudget().
		WithTimeout(10 * time.Minute).
		WithMaxOutputBytes(64 * 1024 * 1024)
}

// exactBinaryIdentityBudget bounds the identity-introspection
// subprocess. `leamas version --json` runs in milliseconds.
func exactBinaryIdentityBudget() *execution.Budget {
	return execution.DefaultBudget().
		WithTimeout(30 * time.Second).
		WithMaxOutputBytes(64 * 1024)
}

// exactBinaryNativeBuildInfoBudget bounds the auxiliary
// native buildinfo subprocess. `go version -m -json <binary>`
// runs in milliseconds.
func exactBinaryNativeBuildInfoBudget() *execution.Budget {
	return execution.DefaultBudget().
		WithTimeout(30 * time.Second).
		WithMaxOutputBytes(64 * 1024)
}

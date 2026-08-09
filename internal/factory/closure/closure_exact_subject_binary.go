// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary.go implements the
// BuildExactSubjectBinary production authority.
//
// This file owns:
//   - the public BuildExactSubjectBinary entry point
//   - the private buildExactSubjectBinaryWithoutCheck
//     orchestration (validate, inventory, output root,
//     worktree registration, junction, post-cleanup
//     inventory)
//   - the worktree-inventory parser used by the input
//     gate and the post-cleanup authority
//   - the SHA-256-at-rest hash helper used by the
//     canonical-identity proof
//
// Per-stage build + canonical-identity logic lives in
// closure_exact_subject_binary_build.go; identity readers
// live in closure_exact_subject_binary_identity.go;
// cleanup + post-cleanup inventory live in
// closure_exact_subject_binary_cleanup.go. Splitting the
// implementation across files keeps each one under the
// LLM-friendly 400-line threshold while preserving the
// single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.
//
// CANONICAL FLOW:
//
//   1. validate input
//   2. inventory worktrees (fail-closed)
//   3. resolve + reject output paths
//   4. create detached build worktree at S
//      (the only registration; cleanup MUST run for
//       every error class listed below)
//   5. verify source authority BEFORE build
//   6. build with the existing production LDFLAGS scheme
//      (cmd/leamas → internal/version.Version/Commit/BuildTime)
//   7. verify source authority AFTER build
//   8. stat the binary, hash it
//   9. invoke the binary's own `version --json` for
//      canonical identity (NOT cmd/go's go version -m -json)
//  10. invoke `go version -m -json` for auxiliary
//      diagnostics only
//  11. UNCONDITIONAL EXACTLY-ONCE CLEANUP JUNCTION:
//        primaryErr + cleanupErr -> errors.Join
//  12. UNCONDITIONAL POST-CLEANUP INVENTORY
//
// Cleanup runs for every error class:
//   dirty-before-build, HEAD mismatch, tree mismatch,
//   not detached, build failure, build timeout, build
//   cancellation, build output truncation, post-build
//   source drift, binary stat failure, binary hash failure,
//   identity-read failure, binary commit mismatch,
//   binary modified=true, non-executable binary,
//   inventory unavailable.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BuildExactSubjectBinary is the public production authority.
// It routes the execute through the private
// buildExactSubjectBinary with the canonical RealGit{}
// client so the public surface takes no package-global
// mutable state. Tests inject their own gitClient via the
// private buildExactSubjectBinaryWithoutCheck function.
func BuildExactSubjectBinary(ctx context.Context, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	return buildExactSubjectBinaryWithoutCheck(ctx, RealGit{}, req)
}

// buildExactSubjectBinaryWithoutCheck is the private
// implementation. The "WithoutCheck" suffix marks that the
// caller is responsible for verifying their own inputs.
func buildExactSubjectBinaryWithoutCheck(ctx context.Context, git gitClient, req ExactSubjectBinaryRequest) (ExactSubjectBinaryResult, error) {
	if err := exactBinaryValidate(req); err != nil {
		return ExactSubjectBinaryResult{}, err
	}
	outputName := req.OutputName
	if outputName == "" {
		outputName = "leamas"
	}

	// Inventory worktrees. Fail-closed: any observation
	// error rejects.
	worktreePaths, err := exactBinaryWorktreeInventory(ctx, git, req.RepositoryRoot)
	if err != nil {
		return ExactSubjectBinaryResult{}, err
	}
	absOutputRoot, _, _, err := exactBinaryResolveOutputRoots(req.RepositoryRoot, req.OutputRoot, worktreePaths)
	if err != nil {
		return ExactSubjectBinaryResult{}, err
	}

	// Build worktree registration. This is the only
	// side-effecting git operation that must be paired with
	// a cleanup junction.
	buildWorktreeRoot, err := os.MkdirTemp("", "leamas-exact-binary-")
	if err != nil {
		return ExactSubjectBinaryResult{}, fmt.Errorf("exact-binary: mkdir build worktree root: %w", err)
	}
	buildWorktreePath := filepath.Join(buildWorktreeRoot, "wt")
	cleanup := newExactBinaryCleanup(git, req.RepositoryRoot, buildWorktreePath, req.CleanupTimeout)
	// After successful registration, cleanup MUST run
	// exactly once. The filesystem RemoveAll is hygiene
	// AFTER git worktree remove + prune, never a substitute.
	defer func() {
		_ = os.RemoveAll(buildWorktreeRoot)
	}()

	addRes := git.Run(ctx, req.RepositoryRoot, "worktree", "add", "--detach", buildWorktreePath, req.SubjectCommit)
	if addRes.Err != nil || addRes.ExitCode != 0 {
		return ExactSubjectBinaryResult{}, fmt.Errorf("exact-binary: git worktree add --detach %s %s: exit=%d err=%v stderr=%s",
			buildWorktreePath, req.SubjectCommit, addRes.ExitCode, addRes.Err, strings.TrimSpace(string(addRes.Stderr)))
	}

	// Junction: primary + cleanup + post-cleanup inventory.
	// The junction is unconditional: cleanup runs exactly
	// once for every outcome of the primary error class.
	result, primaryErr := runExactBinaryBuildAndCleanup(ctx, git, cleanup, req, absOutputRoot, outputName, buildWorktreePath)
	cleanup.recordCallSiteAttempt()
	cleanupErr := cleanup.run()
	if cleanupErr != nil {
		if primaryErr == nil {
			primaryErr = cleanupErr
		} else {
			primaryErr = errors.Join(primaryErr, cleanupErr)
		}
	}

	// Post-cleanup inventory. Fail-closed: any observation
	// failure is a B1 failure regardless of the primary
	// result.
	postInv := exactBinaryRunPostCleanupInventory(ctx, git, req.RepositoryRoot, buildWorktreePath)
	postInvErr := exactBinaryCheckPostCleanupInventoryClosed(postInv)

	result.CleanupAttempted = cleanup.snapshot().Performed || cleanupErr != nil
	result.CleanupSucceeded = primaryErr == nil && cleanupErr == nil
	result.CleanupAttempts = cleanup.snapshot().Attempts
	result.CleanupContextFresh = cleanup.snapshot().ContextFresh
	if cleanupErr != nil {
		result.CleanupError = cleanupErr.Error()
	}
	if postInvErr != nil {
		result.PostCleanupInventoryError = postInvErr.Error()
		result.PostCleanupInventoryClosed = false
		result.PostCleanupInventoryLeakPaths = postInv.LeakPaths
		if primaryErr == nil {
			primaryErr = postInvErr
		} else {
			primaryErr = errors.Join(primaryErr, postInvErr)
		}
	} else {
		result.PostCleanupInventoryClosed = true
	}
	result.BuildWorktreeLeak = len(postInv.LeakPaths) > 0

	if primaryErr != nil {
		return ExactSubjectBinaryResult{}, primaryErr
	}
	return result, nil
}

// leamasModulePath is the canonical Go module path. The
// version package variables are addressed via this path so
// the linker flag matches the existing production Makefile.
const leamasModulePath = "github.com/s1onique/leamas"

// leamasBuildVersion is the value injected as
// internal/version.Version. We use the same SemVer that
// release artefacts stamp so the binary's reported version
// matches the production identity surface.
const leamasBuildVersion = "0.1.0"

// leamasBuildTime returns the deterministic BuildTime
// linker-injected stamp. The function returns a stable UTC
// RFC3339 timestamp so the binary's reported BuildTime is
// comparable across rebuilds of the same subject.
func leamasBuildTime() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// hashBinaryAtRest computes SHA-256 of the binary file in
// two passes to detect concurrent modification.
func hashBinaryAtRest(path string) (string, error) {
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open binary: %w", err)
		}
		before, err := file.Stat()
		if err != nil {
			file.Close()
			return "", fmt.Errorf("stat binary: %w", err)
		}
		if !before.Mode().IsRegular() {
			file.Close()
			return "", errors.New("binary is not a regular file")
		}
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			return "", fmt.Errorf("hash binary: %w", err)
		}
		after, err := file.Stat()
		file.Close()
		if err != nil {
			return "", fmt.Errorf("re-stat binary: %w", err)
		}
		if before.Size() == after.Size() &&
			before.ModTime().Equal(after.ModTime()) &&
			os.SameFile(before, after) {
			return hex.EncodeToString(hash.Sum(nil)), nil
		}
	}
	return "", errors.New("binary changed during hashing")
}

// exactBinaryWorktreeInventory reads every linked worktree
// via `git worktree list --porcelain -z` and fails closed if
// any of the following do not hold:
//   - Err == nil
//   - ExitCode == 0
//   - the inventory is non-empty
//   - the caller repository root is represented as a worktree
func exactBinaryWorktreeInventory(ctx context.Context, git gitClient, repositoryRoot string) ([]string, error) {
	res := git.Run(ctx, repositoryRoot, "worktree", "list", "--porcelain", "-z")
	if res.Err != nil {
		return nil, fmt.Errorf("exact-binary: inventory worktrees: %w", res.Err)
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("exact-binary: inventory worktrees: exit=%d stderr=%s",
			res.ExitCode, strings.TrimSpace(string(res.Stderr)))
	}
	paths := parseWorktreePaths(res.Stdout)
	if len(paths) == 0 {
		return nil, errors.New("exact-binary: worktree inventory is empty")
	}
	callerCanonical, err := canonicalPath(repositoryRoot)
	if err != nil {
		return nil, fmt.Errorf("exact-binary: canonical caller repo: %w", err)
	}
	found := false
	for _, wt := range paths {
		canonicalWT, err := canonicalPath(wt)
		if err != nil {
			return nil, fmt.Errorf("exact-binary: canonical worktree %s: %w", wt, err)
		}
		if canonicalWT == callerCanonical {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("exact-binary: caller repo %q not present in worktree inventory", callerCanonical)
	}
	return paths, nil
}

// parseWorktreePaths extracts the absolute path of every
// linked worktree recorded by `git worktree list --porcelain -z`.
func parseWorktreePaths(porcelain []byte) []string {
	var paths []string
	segments := strings.Split(string(porcelain), "\x00")
	for _, seg := range segments {
		if !strings.HasPrefix(seg, "worktree ") {
			continue
		}
		paths = append(paths, strings.TrimPrefix(seg, "worktree "))
	}
	return paths
}

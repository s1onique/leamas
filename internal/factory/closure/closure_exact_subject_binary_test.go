// SPDX-License-Identifier: Apache-2.0

package closure

// closure_exact_subject_binary_test.go provides the
// TestClosureExactSubjectBinaryAuthority umbrella required
// by ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01-CORRECTION02-B1-R3.
//
// The umbrella exercises the production
// BuildExactSubjectBinary authority against the REAL Leamas
// repository and a real `go build` + `leamas version --json`
// subprocess invocation. The required matrix covers the
// happy path plus input-validation rejections and
// output-in-confined-root rejections. Failure-matrix rows
// live in closure_exact_subject_binary_failures_test.go so
// this file stays under the LLM-friendly 400-line threshold.

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// sha256Regex matches the canonical SHA-256 hex digest
// produced by the production hash authority. The umbrella
// test uses it to prove the binary SHA field is
// syntactically valid.
var sha256Regex = regexp.MustCompile("^[0-9a-f]{64}$")

// exactBinaryOutputRoot returns a per-test temp directory
// that is guaranteed to be outside every worktree path.
func exactBinaryOutputRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs output root: %v", err)
	}
	return abs
}

// exactBinaryLeamasSubject resolves the REAL Leamas
// repository and returns RepositoryRoot, SubjectCommit and
// SubjectTree observed from the working tree. The umbrella
// test requires the caller source commit to be committed
// (not dirty); a dirty working tree is itself a B1 stop
// condition.
func exactBinaryLeamasSubject(t *testing.T) (repoRoot, subject, subjectTree string) {
	t.Helper()
	// RepositoryRoot = `git rev-parse --show-toplevel`.
	repoRoot = mustRunGit(t, ".", "rev-parse", "--show-toplevel")
	// S = `git rev-parse HEAD`.
	subject = mustRunGit(t, repoRoot, "rev-parse", "HEAD")
	// S_TREE = `git rev-parse HEAD^{tree}`.
	subjectTree = mustRunGit(t, repoRoot, "rev-parse", "HEAD^{tree}")
	// Caller source commit MUST be committed (not dirty).
	// Untracked files do NOT count as "source not committed"
	// because the source commit is independent of the
	// working tree's untracked files. We therefore pass
	// --untracked-files=no so the pre-condition only fires
	// on staged / unstaged modifications.
	porcelain := mustRunGit(t, repoRoot, "status", "--porcelain=v2", "--untracked-files=no")
	if porcelain != "" {
		t.Fatalf("real Leamas umbrella requires a committed caller source; dirty status:\n%s", porcelain)
	}
	return
}

// TestClosureExactSubjectBinaryAuthority is the umbrella
// test for the production BuildExactSubjectBinary authority.
//
// Happy row: build from exact-S against the REAL Leamas
// repository, observe the binary's own `version --json`
// output, verify the captured worktree is gone after the
// call returns. Independently re-hashes the binary to prove
// the reported SHA matches the on-disk content.
func TestClosureExactSubjectBinaryAuthority(t *testing.T) {
	dir, subject, subjectTree := exactBinaryLeamasSubject(t)
	outputRoot := exactBinaryOutputRoot(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	res, err := BuildExactSubjectBinary(ctx, ExactSubjectBinaryRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    subjectTree,
		OutputRoot:     outputRoot,
		OutputName:     "leamas-subject",
	})
	if err != nil {
		t.Fatalf("BuildExactSubjectBinary failed: %v", err)
	}

	// Source authority.
	if res.SourceCommit != subject {
		t.Fatalf("SourceCommit %s != subject %s", res.SourceCommit, subject)
	}
	if res.SourceTree != subjectTree {
		t.Fatalf("SourceTree %s != subject tree %s", res.SourceTree, subjectTree)
	}
	if !res.SourceClean {
		t.Fatal("SourceClean is false")
	}
	if !res.SourceDetached {
		t.Fatal("SourceDetached is false")
	}

	// Binary identity — canonical authority.
	if res.BinaryCommit != subject {
		t.Fatalf("BinaryCommit %s != subject %s", res.BinaryCommit, subject)
	}
	if res.BinaryModified {
		t.Fatal("BinaryModified is true; source must be clean")
	}
	if !res.Executable {
		t.Fatal("Executable is false")
	}
	if !res.OutputOutsideAllWorktrees {
		t.Fatal("OutputOutsideAllWorktrees is false")
	}
	if !sha256Regex.MatchString(res.BinarySHA256) {
		t.Fatalf("BinarySHA256 %q is not a 64-character hex digest", res.BinarySHA256)
	}
	if !filepath.IsAbs(res.BinaryPath) {
		t.Fatalf("BinaryPath %q is not absolute", res.BinaryPath)
	}

	// Bounded-result proof.
	if !res.BuildBounded {
		t.Fatalf("BuildBounded is false (code=%s)", res.BuildErrorCode)
	}
	if !res.IdentityBounded {
		t.Fatalf("IdentityBounded is false (code=%s)", res.IdentityErrorCode)
	}

	// Cleanup proof.
	if !res.CleanupAttempted {
		t.Fatal("CleanupAttempted is false")
	}
	if !res.CleanupSucceeded {
		t.Fatalf("CleanupSucceeded is false (err=%s)", res.CleanupError)
	}
	if res.CleanupAttempts != 1 {
		t.Fatalf("CleanupAttempts %d != 1", res.CleanupAttempts)
	}
	if !res.CleanupContextFresh {
		t.Fatal("CleanupContextFresh is false")
	}
	if !res.PostCleanupInventoryClosed {
		t.Fatalf("PostCleanupInventoryClosed is false (err=%s)", res.PostCleanupInventoryError)
	}
	if res.BuildWorktreeLeak {
		t.Fatalf("BuildWorktreeLeak is true (paths=%v)", res.PostCleanupInventoryLeakPaths)
	}

	// Independently re-hash the binary. This proves the
	// reported SHA matches the on-disk content and that
	// the umbrella did NOT trust the runner's word alone.
	sum, err := hashBinaryAtRest(res.BinaryPath)
	if err != nil {
		t.Fatalf("post-build hash: %v", err)
	}
	if sum != res.BinarySHA256 {
		t.Fatalf("binary SHA mismatch: reported %s, recomputed %s", res.BinarySHA256, sum)
	}

	// Auxiliary native buildinfo diagnostics. Absence is
	// acceptable but, when present, MUST match the
	// canonical identity.
	if res.NativeVCSRevisionPresent && res.NativeVCSRevision != subject {
		t.Fatalf("NativeVCSRevision %s != subject %s (canonical mismatch)",
			res.NativeVCSRevision, subject)
	}
	if res.NativeVCSModifiedPresent && res.NativeVCSModified {
		t.Fatalf("NativeVCSModified true while BinaryModified false")
	}
}

// TestClosureExactSubjectBinary_RejectsOutputInsideCallerRepo
// proves the output-in-caller-repo negative row fails
// closed. The test does NOT rely on a real `go build`; the
// caller-repo containment check must reject before any
// subprocess is invoked.
func TestClosureExactSubjectBinary_RejectsOutputInsideCallerRepo(t *testing.T) {
	dir, subject, subjectTree := exactBinaryLeamasSubject(t)
	badOutputRoot := filepath.Join(dir, "build")
	_, err := BuildExactSubjectBinary(context.Background(), ExactSubjectBinaryRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    subjectTree,
		OutputRoot:     badOutputRoot,
	})
	if err == nil {
		t.Fatal("BuildExactSubjectBinary must reject output inside caller repo")
	}
	if !strings.Contains(err.Error(), "inside caller repo") {
		t.Fatalf("error must mention caller repo containment, got: %v", err)
	}
}

// TestClosureExactSubjectBinary_RejectsOutputInsideLinkedWorktree
// proves the output-inside-linked-worktree negative row
// fails closed.
func TestClosureExactSubjectBinary_RejectsOutputInsideLinkedWorktree(t *testing.T) {
	dir, subject, subjectTree := exactBinaryLeamasSubject(t)
	subjectWorktree := filepath.Join(t.TempDir(), "subject-wt")
	mustRunGit(t, dir, "worktree", "add", "--detach", subjectWorktree, subject)
	defer mustRunGit(t, dir, "worktree", "remove", "--force", subjectWorktree)
	badOutputRoot := filepath.Join(subjectWorktree, "build")
	_, err := BuildExactSubjectBinary(context.Background(), ExactSubjectBinaryRequest{
		RepositoryRoot: dir,
		SubjectCommit:  subject,
		SubjectTree:    subjectTree,
		OutputRoot:     badOutputRoot,
	})
	if err == nil {
		t.Fatal("BuildExactSubjectBinary must reject output inside linked worktree")
	}
	if !strings.Contains(err.Error(), "linked worktree") {
		t.Fatalf("error must mention linked worktree, got: %v", err)
	}
}

// TestClosureExactSubjectBinary_RejectsInputValidation
// proves the authority rejects empty required inputs before
// any side-effect.
func TestClosureExactSubjectBinary_RejectsInputValidation(t *testing.T) {
	dir, subject, subjectTree := exactBinaryLeamasSubject(t)
	cases := []struct {
		name string
		req  ExactSubjectBinaryRequest
		want string
	}{
		{
			name: "empty repository root",
			req: ExactSubjectBinaryRequest{
				SubjectCommit: subject,
				SubjectTree:   subjectTree,
				OutputRoot:    t.TempDir(),
			},
			want: "repository root is empty",
		},
		{
			name: "empty subject commit",
			req: ExactSubjectBinaryRequest{
				RepositoryRoot: dir,
				SubjectTree:    subjectTree,
				OutputRoot:     t.TempDir(),
			},
			want: "subject commit is empty",
		},
		{
			name: "empty subject tree",
			req: ExactSubjectBinaryRequest{
				RepositoryRoot: dir,
				SubjectCommit:  subject,
				OutputRoot:     t.TempDir(),
			},
			want: "subject tree is empty",
		},
		{
			name: "empty output root",
			req: ExactSubjectBinaryRequest{
				RepositoryRoot: dir,
				SubjectCommit:  subject,
				SubjectTree:    subjectTree,
			},
			want: "output root is empty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildExactSubjectBinary(context.Background(), tc.req)
			if err == nil {
				t.Fatalf("must reject %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error must mention %q, got: %v", tc.want, err)
			}
		})
	}
}

// ensure execution and os imports are referenced even when
// the test infrastructure changes. The execution import
// keeps the package surface alive; the os import is needed
// by helper tests below.
var (
	_ = execution.NewExecutor
	_ = os.WriteFile
)

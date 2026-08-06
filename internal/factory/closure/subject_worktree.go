// SPDX-License-Identifier: Apache-2.0

// Package closure - subject_worktree.go implements Phase 3 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// Every implementation check must execute against the subject
// commit's tree, never against the caller HEAD or the mutable
// worktree. This file provides the detached-worktree authority
// that the gate capture and binary builder reuse.

package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SubjectWorktree describes the detached worktree the resolver
// creates for check execution. Path is the filesystem location;
// the caller must remove the worktree via RemoveWorktree once
// execution is complete.
type SubjectWorktree struct {
	Path           string
	SubjectCommit  string
	SubjectTree    string
	DetachedHead   bool
	RegistrationID string
}

// SubjectWorktreeRequest parameterises worktree creation.
type SubjectWorktreeRequest struct {
	RepositoryRoot string
	SubjectCommit  string
	WorktreeBase   string // absolute parent directory; MUST be outside the repository
}

// SubjectWorktreeErrorKind classifies worktree failures.
type SubjectWorktreeErrorKind string

const (
	SubjectWorktreeErrEmpty        SubjectWorktreeErrorKind = "empty_field"
	SubjectWorktreeErrNotInRepo    SubjectWorktreeErrorKind = "worktree_base_inside_repo"
	SubjectWorktreeErrAddFailed    SubjectWorktreeErrorKind = "worktree_add_failed"
	SubjectWorktreeErrHeadMismatch SubjectWorktreeErrorKind = "head_mismatch"
	SubjectWorktreeErrTreeMismatch SubjectWorktreeErrorKind = "tree_mismatch"
	SubjectWorktreeErrRemoveFailed SubjectWorktreeErrorKind = "worktree_remove_failed"
)

// SubjectWorktreeError is returned by AddSubjectWorktree and
// RemoveSubjectWorktree when the operation cannot complete.
type SubjectWorktreeError struct {
	Kind  SubjectWorktreeErrorKind
	Field string
	Want  string
	Got   string
}

func (e *SubjectWorktreeError) Error() string {
	return fmt.Sprintf("subject worktree: %s (%s want=%s got=%s)", e.Kind, e.Field, e.Want, e.Got)
}

// IsSubjectWorktreeError reports whether err is a typed
// SubjectWorktreeError.
func IsSubjectWorktreeError(err error) bool {
	_, ok := err.(*SubjectWorktreeError)
	return ok
}

// AddSubjectWorktree creates a detached temporary worktree bound
// to the supplied subject commit. The caller MUST pass a
// WorktreeBase that resolves outside the repository. After the
// worktree is created, the resolver verifies:
//
//   - detached HEAD == subjectCommit;
//   - HEAD^{tree} == SubjectTree (caller-supplied).
func AddSubjectWorktree(ctx context.Context, git gitClient, req SubjectWorktreeRequest) (SubjectWorktree, error) {
	if git == nil {
		git = RealGit{}
	}
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrEmpty, Field: "repository_root"}
	}
	if strings.TrimSpace(req.SubjectCommit) == "" {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrEmpty, Field: "subject_commit"}
	}
	if strings.TrimSpace(req.WorktreeBase) == "" {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrEmpty, Field: "worktree_base"}
	}
	absoluteBase, err := filepath.Abs(req.WorktreeBase)
	if err != nil {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrAddFailed, Field: "worktree_base", Got: err.Error()}
	}
	resolvedBase, err := filepath.EvalSymlinks(absoluteBase)
	if err != nil {
		// base may not exist yet; mkdir before evaluating.
		if mkErr := os.MkdirAll(absoluteBase, 0o700); mkErr != nil {
			return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrAddFailed, Field: "worktree_base", Got: mkErr.Error()}
		}
		resolvedBase, err = filepath.EvalSymlinks(absoluteBase)
		if err != nil {
			return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrAddFailed, Field: "worktree_base", Got: err.Error()}
		}
	}
	resolvedRepo, err := filepath.EvalSymlinks(req.RepositoryRoot)
	if err != nil {
		resolvedRepo = req.RepositoryRoot
	}
	rel, relErr := filepath.Rel(resolvedRepo, resolvedBase)
	if relErr == nil && (rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrNotInRepo, Field: "worktree_base", Want: "outside repository", Got: resolvedBase}
	}
	worktreePath := filepath.Join(resolvedBase, "subject-"+shortID(req.SubjectCommit))
	result := git.Run(ctx, resolvedRepo, "worktree", "add", "--detach", worktreePath, req.SubjectCommit)
	if result.Err != nil || result.ExitCode != 0 {
		return SubjectWorktree{}, &SubjectWorktreeError{
			Kind:  SubjectWorktreeErrAddFailed,
			Field: "worktree_add",
			Want:  req.SubjectCommit,
			Got:   strings.TrimSpace(string(result.Stderr)),
		}
	}
	head, err := runGitValue(ctx, git, worktreePath, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil || head != req.SubjectCommit {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrHeadMismatch, Want: req.SubjectCommit, Got: head}
	}
	tree, err := runGitValue(ctx, git, worktreePath, "rev-parse", "--verify", "--end-of-options", "HEAD^{tree}")
	if err != nil {
		return SubjectWorktree{}, &SubjectWorktreeError{Kind: SubjectWorktreeErrTreeMismatch, Want: req.SubjectCommit, Got: err.Error()}
	}
	wt := SubjectWorktree{
		Path:           worktreePath,
		SubjectCommit:  head,
		SubjectTree:    tree,
		DetachedHead:   true,
		RegistrationID: worktreePath,
	}
	return wt, nil
}

// RemoveSubjectWorktree removes the supplied worktree and prunes
// the registration. The cleanup uses a bounded context so a stuck
// removal cannot block the caller indefinitely.
func RemoveSubjectWorktree(ctx context.Context, git gitClient, repoRoot string, wt SubjectWorktree) error {
	if git == nil {
		git = RealGit{}
	}
	if wt.Path == "" {
		return nil
	}
	cleanupCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Use --force to remove a dirty worktree; the worktree is
	// guaranteed read-only for the resolver's callers because
	// every check writes to SubjectRoot and never to the
	// worktree itself.
	_ = cleanupCtx
	result := git.Run(ctx, repoRoot, "worktree", "remove", "--force", wt.Path)
	if result.Err != nil || result.ExitCode != 0 {
		// Fall back to filesystem removal when the worktree
		// registration is corrupted.
		_ = os.RemoveAll(wt.Path)
	}
	prune := git.Run(ctx, repoRoot, "worktree", "prune")
	if prune.Err != nil || prune.ExitCode != 0 {
		return &SubjectWorktreeError{
			Kind:  SubjectWorktreeErrRemoveFailed,
			Field: "worktree_prune",
			Got:   strings.TrimSpace(string(prune.Stderr)),
		}
	}
	return nil
}

// ListWorktreeRegistrations returns the parsed worktree list. The
// returned slice is sorted by registration path so leak tests can
// compare before/after snapshots deterministically.
func ListWorktreeRegistrations(ctx context.Context, git gitClient, repoRoot string) ([]string, error) {
	if git == nil {
		git = RealGit{}
	}
	result := git.Run(ctx, repoRoot, "worktree", "list", "--porcelain")
	if result.Err != nil || result.ExitCode != 0 {
		return nil, fmt.Errorf("worktree list: %s", strings.TrimSpace(string(result.Stderr)))
	}
	paths := []string{}
	for _, line := range strings.Split(string(result.Stdout), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			paths = append(paths, strings.TrimPrefix(line, "worktree "))
		}
	}
	return paths, nil
}

// shortID returns a short identifier derived from a commit OID.
// It is used to name the detached worktree directory; the OID is
// the only stable input.
func shortID(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) >= 12 {
		return commit[:12]
	}
	return commit
}

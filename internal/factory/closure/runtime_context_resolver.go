// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_context_resolver.go implements the
// production RuntimeContextResolver required by Phase 1 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// The resolver is the single authority for resolving the freeze
// commit, subject commit, plan blob, and evidence directory.
// Every field is fetched via the supplied gitClient so the
// resolver is hermetically testable. The resolver NEVER reads
// ambient shell variables.

package closure

import (
	"context"
	"path/filepath"
	"strings"
	"time"
)

// runtimeResolver is the production RuntimeContextResolver.
type runtimeResolver struct {
	git  gitClient
	now  func() time.Time
	rand func() string
}

// NewRuntimeContextResolver constructs the production resolver.
func NewRuntimeContextResolver() RuntimeContextResolver {
	return &runtimeResolver{git: RealGit{}, now: time.Now}
}

// WithGitClient is a test seam.
func WithGitClient(r RuntimeContextResolver, git gitClient) RuntimeContextResolver {
	if concrete, ok := r.(*runtimeResolver); ok {
		concrete.git = git
		return r
	}
	return r
}

// Resolve satisfies RuntimeContextResolver.
func (r *runtimeResolver) Resolve(
	ctx context.Context,
	repositoryRoot string,
	actID string,
	freezeRevision string,
	subjectRevision string,
	planPath string,
	evidenceDirectory string,
) (RuntimeContext, error) {
	if r == nil || r.git == nil {
		r.git = RealGit{}
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.rand == nil {
		r.rand = randomHexRunID
	}

	if strings.TrimSpace(repositoryRoot) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "repository_root", Kind: "empty_field"}
	}
	if strings.TrimSpace(actID) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "act_id", Kind: "empty_field"}
	}
	if strings.TrimSpace(freezeRevision) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "freeze", Kind: "empty_field"}
	}
	if strings.TrimSpace(subjectRevision) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "subject", Kind: "empty_field"}
	}
	if strings.TrimSpace(planPath) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "plan_path", Kind: "empty_field"}
	}
	if strings.TrimSpace(evidenceDirectory) == "" {
		return RuntimeContext{}, &RuntimeContextError{Field: "evidence_directory", Kind: "empty_field"}
	}
	if frozenPlanPathRejected(planPath) {
		return RuntimeContext{}, &RuntimeContextError{Field: "plan_path", Kind: "plan_path_invalid", Want: "strictly confined repository-relative path", Got: planPath}
	}

	absoluteRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return RuntimeContext{}, &RuntimeContextError{Field: "repository_root", Kind: "oid_mismatch", Want: "absolute path", Got: err.Error()}
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		resolvedRoot = absoluteRoot
	}

	cleanEvidence, err := filepath.Abs(evidenceDirectory)
	if err != nil {
		return RuntimeContext{}, &RuntimeContextError{Field: "evidence_directory", Kind: "plan_path_invalid", Want: "absolute path", Got: err.Error()}
	}

	if err := checkCleanWorktree(ctx, r.git, resolvedRoot); err != nil {
		return RuntimeContext{}, err
	}

	format, err := readObjectFormat(ctx, r.git, resolvedRoot)
	if err != nil {
		return RuntimeContext{}, err
	}
	if format != "sha1" {
		return RuntimeContext{}, &RuntimeContextError{
			Field: "object_format",
			Kind:  "unsupported_object_format",
			Got:   format,
		}
	}

	freezeCommit, err := resolveOID(ctx, r.git, resolvedRoot, freezeRevision+"^{commit}", "freeze_commit")
	if err != nil {
		return RuntimeContext{}, err
	}
	subjectCommit, err := resolveOID(ctx, r.git, resolvedRoot, subjectRevision+"^{commit}", "subject_commit")
	if err != nil {
		return RuntimeContext{}, err
	}
	if freezeCommit == subjectCommit {
		return RuntimeContext{}, &RuntimeContextError{
			Field: "freeze",
			Kind:  "freeze_equals_subject",
			Got:   freezeCommit,
		}
	}
	if !runtimeIsAncestor(ctx, r.git, resolvedRoot, freezeCommit, subjectCommit) {
		return RuntimeContext{}, &RuntimeContextError{
			Field: "freeze",
			Kind:  "freeze_not_ancestor",
			Want:  freezeCommit,
			Got:   subjectCommit,
		}
	}
	freezeTree, err := resolveOID(ctx, r.git, resolvedRoot, freezeCommit+"^{tree}", "freeze_tree")
	if err != nil {
		return RuntimeContext{}, err
	}
	subjectTree, err := resolveOID(ctx, r.git, resolvedRoot, subjectCommit+"^{tree}", "subject_tree")
	if err != nil {
		return RuntimeContext{}, err
	}

	planBlob, planBytes, err := resolveFrozenPlanBlob(ctx, r.git, resolvedRoot, freezeCommit, planPath)
	if err != nil {
		return RuntimeContext{}, err
	}
	planSHA := frozenPlanSHA256(planBytes)

	started := r.now().UTC().Format(time.RFC3339Nano)
	rc := RuntimeContext{
		ACTID:             actID,
		RepositoryRoot:    resolvedRoot,
		RunID:             r.rand(),
		FreezeCommit:      freezeCommit,
		FreezeTree:        freezeTree,
		SubjectCommit:     subjectCommit,
		SubjectTree:       subjectTree,
		PlanPath:          planPath,
		PlanBlob:          planBlob,
		PlanSHA256:        planSHA,
		EvidenceDirectory: cleanEvidence,
		StartedAt:         started,
	}
	return rc, nil
}

// checkCleanWorktree refuses to produce a RuntimeContext when
// the caller worktree is dirty.
func checkCleanWorktree(ctx context.Context, git gitClient, root string) error {
	if git == nil {
		git = RealGit{}
	}
	result := git.Run(ctx, root, "status", "--porcelain=v1", "--untracked-files=normal")
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return &RuntimeContextError{Field: "worktree_status", Kind: "dirty_worktree", Got: detail}
	}
	if strings.TrimSpace(string(result.Stdout)) != "" {
		return &RuntimeContextError{Field: "worktree_status", Kind: "dirty_worktree"}
	}
	return nil
}

// readObjectFormat invokes `git rev-parse --show-object-format`.
func readObjectFormat(ctx context.Context, git gitClient, root string) (string, error) {
	result := git.Run(ctx, root, "rev-parse", "--show-object-format")
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return "", &RuntimeContextError{Field: "object_format", Kind: "unsupported_object_format", Got: detail}
	}
	format := strings.TrimSpace(string(result.Stdout))
	if format == "" {
		return "", &RuntimeContextError{Field: "object_format", Kind: "unsupported_object_format", Got: "<empty>"}
	}
	return format, nil
}

// resolveOID runs `git rev-parse --verify --end-of-options <expr>`.
func resolveOID(ctx context.Context, git gitClient, root, expression, field string) (string, error) {
	result := git.Run(ctx, root, "rev-parse", "--verify", "--end-of-options", expression)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return "", &RuntimeContextError{Field: field, Kind: "oid_mismatch", Want: expression, Got: detail}
	}
	value := strings.TrimSpace(string(result.Stdout))
	if err := ValidateOID(field, value); err != nil {
		return "", &RuntimeContextError{Field: field, Kind: "oid_mismatch", Want: "40 hex chars", Got: value}
	}
	return value, nil
}

// runtimeIsAncestor is the boolean-only variant used by the
// resolver.
func runtimeIsAncestor(ctx context.Context, git gitClient, root, ancestor, descendant string) bool {
	result := git.Run(ctx, root, "merge-base", "--is-ancestor", ancestor, descendant)
	return result.Err == nil && result.ExitCode == 0
}

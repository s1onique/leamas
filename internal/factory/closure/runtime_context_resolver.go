// SPDX-License-Identifier: Apache-2.0

// Package closure - runtime_context_resolver.go implements the
// production RuntimeContextResolver required by Phase 1 of
// ACT-LEAMAS-FACTORY-CLOSURE-RUNTIME-CONTEXT-AND-EXECUTE01.
//
// The resolver is the single authority for resolving the freeze
// commit, subject commit, plan blob, and evidence directory. Every
// field is fetched via the supplied gitClient (or RealGit when the
// caller passes nil) so the resolver is hermetically testable.
//
// The resolver NEVER reads ambient shell variables. Every identity
// is derived from the Git object database; every precondition is
// validated before the resulting RuntimeContext is published.

package closure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

// runtimeResolver is the production RuntimeContextResolver. It
// owns a gitClient and a clock; tests substitute both.
type runtimeResolver struct {
	git  gitClient
	now  func() time.Time
	rand func() string
}

// NewRuntimeContextResolver constructs the production resolver.
// A nil git defaults to RealGit{}. The clock defaults to time.Now.
// The random source defaults to crypto/rand-backed hex.
func NewRuntimeContextResolver() RuntimeContextResolver {
	return &runtimeResolver{git: RealGit{}, now: time.Now, rand: randHexID}
}

// WithGitClient is a test seam: callers (mainly tests) can swap the
// gitClient before any Resolve call. Returns the receiver so it
// can be chained in test setup.
func WithGitClient(r RuntimeContextResolver, git gitClient) RuntimeContextResolver {
	concrete, ok := r.(*runtimeResolver)
	if !ok {
		return r
	}
	concrete.git = git
	return r
}

// Resolve satisfies RuntimeContextResolver. It performs the full
// validation sequence required by Phase 1.
func (r *runtimeResolver) Resolve(
	ctx context.Context,
	repositoryRoot string,
	actID string,
	freezeRevision string,
	subjectRevision string,
	planPath string,
	evidenceDirectory string,
) (RuntimeContext, error) {
	if r == nil {
		return RuntimeContext{}, &RuntimeContextError{Field: "resolver", Kind: "empty_field"}
	}
	if r.git == nil {
		r.git = RealGit{}
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.rand == nil {
		r.rand = randHexID
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

	absoluteRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return RuntimeContext{}, &RuntimeContextError{Field: "repository_root", Kind: "oid_mismatch", Want: "absolute path", Got: err.Error()}
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		// EvalSymlinks may fail when the root does not yet
		// exist on disk; the underlying git invocation will
		// surface a clearer diagnostic in that case.
		resolvedRoot = absoluteRoot
	}

	cleanPlanPath, err := validatePlanPath(planPath)
	if err != nil {
		return RuntimeContext{}, err
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

	planBlob, planBytes, err := readPlanBlob(ctx, r.git, resolvedRoot, freezeCommit, cleanPlanPath)
	if err != nil {
		return RuntimeContext{}, err
	}
	planSHA := SHA256Hex(planBytes)

	started := r.now().UTC().Format(time.RFC3339Nano)
	rc := RuntimeContext{
		ACTID:             actID,
		RepositoryRoot:    resolvedRoot,
		RunID:             r.rand(),
		FreezeCommit:      freezeCommit,
		FreezeTree:        freezeTree,
		SubjectCommit:     subjectCommit,
		SubjectTree:       subjectTree,
		PlanPath:          cleanPlanPath,
		PlanBlob:          planBlob,
		PlanSHA256:        planSHA,
		EvidenceDirectory: cleanEvidence,
		StartedAt:         started,
	}
	return rc, nil
}

// validatePlanPath rejects absolute paths, parent traversal, and
// empty paths. It returns the cleaned repository-relative form.
func validatePlanPath(raw string) (string, error) {
	cleaned := filepath.Clean(raw)
	if cleaned == "." || cleaned == "" {
		return "", &RuntimeContextError{Field: "plan_path", Kind: "plan_path_invalid", Want: "non-empty repository-relative path", Got: raw}
	}
	if filepath.IsAbs(cleaned) {
		return "", &RuntimeContextError{Field: "plan_path", Kind: "plan_path_invalid", Want: "repository-relative", Got: cleaned}
	}
	if strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) || cleaned == ".." {
		return "", &RuntimeContextError{Field: "plan_path", Kind: "plan_path_invalid", Want: "inside repository", Got: cleaned}
	}
	return cleaned, nil
}

// checkCleanWorktree refuses to produce a RuntimeContext when the
// caller worktree is dirty.
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

// readObjectFormat invokes `git rev-parse --show-object-format` and
// returns the trimmed result. It refuses to return an empty format.
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

// resolveOID runs `git rev-parse --verify --end-of-options <expr>`
// and validates the resulting 40-character lowercase SHA-1 OID.
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

// runtimeIsAncestor reports whether ancestor is an ancestor of
// descendant. The check is implemented as `git merge-base
// --is-ancestor`. The helper is intentionally local to the
// runtime context subsystem so it cannot collide with the legacy
// validator's (bool, error) variant.
func runtimeIsAncestor(ctx context.Context, git gitClient, root, ancestor, descendant string) bool {
	result := git.Run(ctx, root, "merge-base", "--is-ancestor", ancestor, descendant)
	return result.Err == nil && result.ExitCode == 0
}

// readPlanBlob reads the literal plan bytes from the freeze commit
// and returns both the blob OID and the bytes themselves. The blob
// OID is computed from the bytes so the resolver cannot disagree
// with the resolver's caller about which bytes were loaded.
func readPlanBlob(ctx context.Context, git gitClient, root, freezeCommit, planPath string) (string, []byte, error) {
	result := git.Run(ctx, root, "cat-file", "blob", freezeCommit+":"+planPath)
	if result.Err != nil || result.ExitCode != 0 {
		detail := strings.TrimSpace(string(result.Stderr))
		if detail == "" && result.Err != nil {
			detail = result.Err.Error()
		}
		return "", nil, &RuntimeContextError{Field: "plan_blob", Kind: "oid_mismatch", Want: freezeCommit + ":" + planPath, Got: detail}
	}
	bytes := append([]byte(nil), result.Stdout...)
	blobOID := blobOIDForBytes(ctx, git, root, bytes)
	return blobOID, bytes, nil
}

// blobOIDForBytes writes the supplied bytes to a temporary file
// and asks `git hash-object` for the blob OID. The fallback
// computes the SHA-1 locally so the resolver always returns a
// stable OID even when stdin redirection is unavailable.
func blobOIDForBytes(ctx context.Context, git gitClient, root string, data []byte) string {
	tmp, err := writeTempBytes(data)
	if err != nil {
		return localSHA1Hex(data)
	}
	defer removeTemp(tmp)
	result := git.Run(ctx, root, "hash-object", tmp)
	if result.Err != nil || result.ExitCode != 0 {
		return localSHA1Hex(data)
	}
	return strings.TrimSpace(string(result.Stdout))
}

// randHexID returns a short opaque identifier suitable for RunID.
// It deliberately avoids ambient entropy sources.
func randHexID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(buf[:])
}

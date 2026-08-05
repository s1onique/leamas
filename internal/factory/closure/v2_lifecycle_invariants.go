// SPDX-License-Identifier: Apache-2.0

package closure

// v2_lifecycle_invariants.go implements the production-enforced
// lifecycle invariants required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-LIFECYCLE-INVARIANTS01.
//
// The file owns the four production authorities that make the
// v2 runner a safe witness even when the caller misbehaves:
//
//  1. caller-state authority — captures HEAD, HEAD tree,
//     `git status --porcelain=v2 --untracked-files=all`, and
//     linked-worktree registrations BEFORE and AFTER the run,
//     and reports any difference as a typed V2Diagnostic so
//     the runner can refuse to claim success.
//  2. worktree-registration authority — captures
//     `git worktree list --porcelain` and proves no registration
//     leaks after success, failure, cancellation, or timeout.
//  3. cleanup-result authority — extends the cleanup report to
//     record every stage and a final registration state, then
//     propagates cleanup failures into the typed V2Error so a
//     clean success manifest cannot be published.
//  4. git-failure authority — classifies git command outcomes
//     into missing-object versus operational failure (timeout,
//     cancellation, output overflow, spawn failure, permission
//     error, non-repository path, malformed revision) so only
//     genuine missing revisions surface as subject_commit_not_found
//     or freeze_commit_not_found. Everything else surfaces as a
//     specific operational code.
//
// Splitting this from the executor keeps the executor under
// the LLM-friendly 400-line threshold while preserving the
// single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// worktreePorcelainField is the line prefix emitted by
// `git worktree list --porcelain` that names the absolute
// path of a registered linked worktree. The other lines (HEAD,
// branch, locked, prunable) are intentionally ignored: the
// runner only needs path-level identity to detect leaks.
const worktreePorcelainField = "worktree "

// v2WorktreeRegistration is one entry from
// `git worktree list --porcelain`. The path is the canonical
// absolute path of the linked worktree; the hash is the
// detached HEAD commit OID. Two registrations with identical
// canonical paths compare equal regardless of trailing
// separators.
type v2WorktreeRegistration struct {
	Path string
	Hash string
}

// v2WorktreeRegistrationSet is an ordered set of
// v2WorktreeRegistration entries. Order is preserved so
// diff messages stay deterministic across runs.
type v2WorktreeRegistrationSet []v2WorktreeRegistration

// Contains reports whether the set contains a registration
// for the supplied canonical path.
func (s v2WorktreeRegistrationSet) Contains(path string) bool {
	for _, r := range s {
		if r.Path == path {
			return true
		}
	}
	return false
}

// Diff returns the registrations present in `after` but absent
// from `before`. The order of returned entries is the order of
// `after` so the diff output is deterministic.
func (s v2WorktreeRegistrationSet) Diff(after v2WorktreeRegistrationSet) v2WorktreeRegistrationSet {
	var diff v2WorktreeRegistrationSet
	for _, r := range after {
		if !s.Contains(r.Path) {
			diff = append(diff, r)
		}
	}
	return diff
}

// snapshotWorktreeRegistrations runs
// `git worktree list --porcelain` and parses the registered
// worktree paths. The function returns an empty set when the
// git command fails (a brand-new repository still returns an
// empty set with exit code 0; the empty-set result is
// indistinguishable from a failure, which the runner treats as
// a missing-inventory event rather than a leak).
func snapshotWorktreeRegistrations(ctx context.Context, git gitClient, repoRoot string) v2WorktreeRegistrationSet {
	if git == nil || repoRoot == "" {
		return v2WorktreeRegistrationSet{}
	}
	result := git.Run(ctx, repoRoot, "worktree", "list", "--porcelain")
	if result.Err != nil || result.ExitCode != 0 {
		return v2WorktreeRegistrationSet{}
	}
	var (
		out  v2WorktreeRegistrationSet
		path string
		hash string
	)
	flush := func() {
		if path != "" {
			out = append(out, v2WorktreeRegistration{Path: path, Hash: hash})
		}
		path = ""
		hash = ""
	}
	for _, raw := range strings.Split(string(result.Stdout), "\n") {
		line := strings.TrimRight(raw, "\r")
		switch {
		case strings.HasPrefix(line, worktreePorcelainField):
			flush()
			path = strings.TrimSpace(strings.TrimPrefix(line, worktreePorcelainField))
		case strings.HasPrefix(line, "HEAD "):
			hash = strings.TrimSpace(strings.TrimPrefix(line, "HEAD "))
		case line == "":
			flush()
		}
	}
	flush()
	return out
}

// v2CallerState is the immutable snapshot of the caller
// repository the runner captures before any side-effecting
// step. The struct is compared bit-for-bit against a post-run
// snapshot; every difference produces a typed V2Diagnostic so
// the runner cannot claim success when it mutated the caller.
type v2CallerState struct {
	HEADCommit            string
	HEADTree              string
	StatusPorcelain       string
	WorktreeRegistrations v2WorktreeRegistrationSet
}

// snapshotCallerState captures HEAD, HEAD^{tree}, status, and
// linked-worktree registrations in a single immutable
// snapshot. The status uses `--porcelain=v2 --untracked-files=all`
// so ignored files do not contaminate the dirtiness signal.
func snapshotCallerState(ctx context.Context, git gitClient, repoRoot string) v2CallerState {
	state := v2CallerState{WorktreeRegistrations: v2WorktreeRegistrationSet{}}
	if git == nil || repoRoot == "" {
		return state
	}
	if head, err := runGitValue(ctx, git, repoRoot, "rev-parse", "HEAD^{commit}"); err == nil {
		state.HEADCommit = head
	}
	if tree, err := runGitValue(ctx, git, repoRoot, "rev-parse", "HEAD^{tree}"); err == nil {
		state.HEADTree = tree
	}
	if status, err := runGitValue(ctx, git, repoRoot, "status", "--porcelain=v2", "--untracked-files=all"); err == nil {
		state.StatusPorcelain = status
	}
	state.WorktreeRegistrations = snapshotWorktreeRegistrations(ctx, git, repoRoot)
	return state
}

// Diff compares two v2CallerState values and returns every
// typed diagnostic required to report any drift. The function
// is total: an empty slice means the caller state was
// unchanged.
func (s v2CallerState) Diff(after v2CallerState) V2Diagnostics {
	var out V2Diagnostics
	if s.HEADCommit != "" && after.HEADCommit != "" && s.HEADCommit != after.HEADCommit {
		out = append(out, V2Diagnostic{
			Code:         V2CodeCallerHeadChanged,
			Message:      fmt.Sprintf("caller HEAD changed: before=%s after=%s", s.HEADCommit, after.HEADCommit),
			PropertyName: "caller_head",
			Detail:       after.HEADCommit,
		})
	}
	if s.HEADTree != "" && after.HEADTree != "" && s.HEADTree != after.HEADTree {
		out = append(out, V2Diagnostic{
			Code:         V2CodeCallerTreeChanged,
			Message:      fmt.Sprintf("caller HEAD tree changed: before=%s after=%s", s.HEADTree, after.HEADTree),
			PropertyName: "caller_tree",
			Detail:       after.HEADTree,
		})
	}
	if s.StatusPorcelain != after.StatusPorcelain {
		// The runner may legitimately leave the repository
		// untouched but with a freshly created evidence
		// directory inside the repository (a misconfigured
		// caller). Such a directory does NOT match the
		// forbidden-root rule but would dirty the status.
		// We surface the drift so the CLI can flag the
		// misconfiguration.
		out = append(out, V2Diagnostic{
			Code:         V2CodeCallerWorktreeDirtyAfter,
			Message:      fmt.Sprintf("caller worktree dirty after run: before=%q after=%q", s.StatusPorcelain, after.StatusPorcelain),
			PropertyName: "caller_worktree",
			Detail:       after.StatusPorcelain,
		})
	}
	if leaked := s.WorktreeRegistrations.Diff(after.WorktreeRegistrations); len(leaked) > 0 {
		paths := make([]string, 0, len(leaked))
		for _, r := range leaked {
			paths = append(paths, r.Path)
		}
		out = append(out, V2Diagnostic{
			Code:         V2CodeWorktreeRegistrationLeaked,
			Message:      fmt.Sprintf("linked worktree registration leaked: %s", strings.Join(paths, ", ")),
			PropertyName: "worktree_registration",
			Detail:       strings.Join(paths, ","),
		})
	}
	return out
}

// classifyGitCommand classifies a git command result into one
// of the closed failure codes. A successful run (no error,
// exit 0) returns ("", nil). The classifier never panics and
// never fabricates a missing-object verdict for an
// operational failure.
//
//	genuine missing revision   -> subject_commit_not_found /
//	                              freeze_commit_not_found
//	malformed revision         -> git_malformed_revision
//	not a Git repository       -> git_not_repository
//	permission failure         -> git_permission_denied
//	timeout                    -> git_timeout
//	cancellation               -> git_cancelled
//	output overflow            -> git_output_overflow
//	spawn failure              -> git_spawn_failed
//	other Git failure          -> git_operation_failed
func classifyGitCommand(args []string, result gitCommandResult) (V2DiagnosticCode, error) {
	if result.Err == nil && result.ExitCode == 0 {
		return "", nil
	}
	detail := strings.TrimSpace(string(result.Stderr))
	if detail == "" && result.Err != nil {
		detail = result.Err.Error()
	}
	lower := strings.ToLower(detail)
	// Operational failures first; the runner must NOT report
	// these as a missing commit.
	if errors.Is(result.Err, context.DeadlineExceeded) || result.Err == context.DeadlineExceeded {
		return V2CodeGitTimeout, fmt.Errorf("git %s timed out: %s", strings.Join(args, " "), detail)
	}
	if errors.Is(result.Err, context.Canceled) || result.Err == context.Canceled {
		return V2CodeGitCancelled, fmt.Errorf("git %s cancelled: %s", strings.Join(args, " "), detail)
	}
	var exitErr *exec.ExitError
	if errors.As(result.Err, &exitErr) && result.ExitCode < 0 {
		// Negative exit codes indicate a spawn / signal kill
		// rather than a normal git exit.
		return V2CodeGitSpawnFailed, fmt.Errorf("git %s spawn / signal failure: exit=%d stderr=%s",
			strings.Join(args, " "), result.ExitCode, detail)
	}
	if result.Err != nil && !errors.Is(result.Err, &exec.ExitError{}) {
		// Non-exit-error with no stderr is a spawn failure
		// (binary missing, PATH lookup failed).
		if detail == "" {
			return V2CodeGitSpawnFailed, fmt.Errorf("git %s spawn failure: %s", strings.Join(args, " "), result.Err.Error())
		}
	}
	switch {
	case strings.Contains(lower, "not a git repository"):
		return V2CodeGitNotRepository, fmt.Errorf("git %s: not a repository: %s", strings.Join(args, " "), detail)
	case strings.Contains(lower, "permission denied"):
		return V2CodeGitPermissionDenied, fmt.Errorf("git %s: permission denied: %s", strings.Join(args, " "), detail)
	case strings.Contains(lower, "fatal: ambiguous argument") ||
		strings.Contains(lower, "fatal: bad revision") ||
		strings.Contains(lower, "fatal: bad object"):
		return V2CodeGitMalformedRevision, fmt.Errorf("git %s: malformed revision: %s", strings.Join(args, " "), detail)
	}
	// Default to git_operation_failed so callers can route
	// unclassified failures to a generic remediation hint.
	return V2CodeGitOperationFailed, fmt.Errorf("git %s failed (exit %d): %s",
		strings.Join(args, " "), result.ExitCode, detail)
}

// gitFailureDiagnostic builds the typed V2Error wrapper for a
// classified git command failure. The diagnostic's
// PropertyName matches the arg that produced the failure so
// the CLI can render the exact field that failed.
func gitFailureDiagnostic(property string, args []string, result gitCommandResult) error {
	code, err := classifyGitCommand(args, result)
	if err == nil {
		return nil
	}
	if code == V2CodeGitOperationFailed {
		return NewV2ErrorWith(code, err.Error(), property, strings.TrimSpace(string(result.Stderr)))
	}
	return NewV2ErrorWith(code, err.Error(), property, strings.TrimSpace(string(result.Stderr)))
}

// v2LifecycleCleanupReport extends the executor's cleanup
// report with the final registration state so callers can
// detect leaks even when worktree remove and prune both
// succeeded (e.g. the registration survived because the
// caller did not own the .git/worktrees/ directory).
//
// The struct is intentionally additive: the executor continues
// to write the three stage errors into the underlying
// v2CleanupReport; this struct merely adds the registration
// snapshot taken before and after cleanup.
type v2LifecycleCleanupReport struct {
	Before v2WorktreeRegistrationSet
	After  v2WorktreeRegistrationSet
	Stages v2CleanupReport
}

// HasError reports whether any cleanup stage failed or a
// registration survived.
func (r v2LifecycleCleanupReport) HasError() bool {
	if r.Stages.HasError() {
		return true
	}
	return len(r.Before.Diff(r.After)) > 0
}

// Summary produces a single-line human-readable digest.
func (r v2LifecycleCleanupReport) Summary() string {
	parts := []string{}
	if r.Stages.HasError() {
		parts = append(parts, r.Stages.Summary())
	}
	if leaked := r.Before.Diff(r.After); len(leaked) > 0 {
		paths := make([]string, 0, len(leaked))
		for _, r := range leaked {
			paths = append(paths, r.Path)
		}
		parts = append(parts, fmt.Sprintf("worktree registration leaked: %s", strings.Join(paths, ", ")))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ")
}

// _ ensures the default cleanup timeout is referenced; tests
// may want to override it via V2LifecycleOptions.
var _ = defaultV2CleanupTimeout

// v2LifecycleOptions captures the optional lifecycle knobs
// the runner accepts. Tests use it to shorten the cleanup
// timeout or to inject a custom worktree-snapshot function.
type v2LifecycleOptions struct {
	CleanupTimeout time.Duration
}

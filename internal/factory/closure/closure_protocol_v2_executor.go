// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_executor.go provides the immutable
// subject-tree executor. The executor creates a detached
// temporary worktree at the subject commit S and verifies the
// observed tree OID equals S^{tree} before any checks run.
// The caller checkout and refs MUST remain unchanged; cleanup
// is bounded and best-effort with diagnostics on failure.
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-
// CORRECTION02 enforces:
//
//   - cleanup runs in a fresh bounded context (Phase 5)
//   - cleanup result reaches the caller and prevents clean
//     success on failure (Phase 4)
//   - linked worktree registration is removed via
//     `git worktree remove --force` + `git worktree prune`
//     before filesystem removal (Phase 6)

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

// defaultV2CleanupTimeout bounds the linked-worktree
// deregistration phase. The value is generous enough for any
// reasonable worktree but small enough to fail fast on a
// hung git.
const defaultV2CleanupTimeout = 30 * time.Second

// V2SubjectExecutor executes plan checks against exactly the
// tree of subject commit S. The worktree is created detached
// and removed before the function returns.
type V2SubjectExecutor interface {
	ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error)
}

// V2ExecuteRequest captures the inputs the executor needs to
// run checks against S^{tree}.
type V2ExecuteRequest struct {
	RepositoryRoot  string
	SubjectCommit   string
	SubjectTree     string
	EvidenceDir     string
	Checks          []PlanCheck
	CommandExecutor commandExecutor
	Now             func() time.Time
}

// V2ExecuteResult captures the deterministic outputs of the
// subject-tree execution. CheckResults mirrors the v1 schema
// so the manifest can reuse the existing parser.
type V2ExecuteResult struct {
	ObservedTree string
	CheckResults []CheckResult
	Evidence     []EvidenceRecord
	CleanupError string
}

// GitV2SubjectExecutor creates a detached worktree and runs
// checks via the supplied command executor. The executor is
// safe to call with a nil command executor (defaults to
// boundedCommandExecutor).
type GitV2SubjectExecutor struct {
	Git gitClient
}

// NewGitV2SubjectExecutor constructs an executor that defaults
// to RealGit when no client is supplied.
func NewGitV2SubjectExecutor(g gitClient) *GitV2SubjectExecutor {
	if g == nil {
		g = RealGit{}
	}
	return &GitV2SubjectExecutor{Git: g}
}

// v2CleanupReport records the three cleanup stages that
// Git's linked-worktree machinery requires. HasError reports
// whether any stage failed; Summary produces a single-line
// human-readable digest for V2ExecuteResult.CleanupError.
type v2CleanupReport struct {
	WorktreeRemoveError   error
	PruneError            error
	FilesystemRemoveError error
}

func (r v2CleanupReport) HasError() bool {
	return r.WorktreeRemoveError != nil || r.PruneError != nil || r.FilesystemRemoveError != nil
}

func (r v2CleanupReport) Summary() string {
	var parts []string
	if r.WorktreeRemoveError != nil {
		parts = append(parts, r.WorktreeRemoveError.Error())
	}
	if r.PruneError != nil {
		parts = append(parts, r.PruneError.Error())
	}
	if r.FilesystemRemoveError != nil {
		parts = append(parts, r.FilesystemRemoveError.Error())
	}
	return strings.Join(parts, "; ")
}

// ExecuteSubjectChecks creates a temporary detached worktree
// at the supplied subject commit, verifies the observed tree,
// runs the supplied checks, and cleans up. The function never
// touches the caller checkout.
//
// Failure modes propagate as V2Error:
//   - subject commit / tree mismatch
//   - worktree creation failure
//   - observed tree OID != subject tree OID
//   - check execution failure
//   - cleanup failure (Phase 4: prevents clean success)
func (e *GitV2SubjectExecutor) ExecuteSubjectChecks(ctx context.Context, req V2ExecuteRequest) (V2ExecuteResult, error) {
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"repository root is empty", "repository_root", "")
	}
	if strings.TrimSpace(req.SubjectCommit) == "" {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"subject commit is empty", "subject_commit", "")
	}
	if strings.TrimSpace(req.SubjectTree) == "" {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"subject tree is empty", "subject_tree", "")
	}
	if strings.TrimSpace(req.EvidenceDir) == "" {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeRequestIncomplete,
			"evidence directory is empty", "evidence_directory", "")
	}
	// Phase 5 (CORRECTION02): cleanup runs in a fresh bounded
	// context that is detached from the caller's cancellation.
	// Hitting Ctrl-C on the parent process must not strand a
	// worktree registration.
	cleanupContext, cancelCleanup := context.WithTimeout(
		context.Background(), defaultV2CleanupTimeout,
	)
	defer cancelCleanup()
	cleanupGit := func(args ...string) gitCommandResult {
		return e.Git.Run(cleanupContext, req.RepositoryRoot, args...)
	}
	worktreePath, err := os.MkdirTemp("", "leamas-v2-worktree-*")
	if err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create temp dir: %s", err.Error()),
			"execution_tree", err.Error())
	}
	// Phase 3 (LIFECYCLE-INVARIANTS01): capture the
	// worktree registrations BEFORE we add ours so the
	// lifecycle cleanup report can detect any leaked
	// registration after worktree remove + prune + os.RemoveAll.
	beforeReg := snapshotWorktreeRegistrations(cleanupContext, e.Git, req.RepositoryRoot)
	// Cleanup runs in-scope so the result captures its
	// outcome. The previous deferred-local-variable pattern
	// leaked the report; this model folds the report into the
	// final return BEFORE propagating.
	cleanup := func() v2LifecycleCleanupReport {
		report := v2LifecycleCleanupReport{Before: beforeReg}
		rmResult := cleanupGit("worktree", "remove", "--force", worktreePath)
		if rmResult.Err != nil || rmResult.ExitCode != 0 {
			report.Stages.WorktreeRemoveError = fmt.Errorf("git worktree remove --force %s: %s",
				worktreePath, strings.TrimSpace(string(rmResult.Stderr)))
		}
		pruneResult := cleanupGit("worktree", "prune")
		if pruneResult.Err != nil || pruneResult.ExitCode != 0 {
			report.Stages.PruneError = fmt.Errorf("git worktree prune: %s",
				strings.TrimSpace(string(pruneResult.Stderr)))
		}
		if err := os.RemoveAll(worktreePath); err != nil {
			report.Stages.FilesystemRemoveError = err
		}
		report.After = snapshotWorktreeRegistrations(cleanupContext, e.Git, req.RepositoryRoot)
		return report
	}

	addResult := e.Git.Run(ctx, req.RepositoryRoot, "worktree", "add", "--detach", worktreePath, req.SubjectCommit)
	if addResult.Err != nil || addResult.ExitCode != 0 {
		_ = cleanup()
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git worktree add --detach %s %s failed: %s",
				worktreePath, req.SubjectCommit,
				strings.TrimSpace(string(addResult.Stderr))),
			"execution_tree", "")
	}
	// Build the success path step by step. Each step can
	// short-circuit with an error, but cleanup ALWAYS runs
	// before the function returns.
	var (
		checks       []CheckResult
		evidence     []EvidenceRecord
		observedTree string
	)
	observedTree, err = runGitValue(ctx, e.Git, worktreePath, "rev-parse", "HEAD^{tree}")
	if err != nil {
		report := cleanup()
		result := V2ExecuteResult{CleanupError: report.Summary()}
		return result, wrapWithCleanup(err, report)
	}
	if observedTree != req.SubjectTree {
		err = NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("observed tree %s does not match subject tree %s",
				observedTree, req.SubjectTree),
			"execution_tree", observedTree)
		report := cleanup()
		result := V2ExecuteResult{
			ObservedTree: observedTree,
			CleanupError: report.Summary(),
		}
		return result, wrapWithCleanup(err, report)
	}
	if err = os.MkdirAll(req.EvidenceDir, 0o700); err != nil {
		err = NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create evidence dir: %s", err.Error()),
			"evidence_directory", err.Error())
		report := cleanup()
		result := V2ExecuteResult{
			ObservedTree: observedTree,
			CleanupError: report.Summary(),
		}
		return result, wrapWithCleanup(err, report)
	}
	executor := req.CommandExecutor
	if executor == nil {
		executor = boundedCommandExecutor{}
	}
	checks, evidence, err = executeChecks(ctx, checkExecutionRequest{
		RepositoryRoot:    worktreePath,
		EvidenceDirectory: req.EvidenceDir,
		SubjectTreeOID:    observedTree,
		Checks:            req.Checks,
		Now:               req.Now,
	}, executor)
	if err != nil {
		err = NewV2ErrorWith(V2CodeExecutionFailed,
			fmt.Sprintf("execute checks: %s", err.Error()),
			"checks", err.Error())
	}
	// Cleanup is the LAST step before the function returns,
	// and its report is folded into the result AND the
	// surfaced error.
	report := cleanup()
	result := V2ExecuteResult{
		ObservedTree: observedTree,
		CheckResults: checks,
		Evidence:     evidence,
		CleanupError: report.Summary(),
	}
	if report.HasError() && err == nil {
		err = NewV2ErrorWith(V2CodeCleanupFailed,
			fmt.Sprintf("cleanup failed: %s", report.Summary()),
			"cleanup", report.Summary())
	}
	if err != nil {
		return V2ExecuteResult{}, err
	}
	return result, nil
}

// wrapWithCleanup annotates an existing V2Error with the
// cleanup report so the CLI can render both the original
// failure cause and the cleanup outcome. When the supplied
// error is not already a V2Error, a new V2Error is built.
func wrapWithCleanup(original error, report v2LifecycleCleanupReport) error {
	if !report.HasError() {
		return original
	}
	if original == nil {
		return NewV2ErrorWith(V2CodeCleanupFailed,
			report.Summary(), "cleanup", report.Summary())
	}
	if v2err, ok := original.(*V2Error); ok {
		v2err.Diags = append(v2err.Diags, V2Diagnostic{
			Code:         V2CodeCleanupFailed,
			Message:      report.Summary(),
			PropertyName: "cleanup",
			Detail:       report.Summary(),
		})
		return v2err
	}
	return NewV2ErrorWith(V2CodeCleanupFailed,
		fmt.Sprintf("%s; cleanup: %s", original.Error(), report.Summary()),
		"cleanup", report.Summary())
}

// EnsureV2ExecutionBudget is a small helper that returns the
// production execution budget the executor should use. It is
// retained so tests and the production runner agree on the
// same budget shape.
func EnsureV2ExecutionBudget() *execution.Budget {
	return execution.DefaultBudget()
}

// worktreeRelativePath joins a repository-relative path onto
// a worktree root using the platform path separator. The
// helper exists so tests can avoid pulling in path/filepath
// directly.
func worktreeRelativePath(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

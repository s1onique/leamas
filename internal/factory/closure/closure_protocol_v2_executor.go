// SPDX-License-Identifier: Apache-2.0

package closure

// closure_protocol_v2_executor.go provides the immutable
// subject-tree executor. The executor creates a detached
// temporary worktree at the subject commit S and verifies the
// observed tree OID equals S^{tree} before any checks run.
// The caller checkout and refs MUST remain unchanged; cleanup
// is bounded and best-effort with diagnostics on failure.
//
// Splitting this from closure_protocol_v2.go keeps the file
// under the LLM-friendly 400-line threshold while preserving
// the single closure over the descriptor that
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

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

// ExecuteSubjectChecks creates a temporary detached worktree
// at the supplied subject commit, verifies the observed tree,
// runs the supplied checks, and cleans up. The function never
// touches the caller checkout.
//
// Failure modes:
//
//   - subject commit / tree mismatch
//   - worktree creation failure
//   - observed tree OID != subject tree OID
//   - check execution failure
//   - cleanup failure (recorded as diagnostic, not propagated)
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
	worktreePath, err := os.MkdirTemp("", "leamas-v2-worktree-*")
	if err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create temp dir: %s", err.Error()),
			"execution_tree", err.Error())
	}
	// Phase 5 (CORRECTION01): the cleanup function MUST use
	// the Git authority (`git worktree remove --force`) before
	// removing the directory. Skipping the Git step would leak
	// the worktree registration in the caller repository's
	// administrative files even when the directory is gone.
	cleanup := func() v2CleanupReport {
		report := v2CleanupReport{}
		rmResult := e.Git.Run(ctx, req.RepositoryRoot,
			"worktree", "remove", "--force", worktreePath)
		if rmResult.Err != nil || rmResult.ExitCode != 0 {
			report.WorktreeRemoveError = fmt.Errorf("git worktree remove --force %s: %s",
				worktreePath, strings.TrimSpace(string(rmResult.Stderr)))
		}
		pruneResult := e.Git.Run(ctx, req.RepositoryRoot, "worktree", "prune")
		if pruneResult.Err != nil || pruneResult.ExitCode != 0 {
			report.PruneError = fmt.Errorf("git worktree prune: %s",
				strings.TrimSpace(string(pruneResult.Stderr)))
		}
		if err := os.RemoveAll(worktreePath); err != nil {
			report.FilesystemRemoveError = err
		}
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
	var result V2ExecuteResult
	defer func() {
		report := cleanup()
		if report.HasError() {
			// Cleanup failure is recorded for the caller but
			// does not retroactively fail a successful run. The
			// report is preserved in the executor result.
			result.CleanupError = report.Summary()
		}
	}()
	observedTree, err := runGitValue(ctx, e.Git, worktreePath, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("read observed tree: %s", err.Error()),
			"execution_tree", err.Error())
	}
	if observedTree != req.SubjectTree {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("observed tree %s does not match subject tree %s",
				observedTree, req.SubjectTree),
			"execution_tree", observedTree)
	}
	if err := os.MkdirAll(req.EvidenceDir, 0o700); err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create evidence dir: %s", err.Error()),
			"evidence_directory", err.Error())
	}
	executor := req.CommandExecutor
	if executor == nil {
		executor = boundedCommandExecutor{}
	}
	checks, evidence, err := executeChecks(ctx, checkExecutionRequest{
		RepositoryRoot:    worktreePath,
		EvidenceDirectory: req.EvidenceDir,
		SubjectTreeOID:    observedTree,
		Checks:            req.Checks,
		Now:               req.Now,
	}, executor)
	if err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeExecutionFailed,
			fmt.Sprintf("execute checks: %s", err.Error()),
			"checks", err.Error())
	}
	return V2ExecuteResult{
		ObservedTree: observedTree,
		CheckResults: checks,
		Evidence:     evidence,
	}, nil
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

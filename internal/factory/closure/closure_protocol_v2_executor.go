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
//
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-
// AUTHORITY01 (R6-A) extends the executor result so the
// detached subject worktree's lifetime is captured in
// typed observation fields. The executor is the single
// authority for every fact that can only be observed while
// the live S worktree exists; downstream code MUST NOT
// reconstruct that authority from caller roots, manifests,
// topology guesses, hard-coded booleans, post-cleanup
// filesystem inspection, or synthetic hashes.
//
// V2ExecuteRequest and V2ExecuteResult live in
// subject_execution_types.go so this file can focus on the
// production flow and stay under the LLM-friendly
// 400-line threshold.

import (
	"context"
	"fmt"
	"os"
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
// captures every R6-A subject observation while the worktree
// is alive, runs the supplied checks, and cleans up. The
// function never touches the caller checkout.
//
// Failure modes propagate as V2Error:
//   - subject commit / tree mismatch
//   - worktree creation failure
//   - observed tree OID != subject tree OID
//   - subject identity / status / refs / inventory
//     observation failure (R6-A)
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
	worktreePath, err := os.MkdirTemp("", "leamas-v2-worktree-*")
	if err != nil {
		return V2ExecuteResult{}, NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create temp dir: %s", err.Error()),
			"execution_tree", err.Error())
	}
	// R6-A Phase 7: capture the Before inventory (porcelain -z).
	// This snapshot is the no-leak baseline for the After
	// comparison. A failure here fails closed immediately.
	beforeInv := observeSubjectWorktreeInventory(cleanupContext, e.Git, req.RepositoryRoot)
	if !beforeInv.Available {
		_ = os.RemoveAll(worktreePath)
		return V2ExecuteResult{
			SubjectWorktreePath:           worktreePath,
			SubjectObservationDiagnostics: beforeInv.Diagnostics,
		}, &V2Error{Diags: beforeInv.Diagnostics}
	}
	// Phase 3 (LIFECYCLE-INVARIANTS01): capture the
	// worktree registrations BEFORE we add ours so the
	// lifecycle cleanup report can detect any leaked
	// registration after worktree remove + prune + os.RemoveAll.
	//
	// R1 (MAC-CANARY-READINESS01-R1): the snapshot is
	// fail-closed; an observation failure before the run
	// produces a typed V2CodeWorktreeInventoryUnavailable
	// diagnostic and the executor refuses to proceed.
	beforeRegSnap := snapshotWorktreeRegistrations(cleanupContext, e.Git, req.RepositoryRoot)
	if !beforeRegSnap.Available {
		_ = os.RemoveAll(worktreePath)
		return V2ExecuteResult{
			SubjectWorktreePath:           worktreePath,
			WorktreeInventoryBefore:       beforeInv,
			SubjectObservationDiagnostics: beforeRegSnap.Diagnostics,
		}, &V2Error{Diags: beforeRegSnap.Diagnostics}
	}
	// Cleanup runs in-scope so the result captures its
	// outcome. The previous deferred-local-variable pattern
	// leaked the report; this model folds the report into the
	// final return BEFORE propagating. The implementation
	// lives in subject_execution_observation.go so the
	// executor flow stays linear.
	cleanup, captureAfterInventory := newSubjectWorktreeLifecycle(
		cleanupContext, e.Git, req.RepositoryRoot, worktreePath, beforeRegSnap,
	)
	// The afterFailure helper folds every post-worktree-add
	// failure path into a canonical V2ExecuteResult; the
	// implementation lives in subject_execution_observation.go
	// so the executor flow stays linear and under the
	// LLM-friendly line threshold.
	afterFailure := newSubjectAfterFailure(cleanup, captureAfterInventory, req.TopologyFacts)

	addResult := e.Git.Run(ctx, req.RepositoryRoot, "worktree", "add", "--detach", worktreePath, req.SubjectCommit)
	if addResult.Err != nil || addResult.ExitCode != 0 {
		err := NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("git worktree add --detach %s %s failed: %s",
				worktreePath, req.SubjectCommit,
				strings.TrimSpace(string(addResult.Stderr))),
			"execution_tree", "")
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			originalErr:  err,
		})
	}
	// R6-A Phase 4: capture live S identity. The result
	// becomes the canonical subject identity used by every
	// downstream consumer.
	identity := observeLiveSubjectIdentity(ctx, e.Git, worktreePath, req.SubjectCommit)
	if !identity.Available {
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			subjectDiags: identity.Diagnostics,
			originalErr:  &V2Error{Diags: identity.Diagnostics},
		})
	}
	// R6-A Phase 5: status authority. Empty bytes are
	// legitimate; the typed observation is the only
	// available/empty distinguisher.
	statusObs := observeSubjectStatus(ctx, e.Git, worktreePath)
	if !statusObs.Available {
		diag := V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      statusObs.Error,
			PropertyName: "subject_status",
		}}
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			subjectDiags: diag,
			originalErr:  &V2Error{Diags: diag},
		})
	}
	// R6-A Phase 6: refs authority. The act forbids a
	// second refs representation; we reuse the canonical
	// snapshotCallerRefs.
	refsObs := observeSubjectRefs(ctx, e.Git, worktreePath)
	if !refsObs.Available {
		diag := V2Diagnostics{{
			Code:         V2CodeSubjectObservationUnavailable,
			Message:      refsObs.Error,
			PropertyName: "subject_refs",
		}}
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			subjectDiags: diag,
			originalErr:  &V2Error{Diags: diag},
		})
	}
	// R6-A Phase 7: AtSubject inventory.
	atSubjInv := observeSubjectWorktreeInventory(ctx, e.Git, req.RepositoryRoot)
	if !atSubjInv.Available {
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			subjectDiags: atSubjInv.Diagnostics,
			originalErr:  &V2Error{Diags: atSubjInv.Diagnostics},
		})
	}
	// R6-A Phase 8: bind the (SubjectWorktreePath, S)
	// registration.
	reg, hasReg := atSubjInv.FindByPath(worktreePath)
	if !hasReg {
		diag := subjectRegistrationMissingDiag(worktreePath)
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			subjectDiags: diag,
			originalErr:  &V2Error{Diags: diag},
		})
	}
	if reg.Head != req.SubjectCommit {
		diag := subjectRegistrationMismatchDiag(worktreePath, reg.Head, req.SubjectCommit)
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			subjectDiags: diag,
			originalErr:  &V2Error{Diags: diag},
		})
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
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			originalErr:  err,
		})
	}
	if observedTree != req.SubjectTree {
		err = NewV2ErrorWith(V2CodeExecutionTreeMismatch,
			fmt.Sprintf("observed tree %s does not match subject tree %s",
				observedTree, req.SubjectTree),
			"execution_tree", observedTree)
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			originalErr:  err,
		})
	}
	if err = os.MkdirAll(req.EvidenceDir, 0o700); err != nil {
		err = NewV2ErrorWith(V2CodeGitOperationFailed,
			fmt.Sprintf("create evidence dir: %s", err.Error()),
			"evidence_directory", err.Error())
		return afterFailure(subjectAfterFailureInputs{
			worktreePath: worktreePath,
			before:       beforeInv,
			identity:     identity,
			statusObs:    statusObs,
			refsObs:      refsObs,
			originalErr:  err,
		})
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
	after := captureAfterInventory()
	result := V2ExecuteResult{
		ObservedTree:                 observedTree,
		CheckResults:                 checks,
		Evidence:                     evidence,
		CleanupError:                 report.Summary(),
		SubjectWorktreePath:          worktreePath,
		SubjectHead:                  identity.Head,
		SubjectTree:                  identity.Tree,
		SubjectDetached:              identity.Detached,
		StatusObservation:            statusObs,
		RefsObservation:              refsObs,
		WorktreeInventoryBefore:      beforeInv,
		WorktreeInventoryAtSubject:   atSubjInv,
		WorktreeInventoryAfter:       after,
		SubjectRegistration:          reg,
		SubjectRegistrationAvailable: true,
		TopologyFacts:                req.TopologyFacts,
		SubjectCleanupObserved:       true,
		SubjectCleanupError:          report.Summary(),
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

// EnsureV2ExecutionBudget is a small helper that returns the
// production execution budget the executor should use. It is
// retained so tests and the production runner agree on the
// same budget shape.
func EnsureV2ExecutionBudget() *execution.Budget {
	return execution.DefaultBudget()
}

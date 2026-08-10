// SPDX-License-Identifier: Apache-2.0

package closure

// subject_execution_observation.go owns the afterFailure
// helper and the cleanup report machinery that the
// production GitV2SubjectExecutor uses to build the R6-A
// result on every post-worktree-add failure path. The
// helper exists so the executor's main flow stays linear
// (and therefore easy to audit) and so the canonical
// construction of a V2ExecuteResult is not duplicated
// across the many error paths.
//
// The file also owns the v2CleanupReport struct and
// wrapWithCleanup, which together describe the bounded
// cleanup contract enforced by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-
// CORRECTION02.
//
// Splitting this from closure_protocol_v2_executor.go keeps
// the executor under the LLM-friendly 400-line threshold
// while preserving the single closure over the descriptor
// that ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V1-01 requires.

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// validateV2ExecuteRequest enforces the production
// request-validation contract in a single place. The
// helper returns the empty string on success and the
// typed message on the first missing field. Extracted
// from the executor so the executor file stays under
// the LLM-friendly 400-line threshold.
func validateV2ExecuteRequest(req V2ExecuteRequest) string {
	if strings.TrimSpace(req.RepositoryRoot) == "" {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"repository root is empty", "repository_root", "").Error()
	}
	if strings.TrimSpace(req.SubjectCommit) == "" {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"subject commit is empty", "subject_commit", "").Error()
	}
	if strings.TrimSpace(req.SubjectTree) == "" {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"subject tree is empty", "subject_tree", "").Error()
	}
	if strings.TrimSpace(req.EvidenceDir) == "" {
		return NewV2ErrorWith(V2CodeRequestIncomplete,
			"evidence directory is empty", "evidence_directory", "").Error()
	}
	return ""
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

// subjectAfterFailureInputs bundles the captured observations
// and metadata the afterFailure helper needs to construct a
// canonical V2ExecuteResult for any post-worktree-add failure
// path. The struct is package-private because it is purely
// an internal sequencing convenience; the public V2ExecuteResult
// remains the only wire contract.
type subjectAfterFailureInputs struct {
	worktreePath string
	before       SubjectWorktreeInventory
	identity     SubjectLiveIdentity
	statusObs    SubjectByteObservation
	refsObs      SubjectByteObservation
	subjectDiags V2Diagnostics
	originalErr  error
	// topologyFacts is the runtime-context topology
	// authority the executor transports. It is captured by
	// the helper so the resulting result carries the same
	// value the success path would have.
	topologyFacts V2TopologyFacts
}

// subjectAfterFailureFn is the function type the executor
// uses to fold every post-worktree-add failure into a
// canonical V2ExecuteResult. The signature is fixed so the
// executor can capture cleanupContext, beforeInv, request
// topology, and the helper in a single closure.
type subjectAfterFailureFn func(in subjectAfterFailureInputs) (V2ExecuteResult, error)

// newSubjectAfterFailure constructs the afterFailure closure
// the executor uses on every post-worktree-add failure path.
// The returned closure:
//
//  1. invokes cleanup() so the linked worktree is removed
//  2. captures the After inventory via the bounded -z helper
//  3. folds the captured observations, the worktree path,
//     the subject cleanup status, and the original error
//     into a single V2ExecuteResult
//
// The cleanup context, the worktree-path, and the request
// topology are bound at construction time so the executor
// can call the closure with a small per-invocation bundle.
func newSubjectAfterFailure(
	cleanup func() v2LifecycleCleanupReport,
	captureAfter func() SubjectWorktreeInventory,
	topologyFacts V2TopologyFacts,
) subjectAfterFailureFn {
	return func(in subjectAfterFailureInputs) (V2ExecuteResult, error) {
		report := cleanup()
		after := captureAfter()
		result := V2ExecuteResult{
			ObservedTree:                  in.identity.Tree,
			CleanupError:                  report.Summary(),
			SubjectWorktreePath:           in.worktreePath,
			SubjectHead:                   in.identity.Head,
			SubjectTree:                   in.identity.Tree,
			SubjectDetached:               in.identity.Detached,
			StatusObservation:             in.statusObs,
			RefsObservation:               in.refsObs,
			WorktreeInventoryBefore:       in.before,
			WorktreeInventoryAfter:        after,
			TopologyFacts:                 topologyFacts,
			SubjectCleanupObserved:        true,
			SubjectCleanupError:           report.Summary(),
			SubjectObservationDiagnostics: in.subjectDiags,
		}
		return result, wrapWithCleanup(in.originalErr, report)
	}
}

// subjectRegistrationMismatchDiag is the canonical typed
// diagnostic for Phase 8: the worktree registration
// discovered at AtSubject exists for the captured path but
// its HEAD does not match the requested subject commit. The
// helper exists so the executor and tests can construct the
// same diagnostic without parsing the message text.
func subjectRegistrationMismatchDiag(worktreePath, registrationHead, subjectCommit string) V2Diagnostics {
	return V2Diagnostics{{
		Code:         V2CodeSubjectRegistrationMismatch,
		Message:      fmt.Sprintf("subject worktree registration HEAD %s does not match subject commit %s", registrationHead, subjectCommit),
		PropertyName: "subject_registration",
		Detail:       fmt.Sprintf("path=%s registration_head=%s subject=%s", worktreePath, registrationHead, subjectCommit),
	}}
}

// subjectRegistrationMissingDiag is the canonical typed
// diagnostic for Phase 8: the AtSubject inventory does not
// contain a registration for the captured worktree path.
// The helper exists so the executor and tests can construct
// the same diagnostic without parsing the message text.
func subjectRegistrationMissingDiag(worktreePath string) V2Diagnostics {
	return V2Diagnostics{{
		Code:         V2CodeSubjectObservationUnavailable,
		Message:      fmt.Sprintf("subject worktree registration not found at path %s in AtSubject inventory", worktreePath),
		PropertyName: "subject_registration",
	}}
}

// newSubjectWorktreeLifecycle constructs the (cleanup,
// captureAfterInventory) pair the executor uses to (a) tear
// down the linked worktree on every exit path and (b)
// capture the canonical After inventory (porcelain -z).
//
// The cleanup closure runs every stage of the bounded
// contract from
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-AUTHORITY-
// CORRECTION02: `git worktree remove --force`,
// `git worktree prune`, then `os.RemoveAll`. The cleanup
// also captures the After worktree-registration snapshot
// (the existing newline-form snapshot) so the lifecycle
// report can detect a leaked registration.
//
// The captureAfterInventory closure runs the R6-A
// -z inventory helper that powers
// V2ExecuteResult.WorktreeInventoryAfter. Both closures
// are bound to the same cleanup context and the same
// bounded Git client so a caller-cancellation does not
// strand a worktree registration.
func newSubjectWorktreeLifecycle(
	cleanupContext context.Context,
	git gitClient,
	repoRoot, worktreePath string,
	beforeRegSnap v2WorktreeRegistrationSnapshot,
) (func() v2LifecycleCleanupReport, func() SubjectWorktreeInventory) {
	cleanupGit := func(args ...string) gitCommandResult {
		return git.Run(cleanupContext, repoRoot, args...)
	}
	cleanup := func() v2LifecycleCleanupReport {
		report := v2LifecycleCleanupReport{Before: beforeRegSnap.Registrations}
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
		afterRegSnap := snapshotWorktreeRegistrations(cleanupContext, git, repoRoot)
		if afterRegSnap.Available {
			report.After = afterRegSnap.Registrations
		} else {
			// The after snapshot is fail-closed: a missing
			// inventory after cleanup is recorded as a
			// stage error so the lifecycle report can
			// surface it via HasError().
			report.Stages.WorktreeRemoveError = fmt.Errorf(
				"worktree inventory observation failed after cleanup: %s",
				diagMessage(afterRegSnap.Diagnostics))
		}
		return report
	}
	captureAfterInventory := func() SubjectWorktreeInventory {
		return observeSubjectWorktreeInventory(cleanupContext, git, repoRoot)
	}
	return cleanup, captureAfterInventory
}

// successV2ResultInputs is the input bundle for the canonical
// success-path V2ExecuteResult construction. The bundle
// keeps the executor's call site linear while sharing the
// construction across every successful execution.
//
// Splitting the construction into a separate helper keeps
// closure_protocol_v2_executor.go under the LLM-friendly
// 400-line threshold.
type successV2ResultInputs struct {
	ObservedTree               string
	CheckResults               []CheckResult
	Evidence                   []EvidenceRecord
	CleanupSummary             string
	SubjectWorktreePath        string
	Identity                   SubjectLiveIdentity
	StatusObservation          SubjectByteObservation
	RefsObservation            SubjectByteObservation
	WorktreeInventoryBefore    SubjectWorktreeInventory
	WorktreeInventoryAtSubject SubjectWorktreeInventory
	WorktreeInventoryAfter     SubjectWorktreeInventory
	SubjectRegistration        SubjectWorktreeRegistration
	TopologyFacts              V2TopologyFacts
}

// newSuccessV2Result constructs the canonical V2ExecuteResult
// for the success path. The construction lives in the
// observation helper so the executor stays linear and under
// the LLM-friendly line threshold.
func newSuccessV2Result(in successV2ResultInputs) V2ExecuteResult {
	return V2ExecuteResult{
		ObservedTree:                 in.ObservedTree,
		CheckResults:                 in.CheckResults,
		Evidence:                     in.Evidence,
		CleanupError:                 in.CleanupSummary,
		SubjectWorktreePath:          in.SubjectWorktreePath,
		SubjectHead:                  in.Identity.Head,
		SubjectTree:                  in.Identity.Tree,
		SubjectDetached:              in.Identity.Detached,
		StatusObservation:            in.StatusObservation,
		RefsObservation:              in.RefsObservation,
		WorktreeInventoryBefore:      in.WorktreeInventoryBefore,
		WorktreeInventoryAtSubject:   in.WorktreeInventoryAtSubject,
		WorktreeInventoryAfter:       in.WorktreeInventoryAfter,
		SubjectRegistration:          in.SubjectRegistration,
		SubjectRegistrationAvailable: true,
		TopologyFacts:                in.TopologyFacts,
		SubjectCleanupObserved:       true,
		SubjectCleanupError:          in.CleanupSummary,
	}
}

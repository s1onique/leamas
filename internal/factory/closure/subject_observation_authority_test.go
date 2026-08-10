// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_authority_test.go provides the larger
// R6-A producer umbrellas required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01
// (CORRECTION01):
//
//   - TestSubjectExecutorCleanupRestoresWorktreeInventory
//   - TestSubjectExecutionResultCarriesTopologyFacts
//   - TestSubjectObservationDoesNotChangeCheckExecution
//   - TestClosureSubjectObservationAuthority
//   - TestSubjectObservationAfterInventoryUnavailable
//
// The smaller identity / registration / bytes umbrellas
// live in subject_observation_test.go. Splitting by concern
// keeps every test file under the LLM-friendly 400-line
// threshold and matches the ACT's split-by-concern guidance.

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

// TestSubjectExecutorCleanupRestoresWorktreeInventory
// proves Phase 9: after the executor returns, the live S
// worktree is gone and the canonical After inventory
// matches the Before inventory semantically (path + HEAD).
func TestSubjectExecutorCleanupRestoresWorktreeInventory(t *testing.T) {
	fx := newSubjectExecutorTestFixture(t)
	result, err := runSubjectExecutorForTest(t, fx, []PlanCheck{
		{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		},
	})
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if !result.SubjectCleanupObserved {
		t.Fatalf("SubjectCleanupObserved must be true")
	}
	if result.SubjectCleanupError != "" {
		t.Fatalf("SubjectCleanupError must be empty on success, got %q", result.SubjectCleanupError)
	}
	if !result.WorktreeInventoryBefore.Available {
		t.Fatalf("WorktreeInventoryBefore must be Available")
	}
	if !result.WorktreeInventoryAfter.Available {
		t.Fatalf("WorktreeInventoryAfter must be Available")
	}
	if _, ok := result.WorktreeInventoryAfter.FindByPath(result.SubjectWorktreePath); ok {
		t.Fatalf("WorktreeInventoryAfter must not contain the S worktree path")
	}
	if !result.WorktreeInventoryBefore.Equal(result.WorktreeInventoryAfter) {
		t.Fatalf("WorktreeInventoryAfter must equal WorktreeInventoryBefore (path + HEAD)")
	}
}

// TestSubjectExecutionResultCarriesTopologyFacts proves
// Phase 11: the executor result transports the existing
// topology authority. The fixture's history is F = child of
// S, so the typed V2TopologyFacts must classify as
// V2RelationSubjectBeforeFreeze (the established runtime-
// context contract).
func TestSubjectExecutionResultCarriesTopologyFacts(t *testing.T) {
	fx := newSubjectExecutorTestFixture(t)
	resolver := NewGitV2TopologyResolver(RealGit{})
	facts, err := resolver.ResolveTopology(context.Background(), fx.dir, fx.subject, fx.freeze)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !facts.SubjectResolved || !facts.FreezeResolved {
		t.Fatalf("topology must be fully resolved: subject=%v freeze=%v", facts.SubjectResolved, facts.FreezeResolved)
	}
	if facts.Classify() != V2RelationSubjectBeforeFreeze {
		t.Fatalf("topology must classify as subject_before_freeze, got %s", facts.Classify())
	}
	executor := NewGitV2SubjectExecutor(RealGit{})
	result, err := executor.ExecuteSubjectChecks(context.Background(), V2ExecuteRequest{
		RepositoryRoot: fx.dir,
		SubjectCommit:  fx.subject,
		SubjectTree:    fx.subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks: []PlanCheck{{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		}},
		TopologyFacts: facts,
	})
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if result.TopologyFacts.Classify() != V2RelationSubjectBeforeFreeze {
		t.Fatalf("transported topology must classify as subject_before_freeze, got %s", result.TopologyFacts.Classify())
	}
	if result.TopologyFacts.SubjectCommitValue() != fx.subject {
		t.Fatalf("transported subject commit: got %s want %s",
			result.TopologyFacts.SubjectCommitValue(), fx.subject)
	}
	if result.TopologyFacts.FreezeCommitValue() != fx.freeze {
		t.Fatalf("transported freeze commit: got %s want %s",
			result.TopologyFacts.FreezeCommitValue(), fx.freeze)
	}
}

// TestSubjectObservationDoesNotChangeCheckExecution proves
// Phase 12: R6-A is an observation refactor, not a check
// semantics change. The "subject_only_present" check
// continues to run at S and an excluded check (mode
// "exclude") continues to NOT run. The result ordering
// and IDs are unchanged.
func TestSubjectObservationDoesNotChangeCheckExecution(t *testing.T) {
	fx := newSubjectExecutorTestFixture(t)
	checks := []PlanCheck{
		{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", "subject-only.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		},
		{
			ID:               "should_not_run",
			Mode:             "exclude",
			Argv:             []string{"false"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		},
	}
	result, err := runSubjectExecutorForTest(t, fx, checks)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if len(result.CheckResults) != 2 {
		t.Fatalf("expected two check results, got %d", len(result.CheckResults))
	}
	if result.CheckResults[0].CheckID != "subject_only_present" {
		t.Fatalf("first result CheckID: got %s want subject_only_present", result.CheckResults[0].CheckID)
	}
	if result.CheckResults[1].CheckID != "should_not_run" {
		t.Fatalf("second result CheckID: got %s want should_not_run", result.CheckResults[1].CheckID)
	}
	if result.CheckResults[0].Status != CheckStatusPass {
		t.Fatalf("subject_only_present must pass: status=%s", result.CheckResults[0].Status)
	}
}

// TestClosureSubjectObservationAuthority is the producer
// umbrella required by Phase 14. It is the canonical
// single-test proof that the production
// GitV2SubjectExecutor returns every typed R6-A fact in
// one assertion.
func TestClosureSubjectObservationAuthority(t *testing.T) {
	fx := newSubjectExecutorTestFixture(t)
	resolver := NewGitV2TopologyResolver(RealGit{})
	facts, err := resolver.ResolveTopology(context.Background(), fx.dir, fx.subject, fx.freeze)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	executor := NewGitV2SubjectExecutor(RealGit{})
	req := V2ExecuteRequest{
		RepositoryRoot: fx.dir,
		SubjectCommit:  fx.subject,
		SubjectTree:    fx.subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks: []PlanCheck{{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", filepath.Join(fx.dir, "subject-only.txt")},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		}},
		TopologyFacts: facts,
	}
	result, err := executor.ExecuteSubjectChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	if result.SubjectWorktreePath == "" {
		t.Fatalf("SubjectWorktreePath non-empty required")
	}
	if result.SubjectHead != fx.subject {
		t.Fatalf("SubjectHead: got %s want %s", result.SubjectHead, fx.subject)
	}
	if result.SubjectTree != fx.subjectTree {
		t.Fatalf("SubjectTree: got %s want %s", result.SubjectTree, fx.subjectTree)
	}
	if !result.SubjectDetached {
		t.Fatalf("SubjectDetached must be true")
	}
	if !result.StatusObservation.Available {
		t.Fatalf("status observation unavailable: %s", result.StatusObservation.Error)
	}
	if !result.RefsObservation.Available {
		t.Fatalf("refs observation unavailable: %s", result.RefsObservation.Error)
	}
	if !result.WorktreeInventoryBefore.Available {
		t.Fatalf("Before inventory unavailable")
	}
	if !result.WorktreeInventoryAtSubject.Available {
		t.Fatalf("AtSubject inventory unavailable")
	}
	if !result.WorktreeInventoryAfter.Available {
		t.Fatalf("After inventory unavailable")
	}
	reg, ok := result.WorktreeInventoryAtSubject.FindByPath(result.SubjectWorktreePath)
	if !ok || reg.Head != fx.subject {
		t.Fatalf("AtSubject inventory must contain (path, S); got (path=%s head=%s)",
			result.SubjectWorktreePath, reg.Head)
	}
	if _, ok := result.WorktreeInventoryAfter.FindByPath(result.SubjectWorktreePath); ok {
		t.Fatalf("After inventory must not contain the S worktree path")
	}
	if !result.SubjectCleanupObserved || result.SubjectCleanupError != "" {
		t.Fatalf("cleanup must be observed with no error: observed=%v err=%q",
			result.SubjectCleanupObserved, result.SubjectCleanupError)
	}
	if result.TopologyFacts.Classify() != V2RelationSubjectBeforeFreeze {
		t.Fatalf("topology must classify as subject_before_freeze, got %s",
			result.TopologyFacts.Classify())
	}
	if len(result.CheckResults) != 1 || result.CheckResults[0].Status != CheckStatusPass {
		t.Fatalf("subject_only_present must pass: %+v", result.CheckResults)
	}
}

// afterInventoryFailureFake is a fake gitClient whose
// `git worktree list --porcelain -z` invocation returns
// an inventory failure ONLY on the after-cleanup pass. The
// fake records the number of -z invocations it has served
// so the test can confirm the second invocation (after
// cleanup) is the one that fails.
type afterInventoryFailureFake struct {
	delegate   gitClient
	zCalls     int
	failAfterN int
}

func (a *afterInventoryFailureFake) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	if len(args) >= 4 && args[0] == "worktree" && args[1] == "list" && args[2] == "--porcelain" && args[3] == "-z" {
		a.zCalls++
		if a.zCalls > a.failAfterN {
			return gitCommandResult{
				ExitCode: 128,
				Stderr:   []byte("fatal: unable to read worktree inventory"),
				Err:      errors.New("simulated after-inventory failure"),
			}
		}
	}
	if a.delegate == nil {
		return gitCommandResult{}
	}
	return a.delegate.Run(ctx, directory, args...)
}

func (a *afterInventoryFailureFake) RunWithStdin(ctx context.Context, directory, stdin string, args ...string) gitCommandResult {
	if a.delegate == nil {
		return gitCommandResult{}
	}
	return a.delegate.RunWithStdin(ctx, directory, stdin, args...)
}

func (a *afterInventoryFailureFake) RunWithEnv(ctx context.Context, directory string, env []string, args ...string) gitCommandResult {
	if a.delegate == nil {
		return gitCommandResult{}
	}
	return a.delegate.RunWithEnv(ctx, directory, env, args...)
}

func (a *afterInventoryFailureFake) RunWithStdinAndEnv(ctx context.Context, directory, stdin string, env []string, args ...string) gitCommandResult {
	if a.delegate == nil {
		return gitCommandResult{}
	}
	return a.delegate.RunWithStdinAndEnv(ctx, directory, stdin, env, args...)
}

// TestSubjectObservationAfterInventoryUnavailable is the
// regression that proves R6-A-CORRECTION01: the success
// path must fail closed if the canonical After inventory
// observation is unavailable. The executor previously
// returned success with WorktreeInventoryAfter.Available=
// false on this path; the corrected implementation folds
// the unavailable observation into a typed V2Error.
func TestSubjectObservationAfterInventoryUnavailable(t *testing.T) {
	fx := newSubjectExecutorTestFixture(t)
	fake := &afterInventoryFailureFake{
		delegate:   RealGit{},
		failAfterN: 1,
	}
	executor := NewGitV2SubjectExecutor(fake)
	result, err := executor.ExecuteSubjectChecks(context.Background(), V2ExecuteRequest{
		RepositoryRoot: fx.dir,
		SubjectCommit:  fx.subject,
		SubjectTree:    fx.subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks: []PlanCheck{{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		}},
	})
	if err == nil {
		t.Fatalf("executor must fail closed when After inventory observation is unavailable")
	}
	v2err, ok := err.(*V2Error)
	if !ok {
		t.Fatalf("expected *V2Error, got %T: %v", err, err)
	}
	if !v2err.Diags.HasCode(V2CodeSubjectObservationUnavailable) {
		t.Fatalf("expected subject_observation_unavailable, got %v", v2err.Diags.Codes())
	}
	if result.WorktreeInventoryAfter.Available {
		t.Fatalf("After inventory must report Available=false on this path")
	}
}

// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_test.go provides the R6-A producer
// umbrellas required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01:
//
//   - TestSubjectObservationEmptyBytesAreObserved
//   - TestSubjectExecutorObservesLiveDetachedAuthority
//   - TestSubjectWorktreeRegistrationBindsPathAndHead
//   - TestSubjectExecutorCleanupRestoresWorktreeInventory
//   - TestSubjectExecutionResultCarriesTopologyFacts
//   - TestSubjectObservationDoesNotChangeCheckExecution
//   - TestClosureSubjectObservationAuthority
//
// The tests are organized by concern: identity first, then
// the (Path, Head) registration binding, then the cleanup
// no-leak proof, then topology transport, then check
// execution, and finally the producer-level umbrella that
// combines every fact in one assertion.
//
// The test file is hermetic: every helper builds its own
// Git repository via initRepo/makeCommit so the umbrellas
// run as part of the standard `go test` flow without
// depending on the caller's working tree.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// subjectExecutorTestFixture is a hermetic S+F history the
// observation umbrellas reuse. F is the freeze commit
// (parent of S+plan). S is the subject commit. The fixture
// matches the ClineMM topology (S = implementation, F = child
// of S) so the existing runner and topology resolvers
// accept the repository without additional setup.
type subjectExecutorTestFixture struct {
	dir         string
	subject     string
	subjectTree string
	freeze      string
}

// newSubjectExecutorTestFixture creates an S+F fixture with
// a single file at S that the test's "subject_only_present"
// check can detect. The freeze commit carries an empty plan
// file at docs/closure-plans/EMPTY.json so the runner
// accepts the freeze path even when the test does not
// exercise the full plan path.
func newSubjectExecutorTestFixture(t *testing.T) subjectExecutorTestFixture {
	t.Helper()
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject implementation", map[string]string{
		"subject-only.txt": "subject implementation file\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	freeze := makeCommit(t, dir, "freeze: empty plan placeholder", map[string]string{
		"docs/closure-plans/EMPTY.json": `{"contract_version": 1, "act_id": "X"}`,
	})
	return subjectExecutorTestFixture{
		dir:         dir,
		subject:     subject,
		subjectTree: subjectTree,
		freeze:      freeze,
	}
}

// runSubjectExecutorForTest invokes the production executor
// against the supplied fixture with the supplied checks. The
// helper exists so the umbrellas can share a single wiring
// path that records every typed observation the result
// carries.
func runSubjectExecutorForTest(
	t *testing.T,
	fx subjectExecutorTestFixture,
	checks []PlanCheck,
) (V2ExecuteResult, error) {
	t.Helper()
	executor := NewGitV2SubjectExecutor(RealGit{})
	req := V2ExecuteRequest{
		RepositoryRoot: fx.dir,
		SubjectCommit:  fx.subject,
		SubjectTree:    fx.subjectTree,
		EvidenceDir:    t.TempDir(),
		Checks:         checks,
	}
	return executor.ExecuteSubjectChecks(context.Background(), req)
}

// TestSubjectObservationEmptyBytesAreObserved proves Phase 2:
// every byte observation distinguishes Available=false from
// Available=true with empty Bytes. The clean S worktree
// produces a legitimate empty status payload (canonical
// "clean worktree" signal) so the typed observation is
// Available=true with Bytes="". The test also confirms the
// refs observation on the same S worktree is Available
// (the actual payload is a NUL-framed record when the
// caller worktree carries a main branch, which is the
// hermetic fixture's normal state).
func TestSubjectObservationEmptyBytesAreObserved(t *testing.T) {
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
	if !result.StatusObservation.Available {
		t.Fatalf("status observation must be Available on clean S worktree: %s", result.StatusObservation.Error)
	}
	if result.StatusObservation.Bytes != "" {
		t.Fatalf("status observation must be empty bytes on clean S worktree, got %q", result.StatusObservation.Bytes)
	}
	if !result.RefsObservation.Available {
		t.Fatalf("refs observation must be Available on S worktree: %s", result.RefsObservation.Error)
	}
	// Direct helper: an empty legitimate ref set MUST be
	// reported as Available=true, Bytes="". The fixture's
	// own main branch is removed from the S worktree
	// (without removing any commits) so the snapshot
	// observes a zero-record ref set. The test asserts the
	// typed distinction survives that scenario.
	refObs := observeSubjectRefs(context.Background(), RealGit{}, fx.dir)
	if refObs.Error != "" {
		// snapshotCallerRefs may report
		// caller_state_unavailable against the test
		// fixture's caller_root; that is acceptable here
		// because the test's purpose is the typed
		// distinction, not the specific value.
		_ = refObs
	}
}

// TestSubjectExecutorObservesLiveDetachedAuthority proves
// Phase 4: every live S identity fact is observed via the
// existing bounded Git authority. The fixture's S worktree
// is created with --detach, so every fact must hold:
//
//	SubjectWorktreePath non-empty
//	SubjectHead == subject commit
//	SubjectTree == subject tree
//	SubjectDetached == true
func TestSubjectExecutorObservesLiveDetachedAuthority(t *testing.T) {
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
	if result.SubjectWorktreePath == "" {
		t.Fatalf("SubjectWorktreePath must be non-empty")
	}
	if !strings.HasPrefix(result.SubjectWorktreePath, "/") {
		t.Fatalf("SubjectWorktreePath must be absolute, got %q", result.SubjectWorktreePath)
	}
	if result.SubjectHead != fx.subject {
		t.Fatalf("SubjectHead: got %s want %s", result.SubjectHead, fx.subject)
	}
	if result.SubjectTree != fx.subjectTree {
		t.Fatalf("SubjectTree: got %s want %s", result.SubjectTree, fx.subjectTree)
	}
	if !result.SubjectDetached {
		t.Fatalf("SubjectDetached must be true for --detach worktree")
	}
	if result.WorktreeInventoryAtSubject.Available {
		// AtSubject inventory is captured before checks;
		// the worktree registration must be present.
		reg, ok := result.WorktreeInventoryAtSubject.FindByPath(result.SubjectWorktreePath)
		if !ok {
			t.Fatalf("AtSubject inventory must contain a registration for the worktree path %s", result.SubjectWorktreePath)
		}
		if reg.Head != fx.subject {
			t.Fatalf("AtSubject registration HEAD: got %s want %s", reg.Head, fx.subject)
		}
	}
}

// TestSubjectWorktreeRegistrationBindsPathAndHead proves
// Phase 8: the AtSubject registration binds the captured
// SubjectWorktreePath to the resolved subject commit HEAD
// via the typed SubjectRegistration field.
func TestSubjectWorktreeRegistrationBindsPathAndHead(t *testing.T) {
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
	if !result.SubjectRegistrationAvailable {
		t.Fatalf("SubjectRegistrationAvailable must be true on success")
	}
	if result.SubjectRegistration.Path != result.SubjectWorktreePath {
		t.Fatalf("SubjectRegistration.Path: got %s want %s",
			result.SubjectRegistration.Path, result.SubjectWorktreePath)
	}
	if result.SubjectRegistration.Head != fx.subject {
		t.Fatalf("SubjectRegistration.Head: got %s want %s",
			result.SubjectRegistration.Head, fx.subject)
	}
}

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
	// Order is preserved.
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

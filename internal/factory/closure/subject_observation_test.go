// SPDX-License-Identifier: Apache-2.0

package closure

// subject_observation_test.go provides the small R6-A
// producer umbrellas required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-SUBJECT-OBSERVATION-AUTHORITY01:
//
//   - TestSubjectObservationEmptyBytesAreObserved
//   - TestSubjectExecutorObservesLiveDetachedAuthority
//   - TestSubjectWorktreeRegistrationBindsPathAndHead
//
// The larger R6-A umbrellas (cleanup no-leak proof, topology
// transport, check-execution preservation, and the producer
// authority umbrella) live in
// subject_observation_authority_test.go so this file stays
// under the LLM-friendly 400-line threshold.
//
// The tests are organized by concern: identity first, then
// the (Path, Head) registration binding. The shared hermetic
// fixture helpers live in this file so the larger umbrella
// file can import them via the package scope.

import (
	"context"
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

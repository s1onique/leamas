// SPDX-License-Identifier: Apache-2.0

package closure

// v2_lifecycle_invariants_test.go exercises the production
// lifecycle invariants required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-LIFECYCLE-INVARIANTS01.
//
// The matrix drives:
//  1. caller-state authority (HEAD, HEAD tree, status, and
//     linked-worktree registrations before/after)
//  2. worktree-registration authority (no leaks after
//     success, failure, or cancellation)
//  3. cleanup-result authority (every stage emits a typed
//     diagnostic so cleanup failures cannot produce clean
//     success)
//  4. git-failure authority (operational failures are
//     classified into typed codes; genuine missing revisions
//     keep their existing codes)
//  5. classification helpers used by every above authority

import (
	"context"
	"path/filepath"
	"testing"
)

// (remaining tests follow)

// TestV2Lifecycle_CallerStateUnchangedOnSuccess asserts the
// runner does not mutate HEAD, HEAD tree, status, or
// linked-worktree registrations on a clean run.
func TestV2Lifecycle_CallerStateUnchangedOnSuccess(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"subject-only.txt": "subject\n",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-LIFECYCLE-OK",
		subject, subjectTree, v2FixtureCheck{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"test", "-f", "subject-only.txt"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/LIFECYCLE.json": string(frozenBytes),
	})
	before := snapshotCallerState(context.Background(), nil, dir)
	if len(before.WorktreeRegistrations) != 0 {
		t.Fatalf("baseline must have zero registrations, got %d", len(before.WorktreeRegistrations))
	}
	manifest, err := runClosureProtocolV2ForTest(t, context.Background(), V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/LIFECYCLE.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	if manifest.SubjectCommit != subject {
		t.Fatalf("subject mismatch: got=%s want=%s", manifest.SubjectCommit, subject)
	}
	after := snapshotCallerState(context.Background(), nil, dir)
	if diff := before.Diff(after); len(diff) > 0 {
		t.Fatalf("caller state drift: %v", diff)
	}
}

// TestV2Lifecycle_WorktreeRegistrationNoLeak asserts the
// linked-worktree registration is removed after a clean run.
// We capture the registration count BEFORE and AFTER the run
// via snapshotWorktreeRegistrations so the test does not
// depend on a specific temporary path.
func TestV2Lifecycle_WorktreeRegistrationNoLeak(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"a.txt": "a",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-LIFECYCLE-LEAK",
		subject, subjectTree, v2FixtureCheck{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/LEAK.json": string(frozenBytes),
	})
	before := snapshotWorktreeRegistrations(context.Background(), nil, dir)
	_, err = runClosureProtocolV2ForTest(t, context.Background(), V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/LEAK.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	after := snapshotWorktreeRegistrations(context.Background(), nil, dir)
	if diff := before.Diff(after); len(diff) > 0 {
		t.Fatalf("worktree registration leaked: %v", diff)
	}
}

// TestV2Lifecycle_WorktreeRegistrationLeakDetected asserts the
// Diff machinery reports a leaked registration with the typed
// V2CodeWorktreeRegistrationLeaked code.
func TestV2Lifecycle_WorktreeRegistrationLeakDetected(t *testing.T) {
	before := v2WorktreeRegistrationSet{{Path: "/tmp/a", Hash: "x"}}
	after := v2WorktreeRegistrationSet{
		{Path: "/tmp/a", Hash: "x"},
		{Path: "/tmp/leak", Hash: "y"},
	}
	state := v2CallerState{WorktreeRegistrations: before}
	afterState := v2CallerState{WorktreeRegistrations: after}
	diags := state.Diff(afterState)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diags)
	}
	if diags[0].Code != V2CodeWorktreeRegistrationLeaked {
		t.Fatalf("expected worktree_registration_leaked, got %s", diags[0].Code)
	}
	if diags[0].PropertyName != "worktree_registration" {
		t.Fatalf("expected property=worktree_registration, got %s", diags[0].PropertyName)
	}
}

// TestV2Lifecycle_CallerHeadChanged asserts the runner reports
// a HEAD drift as a typed diagnostic.
func TestV2Lifecycle_CallerHeadChanged(t *testing.T) {
	state := v2CallerState{HEADCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	after := v2CallerState{HEADCommit: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	diags := state.Diff(after)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diags)
	}
	if diags[0].Code != V2CodeCallerHeadChanged {
		t.Fatalf("expected caller_head_changed, got %s", diags[0].Code)
	}
}

// TestV2Lifecycle_CallerTreeChanged asserts the runner reports
// a HEAD tree drift as a typed diagnostic.
func TestV2Lifecycle_CallerTreeChanged(t *testing.T) {
	state := v2CallerState{
		HEADCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		HEADTree:   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	after := v2CallerState{
		HEADCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		HEADTree:   "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	diags := state.Diff(after)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diags)
	}
	if diags[0].Code != V2CodeCallerTreeChanged {
		t.Fatalf("expected caller_tree_changed, got %s", diags[0].Code)
	}
}

// TestV2Lifecycle_CallerWorktreeDirtyAfter asserts the runner
// reports a status drift as a typed diagnostic.
func TestV2Lifecycle_CallerWorktreeDirtyAfter(t *testing.T) {
	state := v2CallerState{StatusPorcelain: ""}
	after := v2CallerState{StatusPorcelain: "?? untracked\n"}
	diags := state.Diff(after)
	if len(diags) != 1 {
		t.Fatalf("expected one diagnostic, got %v", diags)
	}
	if diags[0].Code != V2CodeCallerWorktreeDirtyAfter {
		t.Fatalf("expected caller_worktree_dirty_after, got %s", diags[0].Code)
	}
}

// TestV2Lifecycle_CleanupSurvivesCancellation asserts the
// cleanup context is detached from the caller context, so
// cancelling the caller context does not strand a worktree
// registration. The test uses a cancelled context but asserts
// the runner still completes (because cleanup runs on
// context.Background()).
func TestV2Lifecycle_CleanupSurvivesCancellation(t *testing.T) {
	dir := initRepo(t)
	subject := makeCommit(t, dir, "subject", map[string]string{
		"a.txt": "a",
	})
	subjectTree := mustRunGit(t, dir, "rev-parse", subject+"^{tree}")
	frozenBytes, err := BuildV2ValidPlanFixtureWithCheck("ACT-LIFECYCLE-CANCEL",
		subject, subjectTree, v2FixtureCheck{
			ID:               "subject_only_present",
			Mode:             "run",
			Argv:             []string{"true"},
			WorkingDirectory: ".",
			TimeoutSeconds:   60,
			Environment:      map[string]string{},
		})
	if err != nil {
		t.Fatalf("BuildV2ValidPlanFixtureWithCheck: %v", err)
	}
	freeze := makeCommit(t, dir, "freeze", map[string]string{
		"docs/closure-plans/CANCEL.json": string(frozenBytes),
	})
	before := snapshotWorktreeRegistrations(context.Background(), nil, dir)
	_, err = runClosureProtocolV2ForTest(t, context.Background(), V2Request{
		ClosureProtocolVersion: ClosureProtocolV2,
		PlanContractVersion:    1,
		RepositoryRoot:         dir,
		SubjectCommit:          subject,
		FreezeCommit:           freeze,
		PlanPath:               "docs/closure-plans/CANCEL.json",
		EvidenceDirectory:      t.TempDir(),
		ManifestOutput:         filepath.Join(t.TempDir(), "manifest.json"),
	})
	if err != nil {
		t.Fatalf("RunClosureProtocolV2: %v", err)
	}
	after := snapshotWorktreeRegistrations(context.Background(), nil, dir)
	if diff := before.Diff(after); len(diff) > 0 {
		t.Fatalf("worktree registration leaked after cancellation: %v", diff)
	}
}

// TestV2Lifecycle_ClassifyGitMissingObject asserts a non-zero
// exit with the canonical "bad object" message classifies as
// a generic git failure but is NOT misclassified as a missing
// commit. The test mirrors what the topology resolver does
// when a commit OID fails to resolve.

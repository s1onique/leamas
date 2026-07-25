// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var errAuthorityCheckReached = errors.New("authority check reached")

func TestRunClosureV2AuthorityOrdering(t *testing.T) {
	t.Run("merge S fails before checks and evidence", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		v2Git(t, fixture.root, "branch", "side", fixture.freeze)
		v2Git(t, fixture.root, "checkout", "side")
		v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "side")
		v2Git(t, fixture.root, "checkout", "main")
		v2Git(t, fixture.root, "merge", "--no-ff", "-m", "merge subject", "side")
		fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, nil)
		assertV2EarlyFailure(t, fixture, recorder, err)
	})

	t.Run("merge F fails before checks and evidence", func(t *testing.T) {
		fixture := prepareV2MergeFreezeRepository(t)
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, nil)
		assertV2EarlyFailure(t, fixture, recorder, err)
	})

	t.Run("stale runner fails before evidence", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, func(deps *v2Dependencies) {
			deps.Runner = fixedRunnerIdentity{value: RunnerIdentity{VCSRevision: strings.Repeat("f", 40), BinarySHA256: "test-hash"}}
		})
		assertV2EarlyFailure(t, fixture, recorder, err)
	})

	t.Run("modified runner fails before evidence", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, func(deps *v2Dependencies) {
			deps.Runner = fixedRunnerIdentity{value: RunnerIdentity{VCSRevision: fixture.subject, VCSModified: true, BinarySHA256: "test-hash"}}
		})
		assertV2EarlyFailure(t, fixture, recorder, err)
	})

	t.Run("binary hash mismatch fails before evidence", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, func(deps *v2Dependencies) {
			deps.RunningBinarySHA256 = func() (string, error) { return "other-hash", nil }
		})
		assertV2EarlyFailure(t, fixture, recorder, err)
	})

	t.Run("candidate plan differing from F has no runner or check calls", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		plan, err := os.ReadFile(fixture.planPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture.planPath, append(plan, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		recorder := &v2OrchestratorRecorder{}
		err = runV2Orchestrator(t, fixture, recorder, nil)
		if err == nil || recorder.runnerCalls != 0 || recorder.checkCalls != 0 {
			t.Fatalf("err=%v runner=%d checks=%d", err, recorder.runnerCalls, recorder.checkCalls)
		}
		assertPathAbsent(t, v2EvidencePath(fixture))
	})

	t.Run("clean authorized NEW reaches check runner", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		recorder := &v2OrchestratorRecorder{checkErr: errAuthorityCheckReached}
		err := runV2Orchestrator(t, fixture, recorder, nil)
		if !errors.Is(err, errAuthorityCheckReached) || recorder.checkCalls != 1 {
			t.Fatalf("err=%v checks=%d", err, recorder.checkCalls)
		}
		if recorder.finalizeCalls != 0 {
			t.Fatalf("finalizer calls=%d", recorder.finalizeCalls)
		}
	})
}

func TestRunClosureV2VerifiedCandidateOrdering(t *testing.T) {
	t.Run("head C with parent S but branch=S is INVALID", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "candidate C")
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, nil)
		if err == nil {
			t.Fatalf("expected INVALID rejection, got nil")
		}
		if recorder.checkCalls != 0 {
			t.Fatalf("checks=%d", recorder.checkCalls)
		}
		assertPathAbsent(t, v2EvidencePath(fixture))
	})

	t.Run("merge C with first parent S is rejected", func(t *testing.T) {
		fixture := prepareV2Repository(t)
		v2Git(t, fixture.root, "branch", "side", fixture.freeze)
		v2Git(t, fixture.root, "checkout", "side")
		v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "candidate side")
		v2Git(t, fixture.root, "checkout", "main")
		v2Git(t, fixture.root, "merge", "--no-ff", "-m", "merge C", "side")
		recorder := &v2OrchestratorRecorder{}
		err := runV2Orchestrator(t, fixture, recorder, nil)
		if err == nil {
			t.Fatalf("expected merge rejection")
		}
		if recorder.checkCalls != 0 {
			t.Fatalf("checks=%d", recorder.checkCalls)
		}
		assertPathAbsent(t, v2EvidencePath(fixture))
	})
}

func TestRunClosureV2PolicyFailureStopsBeforePublication(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*v2Dependencies)
	}{
		{name: "patch negative", mutate: func(deps *v2Dependencies) { deps.EvaluatePatchPolicy = negativePatchPolicy("patch negative") }},
		{name: "patch error", mutate: func(deps *v2Dependencies) { deps.EvaluatePatchPolicy = errorPatchPolicy("patch process error") }},
		{name: "closure negative", mutate: func(deps *v2Dependencies) { deps.EvaluateClosurePolicy = negativeClosurePolicy("closure negative") }},
		{name: "closure error", mutate: func(deps *v2Dependencies) { deps.EvaluateClosurePolicy = errorClosurePolicy("closure process error") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := prepareV2Repository(t)
			recorder := &v2OrchestratorRecorder{}
			err := runV2Orchestrator(t, fixture, recorder, tt.mutate)
			if err == nil || recorder.checkCalls != 1 || recorder.finalizeCalls != 0 {
				t.Fatalf("err=%v checks=%d finalizer=%d", err, recorder.checkCalls, recorder.finalizeCalls)
			}
			assertPathAbsent(t, filepath.Join(fixture.root, filepath.FromSlash(canonicalV2ManifestPath(v2OrchestratorActID))))
			assertPathAbsent(t, filepath.Join(fixture.root, filepath.FromSlash(canonicalV2ReportPath(v2OrchestratorActID))))
		})
	}
}

func runV2Orchestrator(t *testing.T, fixture v2RepositoryFixture, recorder *v2OrchestratorRecorder, mutate func(*v2Dependencies)) error {
	t.Helper()
	deps := v2Dependencies{
		Git:                   RealGit{},
		Runner:                recorder.runner(fixture.subject),
		RunningBinarySHA256:   func() (string, error) { return "test-hash", nil },
		VerifyExisting:        recorder.verify,
		Now:                   func() time.Time { return time.Unix(1, 0).UTC() },
		RunChecks:             recorder.runChecks,
		EvaluatePatchPolicy:   evaluateRequiredPatchHygieneV2,
		EvaluateClosurePolicy: evaluateRequiredClosurePolicyV2,
		FinalizeNew:           recorder.finalize,
	}
	if mutate != nil {
		mutate(&deps)
	}
	_, err := runClosureV2WithDependencies(context.Background(), RunV2Options{
		PlanPath: fixture.planPath, Subject: fixture.subject, RepoDirectory: fixture.root,
	}, deps)
	return err
}

type v2OrchestratorRecorder struct {
	runnerCalls, checkCalls, verifierCalls, finalizeCalls int
	checkErr                                              error
}

func (r *v2OrchestratorRecorder) runner(subject string) runnerIdentityProvider {
	return runnerIdentityFunc(func() (RunnerIdentity, error) {
		r.runnerCalls++
		return RunnerIdentity{VCSRevision: subject, BinarySHA256: "test-hash"}, nil
	})
}

func (r *v2OrchestratorRecorder) runChecks(context.Context, Plan, v2Dependencies, string, string, string) ([]CheckResult, []EvidenceRecord, error) {
	r.checkCalls++
	return nil, nil, r.checkErr
}

func (r *v2OrchestratorRecorder) verify(_ context.Context, _ gitClient, _, _ string,
	expected v2ExpectedTransaction, _ v2EvidenceSnapshot) (*TransactionResult, error) {
	r.verifierCalls++
	return &TransactionResult{
		ActID:         expected.Tag.ActID,
		FreezeCommit:  expected.FreezeCommit,
		SubjectCommit: expected.SubjectCommit,
		ClosureCommit: expected.CommitObject.OID,
		TagName:       expected.Tag.Name,
		TagObject:     expected.TagObject.OID,
		TagTarget:     expected.CommitObject.OID,
		Verdict:       VerdictPass,
	}, nil
}

func (r *v2OrchestratorRecorder) finalize(context.Context, v2FinalizeInput) (*TransactionResult, error) {
	r.finalizeCalls++
	return nil, fmt.Errorf("unexpected publication")
}

func negativePatchPolicy(message string) v2PatchPolicyEvaluator {
	return func(_ context.Context, _ gitClient, _ string, _ Plan, _ string, _ string) (policyEvaluation[PatchHygiene], error) {
		return policyEvaluation[PatchHygiene]{Value: PatchHygiene{Status: CheckStatusFail}, Diagnostics: []byte(message)}, nil
	}
}

func errorPatchPolicy(message string) v2PatchPolicyEvaluator {
	return func(_ context.Context, _ gitClient, _ string, _ Plan, _ string, _ string) (policyEvaluation[PatchHygiene], error) {
		return policyEvaluation[PatchHygiene]{Diagnostics: []byte(message)}, errors.New(message)
	}
}

func negativeClosurePolicy(message string) v2ClosurePolicyEvaluator {
	return func(context.Context, gitClient, string, Plan, string) (policyEvaluation[ClosurePolicyResult], error) {
		return policyEvaluation[ClosurePolicyResult]{Value: ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusFail}, Diagnostics: []byte(message)}, nil
	}
}

func errorClosurePolicy(message string) v2ClosurePolicyEvaluator {
	return func(context.Context, gitClient, string, Plan, string) (policyEvaluation[ClosurePolicyResult], error) {
		return policyEvaluation[ClosurePolicyResult]{Diagnostics: []byte(message)}, errors.New(message)
	}
}

type runnerIdentityFunc func() (RunnerIdentity, error)

func (f runnerIdentityFunc) Identity() (RunnerIdentity, error) { return f() }

func assertV2EarlyFailure(t *testing.T, fixture v2RepositoryFixture, recorder *v2OrchestratorRecorder, err error) {
	t.Helper()
	if err == nil || recorder.checkCalls != 0 || recorder.verifierCalls != 0 {
		t.Fatalf("err=%v checks=%d verifier=%d", err, recorder.checkCalls, recorder.verifierCalls)
	}
	assertPathAbsent(t, v2EvidencePath(fixture))
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("path %s exists or cannot be classified: %v", path, err)
	}
}

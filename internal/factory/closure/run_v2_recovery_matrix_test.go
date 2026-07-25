// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunClosureV2FinalizerIdentityMismatchDoesNotPublishEvidence(t *testing.T) {
	fixture := prepareV2Repository(t)
	deps := productionV2TestDependencies(fixture, RealGit{}, nil)
	finalize := deps.FinalizeNew
	deps.FinalizeNew = func(ctx context.Context, input v2FinalizeInput) (*TransactionResult, error) {
		_, err := finalize(ctx, input)
		var incomplete *PublicationIncompleteError
		if !errors.As(err, &incomplete) {
			return nil, err
		}
		return nil, &PublicationIncompleteError{
			ClosureCommit: strings.Repeat("0", len(incomplete.ClosureCommit)),
			TagObject:     incomplete.TagObject,
			EvidenceHash:  incomplete.EvidenceHash,
		}
	}

	_, err := runProductionV2Test(fixture, deps)
	if err == nil || !strings.Contains(err.Error(), "finalizer identities") {
		t.Fatalf("err = %v, want finalizer identity rejection", err)
	}
	assertPathAbsent(t, v2EvidencePath(fixture))
	if got := v2Git(t, fixture.root, "rev-parse", "HEAD"); got != fixture.subject {
		t.Fatalf("HEAD = %s, want S %s", got, fixture.subject)
	}
}

func TestRunClosureV2RecoveryAuthorityIsUnconditionalAndPrecedesObjectWrites(t *testing.T) {
	fixture := prepareV2Repository(t)
	failureGit := &v2FailingGit{failCommand: "update-ref"}
	deps := productionV2TestDependencies(fixture, failureGit, nil)
	if _, err := runProductionV2Test(fixture, deps); err == nil {
		t.Fatal("injected ref-publication failure was accepted")
	}
	if _, err := os.Lstat(v2EvidencePath(fixture)); err != nil {
		t.Fatalf("PREPARED evidence missing: %v", err)
	}

	recorder := &v2RecordingGit{delegate: RealGit{}}
	stale := productionV2TestDependencies(fixture, recorder, nil)
	stale.Runner = fixedRunnerIdentity{value: RunnerIdentity{
		VCSRevision: strings.Repeat("f", len(fixture.subject)), BinarySHA256: testV2BinaryHash,
	}}
	_, err := runProductionV2Test(fixture, stale)
	if err == nil || !strings.Contains(err.Error(), "runner") {
		t.Fatalf("err = %v, want stale runner rejection", err)
	}
	if len(recorder.objectWrites) != 0 {
		t.Fatalf("object-writing commands preceded authority rejection: %v", recorder.objectWrites)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "HEAD"); got != fixture.subject {
		t.Fatalf("HEAD changed to %s", got)
	}
}

func TestRunClosureV2RejectsCanonicalCollisionWithoutMutation(t *testing.T) {
	fixture := prepareV2Repository(t)
	deps := productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "reset"}, nil)
	if _, err := runProductionV2Test(fixture, deps); err == nil {
		t.Fatal("injected convergence failure was accepted")
	}

	collisionPath := filepath.Join(fixture.root, filepath.FromSlash(canonicalV2ManifestPath(v2OrchestratorActID)))
	if err := os.MkdirAll(filepath.Dir(collisionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	collision := []byte("user-owned collision\n")
	if err := os.WriteFile(collisionPath, collision, 0o644); err != nil {
		t.Fatal(err)
	}
	branchBefore := v2Git(t, fixture.root, "rev-parse", "refs/heads/main")

	_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
	if err == nil || !strings.Contains(err.Error(), "convergence") {
		t.Fatalf("err = %v, want bounded convergence rejection", err)
	}
	got, readErr := os.ReadFile(collisionPath)
	if readErr != nil || string(got) != string(collision) {
		t.Fatalf("collision bytes changed: got=%q err=%v", got, readErr)
	}
	if branchAfter := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); branchAfter != branchBefore {
		t.Fatalf("branch changed from %s to %s", branchBefore, branchAfter)
	}
}

func TestRunClosureV2ProductionLifecycle(t *testing.T) {
	fixture := prepareV2Repository(t)
	checks := 0
	deps := productionV2TestDependencies(fixture, RealGit{}, &checks)
	result, err := runProductionV2Test(fixture, deps)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteV2Result(t, fixture, result)
	if checks != 1 {
		t.Fatalf("NEW checks = %d, want 1", checks)
	}

	checks = 0
	again, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, &checks))
	if err != nil {
		t.Fatal(err)
	}
	assertSameV2Identities(t, result, again)
	if checks != 0 {
		t.Fatalf("VERIFIED checks = %d, want 0", checks)
	}
}

const testV2BinaryHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func productionV2TestDependencies(fixture v2RepositoryFixture, git gitClient, checks *int) v2Dependencies {
	return v2Dependencies{
		Git:                 git,
		Runner:              fixedRunnerIdentity{value: RunnerIdentity{VCSRevision: fixture.subject, BinarySHA256: testV2BinaryHash}},
		RunningBinarySHA256: func() (string, error) { return testV2BinaryHash, nil },
		VerifyExisting:      verifyExistingTransactionExact,
		Now:                 func() time.Time { return time.Unix(1, 0).UTC() },
		RunChecks: func(_ context.Context, _ Plan, _ v2Dependencies, _, evidenceDir, subjectTree string) ([]CheckResult, []EvidenceRecord, error) {
			if checks != nil {
				*checks++
			}
			stdout, err := writeDetachedOutput(evidenceDir, "authority-check.stdout", nil)
			if err != nil {
				return nil, nil, err
			}
			stderr, err := writeDetachedOutput(evidenceDir, "authority-check.stderr", nil)
			if err != nil {
				return nil, nil, err
			}
			exitCode := 0
			return []CheckResult{{
				CheckID: "authority-check", SubjectTreeOID: subjectTree, Argv: []string{"go", "version"},
				WorkingDirectory: ".", OverriddenEnvironment: []string{},
				ExitCode: &exitCode, Status: CheckStatusPass, CleanupStatus: CleanupPass,
				StdoutSHA256: stdout.SHA256, StdoutByteCount: stdout.ByteCount,
				StderrSHA256: stderr.SHA256, StderrByteCount: stderr.ByteCount,
			}}, []EvidenceRecord{stdout, stderr}, nil
		},
		EvaluatePatchPolicy: func(context.Context, gitClient, string, Plan, string, string) (policyEvaluation[PatchHygiene], error) {
			return policyEvaluation[PatchHygiene]{Passed: true, Value: PatchHygiene{Status: CheckStatusPass}}, nil
		},
		EvaluateClosurePolicy: func(context.Context, gitClient, string, Plan, string) (policyEvaluation[ClosurePolicyResult], error) {
			return policyEvaluation[ClosurePolicyResult]{Passed: true, Value: ClosurePolicyResult{TrackedFullDigestStatus: CheckStatusPass}}, nil
		},
		FinalizeNew:     defaultV2FinalizeNew,
		PublishEvidence: publishV2Evidence,
	}
}

func runProductionV2Test(fixture v2RepositoryFixture, deps v2Dependencies) (*TransactionResult, error) {
	return runClosureV2WithDependencies(context.Background(), RunV2Options{
		PlanPath: fixture.planPath, Subject: fixture.subject, RepoDirectory: fixture.root,
	}, deps)
}

func assertCompleteV2Result(t *testing.T, fixture v2RepositoryFixture, result *TransactionResult) {
	t.Helper()
	if result == nil || result.FreezeCommit != fixture.freeze || result.SubjectCommit != fixture.subject ||
		result.ClosureCommit == "" || result.ClosureTree == "" || result.TagObject == "" ||
		result.TagTarget != result.ClosureCommit || result.EvidenceHash == "" || result.Verdict != VerdictPass {
		t.Fatalf("incomplete transaction result: %+v", result)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != result.ClosureCommit {
		t.Fatalf("branch = %s, want C %s", got, result.ClosureCommit)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/tags/"+result.TagName+"^{tag}"); got != result.TagObject {
		t.Fatalf("tag = %s, want T %s", got, result.TagObject)
	}
}

func assertSameV2Identities(t *testing.T, first, second *TransactionResult) {
	t.Helper()
	if first.FreezeCommit != second.FreezeCommit || first.SubjectCommit != second.SubjectCommit ||
		first.ClosureCommit != second.ClosureCommit || first.ClosureTree != second.ClosureTree ||
		first.TagName != second.TagName || first.TagObject != second.TagObject ||
		first.EvidenceHash != second.EvidenceHash {
		t.Fatalf("transaction identities changed:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

type v2FailingGit struct {
	delegate    RealGit
	failCommand string
	failed      bool
}

func (g *v2FailingGit) failure(args []string) (gitCommandResult, bool) {
	if !g.failed && len(args) != 0 && args[0] == g.failCommand {
		g.failed = true
		return gitCommandResult{ExitCode: 1, Err: errors.New("injected failure"), Stderr: []byte("injected failure")}, true
	}
	return gitCommandResult{}, false
}

func (g *v2FailingGit) Run(ctx context.Context, dir string, args ...string) gitCommandResult {
	if result, fail := g.failure(args); fail {
		return result
	}
	return g.delegate.Run(ctx, dir, args...)
}
func (g *v2FailingGit) RunWithStdin(ctx context.Context, dir, stdin string, args ...string) gitCommandResult {
	if result, fail := g.failure(args); fail {
		return result
	}
	return g.delegate.RunWithStdin(ctx, dir, stdin, args...)
}
func (g *v2FailingGit) RunWithEnv(ctx context.Context, dir string, env []string, args ...string) gitCommandResult {
	if result, fail := g.failure(args); fail {
		return result
	}
	return g.delegate.RunWithEnv(ctx, dir, env, args...)
}
func (g *v2FailingGit) RunWithStdinAndEnv(ctx context.Context, dir, stdin string, env []string, args ...string) gitCommandResult {
	if result, fail := g.failure(args); fail {
		return result
	}
	return g.delegate.RunWithStdinAndEnv(ctx, dir, stdin, env, args...)
}

type v2RecordingGit struct {
	delegate     RealGit
	objectWrites []string
}

func (g *v2RecordingGit) record(args []string) {
	if len(args) == 0 {
		return
	}
	switch args[0] {
	case "hash-object", "commit-tree", "mktag", "write-tree", "update-index":
		g.objectWrites = append(g.objectWrites, args[0])
	}
}
func (g *v2RecordingGit) Run(ctx context.Context, dir string, args ...string) gitCommandResult {
	g.record(args)
	return g.delegate.Run(ctx, dir, args...)
}
func (g *v2RecordingGit) RunWithStdin(ctx context.Context, dir, stdin string, args ...string) gitCommandResult {
	g.record(args)
	return g.delegate.RunWithStdin(ctx, dir, stdin, args...)
}
func (g *v2RecordingGit) RunWithEnv(ctx context.Context, dir string, env []string, args ...string) gitCommandResult {
	g.record(args)
	return g.delegate.RunWithEnv(ctx, dir, env, args...)
}
func (g *v2RecordingGit) RunWithStdinAndEnv(ctx context.Context, dir, stdin string, env []string, args ...string) gitCommandResult {
	g.record(args)
	return g.delegate.RunWithStdinAndEnv(ctx, dir, stdin, env, args...)
}

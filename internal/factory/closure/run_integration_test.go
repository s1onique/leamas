package closure

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/s1onique/leamas/internal/execution"
)

type fixedRunnerIdentity struct{ value RunnerIdentity }

func (f fixedRunnerIdentity) Identity() (RunnerIdentity, error) { return f.value, nil }

func prepareRunnableRepository(t *testing.T) (string, string, string) {
	t.Helper()
	repository, baseline := newGitRepository(t)
	baselineTree, err := runGitValue(context.Background(), realGitClient{}, repository, "rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	plan := canonicalPlan()
	plan.Baseline = Baseline{CommitOID: baseline, TreeOID: baselineTree}
	plan.PolicyProfile = ""
	plan.RunnerBinding = ""
	plan.Checks = []PlanCheck{
		{ID: "first", Mode: CheckModeRun, Argv: []string{"go", "version"}, WorkingDirectory: ".", TimeoutSeconds: 30, Environment: map[string]string{}},
		{ID: "second", Mode: CheckModeRun, Argv: []string{"go", "env", "GOOS"}, WorkingDirectory: ".", TimeoutSeconds: 30, Environment: map[string]string{}},
	}
	plan.Artifacts = []PlanArtifact{}
	planPath := filepath.Join(repository, "docs", "closure-plans", plan.ActID+".json")
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(planPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "docs/closure-plans/" + plan.ActID + ".json"}, {"commit", "-m", "add plan"}} {
		if _, err := runGitValue(context.Background(), realGitClient{}, repository, args...); err != nil {
			t.Fatal(err)
		}
	}
	subject, err := runGitValue(context.Background(), realGitClient{}, repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return repository, planPath, subject
}

func runOptionsForTest(t *testing.T, repository, planPath, subject string) RunOptions {
	t.Helper()
	detached := t.TempDir()
	relative, err := filepath.Rel(repository, planPath)
	if err != nil {
		relative = planPath
	}
	return RunOptions{
		PlanPath:            planPath,
		Subject:             subject,
		EvidenceDirectory:   filepath.Join(detached, "evidence"),
		ManifestOutput:      filepath.Join(detached, "manifest.json"),
		RepositoryDirectory: repository,
		PlanFreeze:          subject + ":" + relative,
	}
}

func passingRunDependencies(subject string, executor commandExecutor) runDependencies {
	return runDependencies{
		Git:      realGitClient{},
		Commands: executor,
		Runner:   fixedRunnerIdentity{value: RunnerIdentity{LeamasVersion: "0.1.0", BinarySHA256: strings.Repeat("a", 64), VCSRevision: subject}},
		Now:      func() time.Time { return time.Date(2026, 7, 23, 7, 0, 0, 0, time.UTC) },
	}
}

func TestClosureRunProducesVerifiedManifest(t *testing.T) {
	repository, planPath, subject := prepareRunnableRepository(t)
	executor := &recordingExecutor{results: []*execution.Result{successExecution("one", ""), successExecution("two", "")}}
	manifest, data, err := runClosureWithDependencies(context.Background(), runOptionsForTest(t, repository, planPath, subject), passingRunDependencies(subject, executor))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Verdict != VerdictPass || len(data) == 0 || executor.calls != 2 {
		t.Fatalf("manifest=%+v data=%d calls=%d", manifest, len(data), executor.calls)
	}
}

func TestClosureRunRequiresCleanWorktreeAfterExecution(t *testing.T) {
	repository, planPath, subject := prepareRunnableRepository(t)
	executor := &dirtyingExecutor{repository: repository}
	manifest, _, err := runClosureWithDependencies(context.Background(), runOptionsForTest(t, repository, planPath, subject), passingRunDependencies(subject, executor))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Repository.WorkingTreeCleanAfter || manifest.Verdict != VerdictFail {
		t.Fatalf("manifest = %+v", manifest)
	}
}

type dirtyingExecutor struct {
	repository string
	calls      int
}

func (d *dirtyingExecutor) Execute(_ context.Context, _ *execution.Request) *execution.Result {
	d.calls++
	if d.calls == 1 {
		_ = os.WriteFile(filepath.Join(d.repository, "dirty-after.txt"), []byte("dirty"), 0o644)
	}
	return successExecution("", "")
}

var _ = strings.HasPrefix

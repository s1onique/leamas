package closure

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

type closureFixture struct {
	repository    string
	manifestPath  string
	reportPath    string
	closureCommit string
	tag           string
}

func prepareClosureFixture(t *testing.T) closureFixture {
	t.Helper()
	repo, freezeArg, subject, planPath := prepareFreezeAndSubject(t)
	executor := &recordingExecutor{results: []*execution.Result{successExecution("one", "")}}
	detached := t.TempDir()
	options := RunOptions{
		PlanPath:            planPath,
		Subject:             subject,
		EvidenceDirectory:   filepath.Join(detached, "evidence"),
		ManifestOutput:      filepath.Join(detached, "manifest.json"),
		RepositoryDirectory: repo,
		PlanFreeze:          freezeArg,
	}
	manifest, manifestBytes, err := runClosureWithDependencies(context.Background(), options, passingRunDependencies(subject, executor))
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := LoadPlan(planPath)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Render(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(repo, "docs", "closure-manifests", manifest.ActID+".json")
	reportPath := filepath.Join(repo, "docs", "close-reports", manifest.ActID+".md")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, report, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "docs/closure-manifests", "docs/close-reports"}, {"commit", "-m", "close act"}} {
		if _, err := runGitValue(context.Background(), RealGit{}, repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	closureCommit, err := runGitValue(context.Background(), RealGit{}, repo, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return closureFixture{repository: repo, manifestPath: manifestPath, reportPath: reportPath, closureCommit: closureCommit, tag: "act/leamas-test01"}
}

func (f closureFixture) options() TagOptions {
	return TagOptions{RepositoryDirectory: f.repository, ManifestPath: f.manifestPath, ReportPath: f.reportPath, TagName: f.tag, Target: "HEAD"}
}

func TestClosureTagCreatesAnnotatedTag(t *testing.T) {
	fixture := prepareClosureFixture(t)
	message, err := CreateTag(context.Background(), fixture.options())
	if err != nil {
		t.Fatal(err)
	}
	objectType, err := runGitValue(context.Background(), RealGit{}, fixture.repository, "cat-file", "-t", "refs/tags/"+fixture.tag)
	if err != nil || objectType != "tag" {
		t.Fatalf("type = %q, %v", objectType, err)
	}
	if !bytes.Contains(message, []byte("LEAMAS_CLOSURE_TAG_CONTRACT_VERSION: 1\n")) {
		t.Fatalf("message = %q", message)
	}
}

func TestClosureTagMessageExactBytes(t *testing.T) {
	fixture := prepareClosureFixture(t)
	want, err := CreateTag(context.Background(), fixture.options())
	if err != nil {
		t.Fatal(err)
	}
	raw := RealGit{}.Run(context.Background(), fixture.repository, "cat-file", "tag", "refs/tags/"+fixture.tag)
	if raw.Err != nil {
		t.Fatal(raw.Err)
	}
	_, got, found := bytes.Cut(raw.Stdout, []byte("\n\n"))
	if !found || !bytes.Equal(got, want) {
		t.Fatalf("tag message mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestClosureTagTargetsClosureCommit(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	target, err := runGitValue(context.Background(), RealGit{}, fixture.repository, "rev-parse", fixture.tag+"^{commit}")
	if err != nil || target != fixture.closureCommit {
		t.Fatalf("target = %q, %v", target, err)
	}
}

func TestClosureTagRejectsDirtyWorktree(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if err := os.WriteFile(filepath.Join(fixture.repository, "dirty"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTag(context.Background(), fixture.options()); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagRejectsExistingTag(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTag(context.Background(), fixture.options()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagRejectsWrongTarget(t *testing.T) {
	fixture := prepareClosureFixture(t)
	options := fixture.options()
	options.Target = "HEAD^"
	if _, err := CreateTag(context.Background(), options); err == nil || !strings.Contains(err.Error(), "HEAD") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureTagHasNoForcePath(t *testing.T) {
	fixture := prepareClosureFixture(t)
	options := fixture.options()
	if strings.Contains(strings.ToLower(strings.Join([]string{options.TagName, options.Target}, " ")), "force") {
		t.Fatal("TagOptions unexpectedly exposes a force path")
	}
}

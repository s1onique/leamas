package closure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (f closureFixture) statusOptions() StatusOptions {
	return StatusOptions{RepositoryDirectory: f.repository, ManifestPath: f.manifestPath, ReportPath: f.reportPath, TagName: f.tag}
}

func TestClosureStatusImplemented(t *testing.T) {
	fixture := prepareClosureFixture(t)
	manifest, _, err := LoadManifest(fixture.manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	plan, _, err := LoadPlan(filepath.Join(fixture.repository, filepath.FromSlash(manifest.Plan.Path)))
	if err != nil {
		t.Fatal(err)
	}
	manifest.Runner.VCSModified = true
	manifest.Verdict = VerdictFail
	manifestBytes, err := MarshalManifest(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Render(manifest, plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.reportPath, report, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "docs/closure-manifests", "docs/close-reports"}, {"commit", "-m", "record failed closure"}} {
		if _, err := runGitValue(context.Background(), RealGit{}, fixture.repository, args...); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Status(context.Background(), fixture.statusOptions())
	if err != nil || result.State != LifecycleImplemented {
		t.Fatalf("status = %+v, %v", result, err)
	}
}

func TestClosureStatusVerified(t *testing.T) {
	fixture := prepareClosureFixture(t)
	result, err := Status(context.Background(), fixture.statusOptions())
	if err != nil || result.State != LifecycleVerified {
		t.Fatalf("status = %+v, %v", result, err)
	}
}

func TestClosureStatusClosedLocal(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	result, err := Status(context.Background(), fixture.statusOptions())
	if err != nil || result.State != LifecycleClosedLocal {
		t.Fatalf("status = %+v, %v", result, err)
	}
}

func TestClosureStatusPublished(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(t.TempDir(), "remote.git")
	if _, err := runGitValue(context.Background(), RealGit{}, ".", "init", "--bare", remote); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"remote", "add", "origin", remote},
		{"push", "origin", "main"},
		{"push", "origin", "refs/tags/" + fixture.tag},
	} {
		if _, err := runGitValue(context.Background(), RealGit{}, fixture.repository, args...); err != nil {
			t.Fatal(err)
		}
	}
	options := fixture.statusOptions()
	options.Remote = "origin"
	result, err := Status(context.Background(), options)
	if err != nil || result.State != LifecyclePublished {
		t.Fatalf("status = %+v, %v", result, err)
	}
}

func TestClosureStatusRejectsMovedLocalTag(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	if _, err := runGitValue(context.Background(), RealGit{}, fixture.repository, "tag", "--force", fixture.tag, "HEAD^"); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), fixture.statusOptions()); err == nil {
		t.Fatal("Status() accepted moved local tag")
	}
}

func TestClosureStatusRejectsRemoteTagObjectMismatch(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	client := &remoteMutationGitClient{delegate: RealGit{}, mutateDirect: true}
	options := fixture.statusOptions()
	options.Remote = "origin"
	if _, err := statusWithGit(context.Background(), options, client); err == nil || !strings.Contains(err.Error(), "tag object") {
		t.Fatalf("error = %v", err)
	}
}

func TestClosureStatusRejectsRemotePeeledTargetMismatch(t *testing.T) {
	fixture := prepareClosureFixture(t)
	if _, err := CreateTag(context.Background(), fixture.options()); err != nil {
		t.Fatal(err)
	}
	client := &remoteMutationGitClient{delegate: RealGit{}, mutatePeeled: true}
	options := fixture.statusOptions()
	options.Remote = "origin"
	if _, err := statusWithGit(context.Background(), options, client); err == nil || !strings.Contains(err.Error(), "peeled") {
		t.Fatalf("error = %v", err)
	}
}

type remoteMutationGitClient struct {
	delegate     gitClient
	mutateDirect bool
	mutatePeeled bool
}

func (c *remoteMutationGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	if len(args) > 0 && args[0] == "ls-remote" {
		tagName := strings.TrimPrefix(args[len(args)-2], "refs/tags/")
		localTag := c.delegate.Run(ctx, directory, "rev-parse", "refs/tags/"+tagName)
		closure := c.delegate.Run(ctx, directory, "rev-parse", "HEAD")
		branch := c.delegate.Run(ctx, directory, "symbolic-ref", "--short", "HEAD")
		direct := strings.TrimSpace(string(localTag.Stdout))
		peeled := strings.TrimSpace(string(closure.Stdout))
		if c.mutateDirect {
			direct = strings.Repeat("a", 40)
		}
		if c.mutatePeeled {
			peeled = strings.Repeat("b", 40)
		}
		branchName := strings.TrimSpace(string(branch.Stdout))
		output := strings.TrimSpace(string(closure.Stdout)) + "\trefs/heads/" + branchName + "\n" +
			direct + "\trefs/tags/" + tagName + "\n" +
			peeled + "\trefs/tags/" + tagName + "^{}\n"
		return gitCommandResult{Stdout: []byte(output)}
	}
	return c.delegate.Run(ctx, directory, args...)
}

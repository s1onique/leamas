// SPDX-License-Identifier: Apache-2.0

// SPDX-License-Identifier: Apache-2.0

package gate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/dupcode"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// initTempGitRepoForDupcodeUpdate creates a minimal committed temp
// git repository that DetectExecutionContext can observe. It exists so
// mutation-path tests can drive DetectExecutionContext through the
// production observation path (no synthetic local context). The
// helper uses execution.RunGit because the executable-contract-first
// verifier forbids direct os/exec use outside internal/execution.
func initTempGitRepoForDupcodeUpdate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.email", "dupcode-update@test.local"},
		{"config", "user.name", "Dupcode Update Test"},
	} {
		out, err := execution.RunGit(ctx, dir, args...)
		if err != nil || out.ExitCode != 0 {
			t.Fatalf("git %v: %v\n%s", args, err, out.Stderr)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	for _, args := range [][]string{
		{"add", "."},
		{"commit", "-m", "seed"},
	} {
		out, err := execution.RunGit(ctx, dir, args...)
		if err != nil || out.ExitCode != 0 {
			t.Fatalf("git %v: %v\n%s", args, err, out.Stderr)
		}
	}
	return dir
}

// TestDupcodeUpdateExplicitLocalAdmittedExactlyOnce proves that an
// explicitly classified local context admits the mutation and each
// protected operation runs exactly once. The observer drives
// DetectExecutionContext against a real temp git repository so the
// local classification is recorded through the production observation
// path.
func TestDupcodeUpdateExplicitLocalAdmittedExactlyOnce(t *testing.T) {
	dir := initTempGitRepoForDupcodeUpdate(t)
	baselinePath := dir + "/.factory/dupcode-baseline.json"
	reportWritten := dupcode.Report{
		Root: dir,
		Findings: []dupcode.Finding{
			{Fingerprint: "local-fp", TokenCount: 100, LineCount: 25, Occurrences: []dupcode.Occurrence{{Path: "u.go", StartLine: 1, EndLine: 25}}},
		},
		Thresholds: dupcode.BaselineThresholds{MinLines: 40, MinTokens: 400},
	}
	r := &countingDupcodeRunner{reportToReturn: reportWritten}
	outcome := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), dir, DupcodeUpdateBaselineSpec{
			BaselinePath: baselinePath, MinLines: 40, MinTokens: 400,
		}, newTempRepoObserver(dir), makeUpdateDeps(r),
	)
	if outcome.Dispatch.Error != nil {
		t.Fatalf("expected admission: error=%v", outcome.Dispatch.Error)
	}
	if len(outcome.Dispatch.Findings) != 0 {
		t.Errorf("expected empty Dispatch.Findings on success, got %d", len(outcome.Dispatch.Findings))
	}
	if got := r.newRunnerCalls.Load(); got != 1 {
		t.Errorf("newRunnerCalls = %d, want 1", got)
	}
	if got := r.scanCalls.Load(); got != 1 {
		t.Errorf("scanCalls = %d, want 1", got)
	}
	if got := r.writeCalls.Load(); got != 1 {
		t.Errorf("writeCalls = %d, want 1 (the binder must call WriteBaseline exactly once on admission)", got)
	}
	if outcome.Report.Root != dir {
		t.Errorf("outcome.Report.Root = %q, want %q", outcome.Report.Root, dir)
	}
}

// githubActionsCIContextObserver returns a context where GITHUB_ACTIONS=true
// AND CI=true (the "fully valid CI" case the prior contract handled).
type githubActionsCIContextObserver struct{}

func (g *githubActionsCIContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubActions:   "true",
		AuthorityMarker: verifierauthority.AuthorityMarker,
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// githubActionsCIFalseObserver returns GITHUB_ACTIONS=true with CI=false.
type githubActionsCIFalseObserver struct{}

func (g *githubActionsCIFalseObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "false",
		GitHubActions:   "true",
		AuthorityMarker: verifierauthority.AuthorityMarker,
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// githubActionsNoAuthorityMarkerObserver returns GITHUB_ACTIONS=true but no
// authority marker.
type githubActionsNoAuthorityMarkerObserver struct{}

func (g *githubActionsNoAuthorityMarkerObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		GitHubActions:   "true",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// genericCIObserver returns CI=true without GITHUB_ACTIONS.
type genericCIObserver struct{}

func (g *genericCIObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "true",
		GitHubSHA:       "abc123def456abc123def456abc123def456abcd",
		GitHubWorkspace: root,
		HeadCommit:      "abc123def456abc123def456abc123def456abcd",
		WorktreeStatus:  "",
		RepositoryRoot:  root,
		WorkspaceRoot:   root,
	}
}

// partialGitHubContextObserver returns GITHUB_ACTIONS=true but with a
// partial context (no SHA, no workspace, no head commit).
type partialGitHubContextObserver struct{}

func (p *partialGitHubContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		GitHubActions:  "true",
		RepositoryRoot: root,
	}
}

// contradictoryContextObserver returns a contradictory context
// (CI=false AND GITHUB_ACTIONS=false BUT authority marker set).
type contradictoryContextObserver struct{}

func (c *contradictoryContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		CI:              "false",
		GitHubActions:   "false",
		AuthorityMarker: "some-marker",
		RepositoryRoot:  root,
	}
}

// unknownContextObserver returns a context with random unrelated fields.
type unknownContextObserver struct{}

func (u *unknownContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{
		RepositoryRoot: root,
		HeadCommit:     "deadbeef",
	}
}

// emptyUnclassifiedContextObserver returns a fully empty context. The
// environment is classified as EnvironmentUnknown, NOT EnvironmentLocal.
type emptyUnclassifiedContextObserver struct{}

func (e *emptyUnclassifiedContextObserver) Observe(ctx context.Context, root string) verifierauthority.ExecutionContext {
	return verifierauthority.ExecutionContext{}
}

// assertDenied proves the typed dispatch denied the update, recorded
// no protected work, and emitted a denial finding of the expected kind.
func assertDenied(t *testing.T, outcome DupcodeUpdateBaselineOutcome, r *countingDupcodeRunner) {
	t.Helper()
	if outcome.Dispatch.Error == nil {
		t.Fatalf("expected denial, got no error; findings=%v", outcome.Dispatch.Findings)
	}
	if len(outcome.Dispatch.Findings) != 1 {
		t.Fatalf("expected exactly one denial finding, got %d", len(outcome.Dispatch.Findings))
	}
	if outcome.Dispatch.Findings[0].Kind != "verifier_execution_authority_denied" {
		t.Errorf("finding kind = %q, want %q", outcome.Dispatch.Findings[0].Kind, "verifier_execution_authority_denied")
	}
	if got := r.newRunnerCalls.Load(); got != 0 {
		t.Errorf("newRunnerCalls = %d, want 0 (denial before factory)", got)
	}
	if got := r.scanCalls.Load(); got != 0 {
		t.Errorf("scanCalls = %d, want 0 (denial before scan)", got)
	}
	if got := r.writeCalls.Load(); got != 0 {
		t.Errorf("writeCalls = %d, want 0 (denial before write)", got)
	}
}

// TestDupcodeUpdateGitHubActionsDeniedWhenCITrue proves a fully valid
// GitHub Actions context denies the mutation.
func TestDupcodeUpdateGitHubActionsDeniedWhenCITrue(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &githubActionsCIContextObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateGitHubActionsDeniedWhenCIFalse proves the GitHub
// Actions classification holds even when CI is overwritten to "false".
func TestDupcodeUpdateGitHubActionsDeniedWhenCIFalse(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &githubActionsCIFalseObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateGitHubActionsDeniedWithoutAuthorityMarker proves the
// GitHub Actions classification holds even when the authority marker is
// missing.
func TestDupcodeUpdateGitHubActionsDeniedWithoutAuthorityMarker(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &githubActionsNoAuthorityMarkerObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateOtherCIDenied proves a generic CI context denies the mutation.
func TestDupcodeUpdateOtherCIDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &genericCIObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdatePartialGitHubContextDenied proves a partially
// populated GitHub Actions context denies the mutation.
func TestDupcodeUpdatePartialGitHubContextDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &partialGitHubContextObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateContradictoryContextDenied proves a contradictory
// context denies the mutation.
func TestDupcodeUpdateContradictoryContextDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &contradictoryContextObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateUnknownContextDenied proves an unclassified context
// denies the mutation.
func TestDupcodeUpdateUnknownContextDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &unknownContextObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestDupcodeUpdateEmptyUnclassifiedContextDenied proves a fully empty
// context denies the mutation. The empty context cannot be silently
// promoted to EnvironmentLocal.
func TestDupcodeUpdateEmptyUnclassifiedContextDenied(t *testing.T) {
	r := &countingDupcodeRunner{}
	out := dispatchDupcodeUpdateBaselineTypedWith(
		context.Background(), ".", DupcodeUpdateBaselineSpec{
			BaselinePath: ".factory/dupcode-baseline.json", MinLines: 40, MinTokens: 400,
		}, &emptyUnclassifiedContextObserver{}, makeUpdateDeps(r),
	)
	assertDenied(t, out, r)
}

// TestClassifyExecutionEnvironmentIsFailClosed is a direct classification
// test that does not require a full dispatch roundtrip.
func TestClassifyExecutionEnvironmentIsFailClosed(t *testing.T) {
	tests := []struct {
		name string
		ec   verifierauthority.ExecutionContext
		want verifierauthority.ExecutionEnvironmentKind
	}{
		{
			name: "github_actions_with_ci_true",
			ec: verifierauthority.ExecutionContext{
				GitHubActions: "true",
				CI:            "true",
			},
			want: verifierauthority.EnvironmentGitHubActions,
		},
		{
			name: "github_actions_with_ci_false",
			ec: verifierauthority.ExecutionContext{
				GitHubActions: "true",
				CI:            "false",
			},
			want: verifierauthority.EnvironmentGitHubActions,
		},
		{
			name: "github_actions_without_marker",
			ec: verifierauthority.ExecutionContext{
				GitHubActions: "true",
			},
			want: verifierauthority.EnvironmentGitHubActions,
		},
		{
			name: "ci_only",
			ec: verifierauthority.ExecutionContext{
				CI: "true",
			},
			want: verifierauthority.EnvironmentCI,
		},
		{
			name: "empty_context",
			ec:   verifierauthority.ExecutionContext{},
			want: verifierauthority.EnvironmentUnknown,
		},
		{
			name: "contradictory",
			ec: verifierauthority.ExecutionContext{
				CI:              "false",
				GitHubActions:   "false",
				AuthorityMarker: "marker",
			},
			want: verifierauthority.EnvironmentUnknown,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := verifierauthority.ClassifyExecutionEnvironment(tc.ec)
			if got != tc.want {
				t.Errorf("ClassifyExecutionEnvironment(%+v) = %q, want %q", tc.ec, got, tc.want)
			}
		})
	}
}

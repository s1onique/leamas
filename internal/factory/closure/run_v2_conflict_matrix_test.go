// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunClosureV2PublicationConflictMatrixChangesNeitherTransactionRef(t *testing.T) {
	t.Run("branch CAS", func(t *testing.T) {
		fixture := prepareV2RecoveryState(t, "PREPARED")
		tree := v2Git(t, fixture.root, "rev-parse", fixture.subject+"^{tree}")
		moved := oidGit(t, fixture.root, "commit-tree", tree, "-p", fixture.subject, "-m", "concurrent branch move")
		git := &v2BranchConflictGit{repo: fixture.root, subject: fixture.subject, moved: moved}

		_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, git, nil))
		if err == nil || !strings.Contains(err.Error(), "publish refs") {
			t.Fatalf("err = %v, want branch CAS publication rejection", err)
		}
		if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != moved {
			t.Fatalf("branch = %s, want concurrent value %s", got, moved)
		}
		assertV2TagAbsent(t, fixture)
	})

	tagConflicts := []struct {
		name   string
		create func(*testing.T, v2RepositoryFixture, string)
	}{
		{name: "annotated tag", create: func(t *testing.T, f v2RepositoryFixture, name string) {
			v2Git(t, f.root, "tag", "-a", name, "-m", "conflicting annotation", f.subject)
		}},
		{name: "lightweight tag", create: func(t *testing.T, f v2RepositoryFixture, name string) {
			v2Git(t, f.root, "tag", name, f.subject)
		}},
	}
	for _, conflict := range tagConflicts {
		t.Run(conflict.name, func(t *testing.T) {
			fixture := prepareV2RecoveryState(t, "PREPARED")
			name := canonicalV2TagName(v2OrchestratorActID)
			conflict.create(t, fixture, name)
			tagBefore := v2Git(t, fixture.root, "rev-parse", "refs/tags/"+name)

			_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
			if err == nil {
				t.Fatal("tag conflict was accepted")
			}
			if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != fixture.subject {
				t.Fatalf("branch partially changed to %s", got)
			}
			if got := v2Git(t, fixture.root, "rev-parse", "refs/tags/"+name); got != tagBefore {
				t.Fatalf("conflicting tag changed from %s to %s", tagBefore, got)
			}
		})
	}

	t.Run("reference-transaction hook", func(t *testing.T) {
		fixture := prepareV2RecoveryState(t, "PREPARED")
		hook := filepath.Join(fixture.root, ".git", "hooks", "reference-transaction")
		contents := "#!/bin/sh\nif [ \"$1\" = prepared ]; then\n  exit 1\nfi\nexit 0\n"
		if err := os.WriteFile(hook, []byte(contents), 0o755); err != nil {
			t.Fatal(err)
		}

		_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
		if err == nil || !strings.Contains(err.Error(), "publish refs") {
			t.Fatalf("err = %v, want hook publication rejection", err)
		}
		if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != fixture.subject {
			t.Fatalf("branch partially changed to %s", got)
		}
		assertV2TagAbsent(t, fixture)
	})
}

func assertV2TagAbsent(t *testing.T, fixture v2RepositoryFixture) {
	t.Helper()
	result := RealGit{}.Run(t.Context(), fixture.root, "show-ref", "--verify", "--quiet",
		"refs/tags/"+canonicalV2TagName(v2OrchestratorActID))
	if result.ExitCode == 0 {
		t.Fatal("closure tag exists")
	}
}

type v2BranchConflictGit struct {
	RealGit
	repo, subject, moved string
	injected             bool
}

func (g *v2BranchConflictGit) RunWithStdin(ctx context.Context, dir, stdin string, args ...string) gitCommandResult {
	if !g.injected && len(args) != 0 && args[0] == "update-ref" {
		g.injected = true
		result := g.RealGit.Run(ctx, g.repo, "update-ref", "refs/heads/main", g.moved, g.subject)
		if result.Err != nil || result.ExitCode != 0 {
			return gitCommandResult{ExitCode: 1, Err: errors.New("inject branch conflict"), Stderr: result.Stderr}
		}
	}
	return g.RealGit.RunWithStdin(ctx, dir, stdin, args...)
}

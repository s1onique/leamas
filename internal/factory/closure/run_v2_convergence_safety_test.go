// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClassifyV2TransactionStateUsesCapturedEvidenceWithoutFilesystemAccess(t *testing.T) {
	fixture := prepareV2Repository(t)
	expected := v2ExpectedTransaction{Tag: v2TagObject{
		Name: canonicalV2TagName(v2OrchestratorActID), ActID: v2OrchestratorActID,
	}}
	state, err := classifyV2TransactionState(
		t.Context(), RealGit{}, fixture.root, fixture.subject, fixture.subject, "main",
		expected, v2EvidenceSnapshot{Present: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if state != v2StatePrepared {
		t.Fatalf("state = %v, want PREPARED from supplied snapshot", state)
	}
	assertPathAbsent(t, v2EvidencePath(fixture))
}

func TestRunClosureV2BoundedConvergenceRejectsAndPreservesEveryOtherState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, v2RepositoryFixture)
	}{
		{name: "unrelated tracked modification", mutate: func(t *testing.T, f v2RepositoryFixture) {
			writeV2Collision(t, filepath.Join(f.root, "README.md"), "tracked modification\n")
		}},
		{name: "unrelated staged modification", mutate: func(t *testing.T, f v2RepositoryFixture) {
			writeV2Collision(t, filepath.Join(f.root, "README.md"), "staged modification\n")
			v2Git(t, f.root, "add", "README.md")
		}},
		{name: "unrelated untracked file", mutate: func(t *testing.T, f v2RepositoryFixture) {
			writeV2Collision(t, filepath.Join(f.root, "local-user.txt"), "untracked\n")
		}},
		{name: "canonical untracked collision", mutate: func(t *testing.T, f v2RepositoryFixture) {
			writeV2Collision(t, v2CanonicalWorktreePath(f, canonicalV2ManifestPath(v2OrchestratorActID)), "user manifest\n")
		}},
		{name: "canonical staged collision", mutate: func(t *testing.T, f v2RepositoryFixture) {
			path := canonicalV2ManifestPath(v2OrchestratorActID)
			writeV2Collision(t, v2CanonicalWorktreePath(f, path), "staged user manifest\n")
			v2Git(t, f.root, "add", path)
		}},
		{name: "canonical worktree modification", mutate: func(t *testing.T, f v2RepositoryFixture) {
			path := canonicalV2ManifestPath(v2OrchestratorActID)
			canonical := rawGitStdout(t, f.root, "show", "HEAD:"+path)
			writeV2Collision(t, v2CanonicalWorktreePath(f, path), string(canonical))
			v2Git(t, f.root, "add", path)
			writeV2Collision(t, v2CanonicalWorktreePath(f, path), "modified after canonical index\n")
		}},
		{name: "rename", mutate: func(t *testing.T, f v2RepositoryFixture) {
			v2Git(t, f.root, "mv", "README.md", "MOVED.md")
		}},
		{name: "third staged deletion", mutate: func(t *testing.T, f v2RepositoryFixture) {
			v2Git(t, f.root, "rm", "README.md")
		}},
		{name: "type change", mutate: func(t *testing.T, f v2RepositoryFixture) {
			path := filepath.Join(f.root, "README.md")
			if err := os.Chmod(path, 0o755); err != nil {
				t.Fatal(err)
			}
			v2Git(t, f.root, "add", "README.md")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := prepareV2RecoveryState(t, "POST_REF")
			test.mutate(t, fixture)
			before := captureV2StableState(t, fixture)
			_, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
			if err == nil || !strings.Contains(err.Error(), "bounded convergence") {
				t.Fatalf("err = %v, want bounded convergence rejection", err)
			}
			after := captureV2StableState(t, fixture)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("rejected state was mutated:\nbefore=%#v\nafter=%#v", before, after)
			}
		})
	}
}

func writeV2Collision(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func v2CanonicalWorktreePath(fixture v2RepositoryFixture, path string) string {
	return filepath.Join(fixture.root, filepath.FromSlash(path))
}

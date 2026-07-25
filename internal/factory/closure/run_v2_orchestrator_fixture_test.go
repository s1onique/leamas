// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s1onique/leamas/internal/execution"
)

const v2OrchestratorActID = "ACT-LEAMAS-V2-ORCHESTRATOR01"

type v2RepositoryFixture struct {
	root         string
	planPath     string
	baseline     string
	freeze       string
	freezeParent string // parent(F) — populated by prepareV2MultiFreezeRepository
	subject      string
}

func prepareV2Repository(t *testing.T) v2RepositoryFixture {
	t.Helper()
	fixture := initializeV2Repository(t)
	writeV2Plan(t, fixture)
	v2Git(t, fixture.root, "add", "docs/closure-plans")
	v2Git(t, fixture.root, "commit", "-m", "freeze plan")
	fixture.freeze = v2Git(t, fixture.root, "rev-parse", "HEAD")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "subject")
	fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")
	return fixture
}

// prepareV2MultiFreezeRepository builds a fixture whose history has
// FOUR commits instead of the usual three, matching the
// "B → F1 → F2 → S" topology that the digest asks the topology test
// to pin. Here:
//
//	baseline      (B) = fixture.baseline         (plan.baseline)
//	prerequisite  (P) = fixture.freezeParent     (parent of F)
//	freeze plan   (F) = fixture.freeze           (introduces plan blob)
//	subject       (S) = fixture.subject          (allow-empty)
//
// This lets us prove the critical property:
//
//	plan.baseline (B) != parent(F)  (P)
//
// which is the property the B↔P naming confusion silently violated.
// See ProvenanceTopology in run_v2_authority.go.
func prepareV2MultiFreezeRepository(t *testing.T) v2RepositoryFixture {
	t.Helper()
	fixture := initializeV2Repository(t)
	// P (prerequisite) — an empty commit strictly between B and F so
	// parent(F) is NOT equal to plan.baseline.
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "prerequisite")
	fixture.freezeParent = v2Git(t, fixture.root, "rev-parse", "HEAD")
	writeV2Plan(t, fixture)
	v2Git(t, fixture.root, "add", "docs/closure-plans")
	v2Git(t, fixture.root, "commit", "-m", "freeze plan")
	fixture.freeze = v2Git(t, fixture.root, "rev-parse", "HEAD")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "subject")
	fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")
	return fixture
}

func prepareV2MergeFreezeRepository(t *testing.T) v2RepositoryFixture {
	t.Helper()
	fixture := initializeV2Repository(t)
	v2Git(t, fixture.root, "branch", "side")
	v2Git(t, fixture.root, "checkout", "side")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "side")
	v2Git(t, fixture.root, "checkout", "main")
	v2Git(t, fixture.root, "merge", "--no-ff", "--no-commit", "side")
	writeV2Plan(t, fixture)
	v2Git(t, fixture.root, "add", "docs/closure-plans")
	v2Git(t, fixture.root, "commit", "-m", "merge freeze")
	fixture.freeze = v2Git(t, fixture.root, "rev-parse", "HEAD")
	v2Git(t, fixture.root, "commit", "--allow-empty", "-m", "subject")
	fixture.subject = v2Git(t, fixture.root, "rev-parse", "HEAD")
	return fixture
}

func initializeV2Repository(t *testing.T) v2RepositoryFixture {
	t.Helper()
	return initializeV2RepositoryWithFormat(t, ObjectFormatSHA1)
}

func initializeV2RepositoryWithFormat(t *testing.T, objectFormat ObjectFormat) v2RepositoryFixture {
	t.Helper()
	root := t.TempDir()
	args := []string{"init", "-b", "main"}
	if objectFormat == ObjectFormatSHA256 {
		args = append(args, "--object-format=sha256")
	}
	v2Git(t, root, args...)
	v2Git(t, root, "config", "user.name", "V2 Test")
	v2Git(t, root, "config", "user.email", "v2@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("baseline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".factory/closure-evidence/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v2Git(t, root, "add", "README.md", ".gitignore")
	v2Git(t, root, "commit", "-m", "baseline")
	return v2RepositoryFixture{
		root:     root,
		planPath: filepath.Join(root, "docs", "closure-plans", v2OrchestratorActID+".json"),
		baseline: v2Git(t, root, "rev-parse", "HEAD"),
	}
}

func writeV2Plan(t *testing.T, fixture v2RepositoryFixture) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(fixture.planPath), 0o755); err != nil {
		t.Fatal(err)
	}
	baselineTree := v2Git(t, fixture.root, "rev-parse", fixture.baseline+"^{tree}")
	plan := fmt.Sprintf(`{
  "contract_version": 1,
  "act_id": %q,
  "baseline": {"commit_oid": %q, "tree_oid": %q},
  "execution": {"mode": "serial_fail_fast"},
  "checks": [{"id":"authority-check","mode":"run","argv":["go","version"],"working_directory":".","timeout_seconds":60,"environment":{}}],
  "artifacts": [],
  "policy": {"require_clean_before":true,"require_clean_after":true,"forbid_tracked_full_digests":true,"require_diff_check":true}
}
`, v2OrchestratorActID, fixture.baseline, baselineTree)
	if err := os.WriteFile(fixture.planPath, []byte(plan), 0o644); err != nil {
		t.Fatal(err)
	}
}

func v2Git(t *testing.T, directory string, args ...string) string {
	t.Helper()
	result, err := execution.RunGit(context.Background(), directory, args...)
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("git %v: %v (exit %d)\nstdout=%s\nstderr=%s", args, err, result.ExitCode, result.Stdout, result.Stderr)
	}
	return strings.TrimSpace(string(result.Stdout))
}

func v2EvidencePath(fixture v2RepositoryFixture) string {
	return evidenceDirectoryPath(fixture.root, v2OrchestratorActID, fixture.subject)
}

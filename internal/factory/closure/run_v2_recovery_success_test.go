// SPDX-License-Identifier: Apache-2.0

package closure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type v2BuiltIdentities struct {
	closure  string
	tag      string
	evidence string
}

func TestRunClosureV2NEWPublishesOneQualifiedTransaction(t *testing.T) {
	fixture := prepareV2Repository(t)
	checks := 0
	deps := productionV2TestDependencies(fixture, RealGit{}, &checks)
	built := captureV2BuiltIdentities(&deps)
	publishes := 0
	publish := deps.PublishEvidence
	deps.PublishEvidence = func(staging, final string, evidence v2QualifiedEvidence) (v2QualifiedEvidence, error) {
		publishes++
		return publish(staging, final, evidence)
	}
	result, err := runProductionV2Test(fixture, deps)
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteV2Result(t, fixture, result)
	assertBuiltV2Identities(t, built, result)
	if checks != 1 || publishes != 1 {
		t.Fatalf("checks = %d, publishes = %d; want 1 each", checks, publishes)
	}
	assertSinglePublishedEvidenceDirectory(t, fixture)
	published, err := readQualifiedV2Evidence(v2EvidencePath(fixture), v2OrchestratorActID, fixture.subject)
	if err != nil || published.IndexHash != result.EvidenceHash {
		t.Fatalf("published E does not match result: hash=%s err=%v", published.IndexHash, err)
	}
	tagBytes := rawGitStdout(t, fixture.root, "cat-file", "tag", result.TagObject)
	if !strings.Contains(string(tagBytes), "EVIDENCE_INDEX_SHA256 "+result.EvidenceHash+"\n") {
		t.Fatalf("T does not bind published E %s", result.EvidenceHash)
	}
}

func TestRunClosureV2PreparedRecovery(t *testing.T) {
	fixture := prepareV2Repository(t)
	checks := 0
	deps := productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "update-ref"}, &checks)
	built := captureV2BuiltIdentities(&deps)
	if _, err := runProductionV2Test(fixture, deps); err == nil {
		t.Fatal("injected publication interruption was accepted")
	}
	if checks != 1 {
		t.Fatalf("initial checks = %d, want 1", checks)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "HEAD"); got != fixture.subject {
		t.Fatalf("PREPARED HEAD = %s, want S", got)
	}

	checks = 0
	result, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, &checks))
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteV2Result(t, fixture, result)
	assertBuiltV2Identities(t, built, result)
	if checks != 0 {
		t.Fatalf("recovery checks = %d, want 0", checks)
	}
	assertSinglePublishedEvidenceDirectory(t, fixture)
}

func TestRunClosureV2PostRefRecovery(t *testing.T) {
	fixture := prepareV2Repository(t)
	checks := 0
	deps := productionV2TestDependencies(fixture, &v2FailingGit{failCommand: "reset"}, &checks)
	built := captureV2BuiltIdentities(&deps)
	if _, err := runProductionV2Test(fixture, deps); err == nil {
		t.Fatal("injected convergence interruption was accepted")
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/heads/main"); got != built.closure {
		t.Fatalf("post-ref branch = %s, want C %s", got, built.closure)
	}
	if got := v2Git(t, fixture.root, "rev-parse", "refs/tags/"+canonicalV2TagName(v2OrchestratorActID)+"^{tag}"); got != built.tag {
		t.Fatalf("post-ref tag = %s, want T %s", got, built.tag)
	}

	checks = 0
	result, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, &checks))
	if err != nil {
		t.Fatal(err)
	}
	assertCompleteV2Result(t, fixture, result)
	assertBuiltV2Identities(t, built, result)
	if checks != 0 {
		t.Fatalf("recovery checks = %d, want 0", checks)
	}
}

func TestRunClosureV2VerifiedRerunHasNoMutation(t *testing.T) {
	fixture := prepareV2Repository(t)
	first, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, nil))
	if err != nil {
		t.Fatal(err)
	}
	before := captureV2StableState(t, fixture)
	checks := 0
	second, err := runProductionV2Test(fixture, productionV2TestDependencies(fixture, RealGit{}, &checks))
	if err != nil {
		t.Fatal(err)
	}
	after := captureV2StableState(t, fixture)
	assertSameV2Identities(t, first, second)
	if checks != 0 {
		t.Fatalf("VERIFIED checks = %d, want 0", checks)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("VERIFIED rerun mutated repository:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func captureV2BuiltIdentities(deps *v2Dependencies) *v2BuiltIdentities {
	built := &v2BuiltIdentities{}
	finalize := deps.FinalizeNew
	deps.FinalizeNew = func(ctx context.Context, input v2FinalizeInput) (*TransactionResult, error) {
		result, err := finalize(ctx, input)
		if incomplete, ok := err.(*PublicationIncompleteError); ok {
			built.closure = incomplete.ClosureCommit
			built.tag = incomplete.TagObject
			built.evidence = incomplete.EvidenceHash
		}
		return result, err
	}
	return built
}

func assertBuiltV2Identities(t *testing.T, built *v2BuiltIdentities, result *TransactionResult) {
	t.Helper()
	if built.closure == "" || built.tag == "" || built.evidence == "" ||
		built.closure != result.ClosureCommit || built.tag != result.TagObject || built.evidence != result.EvidenceHash {
		t.Fatalf("built identities %+v do not match result %+v", built, result)
	}
}

func assertSinglePublishedEvidenceDirectory(t *testing.T, fixture v2RepositoryFixture) {
	t.Helper()
	parent := filepath.Dir(v2EvidencePath(fixture))
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != fixture.subject || !entries[0].IsDir() {
		t.Fatalf("evidence publication entries = %v, want one final S directory", entries)
	}
}

type v2StableState struct {
	repository repositoryWindow
	objects    string
	files      map[string]v2StableFile
}

type v2StableFile struct {
	bytes   string
	mode    os.FileMode
	modTime int64
}

func captureV2StableState(t *testing.T, fixture v2RepositoryFixture) v2StableState {
	t.Helper()
	state := v2StableState{
		repository: captureRepositoryWindow(t, fixture.root),
		objects:    v2Git(t, fixture.root, "cat-file", "--batch-all-objects", "--batch-check=%(objectname)"),
		files:      make(map[string]v2StableFile),
	}
	roots := []string{
		v2EvidencePath(fixture),
		filepath.Join(fixture.root, filepath.FromSlash(canonicalV2ManifestPath(v2OrchestratorActID))),
		filepath.Join(fixture.root, filepath.FromSlash(canonicalV2ReportPath(v2OrchestratorActID))),
	}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(fixture.root, path)
			if err != nil {
				return err
			}
			state.files[rel] = v2StableFile{bytes: string(data), mode: info.Mode(), modTime: info.ModTime().UnixNano()}
			return nil
		})
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	return state
}

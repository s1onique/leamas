// SPDX-License-Identifier: Apache-2.0

package authority

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestClosureTagTargetMayBeAncestorOfCurrentHead(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	closure := git(t, repo, "rev-parse", "HEAD")
	git(t, repo, "tag", "--annotate", "--message", "closure", "act/"+fixtureActID)
	if err := os.WriteFile(filepath.Join(repo, "after.txt"), []byte("after\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "after.txt")
	git(t, repo, "commit", "-m", "post-closure descendant")

	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve descendant: %v", err)
	}
	if resolved.TagTarget != closure {
		t.Fatalf("tag target=%q want closure commit %q", resolved.TagTarget, closure)
	}
}

func TestClosureTagTargetEqualToCurrentHeadIsAccepted(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	closure := git(t, repo, "rev-parse", "HEAD")
	git(t, repo, "tag", "--annotate", "--message", "closure", "act/"+fixtureActID)

	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve equal target: %v", err)
	}
	if resolved.TagTarget != closure || resolved.AuthorityStatus == AuthorityMissingAuthority {
		t.Fatalf("resolved=%+v; want accepted annotated tag at HEAD", resolved)
	}
}

func TestClosureTagTargetUnrelatedToCurrentHeadIsRejected(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	git(t, repo, "tag", "--annotate", "--message", "closure", "act/"+fixtureActID)
	branch := git(t, repo, "branch", "--show-current")
	git(t, repo, "checkout", "--orphan", "unrelated")
	git(t, repo, "rm", "-rf", ".")
	if err := os.WriteFile(filepath.Join(repo, "unrelated.txt"), []byte("unrelated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "unrelated.txt")
	git(t, repo, "commit", "-m", "unrelated query")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatal("Resolve accepted closure tag unrelated to query HEAD")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("error type=%T: %v", err, err)
	}
	if authErr.Status == AuthorityAuthoritativeClosed || authErr.Status == AuthorityAuthoritativeClosedLocal {
		t.Fatalf("unrelated tag produced authoritative status: %v", authErr)
	}
	git(t, repo, "checkout", branch)
}

func TestLightweightClosureTagIsRejected(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	git(t, repo, "tag", "act/"+fixtureActID)

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatal("Resolve accepted lightweight closure tag")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("error type=%T: %v", err, err)
	}
	if authErr.Status != AuthorityTagMismatch && authErr.Status != AuthorityInvalidArtifact {
		t.Fatalf("status=%s; want tag/artifact rejection", authErr.Status)
	}
}

func TestAnnotatedTagPeelingToNonCommitIsRejected(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	tree := git(t, repo, "rev-parse", "HEAD^{tree}")
	git(t, repo, "tag", "--annotate", "--message", "tree target", "act/"+fixtureActID, tree)

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatal("Resolve accepted annotated tag peeling to non-commit")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) {
		t.Fatalf("error type=%T: %v", err, err)
	}
	if authErr.Status != AuthorityInvalidGitObject && authErr.Status != AuthorityTagMismatch && authErr.Status != AuthorityInvalidArtifact {
		t.Fatalf("status=%s; want non-commit rejection", authErr.Status)
	}
}

func TestNewestReachableClosureCandidateByAncestryIsSelected(t *testing.T) {
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	oldID := fixtureActID
	git(t, repo, "tag", "--annotate", "--message", "old closure", "act/"+oldID)

	newID := "ACT-LEAMAS-AUTHORITY-NEWER01"
	if err := os.WriteFile(filepath.Join(repo, "new-feature.go"), []byte("package newer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "new-feature.go")
	git(t, repo, "commit", "-q", "-m", "new subject")
	newSubject := git(t, repo, "rev-parse", "HEAD")
	writeManifestAndReport(t, repo, newID, newSubject)
	git(t, repo, "tag", "--annotate", "--message", "new closure", "act/"+newID)

	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve candidates: %v", err)
	}
	if resolved.ActID != newID || resolved.TagName != "act/"+newID {
		t.Fatalf("selected %+v; want newer ancestry-maximal candidate", resolved)
	}
}

func TestOlderAncestorCandidateIsDiscarded(t *testing.T) {
	// This is intentionally a separate assertion from the named
	// "newest" test: the older candidate must not participate in an
	// ambiguity once a descendant closure exists.
	repo := newCandidateRepo(t)
	_, _, _ = fixtureClosedLocal(t, repo)
	git(t, repo, "tag", "--annotate", "--message", "old", "act/"+fixtureActID)
	newID := "ACT-LEAMAS-AUTHORITY-DESCENDANT01"
	if err := os.WriteFile(filepath.Join(repo, "descendant.go"), []byte("package descendant\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "descendant.go")
	git(t, repo, "commit", "-q", "-m", "subject descendant")
	writeManifestAndReport(t, repo, newID, git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "tag", "--annotate", "--message", "descendant", "act/"+newID)

	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.ActID != newID {
		t.Fatalf("selected act %q; want %q", resolved.ActID, newID)
	}
}

func TestIncomparableClosureCandidatesAreAmbiguous(t *testing.T) {
	repo := newCandidateRepo(t)
	freeze, _, _ := fixtureClosedLocal(t, repo)
	firstID := fixtureActID
	git(t, repo, "tag", "--annotate", "--message", "first", "act/"+firstID)
	mainBranch := git(t, repo, "branch", "--show-current")

	git(t, repo, "checkout", "-b", "second", freeze)
	if err := os.WriteFile(filepath.Join(repo, "second.go"), []byte("package second\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "second.go")
	git(t, repo, "commit", "-q", "-m", "second subject")
	secondID := "ACT-LEAMAS-AUTHORITY-INCOMPARABLE01"
	writeManifestAndReport(t, repo, secondID, git(t, repo, "rev-parse", "HEAD"))
	git(t, repo, "tag", "--annotate", "--message", "second", "act/"+secondID)
	git(t, repo, "checkout", mainBranch)
	git(t, repo, "merge", "--no-ff", "second", "-m", "join incomparable closures")

	_, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err == nil {
		t.Fatal("Resolve selected one of incomparable candidates")
	}
	var authErr *AuthorityResolutionError
	if !errors.As(err, &authErr) || authErr.Status != AuthorityAmbiguousAuthority {
		t.Fatalf("error=%T %v; want AmbiguousAuthority", err, err)
	}
}

func TestCandidateSelectionIsIndependentOfTagEnumerationOrder(t *testing.T) {
	candidates := []closureCandidate{
		{TagName: "act/one", ActID: "ACT-ONE", ClosureCommit: "1111111"},
		{TagName: "act/two", ActID: "ACT-TWO", ClosureCommit: "2222222"},
	}
	ancestor := func(a, b string) bool {
		return a == b || a == "1111111" && b == "2222222"
	}
	for _, order := range [][]closureCandidate{
		candidates,
		{candidates[1], candidates[0]},
	} {
		selected, err := selectMaximalCandidates(order, ancestor)
		if err != nil {
			t.Fatalf("selectMaximalCandidates: %v", err)
		}
		if len(selected) != 1 || selected[0].ClosureCommit != "2222222" {
			t.Fatalf("order-dependent selection: %+v", selected)
		}
	}
}

func newCandidateRepo(t *testing.T) string {
	t.Helper()
	return newFixtureRepo(t)
}

func writeManifestAndReport(t *testing.T, repo, actID, subject string) {
	t.Helper()
	manifestDir := filepath.Join(repo, "docs", "closure-manifests")
	reportDir := filepath.Join(repo, "docs", "close-reports")
	planDir := filepath.Join(repo, "docs", "closure-plans")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(planDir, 0o755); err != nil {
		t.Fatal(err)
	}
	freeze := git(t, repo, "rev-parse", subject+"^")
	manifest := manifestJSON(actID, freeze, subject)
	if err := os.WriteFile(filepath.Join(planDir, actID+".json"), []byte("{\"act_id\":\""+actID+"\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, actID+".json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, actID+".md"), []byte("closure report\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "docs/closure-manifests", "docs/close-reports", "docs/closure-plans")
	git(t, repo, "commit", "-q", "-m", "closure artifacts "+actID)
}

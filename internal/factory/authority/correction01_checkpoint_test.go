package authority

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPostClosureHeadMayDifferFromExecutionSnapshot(t *testing.T) {
	repo := newFixtureRepo(t)
	freeze, subject, _ := fixtureClosedLocal(t, repo)
	git(t, repo, "tag", "--annotate", "--message", "closure", "act/"+fixtureActID)
	if err := os.WriteFile(filepath.Join(repo, "later.txt"), []byte("descendant\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "later.txt")
	git(t, repo, "commit", "-m", "descendant")
	resolved, err := Resolve(ResolverOptions{RepoRoot: repo})
	if err != nil {
		t.Fatalf("Resolve after descendant: %v", err)
	}
	if resolved.SubjectEnd != subject || resolved.DigestRange != freeze+".."+subject {
		t.Fatalf("subject range changed after closure descendant: %+v", resolved)
	}
}

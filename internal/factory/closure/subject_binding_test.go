package closure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newGitRepository(t *testing.T) (string, string) {
	t.Helper()
	repository := t.TempDir()
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"config", "user.name", "Closure Test"},
		{"config", "user.email", "closure@example.invalid"},
	} {
		if _, err := runGitValue(context.Background(), RealGit{}, repository, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if err := os.WriteFile(filepath.Join(repository, "subject.txt"), []byte("subject\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "subject.txt"}, {"commit", "-m", "subject"}} {
		if _, err := runGitValue(context.Background(), RealGit{}, repository, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	commit, err := runGitValue(context.Background(), RealGit{}, repository, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return repository, commit
}

func TestClosureRunRequiresCleanWorktree(t *testing.T) {
	repository, subject := newGitRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := snapshotSubject(context.Background(), RealGit{}, repository, subject)
	if err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("snapshot error = %v", err)
	}
}

func TestClosureRunRequiresHeadEqualsSubject(t *testing.T) {
	repository, subject := newGitRepository(t)
	if err := os.WriteFile(filepath.Join(repository, "second.txt"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "second.txt"}, {"commit", "-m", "second"}} {
		if _, err := runGitValue(context.Background(), RealGit{}, repository, args...); err != nil {
			t.Fatal(err)
		}
	}
	_, err := snapshotSubject(context.Background(), RealGit{}, repository, subject)
	if err == nil || !strings.Contains(err.Error(), "HEAD") {
		t.Fatalf("snapshot error = %v", err)
	}
}

func TestClosureRunBindsFullCommitAndTree(t *testing.T) {
	repository, subject := newGitRepository(t)
	snapshot, err := snapshotSubject(context.Background(), RealGit{}, repository, subject[:12])
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SubjectCommitOID != subject || len(snapshot.SubjectTreeOID) != len(subject) || snapshot.HeadCommitOID != subject {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestClosureRunRejectsSubjectTreeMismatch(t *testing.T) {
	repository, subject := newGitRepository(t)
	client := &treeMismatchGitClient{delegate: RealGit{}}
	_, err := snapshotSubject(context.Background(), client, repository, subject)
	if err == nil || !strings.Contains(err.Error(), "tree") {
		t.Fatalf("snapshot error = %v", err)
	}
}

type treeMismatchGitClient struct {
	delegate  gitClient
	treeCalls int
}

func (c *treeMismatchGitClient) Run(ctx context.Context, directory string, args ...string) gitCommandResult {
	result := c.delegate.Run(ctx, directory, args...)
	if len(args) >= 3 && args[0] == "rev-parse" && args[1] == "--verify" && strings.HasSuffix(args[len(args)-1], "^{tree}") {
		c.treeCalls++
		if c.treeCalls == 2 {
			result.Stdout = []byte(strings.Repeat("f", 40) + "\n")
		}
	}
	return result
}

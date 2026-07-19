package releasedeb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPublicationRejectsPublicationHazards(t *testing.T) {
	cases := []struct {
		name string
		edit func(t *testing.T, repository string)
		want string
	}{
		{
			name: "dirty tree",
			edit: func(t *testing.T, repository string) {
				writeGitFile(t, repository, "dirty.txt", "uncommitted\n")
			},
			want: "publication tree is dirty",
		},
		{
			name: "tag not at HEAD",
			edit: func(t *testing.T, repository string) {
				writeGitFile(t, repository, "later.txt", "later\n")
				runGit(t, repository, "add", "later.txt")
				runGit(t, repository, "commit", "-m", "later")
			},
			want: "does not point at HEAD",
		},
		{
			name: "remote tag absent",
			edit: func(t *testing.T, repository string) {},
			want: "remote tag v0.1.0 does not exist",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repository, remote := newPublicationRepository(t, tc.name != "remote tag absent")
			tc.edit(t, repository)
			err := CheckPublication(context.Background(), PublicationConfig{
				Repository: repository,
				Remote:     remote,
				Tag:        "v0.1.0",
				Assets:     []string{"dist/leamas_0.1.0_amd64.deb", "dist/SHA256SUMS"},
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CheckPublication error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestCheckPublicationRejectsDuplicateReleaseAssets(t *testing.T) {
	repository, remote := newPublicationRepository(t, true)
	err := CheckPublication(context.Background(), PublicationConfig{
		Repository: repository,
		Remote:     remote,
		Tag:        "v0.1.0",
		Assets:     []string{"dist/one.deb", "other/one.deb"},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate release asset name") {
		t.Fatalf("CheckPublication error = %v, want duplicate asset rejection", err)
	}
}

func TestCheckPublicationAcceptsCleanPushedTagAtHead(t *testing.T) {
	repository, remote := newPublicationRepository(t, true)
	if err := CheckPublication(context.Background(), PublicationConfig{
		Repository: repository,
		Remote:     remote,
		Tag:        "v0.1.0",
		Assets:     []string{"leamas_0.1.0_amd64.deb", "SHA256SUMS"},
	}); err != nil {
		t.Fatalf("CheckPublication() error = %v", err)
	}
}

func newPublicationRepository(t *testing.T, pushTag bool) (string, string) {
	t.Helper()
	repository := t.TempDir()
	remote := filepath.Join(t.TempDir(), "origin.git")
	runGit(t, repository, "init", "-b", "main")
	runGit(t, repository, "config", "user.email", "release-test@example.invalid")
	runGit(t, repository, "config", "user.name", "Release Test")
	writeGitFile(t, repository, "README", "release test\n")
	runGit(t, repository, "add", "README")
	runGit(t, repository, "commit", "-m", "initial")
	runGit(t, repository, "tag", "-a", "v0.1.0", "-m", "v0.1.0")
	runGit(t, remote, "init", "--bare")
	runGit(t, repository, "remote", "add", "origin", remote)
	if pushTag {
		runGit(t, repository, "push", "origin", "refs/tags/v0.1.0")
	}
	return repository, remote
}

func writeGitFile(t *testing.T, repository, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repository, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, directory string, args ...string) string {
	t.Helper()
	if directory != "" {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	output, err := commandOutput(context.Background(), directory, "git", args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

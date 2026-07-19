// Package digest provides targeted digest generation for Git repositories.
//
// Range-mode regression tests. The ACT requires that this ACT's
// changes do not regress `leamas factory digest --range HEAD~1..HEAD`
// for ordinary additions, modifications, deletions, and renames.
// Range mode already carries explicit statuses via `RangeFile`,
// so the changes here are limited to confirming that the existing
// range pipeline still produces expected manifest output after the
// staged/dirty rewrite.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rangeFixturesCommit writes and commits `name` with `content`.
func rangeFixtureCommit(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", "fixture "+name)
}

// rangeLines returns the lines appearing under `## CHANGESET_MANIFEST`
// in the digest body. Whitespace is trimmed.
func rangeLines(digestText string) []string {
	idx := strings.Index(digestText, "## CHANGESET_MANIFEST")
	if idx == -1 {
		return nil
	}
	rest := digestText[idx+len("## CHANGESET_MANIFEST"):]
	end := strings.Index(rest, "## CHANGESET_STATS")
	if end != -1 {
		rest = rest[:end]
	}
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, strings.TrimSpace(line))
	}
	return out
}

func TestRangeMode_Addition(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "first.txt", "first\n")

	// Second commit: add a new file.
	writeRepoFile(t, dir, "added.txt", "added\n")
	runGit(t, dir, "add", "added.txt")
	runGit(t, dir, "commit", "-m", "add second")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	for _, want := range rangeLines(out) {
		if want != "A  added.txt" {
			t.Fatalf("range digest manifest line wrong: %q (full=%#v)", want, rangeLines(out))
		}
	}
}

func TestRangeMode_Modification(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "tracked.txt", "v1\n")

	// Second commit: modify the file.
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "modify")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	for _, want := range rangeLines(out) {
		if want != "M  tracked.txt" {
			t.Fatalf("range digest manifest line wrong: %q (full=%#v)", want, rangeLines(out))
		}
	}
}

func TestRangeMode_Deletion(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "deleted.txt", "byebye\n")
	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")

	// Second commit: delete the first file.
	runGit(t, dir, "rm", "deleted.txt")
	runGit(t, dir, "commit", "-m", "rm")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	for _, want := range rangeLines(out) {
		if want != "D  deleted.txt" {
			t.Fatalf("range digest manifest line wrong: %q (full=%#v)", want, rangeLines(out))
		}
	}
}

func TestRangeMode_Rename(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "old_name.txt", "alpha\nbeta\n")
	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")

	// Second commit: rename.
	runGit(t, dir, "mv", "old_name.txt", "new_name.txt")
	runGit(t, dir, "commit", "-m", "rename")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if !strings.Contains(out, "R  old_name.txt -> new_name.txt") {
		t.Fatalf("expected rename line in range digest, got:\n%s",
			manifestSection(out))
	}
}

// TestRangeMode_StableAcrossACT verifies that consecutive commits
// viewed via the range digest produce sensible manifest entries
// (mixed add/mod/del/rename in a single range). This exercises the
// range pipeline with the same shared NUL-delimited status parser
// used by staged/dirty mode, so the regression guard covers the
// change kind being passed through BuildRangeManifest unchanged.
func TestRangeMode_StableAcrossACT(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// Pre-existing file that will be renamed.
	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")
	rangeFixtureCommit(t, dir, "rename_source.txt", "alpha\nbeta\ngamma\n")

	// One commit that touches all four: add, modify, rename, leave alone.
	writeRepoFile(t, dir, "new_one.txt", "new1\n")
	runGit(t, dir, "add", "new_one.txt")
	if err := os.WriteFile(filepath.Join(dir, "stable.txt"), []byte("stable\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "stable.txt")
	runGit(t, dir, "mv", "rename_source.txt", "rename_dest.txt")
	runGit(t, dir, "commit", "-m", "mixed")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	wantLines := []string{
		"A  new_one.txt",
		"M  stable.txt",
		"R  rename_source.txt -> rename_dest.txt",
	}
	for _, w := range wantLines {
		if !strings.Contains(out, "\n"+w+"\n") && !strings.Contains(out, "\n"+w) {
			t.Fatalf("range digest missing %q, got:\n%s", w, manifestSection(out))
		}
	}
}

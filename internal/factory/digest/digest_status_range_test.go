// Package digest provides targeted digest generation for Git repositories.
//
// Range-mode regression tests. The ACT requires that this ACT's
// changes do not regress `leamas factory digest --range HEAD~1..HEAD`
// for ordinary additions, modifications, deletions, and renames.
// Each test uses exact equality assertions against the rendered
// manifest lines so that an unexpected empty manifest would fail
// the test instead of silently passing.
package digest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// rangeFixtureCommit writes and commits `name` with `content`.
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
// in the digest body, in the order they were rendered, with
// surrounding whitespace trimmed. Each non-empty line corresponds
// to one manifest record.
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

// assertManifestLinesExact is the canonical range-mode assertion. It
// requires an exact, ordered match between the rendered manifest
// lines and the expected list. An empty list in the rendered
// output fails the test.
func assertManifestLinesExact(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("manifest mismatch\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestRangeMode_Addition(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "first.txt", "first\n")

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
	assertManifestLinesExact(t, rangeLines(out), []string{"A  added.txt"})
}

func TestRangeMode_Modification(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "tracked.txt", "v1\n")

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
	assertManifestLinesExact(t, rangeLines(out), []string{"M  tracked.txt"})
}

func TestRangeMode_Deletion(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "deleted.txt", "byebye\n")
	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")

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
	assertManifestLinesExact(t, rangeLines(out), []string{"D  deleted.txt"})
}

func TestRangeMode_Rename(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "old_name.txt", "alpha\nbeta\n")
	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")

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
	assertManifestLinesExact(t, rangeLines(out), []string{
		"R  old_name.txt -> new_name.txt",
	})
}

// TestRangeMode_MixedAllKinds verifies that a single range commit
// that exercises an addition, a modification, a deletion, and a
// rename at once produces the four corresponding manifest lines.
// Per the reviewer: the previous implementation only covered three
// of the four; this test makes the "all four in one range" coverage
// load-bearing rather than advisory.
func TestRangeMode_MixedAllKinds(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "stable.txt", "stable\n")
	rangeFixtureCommit(t, dir, "rename_source.txt", "alpha\nbeta\ngamma\n")
	rangeFixtureCommit(t, dir, "doomed.txt", "byebye\n")

	// One commit touches all four: add, modify, rename, delete.
	writeRepoFile(t, dir, "new_one.txt", "new1\n")
	runGit(t, dir, "add", "new_one.txt")
	if err := os.WriteFile(filepath.Join(dir, "stable.txt"), []byte("stable\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "stable.txt")
	runGit(t, dir, "mv", "rename_source.txt", "rename_dest.txt")
	runGit(t, dir, "rm", "doomed.txt")
	runGit(t, dir, "commit", "-m", "mixed")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	want := []string{
		// Sorted lexicographically by path; the digest sorts entries
		// by `Path` (the destination for renames).
		"D  doomed.txt",
		"A  new_one.txt",
		"R  rename_source.txt -> rename_dest.txt",
		"M  stable.txt",
	}
	assertManifestLinesExact(t, rangeLines(out), want)
}

// TestRangeMode_TypeChange verifies that a regular-file -> symlink
// change in a commit renders as `T` in the manifest, end to end.
// Symlink creation requires a Unix-like filesystem; we attempt the
// commit and skip if the platform rejects it.
func TestRangeMode_TypeChange(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	rangeFixtureCommit(t, dir, "linked.go", "alpha\nbeta\n")

	// Replace the file with a symlink. Skip if unsupported.
	if err := os.Remove(filepath.Join(dir, "linked.go")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink("elsewhere", filepath.Join(dir, "linked.go")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "type change")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	// The exact rendered line depends on whether Git reports a
	// rename + mode, or a pure type change. Both render with the
	// leading "T  " status (or "T  " in a rename pair) and contain
	// the destination path.
	assertManifestLinesExact(t, rangeLines(out), []string{"T  linked.go"})
}

// TestRangeMode_CopyWithModifiedSource exercises the C path of
// `BuildRangeManifest`. When the source file is also modified in
// the same commit, Git's `--find-copies` will detect the copy. The
// digest must render the source-and-destination pair in the canonical
// `C  source.go -> copy.go` form, and `CHANGESET_STATS` must
// record the copy in `copied_files=1`.
func TestRangeMode_CopyWithModifiedSource(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	rangeFixtureCommit(t, dir, "source.go", "alpha\nbeta\n")

	// Same commit: modify source AND add a copy of the modified
	// content. Git's --find-copies then surfaces the copy.
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("alpha\nbeta\ngamma\n"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := copyFileLike(filepath.Join(dir, "source.go"), filepath.Join(dir, "copy.go")); err != nil {
		t.Fatalf("copy: %v", err)
	}
	runGit(t, dir, "add", "source.go", "copy.go")
	runGit(t, dir, "commit", "-m", "copy with modified source")

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeRange,
		Range:    "HEAD~1..HEAD",
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	assertManifestLinesExact(t, rangeLines(out), []string{
		"C  source.go -> copy.go",
		"M  source.go",
	})
	if got := digestStatValue(out, "copied_files"); got != "1" {
		t.Fatalf("copied_files = %q, want 1", got)
	}
	if got := digestStatValue(out, "modified_files"); got != "1" {
		t.Fatalf("modified_files = %q, want 1", got)
	}
}

// copyFileLike is a tiny read/write helper local to this test file.
func copyFileLike(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

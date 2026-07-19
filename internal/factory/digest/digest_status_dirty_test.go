// Package digest provides targeted digest generation for Git repositories.
//
// Integration tests for dirty-mode status classification.
//
// The ACT specifies a contract table for dirty mode manifest statuses:
// each scenario below verifies a row from that table by inspecting the
// rendered digest, with the path typically also compared against the
// authoritative `git diff --name-status -z --find-renames HEAD --`
// output of the same repository.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dirtyDigestAndOracle generates a dirty-mode digest for `dir` and
// returns both the digest body and the parsed authoritative Git
// oracle (`git diff --name-status -z --find-renames=<n>% HEAD --`,
// plus untracked files as `?` records).
func dirtyDigestAndOracle(t *testing.T, dir string) (string, []GitChange) {
	t.Helper()
	out, err := Generate(Options{RepoRoot: dir, Mode: ModeDirty})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	baseRef, err := dirtyOracleBaseRef(dir)
	if err != nil {
		t.Fatalf("oracle base ref: %v", err)
	}
	oracleBytes := runGitCaptured(t, dir,
		"diff", "--name-status", "-z",
		fmt.Sprintf("--find-renames=%d%%", RenameSimilarityThreshold),
		fmt.Sprintf("--find-copies=%d%%", RenameSimilarityThreshold),
		baseRef, "--",
	)
	oracle, parseErr := ParseGitStatusRecords(oracleBytes)
	if parseErr != nil {
		t.Fatalf("oracle parse failed: %v\nraw:%q", parseErr, oracleBytes)
	}
	untrackedBytes := runGitCaptured(t, dir,
		"ls-files", "--others", "--exclude-standard", "-z",
	)
	for _, p := range strings.Split(untrackedBytes, "\x00") {
		if p == "" {
			continue
		}
		oracle = append(oracle, GitChange{Kind: KindUntracked, Path: p})
	}
	return out, oracle
}

func dirtyOracleBaseRef(dir string) (string, error) {
	if _, code := RunGitWithExitCodeForTest(dir, []string{"rev-parse", "--verify", "HEAD"}); code == 0 {
		return "HEAD", nil
	}
	return "4b825dc642cb6eb9a060e54bf8d69288fbee4904", nil
}

// lineFor returns the first manifest line whose path matches `path`.
// Lines are either `STATUS  PATH` or `STATUS  OLD -> NEW`; the path
// we look for is always the rightmost token.
func lineFor(digestText, path string) (string, bool) {
	for _, l := range digestManifestLines(digestText) {
		fields := strings.Fields(l)
		if len(fields) >= 2 && fields[len(fields)-1] == path {
			return l, true
		}
	}
	return "", false
}

// dirtySetupCommit writes `name` with `content`, stages and commits it.
// It returns the absolute path of the file.
func dirtySetupCommit(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", name)
	runGit(t, dir, "commit", "-m", "init "+name)
	return p
}

func TestDirtyStatus_ModifiedOnlyInWorktree(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "a.go", "v1\n")

	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, oracle := dirtyDigestAndOracle(t, dir)

	if line, ok := lineFor(out, "a.go"); !ok || !strings.HasPrefix(line, "M ") {
		t.Fatalf("expected M  a.go in manifest, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindModified || oracle[0].Path != "a.go" {
		t.Fatalf("oracle unexpected: %#v", oracle)
	}
}

func TestDirtyStatus_ModifiedAndStaged(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "a.go", "v1\n")

	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "a.go")

	out, oracle := dirtyDigestAndOracle(t, dir)

	if line, ok := lineFor(out, "a.go"); !ok || !strings.HasPrefix(line, "M ") {
		t.Fatalf("expected M  a.go, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindModified {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_ModifiedBothStagedAndUnstaged(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "a.go", "v1\n")
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "a.go")
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("v1\nv2\nv3\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, oracle := dirtyDigestAndOracle(t, dir)
	if line, ok := lineFor(out, "a.go"); !ok || !strings.HasPrefix(line, "M ") {
		t.Fatalf("expected M  a.go, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindModified {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_NewlyAddedOnly(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "new.go", "fresh\n")
	runGit(t, dir, "add", "new.go")

	out, oracle := dirtyDigestAndOracle(t, dir)
	if line, ok := lineFor(out, "new.go"); !ok || !strings.HasPrefix(line, "A ") {
		t.Fatalf("expected A  new.go, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindAdded {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_AddedThenWorktreeEdit(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "added.go", "v1\n")
	runGit(t, dir, "add", "added.go")
	if err := os.WriteFile(filepath.Join(dir, "added.go"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, oracle := dirtyDigestAndOracle(t, dir)
	if line, ok := lineFor(out, "added.go"); !ok || !strings.HasPrefix(line, "A ") {
		t.Fatalf("expected A  added.go, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindAdded {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_DeletionStaged(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "victim.go", "x\n")
	runGit(t, dir, "rm", "victim.go")

	out, oracle := dirtyDigestAndOracle(t, dir)
	if line, ok := lineFor(out, "victim.go"); !ok || !strings.HasPrefix(line, "D ") {
		t.Fatalf("expected D  victim.go, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindDeleted {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_RenamedStaged(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "old.go", "package x\nfunc Old() {}\n")
	runGit(t, dir, "mv", "old.go", "new.go")
	runGit(t, dir, "add", "-A")

	out, oracle := dirtyDigestAndOracle(t, dir)

	if !strings.Contains(manifestSection(out), "R  old.go -> new.go") {
		t.Fatalf("expected 'R  old.go -> new.go' in manifest, got:\n%s", manifestSection(out))
	}
	if len(oracle) != 1 || oracle[0].Kind != KindRenamed {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_RenamedStagedThenUnstagedEdit(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "old.go", "package x\nfunc Old() {}\n")
	runGit(t, dir, "mv", "old.go", "new.go")
	runGit(t, dir, "add", "-A")
	// The relative change is small enough to keep similarity above the
	// digest's --find-renames=30% threshold so rename detection still
	// fires after the worktree edit.
	if err := os.WriteFile(filepath.Join(dir, "new.go"),
		[]byte("package x\nfunc New() {}\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, oracle := dirtyDigestAndOracle(t, dir)
	if !strings.Contains(manifestSection(out), "R  old.go -> new.go") {
		t.Fatalf("expected rename line in manifest even after unstaged edit, got:\n%s",
			manifestSection(out))
	}
	if len(oracle) != 1 || oracle[0].Kind != KindRenamed {
		t.Fatalf("oracle: %#v", oracle)
	}
}

func TestDirtyStatus_UntrackedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "stray.txt", "loose\n")

	out, oracle := dirtyDigestAndOracle(t, dir)
	if line, ok := lineFor(out, "stray.txt"); !ok || !strings.HasPrefix(line, "?") {
		t.Fatalf("expected ?  stray.txt, got line=%q ok=%v", line, ok)
	}
	if len(oracle) != 1 || oracle[0].Kind != KindUntracked {
		t.Fatalf("oracle: %#v", oracle)
	}
}

// TestDirtyStatus_MixedDeterministicOrder verifies the manifest is
// deterministic across the three categories (tracked added / tracked
// modified / untracked). The manifest is sorted lexicographically by
// path, which is the canonical deterministic order the digest commits
// to. Two repeated digests on the same repository state must yield
// identical manifest lines.
func TestDirtyStatus_MixedDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	dirtySetupCommit(t, dir, "tracked.go", "v1\n")
	if err := os.WriteFile(filepath.Join(dir, "tracked.go"), []byte("v1\nv2\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, dir, "add", "tracked.go")
	writeRepoFile(t, dir, "staged_new.go", "package x\n")
	runGit(t, dir, "add", "staged_new.go")
	writeRepoFile(t, dir, "stray.go", "stray\n")

	out, _ := dirtyDigestAndOracle(t, dir)
	lines := digestManifestLines(out)

	// All three categories must appear in the manifest.
	wantPaths := []string{"staged_new.go", "stray.go", "tracked.go"}
	for _, p := range wantPaths {
		found := false
		for _, l := range lines {
			fields := strings.Fields(l)
			if len(fields) >= 2 && fields[len(fields)-1] == p {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %q in manifest lines, got %#v", p, lines)
		}
	}

	// Manifest is sorted by Path; verify that invariant directly so the
	// contract is fixed by a property the tests own, not by a hidden
	// detail of BuildManifest.
	for i := 1; i < len(lines); i++ {
		prev := manifestPathOf(lines[i-1])
		cur := manifestPathOf(lines[i])
		if prev == "" || cur == "" {
			continue
		}
		if prev > cur {
			t.Fatalf("manifest not sorted by path at index %d: %q > %q\nlines=%#v",
				i, prev, cur, lines)
		}
	}

	// Two repeated digests must yield identical manifest lines (the
	// contract for deterministic ordering).
	out2, _ := dirtyDigestAndOracle(t, dir)
	lines2 := digestManifestLines(out2)
	if !equalStringSlices(lines, lines2) {
		t.Fatalf("manifest not deterministic:\nfirst:  %#v\nsecond: %#v", lines, lines2)
	}
}

// manifestPathOf returns the path that the manifest line refers to.
// Lines are `STATUS  PATH` or `STATUS  OLD -> NEW`; we return PATH
// (or NEW for renames/copies).
func manifestPathOf(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return ""
	}
	return fields[len(fields)-1]
}

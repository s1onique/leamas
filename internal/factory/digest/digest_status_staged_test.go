// Package digest provides targeted digest generation for Git repositories.
//
// Integration tests for staged-mode status classification.
//
// The ACT-LEAMAS-FACTORY-DIGEST-STAGED-STATUS-CLASSIFICATION01 contract
// requires CHANGESET_MANIFEST in staged mode to agree path-for-path
// with `git diff --cached --name-status HEAD --`. These tests exercise
// each document scenario in an isolated temporary Git repository.
package digest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// runGitCaptured is a test helper that runs `git` in `dir`, returns
// stdout, and fails the test on non-zero exit. Stderr is included in
// the failure message.
func runGitCaptured(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, code := RunGitWithExitCodeForTest(dir, args)
	if code != 0 {
		t.Fatalf("git %v failed in %s (exit %d, stdout=%q)", args, dir, code, output)
	}
	return output
}

// expectedManifestLines renders parsed GitChange records into the
// manifest line format used by RenderManifest. The digest sorts by
// Path then formats each entry, so the helper mirrors that ordering.
func expectedManifestLines(changes []GitChange) []string {
	sorted := make([]GitChange, len(changes))
	copy(sorted, changes)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})
	lines := make([]string, 0, len(sorted))
	for _, ch := range sorted {
		if ch.OldPath != "" && ch.OldPath != ch.Path {
			lines = append(lines, string(ch.Kind)+"  "+ch.OldPath+" -> "+ch.Path)
		} else {
			lines = append(lines, string(ch.Kind)+"  "+ch.Path)
		}
	}
	return lines
}

// manifestSection returns the substring of a digest starting at
// `## CHANGESET_MANIFEST` and ending before the next `## ` heading.
func manifestSection(digestText string) string {
	const start = "## CHANGESET_MANIFEST"
	const nextHeading = "## CHANGESET_STATS"
	idx := strings.Index(digestText, start)
	if idx == -1 {
		return ""
	}
	rest := digestText[idx+len(start):]
	end := strings.Index(rest, nextHeading)
	if end == -1 {
		return rest
	}
	return rest[:end]
}

// digestManifestLines returns the non-empty lines of the
// CHANGESET_MANIFEST section of a digest, excluding the heading line
// and the trailing blank line that precedes the next section.
func digestManifestLines(digestText string) []string {
	section := manifestSection(digestText)
	var lines []string
	for _, l := range strings.Split(section, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		lines = append(lines, l)
	}
	return lines
}

// digestStatValue parses one `key=value` field from CHANGESET_STATS.
func digestStatValue(digestText, key string) string {
	const marker = "## CHANGESET_STATS"
	idx := strings.Index(digestText, marker)
	if idx == -1 {
		return ""
	}
	rest := digestText[idx+len(marker):]
	end := strings.Index(rest, "## ")
	if end == -1 {
		rest = digestText[idx:]
	} else {
		rest = rest[:end]
	}
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}

// stagedOracleBaseRef returns the rev to feed the staged oracle
// command. For normal repositories this is `HEAD`. For unborn HEAD
// (no commits yet) we fall back to Git's empty-tree SHA so the
// staged oracle still runs against the same baseline the digest
// itself uses.
func stagedOracleBaseRef(t *testing.T, dir string) string {
	t.Helper()
	if _, code := RunGitWithExitCodeForTest(dir, []string{"rev-parse", "--verify", "HEAD"}); code == 0 {
		return "HEAD"
	}
	return "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
}

// stagedOracleArgs builds the exact git diff args the digest uses for
// the staged mode. Tests use this oracle as the authoritative source
// so the manifest can be reconciled against actual Git output rather
// than against a duplicate fixture.
func stagedOracleArgs(baseRef string) []string {
	return []string{
		"diff", "--cached", "--name-status", "-z",
		fmt.Sprintf("--find-renames=%d%%", RenameSimilarityThreshold),
		"--find-copies", baseRef, "--",
	}
}

// requireStagedAgreementAgainstOracle enforces exact agreement between
// the rendered staged manifest and the authoritative Git oracle for
// the same repository state.
//
// The oracle is parsed from `git diff --cached --name-status -z --find-renames --find-copies <base> --`,
// where `<base>` is HEAD for normal repositories and the empty-tree
// SHA otherwise. Tests must not duplicate expected status between the
// digest side and the fixture: the digest manifest is compared
// against the parsed oracle, full stop.
func requireStagedAgreementAgainstOracle(t *testing.T, dir string) {
	t.Helper()

	baseRef := stagedOracleBaseRef(t, dir)
	oracleOut := runGitCaptured(t, dir, stagedOracleArgs(baseRef)...)
	oracle, err := ParseGitStatusRecords(oracleOut)
	if err != nil {
		t.Fatalf("oracle parse failed: %v\nraw:%q", err, oracleOut)
	}
	wantLines := expectedManifestLines(oracle)

	wantStats := map[string]int{
		"added_files":     0,
		"modified_files":  0,
		"deleted_files":   0,
		"renamed_files":   0,
		"copied_files":    0,
		"untracked_files": 0,
		"unmerged_files":  0,
	}
	for _, ch := range oracle {
		switch ch.Kind {
		case KindAdded:
			wantStats["added_files"]++
		case KindModified:
			wantStats["modified_files"]++
		case KindDeleted:
			wantStats["deleted_files"]++
		case KindRenamed:
			wantStats["renamed_files"]++
		case KindCopied:
			wantStats["copied_files"]++
		case KindUnmerged:
			wantStats["unmerged_files"]++
		}
	}

	out, err := Generate(Options{
		RepoRoot: dir,
		Mode:     ModeStaged,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	gotLines := digestManifestLines(out)
	if !equalStringSlices(gotLines, wantLines) {
		t.Fatalf("manifest mismatch\nwant: %#v\ngot:  %#v", wantLines, gotLines)
	}

	for key, want := range wantStats {
		got := digestStatValue(out, key)
		if got != intToString(want) {
			t.Fatalf("stats mismatch for %q: got %q, want %q", key, got, intToString(want))
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeRepoFile is a tiny helper to materialise a file in `dir` with the
// given content. Returns the absolute path.
func writeRepoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestStagedStatus_ModifiedExistingFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "tracked.go", "v1\n")
	runGit(t, dir, "add", "tracked.go")
	runGit(t, dir, "commit", "-m", "initial")
	writeRepoFile(t, dir, "tracked.go", "v1\nv2\n")
	runGit(t, dir, "add", "tracked.go")

	requireStagedAgreementAgainstOracle(t, dir)
}

func TestStagedStatus_NewlyAddedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "new.go", "fresh\n")
	runGit(t, dir, "add", "new.go")

	requireStagedAgreementAgainstOracle(t, dir)
}

func TestStagedStatus_DeletedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "victim.go", "1\n2\n3\n")
	runGit(t, dir, "add", "victim.go")
	runGit(t, dir, "commit", "-m", "initial")
	runGit(t, dir, "rm", "victim.go")

	requireStagedAgreementAgainstOracle(t, dir)
}

func TestStagedStatus_RenamedFile(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "old_name.go", "package x\nfunc Old() {}\n")
	runGit(t, dir, "add", "old_name.go")
	runGit(t, dir, "commit", "-m", "initial")
	runGit(t, dir, "mv", "old_name.go", "new_name.go")
	runGit(t, dir, "add", "-A")

	requireStagedAgreementAgainstOracle(t, dir)
}

// TestStagedStatus_FourAddedOneModified reproduces the original defect:
// an existing tracked file is modified in HEAD, four new files are
// staged. The digest must report `modified_files=1` and `added_files=4`.
func TestStagedStatus_FourAddedOneModified(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// Existing tracked file in HEAD.
	writeRepoFile(t, dir, "internal/factory/gate/gate.go", "package gate\n")
	runGit(t, dir, "add", "internal/factory/gate/gate.go")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify and stage the existing file.
	writeRepoFile(t, dir, "internal/factory/gate/gate.go", "package gate\n// change\n")
	runGit(t, dir, "add", "internal/factory/gate/gate.go")

	// Add and stage four new files.
	for _, p := range []string{
		"new_one.go",
		"new_two.go",
		"new_three.go",
		"new_four.go",
	} {
		writeRepoFile(t, dir, p, "package x\n// new\n")
		runGit(t, dir, "add", p)
	}

	out, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if got := digestStatValue(out, "added_files"); got != "4" {
		t.Fatalf("added_files = %q, want 4", got)
	}
	if got := digestStatValue(out, "modified_files"); got != "1" {
		t.Fatalf("modified_files = %q, want 1", got)
	}

	// The exact file that triggered the defect must render M, never A.
	lines := digestManifestLines(out)
	foundModifiedGate := false
	for _, line := range lines {
		if line == "M  internal/factory/gate/gate.go" {
			foundModifiedGate = true
			continue
		}
		if strings.HasSuffix(line, "internal/factory/gate/gate.go") && !strings.HasPrefix(line, "M ") {
			t.Fatalf("expected gate.go to render M, got line %q", line)
		}
	}
	if !foundModifiedGate {
		t.Fatalf("expected M  internal/factory/gate/gate.go, manifest = %#v", lines)
	}

	// Reconcile against literal Git oracle.
	requireStagedAgreementAgainstOracle(t, dir)
}

func TestStagedStatus_FilenamesWithSpacesAndUnicode(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "file with spaces.go", "package x\n")
	writeRepoFile(t, dir, "путь/файл.go", "package x\n")
	runGit(t, dir, "add", "file with spaces.go", "путь/файл.go")

	requireStagedAgreementAgainstOracle(t, dir)
}

func TestStagedStatus_EmptyChangeset(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	writeRepoFile(t, dir, "stable.go", "package x\n")
	runGit(t, dir, "add", "stable.go")
	runGit(t, dir, "commit", "-m", "initial")

	out, err := Generate(Options{RepoRoot: dir, Mode: ModeStaged})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if got := digestStatValue(out, "files_changed"); got != "0" {
		t.Fatalf("files_changed = %q, want 0", got)
	}
	if !strings.Contains(manifestSection(out), "(no changed files)") {
		t.Fatal("empty manifest must include '(no changed files)'")
	}
}

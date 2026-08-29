// SPDX-License-Identifier: Apache-2.0

// Evidence-overlay mandatory tests (M01-M12) plus a small set
// of highly-desirable matrix rows (M13-M22). The tests prove
// the MVP safety invariants:
//
//   - explicit ignored-untracked files are included (M01, M02)
//   - unrelated ignored namespaces are excluded (M03)
//   - clean subject remains clean even with an overlay (M04)
//   - the overlay is rendered separately from the subject (M05)
//   - path escape, absolute paths, .git, and symlinks are
//     rejected (M06, M07, M08, M09)
//   - redaction is reused (M10)
//   - absence of --evidence-path preserves prior behavior (M11)
//   - the collection is read-only against its target (M12)
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeEvidenceFixture builds a hermetic temporary repository that
// satisfies the binary-mvp01 contract. The caller receives the
// repoRoot; the directory is cleaned up via t.TempDir.
func makeEvidenceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGitNoErr(t, dir, "init", "-q", "--initial-branch=main")
	mustGitNoErr(t, dir, "config", "user.email", "test@example.com")
	mustGitNoErr(t, dir, "config", "user.name", "Test User")
	mustGitNoErr(t, dir, "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"),
		[]byte("tracked\n"), 0644); err != nil {
		t.Fatalf("write tracked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"),
		[]byte("ignored-a/\nignored-b/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tracked-evidence.log"),
		[]byte("plain log\n"), 0644); err != nil {
		t.Fatalf("write tracked log: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored-a", "nested"),
		0755); err != nil {
		t.Fatalf("mkdir ignored-a: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored-b"), 0755); err != nil {
		t.Fatalf("mkdir ignored-b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored-a", "a.txt"),
		[]byte("ignored a\n"), 0644); err != nil {
		t.Fatalf("write ignored-a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored-a", "nested", "b.txt"),
		[]byte("ignored nested\n"), 0644); err != nil {
		t.Fatalf("write ignored-a/nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored-b", "secret.txt"),
		[]byte("TOP-SECRET\n"), 0644); err != nil {
		t.Fatalf("write ignored-b: %v", err)
	}

	mustGitNoErr(t, dir, "add", "tracked.txt", ".gitignore",
		"tracked-evidence.log")
	mustGitNoErr(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

// mustGitNoErr runs `git <args...>` in dir, succeeding silently
// when the command is well-formed but produces no stdout (e.g.
// `git init` on an empty dir).
func mustGitNoErr(t *testing.T, dir string, args ...string) {
	t.Helper()
	RunGitForTest(dir, append([]string{}, args...))
}

// firstNLines returns the first n lines of s for diagnostic
// diffing in error messages.
func firstNLines(s string, n int) string {
	parts := strings.Split(s, "\n")
	if len(parts) < n {
		return s
	}
	return strings.Join(parts[:n], "\n")
}

// ----- M01: explicit file (via direct root spec) -------------------

func TestEvidence_Collect_M01_ExplicitFile(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a/a.txt"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(overlay.Files) != 1 || overlay.Files[0].Path != "ignored-a/a.txt" {
		t.Fatalf("expected exactly ignored-a/a.txt, got %+v", overlay.Files)
	}
}

// ----- M02: recursive directory -------------------------------------

func TestEvidence_Collect_M02_RecursiveDirectory(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(overlay.Files) < 2 {
		t.Fatalf("expected >= 2 files, got %d", len(overlay.Files))
	}
	for i := 1; i < len(overlay.Files); i++ {
		if overlay.Files[i-1].Path > overlay.Files[i].Path {
			t.Errorf("not sorted: %+v", overlay.Files)
		}
	}
}

// ----- M03: unrelated ignored namespace excluded -------------------

func TestEvidence_Collect_M03_UnrelatedExcluded(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	var sawA, sawNested bool
	for _, f := range overlay.Files {
		if f.Path == "ignored-a/a.txt" {
			sawA = true
		}
		if f.Path == "ignored-a/nested/b.txt" {
			sawNested = true
		}
		if f.Path == "ignored-b/secret.txt" {
			t.Errorf("M03 violation: ignored-b/secret.txt surfaced")
		}
	}
	if !sawA || !sawNested {
		t.Fatalf("missing expected ignored-a files: %+v", overlay.Files)
	}
}

// ----- M04: clean subject remains clean ----------------------------

func TestEvidence_Generate_M04_CleanRemainsClean(t *testing.T) {
	dir := makeEvidenceFixture(t)
	// No subject changes - the repo is clean.
	out, err := Generate(Options{
		RepoRoot:      dir,
		Mode:          ModeDirty,
		EvidencePaths: []string{"ignored-a"},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Subject manifest must remain empty: no A / M / ? lines.
	if strings.Contains(out, "## CHANGESET_MANIFEST\nA ") ||
		strings.Contains(out, "## CHANGESET_MANIFEST\nM ") ||
		strings.Contains(out, "## CHANGESET_MANIFEST\n? ") ||
		strings.Contains(out, "## CHANGESET_MANIFEST\nD ") {
		t.Errorf("M04 violation: subject classified changes when subject was clean")
	}
	if !strings.Contains(out, "## EVIDENCE_OVERLAY (experimental)") {
		t.Errorf("M04 violation: overlay section absent")
	}
}

// ----- M05: evidence has its own representation -------------------

func TestEvidence_Render_M05_SeparateRepresentation(t *testing.T) {
	dir := makeEvidenceFixture(t)
	out := RenderEvidenceOverlay(EvidenceOverlay{
		RepoRoot: dir,
		Roots:    []EvidenceRoot{{Path: "ignored-a"}},
		Files:    []EvidenceFile{{Path: "ignored-a/a.txt", Size: 11}},
	})
	want := []string{
		"## EVIDENCE_OVERLAY (experimental)",
		"repo_root:",
		"roots:",
		"- ignored-a",
		"### ignored-a/a.txt  size=11",
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("missing %q in:\n%s", w, out)
		}
	}
}

// ----- M06: .. escape rejected -------------------------------------

// TestEvidence_Validate_M06_DotDotEscape covers C5: any path
// segment equal to `..` is rejected, even when `filepath.Clean`
// would normalize inside the repository.
func TestEvidence_Validate_M06_DotDotEscape(t *testing.T) {
	dir := makeEvidenceFixture(t)
	cases := []string{"../foo", "a/../../etc", "..", "ignored-a/../etc"}
	for _, c := range cases {
		_, err := ValidateEvidenceRoots(dir, []string{c})
		if err == nil {
			t.Errorf("expected error for %q", c)
			continue
		}
		if !strings.Contains(err.Error(), "evidence:") {
			t.Errorf("error for %q missing prefix: %v", c, err)
		}
	}
}

// TestEvidence_Validate_C5_DotDotTableMatrix exhaustively
// enumerates the C5 policy: ANY segment equal to `..` rejects;
// filenames containing the literal `..` characters are
// accepted.
func TestEvidence_Validate_C5_DotDotTableMatrix(t *testing.T) {
	dir := makeEvidenceFixture(t)
	rejected := []string{
		"..",
		"../foo",
		"a/../b",
		"a/b/../../c",
		"ignored-a/../etc",
	}
	accepted := []string{
		"report..json",
		"build..",
		"..hidden",
		"a/..report/b",
		"..foo",
	}
	for _, c := range rejected {
		if _, err := ValidateEvidenceRoots(dir, []string{c}); err == nil {
			t.Errorf("C5: expected rejection for %q", c)
		}
	}
	for _, c := range accepted {
		roots, err := ValidateEvidenceRoots(dir, []string{c})
		if err != nil {
			t.Errorf("C5: unexpected rejection for %q: %v", c, err)
			continue
		}
		if len(roots) != 1 || roots[0].Rejected {
			t.Errorf("C5: filename rejected: %q -> %+v", c, roots)
		}
	}
}

// ----- M07: absolute escape rejected -------------------------------

func TestEvidence_Validate_M07_AbsoluteEscape(t *testing.T) {
	dir := makeEvidenceFixture(t)
	cases := []string{
		"/Users/example/.ssh",
		"/tmp/evidence",
		"\\\\server\\share",
	}
	for _, c := range cases {
		_, err := ValidateEvidenceRoots(dir, []string{c})
		if err == nil {
			t.Errorf("expected error for %q", c)
			continue
		}
		if !strings.Contains(err.Error(), "absolute") {
			t.Errorf("error for %q should mention absolute, got: %v", c, err)
		}
	}
}

// ----- M08: .git rejected -------------------------------------------

// TestEvidence_Validate_M08_DotGitRejected covers C4: only the
// `.git` metadata tree is rejected. `.gitignore` is an ordinary
// working-tree file and must NOT be rejected.
func TestEvidence_Validate_M08_DotGitRejected(t *testing.T) {
	dir := makeEvidenceFixture(t)
	for _, c := range []string{".git", ".git/", ".git/objects", ".git/config"} {
		_, err := ValidateEvidenceRoots(dir, []string{c})
		if err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

// TestEvidence_Validate_C4_DotGitIgnoreAccepted covers C4:
// `.gitignore` is an ordinary repository path, not Git metadata.
func TestEvidence_Validate_C4_DotGitIgnoreAccepted(t *testing.T) {
	dir := makeEvidenceFixture(t)
	roots, err := ValidateEvidenceRoots(dir, []string{".gitignore"})
	if err != nil {
		t.Fatalf("unexpected error for .gitignore: %v", err)
	}
	if len(roots) != 1 || roots[0].Rejected {
		t.Fatalf(".gitignore must NOT be classified as Git metadata: %+v", roots)
	}
}

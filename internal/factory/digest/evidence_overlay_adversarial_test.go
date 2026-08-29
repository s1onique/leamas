// SPDX-License-Identifier: Apache-2.0

// Adversarial tests for the CORRECTION02 safety properties.
// These tests must execute (the prior correction ACT was
// blocked by sandbox gates); the C7 timing-sensitive swap is
// deterministic by construction because we mutate the
// filesystem BEFORE invoking the renderer. The production
// binary itself relies on os.OpenRoot which uses platform
// openat semantics on Unix to prevent symlink escape.
package digest

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// makeReporootDir returns a hermetic repo root for C7 tests.
func makeReporootDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	mustGitNoErr(t, dir, "init", "-q", "--initial-branch=main")
	mustGitNoErr(t, dir, "config", "user.email", "test@example.com")
	mustGitNoErr(t, dir, "config", "user.name", "Test")
	mustGitNoErr(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	mustGitNoErr(t, dir, "add", "README")
	mustGitNoErr(t, dir, "commit", "-q", "-m", "seed")
	return dir, func() {}
}

// collectFor runs the production collect path against root.
func collectFor(t *testing.T, root, pathspec string) EvidenceOverlay {
	t.Helper()
	roots, err := ValidateEvidenceRoots(root, []string{pathspec})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(root, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return overlay
}

// ----- C7.1 -----

// TestEvidence_C7_FileSwapOutsideSymlink proves the renderer
// cannot be tricked into reading outside the authorized repo
// when the leaf file is swapped for an external symlink after
// collection.
func TestEvidence_C7_FileSwapOutsideSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)

	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	safePath := filepath.Join(dir, "ignored", "file.txt")
	if err := os.WriteFile(safePath, []byte("SAFE\n"), 0644); err != nil {
		t.Fatalf("write safe: %v", err)
	}

	overlay := collectFor(t, dir, "ignored")

	// Replace the file with a symlink to outside content.
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("OUTSIDE_SECRET\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.Remove(safePath); err != nil {
		t.Fatalf("remove safe: %v", err)
	}
	if err := os.Symlink(outsideFile, safePath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "OUTSIDE_SECRET") {
		t.Fatalf("C7.1 violation: renderer ingested outside content after leaf swap\n%s", rendered)
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "no such file") {
		t.Errorf("C7.1: renderer must surface visible failure\n%s", rendered)
	}
}

// ----- C7.2 -----

// TestEvidence_C7_AncestorSwapOutsideSymlink proves the
// renderer cannot be tricked when an ancestor directory is
// swapped to point outside the repo between collection and
// render.
func TestEvidence_C7_AncestorSwapOutsideSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)

	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored", "sub"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	safePath := filepath.Join(dir, "ignored", "sub", "file.txt")
	if err := os.WriteFile(safePath, []byte("SAFE\n"), 0644); err != nil {
		t.Fatalf("write safe: %v", err)
	}

	overlay := collectFor(t, dir, "ignored/sub")

	// Replace ignored/sub with a symlink to outside.
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"),
		[]byte("OUTSIDE_SUB_SECRET\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	subPath := filepath.Join(dir, "ignored", "sub")
	if err := os.Remove(subPath); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink(outside, subPath); err != nil {
		t.Fatalf("symlink ancestor: %v", err)
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "OUTSIDE_SUB_SECRET") {
		t.Fatalf("C7.2 violation: renderer ingested outside content after ancestor swap\n%s", rendered)
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "no such file") {
		t.Errorf("C7.2: renderer must surface visible failure\n%s", rendered)
	}
}

// ----- C7.3 -----

// TestEvidence_C7_CWDIndependencePreserved verifies that the
// traversal-resistant render path still binds to
// overlay.RepoRoot and never resolves through the caller cwd.
func TestEvidence_C7_CWDIndependencePreserved(t *testing.T) {
	repoA, _ := makeReporootDir(t)
	repoB := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoB, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoB, "ignored", "result.txt"),
		[]byte("FROM-B\n"), 0644); err != nil {
		t.Fatalf("write repoB: %v", err)
	}
	gi := filepath.Join(repoA, ".gitignore")
	if err := os.WriteFile(gi, []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoA, "ignored", "result.txt"),
		[]byte("FROM-A\n"), 0644); err != nil {
		t.Fatalf("write repoA: %v", err)
	}
	overlay := collectFor(t, repoA, "ignored")
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoB); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	rendered := RenderEvidenceOverlay(overlay)
	if !strings.Contains(rendered, "FROM-A") {
		t.Errorf("C7.3: missing FROM-A\n%s", rendered)
	}
	if strings.Contains(rendered, "FROM-B") {
		t.Errorf("C7.3: leaked FROM-B")
	}
}

// ----- C7.4 -----

// TestEvidence_C7_RegularFileRender proves the security fix
// does not break the normal evidence path.
func TestEvidence_C7_RegularFileRender(t *testing.T) {
	dir, _ := makeReporootDir(t)
	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "regular evidence body\n"
	if err := os.WriteFile(filepath.Join(dir, "ignored", "data.txt"),
		[]byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	overlay := collectFor(t, dir, "ignored")
	rendered := RenderEvidenceOverlay(overlay)
	if !strings.Contains(rendered, body) {
		t.Errorf("C7.4: regular file content missing\n%s", rendered)
	}
	if strings.Contains(rendered, "read_failed") ||
		strings.Contains(rendered, "(binary file)") {
		t.Errorf("C7.4: spurious failure / binary markers\n%s", rendered)
	}
}

// ----- C7.5 -----

// TestEvidence_C7_RaceFailureVisible proves that when the
// file disappears between collection and render, the renderer
// surfaces a visible failure rather than emitting silent
// empty content.
func TestEvidence_C7_RaceFailureVisible(t *testing.T) {
	dir, _ := makeReporootDir(t)
	gi := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gi, []byte("ignored/\n"), 0644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(dir, "ignored", "race.txt")
	if err := os.WriteFile(target, []byte("RACE_BODY\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	overlay := collectFor(t, dir, "ignored/race.txt")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "RACE_BODY") {
		t.Errorf("C7.5: deleted file's stale content rendered")
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "no such file") {
		t.Errorf("C7.5: visible failure missing\n%s", rendered)
	}
}

// ----- C8 typed sentinel classification -----

// TestEvidence_C8_SymlinkSentinel proves the collector uses
// errEvidenceSymlink and surfaces OmitSymlink without
// inspecting error strings.
func TestEvidence_C8_SymlinkSentinel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Symlink("/nonexistent-target",
		filepath.Join(dir, "ignored", "dead")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	// Cross-check: InspectSymlink must return true for the
	// raw error returned by inspectEvidenceFile.
	entry, rawErr := inspectEvidenceFile(dir, "ignored/dead")
	if entry.Size != 0 || rawErr == nil {
		t.Fatalf("inspectEvidenceFile did not return an error: %+v", entry)
	}
	if !InspectSymlink(rawErr) {
		t.Errorf("C8: InspectSymlink did not classify symlink error: %v", rawErr)
	}
	if !errors.Is(rawErr, errEvidenceSymlink) {
		t.Errorf("C8: errors.Is did not classify symlink: %v", rawErr)
	}
	// And the omission reason is the stable external code.
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored/dead" && om.Reason == OmitSymlink {
			found = true
		}
	}
	if !found {
		t.Errorf("C8: symlink omission missing: %+v", overlay.Omitted)
	}
}

// TestEvidence_C8_NonRegularSentinel proves non-regular file
// modes classify through the typed sentinel.
func TestEvidence_C8_NonRegularSentinel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := unixMkfifo(filepath.Join(dir, "ignored", "fifo"), 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	entry, rawErr := inspectEvidenceFile(dir, "ignored/fifo")
	if rawErr == nil {
		t.Fatalf("inspectEvidenceFile did not error for FIFO")
	}
	if !InspectNonRegular(rawErr) {
		t.Errorf("C8: InspectNonRegular did not classify FIFO: %v", rawErr)
	}
	if !errors.Is(rawErr, errEvidenceNonRegular) {
		t.Errorf("C8: errors.Is did not classify FIFO: %v", rawErr)
	}
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored/fifo" && om.Reason == OmitNonRegular {
			found = true
		}
	}
	if !found {
		t.Errorf("C8: FIFO omission missing: %+v", overlay.Omitted)
	}
}

// TestEvidence_C8_InspectFailedNonSentinel proves raw fs errors
// (e.g. permission, missing) are NOT classified as the
// symlink/non-regular sentinels; they flow through as plain
// wrapped errors.
func TestEvidence_C8_InspectFailedNonSentinel(t *testing.T) {
	dir, _ := makeReporootDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	entry, rawErr := inspectEvidenceFile(dir, "ignored/does-not-exist")
	if entry.Size != 0 || rawErr == nil {
		t.Fatalf("expected error for missing path, got %+v", entry)
	}
	if InspectSymlink(rawErr) || InspectNonRegular(rawErr) {
		t.Errorf("C8: InspectSymlink/NonRegular wrongly matched fs error: %v", rawErr)
	}
	if errors.Is(rawErr, errEvidenceSymlink) ||
		errors.Is(rawErr, errEvidenceNonRegular) {
		t.Errorf("C8: errors.Is wrongly matched fs error: %v", rawErr)
	}
}

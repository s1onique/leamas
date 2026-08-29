// SPDX-License-Identifier: Apache-2.0

// C7 adversarial tests for the local evidence overlay.
//
// The renderer is read-only. Critically, every preview read is
// opened through os.OpenRoot (the Go >= 1.24 standard-library
// traversal-resistant API) so a symlink or ancestor swap
// between collection and render cannot redirect the read
// outside the authorized repo root. The C7 tests prove this
// invariant deterministically by mutating the filesystem
// BEFORE invoking the renderer.
package digest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ----- C7.1 -----

// TestEvidence_C7_FileSwapOutsideSymlink proves the renderer
// cannot be tricked into reading outside the authorized repo
// when the leaf file is swapped for an external symlink after
// collection. C9: an unsafe control helper proves the
// laboratory can detect the leak the secure implementation
// prevents.
func TestEvidence_C7_FileSwapOutsideSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "ignored/")
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	safePath := filepath.Join(dir, "ignored", "file.txt")
	if err := os.WriteFile(safePath, []byte("SAFE\n"), 0644); err != nil {
		t.Fatalf("write safe: %v", err)
	}

	overlay := collectFor(t, dir, "ignored")

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

	// C9 control: an unsafe open sees the outside content.
	control, cErr := unsafeControlOpen(t, dir, "ignored/file.txt")
	if cErr != nil {
		t.Fatalf("unsafe control open failed (lab broken): %v", cErr)
	}
	if !strings.Contains(control, "OUTSIDE_SECRET") {
		t.Fatalf("C7.1 laboratory broken: unsafe control did not see canary")
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "OUTSIDE_SECRET") {
		t.Fatalf("C7.1 violation: renderer ingested outside content after leaf swap\n%s", rendered)
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "no such file") &&
		!strings.Contains(rendered, "object changed") {
		t.Errorf("C7.1: renderer must surface visible failure\n%s", rendered)
	}
}

// ----- C7.2 (C9: repair) -----

// TestEvidence_C7_AncestorSwapOutsideSymlink (C9 repair) proves
// the renderer cannot be tricked when an ancestor directory is
// swapped to point outside the repo between collection and
// render. The repair guarantees:
//
//   - the swap sequence is deterministic (file removed first,
//     then parent directory, then symlink created);
//   - the canary file lives at exactly the pathname the unsafe
//     implementation would traverse to (outside/file.txt);
//   - a test-only unsafe control helper proves the laboratory
//     can detect the leak the secure implementation prevents.
func TestEvidence_C7_AncestorSwapOutsideSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "ignored/")
	if err := os.MkdirAll(filepath.Join(dir, "ignored", "sub"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	safePath := filepath.Join(dir, "ignored", "sub", "file.txt")
	if err := os.WriteFile(safePath, []byte("SAFE\n"), 0644); err != nil {
		t.Fatalf("write safe: %v", err)
	}

	overlay := collectFor(t, dir, "ignored/sub")

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "file.txt"),
		[]byte("OUTSIDE_SUB_SECRET\n"), 0644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	// C9 repair: deterministic swap sequence.
	subPath := filepath.Join(dir, "ignored", "sub")
	if err := os.Remove(safePath); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	if err := os.Remove(subPath); err != nil {
		t.Fatalf("remove dir: %v", err)
	}
	if err := os.Symlink(outside, subPath); err != nil {
		t.Fatalf("symlink ancestor: %v", err)
	}

	// C9 control: unsafe open must see the canary.
	control, cErr := unsafeControlOpen(t, dir, "ignored/sub/file.txt")
	if cErr != nil {
		t.Fatalf("unsafe control open failed (lab broken): %v", cErr)
	}
	if !strings.Contains(control, "OUTSIDE_SUB_SECRET") {
		t.Fatalf("C9 unsafe control did not see canary: %q", control)
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "OUTSIDE_SUB_SECRET") {
		t.Fatalf("C7.2 violation: renderer ingested outside content after ancestor swap\n%s", rendered)
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "no such file") &&
		!strings.Contains(rendered, "object changed") {
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
	ignorePath(t, repoA, "ignored/")
	if err := os.MkdirAll(filepath.Join(repoA, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir repoA: %v", err)
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
	ignorePath(t, dir, "ignored/")
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
	ignorePath(t, dir, "ignored/")
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

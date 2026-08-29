// SPDX-License-Identifier: Apache-2.0

// C1/C3/C6 correction tests for the local evidence overlay MVP.
// The reviewer of the blocked Binary MVP01 surfaced these
// defects: repo-root binding (C1), silent omission (C3), and
// silent render read failures (C6).
package digest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ----- C1: repo-root-bound render ------------------------------------

// TestEvidence_C1_RepoRootBoundRender proves that evidence
// content is read relative to the overlay's RepoRoot and not
// to the caller's cwd. This is the central defect that the
// reviewer of the blocked Binary MVP01 surfaced.
//
// Two repos with identical relative paths are constructed. The
// renderer is invoked from inside repo-B, while the overlay
// belongs to repo-A. The rendered content MUST come from
// repo-A.
func TestEvidence_C1_RepoRootBoundRender(t *testing.T) {
	repoA := makeEvidenceFixture(t)
	repoB := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoB, "ignored-a"), 0755); err != nil {
		t.Fatalf("mkdir repoB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoB, "ignored-a", "a.txt"),
		[]byte("DECOY-FROM-REPO-B\n"), 0644); err != nil {
		t.Fatalf("write repoB decoy: %v", err)
	}
	roots, err := ValidateEvidenceRoots(repoA, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate repoA: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(repoA, roots)
	if err != nil {
		t.Fatalf("collect repoA: %v", err)
	}

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repoB); err != nil {
		t.Fatalf("chdir repoB: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	rendered := RenderEvidenceOverlay(overlay)
	if !strings.Contains(rendered, "ignored a\n") {
		t.Errorf("C1 violation: rendered overlay does not contain repo-A evidence content\n%s", rendered)
	}
	if strings.Contains(rendered, "DECOY-FROM-REPO-B") {
		t.Errorf("C1 violation: renderer ingested decoy content from cwd")
	}
	if !strings.Contains(rendered, "repo_root:") {
		t.Errorf("C1: rendered overlay must surface repo_root for traceability")
	}
}

// TestEvidence_CWDIndependence is the formal acceptance gate
// for the central defect.
func TestEvidence_CWDIndependence(t *testing.T) {
	repoA := makeEvidenceFixture(t)
	repoB := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoB, "ignored"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoB, "ignored", "result.txt"),
		[]byte("FROM-B\n"), 0644); err != nil {
		t.Fatalf("write repoB: %v", err)
	}
	gi := filepath.Join(repoA, ".gitignore")
	b, _ := os.ReadFile(gi)
	if err := os.WriteFile(gi, append(b, []byte("ignored/\n")...), 0644); err != nil {
		t.Fatalf("write gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoA, "ignored", "result.txt"),
		[]byte("FROM-A\n"), 0644); err != nil {
		t.Fatalf("write repoA: %v", err)
	}
	roots, err := ValidateEvidenceRoots(repoA, []string{"ignored"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(repoA, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
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
		t.Errorf("CWD_INDEPENDENCE: missing FROM-A\n%s", rendered)
	}
	if strings.Contains(rendered, "FROM-B") {
		t.Errorf("CWD_INDEPENDENCE: leaked FROM-B")
	}
}

// ----- C3: deterministic omission accounting ------------------------

// TestEvidence_C3_OversizeFileRecorded creates an ignored file
// larger than MaxEvidencePerFileBytes and verifies it appears in
// the omissions section with reason "file_too_large".
func TestEvidence_C3_OversizeFileRecorded(t *testing.T) {
	dir := makeEvidenceFixture(t)
	big := make([]byte, MaxEvidencePerFileBytes+1)
	for i := range big {
		big[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored-a", "big.bin"),
		big, 0644); err != nil {
		t.Fatalf("write big: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored-a/big.bin" && om.Reason == OmitFileTooLarge {
			found = true
		}
	}
	if !found {
		t.Errorf("C3 violation: oversize file not recorded: %+v", overlay.Omitted)
	}
}

// TestEvidence_C3_SpecialFileRecorded creates a FIFO under an
// ignored directory and verifies the collector records it as a
// non-regular omission rather than silently dropping it.
func TestEvidence_C3_SpecialFileRecorded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO semantics differ on Windows")
	}
	dir := makeEvidenceFixture(t)
	if err := unixMkfifo(filepath.Join(dir, "ignored-a", "fifo"), 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored-a/fifo" && om.Reason == OmitNonRegular {
			found = true
		}
	}
	if !found {
		t.Errorf("C3 violation: FIFO not recorded: %+v", overlay.Omitted)
	}
}

// TestEvidence_C3_SymlinkFileRecorded proves that symlinks
// discovered at inspect time are recorded with reason
// "symlink" rather than disappearing.
func TestEvidence_C3_SymlinkFileRecorded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir := makeEvidenceFixture(t)
	if err := os.Symlink("/nonexistent-target",
		filepath.Join(dir, "ignored-a", "dead")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	found := false
	for _, om := range overlay.Omitted {
		if om.Path == "ignored-a/dead" && om.Reason == OmitSymlink {
			found = true
		}
	}
	if !found {
		t.Errorf("C3 violation: symlink not recorded: %+v", overlay.Omitted)
	}
}

// ----- C6: render read failure must be visible ---------------------

// TestEvidence_C6_RenderReadFailureVisible deletes a file
// between collection and rendering; the renderer must surface
// the failure rather than emitting silent empty content.
func TestEvidence_C6_RenderReadFailureVisible(t *testing.T) {
	dir := makeEvidenceFixture(t)
	target := filepath.Join(dir, "ignored-a", "a.txt")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.WriteFile(target, []byte("post-delete-content\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	roots, err := ValidateEvidenceRoots(dir, []string{"ignored-a/a.txt"})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	overlay, err := CollectEvidenceOverlay(dir, roots)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove again: %v", err)
	}
	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "post-delete-content") {
		t.Errorf("C6 violation: deleted file's stale content was rendered")
	}
	if !strings.Contains(rendered, "read_failed") {
		t.Errorf("C6 violation: render did not surface read failure marker\n%s", rendered)
	}
}

// unixMkfifo creates a FIFO without importing the unix package.
func unixMkfifo(path string, mode uint32) error {
	cmd := exec.Command("mkfifo", path)
	if err := cmd.Run(); err != nil {
		return err
	}
	return os.Chmod(path, os.FileMode(mode))
}

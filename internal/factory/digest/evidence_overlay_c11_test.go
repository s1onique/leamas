// SPDX-License-Identifier: Apache-2.0

// C11 object-identity tests for the local evidence overlay.
//
// os.Root prevents reads from escaping overlay.RepoRoot but
// does NOT prevent symlink substitution that stays inside
// the root. To close that gap, the collector hands the Lstat
// os.FileInfo to the renderer, which verifies at read time
// (via os.SameFile) that the opened filesystem object is
// still the same object that was authorized at collect time.
//
// In-place content mutation of the SAME inode is explicitly
// deferred (IN_PLACE_CONTENT_MUTATION_ATOMICITY=DEFERRED);
// this mechanism guarantees pathname / object substitution
// cannot silently redirect the read, not byte-level
// atomicity.
package digest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestEvidence_C11_InternalSymlinkSubstitutionBlocked proves
// that even when a swap target stays inside the repository,
// the renderer rejects the substitution because the opened
// object is no longer the same os.FileInfo captured at
// collection time.
func TestEvidence_C11_InternalSymlinkSubstitutionBlocked(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "evidence/")
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0755); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "private"), 0755); err != nil {
		t.Fatalf("mkdir private: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "evidence", "result.txt"),
		[]byte("ORIGINAL\n"), 0644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "private", "secret.txt"),
		[]byte("INTERNAL_SECRET\n"), 0644); err != nil {
		t.Fatalf("write private: %v", err)
	}

	overlay := collectFor(t, dir, "evidence")

	// Substitute the evidence leaf with a same-repo symlink
	// to a different ignored file.
	target := filepath.Join(dir, "evidence", "result.txt")
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.Symlink(
		filepath.Join(dir, "private", "secret.txt"),
		target,
	); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// Control: unsafe open must see the canary so the
	// laboratory can detect the substitution.
	control, cErr := unsafeControlOpen(t, dir, "evidence/result.txt")
	if cErr != nil {
		t.Fatalf("unsafe control: %v", cErr)
	}
	if !strings.Contains(control, "INTERNAL_SECRET") {
		t.Fatalf("C11 lab broken: unsafe control did not see canary: %q", control)
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "INTERNAL_SECRET") {
		t.Fatalf("C11 violation: renderer ingested substituted object\n%s", rendered)
	}
	if strings.Contains(rendered, "ORIGINAL") {
		t.Errorf("C11: original content rendered after substitution")
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "object changed") {
		t.Errorf("C11: visible failure missing\n%s", rendered)
	}
}

// TestEvidence_C11_RegularFileSubstitutionBlocked proves the
// renderer rejects when the same pathname is recreated with
// a different inode after collection.
func TestEvidence_C11_RegularFileSubstitutionBlocked(t *testing.T) {
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "evidence/")
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0755); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}
	target := filepath.Join(dir, "evidence", "result.txt")
	if err := os.WriteFile(target, []byte("ORIGINAL\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	overlay := collectFor(t, dir, "evidence")

	// Recreate the file with a fresh inode.
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := os.WriteFile(target, []byte("SUBSTITUTE\n"), 0644); err != nil {
		t.Fatalf("write substitute: %v", err)
	}

	// Control: unsafe open must see the substitute so the
	// laboratory can detect the substitution.
	control, cErr := unsafeControlOpen(t, dir, "evidence/result.txt")
	if cErr != nil {
		t.Fatalf("unsafe control: %v", cErr)
	}
	if !strings.Contains(control, "SUBSTITUTE") {
		t.Fatalf("C11 lab broken: unsafe control did not see canary: %q", control)
	}

	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "SUBSTITUTE") {
		t.Fatalf("C11 violation: renderer ingested substitute file\n%s", rendered)
	}
	if strings.Contains(rendered, "ORIGINAL") {
		t.Errorf("C11: original content rendered after substitution")
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "object changed") {
		t.Errorf("C11: visible failure missing\n%s", rendered)
	}
}

// TestEvidence_C11_SpecialFileSubstitutionDoesNotRender proves
// that when a collected regular file is replaced by a FIFO
// before rendering, the renderer does not block waiting for
// data and does not render content. The mechanism is: the
// opened file's Stat reports a different mode (FIFO) and a
// different inode, so os.SameFile fails closed before any
// blocking read is attempted.
func TestEvidence_C11_SpecialFileSubstitutionDoesNotRender(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FIFO semantics differ on Windows")
	}
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "evidence/")
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0755); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}
	target := filepath.Join(dir, "evidence", "result.txt")
	if err := os.WriteFile(target, []byte("ORIGINAL\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	overlay := collectFor(t, dir, "evidence")

	// Replace with FIFO.
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := unixMkfifo(target, 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}

	// Render must not block. If the implementation attempted
	// to open the FIFO without O_NONBLOCK the test would hang
	// indefinitely, which is also a CI failure, but we
	// additionally enforce a watchdog so the test outcome is
	// explicit.
	done := make(chan string, 1)
	go func() {
		done <- RenderEvidenceOverlay(overlay)
	}()
	select {
	case rendered := <-done:
		if !strings.Contains(rendered, "read_failed") &&
			!strings.Contains(rendered, "object changed") {
			t.Errorf("C11: visible failure missing for FIFO substitution\n%s", rendered)
		}
		if strings.Contains(rendered, "ORIGINAL") {
			t.Errorf("C11: original content rendered after FIFO substitution")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("C11 violation: render blocked on FIFO substitution")
	}
}

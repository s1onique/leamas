// SPDX-License-Identifier: Apache-2.0

// C12 nonblocking safe-open tests for the local evidence
// overlay.
//
// A pathname that was collected as a regular evidence file
// but is later replaced by a FIFO (or other special object)
// MUST NOT block the render thread during open. The
// production implementation must:
//
//   - open with O_NONBLOCK on Unix so the open() syscall
//     returns immediately even when the FIFO has no writer;
//   - Stat the opened file and reject non-regular objects
//     before any read;
//   - emit a visible failure rather than rendering content.
//
// The outer watchdog is catastrophic protection only; the
// production call itself must return within a tight inner
// budget that would not be reachable if the implementation
// used plain O_RDONLY on a FIFO with no writer.
package digest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestEvidence_C12_FIFOOpenNonblocking proves the production
// safe-open returns promptly on a FIFO substitution. The test
// uses a tight inner budget (1s) — long enough for a
// nonblocking open + stat to fail-closed, but shorter than
// any plausible blocking-open scenario. A catastrophic outer
// watchdog (5s) remains as last-resort protection.
func TestEvidence_C12_FIFOOpenNonblocking(t *testing.T) {
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
	originalInfo, sErr := os.Lstat(target)
	if sErr != nil {
		t.Fatalf("lstat original: %v", sErr)
	}

	// Replace with FIFO.
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := unixMkfifo(target, 0644); err != nil {
		t.Skipf("mkfifo unsupported: %v", err)
	}
	defer os.Remove(target) // best-effort cleanup

	// C12: assert the production safe-open returns within a
	// tight budget. The previous implementation hung
	// indefinitely on a plain root.Open of a FIFO with no
	// writer; the budget here is intentionally much shorter
	// than that hang.
	const innerBudget = 1 * time.Second
	done := make(chan struct{})
	var renderErr error
	var rendered string
	go func() {
		defer close(done)
		rendered = RenderEvidenceOverlay(overlay)
	}()
	select {
	case <-done:
	case <-time.After(innerBudget):
		// Outer catastrophic protection: if the inner budget
		// is exceeded, the production call is blocking. The
		// process is left to continue so cleanup runs, but
		// the test fails immediately.
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = renderErr
			t.Fatalf("C12 violation: render blocked on FIFO substitution beyond catastrophic watchdog")
		}
		t.Fatalf("C12 violation: render blocked on FIFO substitution (inner budget %v exceeded)", innerBudget)
	}

	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "not a regular file") &&
		!strings.Contains(rendered, "object changed") {
		t.Errorf("C12: visible failure missing for FIFO substitution\n%s", rendered)
	}
	if strings.Contains(rendered, "ORIGINAL") {
		t.Errorf("C12: original content rendered after FIFO substitution")
	}
	if originalInfo != nil {
		// sanity: original was a regular file (the test
		// should never be confused about which object was
		// authorized).
		if !originalInfo.Mode().IsRegular() {
			t.Fatalf("C12 lab broken: original was not regular")
		}
	}
}

// TestEvidence_C12_RegularFileRender proves the nonblocking
// open does not regress ordinary evidence rendering.
func TestEvidence_C12_RegularFileRender(t *testing.T) {
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "evidence/")
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := "regular evidence body\n"
	if err := os.WriteFile(filepath.Join(dir, "evidence", "data.txt"),
		[]byte(body), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	overlay := collectFor(t, dir, "evidence")
	rendered := RenderEvidenceOverlay(overlay)
	if !strings.Contains(rendered, body) {
		t.Errorf("C12 regression: regular file content missing\n%s", rendered)
	}
	if strings.Contains(rendered, "read_failed") ||
		strings.Contains(rendered, "not a regular file") {
		t.Errorf("C12 regression: spurious failure on regular file\n%s", rendered)
	}
}

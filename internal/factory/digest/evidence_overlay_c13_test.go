// SPDX-License-Identifier: Apache-2.0

// C13 identity-hardened substitution tests for the local
// evidence overlay.
//
// The prior C11 laboratory recreated a regular file at the
// same pathname after collection. Theoretically, the kernel
// could reuse the freed inode, causing os.SameFile to return
// true even though the bytes differ. The C13 laboratory:
//
//   - creates both ORIGINAL and SUBSTITUTE at distinct
//     pathnames BEFORE collection;
//   - asserts the two os.FileInfo entries are NOT the same
//     filesystem object (negative SameFile) BEFORE the swap;
//   - atomically renames SUBSTITUTE over the target so the
//     kernel cannot reuse the original inode;
//   - proves the unsafe control sees SUBSTITUTE;
//   - proves the secure renderer does not.
//
// In-place content mutation of the SAME inode remains
// explicitly deferred.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEvidence_C13_SubstituteIdentityDistinctHardened is the
// C13 identity-hardened substitution adversary. It uses a
// rename-based atomic replacement so the kernel cannot
// recycle the original inode.
func TestEvidence_C13_SubstituteIdentityDistinctHardened(t *testing.T) {
	dir, _ := makeReporootDir(t)
	ignorePath(t, dir, "evidence/")
	if err := os.MkdirAll(filepath.Join(dir, "evidence"), 0755); err != nil {
		t.Fatalf("mkdir evidence: %v", err)
	}

	// Create ORIGINAL and SUBSTITUTE at distinct pathnames
	// before collection.
	target := filepath.Join(dir, "evidence", "result.txt")
	substitute := filepath.Join(dir, "evidence", ".result.substitute")
	if err := os.WriteFile(target, []byte("ORIGINAL\n"), 0644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := os.WriteFile(substitute, []byte("SUBSTITUTE\n"), 0644); err != nil {
		t.Fatalf("write substitute: %v", err)
	}

	originalInfo, oErr := os.Lstat(target)
	if oErr != nil {
		t.Fatalf("lstat original: %v", oErr)
	}
	substituteInfo, sErr := os.Lstat(substitute)
	if sErr != nil {
		t.Fatalf("lstat substitute: %v", sErr)
	}

	// C13 precondition: the two objects must NOT be the same
	// filesystem object before any swap. This rules out an
	// accidental inode reuse by the kernel and ensures the
	// laboratory can detect the substitution.
	if os.SameFile(originalInfo, substituteInfo) {
		t.Fatalf("C13 lab broken: original and substitute share an inode")
	}

	overlay := collectFor(t, dir, "evidence/result.txt")

	// Atomic rename over the target. The original inode
	// remains allocated for the duration of rename(2) on the
	// same filesystem.
	if err := os.Rename(substitute, target); err != nil {
		t.Fatalf("rename substitute: %v", err)
	}

	// Unsafe control must see SUBSTITUTE so the laboratory
	// can detect the substitution.
	control, cErr := unsafeControlOpen(t, dir, "evidence/result.txt")
	if cErr != nil {
		t.Fatalf("unsafe control: %v", cErr)
	}
	if !strings.Contains(control, "SUBSTITUTE") {
		t.Fatalf("C13 unsafe control did not see substitute: %q", control)
	}

	// Secure renderer must reject.
	rendered := RenderEvidenceOverlay(overlay)
	if strings.Contains(rendered, "SUBSTITUTE") {
		t.Fatalf("C13 violation: renderer ingested substitute file\n%s", rendered)
	}
	if strings.Contains(rendered, "ORIGINAL") {
		t.Errorf("C13: original content rendered after rename substitution")
	}
	if !strings.Contains(rendered, "read_failed") &&
		!strings.Contains(rendered, "object changed") {
		t.Errorf("C13: visible failure missing\n%s", rendered)
	}
}

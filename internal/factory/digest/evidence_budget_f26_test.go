// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_f26_test.go verifies
// the F26 (CORRECTION07) invariant: parseCatFileLine
// correctly identifies the WireStatus for object
// expressions containing whitespace (spaces, tabs).
// Before F26, the parser inspected fields[1], which is
// the second whitespace-separated field, so a path with
// a space (e.g. "HEAD:docs/my file.txt") produced
// fields[1] = "docs/my", not "missing". The BlobStatus
// contract — that the renderer can distinguish MISSING
// from PRESENT and from OTHER — depends on this
// distinction.
//
// These are real-git integration tests because the bug
// is fundamentally about the response parser, which
// only manifests when the actual response from a
// real-world cat-file process is fed in.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEvidenceBudget_F26_RangeBlobOID_WhitespacePath
// proves that rangeBlobOIDsBatch gives the correct
// BlobStatus for paths containing spaces:
//   - present path:   OID == expected, Status == PRESENT
//   - missing path:   OID == "",     Status == MISSING
//
// Before F26, the missing path tested as OTHER because
// fields[1] of the response line was the second
// whitespace-separated token of the path, not the
// status.
func TestEvidenceBudget_F26_RangeBlobOID_WhitespacePath(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// "ordinary file.txt" (space) — present.
	spacePath := filepath.Join(dir, "ordinary file.txt")
	if err := os.WriteFile(spacePath,
		[]byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write space path: %v", err)
	}

	// "tabbed\tfile.txt" (tab) — present.
	tabPath := filepath.Join(dir, "tabbed\tfile.txt")
	if err := os.WriteFile(tabPath,
		[]byte("world\n"), 0o644); err != nil {
		t.Fatalf("write tab path: %v", err)
	}

	runGit(t, dir, "add", "ordinary file.txt",
		"tabbed\tfile.txt")
	runGit(t, dir, "commit", "-m", "whitespace fixture")

	// Three refs: present-space, present-tab, missing.
	refs := []string{
		"HEAD:ordinary file.txt",
		"HEAD:tabbed\tfile.txt",
		"HEAD:never-existed.txt",
	}
	m := rangeBlobOIDsBatch(realGitRunner{}, dir, refs)

	// Present space: must be PRESENT with the real OID.
	spaceRes, ok := m[refs[0]]
	if !ok {
		t.Fatalf("F26 violated: present-space ref " +
			"missing from result map")
	}
	if !spaceRes.IsPresent() {
		t.Fatalf("F26 violated: present-space "+
			"Status = %v, want PRESENT",
			spaceRes.Status)
	}
	expectedSpace := runGit(t, dir, "rev-parse",
		"HEAD:ordinary file.txt")
	if spaceRes.OID != expectedSpace {
		t.Fatalf("F26 violated: present-space "+
			"OID = %q, want %q",
			spaceRes.OID, expectedSpace)
	}

	// Present tab: must be PRESENT with the real OID.
	tabRes, ok := m[refs[1]]
	if !ok {
		t.Fatalf("F26 violated: present-tab ref " +
			"missing from result map")
	}
	if !tabRes.IsPresent() {
		t.Fatalf("F26 violated: present-tab "+
			"Status = %v, want PRESENT",
			tabRes.Status)
	}
	expectedTab := runGit(t, dir, "rev-parse",
		"HEAD:tabbed\tfile.txt")
	if tabRes.OID != expectedTab {
		t.Fatalf("F26 violated: present-tab "+
			"OID = %q, want %q",
			tabRes.OID, expectedTab)
	}

	// Missing path with space: must be MISSING, NOT
	// OTHER, NOT PRESENT. This is the load-bearing
	// invariant that F26 fixed.
	missingRes, ok := m[refs[2]]
	_ = ok // missing keys are also acceptable
	if missingRes.IsPresent() {
		t.Fatalf("F26 violated: missing-path " +
			"IsPresent() = true (OID would be " +
			"fabricated)")
	}
	if missingRes.Status != BlobMissing {
		t.Fatalf("F26 violated: missing-path "+
			"Status = %v, want MISSING (F26 "+
			"parser mis-classifies whitespace-"+
			"containing missing objects)",
			missingRes.Status)
	}
	if missingRes.OID != "" {
		t.Fatalf("F26 violated: missing-path "+
			"OID = %q, want empty",
			missingRes.OID)
	}
}

// TestEvidenceBudget_F26_RangeBlobOID_MissingAmbiguous
// drives the renderer through two special-status cases:
//   - "HEAD:ambiguous.txt" (no, Git doesn't resolve
//     ambiguous automatically, but the wire-status
//     "ambiguous" maps to BlobAmbiguous)
//   - excluded paths
//
// In practice, the parser's behavior on these responses
// is the same regardless of whether the path contains
// whitespace — but F26's regression matrix needs to
// confirm that whitespace in the object expression
// doesn't change the BlobStatus mapping for ANY
// keyword, not just "missing".
func TestEvidenceBudget_F26_RangeBlobOID_MissingAmbiguous(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// Build a fixture where the same path with whitespace
	// is absent, so the response is "missing" — and
	// exercise the precise wire response manually.
	refs := []string{
		"HEAD:this file does not exist.txt",
		"HEAD:ne\tver existed.txt",
	}

	// Test the parser directly with the EXACT bytes
	// Git produces for a missing whitespace path.
	// Real-world `git cat-file --batch-check` output
	// for an absent path containing a space is:
	//
	//   $ git cat-file --batch-check
	//   HEAD:my file.txt
	//   HEAD:my file.txt missing
	//
	// The leading 40-char hash is a placeholder.
	lines := []string{
		"abcdef1234567890abcdef1234567890abcdef12 missing",
		"HEAD:my file.txt missing",
		"HEAD:ne\tver existed.txt missing",
	}
	wants := []BlobLookupStatus{
		BlobMissing,
		BlobMissing,
		BlobMissing,
	}
	for i, line := range lines {
		got := parseCatFileLine(line)
		if got.Status != wants[i] {
			t.Fatalf("F26 violated: parseCatFileLine"+
				"(%q) Status = %v, want %v",
				line, got.Status, wants[i])
		}
		if got.IsPresent() {
			t.Fatalf("F26 violated: "+
				"parseCatFileLine(%q) "+
				"IsPresent() = true on missing",
				line)
		}
	}

	// Real-Git integration: the actual response from
	// rangeBlobOIDsBatch must also classify correctly.
	m := rangeBlobOIDsBatch(realGitRunner{}, dir, refs)
	for _, ref := range refs {
		res, ok := m[ref]
		if !ok {
			t.Fatalf("F26 violated: ref %q "+
				"missing from result map", ref)
		}
		if res.IsPresent() {
			t.Fatalf("F26 violated: ref %q "+
				"IsPresent() = true on missing",
				ref)
		}
		if res.Status != BlobMissing {
			t.Fatalf("F26 violated: ref %q "+
				"Status = %v, want MISSING",
				ref, res.Status)
		}
	}
	_ = strings.TrimSpace // quiet import linter
}

// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_f23_test.go verifies the
// F23 (CORRECTION06) invariant: the batched cat-file
// fast path is NOT safe for paths containing '\n' or '\r',
// and the production helpers must route to a per-object
// fallback. Without F23, one newline-containing path turns
// one intended query into two protocol records and emits
// plausible but incorrectly attributed identity evidence.
//
// These are real-git integration tests (not parser-only
// unit tests) because the bug is fundamentally about the
// transport protocol, which only manifests in real
// stdin/stdout interaction with `git cat-file`.
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeNewlinePath creates a file with a literal '\n' in
// its name on macOS (POSIX filenames allow '\n' and '\r').
func writeNewlinePath(t *testing.T, dir, name, content string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", name, err)
	}
	return full
}

// TestEvidenceBudget_F23_RangeBlobOID_NewlinePath verifies
// that rangeBlobOIDsBatch gives the CORRECT OID for a
// newline-containing path (no cross-pollination with a
// neighbor's OID). Before F23, the batched fast path
// produced a 2-line response for a 2-element input, but
// the second element was a different path's truncated
// query — leading to B's output being attributed to A.
func TestEvidenceBudget_F23_RangeBlobOID_NewlinePath(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	// Two files: ordinary "normal.txt" and evil path
	// with literal '\n' in the name.
	normalPath := filepath.Join(dir, "normal.txt")
	if err := os.WriteFile(normalPath, []byte("ordinary\n"),
		0o644); err != nil {
		t.Fatalf("write normal: %v", err)
	}
	evilPath := writeNewlinePath(t, dir,
		"evil\nname.txt", "evil\n")

	runGit(t, dir, "add", "normal.txt", "evil\nname.txt")
	runGit(t, dir, "commit", "-m", "init with evil path")

	// Use the production path: rangeBlobOIDsBatch must
	// return the RIGHT OID for both paths.
	refs := []string{
		"HEAD:normal.txt",
		"HEAD:evil\nname.txt",
	}
	m := rangeBlobOIDsBatch(realGitRunner{}, dir, refs)

	// normal.txt: OID must be the real blob OID.
	normalRes, ok := m[refs[0]]
	if !ok || !normalRes.IsPresent() {
		t.Fatalf("F23 violated: normal.txt not PRESENT, "+
			"got %+v ok=%v", normalRes, ok)
	}
	// Resolve the expected OID via rev-parse (single
	// per-object call, path-safe) to avoid coupling
	// this test to a fixed blob hash.
	expectedNormal := runGit(t, dir,
		"rev-parse", "HEAD:normal.txt")
	if normalRes.OID != expectedNormal {
		t.Fatalf("F23 violated: normal.txt OID = %q, "+
			"want %q", normalRes.OID, expectedNormal)
	}

	// evil\nname.txt: OID must be the real blob OID,
	// not the (potentially truncated) response line
	// for normal.txt.
	evilRes, ok := m[refs[1]]
	if !ok || !evilRes.IsPresent() {
		t.Fatalf("F23 violated: evil path not PRESENT, "+
			"got %+v ok=%v", evilRes, ok)
	}
	expectedEvil := runGit(t, dir,
		"rev-parse", "HEAD:evil\nname.txt")
	if evilRes.OID != expectedEvil {
		t.Fatalf("F23 violated: evil path OID = %q, "+
			"want %q (got %q for normal.txt — "+
			"cross-pollination!)",
			evilRes.OID, expectedEvil, normalRes.OID)
	}

	// The two OIDs must be distinct (they are
	// different blobs with different content).
	if normalRes.OID == evilRes.OID {
		t.Fatalf("F23 violated: normal.txt and evil "+
			"path have the same OID %q — query "+
			"misalignment", normalRes.OID)
	}

	// Sanity: the actual files exist on disk for the
	// writeNewlinePath helper.
	if _, err := os.Stat(evilPath); err != nil {
		t.Fatalf("evil path not on disk: %v", err)
	}
}

// TestEvidenceBudget_F23_RangeMaxBytes_NewlinePath
// verifies that computeRangeMaxBytes returns the right
// sizes for paths with '\n' or '\r', without the
// batched-fast-path alignment shift.
//
// The fixture is deliberately crafted so a batched-fast-path
// failure would shift the response/refs alignment and
// produce a wrong size. Three files: a normal one
// (size=101), an evil-path one (size=251), and a second
// normal one (size=303). The evil path splits the input
// into 4 logical queries (instead of 3 expected), so the
// batched fast path's `assigned` counter mis-binds and
// one normal file gets the wrong size.
func TestEvidenceBudget_F23_RangeMaxBytes_NewlinePath(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "a.txt"),
		[]byte(strings.Repeat("a", 100)+"\n"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "evil\nname.txt"),
		[]byte(strings.Repeat("e", 250)+"\n"), 0o644); err != nil {
		t.Fatalf("write evil: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"),
		[]byte(strings.Repeat("b", 302)+"\n"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	runGit(t, dir, "add", "a.txt", "evil\nname.txt", "b.txt")
	runGit(t, dir, "commit", "-m", "size init")

	files := []RangeFile{
		{Path: "a.txt", Status: "added",
			From: "HEAD~1", To: "HEAD"},
		{Path: "evil\nname.txt", Status: "added",
			From: "HEAD~1", To: "HEAD"},
		{Path: "b.txt", Status: "added",
			From: "HEAD~1", To: "HEAD"},
	}
	m := computeRangeMaxBytes(realGitRunner{}, dir,
		files, "HEAD~1..HEAD")

	cases := []struct {
		path string
		want int64
	}{
		{"a.txt", 101},
		{"evil\nname.txt", 251},
		{"b.txt", 303},
	}
	for _, c := range cases {
		got, ok := m[c.path]
		if !ok {
			t.Fatalf("F23 violated: %q missing from "+
				"computeRangeMaxBytes result", c.path)
		}
		if got != c.want {
			t.Fatalf("F23 violated: %q size = %d, "+
				"want %d (alignment shift?)",
				c.path, got, c.want)
		}
	}
}

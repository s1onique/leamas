// Package dupcode provides tests for the baseline path-normalization contract.
//
// These tests pin the observable behavior of NormalizeOccurrencePath, in
// particular the path-containment check. A prefix-based ".." guard rejects
// legitimate local paths whose names happen to start with ".."
// (e.g. "..generated.go"); the contract must use a true path-component
// containment check (filepath.IsLocal) so escape is detected without
// false positives.
package dupcode

import (
	"path/filepath"
	"testing"
)

// TestNormalizeOccurrencePath_LocalPaths asserts that paths that stay
// inside the root are returned as fixture-root-relative, slash-normalized.
func TestNormalizeOccurrencePath_LocalPaths(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "repo")
	nested := filepath.Join(root, "subdir")

	cases := []struct {
		name string
		root string
		p    string
		want string
	}{
		{
			name: "sibling",
			root: root,
			p:    filepath.Join(root, "a.go"),
			want: "a.go",
		},
		{
			name: "nested",
			root: root,
			p:    filepath.Join(root, "subdir", "b.go"),
			want: filepath.ToSlash(filepath.Join("subdir", "b.go")),
		},
		{
			name: "deeper nesting",
			root: root,
			p:    filepath.Join(nested, "deep", "c.go"),
			want: filepath.ToSlash(filepath.Join("subdir", "deep", "c.go")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeOccurrencePath(tc.root, tc.p)
			if got != tc.want {
				t.Errorf("NormalizeOccurrencePath(%q, %q) = %q, want %q",
					tc.root, tc.p, got, tc.want)
			}
		})
	}
}

// TestNormalizeOccurrencePath_LocalNameStartingWithDotDot asserts that a
// filename whose name starts with ".." but does NOT contain a parent
// directory component is treated as a legitimate local path.
//
// This is the regression case for the prefix-based guard
// `strings.HasPrefix(rel, "..")`, which incorrectly rejects any string
// beginning with ".." regardless of whether it is a path component.
func TestNormalizeOccurrencePath_LocalNameStartingWithDotDot(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "repo")

	// A file named "..generated.go" must be accepted as a local path.
	// filepath.IsLocal returns true for "..generated.go" because it is
	// a single local name (no parent-directory component).
	p := filepath.Join(root, "..generated.go")
	got := NormalizeOccurrencePath(root, p)
	want := "..generated.go"
	if got != want {
		t.Errorf("NormalizeOccurrencePath(%q, %q) = %q, want %q "+
			"(local file whose name starts with \"..\" must be accepted)",
			root, p, got, want)
	}
}

// TestNormalizeOccurrencePath_OutsideRootFallback asserts the exact
// fallback behavior when the input path cannot be represented as a local
// path beneath root:
//
//   - filepath.Rel returns a "../..." form for sibling/sibling-of-ancestor
//     paths, which filepath.IsLocal rejects.
//   - filepath.Rel returns an absolute path when the inputs live on
//     different volumes (Windows) or the relative computation cannot be
//     performed.
//
// In either case the function falls back to filepath.ToSlash(p), the
// slash-normalized original. The test asserts the exact output rather than
// just "not local" so that any silent change to the fallback shape (e.g.
// dropping the slash normalization, returning an empty string, returning
// a different normalization) is caught.
func TestNormalizeOccurrencePath_OutsideRootFallback(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "repo")
	otherDir := filepath.Join(tmpDir, "outside")

	cases := []struct {
		name string
		root string
		p    string
	}{
		{
			name: "sibling-of-root (Rel returns ../...)",
			root: root,
			p:    filepath.Join(otherDir, "b.go"),
		},
		{
			name: "absolute path outside root",
			root: root,
			p:    filepath.Join(tmpDir, "elsewhere", "c.go"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeOccurrencePath(tc.root, tc.p)
			// Documented fallback: slash-normalized original.
			want := filepath.ToSlash(tc.p)
			if got != want {
				t.Errorf("NormalizeOccurrencePath(%q, %q) = %q, want fallback %q",
					tc.root, tc.p, got, want)
			}
		})
	}
}
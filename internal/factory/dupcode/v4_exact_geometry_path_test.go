// Package dupcode provides exact geometry contract tests for the V4 algorithm.
//
// This file groups the path-projector contract tests. It exercises the
// documented edge contracts of normalizeFixturePath directly with a
// table-driven test.
//
// Where platform behavior differs, assertions are portable. No
// symlink-containment claims are made; filepath.IsLocal is lexical only.
//
// Sibling files in this contract group:
//
//   - v4_exact_geometry_support_test.go (normalizeFixturePath definition)
//   - v4_exact_geometry_bodies_test.go (body-separation contracts)
//   - v4_exact_geometry_internal_test.go (internal token-span tests)
//   - v4_exact_geometry_determinism_test.go (Determinism)
//   - v4_exact_geometry_ordering_test.go (CanonicalFindingOrdering,
//     CanonicalOccurrenceOrdering)
package dupcode

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizeFixturePath_Contract is a table-driven contract test for
// normalizeFixturePath. It exercises the documented edge cases:
//
//   - nested/file.go:        accepted and preserved
//   - ..generated.go:        accepted
//   - ../outside.go:         rejected
//   - nested/../../outside.go: rejected
//   - absolute in-root path: accepted and relativized
//   - absolute out-of-root:  rejected
//   - empty root:            rejected
//   - empty occurrence path: rejected
//
// Where platform behavior differs, assertions are portable (use
// filepath.IsAbs and filepath.ToSlash for normalization, but never
// assume a specific path separator beyond slash for comparison).
func TestNormalizeFixturePath_Contract(t *testing.T) {
	root := t.TempDir()

	cases := []struct {
		name        string
		fixtureRoot string
		occPath     string
		wantOK      bool
		want        string
	}{
		{
			name:        "nested/file.go accepted and preserved",
			fixtureRoot: root,
			occPath:     "nested/file.go",
			wantOK:      true,
			want:        "nested/file.go",
		},
		{
			name:        "..generated.go accepted (legitimate local name)",
			fixtureRoot: root,
			occPath:     "..generated.go",
			wantOK:      true,
			want:        "..generated.go",
		},
		{
			name:        "../outside.go rejected (escapes root)",
			fixtureRoot: root,
			occPath:     "../outside.go",
			wantOK:      false,
		},
		{
			name:        "nested/../../outside.go rejected (escapes root)",
			fixtureRoot: root,
			occPath:     "nested/../../outside.go",
			wantOK:      false,
		},
		{
			name:        "absolute in-root path accepted and relativized",
			fixtureRoot: root,
			occPath:     filepath.Join(root, "nested", "file.go"),
			wantOK:      true,
			want:        "nested/file.go",
		},
		{
			name:        "absolute out-of-root rejected",
			fixtureRoot: root,
			occPath:     filepath.Join(filepath.Dir(root), "outside.go"),
			wantOK:      false,
		},
		{
			name:        "empty root rejected",
			fixtureRoot: "",
			occPath:     "anything.go",
			wantOK:      false,
		},
		{
			name:        "empty occurrence path rejected",
			fixtureRoot: root,
			occPath:     "",
			wantOK:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeFixturePath(tc.fixtureRoot, tc.occPath)
			if tc.wantOK {
				if err != nil {
					t.Fatalf("expected acceptance, got error: %v", err)
				}
				if got != tc.want {
					t.Errorf("normalizeFixturePath() = %q, want %q", got, tc.want)
				}
				// Accepted paths must be portable slash-separated.
				if strings.Contains(got, "\\") {
					t.Errorf("accepted path %q must use forward slashes only", got)
				}
			} else {
				if err == nil {
					t.Errorf("expected rejection, got accepted path %q", got)
				}
			}
		})
	}
}

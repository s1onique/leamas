// SPDX-License-Identifier: Apache-2.0

package closure

// v2_path_resolution_test.go focuses on the deepest-ancestor
// resolver invariants required by
// ACT-LEAMAS-FACTORY-CLOSURE-PROTOCOL-V2-RUNNER-PATH-AUTHORITY01.
// Splitting these from v2_path_matrix_test.go keeps both
// files under the LLM-friendly 400-line threshold.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestV2PathResolution_DeepestExistingAncestor asserts the
// canonical resolver finds the deepest existing ancestor
// when the leaf does not exist. The test mounts a small
// nested directory and asks the resolver to canonicalise a
// path whose leaf is intentionally missing.
func TestV2PathResolution_DeepestExistingAncestor(t *testing.T) {
	root := t.TempDir()
	for _, seg := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(root, seg), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", seg, err)
		}
	}
	// Leaf "c" does not exist; the resolver must walk up to
	// b (the deepest existing ancestor) and append the
	// nonexistent suffix lexically.
	leaf := filepath.Join(root, "a", "b", "c")
	got, err := canonicalisePathDetached(leaf)
	if err != nil {
		t.Fatalf("canonicalisePathDetached: %v", err)
	}
	want := filepath.Join(root, "a", "b", "c")
	if got != want {
		t.Fatalf("canonicalise: got=%s want=%s", got, want)
	}
}

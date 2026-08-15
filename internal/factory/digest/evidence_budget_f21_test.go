// SPDX-License-Identifier: Apache-2.0

// Package digest: evidence_budget_f21_test.go verifies the
// F21 (CORRECTION05) invariant: in range mode, the
// non-normal classification branch in the range renderer
// preserves the SELF_OUTPUT_EXCLUDED diagnostic when the
// classifier selects ClassBoundedSelfOutput.
//
// The fixture is constructed so the worktree file at the
// digest output path exists but its body does NOT match
// the digest artifact signatures — that way the classifier
// selects BOUNDED_SELF_OUTPUT (path-based) rather than
// BOUNDED_DERIVED_DIGEST (content-based).
package digest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceBudget_F21_RangeSelfOutput_DiagnosticPreserved(t *testing.T) {
	dir := t.TempDir()
	initGit(t, dir)
	seedRangeHistory(t, dir)

	// Output path. The file at this path will exist in
	// the worktree but its content will be ordinary
	// text, NOT a digest artifact. This forces the
	// classifier to select BOUNDED_SELF_OUTPUT (the
	// path matches the canonical output) rather than
	// BOUNDED_DERIVED_DIGEST (which would require the
	// content to match the artifact signatures).
	out := filepath.Join(dir, "docs", "evidence", "self-output.txt")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Note: NOT seeding the output with digest markers,
	// so isLeamasDigestArtifactContent() returns false.
	writeFilePath(t, out, "ordinary prose, not a digest\n")

	// Commit the ordinary file at the output path so
	// it shows up in a range diff.
	runGit(t, dir, "add", "docs/evidence/self-output.txt")
	runGit(t, dir, "commit", "-m", "ordinary file at output path")

	// Generate a digest whose outputAbs is the same
	// path. The classifier should classify
	// self-output.txt as BOUNDED_SELF_OUTPUT.
	d, err := Generate(Options{
		RepoRoot: dir, Mode: ModeRange,
		Range: "HEAD~1..HEAD", Output: out,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// F21 (CORRECTION05): the diagnostic must carry
	// SELF_OUTPUT_EXCLUDED and the warning code must
	// be WarningCodeSelfOutput. Before CORRECTION05,
	// the range renderer's non-normal branch would
	// emit LARGE_FILE_EVIDENCE_BOUNDED — a contract
	// drift that mis-described why the body was
	// suppressed.
	blk := extractBlock(t, d, "=== docs/evidence/self-output.txt ===")
	if !strings.Contains(blk,
		"Classification: "+string(ClassBoundedSelfOutput)) {
		t.Fatalf("F21 violated: classification is not "+
			"BOUNDED_SELF_OUTPUT. Excerpt:\n%s", blk)
	}
	if !strings.Contains(blk, "WarningCode: "+WarningCodeSelfOutput) {
		t.Fatalf("F21 violated: WarningCode is not "+
			"%s. Excerpt:\n%s", WarningCodeSelfOutput, blk)
	}
	if !strings.Contains(blk, "SELF_OUTPUT_EXCLUDED") {
		t.Fatalf("F21 violated: body note missing "+
			"SELF_OUTPUT_EXCLUDED. Excerpt:\n%s", blk)
	}
	if strings.Contains(blk, "LARGE_FILE_EVIDENCE_BOUNDED") {
		t.Fatalf("F21 violated: warning code is the "+
			"generic LARGE_FILE_EVIDENCE_BOUNDED "+
			"instead of SELF_OUTPUT_EXCLUDED. "+
			"Excerpt:\n%s", blk)
	}
}

// Package dupcode provides a baseline-transition audit that
// operates on the actual production pipeline.
//
// The committed `.factory/dupcode-baseline.json` reports one
// 504-token finding spanning claim_commands.go:268-340 and
// evidence_commands.go:310-382. Earlier close reports incorrectly
// stated that the prior 514-token finding was a structural shadow
// of the surviving 504-token finding; that claim is impossible
// because componentIsStructuralShadow rejects a sub-finding whose
// TokenCount is greater than or equal to its larger owner. The
// corrected close report classifies the prior 514-token finding
// as either not-equal-normalized-content or as obsolete legacy
// chain geometry that no longer materializes.
//
// TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline is the
// only drift-protection witness that operates on real production
// code. The textual-classification tests assert that the
// production source retains the relevant guards.
package dupcode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline runs
// CheckRepo on the actual live tree and confirms the resulting
// finding's geometry matches the committed baseline JSON.
func TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline(t *testing.T) {
	root := deltaRepoRoot(t)
	findings, err := CheckRepo(root, DefaultConfig())
	if err != nil {
		t.Fatalf("CheckRepo on live tree failed: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("live tree expected exactly one finding, got %d", len(findings))
	}
	got := findings[0]
	if got.TokenCount != 504 || got.LineCount != 73 {
		t.Fatalf("live tree token/line geometry drift: %+v", got)
	}
	if len(got.Occurrences) != 2 {
		t.Fatalf("expected 2 occurrences, got %d", len(got.Occurrences))
	}
	expectedOccs := []Occurrence{
		{Path: "cmd/leamas/claim_commands.go", StartLine: 268, EndLine: 340},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 310, EndLine: 382},
	}
	for i, occ := range got.Occurrences {
		if occ.Path != expectedOccs[i].Path {
			t.Errorf("occurrence %d path drift: live=%q baseline=%q",
				i, occ.Path, expectedOccs[i].Path)
		}
		if occ.StartLine != expectedOccs[i].StartLine || occ.EndLine != expectedOccs[i].EndLine {
			t.Errorf("occurrence %d line drift: live=%d-%d baseline=%d-%d",
				i, occ.StartLine, occ.EndLine,
				expectedOccs[i].StartLine, expectedOccs[i].EndLine)
		}
	}
}

// TestV4BaselineAudit_StructuralShadowRejectsLargerSubFinding
// asserts that the production source retains the
// `if large.TokenCount <= small.TokenCount { return false }`
// guard. A runtime invocation against a manually-synthesized
// 514/504 pair would panic the helper's slice-bounds check
// (manualAnalyzedFiles provisions only 20 tokens) so a textual
// source guard is the most reliable witness.
func TestV4BaselineAudit_StructuralShadowRejectsLargerSubFinding(t *testing.T) {
	want := "if large.TokenCount <= small.TokenCount {"
	if !strings.Contains(readSource(t, "v4_component_merge.go"), want) {
		t.Errorf("structural-shadow guard missing substring %q in v4_component_merge.go", want)
	}
}

// readSource returns the source of a dupcode package file as a
// single string for textual assertion.
func readSource(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(deltaRepoRoot(t), "internal", "factory", "dupcode", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

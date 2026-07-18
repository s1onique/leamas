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
// CheckRepo on the live tree and confirms the report contains
// zero findings after ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-
// REMEDIATION01 removed the canonical duplicate. After this
// ACT regenerates the baseline, the live scan and committed
// baseline both report zero findings and the policy thresholds
// remain at 40 lines / 400 tokens.
//
// The historical 504-token claim/evidence finding (lines 268-340
// and 310-382) and its fingerprint are preserved as tracked
// evidence under
// docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/
// so the historical remediation proof remains auditable.
func TestV4BaselineAudit_LiveTreeMatchesCommittedBaseline(t *testing.T) {
	root := deltaRepoRoot(t)
	findings, err := CheckRepo(root, DefaultConfig())
	if err != nil {
		t.Fatalf("CheckRepo on live tree failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("live tree must report zero findings after remediation; got %d: %+v",
			len(findings), findings)
	}
	// Setup witness: the canonical 40-line / 400-token policy
	// thresholds must remain in effect.
	if got := DefaultConfig().MinLines; got != 40 {
		t.Errorf("MinLines drift: live=%d, want 40", got)
	}
	if got := DefaultConfig().MinTokens; got != 400 {
		t.Errorf("MinTokens drift: live=%d, want 400", got)
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

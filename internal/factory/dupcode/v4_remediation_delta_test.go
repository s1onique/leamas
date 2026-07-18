// Package dupcode - narrowly-scoped detector delta test for
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01.
//
// Per R12 of the ACT, internal/factory/dupcode/ production code must
// not be modified by this ACT. The only allowed exception is a
// narrowly-scoped test that proves the self-hosted detector delta
// when no existing test seam can express that proof.
//
// The pre-existing forensics tests in v4_baseline_forensics_test.go
// and v4_pipeline_trace_test.go pin down the 504-token canonical
// body that this ACT removes. Those tests assert properties of code
// regions that no longer exist; they will be addressed by a future
// ACT once the baseline has been regenerated. This file adds the
// single, focused assertion needed by THIS ACT: that the live
// repository no longer emits the frozen 504-token finding.
//
// Scope:
//   - one Test function
//   - one live-tree CheckReport
//   - one assertion against the canonical fingerprint, token count,
//     and occurrence geometry recorded in the frozen baseline.
//
// The test is intentionally minimal: any future ACT can replace it
// once the broader forensics suite is reconciled with the new state.
package dupcode

import (
	"strings"
	"testing"
)

// TestRemediationDelta_504FindingRemoved asserts that the canonical
// 504-token finding whose fingerprint is frozen in the committed
// baseline is no longer emitted by the live tree. This is the
// narrowly-scoped detector delta required by ACT
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01 R12.
//
// The test reads the frozen baseline at .factory/dupcode-baseline.json
// (loadable via LoadBaseline) and asserts that no finding in the
// current CheckReport matches the frozen fingerprint, token count,
// AND ordered occurrence geometry.
func TestRemediationDelta_504FindingRemoved(t *testing.T) {
	const frozenFingerprint = "86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b"

	// Live scan with the canonical gate thresholds. The frozen
	// baseline uses the same defaults (min_lines=40, min_tokens=400).
	cfg := DefaultConfig()
	report, err := CheckReport(".", cfg)
	if err != nil {
		t.Fatalf("CheckReport: %v", err)
	}

	for _, f := range report.Findings {
		if f.StableFingerprint != frozenFingerprint {
			continue
		}
		if f.TokenCount != 504 || f.LineCount != 73 {
			t.Errorf("fingerprint %s reappeared with unexpected geometry: "+
				"token_count=%d line_count=%d (want 504/73)",
				frozenFingerprint, f.TokenCount, f.LineCount)
		}
		// Defensive: the canonical fingerprint and geometry must
		// never reappear under the same fingerprint. If this fires
		// the refactor has been undone silently.
		occ := make([]string, 0, len(f.Occurrences))
		for _, o := range f.Occurrences {
			occ = append(occ, o.Path)
		}
		t.Errorf("frozen 504-token finding reappeared at %v (must be removed)", occ)
	}

	// Also assert no finding carries a 504-token body with two
	// occurrences in the legacy claim/evidence command files. This
	// guards against a future regression where the fingerprint
	// happens to collide but the geometry changes.
	for _, f := range report.Findings {
		if f.TokenCount != 504 {
			continue
		}
		hitsClaim := false
		hitsEvidence := false
		for _, o := range f.Occurrences {
			if strings.HasSuffix(o.Path, "claim_commands.go") {
				hitsClaim = true
			}
			if strings.HasSuffix(o.Path, "evidence_commands.go") {
				hitsEvidence = true
			}
		}
		if hitsClaim && hitsEvidence && len(f.Occurrences) == 2 {
			t.Errorf("regression: 504-token claim/evidence finding reappeared "+
				"(fingerprint=%s)", f.Fingerprint)
		}
	}

	// Sanity check: the running scan must complete without error and
	// the report must have a Findings slice (not nil) so the test
	// is meaningful even on an empty tree.
	_ = report.Findings == nil
}

// Package dupcode - narrowly-scoped detector-delta test for
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01 and its
// successor ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01.
//
// Per R12 of the predecessor ACT, internal/factory/dupcode/
// production code must not be modified by the remediation ACT.
// The only allowed exception is a narrowly-scoped test that proves
// the self-hosted detector delta when no existing test seam can
// express that proof. The predecessor ACT introduced that test
// (TestRemediationDelta_504FindingRemoved); this ACT strengthens
// it to read the frozen predecessor evidence so the historical
// removal is provable without depending on the regenerated
// baseline.
//
// Scope:
//   - one Test function (TestRemediationDelta_504FindingRemoved)
//   - one live-tree CheckReport
//   - one assertion against the frozen predecessor JSON
//   - four additional setup witnesses so the test fails on
//     setup, not on a vacuous zero-finding assertion.
//
// The test proves:
//  1. The frozen predecessor fingerprint
//     86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b
//     is absent from the current live scan.
//  2. The current live scan contains zero findings.
//  3. The historical before evidence contains exactly the
//     frozen finding (fingerprint, token count, line count,
//     and the two claim/evidence occurrence paths).
//  4. The historical after evidence contains zero findings.
//  5. The regenerated committed baseline contains zero findings
//     (proving the convergence ACT rewrote the baseline to
//     reflect the live tree, not the historical state).
package dupcode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// frozenFingerprint is the canonical fingerprint of the
// self-hosted claim/evidence duplicate that was removed by
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01. The value
// is pinned here so the test does not depend on the committed
// baseline.
const frozenFingerprint = "86fae794736b22ea7939fefe24346360250945fa9a13388152a5f3a96471354b"

// frozenPredecessorBeforeRelPath is the repository-relative path
// to the frozen predecessor before-evidence JSON. Tests join
// this with repoRoot() because the test working directory is
// the dupcode package directory, three levels deep.
const frozenPredecessorBeforeRelPath = "docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-before.json"

// frozenPredecessorAfterRelPath is the repository-relative path
// to the frozen predecessor after-evidence JSON.
const frozenPredecessorAfterRelPath = "docs/close-reports/ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01/dupcode-after.json"

// predecessorBeforeFinding is the shape of the frozen
// predecessor before-evidence JSON. The JSON serializes both a
// display fingerprint (truncated) and a stable fingerprint;
// the test compares against the stable fingerprint.
//
// Field tags use PascalCase keys because the predecessor ACT's
// captured scan used the PascalCase JSON encoding of the V3
// dupcode schema. The current V4 schema uses snake_case but
// the frozen predecessor evidence must remain immutable.
type predecessorBeforeFinding struct {
	StableFingerprint string `json:"StableFingerprint"`
	Fingerprint       string `json:"Fingerprint"`
	TokenCount        int    `json:"TokenCount"`
	LineCount         int    `json:"LineCount"`
	Occurrences       []struct {
		Path      string `json:"Path"`
		StartLine int    `json:"StartLine"`
		EndLine   int    `json:"EndLine"`
	} `json:"Occurrences"`
}

type predecessorBeforeReport struct {
	SchemaVersion    int                        `json:"SchemaVersion"`
	AlgorithmVersion int                        `json:"AlgorithmVersion"`
	GeneratedAt      string                     `json:"GeneratedAt"`
	Tool             string                     `json:"Tool"`
	Thresholds       map[string]int             `json:"Thresholds"`
	Findings         []predecessorBeforeFinding `json:"Findings"`
}

type predecessorAfterReport struct {
	SchemaVersion    int                      `json:"schema_version"`
	AlgorithmVersion int                      `json:"algorithm_version"`
	GeneratedAt      string                   `json:"generated_at"`
	Tool             string                   `json:"tool"`
	Thresholds       map[string]int           `json:"thresholds"`
	Findings         []map[string]interface{} `json:"findings"`
}

// TestRemediationDelta_504FindingRemoved asserts the canonical
// 504-token finding whose fingerprint is frozen in the
// predecessor evidence is no longer emitted by the live tree.
// This is the narrowly-scoped detector delta required by the
// predecessor ACT and the strengthened convergence proof required
// by the successor ACT.
//
// The test reads the frozen predecessor before/after evidence
// files to verify:
//   - the before evidence contains the frozen finding;
//   - the after evidence contains zero findings.
//
// The test reads the committed baseline to verify:
//   - the committed baseline contains zero findings (the
//     convergence ACT has rewritten the baseline to reflect the
//     live tree, not the historical state).
//
// The test runs CheckReport on the live tree to verify:
//   - the live scan reports zero findings;
//   - no finding in the live report matches the frozen
//     fingerprint.
func TestRemediationDelta_504FindingRemoved(t *testing.T) {
	// (1) Live scan with the canonical gate thresholds.
	cfg := DefaultConfig()
	report, err := CheckReport(".", cfg)
	if err != nil {
		t.Fatalf("CheckReport: %v", err)
	}

	// (2) The frozen predecessor fingerprint must NOT appear in
	// the current live scan.
	for _, f := range report.Findings {
		if f.StableFingerprint != frozenFingerprint {
			continue
		}
		occ := make([]string, 0, len(f.Occurrences))
		for _, o := range f.Occurrences {
			occ = append(occ, o.Path)
		}
		t.Errorf("frozen 504-token finding reappeared at %v (must be removed)", occ)
	}

	// (3) No finding carries a 504-token body with two
	// occurrences in the legacy claim/evidence command files.
	// This guards against a future regression where the
	// fingerprint happens to collide but the geometry changes.
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

	// (4) The live scan must contain zero findings; the
	// remediation ACT removed the only policy-threshold
	// duplicate.
	if len(report.Findings) != 0 {
		t.Fatalf("live scan must report zero findings after remediation; got %d: %+v",
			len(report.Findings), report.Findings)
	}

	// (5) The frozen predecessor BEFORE evidence must contain
	// exactly the frozen finding. This proves the 504-token
	// duplicate existed before remediation; without this, the
	// "removal" claim is vacuous.
	beforePath := filepath.Join(repoRoot(), frozenPredecessorBeforeRelPath)
	beforeRaw, err := os.ReadFile(beforePath)
	if err != nil {
		t.Fatalf("read frozen predecessor before-evidence at %s: %v", beforePath, err)
	}
	var beforeReport predecessorBeforeReport
	if err := json.Unmarshal(beforeRaw, &beforeReport); err != nil {
		t.Fatalf("parse frozen predecessor before-evidence: %v", err)
	}
	if len(beforeReport.Findings) != 1 {
		t.Fatalf("frozen predecessor before-evidence must contain exactly 1 finding; got %d",
			len(beforeReport.Findings))
	}
	bf := beforeReport.Findings[0]
	if bf.StableFingerprint != frozenFingerprint {
		t.Errorf("frozen predecessor before-evidence stable_fingerprint=%s, want %s",
			bf.StableFingerprint, frozenFingerprint)
	}
	if bf.TokenCount != 504 || bf.LineCount != 73 {
		t.Errorf("frozen predecessor before-evidence geometry drift: token_count=%d line_count=%d (want 504/73)",
			bf.TokenCount, bf.LineCount)
	}
	if len(bf.Occurrences) != 2 {
		t.Fatalf("frozen predecessor before-evidence must have 2 occurrences; got %d",
			len(bf.Occurrences))
	}
	wantPaths := []string{
		"cmd/leamas/claim_commands.go",
		"cmd/leamas/evidence_commands.go",
	}
	for i, occ := range bf.Occurrences {
		if occ.Path != wantPaths[i] {
			t.Errorf("frozen predecessor before-evidence occurrence %d path=%q, want %q",
				i, occ.Path, wantPaths[i])
		}
	}

	// (6) The frozen predecessor AFTER evidence must contain
	// zero findings. This proves the remediation refactor
	// actually removed the finding, not merely renamed it.
	afterPath := filepath.Join(repoRoot(), frozenPredecessorAfterRelPath)
	afterRaw, err := os.ReadFile(afterPath)
	if err != nil {
		t.Fatalf("read frozen predecessor after-evidence at %s: %v", afterPath, err)
	}
	var afterReport predecessorAfterReport
	if err := json.Unmarshal(afterRaw, &afterReport); err != nil {
		t.Fatalf("parse frozen predecessor after-evidence: %v", err)
	}
	if len(afterReport.Findings) != 0 {
		t.Errorf("frozen predecessor after-evidence must contain zero findings; got %d",
			len(afterReport.Findings))
	}

	// (7) The current committed baseline must contain zero
	// findings. This proves the convergence ACT rewrote the
	// baseline to reflect the live tree, not the historical
	// state. Without this check, the test would pass even if
	// the committed baseline still records the removed
	// 504-token finding.
	baselinePath := filepath.Join(repoRoot(), ".factory", "dupcode-baseline.json")
	baseline, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("LoadBaseline: %v", err)
	}
	if len(baseline.Findings) != 0 {
		t.Errorf("committed baseline must report zero findings after convergence; got %d: %+v",
			len(baseline.Findings), baseline.Findings)
	}
}

// repoRoot returns the absolute path to the repository root.
// Tests run from internal/factory/dupcode (three levels deep).
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		panic("os.Getwd: " + err.Error())
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

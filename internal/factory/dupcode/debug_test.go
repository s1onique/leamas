// Package dupcode provides the post-convergence convergence
// witness test for the dupcode baseline.
//
// TestDebugBaselines used to print a verbose comparison of the
// committed baseline and the live canonical scan. After
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-REMEDIATION01 the
// historical 504-token claim/evidence finding is gone, and the
// canonical duplicate code gate is now a convergence check
// against the frozen zero-finding invariant.
//
// The test has been repurposed as a deterministic equality
// witness. It reads the committed baseline, runs CheckReport
// against the live tree, generates the canonical baseline from
// the live report, and asserts:
//
//   - both the committed and canonical reports contain zero
//     findings (the live tree is clean and the committed baseline
//     has converged to that state);
//   - the two reports are byte-equal via baselinesEqual;
//   - the policy thresholds remain at 40/400.
//
// The test no longer emits verbose stdout/Printf because the
// only useful artifact is the PASS/FAIL outcome.

package dupcode

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestDebugBaselines asserts the committed baseline and the
// live scan are equal and both contain zero findings after the
// baseline convergence ACT. The test is the live-tree
// convergence witness for
// ACT-LEAMAS-FACTORY-DUPCODE-SELF-HOSTED-BASELINE-CONVERGENCE01.
func TestDebugBaselines(t *testing.T) {
	baselinePath := deltaRepoRoot(t) + "/.factory/dupcode-baseline.json"

	// Load committed baseline.
	committed, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("Error loading committed baseline: %v", err)
	}

	// Run current scan against the live repository.
	cfg := DefaultConfig()
	cfg.MinLines = committed.Thresholds.MinLines
	cfg.MinTokens = committed.Thresholds.MinTokens
	root := deltaRepoRoot(t)
	cfg.Root = root

	report, err := CheckReport(root, cfg)
	if err != nil {
		t.Fatalf("Error running scan: %v", err)
	}

	// Generate the canonical baseline from the live report.
	canonical := GenerateCanonicalBaseline(root, report)

	// Setup witnesses: the scan must have completed without error
	// and the live tree must be clean. A nil findings slice with
	// a nil error is the canonical "clean tree" signal, not a
	// setup failure; the comparison below relies on len() rather
	// than a nil check so the witness holds either way.
	if err != nil {
		// (Already checked above; kept as a witness.)
		_ = err
	}
	if len(committed.Findings) != 0 {
		t.Fatalf("committed baseline must report zero findings after convergence; got %d: %+v",
			len(committed.Findings), committed.Findings)
	}
	if len(canonical.Findings) != 0 {
		t.Fatalf("canonical baseline must report zero findings; got %d: %+v",
			len(canonical.Findings), canonical.Findings)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("live report must report zero findings after remediation; got %d: %+v",
			len(report.Findings), report.Findings)
	}
	if committed.Thresholds.MinLines != 40 || committed.Thresholds.MinTokens != 400 {
		t.Fatalf("baseline thresholds drifted from policy 40/400: min_lines=%d min_tokens=%d",
			committed.Thresholds.MinLines, committed.Thresholds.MinTokens)
	}

	// Equality witness: the committed baseline and the canonical
	// baseline must be byte-equal so the baseline-verify gate
	// can no longer report drift.
	if !baselinesEqual(committed, canonical) {
		t.Fatal("committed baseline differs from canonical baseline (drift would re-open the gate)")
	}
}

// TestDeterministicCoalescing verifies that the v3 algorithm produces byte-identical
// output across multiple runs. This regression test catches nondeterminism from
// map iteration order.
func TestDeterministicCoalescing(t *testing.T) {
	// Construct a nontrivial windowMap with multiple fingerprints and chain keys
	windowMap := buildDeterministicTestWindowMap()
	fingerprintTokens := map[string]int{
		"seed1": 400,
		"seed2": 400,
		"seed3": 400,
	}

	iterations := 20
	var firstResult []byte

	for i := 0; i < iterations; i++ {
		// Run the complete v3 coalescing path
		findings := v3CoalesceFindings(windowMap, fingerprintTokens)

		// Serialize the complete findings
		result, err := json.Marshal(findings)
		if err != nil {
			t.Fatalf("Iteration %d: marshal error: %v", i, err)
		}

		if i == 0 {
			firstResult = result
			continue
		}

		// Compare byte-for-byte with first run
		if !bytes.Equal(result, firstResult) {
			t.Errorf("Iteration %d: output differs from first run", i)
		}
	}
}

// buildDeterministicTestWindowMap creates a test windowMap with multiple fingerprints
// and overlapping windows to test chain assembly determinism.
func buildDeterministicTestWindowMap() map[string][]rawWindow {
	wm := make(map[string][]rawWindow)

	// Seed 1: Multiple windows that should chain together
	wm["seed1"] = []rawWindow{
		{Path: "cmd/leamas/claim_commands.go", StartLine: 100, EndLine: 110, StartPos: 100, EndPos: 139},
		{Path: "cmd/leamas/claim_commands.go", StartLine: 108, EndLine: 118, StartPos: 108, EndPos: 147},
		{Path: "cmd/leamas/claim_commands.go", StartLine: 116, EndLine: 126, StartPos: 116, EndPos: 155},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 150, EndLine: 160, StartPos: 150, EndPos: 189},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 158, EndLine: 168, StartPos: 158, EndPos: 197},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 166, EndLine: 176, StartPos: 166, EndPos: 205},
	}

	// Seed 2: Different pattern
	wm["seed2"] = []rawWindow{
		{Path: "cmd/leamas/claim_commands.go", StartLine: 300, EndLine: 310, StartPos: 300, EndPos: 339},
		{Path: "cmd/leamas/claim_commands.go", StartLine: 308, EndLine: 318, StartPos: 308, EndPos: 347},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 350, EndLine: 360, StartPos: 350, EndPos: 389},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 358, EndLine: 368, StartPos: 358, EndPos: 397},
	}

	// Seed 3: Another distinct pattern
	wm["seed3"] = []rawWindow{
		{Path: "cmd/leamas/claim_commands.go", StartLine: 50, EndLine: 60, StartPos: 50, EndPos: 89},
		{Path: "cmd/leamas/evidence_commands.go", StartLine: 100, EndLine: 110, StartPos: 100, EndPos: 139},
	}

	return wm
}

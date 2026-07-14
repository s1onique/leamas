package dupcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDebugBaselines(t *testing.T) {
	// Find repo root
	wd, _ := os.Getwd()
	repoRoot := wd
	for {
		if filepath.Base(wd) == "leamas" {
			repoRoot = wd
			break
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}

	baselinePath := filepath.Join(repoRoot, ".factory/dupcode-baseline.json")

	// Load committed baseline
	committed, err := LoadBaseline(baselinePath)
	if err != nil {
		t.Fatalf("Error loading committed: %v", err)
	}

	// Run current scan
	cfg := DefaultConfig()
	cfg.MinLines = committed.Thresholds.MinLines
	cfg.MinTokens = committed.Thresholds.MinTokens
	cfg.Root = repoRoot

	report, err := CheckReport(repoRoot, cfg)
	if err != nil {
		t.Fatalf("Error running scan: %v", err)
	}

	// Generate canonical baseline
	canonical := GenerateCanonicalBaseline(repoRoot, report)

	fmt.Printf("Repo root: %s\n", repoRoot)
	fmt.Printf("Committed findings: %d\n", len(committed.Findings))
	fmt.Printf("Canonical findings: %d\n", len(canonical.Findings))

	for i, cf := range committed.Findings {
		if i >= len(canonical.Findings) {
			fmt.Printf("Extra finding in committed at index %d: %s\n", i, cf.Fingerprint)
			continue
		}
		ccf := canonical.Findings[i]
		if cf.Fingerprint != ccf.Fingerprint {
			fmt.Printf("Finding %d: committed=%s canonical=%s\n", i, cf.Fingerprint, ccf.Fingerprint)
			fmt.Printf("  committed tokens=%d lines=%d\n", cf.TokenCount, cf.LineCount)
			fmt.Printf("  canonical tokens=%d lines=%d\n", ccf.TokenCount, ccf.LineCount)
		}
	}

	if baselinesEqual(committed, canonical) {
		fmt.Println("baselinesEqual: EQUAL")
	} else {
		fmt.Println("baselinesEqual: DIFFER")
	}

	committedJSON, _ := json.MarshalIndent(committed.Findings, "", "  ")
	canonicalJSON, _ := json.MarshalIndent(canonical.Findings, "", "  ")
	fmt.Println("\n--- COMMITTED FINDINGS ---")
	os.Stdout.Write(committedJSON)
	fmt.Println("\n--- CANONICAL FINDINGS ---")
	os.Stdout.Write(canonicalJSON)
	fmt.Println()
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

package dupcode

import (
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

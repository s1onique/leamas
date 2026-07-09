// Package main provides factory verify dupcode handler with baseline support.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// Default thresholds for the quality gate
const (
	DefaultMinLines  = 40
	DefaultMinTokens = 400
)

// BaselineDefaultPath is the default path for the baseline file.
const BaselineDefaultPath = ".factory/dupcode-baseline.json"

func handleFactoryVerifyDupcode() {
	// Reset flag state for this subcommand
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: leamas factory verify dupcode [options]\n")
		flag.CommandLine.PrintDefaults()
	}

	// Parse flags for dupcode subcommand
	baselinePath := flag.String("baseline", BaselineDefaultPath, "Path to baseline file")
	updateBaseline := flag.Bool("update-baseline", false, "Update baseline file with current findings")
	minLines := flag.Int("min-lines", DefaultMinLines, "Minimum lines for duplicate block")
	minTokens := flag.Int("min-tokens", DefaultMinTokens, "Minimum tokens for duplicate block")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")

	// Parse only the arguments after "dupcode"
	// os.Args = ["leamas", "factory", "verify", "dupcode", "--update-baseline", ...]
	// We want to parse ["dupcode", "--update-baseline", ...]
	args := os.Args[4:] // Skip "leamas factory verify"
	if err := flag.CommandLine.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		// Flag parse error - report and exit
		if *jsonOutput {
			fmt.Printf(`{"error": "flag parse error: %v"}`, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(2)
	}

	// Build config
	cfg := dupcode.DefaultConfig()
	cfg.MinLines = *minLines
	cfg.MinTokens = *minTokens

	if *updateBaseline {
		handleUpdateBaseline(*baselinePath, cfg, *jsonOutput)
		return
	}

	handleVerifyBaseline(*baselinePath, cfg, *jsonOutput)
}

func handleUpdateBaseline(baselinePath string, cfg dupcode.Config, jsonOutput bool) {
	// Scan repo
	report, err := dupcode.CheckReport(".", cfg)
	if err != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("scan failed: %v", err)})
		} else {
			fmt.Fprintf(os.Stderr, "Error scanning repository: %v\n", err)
		}
		os.Exit(2)
	}

	// Write baseline
	if err := dupcode.WriteBaseline(baselinePath, report); err != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("failed to write baseline: %v", err)})
		} else {
			fmt.Fprintf(os.Stderr, "Error writing baseline: %v\n", err)
		}
		os.Exit(2)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(map[string]interface{}{
			"baseline":   baselinePath,
			"findings":   len(report.Findings),
			"thresholds": map[string]int{"min_lines": cfg.MinLines, "min_tokens": cfg.MinTokens},
		})
	} else {
		fmt.Printf("Baseline written to: %s\n", baselinePath)
		fmt.Printf("Findings: %d\n", len(report.Findings))
		fmt.Printf("Thresholds: min_lines=%d, min_tokens=%d\n", cfg.MinLines, cfg.MinTokens)
	}

	os.Exit(0)
}

func handleVerifyBaseline(baselinePath string, cfg dupcode.Config, jsonOutput bool) {
	// Check if baseline exists
	if _, err := os.Stat(baselinePath); os.IsNotExist(err) {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{
				"error": "baseline not found",
				"hint":  "run with --update-baseline to create baseline",
			})
		} else {
			fmt.Fprintf(os.Stderr, "Baseline file not found: %s\n", baselinePath)
			fmt.Fprintf(os.Stderr, "Run 'leamas factory verify dupcode --update-baseline' to create a baseline.\n")
		}
		os.Exit(2)
	}

	// Load baseline
	baseline, err := dupcode.LoadBaseline(baselinePath)
	if err != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("failed to load baseline: %v", err)})
		} else {
			fmt.Fprintf(os.Stderr, "Error loading baseline: %v\n", err)
		}
		os.Exit(2)
	}

	// Scan repo
	report, err := dupcode.CheckReport(".", cfg)
	if err != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("scan failed: %v", err)})
		} else {
			fmt.Fprintf(os.Stderr, "Error scanning repository: %v\n", err)
		}
		os.Exit(2)
	}

	// Compare with baseline
	result := dupcode.CompareToBaseline(report, baseline)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		if result.HasChanges {
			enc.Encode(map[string]interface{}{
				"new_findings":      len(result.NewFindings),
				"worsened_findings": len(result.WorsenedFindings),
				"has_changes":       true,
			})
		} else {
			enc.Encode(map[string]interface{}{"has_changes": false})
		}
	} else {
		dupcode.PrintCompareResult(result)
	}

	os.Exit(dupcode.ExitCodeFromCompareResult(result))
}

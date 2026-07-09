// Package main provides the factory coverage command handler.
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/coverage"
)

// handleFactoryCoverage handles the `leamas factory coverage` command.
// It checks a coverage profile against a threshold and optionally generates a module breakdown.
func handleFactoryCoverage() {
	args := os.Args[3:]
	var profilePath string
	var minTotal float64
	var jsonOutputPath string
	var showBreakdown bool = true // default to showing breakdown

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --profile requires a path argument\n")
				printCoverageUsage()
				os.Exit(1)
			}
			profilePath = args[i+1]
			i++
		case "--min-total":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --min-total requires a float argument\n")
				printCoverageUsage()
				os.Exit(1)
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &minTotal); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: invalid --min-total value: %s\n", args[i+1])
				printCoverageUsage()
				os.Exit(1)
			}
			i++
		case "--json-output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "ERROR: --json-output requires a path argument\n")
				printCoverageUsage()
				os.Exit(1)
			}
			jsonOutputPath = args[i+1]
			i++
		case "--breakdown":
			showBreakdown = true
		case "--no-breakdown":
			showBreakdown = false
		default:
			fmt.Fprintf(os.Stderr, "ERROR: unknown flag: %s\n", args[i])
			printCoverageUsage()
			os.Exit(1)
		}
	}

	if profilePath == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --profile is required\n")
		printCoverageUsage()
		os.Exit(1)
	}

	// Parse the raw coverage profile for statement-weighted coverage
	profileReport, err := coverage.ParseProfile(profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage: error parsing profile: %v\n", err)
		os.Exit(1)
	}

	// Check threshold
	threshold := &coverage.Threshold{MinTotalPercent: minTotal}
	if err := coverage.CheckThreshold(coverage.ProfileReportToReport(profileReport), threshold); err != nil {
		covErr, ok := err.(*coverage.Error)
		if ok {
			fmt.Fprintf(os.Stderr, "coverage: %s: %s\n", covErr.Kind, covErr.Message)
		} else {
			fmt.Fprintf(os.Stderr, "coverage: error: %v\n", err)
		}
		os.Exit(1)
	}

	// Print main status line
	fmt.Printf("coverage: total=%.1f%% min=%.1f%% OK\n", profileReport.TotalPercent, minTotal)

	// Print module breakdown by default
	if showBreakdown {
		profileReport.PrintModuleTable()
	}

	// Write JSON output if requested
	if jsonOutputPath != "" {
		jsonData, err := profileReport.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "coverage: error generating JSON: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(jsonOutputPath, jsonData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "coverage: error writing JSON: %v\n", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func printCoverageUsage() {
	fmt.Println("Usage: leamas factory coverage --profile <path> --min-total <float> [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --profile <path>       Path to coverage profile (required)")
	fmt.Println("  --min-total <float>    Minimum total coverage percentage (required)")
	fmt.Println("  --json-output <path>   Write module breakdown JSON to file (optional)")
	fmt.Println("  --breakdown            Show module breakdown (default: true)")
	fmt.Println("  --no-breakdown         Hide module breakdown")
}

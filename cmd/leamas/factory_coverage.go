// Package main provides the factory coverage command handler.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/s1onique/leamas/internal/factory/coverage"
)

// coverageArgs holds parsed arguments for the coverage command.
type coverageArgs struct {
	profilePath       string
	minTotal          float64
	minModulePercents map[string]float64
	jsonOutputPath    string
	showBreakdown     bool
	useDefaultFloors  bool
	printThresholds   bool
	jsonFormat        bool
}

// parseCoverageArgs parses command-line arguments for the coverage command.
// Explicit --min-module values always override --default-module-floors regardless of order.
func parseCoverageArgs(args []string) (coverageArgs, error) {
	result := coverageArgs{
		showBreakdown:     true, // default to showing breakdown
		minModulePercents: make(map[string]float64),
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				return coverageArgs{}, fmt.Errorf("--profile requires a path argument")
			}
			result.profilePath = args[i+1]
			i++
		case "--min-total":
			if i+1 >= len(args) {
				return coverageArgs{}, fmt.Errorf("--min-total requires a float argument")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &result.minTotal); err != nil {
				return coverageArgs{}, fmt.Errorf("invalid --min-total value: %s", args[i+1])
			}
			i++
		case "--min-module":
			if i+1 >= len(args) {
				return coverageArgs{}, fmt.Errorf("--min-module requires a value in the format module=threshold")
			}
			value := args[i+1]
			parts := strings.SplitN(value, "=", 2)
			if len(parts) != 2 {
				return coverageArgs{}, fmt.Errorf("--min-module requires format module=threshold, got: %s", value)
			}
			moduleName := strings.TrimSpace(parts[0])
			thresholdStr := strings.TrimSpace(parts[1])
			if moduleName == "" {
				return coverageArgs{}, fmt.Errorf("--min-module module name cannot be empty")
			}
			threshold, err := strconv.ParseFloat(thresholdStr, 64)
			if err != nil {
				return coverageArgs{}, fmt.Errorf("--min-module threshold must be a valid float: %s", thresholdStr)
			}
			if threshold < 0 {
				return coverageArgs{}, fmt.Errorf("--min-module threshold cannot be negative: %s", thresholdStr)
			}
			if threshold > 100 {
				return coverageArgs{}, fmt.Errorf("--min-module threshold cannot exceed 100: %s", thresholdStr)
			}
			if !coverage.IsKnownEnforcedModule(moduleName) {
				knownModules := strings.Join(coverage.KnownEnforcedModules(), ", ")
				return coverageArgs{}, fmt.Errorf("--min-module unknown module: %s (known: %s)", moduleName, knownModules)
			}
			result.minModulePercents[moduleName] = threshold
			i++
		case "--default-module-floors":
			result.useDefaultFloors = true
		case "--thresholds":
			result.printThresholds = true
		case "--json":
			result.jsonFormat = true
		case "--json-output":
			if i+1 >= len(args) {
				return coverageArgs{}, fmt.Errorf("--json-output requires a path argument")
			}
			result.jsonOutputPath = args[i+1]
			i++
		case "--breakdown":
			result.showBreakdown = true
		case "--no-breakdown":
			result.showBreakdown = false
		default:
			return coverageArgs{}, fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Apply default floors AFTER explicit --min-module values (so explicit wins)
	if result.useDefaultFloors {
		for k, v := range coverage.DefaultModuleThresholds() {
			if _, exists := result.minModulePercents[k]; !exists {
				result.minModulePercents[k] = v
			}
		}
	}

	return result, nil
}

// thresholdsOutput represents the JSON output for thresholds command.
type thresholdsOutput struct {
	SchemaVersion     int               `json:"schema_version"`
	Total             float64           `json:"total"`
	Modules           []moduleThreshold `json:"modules"`
	ReportOnlyModules []string          `json:"report_only_modules"`
}

type moduleThreshold struct {
	Module     string  `json:"module"`
	MinPercent float64 `json:"min_percent"`
}

// printThresholdsOutput prints the canonical default thresholds.
func printThresholdsOutput(stdout io.Writer, jsonFormat bool) {
	threshold := coverage.DefaultThreshold()
	modules := coverage.KnownEnforcedModules()

	if jsonFormat {
		output := thresholdsOutput{
			SchemaVersion:     1,
			Total:             threshold.MinTotalPercent,
			Modules:           make([]moduleThreshold, 0, len(modules)),
			ReportOnlyModules: []string{"other"},
		}
		for _, name := range modules {
			output.Modules = append(output.Modules, moduleThreshold{
				Module:     name,
				MinPercent: threshold.MinModulePercents[name],
			})
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to marshal thresholds: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(stdout, string(data))
	} else {
		fmt.Fprintln(stdout, "coverage thresholds:")
		fmt.Fprintf(stdout, "total >= %.1f\n", threshold.MinTotalPercent)
		for _, name := range modules {
			fmt.Fprintf(stdout, "%s >= %.1f\n", name, threshold.MinModulePercents[name])
		}
		fmt.Fprintln(stdout, "other: report-only")
	}
}

// runFactoryCoverage runs the coverage command with the given arguments.
// It returns 0 on success, non-zero on failure.
func runFactoryCoverage(args []string, stdout, stderr io.Writer) int {
	parsed, err := parseCoverageArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %s\n", err)
		printCoverageUsageTo(stderr)
		return 1
	}

	// Handle threshold printing
	if parsed.printThresholds {
		printThresholdsOutput(stdout, parsed.jsonFormat)
		return 0
	}

	if parsed.profilePath == "" {
		fmt.Fprintf(stderr, "ERROR: --profile is required\n")
		printCoverageUsageTo(stderr)
		return 1
	}

	// Parse the raw coverage profile for statement-weighted coverage
	profileReport, err := coverage.ParseProfile(parsed.profilePath)
	if err != nil {
		fmt.Fprintf(stderr, "coverage: error parsing profile: %v\n", err)
		return 1
	}

	// Check threshold
	threshold := &coverage.Threshold{
		MinTotalPercent:   parsed.minTotal,
		MinModulePercents: parsed.minModulePercents,
	}
	if err := coverage.CheckThreshold(coverage.ProfileReportToReport(profileReport), threshold); err != nil {
		covErr, ok := err.(*coverage.Error)
		if ok {
			fmt.Fprintf(stderr, "coverage: %s: %s\n", covErr.Kind, covErr.Message)
		} else {
			fmt.Fprintf(stderr, "coverage: error: %v\n", err)
		}
		return 1
	}

	// Print main status line
	fmt.Fprintf(stdout, "coverage: total=%.1f%% min=%.1f%% OK\n", profileReport.TotalPercent, parsed.minTotal)

	// Print per-module OK lines for enforced modules that have thresholds
	report := coverage.ProfileReportToReport(profileReport)
	moduleMap := make(map[string]float64)
	for _, m := range report.Modules {
		moduleMap[m.Module] = m.Percent
	}
	moduleOrder := coverage.KnownEnforcedModules()
	for _, moduleName := range moduleOrder {
		minPercent, hasThreshold := parsed.minModulePercents[moduleName]
		if !hasThreshold {
			continue
		}
		actualPercent, exists := moduleMap[moduleName]
		if !exists {
			continue // skip missing modules in OK output
		}
		fmt.Fprintf(stdout, "coverage: module %s=%.1f%% min=%.1f%% OK\n", moduleName, actualPercent, minPercent)
	}

	// Print module breakdown by default
	if parsed.showBreakdown {
		profileReport.PrintModuleTableTo(stdout)
	}

	// Write JSON output if requested
	if parsed.jsonOutputPath != "" {
		jsonData, err := profileReport.ToJSON()
		if err != nil {
			fmt.Fprintf(stderr, "coverage: error generating JSON: %v\n", err)
			return 1
		}
		if err := os.WriteFile(parsed.jsonOutputPath, jsonData, 0644); err != nil {
			fmt.Fprintf(stderr, "coverage: error writing JSON: %v\n", err)
			return 1
		}
	}

	return 0
}

// handleFactoryCoverage handles the `leamas factory coverage` command.
// It checks a coverage profile against a threshold and optionally generates a module breakdown.
func handleFactoryCoverage() {
	os.Exit(runFactoryCoverage(os.Args[3:], os.Stdout, os.Stderr))
}

func printCoverageUsage() {
	fmt.Println("Usage: leamas factory coverage --profile <path> --min-total <float> [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --profile <path>               Path to coverage profile (required)")
	fmt.Println("  --min-total <float>            Minimum total coverage percentage (required)")
	fmt.Println("  --min-module <module>=<float>  Minimum coverage for a module (can be repeated)")
	fmt.Println("  --default-module-floors       Apply default module floors (optional)")
	fmt.Println("  --thresholds [--json]          Print canonical default thresholds")
	fmt.Println("  --json-output <path>           Write module breakdown JSON to file (optional)")
	fmt.Println("  --breakdown                    Show module breakdown (default: true)")
	fmt.Println("  --no-breakdown                 Hide module breakdown")
}

func printCoverageUsageTo(w io.Writer) {
	fmt.Fprintln(w, "Usage: leamas factory coverage --profile <path> --min-total <float> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --profile <path>               Path to coverage profile (required)")
	fmt.Fprintln(w, "  --min-total <float>            Minimum total coverage percentage (required)")
	fmt.Fprintln(w, "  --min-module <module>=<float>  Minimum coverage for a module (can be repeated)")
	fmt.Fprintln(w, "  --default-module-floors       Apply default module floors (optional)")
	fmt.Fprintln(w, "  --thresholds [--json]          Print canonical default thresholds")
	fmt.Fprintln(w, "  --json-output <path>           Write module breakdown JSON to file (optional)")
	fmt.Fprintln(w, "  --breakdown                    Show module breakdown (default: true)")
	fmt.Fprintln(w, "  --no-breakdown                 Hide module breakdown")
}

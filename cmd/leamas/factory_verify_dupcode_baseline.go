// Package main provides factory verify dupcode-baseline handler.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/dupcode"
)

// Default thresholds for the baseline policy
const (
	DefaultBaselineMinLines  = 40
	DefaultBaselineMinTokens = 400
)

// jsonError represents a JSON error response.
type jsonError struct {
	Error string `json:"error"`
}

// printJSONAndExit marshals v as JSON and exits with the given code.
func printJSONAndExit(v any, code int) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal JSON: %v\n", err)
		os.Exit(2)
	}
	fmt.Println(string(data))
	os.Exit(code)
}

func handleFactoryVerifyDupcodeBaseline() {
	// Reset flag state for this subcommand
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: leamas factory verify dupcode-baseline [options]\n")
		flag.CommandLine.PrintDefaults()
	}

	// Parse flags
	baselinePath := flag.String("baseline", ".factory/dupcode-baseline.json", "Path to baseline file")
	minLines := flag.Int("min-lines", DefaultBaselineMinLines, "Expected minimum lines threshold")
	minTokens := flag.Int("min-tokens", DefaultBaselineMinTokens, "Expected minimum tokens threshold")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")

	// Parse only the arguments after "dupcode-baseline"
	// os.Args = ["leamas", "factory", "verify", "dupcode-baseline", ...]
	// We want to parse ["dupcode-baseline", ...]
	args := os.Args[4:] // Skip "leamas factory verify"
	if err := flag.CommandLine.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		if *jsonOutput {
			printJSONAndExit(jsonError{Error: fmt.Sprintf("flag parse error: %v", err)}, 2)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(2)
		}
	}

	// Build policy
	policy := dupcode.BaselinePolicy{
		Path:      *baselinePath,
		MinLines:  *minLines,
		MinTokens: *minTokens,
	}

	// Run verification
	findings, err := dupcode.VerifyBaseline(".", policy)
	if err != nil {
		if *jsonOutput {
			printJSONAndExit(jsonError{Error: fmt.Sprintf("baseline verification error: %v", err)}, 2)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(2)
		}
	}

	// Print results
	if *jsonOutput {
		type jsonFinding struct {
			Path    string `json:"path"`
			Kind    string `json:"kind"`
			Message string `json:"message"`
		}
		var findingsList []jsonFinding
		for _, f := range findings {
			findingsList = append(findingsList, jsonFinding{
				Path:    f.Path,
				Kind:    f.Kind,
				Message: f.Message,
			})
		}

		type jsonResult struct {
			Status   string        `json:"status"`
			Baseline string        `json:"baseline"`
			Findings []jsonFinding `json:"findings,omitempty"`
		}

		result := jsonResult{
			Baseline: *baselinePath,
			Findings: findingsList,
		}
		if len(findings) == 0 {
			result.Status = "ok"
		} else {
			result.Status = "failed"
		}

		code := 0
		if len(findings) > 0 {
			code = 1
		}
		printJSONAndExit(result, code)
	}

	code := dupcode.PrintBaselineVerifyResult("dupcode baseline", findings)
	os.Exit(code)
}

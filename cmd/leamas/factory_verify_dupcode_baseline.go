// Package main provides factory verify dupcode-baseline handler.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
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

// renderDupcodeBaselineDispatchFailure evaluates dispatcher failure channels
// for the standalone dupcode-baseline command. The order is:
//  1. Dispatch.Error
//  2. Dispatch.Findings
//
// Returns exitCode and failed=true when a failure channel was rendered.
func renderDupcodeBaselineDispatchFailure(
	result verifierdispatch.Result,
	jsonOutput bool,
	stdout io.Writer,
	stderr io.Writer,
) (int, bool) {
	if result.Error != nil {
		if jsonOutput {
			printJSONAndExit(jsonError{Error: fmt.Sprintf("dispatcher error: %v", result.Error)}, 1)
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", result.Error)
		}
		return 1, true
	}
	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if jsonOutput {
			type findingError struct {
				Error string `json:"error"`
				Kind  string `json:"kind"`
			}
			printJSONAndExit(findingError{Error: f.Message, Kind: f.Kind}, 1)
		} else {
			fmt.Fprintf(stderr, "Error: %s\n", f.Message)
		}
		return 1, true
	}
	return 0, false
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

	// Construct the data-only spec; no executable closures are built.
	spec := gate.DupcodeBaselineSpec{
		BaselinePath: *baselinePath,
		MinLines:     *minLines,
		MinTokens:    *minTokens,
	}

	// Run verification through the typed dispatch entry point. The command
	// layer never sees the adapter or any executable surface.
	ctx := context.Background()
	result := gate.DispatchDupcodeBaselineVerifyTyped(ctx, ".", spec)

	stdout := io.Writer(os.Stdout)
	stderr := io.Writer(os.Stderr)

	if exitCode, failed := renderDupcodeBaselineDispatchFailure(result.Dispatch, *jsonOutput, stdout, stderr); failed {
		os.Exit(exitCode)
	}

	// Get the typed findings from the authorized runner
	findings := result.Dispatch.Findings

	// Print results with proper exit semantics
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

	code := gate.DupcodeBaselinePrintResult("dupcode baseline", findings)
	os.Exit(code)
}

// _ ensures checks import is referenced; the dupcode_baseline path uses
// checks.Finding through the typed dispatch outcome.
var _ = []checks.Finding(nil)

// Package main provides factory verify dupcode-baseline handler.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierdispatch"
)

// Default thresholds for the baseline policy
const (
	DefaultBaselineMinLines  = 40
	DefaultBaselineMinTokens = 400
)

// dupcodeBaselineTypedDispatcher is the conceptual dispatcher signature
// used by the standalone dupcode-baseline renderer. Tests inject counting
// implementations through the same signature.
type dupcodeBaselineTypedDispatcher func(
	context.Context,
	string,
	gate.DupcodeBaselineSpec,
) gate.DupcodeBaselineOutcome

// jsonError represents a JSON error response.
type jsonError struct {
	Error string `json:"error"`
}

// writeJSON marshals v as JSON to the supplied writer. It never terminates
// the process. Errors are reported via the return value.
func writeJSON(output io.Writer, value any) error {
	enc := json.NewEncoder(output)
	return enc.Encode(value)
}

// runFactoryVerifyDupcodeBaseline parses args, dispatches the typed
// dupcode-baseline request, and writes the human or JSON output to the
// supplied writers. It returns the exit code but never terminates the
// process. Only the outer production wrapper may call os.Exit.
func runFactoryVerifyDupcodeBaseline(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	dispatch dupcodeBaselineTypedDispatcher,
) int {
	fs := flag.NewFlagSet("dupcode-baseline", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: leamas factory verify dupcode-baseline [options]\n")
		fs.PrintDefaults()
	}

	baselinePath := fs.String("baseline", ".factory/dupcode-baseline.json", "Path to baseline file")
	minLines := fs.Int("min-lines", DefaultBaselineMinLines, "Expected minimum lines threshold")
	minTokens := fs.Int("min-tokens", DefaultBaselineMinTokens, "Expected minimum tokens threshold")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		_ = err
		if err == flag.ErrHelp {
			fs.Usage()
			return 0
		}
		if *jsonOutput {
			_ = writeJSON(stdout, jsonError{Error: fmt.Sprintf("flag parse error: %v", err)})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return 2
	}

	if fs.NArg() > 0 {
		if *jsonOutput {
			_ = writeJSON(stdout, jsonError{Error: fmt.Sprintf("unexpected positional argument: %s", fs.Arg(0))})
		} else {
			fmt.Fprintf(stderr, "Error: unexpected positional argument: %s\n", fs.Arg(0))
		}
		return 2
	}

	spec := gate.DupcodeBaselineSpec{
		BaselinePath: *baselinePath,
		MinLines:     *minLines,
		MinTokens:    *minTokens,
	}

	ctx := context.Background()
	outcome := dispatch(ctx, ".", spec)

	if exitCode, failed := renderDupcodeBaselineDispatchFailure(outcome.Dispatch, *jsonOutput, stdout, stderr); failed {
		return exitCode
	}

	findings := outcome.Dispatch.Findings
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
		_ = writeJSON(stdout, result)
	} else {
		code := gate.DupcodeBaselinePrintResult("dupcode baseline", findings)
		return code
	}

	if len(findings) > 0 {
		return 1
	}
	return 0
}

// renderDupcodeBaselineDispatchFailure evaluates dispatcher failure
// channels. The order is:
//  1. Dispatch.Error
//  2. Dispatch.Findings
//
// Returns (exitCode, failed=true) when a failure channel was rendered.
// Never terminates the process and never writes to os.Stdout / os.Stderr
// directly; all output goes through the supplied writers.
func renderDupcodeBaselineDispatchFailure(
	result verifierdispatch.Result,
	jsonOutput bool,
	stdout io.Writer,
	stderr io.Writer,
) (int, bool) {
	if result.Error != nil {
		if jsonOutput {
			_ = writeJSON(stdout, jsonError{Error: fmt.Sprintf("dispatcher error: %v", result.Error)})
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
			_ = writeJSON(stdout, findingError{Error: f.Message, Kind: f.Kind})
		} else {
			fmt.Fprintf(stderr, "Error: %s\n", f.Message)
		}
		return 1, true
	}
	return 0, false
}

// handleFactoryVerifyDupcodeBaseline is the production command handler.
// Only this outer wrapper is permitted to call osExit; all inner
// rendering and dispatch helpers are return-based.
func handleFactoryVerifyDupcodeBaseline() {
	osExit(runFactoryVerifyDupcodeBaseline(
		os.Args[4:],
		os.Stdout,
		os.Stderr,
		gate.DispatchDupcodeBaselineVerifyTyped,
	))
}

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
	"github.com/s1onique/leamas/internal/factory/dupcode"
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

// writeJSON marshals v as JSON to the supplied writer. It returns the
// encoding/write error so callers can handle bounded failures
// explicitly. The function never terminates the process; it never
// reports success classification when the write fails.
func writeJSON(output io.Writer, value any) error {
	enc := json.NewEncoder(output)
	return enc.Encode(value)
}

// runFactoryVerifyDupcodeBaseline parses args, dispatches the typed
// dupcode-baseline request, and writes the human or JSON output to the
// supplied writers. It returns the exit code but never terminates the
// process. Only the outer production wrapper may call os.Exit.
//
// Output contract:
//   - Human success: only "dupcode baseline: OK\n" on supplied stdout.
//   - Human failure: "dupcode baseline: FAILED\n" plus findings on
//     supplied stdout.
//   - JSON: the JSON object is written to supplied stdout. A write
//     failure exits 2 with a diagnostic on supplied stderr; no success
//     classification is emitted.
func runFactoryVerifyDupcodeBaseline(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	dispatch dupcodeBaselineTypedDispatcher,
) int {
	fs := flag.NewFlagSet("dupcode-baseline", flag.ContinueOnError)
	// Discard flag-set-level error output during parsing so we can
	// route diagnostics through the supplied stderr instead.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		// Render usage through the supplied stderr while temporarily
		// directing the FlagSet output there for the duration of
		// PrintDefaults. This restores the help emission that the
		// previous contract required: all flags visible on stderr,
		// nothing on stdout.
		previous := fs.Output()
		fs.SetOutput(stderr)
		fmt.Fprintln(stderr, "Usage: leamas factory verify dupcode-baseline [options]")
		fs.PrintDefaults()
		fs.SetOutput(previous)
	}

	baselinePath := fs.String("baseline", ".factory/dupcode-baseline.json", "Path to baseline file")
	minLines := fs.Int("min-lines", DefaultBaselineMinLines, "Expected minimum lines threshold")
	minTokens := fs.Int("min-tokens", DefaultBaselineMinTokens, "Expected minimum tokens threshold")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fs.Usage()
			return 0
		}
		// FlagSet-level error reporting is suppressed (io.Discard);
		// emit the diagnostic through the supplied writers only.
		if *jsonOutput {
			if jerr := writeJSON(stdout, jsonError{Error: fmt.Sprintf("flag parse error: %v", err)}); jerr != nil {
				fmt.Fprintf(stderr, "json write failure: %v\n", jerr)
				return 2
			}
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return 2
	}

	if fs.NArg() > 0 {
		if *jsonOutput {
			if jerr := writeJSON(stdout, jsonError{Error: fmt.Sprintf("unexpected positional argument: %s", fs.Arg(0))}); jerr != nil {
				fmt.Fprintf(stderr, "json write failure: %v\n", jerr)
				return 2
			}
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
		if jerr := writeJSON(stdout, result); jerr != nil {
			fmt.Fprintf(stderr, "json write failure: %v\n", jerr)
			return 2
		}
	} else {
		// Writer-aware human rendering. Nothing is written to process
		// stdout directly; the supplied stdout carries the success or
		// failure payload exclusively.
		code := dupcode.PrintBaselineVerifyResultTo(stdout, "dupcode baseline", findings)
		return code
	}

	if len(findings) > 0 {
		return 1
	}
	return 0
}

// isAuthorityDenialFinding reports whether a finding originates from
// the dispatcher's authority-denied channel. The dispatcher attaches
// findings of this kind when authority is denied; all other findings
// are normal verification failures and flow through the success path.
//
// This function takes checks.Finding as a parameter, which is the
// real (non-blank-identifier) use of the checks package in this
// command file.
func isAuthorityDenialFinding(f checks.Finding) bool {
	return f.Kind == "verifier_execution_authority_denied"
}

// renderDupcodeBaselineDispatchFailure evaluates dispatcher failure
// channels. The order is:
//  1. Dispatch.Error
//  2. Dispatch.Findings carrying an authority-denial kind
//
// Returns (exitCode, failed=true) when a failure channel was rendered.
// Never terminates the process and never writes to os.Stdout / os.Stderr
// directly; all output goes through the supplied writers. JSON write
// failures during failure-channel rendering return exit code 2 with a
// diagnostic on supplied stderr.
func renderDupcodeBaselineDispatchFailure(
	result verifierdispatch.Result,
	jsonOutput bool,
	stdout io.Writer,
	stderr io.Writer,
) (int, bool) {
	if result.Error != nil {
		if jsonOutput {
			if jerr := writeJSON(stdout, jsonError{Error: fmt.Sprintf("dispatcher error: %v", result.Error)}); jerr != nil {
				fmt.Fprintf(stderr, "json write failure: %v\n", jerr)
				return 2, true
			}
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", result.Error)
		}
		return 1, true
	}
	if len(result.Findings) > 0 && isAuthorityDenialFinding(result.Findings[0]) {
		f := result.Findings[0]
		if jsonOutput {
			type findingError struct {
				Error string `json:"error"`
				Kind  string `json:"kind"`
			}
			if jerr := writeJSON(stdout, findingError{Error: f.Message, Kind: f.Kind}); jerr != nil {
				fmt.Fprintf(stderr, "json write failure: %v\n", jerr)
				return 2, true
			}
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

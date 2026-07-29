// Package main provides factory verify dupcode handler with baseline support.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// Exit codes for the dupcode subcommand.
const (
	ExitSuccess          = 0
	ExitAuthorityFailure = 1
	ExitParseFailure     = 2
)

// Default thresholds for the quality gate.
const (
	DefaultMinLines  = 40
	DefaultMinTokens = 400
)

// BaselineDefaultPath is the default path for the baseline file.
const BaselineDefaultPath = ".factory/dupcode-baseline.json"

// osExit is used by the production wrapper to exit with the handler's return code.
var osExit = os.Exit

// dupcodeCommandArgs extracts the dupcode subcommand args from argv.
// After the upper router has validated "factory verify dupcode", this removes
// the four routing tokens and returns the remaining option flags.
// Returns: (remaining flags, ok) where ok=false means malformed invocation.
func dupcodeCommandArgs(argv []string) ([]string, bool) {
	if len(argv) < 4 {
		return nil, false
	}
	return argv[4:], true
}

// printDupcodeUsage prints the usage information for the dupcode command.
func printDupcodeUsage(fs *flag.FlagSet, output io.Writer) {
	fmt.Fprintln(output, "Usage: leamas factory verify dupcode [options]")
	previous := fs.Output()
	fs.SetOutput(output)
	fs.PrintDefaults()
	fs.SetOutput(previous)
}

// handleFactoryVerifyDupcode is the production entry point for the dupcode subcommand.
// It extracts args using dupcodeCommandArgs and passes them to handleDupcode.
func handleFactoryVerifyDupcode() {
	args, ok := dupcodeCommandArgs(os.Args)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: insufficient arguments for dupcode command")
		osExit(ExitParseFailure)
		return
	}
	osExit(handleDupcode(args, os.Stdout, os.Stderr))
}

// handleDupcode is the internal handler. It parses flags, constructs a
// data-only dispatch spec, and routes through the typed dispatch entry
// point. The command layer holds no adapter surface or executable factories.
func handleDupcode(args []string, stdout, stderr io.Writer) int {
	// Reset flag state for this subcommand
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	// Use a discard writer for FlagSet output to avoid mixing with user-visible output.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		// Usage is printed below when --help is detected
	}

	// Parse flags for dupcode subcommand
	baselinePath := fs.String("baseline", BaselineDefaultPath, "Path to baseline file")
	updateBaseline := fs.Bool("update-baseline", false, "Update baseline file with current findings")
	minLines := fs.Int("min-lines", DefaultMinLines, "Minimum lines for duplicate block")
	minTokens := fs.Int("min-tokens", DefaultMinTokens, "Minimum tokens for duplicate block")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")

	// Parse arguments
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			// Print usage and return success
			printDupcodeUsage(fs, stderr)
			return ExitSuccess
		}
		if *jsonOutput {
			// JSON parse failures go to stdout for compatibility
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("flag parse error: %v", err)})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitParseFailure
	}

	// Check for unexpected positional arguments
	if fs.NArg() > 0 {
		if *jsonOutput {
			// JSON parse failures go to stdout for compatibility
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("unexpected positional argument: %s", fs.Arg(0))})
		} else {
			fmt.Fprintf(stderr, "Error: unexpected positional argument: %s\n", fs.Arg(0))
		}
		return ExitParseFailure
	}

	ctx := context.Background()

	if *updateBaseline {
		spec := gate.DupcodeUpdateBaselineSpec{
			BaselinePath: *baselinePath,
			MinLines:     *minLines,
			MinTokens:    *minTokens,
		}
		return renderUpdateBaselineResult(ctx, spec, *jsonOutput, stdout, stderr)
	}

	spec := gate.DupcodeVerifySpec{
		BaselinePath: *baselinePath,
		MinLines:     *minLines,
		MinTokens:    *minTokens,
	}
	return renderVerifyBaselineResult(ctx, spec, *jsonOutput, stdout, stderr)
}

// renderUpdateBaselineResult drives the typed dispatch and renders output.
// The command layer holds only data (spec, stdout, stderr) and never touches
// the adapter or dupcode packages.
func renderUpdateBaselineResult(ctx context.Context, spec gate.DupcodeUpdateBaselineSpec, jsonOutput bool, stdout, stderr io.Writer) int {
	result := gate.DispatchDupcodeUpdateBaselineTyped(ctx, ".", spec)

	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{
				"error": f.Message,
				"kind":  f.Kind,
			})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", f.Message)
		}
		return ExitAuthorityFailure
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", result.Error)
		}
		return ExitAuthorityFailure
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.Encode(map[string]interface{}{
			"baseline":   spec.BaselinePath,
			"thresholds": map[string]int{"min_lines": spec.MinLines, "min_tokens": spec.MinTokens},
		})
	} else {
		fmt.Fprintf(stdout, "Baseline written to: %s\n", spec.BaselinePath)
		fmt.Fprintf(stdout, "Thresholds: min_lines=%d, min_tokens=%d\n", spec.MinLines, spec.MinTokens)
	}

	return ExitSuccess
}

// renderVerifyBaselineResult drives the typed dispatch and renders output.
func renderVerifyBaselineResult(ctx context.Context, spec gate.DupcodeVerifySpec, jsonOutput bool, stdout, stderr io.Writer) int {
	result := gate.DispatchDupcodeVerifyTyped(ctx, ".", spec)

	if len(result.Findings) > 0 {
		f := result.Findings[0]
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{
				"error": f.Message,
				"kind":  f.Kind,
			})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", f.Message)
		}
		return ExitAuthorityFailure
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", result.Error)
		}
		return ExitAuthorityFailure
	}

	// Successful dispatch produced no findings — render the OK result.
	// The dupcode compare result is in result.Findings (may be empty).
	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.Encode(map[string]interface{}{
			"baseline":   spec.BaselinePath,
			"thresholds": map[string]int{"min_lines": spec.MinLines, "min_tokens": spec.MinTokens},
			"ok":         len(result.Findings) == 0,
		})
	} else {
		if len(result.Findings) > 0 {
			fmt.Fprintf(stdout, "Duplicate code violations found: %d\n", len(result.Findings))
		} else {
			fmt.Fprintf(stdout, "No duplicate code violations found.\n")
		}
	}

	return ExitSuccess
}

// ValidateDupcodeAuthorityWithOperation validates dupcode authority for a specific operation.
func ValidateDupcodeAuthorityWithOperation(operation verifierauthority.VerifierOperation) error {
	ctx := context.Background()
	ec, err := gate.DetectDupcodeExecutionContext(ctx, ".")
	if err != nil {
		return err
	}
	return gate.ValidateDupcodeExecutionAuthority(ec, operation)
}

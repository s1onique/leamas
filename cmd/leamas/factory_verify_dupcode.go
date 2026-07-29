// Package main provides factory verify dupcode handler with baseline support.
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
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

const (
	ExitSuccess          = 0
	ExitAuthorityFailure = 1
	ExitParseFailure     = 2
)

const (
	DefaultMinLines  = 40
	DefaultMinTokens = 400
)

const BaselineDefaultPath = ".factory/dupcode-baseline.json"

var osExit = os.Exit

func dupcodeCommandArgs(argv []string) ([]string, bool) {
	if len(argv) < 4 {
		return nil, false
	}
	return argv[4:], true
}

func printDupcodeUsage(fs *flag.FlagSet, output io.Writer) {
	fmt.Fprintln(output, "Usage: leamas factory verify dupcode [options]")
	previous := fs.Output()
	fs.SetOutput(output)
	fs.PrintDefaults()
	fs.SetOutput(previous)
}

func handleFactoryVerifyDupcode() {
	args, ok := dupcodeCommandArgs(os.Args)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: insufficient arguments for dupcode command")
		osExit(ExitParseFailure)
		return
	}
	osExit(handleDupcode(args, os.Stdout, os.Stderr))
}

func handleDupcode(args []string, stdout, stderr io.Writer) int {
	return handleDupcodeWith(args, stdout, stderr, dupcodeTypedDispatchers{
		verify:         gate.DispatchDupcodeVerifyTyped,
		updateBaseline: gate.DispatchDupcodeUpdateBaselineTyped,
	})
}

type dupcodeTypedDispatchers struct {
	verify         func(context.Context, string, gate.DupcodeVerifySpec) gate.DupcodeVerifyOutcome
	updateBaseline func(context.Context, string, gate.DupcodeUpdateBaselineSpec) gate.DupcodeUpdateBaselineOutcome
}

func handleDupcodeWith(args []string, stdout, stderr io.Writer, dispatchers dupcodeTypedDispatchers) int {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	baselinePath := fs.String("baseline", BaselineDefaultPath, "Path to baseline file")
	updateBaseline := fs.Bool("update-baseline", false, "Update baseline file with current findings")
	minLines := fs.Int("min-lines", DefaultMinLines, "Minimum lines for duplicate block")
	minTokens := fs.Int("min-tokens", DefaultMinTokens, "Minimum tokens for duplicate block")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printDupcodeUsage(fs, stderr)
			return ExitSuccess
		}
		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			_ = enc.Encode(map[string]interface{}{"error": fmt.Sprintf("flag parse error: %v", err)})
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return ExitParseFailure
	}

	if fs.NArg() > 0 {
		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			_ = enc.Encode(map[string]interface{}{"error": fmt.Sprintf("unexpected positional argument: %s", fs.Arg(0))})
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
		return renderUpdateBaselineResult(ctx, spec, *jsonOutput, stdout, stderr, dispatchers)
	}

	spec := gate.DupcodeVerifySpec{
		BaselinePath: *baselinePath,
		MinLines:     *minLines,
		MinTokens:    *minTokens,
	}
	return renderVerifyBaselineResult(ctx, spec, *jsonOutput, stdout, stderr, dispatchers)
}

func renderUpdateBaselineResult(ctx context.Context, spec gate.DupcodeUpdateBaselineSpec, jsonOutput bool, stdout, stderr io.Writer, dispatchers dupcodeTypedDispatchers) int {
	result := dispatchers.updateBaseline(ctx, ".", spec)
	if result.Dispatch.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			_ = enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Dispatch.Error)})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", result.Dispatch.Error)
		}
		return ExitAuthorityFailure
	}
	if jsonOutput {
		enc := json.NewEncoder(stdout)
		_ = enc.Encode(map[string]interface{}{
			"baseline":   spec.BaselinePath,
			"thresholds": map[string]int{"min_lines": spec.MinLines, "min_tokens": spec.MinTokens},
			"findings":   result.Report.Findings,
			"scan_report": map[string]interface{}{
				"min_lines": result.Report.MinLines, "min_tokens": result.Report.MinTokens,
				"finding_count": result.Report.FindingCount,
			},
		})
	} else {
		fmt.Fprintf(stdout, "Baseline written to: %s\n", spec.BaselinePath)
		fmt.Fprintf(stdout, "Thresholds: min_lines=%d, min_tokens=%d\n", spec.MinLines, spec.MinTokens)
		fmt.Fprintf(stdout, "Scan found %d duplicate blocks\n", result.Report.FindingCount)
	}
	return ExitSuccess
}

func renderVerifyBaselineResult(ctx context.Context, spec gate.DupcodeVerifySpec, jsonOutput bool, stdout, stderr io.Writer, dispatchers dupcodeTypedDispatchers) int {
	result := dispatchers.verify(ctx, ".", spec)
	if result.Dispatch.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(stdout)
			_ = enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Dispatch.Error)})
		} else {
			fmt.Fprintf(stderr, "dupcode: %v\n", result.Dispatch.Error)
		}
		return ExitAuthorityFailure
	}
	if jsonOutput {
		enc := json.NewEncoder(stdout)
		_ = enc.Encode(map[string]interface{}{
			"has_changes":       result.Comparison.HasChanges,
			"new_count":         result.Comparison.NewCount,
			"worsened_count":    result.Comparison.WorsenedCount,
			"new_findings":      result.Comparison.NewFindings,
			"worsened_findings": result.Comparison.WorsenedFindings,
			"current_report":    result.Report,
		})
	} else {
		if result.Comparison.HasChanges {
			fmt.Fprintf(stdout, "Duplicate code violations found:\n")
			fmt.Fprintf(stdout, "  New: %d\n", result.Comparison.NewCount)
			fmt.Fprintf(stdout, "  Worsened: %d\n", result.Comparison.WorsenedCount)
		} else {
			fmt.Fprintf(stdout, "No duplicate code violations found.\n")
		}
	}
	return ExitSuccess
}

func ValidateDupcodeAuthorityWithOperation(operation verifierauthority.VerifierOperation) error {
	ctx := context.Background()
	ec, err := gate.DetectDupcodeExecutionContext(ctx, ".")
	if err != nil {
		return err
	}
	return gate.ValidateDupcodeExecutionAuthority(ec, operation)
}

var _ = checks.FileExists

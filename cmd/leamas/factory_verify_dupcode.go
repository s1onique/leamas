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
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
)

// Exit codes for the dupcode subcommand.
const (
	ExitSuccess          = 0
	ExitAuthorityFailure = 1
	ExitParseFailure     = 2
)

// dupcodeDispatchers holds the dispatch functions for verify and update operations.
type dupcodeDispatchers struct {
	verify         gate.DispatchFunc
	updateBaseline gate.DispatchFunc
}

// productionDupcodeDispatchers uses the real gate dispatchers.
var productionDupcodeDispatchers = dupcodeDispatchers{
	verify:         gate.DispatchDupcodeVerify,
	updateBaseline: gate.DispatchDupcodeUpdateBaseline,
}

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
// Expected format: ["leamas", "factory", "verify", "dupcode", ...flags]
// Returns: [...flags] or nil if insufficient args.
func dupcodeCommandArgs(argv []string) []string {
	if len(argv) < 5 {
		return nil
	}
	return argv[4:]
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
	args := dupcodeCommandArgs(os.Args)
	if args == nil {
		fmt.Fprintln(os.Stderr, "Error: insufficient arguments for dupcode command")
		osExit(ExitParseFailure)
		return
	}
	osExit(handleDupcode(args, productionDupcodeDispatchers, os.Stdout, os.Stderr))
}

// handleDupcode is the internal handler that accepts dispatchers for testing.
// It parses flags and routes to either verify or update handler based on flags.
// Returns an exit code: 0 = success, 1 = authority/gate failure, 2 = parse failure.
func handleDupcode(args []string, dispatchers dupcodeDispatchers, stdout, stderr io.Writer) int {
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

	// Build config using protectedverifier adapter
	cfg := protectedverifier.DefaultConfig()
	cfg.MinLines = *minLines
	cfg.MinTokens = *minTokens

	if *updateBaseline {
		return handleUpdateBaselineWithDispatch(*baselinePath, cfg, *jsonOutput, stdout, stderr, dispatchers.updateBaseline)
	}

	return handleVerifyBaselineWithDispatch(*baselinePath, cfg, *jsonOutput, stdout, stderr, dispatchers.verify)
}

// handleUpdateBaselineWithDispatch runs the update baseline handler with the given dispatcher.
func handleUpdateBaselineWithDispatch(baselinePath string, cfg protectedverifier.Config, jsonOutput bool, stdout, stderr io.Writer, dispatcher gate.DispatchFunc) int {
	ctx := context.Background()

	runner := protectedverifier.NewDupcodeRunner()
	var scanReport protectedverifier.Report

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			report, err := runner.RunCheckReport(root, cfg)
			if err != nil {
				return []checks.Finding{
					{
						Path:     "dupcode",
						Kind:     "error",
						Message:  fmt.Sprintf("scan failed: %v", err),
						Severity: checks.SeverityError,
					},
				}
			}

			scanReport = report

			if err := runner.WriteBaseline(baselinePath, report); err != nil {
				return []checks.Finding{
					{
						Path:     "dupcode",
						Kind:     "error",
						Message:  fmt.Sprintf("failed to write baseline: %v", err),
						Severity: checks.SeverityError,
					},
				}
			}

			return nil
		}
	}

	result := dispatcher(ctx, ".", runnerFactory)

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

	findingsCount := len(scanReport.Findings)

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.Encode(map[string]interface{}{
			"baseline":    baselinePath,
			"findings":    findingsCount,
			"thresholds":  map[string]int{"min_lines": cfg.MinLines, "min_tokens": cfg.MinTokens},
			"scan_report": scanReport,
		})
	} else {
		fmt.Fprintf(stdout, "Baseline written to: %s\n", baselinePath)
		fmt.Fprintf(stdout, "Thresholds: min_lines=%d, min_tokens=%d\n", cfg.MinLines, cfg.MinTokens)
		fmt.Fprintf(stdout, "Scan found %d duplicate blocks\n", findingsCount)
	}

	return ExitSuccess
}

// handleVerifyBaselineWithDispatch runs the verify baseline handler with the given dispatcher.
func handleVerifyBaselineWithDispatch(baselinePath string, cfg protectedverifier.Config, jsonOutput bool, stdout, stderr io.Writer, dispatcher gate.DispatchFunc) int {
	ctx := context.Background()

	runner := protectedverifier.NewDupcodeRunner()
	var compareResult protectedverifier.CompareResult
	var report protectedverifier.Report

	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			baseline, err := runner.LoadBaseline(baselinePath)
			if err != nil {
				return []checks.Finding{
					{
						Path:     "dupcode",
						Kind:     "error",
						Message:  fmt.Sprintf("failed to load baseline: %v", err),
						Severity: checks.SeverityError,
					},
				}
			}

			scanReport, err := runner.RunCheckReport(root, cfg)
			if err != nil {
				return []checks.Finding{
					{
						Path:     "dupcode",
						Kind:     "error",
						Message:  fmt.Sprintf("scan failed: %v", err),
						Severity: checks.SeverityError,
					},
				}
			}

			report = scanReport
			comparison := runner.CompareToBaseline(scanReport, baseline)
			compareResult = comparison

			var findings []checks.Finding
			if comparison.HasChanges {
				for _, f := range comparison.NewFindings {
					findings = append(findings, checks.Finding{
						Path:     "dupcode",
						Kind:     "new_duplicate",
						Message:  fmt.Sprintf("new duplicate (fingerprint: %s, tokens: %d)", f.Fingerprint, f.TokenCount),
						Severity: checks.SeverityError,
					})
				}
				for _, f := range comparison.WorsenedFindings {
					findings = append(findings, checks.Finding{
						Path:     "dupcode",
						Kind:     "worsened_duplicate",
						Message:  fmt.Sprintf("worsened duplicate (fingerprint: %s)", f.Fingerprint),
						Severity: checks.SeverityError,
					})
				}
			}

			return findings
		}
	}

	result := dispatcher(ctx, ".", runnerFactory)

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
			"has_changes":       compareResult.HasChanges,
			"new_count":         len(compareResult.NewFindings),
			"worsened_count":    len(compareResult.WorsenedFindings),
			"new_findings":      compareResult.NewFindings,
			"worsened_findings": compareResult.WorsenedFindings,
			"current_report":    report,
		})
	} else {
		if compareResult.HasChanges {
			fmt.Fprintf(stdout, "Duplicate code violations found:\n")
			fmt.Fprintf(stdout, "  New: %d\n", len(compareResult.NewFindings))
			fmt.Fprintf(stdout, "  Worsened: %d\n", len(compareResult.WorsenedFindings))
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

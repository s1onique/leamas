// Package main provides factory verify dupcode handler with baseline support.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/checks"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/protectedverifier"
	"github.com/s1onique/leamas/internal/factory/verifierauthority"
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

// osExit is injectable for testing.
var osExit = os.Exit

// Default thresholds for the quality gate
const (
	DefaultMinLines  = 40
	DefaultMinTokens = 400
)

// BaselineDefaultPath is the default path for the baseline file.
const BaselineDefaultPath = ".factory/dupcode-baseline.json"

// handleFactoryVerifyDupcode is the production entry point for the dupcode subcommand.
func handleFactoryVerifyDupcode() {
	handleDupcode(os.Args[1:], productionDupcodeDispatchers)
}

// handleDupcode is the internal handler that accepts dispatchers for testing.
// It parses flags and routes to either verify or update handler based on flags.
func handleDupcode(args []string, dispatchers dupcodeDispatchers) {
	// Reset flag state for this subcommand
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: leamas factory verify dupcode [options]\n")
		fs.PrintDefaults()
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
			osExit(0)
		}
		if *jsonOutput {
			fmt.Printf(`{"error": "flag parse error: %v"}`, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		osExit(2)
	}

	// Check for unexpected positional arguments
	if fs.NArg() > 0 {
		if *jsonOutput {
			fmt.Printf(`{"error": "unexpected positional argument: %s"}`, fs.Arg(0))
		} else {
			fmt.Fprintf(os.Stderr, "Error: unexpected positional argument: %s\n", fs.Arg(0))
		}
		osExit(2)
	}

	// Build config using protectedverifier adapter
	cfg := protectedverifier.DefaultConfig()
	cfg.MinLines = *minLines
	cfg.MinTokens = *minTokens

	if *updateBaseline {
		handleUpdateBaselineWithDispatch(*baselinePath, cfg, *jsonOutput, dispatchers.updateBaseline)
		return
	}

	handleVerifyBaselineWithDispatch(*baselinePath, cfg, *jsonOutput, dispatchers.verify)
}

// handleUpdateBaselineWithDispatch runs the update baseline handler with the given dispatcher.
func handleUpdateBaselineWithDispatch(baselinePath string, cfg protectedverifier.Config, jsonOutput bool, dispatcher gate.DispatchFunc) {
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
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{
				"error": f.Message,
				"kind":  f.Kind,
			})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", f.Message)
		}
		osExit(1)
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", result.Error)
		}
		osExit(1)
	}

	findingsCount := len(scanReport.Findings)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(map[string]interface{}{
			"baseline":    baselinePath,
			"findings":    findingsCount,
			"thresholds":  map[string]int{"min_lines": cfg.MinLines, "min_tokens": cfg.MinTokens},
			"scan_report": scanReport,
		})
	} else {
		fmt.Printf("Baseline written to: %s\n", baselinePath)
		fmt.Printf("Thresholds: min_lines=%d, min_tokens=%d\n", cfg.MinLines, cfg.MinTokens)
		fmt.Printf("Scan found %d duplicate blocks\n", findingsCount)
	}

	osExit(0)
}

// handleVerifyBaselineWithDispatch runs the verify baseline handler with the given dispatcher.
func handleVerifyBaselineWithDispatch(baselinePath string, cfg protectedverifier.Config, jsonOutput bool, dispatcher gate.DispatchFunc) {
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
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{
				"error": f.Message,
				"kind":  f.Kind,
			})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", f.Message)
		}
		osExit(1)
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", result.Error)
		}
		osExit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
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
			fmt.Printf("Duplicate code violations found:\n")
			fmt.Printf("  New: %d\n", len(compareResult.NewFindings))
			fmt.Printf("  Worsened: %d\n", len(compareResult.WorsenedFindings))
		} else {
			fmt.Printf("No duplicate code violations found.\n")
		}
	}

	osExit(0)
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

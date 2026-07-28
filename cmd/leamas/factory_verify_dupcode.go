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

// Default thresholds for the quality gate
const (
	DefaultMinLines  = 40
	DefaultMinTokens = 400
)

// BaselineDefaultPath is the default path for the baseline file.
const BaselineDefaultPath = ".factory/dupcode-baseline.json"

func handleFactoryVerifyDupcode() {
	// Reset flag state for this subcommand
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: leamas factory verify dupcode [options]\n")
		flag.CommandLine.PrintDefaults()
	}

	// Parse flags for dupcode subcommand
	baselinePath := flag.String("baseline", BaselineDefaultPath, "Path to baseline file")
	updateBaseline := flag.Bool("update-baseline", false, "Update baseline file with current findings")
	minLines := flag.Int("min-lines", DefaultMinLines, "Minimum lines for duplicate block")
	minTokens := flag.Int("min-tokens", DefaultMinTokens, "Minimum tokens for duplicate block")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")

	// Parse only the arguments after "dupcode"
	args := os.Args[4:] // Skip "leamas factory verify"
	if err := flag.CommandLine.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		if *jsonOutput {
			fmt.Printf(`{"error": "flag parse error: %v"}`, err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(2)
	}

	// Build config using protectedverifier adapter
	cfg := protectedverifier.DefaultConfig()
	cfg.MinLines = *minLines
	cfg.MinTokens = *minTokens

	if *updateBaseline {
		handleUpdateBaseline(*baselinePath, cfg, *jsonOutput)
		return
	}

	handleVerifyBaseline(*baselinePath, cfg, *jsonOutput)
}

func handleUpdateBaseline(baselinePath string, cfg protectedverifier.Config, jsonOutput bool) {
	// Route through the central dispatcher with OperationUpdateBaseline.
	// Authority validation happens BEFORE any expensive operations.
	ctx := context.Background()

	// Create runner using the protectedverifier adapter
	runner := protectedverifier.NewDupcodeRunner()

	// Create runner factory - invoked ONLY after authority validation passes
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

			// Write baseline inside the runner factory
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

			return nil // Success
		}
	}

	result := gate.DispatchDupcodeUpdateBaseline(ctx, ".", runnerFactory)

	// Handle authority denial or runner errors
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
		os.Exit(1)
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", result.Error)
		}
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(map[string]interface{}{
			"baseline":   baselinePath,
			"findings":   0,
			"thresholds": map[string]int{"min_lines": cfg.MinLines, "min_tokens": cfg.MinTokens},
		})
	} else {
		fmt.Printf("Baseline written to: %s\n", baselinePath)
		fmt.Printf("Thresholds: min_lines=%d, min_tokens=%d\n", cfg.MinLines, cfg.MinTokens)
	}

	os.Exit(0)
}

func handleVerifyBaseline(baselinePath string, cfg protectedverifier.Config, jsonOutput bool) {
	// Route through the central dispatcher with OperationVerify.
	// Authority validation happens BEFORE any expensive operations.
	ctx := context.Background()

	// Create runner using the protectedverifier adapter
	runner := protectedverifier.NewDupcodeRunner()

	// Create runner factory - invoked ONLY after authority validation passes
	runnerFactory := func() func(root string) []checks.Finding {
		return func(root string) []checks.Finding {
			// Load baseline inside the runner factory
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

			compareResult := runner.CompareToBaseline(report, baseline)

			// Convert to findings
			var findings []checks.Finding
			if compareResult.HasChanges {
				for _, f := range compareResult.NewFindings {
					findings = append(findings, checks.Finding{
						Path:     "dupcode",
						Kind:     "new_duplicate",
						Message:  fmt.Sprintf("new duplicate (fingerprint: %s, tokens: %d)", f.Fingerprint, f.TokenCount),
						Severity: checks.SeverityError,
					})
				}
				for _, f := range compareResult.WorsenedFindings {
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

	result := gate.DispatchDupcodeVerify(ctx, ".", runnerFactory)

	// Handle authority denial
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
		os.Exit(1)
	}

	if result.Error != nil {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.Encode(map[string]interface{}{"error": fmt.Sprintf("runner error: %v", result.Error)})
		} else {
			fmt.Fprintf(os.Stderr, "dupcode: %v\n", result.Error)
		}
		os.Exit(1)
	}

	// Success
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.Encode(map[string]interface{}{"has_changes": false})
	} else {
		fmt.Printf("No duplicate code violations found.\n")
	}

	os.Exit(0)
}

// ValidateDupcodeAuthorityWithOperation validates dupcode authority for a specific operation.
// This is used by handlers that need direct authority validation.
func ValidateDupcodeAuthorityWithOperation(operation verifierauthority.VerifierOperation) error {
	ctx := context.Background()
	ec, err := gate.DetectDupcodeExecutionContext(ctx, ".")
	if err != nil {
		return err
	}
	return gate.ValidateDupcodeExecutionAuthority(ec, operation)
}

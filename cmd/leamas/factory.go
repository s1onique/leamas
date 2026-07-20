// Package main provides the factory subcommand handlers.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/execution"
	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/longtest"
	"github.com/s1onique/leamas/internal/factory/output"
)

func parseFactoryCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", errors.New("missing factory command")
	}
	cmd := args[0]
	switch cmd {
	case "verify", "gate", "factorize", "digest", "coverage", "gate-summary", "output-contract", "doctrine", "test-long":
		return cmd, nil
	default:
		return "", fmt.Errorf("unknown factory command: %s", cmd)
	}
}

func handleFactory() {
	cmd, err := parseFactoryCommand(os.Args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		printFactoryUsage()
		os.Exit(1)
	}
	switch cmd {
	case "verify":
		handleFactoryVerify()
	case "gate":
		handleFactoryGate()
	case "factorize":
		handleFactoryFactorize()
	case "digest":
		handleFactoryDigest()
	case "coverage":
		handleFactoryCoverage()
	case "gate-summary":
		handleFactoryGateSummary()
	case "output-contract":
		handleFactoryOutputContract()
	case "doctrine":
		handleFactoryDoctrine()
	case "test-long":
		handleFactoryTestLong()
	}
}

type gateOptions struct {
	TestMode string
}

func parseGateOptions(args []string) (gateOptions, error) {
	fs := flag.NewFlagSet("factory gate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var mode string
	fs.StringVar(&mode, "test-mode", "full", "test mode: full or short")
	if err := fs.Parse(args); err != nil {
		return gateOptions{}, err
	}
	if fs.NArg() != 0 {
		return gateOptions{}, fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	switch mode {
	case "full", "short":
		return gateOptions{TestMode: mode}, nil
	default:
		return gateOptions{}, fmt.Errorf("invalid --test-mode %q: expected full or short", mode)
	}
}

func handleFactoryGate() {
	args := os.Args[3:]
	opts, err := parseGateOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: %v\n", err)
		os.Exit(1)
	}
	if err := validateLongTestBaseline("."); err != nil {
		fmt.Fprintf(os.Stderr, "long-test baseline: %v\n", err)
		os.Exit(1)
	}
	startedAt := time.Now()
	switch opts.TestMode {
	case "short":
		handleShortMode(startedAt)
	default:
		handleFullMode(startedAt)
	}
}

func handleShortMode(startedAt time.Time) {
	// Remove stale aggregates from prior runs
	os.Remove(".factory/gate-summary.json")
	os.Remove(".factory/gate-long-summary.json")

	fastExitCode := gate.RunGateFast(".")
	if err := writeFastSummary(startedAt, time.Now(), fastExitCode); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write fast summary: %v\n", err)
		os.Exit(1)
	}
	// Short mode does NOT publish an aggregate summary - only fast lane results
	os.Exit(fastExitCode)
}

func handleFullMode(startedAt time.Time) {
	// Clear all prior lane artifacts before executing
	os.Remove(".factory/gate-summary.json")
	os.Remove(".factory/gate-fast-summary.json")
	os.Remove(".factory/gate-long-summary.json")

	fmt.Println("=== FAST LANE ===")
	fastExitCode := gate.RunGateFast(".")
	fastFinishedAt := time.Now()
	if err := writeFastSummary(startedAt, fastFinishedAt, fastExitCode); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write fast summary: %v\n", err)
		os.Exit(1)
	}
	if fastExitCode != 0 {
		fmt.Println("\n*** SKIPPING LONG LANE: fast lane failed ***")
		// Write aggregate with fast=fail, long=skip
		if err := writeAggregateAfterFastFailure(); err != nil {
			fmt.Fprintf(os.Stderr, "factory gate: write aggregate summary: %v\n", err)
		}
		os.Exit(1)
	}
	fmt.Println("\n=== LONG LANE ===")
	longExitCode := runTestLongLane()
	if err := writeAggregateForFullMode(); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write aggregate summary: %v\n", err)
		os.Exit(1)
	}
	if fastExitCode != 0 || longExitCode != 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

func runTestLongLane() int {
	if err := ensureBinary(); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: ensure binary: %v\n", err)
		return 1
	}
	// Compute timeout from baseline entries: sum of ci_timeouts + fixed overhead
	timeout := computeLongLaneTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := execution.RunTestLong(ctx, "./bin/leamas")
	if result.Error != nil && result.ExitCode == -1 {
		fmt.Fprintf(os.Stderr, "factory gate: test-long timed out after %v\n", timeout)
		return 1
	}
	return result.ExitCode
}

// computeLongLaneTimeout computes the timeout for the long lane based on baseline entries.
// The timeout is the sum of all registered test timeouts plus overhead.
// No cap is applied - the registered budgets are preserved.
func computeLongLaneTimeout() time.Duration {
	baseline, err := longtest.LoadBaseline(".")
	if err != nil {
		// Fall back to 30 minutes if baseline can't be loaded
		return 30 * time.Minute
	}
	var total time.Duration
	for _, tt := range baseline.Tests {
		d, err := time.ParseDuration(tt.CITimeout)
		if err != nil {
			continue
		}
		total += d
	}
	// Add fixed overhead: 5 minutes per test for startup/teardown
	overhead := 5 * time.Minute * time.Duration(len(baseline.Tests))
	total += overhead
	// Minimum 10 minutes
	const minTimeout = 10 * time.Minute
	if total < minTimeout {
		return minTimeout
	}
	return total
}

func ensureBinary() error {
	if _, err := os.Stat("bin/leamas"); err == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	result := execution.BuildLeamas(ctx, "bin/leamas")
	return result.Error
}

func validateLongTestBaseline(root string) error {
	baseline, err := longtest.LoadBaseline(root)
	if err != nil {
		return fmt.Errorf("load baseline: %w", err)
	}
	return longtest.ValidateBaseline(baseline)
}

func runFactoryGate(root string, summaryPath string, stderr io.Writer, run func(string) int, now func() time.Time) int {
	startedAt := now()
	exitCode := run(root)
	finishedAt := now()
	if err := gate.WriteGateRunSummary(summaryPath, startedAt, finishedAt, exitCode); err != nil {
		fmt.Fprintf(stderr, "write gate summary: %v\n", err)
		return 1
	}
	return exitCode
}

func handleFactoryFactorize() {
	exitCode := gate.RunFactorize(".")
	os.Exit(exitCode)
}

func handleFactoryGateSummary() {
	args := os.Args[3:]
	outputPath := ".factory/gate-summary.json"
	jsonFormat := false
	for i, arg := range args {
		switch arg {
		case "--output":
			if i+1 < len(args) {
				outputPath = args[i+1]
			}
		case "--json":
			jsonFormat = true
		}
	}
	result := output.NewResult("gate-summary")
	result.AddField("output", outputPath)
	if err := gate.WriteGateSummary(".", outputPath); err != nil {
		result.AddFailure("write_error", err.Error())
		handleGateSummaryError(result, jsonFormat)
	}
	result.SetOK()
	if jsonFormat {
		printJSON(result)
	}
	output.WriteLine(os.Stdout, *result)
	os.Exit(0)
}

func handleGateSummaryError(result *output.Result, jsonFormat bool) {
	if jsonFormat {
		printJSON(result)
	}
	output.WriteLine(os.Stderr, *result)
	os.Exit(1)
}

func printJSON(result *output.Result) {
	data, err := result.JSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gate-summary: error generating JSON: %v\n", err)
		os.Exit(2)
	}
	fmt.Fprintln(os.Stdout, string(data))
}

func handleFactoryOutputContract() {
	findings := output.ContractCheck(".")
	verifier := output.DefaultVerifier()
	cmdCount := len(verifier.Commands)
	result := output.NewResult("output-contract")
	result.AddField("commands", cmdCount)
	result.AddField("checked", cmdCount)
	if len(findings) == 0 {
		result.SetOK()
		output.WriteLine(os.Stdout, *result)
		os.Exit(0)
	}
	for _, f := range findings {
		result.AddFailure(f.Kind, f.Message)
	}
	output.WriteLine(os.Stderr, *result)
	os.Exit(1)
}

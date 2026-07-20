// Package main provides the factory subcommand handlers.
package main

import (
	"context"
	"encoding/json"
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
	fastExitCode := gate.RunGateFast(".")
	if err := writeFastSummary(startedAt, time.Now(), fastExitCode); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write fast summary: %v\n", err)
		os.Exit(1)
	}
	if err := writeAggregateSummary(); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write aggregate summary: %v\n", err)
		os.Exit(1)
	}
	os.Exit(fastExitCode)
}

func handleFullMode(startedAt time.Time) {
	fmt.Println("=== FAST LANE ===")
	fastExitCode := gate.RunGateFast(".")
	fastFinishedAt := time.Now()
	if err := writeFastSummary(startedAt, fastFinishedAt, fastExitCode); err != nil {
		fmt.Fprintf(os.Stderr, "factory gate: write fast summary: %v\n", err)
		os.Exit(1)
	}
	if fastExitCode != 0 {
		fmt.Println("\n*** SKIPPING LONG LANE: fast lane failed ***")
		if err := writeAggregateSummary(); err != nil {
			fmt.Fprintf(os.Stderr, "factory gate: write aggregate summary: %v\n", err)
		}
		os.Exit(1)
	}
	fmt.Println("\n=== LONG LANE ===")
	longExitCode := runTestLongLane()
	if err := writeAggregateSummary(); err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result := execution.RunTestLong(ctx, "./bin/leamas")
	if result.Error != nil && result.ExitCode == -1 {
		fmt.Fprintf(os.Stderr, "factory gate: test-long timed out\n")
		return 1
	}
	return result.ExitCode
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

func writeFastSummary(startedAt, finishedAt time.Time, exitCode int) error {
	status := gate.CheckStatusPass
	if exitCode != 0 {
		status = gate.CheckStatusFail
	}
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   finishedAt.Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: string(status),
		Checks: []gate.Check{
			{Name: "fast-lane", Status: status},
		},
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal fast summary: %w", err)
	}
	return os.WriteFile(".factory/gate-fast-summary.json", data, 0644)
}

func writeAggregateSummary() error {
	summary := gate.GateSummary{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Tool:          "leamas factory gate",
		OverallStatus: "pass",
		Checks:        []gate.Check{},
	}
	fastSummary, err := readFastSummary()
	if err == nil && fastSummary != nil {
		summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane", Status: gate.CheckStatusPass})
		if fastSummary.OverallStatus == "fail" {
			summary.OverallStatus = "fail"
			summary.Checks = append(summary.Checks, gate.Check{Name: "fast-lane-status", Status: gate.CheckStatusFail})
		}
	}
	longSummary, err := readLongSummary()
	if err == nil && longSummary != nil {
		summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane", Status: gate.CheckStatusPass})
		if longSummary.Failed > 0 {
			summary.OverallStatus = "fail"
			summary.Checks = append(summary.Checks, gate.Check{Name: "long-lane-status", Status: gate.CheckStatusFail})
		}
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal aggregate summary: %w", err)
	}
	return os.WriteFile(".factory/gate-summary.json", data, 0644)
}

type fastLaneSummary struct {
	SchemaVersion int    `json:"schema_version"`
	OverallStatus string `json:"overall_status"`
	GeneratedAt   string `json:"generated_at"`
}

func readFastSummary() (*fastLaneSummary, error) {
	data, err := os.ReadFile(".factory/gate-fast-summary.json")
	if err != nil {
		return nil, err
	}
	var s fastLaneSummary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func readLongSummary() (*testLongSummary, error) {
	data, err := os.ReadFile(".factory/gate-long-summary.json")
	if err != nil {
		return nil, err
	}
	var s testLongSummary
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

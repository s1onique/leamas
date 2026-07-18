// Package main provides the factory subcommand handlers.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/s1onique/leamas/internal/factory/gate"
	"github.com/s1onique/leamas/internal/factory/output"
)

// parseFactoryCommand extracts and validates the factory subcommand from args.
// Returns the command name or an error if missing/unknown.
func parseFactoryCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", errors.New("missing factory command")
	}

	cmd := args[0]
	switch cmd {
	case "verify", "gate", "factorize", "digest", "coverage", "gate-summary", "output-contract", "doctrine":
		return cmd, nil
	default:
		return "", fmt.Errorf("unknown factory command: %s", cmd)
	}
}

// handleFactory handles the `leamas factory` subcommand.
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
	}
}

func handleFactoryGate() {
	exitCode := runFactoryGate(
		".",
		".factory/gate-summary.json",
		os.Stderr,
		gate.RunGate,
		time.Now,
	)
	os.Exit(exitCode)
}

func runFactoryGate(
	root string,
	summaryPath string,
	stderr io.Writer,
	run func(string) int,
	now func() time.Time,
) int {
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
	args := os.Args[3:] // Skip: leamas factory gate-summary
	outputPath := ".factory/gate-summary.json"
	jsonFormat := false

	// Parse --output and --json flags
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
		if jsonFormat {
			data, jsonErr := result.JSON()
			if jsonErr != nil {
				fmt.Fprintf(os.Stderr, "gate-summary: error generating JSON: %v\n", jsonErr)
				os.Exit(2)
			}
			fmt.Fprintln(os.Stdout, string(data))
			os.Exit(1)
		}
		output.WriteLine(os.Stderr, *result)
		os.Exit(1)
	}

	result.SetOK()

	if jsonFormat {
		data, err := result.JSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gate-summary: error generating JSON: %v\n", err)
			os.Exit(2)
		}
		fmt.Fprintln(os.Stdout, string(data))
		os.Exit(0)
	}

	output.WriteLine(os.Stdout, *result)
	os.Exit(0)
}

func handleFactoryOutputContract() {
	findings := output.ContractCheck(".")
	verifier := output.DefaultVerifier()
	cmdCount := len(verifier.Commands)

	if len(findings) == 0 {
		// Use output package for consistent formatting
		result := output.NewResult("output-contract")
		result.SetOK()
		result.AddField("commands", cmdCount)
		result.AddField("checked", cmdCount)
		output.WriteLine(os.Stdout, *result)
		os.Exit(0)
	}

	// Report failures
	result := output.NewResult("output-contract")
	result.AddField("commands", cmdCount)
	result.AddField("checked", cmdCount)
	for _, f := range findings {
		result.AddFailure(f.Kind, f.Message)
	}
	output.WriteLine(os.Stderr, *result)
	os.Exit(1)
}

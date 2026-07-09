// Package main provides the factory subcommand handlers.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// parseFactoryCommand extracts and validates the factory subcommand from args.
// Returns the command name or an error if missing/unknown.
func parseFactoryCommand(args []string) (string, error) {
	if len(args) < 1 {
		return "", errors.New("missing factory command")
	}

	cmd := args[0]
	switch cmd {
	case "verify", "gate", "factorize", "digest", "coverage":
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
	}
}

func handleFactoryGate() {
	exitCode := gate.RunGate(".")
	os.Exit(exitCode)
}

func handleFactoryFactorize() {
	exitCode := gate.RunFactorize(".")
	os.Exit(exitCode)
}

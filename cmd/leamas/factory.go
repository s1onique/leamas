// Package main provides the factory subcommand handlers.
package main

import (
	"fmt"
	"os"

	"github.com/s1onique/leamas/internal/factory/gate"
)

// handleFactory handles the `leamas factory` subcommand.
func handleFactory() {
	if len(os.Args) < 3 {
		printFactoryUsage()
		os.Exit(1)
	}

	switch os.Args[2] {
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
	default:
		fmt.Fprintf(os.Stderr, "unknown factory command: %s\n", os.Args[2])
		printFactoryUsage()
		os.Exit(1)
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

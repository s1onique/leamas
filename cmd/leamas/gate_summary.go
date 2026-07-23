package main

import (
	"fmt"
	"io"
	"os"
)

// gateSummaryUsageText is the surface-level help text for the
// `leamas gate-summary` command. It is exposed as a function so the
// text can be locked by golden tests and reused by the schema
// subcommand help.
func gateSummaryUsageText() string {
	return `Usage: leamas gate-summary <subcommand> [args]

Subcommands:
  schema                 JSON Schema introspection
  --help, -h             Show this help

The gate-summary command exposes the Gate Summary wire format that
Leamas decodes from .factory/gate-summary.json.

The emitted JSON Schema describes the Gate Summary wire format.
Decode, normalization, lifecycle, diagnostic, and digest semantics remain
defined by the executable Leamas Gate Summary contract.

Examples:
  leamas gate-summary schema list
  leamas gate-summary schema show v1
  leamas gate-summary schema show v2 > gate-summary-v2.schema.json
`
}

// runGateSummary dispatches the `leamas gate-summary` subcommand
// namespace. It returns the integer exit code the caller should
// propagate to the process.
func runGateSummary(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, gateSummaryUsageText())
		return 1
	}
	switch args[0] {
	case "--help", "-h", "help":
		fmt.Fprint(os.Stdout, gateSummaryUsageText())
		return 0
	case "schema":
		return runGateSummarySchema(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "gate-summary: unknown subcommand %q\n", args[0])
		fmt.Fprint(os.Stderr, gateSummaryUsageText())
		return 1
	}
}

// printGateSummaryUsage prints the surface help to stderr. Tests use
// the printGateSummaryUsageTo form so they can capture the output.
func printGateSummaryUsage() {
	printGateSummaryUsageTo(os.Stderr)
}

// printGateSummaryUsageTo writes the surface help to the given writer.
func printGateSummaryUsageTo(w io.Writer) {
	fmt.Fprint(w, gateSummaryUsageText())
}

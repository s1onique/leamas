package main

import (
	"io"
)

// runFactoryClosePlan routes to the new plan subcommands.
// It is the single dispatcher for factory close plan.
func runFactoryClosePlan(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return closeFailureCode("usage", "expected schema, example, or validate")
	}
	switch args[0] {
	case "schema":
		return runFactoryClosePlanSchema(args[1:], stdout, stderr)
	case "example":
		return runFactoryClosePlanExample(args[1:], stdout, stderr)
	case "validate":
		return runFactoryClosePlanValidate(args[1:], stdout, stderr)
	default:
		return closeFailureCode("usage", "unknown plan subcommand: "+args[0])
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// runFactoryClosePlanExample outputs a valid example plan.
// Before emitting, it validates the example through the composed pipeline
// to guarantee it passes all structural, decode, and semantic checks.
func runFactoryClosePlanExample(args []string, stdout, stderr io.Writer) int {
	// Help must be the sole argument
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan example")
		fmt.Fprintln(stderr, "Output a valid Closure Protocol v1 plan example as JSON.")
		return 0
	}
	if len(args) > 0 {
		return closeUsageError(stderr, "factory close plan example", "accepts no arguments")
	}

	// Generate example plan
	plan := closure.DescriptorExample()
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan example", "marshal failed: "+err.Error())
	}

	// Fail-closed validation: verify example passes composed validation
	result := closure.ValidatePlanComposed(data)
	if !result.Valid {
		return closeUsageError(stderr, "factory close plan example", "example validation failed: example is not a valid plan")
	}

	// Atomic write: encode to buffer first, then single write
	if err := atomicWrite(stdout, data); err != nil {
		return closeUsageError(stderr, "factory close plan example", "output failed")
	}
	return 0
}

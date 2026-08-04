package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// runFactoryClosePlanSchema outputs the Closure Protocol v1 JSON Schema.
func runFactoryClosePlanSchema(args []string, stdout, stderr io.Writer) int {
	// Help must be the sole argument
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan schema")
		fmt.Fprintln(stderr, "Output the Closure Protocol v1 JSON Schema as JSON.")
		return 0
	}
	if len(args) > 0 {
		return closeUsageError(stderr, "factory close plan schema", "accepts no arguments")
	}

	schema, err := closure.JSONSchema()
	if err != nil {
		return closeUsageError(stderr, "factory close plan schema", "schema generation failed")
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan schema", "marshal failed: "+err.Error())
	}

	// Atomic write: encode to buffer first, then single write
	if err := atomicWrite(stdout, data); err != nil {
		return closeUsageError(stderr, "factory close plan schema", "output failed")
	}
	return 0
}

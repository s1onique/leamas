package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// runFactoryClosePlanSchema outputs the Closure Protocol v1 JSON Schema.
func runFactoryClosePlanSchema(args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan schema")
		fmt.Fprintln(stderr, "Output the Closure Protocol v1 JSON Schema as JSON.")
		return 0
	}
	if len(args) > 0 {
		return closeUsageError(stderr, "factory close plan schema", "accepts no arguments")
	}

	schema := closure.JSONSchema()
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan schema", "marshal failed: "+err.Error())
	}

	// Atomic write: encode to buffer first, then single write
	buf := make([]byte, 0, len(data)+1)
	buf = append(buf, data...)
	buf = append(buf, '\n')

	if _, err := stdout.Write(buf); err != nil {
		return 2
	}
	return 0
}

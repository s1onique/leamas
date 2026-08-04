package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// runFactoryClosePlanExample outputs the canonical Closure Protocol v1 plan example.
func runFactoryClosePlanExample(args []string, stdout, stderr io.Writer) int {
	if hasHelpFlag(args) {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan example")
		fmt.Fprintln(stderr, "Output the canonical Closure Protocol v1 plan example as JSON.")
		return 0
	}
	if len(args) > 0 {
		return closeUsageError(stderr, "factory close plan example", "accepts no arguments")
	}

	example := closure.DescriptorExample()
	// Validate the example passes all stages
	raw, err := json.Marshal(example)
	if err != nil {
		return closeUsageError(stderr, "factory close plan example", "marshal failed: "+err.Error())
	}
	result := closure.ValidatePlanComposed(raw)
	if !result.Valid {
		return closeUsageError(stderr, "factory close plan example", "composed validation failed")
	}
	if !result.SemanticValid {
		return closeUsageError(stderr, "factory close plan example", "semantic validation failed")
	}

	data, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan example", "marshal failed: "+err.Error())
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

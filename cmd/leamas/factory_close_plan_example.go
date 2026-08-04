package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryClosePlanExample(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan example")
		fmt.Fprintln(stderr, "Output the canonical Closure Protocol v1 plan example as JSON.")
		return closeSuccessCode()
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
	if _, err := stdout.Write(data); err != nil {
		return closeUsageError(stderr, "factory close plan example", "write failed")
	}
	fmt.Fprintln(stdout)
	return closeSuccessCode()
}

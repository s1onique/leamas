package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/s1onique/leamas/internal/factory/closure"
)

func runFactoryClosePlanSchema(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan schema")
		fmt.Fprintln(stderr, "Output the Closure Protocol v1 plan schema as JSON.")
		return closeSuccessCode()
	}
	if len(args) > 0 {
		return closeUsageError(stderr, "factory close plan schema", "accepts no arguments")
	}

	schema := closure.PlanSchema()
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan schema", "marshal failed: "+err.Error())
	}
	if _, err := stdout.Write(data); err != nil {
		return closeUsageError(stderr, "factory close plan schema", "write failed")
	}
	fmt.Fprintln(stdout)
	return closeSuccessCode()
}

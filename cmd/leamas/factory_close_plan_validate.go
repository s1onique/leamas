package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/closure"
)

const maxPlanBytes = 1024 * 1024 // 1MB max

func runFactoryClosePlanValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan validate [--file <path>] [--stdin]")
		fmt.Fprintln(stderr, "Validate a Closure Protocol v1 plan JSON file.")
		return closeSuccessCode()
	}

	var file string
	var useStdin bool
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return closeUsageError(stderr, "factory close plan validate", "--file requires a value")
			}
			file = args[i+1]
			i++
		case "--stdin":
			useStdin = true
		default:
			return closeUsageError(stderr, "factory close plan validate", "unknown flag: "+args[i])
		}
	}

	// Check input selection
	if file == "" && !useStdin {
		return closeUsageError(stderr, "factory close plan validate", "one of --file or --stdin is required")
	}
	if file != "" && useStdin {
		return closeUsageError(stderr, "factory close plan validate", "--file and --stdin are mutually exclusive")
	}

	// Read input
	var data []byte
	var err error
	if file != "" {
		data, err = os.ReadFile(file)
		if err != nil {
			return closeUsageError(stderr, "factory close plan validate", "file read failed: "+err.Error())
		}
	} else {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return closeUsageError(stderr, "factory close plan validate", "stdin read failed: "+err.Error())
		}
	}

	// Check size
	if len(data) > maxPlanBytes {
		return closeUsageError(stderr, "factory close plan validate", "plan exceeds max size")
	}

	// Validate using composed pipeline
	result := closure.ValidatePlanComposed(data)

	// Marshal result to JSON
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan validate", "result marshal failed")
	}

	if _, err := stdout.Write(resultData); err != nil {
		return closeUsageError(stderr, "factory close plan validate", "write failed")
	}
	fmt.Fprintln(stdout)

	if result.Valid {
		return closeSuccessCode()
	}
	return 1 // Invalid plan, exit 1
}

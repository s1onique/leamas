package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/s1onique/leamas/internal/factory/closure"
)

// planValidateDeps encapsulates dependencies for testing.
type planValidateDeps struct {
	readFile func(path string) ([]byte, error)
	readAll  func(r io.Reader) ([]byte, error)
}

// runFactoryClosePlanValidateWith validates a plan using the provided readers.
// Production binds os.Stdin and real file opening at the outer adapter.
func runFactoryClosePlanValidateWith(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	deps planValidateDeps,
) int {
	if hasHelpFlag(args) {
		fmt.Fprintln(stderr, "Usage: leamas factory close plan validate [--file <path>] [--stdin]")
		fmt.Fprintln(stderr, "Validate a Closure Protocol v1 plan JSON file.")
		return 0
	}

	// Parse flags with consistent argument parser
	var filePath string
	var useStdin bool
	seenFile := false
	seenStdin := false

	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--file":
			if seenFile {
				return closeUsageError(stderr, "factory close plan validate", "repeated --file")
			}
			if i+1 >= len(args) {
				return closeUsageError(stderr, "factory close plan validate", "--file requires a value")
			}
			if args[i+1] == "--file" || args[i+1] == "--stdin" {
				return closeUsageError(stderr, "factory close plan validate", "--file requires a value")
			}
			filePath = args[i+1]
			seenFile = true
			i += 2
		case "--stdin":
			if seenStdin {
				return closeUsageError(stderr, "factory close plan validate", "repeated --stdin")
			}
			useStdin = true
			seenStdin = true
			i++
		default:
			return closeUsageError(stderr, "factory close plan validate", "unknown flag: "+arg)
		}
	}

	// Check mutual exclusivity
	if filePath != "" && useStdin {
		return closeUsageError(stderr, "factory close plan validate", "--file and --stdin are mutually exclusive")
	}

	// Require input selection
	if filePath == "" && !useStdin {
		return closeUsageError(stderr, "factory close plan validate", "one of --file or --stdin is required")
	}

	// Read input with bounded read
	var data []byte
	var err error

	if filePath != "" {
		data, err = deps.readFile(filePath)
		if err != nil {
			return closeUsageError(stderr, "factory close plan validate", "file read failed: "+err.Error())
		}
	} else {
		data, err = deps.readAll(stdin)
		if err != nil {
			return closeUsageError(stderr, "factory close plan validate", "stdin read failed: "+err.Error())
		}
	}

	// Check size: MaxPlanBytes + 1 = rejection before validation
	if len(data) > closure.MaxPlanBytes {
		return closeUsageError(stderr, "factory close plan validate", "plan exceeds max size")
	}

	// Validate using composed pipeline
	result := closure.ValidatePlanComposed(data)

	// Marshal result to JSON with nil arrays serialized as []
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return closeUsageError(stderr, "factory close plan validate", "result marshal failed")
	}

	// Atomic write: encode to buffer first, then single write
	buf := make([]byte, 0, len(resultData)+1)
	buf = append(buf, resultData...)
	buf = append(buf, '\n')

	if _, err := stdout.Write(buf); err != nil {
		return 2
	}

	if result.Valid {
		return 0
	}
	return 1 // Invalid plan
}

// runFactoryClosePlanValidate is the production adapter using os.Stdin and os.ReadFile.
func runFactoryClosePlanValidate(args []string, stdout, stderr io.Writer) int {
	deps := planValidateDeps{
		readFile: os.ReadFile,
		readAll: func(r io.Reader) ([]byte, error) {
			return io.ReadAll(io.LimitReader(r, closure.MaxPlanBytes+1))
		},
	}
	return runFactoryClosePlanValidateWith(args, os.Stdin, stdout, stderr, deps)
}

// hasHelpFlag checks if args contain a help flag.
func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

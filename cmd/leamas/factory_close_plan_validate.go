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
	openFile    func(path string) (io.ReadCloser, error)
	readBounded func(r io.Reader, max int64) ([]byte, error)
}

// runFactoryClosePlanValidateWith validates a plan using the provided readers.
// Production binds os.Open and bounded read at the outer adapter.
func runFactoryClosePlanValidateWith(
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	deps planValidateDeps,
) int {
	// Help flag check
	if len(args) == 1 && isHelpFlag(args[0]) {
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
			if args[i+1] == "--file" || args[i+1] == "--stdin" || isHelpFlag(args[i+1]) {
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

	// Read input with bounded read through authority
	var data []byte
	var err error
	const maxBytes = closure.MaxPlanBytes + 1

	if filePath != "" {
		f, err := deps.openFile(filePath)
		if err != nil {
			return closeUsageError(stderr, "factory close plan validate", "file read failed: "+err.Error())
		}
		data, err = deps.readBounded(f, maxBytes)
		closeErr := f.Close()
		if err != nil {
			if closeErr != nil {
				return closeUsageError(stderr, "factory close plan validate", "file read failed: "+err.Error()+"; close failed: "+closeErr.Error())
			}
			return closeUsageError(stderr, "factory close plan validate", "file read failed: "+err.Error())
		}
		if closeErr != nil {
			return closeUsageError(stderr, "factory close plan validate", "file close failed: "+closeErr.Error())
		}
	} else {
		data, err = deps.readBounded(stdin, maxBytes)
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
	if err := atomicWrite(stdout, resultData); err != nil {
		return closeUsageError(stderr, "factory close plan validate", "output failed")
	}

	if result.Valid {
		return 0
	}
	return 1 // Invalid plan
}

// atomicWrite performs exactly one write and checks the result.
// On failure, exit is 2 with stderr containing a bounded diagnostic.
func atomicWrite(w io.Writer, payload []byte) error {
	n, err := w.Write(payload)
	if err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if n != len(payload) {
		return fmt.Errorf("partial write: wrote %d of %d bytes", n, len(payload))
	}
	return nil
}

// runFactoryClosePlanValidate is the production adapter using os.Open and bounded read.
func runFactoryClosePlanValidate(args []string, stdout, stderr io.Writer) int {
	deps := planValidateDeps{
		openFile: func(path string) (io.ReadCloser, error) {
			return os.Open(path)
		},
		readBounded: func(r io.Reader, max int64) ([]byte, error) {
			return io.ReadAll(io.LimitReader(r, max))
		},
	}
	return runFactoryClosePlanValidateWith(args, os.Stdin, stdout, stderr, deps)
}

// isHelpFlag returns true if arg is a help flag.
func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

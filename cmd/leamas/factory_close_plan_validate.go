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

	// Convert through frozen public CLI DTO before serialisation.
	// The DTO is the authoritative public wire boundary; adding
	// a JSON-tagged field to the internal type does not change
	// the CLI protocol.
	dto := toPlanValidationDTO(result)

	// Marshal DTO to JSON with nil arrays serialized as []
	resultData, err := json.MarshalIndent(dto, "", "  ")
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
		readBounded: productionBoundedRead,
	}
	return runFactoryClosePlanValidateWith(args, os.Stdin, stdout, stderr, deps)
}

// isHelpFlag returns true if arg is a help flag.
func isHelpFlag(arg string) bool {
	return arg == "-h" || arg == "--help"
}

// planValidationDTO is the frozen public CLI wire format for
// validation results. The set of fields and JSON keys is fixed
// so that adding an internal JSON-tagged field cannot silently
// change the CLI protocol.
//
// Diagnostic arrays are always non-nil (rendered as []) so that
// callers can rely on JSON-array, never JSON-null semantics.
type planValidationDTO struct {
	Structural     planStructuralDTO   `json:"structural"`
	Decoded        bool                `json:"decoded"`
	DecodeErrors   []planDiagnosticDTO `json:"decode_errors"`
	SemanticValid  bool                `json:"semantic_valid"`
	SemanticErrors []planDiagnosticDTO `json:"semantic_errors"`
	Valid          bool                `json:"valid"`
}

// planStructuralDTO is the structural-stage subset of the public
// validation wire.
type planStructuralDTO struct {
	Valid           bool                `json:"valid"`
	ContractVersion int                 `json:"contract_version"`
	Errors          []planDiagnosticDTO `json:"errors"`
}

// planDiagnosticDTO is a single frozen diagnostic line. It exposes
// no internal cause, observer, or implementation fields.
type planDiagnosticDTO struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// toPlanValidationDTO converts the internal composed result to
// the frozen public CLI DTO. nil diagnostic arrays become empty
// slices so JSON serialisation always yields [].
func toPlanValidationDTO(r closure.ComposedPlanValidationResult) planValidationDTO {
	decodeErrors := r.DecodeErrors
	if decodeErrors == nil {
		decodeErrors = []closure.PlanValidationError{}
	}
	semanticErrors := r.SemanticErrors
	if semanticErrors == nil {
		semanticErrors = []closure.PlanValidationError{}
	}
	structErrors := r.Structural.Errors
	if structErrors == nil {
		structErrors = []closure.PlanValidationError{}
	}
	return planValidationDTO{
		Structural: planStructuralDTO{
			Valid:           r.Structural.Valid,
			ContractVersion: r.Structural.ContractVersion,
			Errors:          toPlanDiagnosticDTOs(structErrors),
		},
		Decoded:        r.Decoded,
		DecodeErrors:   toPlanDiagnosticDTOs(decodeErrors),
		SemanticValid:  r.SemanticValid,
		SemanticErrors: toPlanDiagnosticDTOs(semanticErrors),
		Valid:          r.Valid,
	}
}

func toPlanDiagnosticDTOs(src []closure.PlanValidationError) []planDiagnosticDTO {
	out := make([]planDiagnosticDTO, len(src))
	for i, e := range src {
		out[i] = planDiagnosticDTO{Path: e.InstancePath, Message: e.Message}
	}
	return out
}

// productionBoundedRead reads up to max bytes from r using a
// bounded LimitReader. It is the production implementation of the
// bounded-read capability and is exported as a named function so
// tests can prove the real bound is honoured with counting or
// infinite readers.
func productionBoundedRead(r io.Reader, max int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, max))
}

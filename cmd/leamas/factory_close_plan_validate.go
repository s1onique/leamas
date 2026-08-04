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

// planDiagnosticDTO is the frozen public diagnostic wire.
// Every field name is part of the CLI contract; renaming or
// removing any field is a breaking change. The diagnostic
// preserves the typed taxonomy established by Closure Protocol
// v1 so callers can identify the failing rule without parsing
// prose:
//
//   - InstancePath is the JSON pointer to the failing value
//     (empty for runtime-only diagnostics).
//   - SchemaPath is the JSON pointer to the failing schema
//     location.
//   - Code is the stable diagnostic code (e.g. "required",
//     "enum_mismatch", "value_too_large").
//   - Keyword is the JSON Schema keyword that failed
//     (e.g. "required", "enum", "maxLength").
//   - Message is the human-readable explanation.
//   - RejectedValue is the value that failed, deep-copied.
//   - AcceptedValues is the closed set of accepted values, [] not null.
//   - PropertyName is the runtime-only field name (e.g.
//     "vcs.revision", "binary_sha256") for runner-authority
//     diagnostics that have no InstancePath.
//
// Cause, observer, and implementation-only fields are
// intentionally absent.
type planDiagnosticDTO struct {
	InstancePath   string `json:"instance_path"`
	SchemaPath     string `json:"schema_path"`
	Code           string `json:"code"`
	Keyword        string `json:"keyword"`
	Message        string `json:"message"`
	RejectedValue  any    `json:"rejected_value,omitempty"`
	AcceptedValues []any  `json:"accepted_values"`
	PropertyName   string `json:"property_name,omitempty"`
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
		out[i] = planDiagnosticDTO{
			InstancePath:   e.InstancePath,
			SchemaPath:     e.SchemaPath,
			Code:           string(e.Code),
			Keyword:        string(e.Keyword),
			Message:        e.Message,
			RejectedValue:  deepCopyAny(e.RejectedValue),
			AcceptedValues: toAnySlice(e.AcceptedValues),
			PropertyName:   e.PropertyName,
		}
	}
	return out
}

// toAnySlice converts a []string to []any without nil entries.
// nil becomes []any{} so JSON serialisation is [] not null.
func toAnySlice(src []string) []any {
	if src == nil {
		return []any{}
	}
	out := make([]any, len(src))
	for i, s := range src {
		out[i] = s
	}
	return out
}

// deepCopyAny returns a deep copy of simple JSON values so
// callers cannot mutate the source through the DTO. Maps and
// slices are recursively copied; primitives are returned as-is;
// nil stays nil.
func deepCopyAny(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			out[k] = deepCopyAny(vv)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			out[i] = deepCopyAny(vv)
		}
		return out
	case []string:
		out := make([]any, len(x))
		for i, vv := range x {
			out[i] = vv
		}
		return out
	default:
		return x
	}
}

// productionBoundedRead reads up to max bytes from r using a
// bounded LimitReader. It is the production implementation of the
// bounded-read capability and is exported as a named function so
// tests can prove the real bound is honoured with counting or
// infinite readers.
func productionBoundedRead(r io.Reader, max int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, max))
}

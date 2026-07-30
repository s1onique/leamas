package closure

import (
	"encoding/json"
	"strings"
)

// PlanValidationCode is the stable, machine-readable diagnostic code
// emitted by the structural validator. The closed set is documented
// in ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 and is the
// single source of truth consumers (CLI subprocess tests, schema
// generators, example generators, future JSON diagnostics) read.
type PlanValidationCode string

const (
	// PlanCodeInvalidJSON signals that the JSON document could not
	// be parsed at all. The instance_path is empty because no
	// structural location is recoverable.
	PlanCodeInvalidJSON PlanValidationCode = "invalid_json"

	// PlanCodeUnsupportedContractVersion signals that the
	// contract_version field is missing, not an integer, or has a
	// value other than the single supported ContractVersionV1.
	PlanCodeUnsupportedContractVersion PlanValidationCode = "unsupported_contract_version"

	// PlanCodeRequiredPropertyMissing signals that a required
	// property is absent from its parent object. The instance_path
	// points at the missing sibling name.
	PlanCodeRequiredPropertyMissing PlanValidationCode = "required_property_missing"

	// PlanCodeInvalidType signals that the JSON type does not
	// match the descriptor (for example: a string in a field
	// declared integer).
	PlanCodeInvalidType PlanValidationCode = "invalid_type"

	// PlanCodeInvalidEnum signals that a string value is not in
	// the descriptor's enum authority.
	PlanCodeInvalidEnum PlanValidationCode = "invalid_enum"

	// PlanCodeUnknownProperty signals that a JSON name does not
	// appear in the descriptor for its parent object.
	PlanCodeUnknownProperty PlanValidationCode = "unknown_property"

	// PlanCodeSemanticConstraintFailed signals that a field passed
	// structural validation but failed a downstream semantic rule
	// (for example: a 41-character OID, or an absent argv on a
	// runnable check). The structural validator does not run
	// semantic rules itself; downstream validators attach this
	// code to their diagnostics so consumers see a single,
	// uniform taxonomy.
	PlanCodeSemanticConstraintFailed PlanValidationCode = "semantic_constraint_failed"
)

// PlanValidationKeyword mirrors the JSON Schema keyword taxonomy
// one-to-one. The closed set is intentionally narrow because the
// descriptor pins every constraint: structural validation does not
// invent new keywords that the schema cannot also produce.
type PlanValidationKeyword string

const (
	KeywordType           PlanValidationKeyword = "type"
	KeywordEnum           PlanValidationKeyword = "enum"
	KeywordRequired       PlanValidationKeyword = "required"
	KeywordConst          PlanValidationKeyword = "const"
	KeywordAdditionalProp PlanValidationKeyword = "additionalProperties"
	KeywordMinItems       PlanValidationKeyword = "minItems"
)

// PlanValidationError is a single structured diagnostic. The struct
// is intentionally small and JSON-marshallable so future CLI flags
// can render it verbatim without translating strings.
type PlanValidationError struct {
	// InstancePath is the canonical JSON Pointer (RFC 6901) at which
	// the diagnostic was raised. The empty string means "the root
	// object".
	InstancePath string `json:"instance_path"`

	// SchemaPath is the canonical JSON Pointer into the descriptor
	// that authorised the diagnostic. The descriptor is treated as
	// the canonical schema source; consumers can map SchemaPath
	// back to a JSON Schema fragment if they wish.
	SchemaPath string `json:"schema_path"`

	// Code is the stable diagnostic code (PlanCode*).
	Code PlanValidationCode `json:"code"`

	// Keyword is the JSON Schema keyword the diagnostic maps to.
	Keyword PlanValidationKeyword `json:"keyword"`

	// Message is the human-readable diagnostic. It is stable enough
	// for snapshot tests but consumers should prefer Code.
	Message string `json:"message"`

	// RejectedValue, when non-nil, is the literal JSON value the
	// validator rejected. It is omitted for diagnostics whose
	// rejected value is structurally unprintable (for example: a
	// duplicate-key collision). The field is JSON-marshallable as
	// "rejected_value" via the tag below.
	RejectedValue any `json:"rejected_value,omitempty"`

	// AcceptedValues, when non-empty, names the closed set of
	// values the descriptor accepts. Currently used by
	// invalid_enum and (in the future) by const-typed diagnostics.
	AcceptedValues []string `json:"accepted_values,omitempty"`

	// PropertyName, when non-empty, names the property that the
	// diagnostic refers to. RequiredPropertyMissing uses it so
	// consumers that cannot parse instance_path still see the
	// missing field by name.
	PropertyName string `json:"property_name,omitempty"`
}

// PlanValidationResult is the structured outcome of a single
// ValidatePlanStructural call. The struct is JSON-marshallable and
// stable enough for future CLI flags to emit it directly.
type PlanValidationResult struct {
	// Valid reports whether the document passed every diagnostic
	// check. A document with Valid==true has zero Errors.
	Valid bool `json:"valid"`

	// ContractVersion reports the contract version that the
	// validator was able to recover from the document. When the
	// JSON is too malformed to parse or contract_version is absent
	// or wrong-typed, the validator returns the zero value (0) and
	// documents the recovery failure through an
	// unsupported_contract_version or invalid_json diagnostic.
	// Consumers MUST treat 0 as "version unrecoverable".
	ContractVersion int `json:"contract_version"`

	// Errors is the deterministic, sorted diagnostic stream. The
	// sort key is (Code, InstancePath, PropertyName, Message) so
	// parity between successive runs is bit-identical.
	Errors []PlanValidationError `json:"errors"`
}

// ValidatePlanStructural walks the supplied JSON bytes through the
// descriptor's structural rules BEFORE the strict Go decoder runs.
// The function returns a populated PlanValidationResult; the
// underlying error is nil so callers can decide how to render the
// diagnostics. Callers that want the legacy error-only API should
// use ValidatePlan instead.
//
// The validator never cascades a structural failure into a semantic
// diagnostic. If the document fails structurally (for example:
// "missing required /contract_version") the validator does NOT
// also report "unsupported execution mode"; the structural stream
// runs to completion, semantic rules run only on documents that
// pass structurally.
func ValidatePlanStructural(data []byte) PlanValidationResult {
	result := PlanValidationResult{Valid: true}
	if len(data) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, PlanValidationError{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "document is empty; expected JSON object",
		})
		return result
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, PlanValidationError{
			InstancePath: "",
			SchemaPath:   "",
			Code:         PlanCodeInvalidJSON,
			Keyword:      KeywordType,
			Message:      "could not decode JSON: " + err.Error(),
		})
		return result
	}
	contract := planContractV1()
	diagnostics := validatePlanObject(contract.Root, root, contract, "")
	result.ContractVersion = recoverContractVersion(root, contract.ContractVersion)
	sortDiagnostics(diagnostics)
	result.Errors = diagnostics
	if len(diagnostics) > 0 {
		result.Valid = false
	}
	return result
}

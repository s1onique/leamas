package closure

import (
	"encoding/json"
	"sort"
	"strings"
)

// PlanValidationCode is the stable, machine-readable diagnostic code
// emitted by the structural validator. The closed set is documented
// in ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01 and
// ACT-LEAMAS-FACTORY-CLOSE-PLAN-CONTRACT-AUTHORITY01-CORRECTION01-
// STRUCTURAL-PARITY-CLOSURE01.
type PlanValidationCode string

const (
	PlanCodeInvalidJSON                PlanValidationCode = "invalid_json"
	PlanCodeUnsupportedContractVersion PlanValidationCode = "unsupported_contract_version"
	PlanCodeRequiredPropertyMissing    PlanValidationCode = "required_property_missing"
	PlanCodeInvalidType                PlanValidationCode = "invalid_type"
	PlanCodeInvalidEnum                PlanValidationCode = "invalid_enum"
	PlanCodeUnknownProperty            PlanValidationCode = "unknown_property"
	PlanCodeSemanticConstraintFailed   PlanValidationCode = "semantic_constraint_failed"
)

// PlanValidationKeyword mirrors the JSON Schema keyword taxonomy.
type PlanValidationKeyword string

const (
	KeywordType           PlanValidationKeyword = "type"
	KeywordEnum           PlanValidationKeyword = "enum"
	KeywordRequired       PlanValidationKeyword = "required"
	KeywordConst          PlanValidationKeyword = "const"
	KeywordAdditionalProp PlanValidationKeyword = "additionalProperties"
	KeywordMinItems       PlanValidationKeyword = "minItems"
	KeywordIfThenElse     PlanValidationKeyword = "if"
)

// PlanValidationError is a single structured diagnostic. The struct
// is JSON-marshallable so future CLI flags can render it verbatim.
type PlanValidationError struct {
	InstancePath   string                `json:"instance_path"`
	SchemaPath     string                `json:"schema_path"`
	Code           PlanValidationCode    `json:"code"`
	Keyword        PlanValidationKeyword `json:"keyword"`
	Message        string                `json:"message"`
	RejectedValue  any                   `json:"rejected_value,omitempty"`
	AcceptedValues []string              `json:"accepted_values,omitempty"`
	PropertyName   string                `json:"property_name,omitempty"`
}

// PlanValidationResult is the structured outcome of a single
// validation pass.
type PlanValidationResult struct {
	Valid           bool                  `json:"valid"`
	ContractVersion int                   `json:"contract_version"`
	Errors          []PlanValidationError `json:"errors"`
}

// ValidatePlanStructural is the canonical structural validator
// entry point. It performs single-document parsing (Phase 1) and
// walks the parsed value against the v1 descriptor, producing a
// deterministic, sorted diagnostic stream.
//
// The function does NOT cascade structural failures into semantic
// failures: when the document fails structurally, semantic rules do
// not run. This matches the directive ACT's "structural failures
// must not cascade into semantic failures" requirement.
func ValidatePlanStructural(data []byte) PlanValidationResult {
	result := PlanValidationResult{Valid: true}
	root, diagnostics := parseClosurePlanDocument(data)
	if len(diagnostics) > 0 {
		result.Valid = false
		result.Errors = diagnostics
		return result
	}
	contract := planContractV1()
	moreDiagnostics := validatePlanObject(contract.Root, root, contract, "")
	result.ContractVersion = recoverContractVersion(root)
	result.Errors = append(result.Errors, moreDiagnostics...)
	sortDiagnostics(result.Errors)
	if len(result.Errors) > 0 {
		result.Valid = false
	}
	return result
}

// recoverContractVersion extracts the contract version from the
// already-parsed root. Returns 0 when the field is missing or
// wrong-typed; callers MUST NOT treat 0 as a successful v1
// detection.
func recoverContractVersion(root any) int {
	if root == nil {
		return 0
	}
	asMap, ok := root.(map[string]any)
	if !ok {
		return 0
	}
	v, present := asMap["contract_version"]
	if !present {
		return 0
	}
	if i, ok := jsonNumberToInteger(v); ok {
		return i
	}
	return 0
}

// validatePlanObject walks a JSON value against the descriptor for
// its parent object. The descriptor's Kind selects the wire-level
// shape: closed (rejects unknown keys) or string_map (accepts any
// key but requires every value to be a JSON string).
func validatePlanObject(object planObjectDescriptor, raw any, contract planContractV1Descriptor, instancePath string) []PlanValidationError {
	if instancePath == "" {
		instancePath = object.Path
	}
	var diagnostics []PlanValidationError
	if raw == nil {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: instancePath,
			SchemaPath:   object.Path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      "value must be a JSON object",
		})
		return diagnostics
	}
	asMap, ok := raw.(map[string]any)
	if !ok {
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath: instancePath,
			SchemaPath:   object.Path,
			Code:         PlanCodeInvalidType,
			Keyword:      KeywordType,
			Message:      "value must be a JSON object, got " + typeNameOf(raw),
		})
		return diagnostics
	}
	// Required-property diagnostics first.
	for _, required := range object.Required {
		if _, present := asMap[required]; !present {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath: canonicalJSONPointer(instancePath, required),
				SchemaPath:   canonicalJSONPointer(object.Path, required),
				Code:         PlanCodeRequiredPropertyMissing,
				Keyword:      KeywordRequired,
				Message:      "missing required property \"" + required + "\"",
				PropertyName: required,
			})
		}
	}
	// Iterate fields in lexicographic order.
	if object.Kind == objectStringMap {
		diagnostics = append(diagnostics, validateStringMap(object, asMap, instancePath)...)
	} else {
		for _, name := range object.fieldNamesSorted() {
			field := object.Fields[name]
			value, present := asMap[name]
			if !present {
				continue
			}
			fieldPath := canonicalJSONPointer(instancePath, name)
			diagnostics = append(diagnostics, validatePlanField(field, fieldPath, value, object, asMap, contract)...)
		}
		// Unknown-property diagnostics last.
		known := make(map[string]struct{}, len(object.Fields))
		for name := range object.Fields {
			known[name] = struct{}{}
		}
		unknownKeys := make([]string, 0)
		for key := range asMap {
			if _, ok := known[key]; !ok {
				unknownKeys = append(unknownKeys, key)
			}
		}
		sort.Strings(unknownKeys)
		for _, key := range unknownKeys {
			diagnostics = append(diagnostics, PlanValidationError{
				InstancePath:  canonicalJSONPointer(instancePath, key),
				SchemaPath:    canonicalJSONPointer(object.Path, key),
				Code:          PlanCodeUnknownProperty,
				Keyword:       KeywordAdditionalProp,
				Message:       "unknown property \"" + key + "\"",
				PropertyName:  key,
				RejectedValue: asMap[key],
			})
		}
	}
	return diagnostics
}

// validateStringMap validates a free-form `additionalProperties:
// {type: "string"}` object. Every key is accepted; every value must
// be a JSON string.
func validateStringMap(object planObjectDescriptor, raw map[string]any, instancePath string) []PlanValidationError {
	var diagnostics []PlanValidationError
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := raw[key]
		fieldPath := canonicalJSONPointer(instancePath, key)
		if _, ok := value.(string); ok {
			continue
		}
		diagnostics = append(diagnostics, PlanValidationError{
			InstancePath:  fieldPath,
			SchemaPath:    canonicalJSONPointer(object.Path, key),
			Code:          PlanCodeInvalidType,
			Keyword:       KeywordType,
			Message:       "property \"" + key + "\" must be a string, got " + typeNameOf(value),
			RejectedValue: value,
		})
	}
	return diagnostics
}

// sortDiagnostics orders the diagnostics deterministically.
func sortDiagnostics(diagnostics []PlanValidationError) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		if diagnostics[i].InstancePath != diagnostics[j].InstancePath {
			return diagnostics[i].InstancePath < diagnostics[j].InstancePath
		}
		if diagnostics[i].PropertyName != diagnostics[j].PropertyName {
			return diagnostics[i].PropertyName < diagnostics[j].PropertyName
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
}

// (e PlanValidationError) String renders a single diagnostic.
func (e PlanValidationError) String() string {
	var b strings.Builder
	b.WriteString(e.InstancePath)
	b.WriteString(":\n")
	b.WriteString("  ")
	b.WriteString(string(e.Code))
	b.WriteString("\n  keyword=")
	b.WriteString(string(e.Keyword))
	b.WriteString("\n  ")
	b.WriteString(e.Message)
	if len(e.AcceptedValues) > 0 {
		b.WriteString("\n  accepted=")
		b.WriteString(strings.Join(e.AcceptedValues, ","))
	}
	return b.String()
}

// FormatValidationDiagnostics renders a PlanValidationResult as the
// concatenation of every error's compact form.
func FormatValidationDiagnostics(result PlanValidationResult) string {
	if result.Valid {
		return ""
	}
	parts := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		parts = append(parts, e.String())
	}
	return strings.Join(parts, "\n")
}

// typeNameOf returns a stable, lowercase JSON type label for
// diagnostic messages.
func typeNameOf(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case json.Number:
		return "number"
	case float64:
		return "number"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "unknown"
	}
}

// itoa is a tiny helper that wraps strconv.Itoa without pulling
// the import into every helper file.
func itoa(i int) string {
	return jsonNumberString(i)
}

func jsonNumberString(i int) string {
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// avoid unused-import warning when json isn't directly referenced.
var _ = json.Marshal

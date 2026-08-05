package closure

import (
	"encoding/json"
	"strconv"
)

// plan_schema_evaluator_helpers.go centralises the small
// type-and-shape helpers the schema evaluator uses. Splitting
// them from plan_schema_evaluator.go keeps the main evaluator
// file under the LLM-friendly 400-line threshold.

// valueMatchesType reports whether value matches the supplied
// JSON Schema type label.
func valueMatchesType(t string, value any) bool {
	switch t {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		switch v := value.(type) {
		case int, int64:
			return true
		case float64:
			// JSON Schema accepts floats that carry an integral
			// value as integers.
			return v == float64(int64(v))
		case json.Number:
			// UseNumber produces a string-typed wrapper; check
			// whether the literal text is an integer.
			return isIntegerText(v.String())
		}
		return false
	case "number":
		switch value.(type) {
		case int, int64, float64:
			return true
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "null":
		return value == nil
	}
	return true // unknown type: permissive
}

// numericValue extracts a numeric value from any and reports
// whether the supplied value was numeric. The function is
// host-width safe: it does not truncate or overflow native
// integers because all comparisons happen in float64.
//
// CORRECTION05: json.Number (produced by json.Decoder.UseNumber)
// is rendered as a float64 so exact-decoded numerics participate
// in the same bound checks as plain float64 values. The original
// text form is preserved when needed via jsonStringifyForSchema.
func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		n, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return n, true
	case string:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// isIntegerText reports whether the supplied literal JSON text
// is a mathematically integral number, including exponent forms
// like 1e0 and 6e2. Non-numeric strings and non-integral forms
// like 1.5 or 1e-1 return false.
func isIntegerText(s string) bool {
	if s == "" {
		return false
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		_, err2 := parseBigInt(s)
		if err2 != nil {
			return false
		}
		return true
	}
	return n == float64(int64(n))
}

// parseBigInt parses a decimal integer literal and returns the
// big integer. It accepts the JSON integer subset only.
func parseBigInt(s string) (any, error) {
	return strconv.ParseInt(s, 10, 64)
}

// isEmpty reports whether the supplied value is the JSON
// zero-value for its type.
func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

// schemaFieldRequiresPresence reports whether the supplied
// schema marks the field as required.
func schemaFieldRequiresPresence(propSchema map[string]any) bool {
	if propSchema == nil {
		return false
	}
	if app, ok := propSchema["x-applicability"].([]any); ok {
		for _, a := range app {
			if m, ok := a.(map[string]any); ok {
				if presence, ok := m["presence"].(string); ok && presence == "required" {
					return true
				}
			}
		}
	}
	return false
}

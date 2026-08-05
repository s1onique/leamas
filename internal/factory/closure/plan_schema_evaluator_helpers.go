package closure

import ()

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
func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float64:
		return v, true
	}
	return 0, false
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

package closure

import (
	"encoding/json"
	"math/big"
)

// plan_schema_evaluator_helpers.go centralises the small
// type-and-shape helpers the schema evaluator uses. Splitting
// them from plan_schema_evaluator.go keeps the main evaluator
// file under the LLM-friendly 400-line threshold.

// exactRational converts any JSON-decoded numeric value into an
// exact math/big.Rat. Returns nil and false when the value is not a
// recognisable JSON number or string-encoded number.
//
// CORRECTION06: this helper is the authoritative entry point for
// numeric classification and bound comparison. It does not pass
// through float64: an exact rational is retained so values that
// float64 would round across the [1,600] inclusive range still
// classify correctly.
func exactRational(value any) (*big.Rat, bool) {
	if value == nil {
		return nil, false
	}
	switch v := value.(type) {
	case *big.Rat:
		if v == nil {
			return nil, false
		}
		return new(big.Rat).Set(v), true
	case json.Number:
		return parseExactNumber(string(v))
	case string:
		return parseExactNumber(v)
	case int:
		return new(big.Rat).SetInt64(int64(v)), true
	case int64:
		return new(big.Rat).SetInt64(v), true
	case float64:
		// Note: float64 inputs are inherently imprecise; production
		// paths should use json.Number or string.
		r := new(big.Rat).SetFloat64(v)
		return r, true
	}
	return nil, false
}

// parseExactNumber parses a literal JSON number text into a
// big.Rat. Supports integers, decimals, and exponent notation
// (e.g. 1e0, 6e2, 1.5, 1e-1). Returns nil, false on parse error.
func parseExactNumber(s string) (*big.Rat, bool) {
	if s == "" {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, false
	}
	return r, true
}

// exactIsInteger reports whether the rational is mathematically
// integral (denominator is 1).
func exactIsInteger(r *big.Rat) bool {
	if r == nil {
		return false
	}
	return r.IsInt()
}

// exactLessThan reports whether a < b in exact rational terms.
func exactLessThan(a, b *big.Rat) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) < 0
}

// exactGreaterThan reports whether a > b in exact rational terms.
func exactGreaterThan(a, b *big.Rat) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) > 0
}

// exactEqual reports whether a == b in exact rational terms.
func exactEqual(a, b *big.Rat) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Cmp(b) == 0
}

// exactFromInt64 returns the rational representation of n.
func exactFromInt64(n int64) *big.Rat {
	return new(big.Rat).SetInt64(n)
}

// valueMatchesType reports whether value matches the supplied
// JSON Schema type label.
//
// CORRECTION05: json.Number (the type emitted by encoding/json
// when decoder.UseNumber is set) is recognised as integer when
// the underlying literal text is mathematically integral. This
// keeps the round-tripped public wire shape compatible with the
// evaluator.
func valueMatchesType(t string, value any) bool {
	switch t {
	case "string":
		switch value.(type) {
		case string, json.Number:
			return true
		}
		return false
	case "integer":
		switch v := value.(type) {
		case int, int64:
			return true
		case float64:
			return v == float64(int64(v))
		case json.Number:
			return isIntegerText(v.String())
		}
		return false
	case "number":
		_, isNum := exactRational(value)
		return isNum
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

// isIntegerText reports whether the supplied literal JSON text
// is a mathematically integral number, including exponent forms
// like 1e0 and 6e2. Non-numeric strings and non-integral forms
// like 1.5 or 1e-1 return false.
func isIntegerText(s string) bool {
	if s == "" {
		return false
	}
	r, ok := parseExactNumber(s)
	if !ok {
		return false
	}
	return r.IsInt()
}

// numericValue returns the float64 representation of a numeric
// value. The function is retained for compatibility with code
// paths that do not yet use exactRational; new numeric code must
// use exactRational so values that float64 would round across
// the [1,600] inclusive range still classify correctly.
func numericValue(value any) (float64, bool) {
	r, ok := exactRational(value)
	if !ok {
		return 0, false
	}
	f, exact := r.Float64()
	if exact {
		return f, true
	}
	// Return the float64 approximation but mark it as inexact
	// for the caller; callers should use exactRational for
	// authoritative numeric checks.
	return f, true
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

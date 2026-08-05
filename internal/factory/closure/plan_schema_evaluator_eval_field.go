package closure

import (
	"fmt"
	"regexp"
	"strings"
)

// plan_schema_evaluator_eval_field.go centralises the per-field
// recursive evaluation. Splitting it from plan_schema_evaluator.go
// keeps the main evaluator file under the LLM-friendly 400-line
// threshold.
//
// CORRECTION04:
//   - The applicability walker now takes an explicit
//     schemaPropertyContext carrying the parent schema,
//     parent instance, property name, and presence flag.
//   - Applicability is checked for ALL schema-declared
//     properties (not just present instances), so absent
//     required fields under matching mode values are rejected.
//   - The walker never depends on propSchema["json_name"].
//   - Absent and explicit null remain distinct.
//   - Required/forbidden checks precede value validation.

// schemaPropertyContext carries the parent context and the
// property under evaluation so the applicability walker can
// resolve sibling conditions without re-discovering the parent.
type schemaPropertyContext struct {
	PropertyName   string
	PropertySchema map[string]any
	ParentSchema   map[string]any
	ParentInstance map[string]any

	Value   any
	Present bool

	SchemaPath   string
	InstancePath string
}

// evalField evaluates a single instance value against a property
// schema. Nested objects and arrays are walked recursively.
func evalField(propSchema map[string]any, value any, leamasExtensions bool) schemaEvaluation {
	if propSchema == nil {
		return acceptAny(leamasExtensions)
	}
	if constVal, ok := propSchema["const"]; ok {
		if !schemaValueEqual(constVal, value) {
			return rejectAny(leamasExtensions, "value does not equal const")
		}
		return acceptAny(leamasExtensions)
	}
	if enum, ok := propSchema["enum"].([]any); ok {
		matched := false
		for _, e := range enum {
			if schemaValueEqual(e, value) {
				matched = true
				break
			}
		}
		if !matched {
			return rejectAny(leamasExtensions, "value not in enum")
		}
		return acceptAny(leamasExtensions)
	}
	typeStr, hasType := propSchema["type"].(string)
	if hasType {
		if !valueMatchesType(typeStr, value) {
			return rejectAny(leamasExtensions, fmt.Sprintf("type %q mismatch", typeStr))
		}
	}
	if minLenRaw, ok := propSchema["minLength"]; ok {
		if minLen, isNum := numericValue(minLenRaw); isNum {
			if s, ok := value.(string); !ok || float64(len(s)) < minLen {
				return rejectAny(leamasExtensions, fmt.Sprintf("minLength %v violated", minLen))
			}
		}
	}
	if pattern, ok := propSchema["pattern"].(string); ok {
		if s, ok := value.(string); !ok {
			return rejectAny(leamasExtensions, "pattern requires string")
		} else {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return rejectAny(leamasExtensions, fmt.Sprintf("pattern unparsable: %v", err))
			} else if !re.MatchString(s) {
				return rejectAny(leamasExtensions, "pattern mismatch")
			}
		}
	}
	// Numeric bound checks: the standard JSON Schema treats
	// integers as a subtype of numbers; the evaluator follows
	// the same convention and rejects only non-integer floats
	// when the type label is "integer". Non-integer floats are
	// rejected by the integer type check; integer-valued floats
	// (60.0) are accepted.
	if hasType && typeStr == "integer" {
		switch v := value.(type) {
		case float64:
			if v != float64(int64(v)) {
				return rejectAny(leamasExtensions, "value is not an integer")
			}
		}
	}
	// CORRECTION05: bound checks must work with both in-memory
	// numeric types and the json.Number values produced by
	// decoder.UseNumber.
	if minRaw, ok := propSchema["minimum"]; ok {
		if min, isNum := numericValue(minRaw); isNum {
			n, isValNum := numericValue(value)
			if !isValNum || n < min {
				return rejectAny(leamasExtensions, fmt.Sprintf("minimum %v violated", min))
			}
		}
	}
	if maxRaw, ok := propSchema["maximum"]; ok {
		if max, isNum := numericValue(maxRaw); isNum {
			n, isValNum := numericValue(value)
			if !isValNum || n > max {
				return rejectAny(leamasExtensions, fmt.Sprintf("maximum %v violated", max))
			}
		}
	}
	// Array constraints.
	if hasType && typeStr == "array" {
		if items, ok := propSchema["items"].(map[string]any); ok {
			arr, isArr := value.([]any)
			if isArr {
				for i, item := range arr {
					subEval := evalField(items, item, leamasExtensions)
					if !subEval.Accept {
						subEval.Issues = []string{fmt.Sprintf("array item %d: %s", i, strings.Join(subEval.Issues, "; "))}
						return subEval
					}
				}
			}
		}
		if minItems, ok := propSchema["minItems"].(float64); ok {
			if arr, isArr := value.([]any); isArr && float64(len(arr)) < minItems {
				return rejectAny(leamasExtensions, fmt.Sprintf("minItems %v violated", minItems))
			}
		}
		if minItems, ok := propSchema["minItems"].(int); ok {
			if arr, isArr := value.([]any); isArr && len(arr) < minItems {
				return rejectAny(leamasExtensions, fmt.Sprintf("minItems %v violated", minItems))
			}
		}
	}
	// Object constraints.
	if hasType && typeStr == "object" {
		if asMap, ok := value.(map[string]any); ok {
			// x-applicability: enforce mode-dependent required/forbidden
			// on absent properties of this object. The walker walks the
			// SCHEMA-declared properties (not the instance properties) so
			// absent required fields are caught even when the instance
			// does not include them.
			if leamasExtensions {
				if sub, ok := propSchema["properties"].(map[string]any); ok {
					for propName, propSchemaAny := range sub {
						psm, ok := propSchemaAny.(map[string]any)
						if !ok {
							continue
						}
						app, ok := normalizeApplicabilityRules(psm["x-applicability"])
						if !ok {
							continue
						}
						_, fieldPresent := asMap[propName]
						ctx := schemaPropertyContext{
							PropertyName:   propName,
							PropertySchema: psm,
							ParentSchema:   propSchema,
							ParentInstance: asMap,
							Value:          nil,
							Present:        fieldPresent,
						}
						if fieldPresent {
							ctx.Value = asMap[propName]
						}
						if res := applyApplicabilityRules(ctx, app, leamasExtensions); !res.Accept {
							return res
						}
					}
				}
			}
			if req, ok := propSchema["required"].([]any); ok {
				for _, r := range req {
					name, ok := r.(string)
					if !ok {
						continue
					}
					if _, present := asMap[name]; !present {
						return rejectAny(leamasExtensions, "required property missing: "+name)
					}
				}
			}
			if sub, ok := propSchema["properties"].(map[string]any); ok {
				for propName, propSchemaAny := range sub {
					propSchemaMap, ok := propSchemaAny.(map[string]any)
					if !ok {
						continue
					}
					val, present := asMap[propName]
					if !present {
						continue
					}
					if subEval := evalField(propSchemaMap, val, leamasExtensions); !subEval.Accept {
						subEval.Issues = []string{fmt.Sprintf("property %s: %s", propName, strings.Join(subEval.Issues, "; "))}
						return subEval
					}
				}
			}
			if addl, ok := propSchema["additionalProperties"]; ok {
				if addlBool, ok := addl.(bool); ok && !addlBool {
					for k := range asMap {
						if sub, ok := propSchema["properties"].(map[string]any); ok {
							if _, found := sub[k]; found {
								continue
							}
						}
						return rejectAny(leamasExtensions, "unknown property: "+k)
					}
				}
			}
		}
	}
	// x-leamas-repository-relative-path: read the actual
	// emitted extension members and pass them to the
	// extension-aware path evaluator.
	if leamasExtensions {
		if ext, ok := propSchema["x-leamas-repository-relative-path"].(map[string]any); ok {
			if s, ok := value.(string); ok {
				ok, reason := portablePathAccepts(s, ext)
				if !ok {
					return rejectAny(leamasExtensions, reason)
				}
			}
		}
	}
	return acceptAny(leamasExtensions)
}

// normalizeApplicabilityRules accepts any of the Go-side shapes the
// public JSON Schema extension may arrive in: []map[string]any (direct
// generator output), []any (post-decoding public wire), or nil/missing
// (no applicability). Each element is normalised to map[string]any.
//
// CORRECTION05: the evaluator must not depend on a Go-specific typed
// slice returned by the generator; an external consumer that decodes
// the public schema bytes into map[string]any will see JSON arrays as
// []any regardless of how the generator stores them internally.
func normalizeApplicabilityRules(raw any) ([]map[string]any, bool) {
	if raw == nil {
		return nil, false
	}
	switch v := raw.(type) {
	case []map[string]any:
		return v, true
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, false
			}
			out = append(out, m)
		}
		return out, true
	default:
		return nil, false
	}
}

// applyApplicabilityRules evaluates the x-applicability rules
// for a single property against the supplied parent context.
// Fail-closed shape validation is performed so malformed rules
// never silently pass.
func applyApplicabilityRules(ctx schemaPropertyContext, rules []map[string]any, leamasExtensions bool) schemaEvaluation {
	seen := make(map[string]bool)
	for idx, m := range rules {
		siblingAny, hasSibling := m["sibling"]
		if !hasSibling {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: sibling absent", idx))
		}
		sibling, ok := siblingAny.(string)
		if !ok || sibling == "" {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: sibling wrong type", idx))
		}
		valueAny, hasValue := m["value"]
		if !hasValue {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: value absent", idx))
		}
		valueStr, ok := valueAny.(string)
		if !ok || valueStr == "" {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: value wrong type", idx))
		}
		presenceAny, hasPresence := m["presence"]
		if !hasPresence {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: presence absent", idx))
		}
		presence, ok := presenceAny.(string)
		if !ok {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: presence wrong type", idx))
		}
		switch presence {
		case "required", "optional", "forbidden":
			// accepted
		default:
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: unknown presence %q", idx, presence))
		}
		key := sibling + "=" + valueStr
		if seen[key] {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability: duplicate rule %s", key))
		}
		seen[key] = true
		// Validate that the sibling is declared in the parent schema.
		if sub, ok := ctx.ParentSchema["properties"].(map[string]any); ok {
			if _, declared := sub[sibling]; !declared {
				return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: sibling %q not declared in parent schema", idx, sibling))
			}
		}
		// Look up the sibling value in the parent instance.
		siblingVal, hasSiblingVal := ctx.ParentInstance[sibling]
		if !hasSiblingVal {
			// Sibling absent: no rule applies.
			continue
		}
		siblingStr, isString := siblingVal.(string)
		if !isString {
			return rejectAny(leamasExtensions, fmt.Sprintf("x-applicability rule %d: sibling %q has incompatible type", idx, sibling))
		}
		if siblingStr != valueStr {
			continue
		}
		// Rule applies.
		switch presence {
		case "required":
			if !ctx.Present {
				return rejectAny(leamasExtensions, "required property missing: "+ctx.PropertyName)
			}
		case "forbidden":
			if ctx.Present {
				return rejectAny(leamasExtensions, "forbidden property present: "+ctx.PropertyName)
			}
		}
	}
	return acceptAny(leamasExtensions)
}

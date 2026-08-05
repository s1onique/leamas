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

// evalField evaluates a single instance value against a property
// schema. Nested objects and arrays are walked recursively.
func evalField(propSchema map[string]any, value any, leamasExtensions bool) schemaEvaluation {
	if propSchema == nil {
		return acceptAny(leamasExtensions)
	}
	// x-applicability: enforce mode-dependent required/forbidden
	// at the field's own level. The walker reads the sibling
	// from the parent context (value) when value is a map; when
	// the value is absent (nil), the walker skips the check
	// because the parent's own object case already fires
	// required_property_missing for the absent field.
	if leamasExtensions {
		if app, ok := propSchema["x-applicability"].([]any); ok {
			if value != nil {
				if parent, ok := value.(map[string]any); ok {
					for _, a := range app {
						m, ok := a.(map[string]any)
						if !ok {
							continue
						}
						sibling, _ := m["sibling"].(string)
						valueStr, _ := m["value"].(string)
						presence, _ := m["presence"].(string)
						if sibling == "" || valueStr == "" {
							continue
						}
						parentVal, present := parent[sibling]
						if !present {
							continue
						}
						parentStr, _ := parentVal.(string)
						if parentStr != valueStr {
							continue
						}
						fieldName, _ := propSchema["json_name"].(string)
						_, fieldPresent := parent[fieldName]
						switch presence {
						case "required":
							if !fieldPresent {
								return rejectAny(leamasExtensions, "required property missing: "+fieldName)
							}
						case "forbidden":
							if fieldPresent {
								return rejectAny(leamasExtensions, "forbidden property present: "+fieldName)
							}
						}
					}
				}
			}
		}
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
	// x-applicability: enforce mode-dependent required/forbidden
	// BEFORE other value constraints so absent fields under
	// required mode fire the required diagnostic consistently.
	if leamasExtensions {
		if app, ok := propSchema["x-applicability"].([]any); ok {
			if parent, ok := value.(map[string]any); ok {
				for _, a := range app {
					m, ok := a.(map[string]any)
					if !ok {
						continue
					}
					sibling, _ := m["sibling"].(string)
					valueStr, _ := m["value"].(string)
					presence, _ := m["presence"].(string)
					if sibling == "" || valueStr == "" {
						continue
					}
					parentVal, parentHasSibling := parent[sibling]
					if !parentHasSibling {
						continue
					}
					parentStr, _ := parentVal.(string)
					if parentStr != valueStr {
						continue
					}
					fieldName, _ := propSchema["json_name"].(string)
					_, fieldPresent := parent[fieldName]
					switch presence {
					case "required":
						if !fieldPresent {
							return rejectAny(leamasExtensions, "required property missing: "+fieldName)
						}
					case "forbidden":
						if fieldPresent {
							return rejectAny(leamasExtensions, "forbidden property present: "+fieldName)
						}
					}
				}
			}
		}
	}
	if minLen, ok := propSchema["minLength"].(float64); ok {
		if s, ok := value.(string); !ok || float64(len(s)) < minLen {
			return rejectAny(leamasExtensions, fmt.Sprintf("minLength %v violated", minLen))
		}
	}
	if minLen, ok := propSchema["minLength"].(int); ok {
		if s, ok := value.(string); !ok || len(s) < minLen {
			return rejectAny(leamasExtensions, fmt.Sprintf("minLength %v violated", minLen))
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
	if min, ok := propSchema["minimum"].(float64); ok {
		n, isNum := numericValue(value)
		if !isNum || float64(n) < min {
			return rejectAny(leamasExtensions, fmt.Sprintf("minimum %v violated", min))
		}
	}
	if min, ok := propSchema["minimum"].(int); ok {
		n, isNum := numericValue(value)
		if !isNum || n < float64(min) {
			return rejectAny(leamasExtensions, fmt.Sprintf("minimum %v violated", min))
		}
	}
	if min, ok := propSchema["minimum"].(int64); ok {
		n, isNum := numericValue(value)
		if !isNum || n < float64(min) {
			return rejectAny(leamasExtensions, fmt.Sprintf("minimum %v violated", min))
		}
	}
	if max, ok := propSchema["maximum"].(float64); ok {
		n, isNum := numericValue(value)
		if !isNum || float64(n) > max {
			return rejectAny(leamasExtensions, fmt.Sprintf("maximum %v violated", max))
		}
	}
	if max, ok := propSchema["maximum"].(int); ok {
		n, isNum := numericValue(value)
		if !isNum || n > float64(max) {
			return rejectAny(leamasExtensions, fmt.Sprintf("maximum %v violated", max))
		}
	}
	if max, ok := propSchema["maximum"].(int64); ok {
		n, isNum := numericValue(value)
		if !isNum || n > float64(max) {
			return rejectAny(leamasExtensions, fmt.Sprintf("maximum %v violated", max))
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
		_ = fmt.Sprintf("object case: %v mode=%v", propSchema["json_name"], value.(map[string]any)["mode"])
		if asMap, ok := value.(map[string]any); ok {
			// x-applicability: enforce mode-dependent required/forbidden
			// on absent properties of this object.
			if leamasExtensions {
				if sub, ok := propSchema["properties"].(map[string]any); ok {
					modeStr, _ := asMap["mode"].(string)
					for propName, propSchemaAny := range sub {
						psm, ok := propSchemaAny.(map[string]any)
						if !ok {
							continue
						}
						app, ok := psm["x-applicability"].([]any)
						if !ok {
							continue
						}
						for _, a := range app {
							m, ok := a.(map[string]any)
							if !ok {
								continue
							}
							sibling, _ := m["sibling"].(string)
							valueStr, _ := m["value"].(string)
							presence, _ := m["presence"].(string)
							if sibling != "mode" || valueStr != modeStr {
								continue
							}
							_, fieldPresent := asMap[propName]
							switch presence {
							case "required":
								if !fieldPresent {
									return rejectAny(leamasExtensions, "required property missing: "+propName)
								}
							case "forbidden":
								if fieldPresent {
									return rejectAny(leamasExtensions, "forbidden property present: "+propName)
								}
							}
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
